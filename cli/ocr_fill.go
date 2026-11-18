package cli

import (
	"bufio"
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/lane"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/ocr"
	"github.com/tamnd/olymmath-agent/pdfsrc"
	"github.com/tamnd/olymmath-agent/site"
)

func init() {
	register(command{"ocr-fill", "read the official papers of the link-only problems of an edition with a vision model, and give each problem whose text passes the checks that text as a new statement (om ocr-fill -files MANIFEST hr-county/2015)", runOCRFill})
}

// fillOptions are the flags of om ocr-fill.
type fillOptions struct {
	root, files, lanes, dpi, work, review string
	dry                                   bool
	// agree is the lowest word signal of the text that the model read at a position with the text that the problem has. Each problem of the paper with a text is such a test of the order of the paper.
	agree float64
}

func runOCRFill(env Env, args []string) int {
	fs := newFlags("ocr-fill", env.Stderr)
	var o fillOptions
	fs.StringVar(&o.root, "content", ".", "the root of the content repository")
	fs.StringVar(&o.files, "files", "", "a file of local copies of the papers: lines of the URL of a paper, a tab, and the path of its PDF, such as copies from the raw store")
	fs.StringVar(&o.lanes, "lanes", "codex,gemini,claude", "the model clients to ask, in order")
	fs.StringVar(&o.dpi, "dpi", "200,300", "the resolutions of the page images. A page whose text has a flag is read again at the next one")
	fs.StringVar(&o.work, "work", filepath.Join(os.TempDir(), "om-ocr"), "the directory for the page images. om ocr-fill deletes them when it ends")
	fs.StringVar(&o.review, "review", "", "a TSV file to which om ocr-fill appends each link-only problem that keeps its link only, with the reason")
	fs.Float64Var(&o.agree, "agree", 0.3, "the lowest word signal of the text that the model read at a position with the text of the problem there")
	fs.BoolVar(&o.dry, "dry", false, "print the texts and write no file")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) == 0 || o.files == "" {
		fmt.Fprintln(env.Stderr, "usage: om ocr-fill -files MANIFEST [-lanes LANES] [-review FILE] [-dry] EDITION|COMPETITION...\nEDITION is an edition such as hr-county/2015, and COMPETITION names all its editions. MANIFEST gives the local copy of each paper. om ocr-fill reads the pages of each paper that has a link-only problem, cuts them into problems in the order of the paper, and gives the text of a problem to its record as a new statement when the problems of the paper line up with the positions and the text passes the checks of om ocr.")
		return ExitUsage
	}
	if n := strings.TrimSpace(o.lanes); n == lane.LocalName || n == ocr.NativeLane {
		fmt.Fprintln(env.Stderr, "om ocr-fill: -lanes takes the model clients, not the local or the native lane")
		return ExitUsage
	}
	files, err := readManifest(o.files)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: -files: %v\n", err)
		return ExitUsage
	}
	dpi, err := parseInts(o.dpi)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: -dpi: %v\n", err)
		return ExitUsage
	}
	corpus, findings := content.Load(o.root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om ocr-fill: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	var eds []*content.Edition
	for _, a := range pos {
		if !strings.Contains(strings.Trim(a, "/"), "/") {
			c := corpus.Competition(a)
			if c == nil {
				fmt.Fprintf(env.Stderr, "om ocr-fill: %q is not a competition\n", a)
				return ExitUsage
			}
			eds = append(eds, c.Editions...)
			continue
		}
		e, err := findEdition(corpus, a)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om ocr-fill: %v\n", err)
			return ExitUsage
		}
		eds = append(eds, e)
	}
	lanes, err := lane.Select(lane.Defaults(), o.lanes)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: %v\n", err)
		return ExitUsage
	}
	cache, err := ocr.DefaultCache()
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: %v\n", err)
		return ExitFailure
	}
	if err := os.MkdirAll(o.work, 0o750); err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: %v\n", err)
		return ExitFailure
	}
	work, err := os.MkdirTemp(o.work, "fill-")
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: %v\n", err)
		return ExitFailure
	}
	defer func() { _ = os.RemoveAll(work) }()
	markup, err := site.NewMarkup(nil)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: %v\n", err)
		return ExitFailure
	}
	chain := &lane.Chain{Lanes: lanes, State: filepath.Join(cache.Dir, "lanes.json")}
	r := &ocr.Reader{Asker: chain, Cache: cache, DPI: dpi, Work: work, Log: env.Stderr}
	f := &filler{env: env, o: o, corpus: corpus, files: files, reader: r, markup: markup}
	code := ExitOK
	for _, e := range eds {
		if err := f.edition(context.Background(), e); err != nil {
			if errors.Is(err, lane.ErrExhausted) {
				fmt.Fprintf(env.Stderr, "om ocr-fill: %s: %v\nThe pages read so far are in the cache. Run again when a lane has its allowance.\n", e.Locator(), err)
				code = ExitLater
				break
			}
			fmt.Fprintf(env.Stderr, "om ocr-fill: %s: %v\n", e.Locator(), err)
			code = ExitFailure
			break
		}
	}
	if err := f.writeReview(); err != nil {
		fmt.Fprintf(env.Stderr, "om ocr-fill: -review: %v\n", err)
		code = ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d papers read, %d problems get a text, %d stay link-only.\n", f.papers, f.filled, len(f.held))
	return code
}

// readManifest reads the lines URL<tab>PATH of a file of local copies.
func readManifest(path string) (map[string]string, error) {
	fh, err := os.Open(path) // #nosec G304 -- a file that the user names.
	if err != nil {
		return nil, err
	}
	defer func() { _ = fh.Close() }()
	out := map[string]string{}
	sc := bufio.NewScanner(fh)
	for sc.Scan() {
		u, p, ok := strings.Cut(strings.TrimSpace(sc.Text()), "\t")
		if ok && u != "" && p != "" {
			out[u] = p
		}
	}
	return out, sc.Err()
}

// filler gives texts to the link-only problems of editions.
type filler struct {
	env    Env
	o      fillOptions
	corpus *content.Corpus
	files  map[string]string
	reader *ocr.Reader
	markup *site.Markup

	papers, filled int
	held           []heldLink
}

// heldLink is a link-only problem that keeps its link only, with the reason.
type heldLink struct {
	at, tag, url string
	page         int
	why          string
}

// fillRec is a problem of an edition with its statement that links to a page of a paper.
type fillRec struct {
	p    *content.Problem
	pos  string
	st   *content.Statement
	page int
	// link tells that the problem has no text.
	link bool
	// shared tells that the problem is at the position only by an appearance, as a problem that the paper prints again with the label of the problem of another paper. It keeps its place in the paper and gets no text from it.
	shared bool
}

// paperGroups gives the problems of the edition for each official paper of the edition, in the order of their positions. A problem is in the group of a paper when a statement links to a page of it.
func paperGroups(e *content.Edition) (map[string][]fillRec, []string) {
	official := map[string]bool{}
	var urls []string
	for _, p := range e.Official {
		if !official[p.URL] {
			official[p.URL] = true
			urls = append(urls, p.URL)
		}
	}
	groups := map[string][]fillRec{}
	for _, p := range e.Problems {
		a := p.Primary()
		if a == nil || a.Locator.Comp != e.Competition.Code || a.Locator.Edition != e.ID {
			continue
		}
		seen := map[string]bool{}
		for i := range p.Statements {
			s := &p.Statements[i]
			if !official[s.URL] || s.Page <= 0 || seen[s.URL] {
				continue
			}
			seen[s.URL] = true
			groups[s.URL] = append(groups[s.URL], fillRec{p: p, pos: a.Locator.Position, st: s, page: s.Page, link: linkOnlyProblem(p)})
		}
	}
	addShared(e, groups)
	for _, g := range groups {
		slices.SortStableFunc(g, func(a, b fillRec) int { return comparePositions(a.pos, b.pos) })
	}
	return groups, urls
}

// addShared puts each problem that is at a position of the edition by an appearance that is not its primary one into the group of the paper whose positions it shares a prefix with, as p115 with p111 to p114, when exactly one paper has them.
func addShared(e *content.Edition, groups map[string][]fillRec) {
	for _, p := range e.Problems {
		for _, a := range p.Appearances {
			if a.Primary || a.Locator.Comp != e.Competition.Code || a.Locator.Edition != e.ID {
				continue
			}
			if pa := p.Primary(); pa != nil && pa.Locator == a.Locator {
				continue
			}
			pre := positionPrefix(a.Locator.Position)
			var hit []string
			for url, g := range groups {
				if slices.ContainsFunc(g, func(r fillRec) bool { return r.p == p && r.pos == a.Locator.Position }) {
					hit = nil
					break
				}
				if slices.ContainsFunc(g, func(r fillRec) bool { return positionPrefix(r.pos) == pre }) {
					hit = append(hit, url)
				}
			}
			if len(hit) == 1 {
				groups[hit[0]] = append(groups[hit[0]], fillRec{p: p, pos: a.Locator.Position, shared: true})
			}
		}
	}
}

// positionPrefix gives a position without its last digit and the letters after it, as p11 for p115 and p12 for p126a, or the position when it has no digit at the end.
func positionPrefix(pos string) string {
	s := strings.TrimRight(pos, "abcdefghijklmnopqrstuvwxyz")
	if s == "" || s[len(s)-1] < '0' || s[len(s)-1] > '9' {
		return pos
	}
	return s[:len(s)-1]
}

// linkOnlyProblem tells that no statement of the problem has a text.
func linkOnlyProblem(p *content.Problem) bool {
	if strings.TrimSpace(p.Statement) != "" {
		return false
	}
	for _, s := range p.Statements {
		if strings.TrimSpace(s.Text) != "" {
			return false
		}
	}
	return true
}

var chunkRE = regexp.MustCompile(`[0-9]+|[^0-9]+`)

// comparePositions orders positions by their letters and their numbers, so h2 comes before h10.
func comparePositions(a, b string) int {
	x, y := chunkRE.FindAllString(a, -1), chunkRE.FindAllString(b, -1)
	for i := 0; i < len(x) && i < len(y); i++ {
		n, errN := strconv.Atoi(x[i])
		m, errM := strconv.Atoi(y[i])
		if errN == nil && errM == nil {
			if c := cmp.Compare(n, m); c != 0 {
				return c
			}
			continue
		}
		if c := strings.Compare(x[i], y[i]); c != 0 {
			return c
		}
	}
	return cmp.Compare(len(x), len(y))
}

// edition reads each paper of the edition that has a link-only problem.
func (f *filler) edition(ctx context.Context, e *content.Edition) error {
	groups, urls := paperGroups(e)
	for _, url := range urls {
		g := groups[url]
		if !slices.ContainsFunc(g, func(r fillRec) bool { return r.link }) {
			continue
		}
		if err := f.paper(ctx, e, url, g); err != nil {
			return err
		}
	}
	return nil
}

// paper reads the pages of one paper and gives the texts to its link-only problems.
func (f *filler) paper(ctx context.Context, e *content.Edition, url string, g []fillRec) error {
	hold := func(why string) {
		for _, r := range g {
			if r.link {
				f.hold(r, url, why)
			}
		}
	}
	first := g[slices.IndexFunc(g, func(r fillRec) bool { return !r.shared })]
	src := f.corpus.Source(first.st.Source)
	if src == nil || !textAllowed(src) {
		hold("the source allows no text")
		return nil
	}
	file := f.files[url]
	if file == "" {
		hold("no local copy of the paper")
		return nil
	}
	in, err := pdfsrc.ReadInfo(ctx, pdfsrc.Exec, file)
	if err != nil {
		hold("the PDF does not open: " + oneLine(err.Error()))
		return nil
	}
	pages := pagesToRead(g, in.Pages)
	read, err := f.reader.Read(ctx, file, pages)
	if errors.Is(err, lane.ErrExhausted) {
		return err
	}
	if err != nil {
		hold("the pages were not read: " + oneLine(err.Error()))
		return nil
	}
	f.papers++
	probs := ocr.Labels(read)
	if why := alignPaper(g, probs, f.o.agree); why != "" {
		fmt.Fprintf(f.env.Stdout, "%s %s: %s\n", e.Locator(), url, why)
		hold(why)
		return nil
	}
	byPage := map[int]ocr.Page{}
	for _, p := range read {
		byPage[p.N] = p
	}
	for i, r := range g {
		if !r.link {
			continue
		}
		pr := probs[i]
		pg := byPage[pr.Page]
		flags := slices.Clone(pr.Flags)
		for _, fl := range pg.Flags {
			if fl == ocr.FlagRefusal && !slices.Contains(flags, fl) {
				flags = append(flags, fl)
			}
		}
		if tailRE.MatchString(lastParagraph(pr.Text)) {
			flags = append(flags, "tail")
		}
		if _, errs := f.markup.HTML(pr.Text); len(errs) > 0 {
			flags = append(flags, "render")
		}
		conf := ocr.Confidence(flags)
		if conf < ocr.Threshold || (r.st.Zone != "Z1" && r.st.Zone != "Z2") {
			why := "the checks found " + strings.Join(flags, ", ")
			if conf >= ocr.Threshold {
				why = "the zone of the statement is " + r.st.Zone
			}
			fmt.Fprintf(f.env.Stdout, "%s/%s: page %d, a link only: %s\n", e.Locator(), r.pos, pr.Page, why)
			f.hold(r, url, why)
			continue
		}
		scan := canon.Scan{Page: pr.Page, Text: pr.Text, Extraction: pg.Extraction(), Confidence: conf}
		fmt.Fprintf(f.env.Stdout, "%s/%s: page %d, text, confidence %.2f, %s\n", e.Locator(), r.pos, pr.Page, conf, scan.Extraction)
		if f.o.dry {
			fmt.Fprintf(f.env.Stdout, "\n%s\n\n", pr.Text)
			f.filled++
			continue
		}
		if err := f.write(e, r, scan); err != nil {
			return fmt.Errorf("%s/%s: %w", e.Locator(), r.pos, err)
		}
		f.filled++
	}
	return nil
}

// tailRE finds a line of the paper that is not a part of a problem: the points of the problems, the time, a note to turn the page. The prompt asks for a heading there, and this catches the answers that do not obey.
var tailRE = regexp.MustCompile(`(?i)\b(?:vrijedi|vrijede|boduje|bodova|okreni|arbeitszeit|hilfsmittel|time allowed|calculators? (?:are|is) not|points each|each problem is worth)\b`)

// lastParagraph gives the last paragraph of a text.
func lastParagraph(s string) string {
	parts := strings.Split(strings.TrimSpace(s), "\n\n")
	return parts[len(parts)-1]
}

// pagesToRead gives the pages from the first page of a problem of the paper to the last one, and the page after it when the paper has one, where the last problem can go on.
func pagesToRead(g []fillRec, total int) []int {
	lo, hi := 0, 0
	for _, r := range g {
		if r.page <= 0 {
			continue
		}
		if lo == 0 {
			lo, hi = r.page, r.page
		}
		lo, hi = min(lo, r.page), max(hi, r.page)
	}
	if lo == 0 {
		return nil
	}
	if hi < total {
		hi++
	}
	var out []int
	for n := lo; n <= hi; n++ {
		out = append(out, n)
	}
	return out
}

// alignPaper tells why the problems that the model read do not line up with the problems of the paper, or "" when they do: the same number of problems, each on the page of its problem, the last digit of each label the last digit of its position, and each text that a problem has agrees with the text that the model read there.
func alignPaper(g []fillRec, probs []ocr.Labeled, agree float64) string {
	if len(probs) != len(g) {
		return fmt.Sprintf("the model read %d problems and the paper has %d problems in the content", len(probs), len(g))
	}
	for i, r := range g {
		pr := probs[i]
		if !r.shared && pr.Page != r.page {
			return fmt.Sprintf("%s is on page %d, and the problem %s that the model read there is on page %d", r.pos, r.page, pr.Label, pr.Page)
		}
		if d, pd := pr.Digit(), ocr.PositionDigit(r.pos); d >= 0 && pd >= 0 && d != pd {
			return fmt.Sprintf("the problem %s that the model read does not end in the digit of %s", pr.Label, r.pos)
		}
		if !r.link && strings.TrimSpace(r.p.Statement) != "" {
			if j := match.Jaccard(match.Shingles(pr.Text), match.Shingles(r.p.Statement)); j < agree {
				return fmt.Sprintf("the text that the model read at %s does not agree with its text (word signal %.2f)", r.pos, j)
			}
		}
	}
	return ""
}

func (f *filler) hold(r fillRec, url, why string) {
	f.held = append(f.held, heldLink{at: r.p.Locator(), tag: r.p.Tag, url: url, page: r.page, why: why})
}

// write adds the text to the file of the problem as a new statement, which becomes the canonical statement.
func (f *filler) write(e *content.Edition, r fillRec, s canon.Scan) error {
	c := &canon.Candidate{Source: r.st.Source, Origin: r.st.Origin, Zone: r.st.Zone, Lang: r.st.Lang, Name: r.st.Name, URL: r.st.URL}
	add, err := canon.AddScan(r.p, c, s)
	if err != nil {
		return err
	}
	path := filepath.Join(f.o.root, filepath.FromSlash(r.p.Path))
	b, err := os.ReadFile(path) // #nosec G304 -- a problem file of the content.
	if err != nil {
		return err
	}
	b, err = canon.Apply(b, add)
	if err != nil {
		return err
	}
	_, rule := canon.Rank(&canon.Candidate{Origin: r.st.Origin, Lang: r.st.Lang}, original(e.Competition))
	b, err = canon.Fill(b, add.ID(c), s.Text, rule, "captured")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644) // #nosec G306 -- content files are public.
}

// writeReview appends the problems that keep their link only to the file of -review.
func (f *filler) writeReview() error {
	if f.o.review == "" || len(f.held) == 0 {
		return nil
	}
	_, statErr := os.Stat(f.o.review)
	fh, err := os.OpenFile(f.o.review, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // #nosec G302 G304 -- a report file that the user names.
	if err != nil {
		return err
	}
	var w io.Writer = fh
	if errors.Is(statErr, os.ErrNotExist) {
		fmt.Fprintln(w, "# The link-only problems that om ocr-fill read with a vision model and did not give a text: the position, the problem, the paper, the page, and the reason.")
	}
	for _, h := range f.held {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", h.at, h.tag, h.url, h.page, h.why)
	}
	return fh.Close()
}
