package cli

import (
	"bytes"
	"cmp"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/ocr"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"demo", "read the papers of the 1st to the 34th German olympiads from olympiade-mathematik.de and write their problems to de-mo and de-mo-r1 to de-mo-r4 (om demo -write)", runDemo})
}

const (
	// demoSource is the archive of the papers of the German olympiad.
	demoSource = "olympiade-mathematik"
	demoSite   = "https://www.olympiade-mathematik.de/"
	// demoDPI renders a page at the scale of the pixels of pdftohtml, 1.5 pixels for each point.
	demoDPI = 108
	// demoInkMin is the number of dark pixels outside the runs of text that make a drawing: the rule of a small fraction has about ten.
	demoInkMin = 6
)

// demoPDFRE is the link of a paper PDF on the page of a grade.
var demoPDFRE = regexp.MustCompile(`pdf/block_a/(\d{5}_a\.pdf)`)

// demoGradePage gives the page of the papers of a grade, from 5 to 12.
func demoGradePage(grade int) string {
	return fmt.Sprintf("%sindex.php?option=com_klassen&view=klassen&klasse=%d&Itemid=%d", demoSite, grade, 112+grade)
}

// demoAt gives the locator of problem n of a paper. The edition is the year of the final: the 1st olympiad, 1961/62, is 1962. The final paper of grade 12, the highest grade, is de-mo; the other papers are de-mo-r1 to de-mo-r4 with the position p<grade><n>.
func demoAt(p official.DemoPaper, n string) string {
	year := 1961 + p.Olympiad
	if p.Round == 4 && p.Grade == 12 {
		return fmt.Sprintf("de-mo/%d/p%s", year, n)
	}
	return fmt.Sprintf("de-mo-r%d/%d/p%d%s", p.Round, year, p.Grade, n)
}

// demoName is the name of the statement of a paper.
func demoName(p official.DemoPaper) string {
	return fmt.Sprintf("olympiade-mathematik.de, %d. Mathematik-Olympiade, %d. Stufe, Klasse %d", p.Olympiad, p.Round, p.Grade)
}

// demoRead is one problem that om demo read, with the paper and the URL of its PDF.
type demoRead struct {
	at    string
	paper official.DemoPaper
	url   string
	prob  official.DemoProblem
	flags []string
	conf  float64
}

func runDemo(env Env, args []string) int {
	fs := newFlags("demo", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 1, "the first olympiad")
	to := fs.Int("to", 34, "the last olympiad")
	write := fs.Bool("write", false, "write the problems and the official papers of the editions")
	explain := fs.Bool("explain", false, "print each problem, its position and its flags")
	maxAge := fs.Duration("max-age", 30*24*time.Hour, "use a capture of the raw store that is younger than this, with no request")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om demo: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	src := corpus.Source(demoSource)
	if src == nil {
		fmt.Fprintf(env.Stderr, "om demo: the content has no source %s\n", demoSource)
		return ExitUsage
	}
	for _, c := range []string{"de-mo", "de-mo-r1", "de-mo-r2", "de-mo-r3", "de-mo-r4"} {
		if corpus.Competition(c) == nil {
			fmt.Fprintf(env.Stderr, "om demo: the content has no competition %s\n", c)
			return ExitUsage
		}
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om demo: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	fc.Delay = 2 * time.Second
	ctx := context.Background()
	opt := fetch.Options{MaxAge: *maxAge}
	var files []string
	for g := 5; g <= 12; g++ {
		page, err := demoGet(ctx, fc, demoGradePage(g), opt)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om demo: %v\n", err)
			return ExitFailure
		}
		for _, m := range demoPDFRE.FindAllStringSubmatch(string(page), -1) {
			p, ok := official.DemoFile(m[1])
			if ok && p.Olympiad >= *from && p.Olympiad <= *to && !slices.Contains(files, m[1]) {
				files = append(files, m[1])
			}
		}
	}
	slices.Sort(files)
	code := ExitOK
	var reads []*demoRead
	byCode := map[string]*demoRead{}
	papers := map[string][]string{}
	for _, f := range files {
		paper, _ := official.DemoFile(f)
		u := demoSite + "pdf/block_a/" + f
		probs, err := demoPaper(ctx, fc, u, opt)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om demo: %s: %v\n", f, err)
			code = ExitFailure
			continue
		}
		if len(probs) == 0 {
			fmt.Fprintf(env.Stderr, "om demo: %s: no problem\n", f)
		}
		for _, p := range probs {
			pp, n, ok := official.DemoCode(p.Code)
			if !ok || pp != paper || byCode[p.Code] != nil {
				fmt.Fprintf(env.Stderr, "om demo: %s: problem %s is not a problem of the paper or is twice\n", f, p.Code)
				continue
			}
			flags := append(ocr.Check(p.Text), p.Flags...)
			if p.Figure {
				flags = append(flags, ocr.FlagFigure)
			}
			r := &demoRead{at: demoAt(paper, n), paper: paper, url: u, prob: p, flags: flags, conf: ocr.Confidence(flags)}
			reads = append(reads, r)
			byCode[p.Code] = r
			ed := r.at[:strings.LastIndex(r.at, "/")]
			if !slices.Contains(papers[ed], u) {
				papers[ed] = append(papers[ed], u)
			}
		}
	}
	// A heading "Aufgabe 341042 = 340943:" tells that the papers of two grades share a problem: one record has both positions.
	group := map[string]string{}
	var find func(string) string
	find = func(c string) string {
		if g, ok := group[c]; ok && g != c {
			r := find(g)
			group[c] = r
			return r
		}
		return c
	}
	for _, r := range reads {
		for _, s := range r.prob.Same {
			if byCode[s] == nil {
				continue
			}
			a, b := find(r.prob.Code), find(s)
			if a != b {
				group[max(a, b)] = min(a, b)
			}
		}
	}
	members := map[string][]*demoRead{}
	var roots []string
	for _, r := range reads {
		g := find(r.prob.Code)
		if members[g] == nil {
			roots = append(roots, g)
		}
		members[g] = append(members[g], r)
	}
	owners := official.Owners(corpus)
	allowed := textAllowed(src)
	type record struct {
		ms   []*demoRead
		text *demoRead
	}
	var news []record
	stats := map[string]*[4]int{}
	stat := func(at string) *[4]int {
		c := at[:strings.Index(at, "/")]
		if stats[c] == nil {
			stats[c] = &[4]int{}
		}
		return stats[c]
	}
	reasons := map[string]int{}
	linked, appeared := 0, 0
	for _, g := range roots {
		ms := slices.Clone(members[g])
		slices.SortStableFunc(ms, func(a, b *demoRead) int {
			if (a.paper.Round == 4 && a.paper.Grade == 12) != (b.paper.Round == 4 && b.paper.Grade == 12) {
				if a.paper.Round == 4 && a.paper.Grade == 12 {
					return -1
				}
				return 1
			}
			return cmp.Or(b.paper.Grade-a.paper.Grade, cmp.Compare(a.at, b.at))
		})
		best := ms[0]
		for _, m := range ms {
			if m.conf > best.conf {
				best = m
			}
		}
		if *explain {
			for _, m := range ms {
				fmt.Fprintf(env.Stdout, "%s\t%s\t%.2f\t%s\t%s\n", m.at, m.prob.Code, m.conf, strings.Join(m.flags, ","), strings.ReplaceAll(m.prob.Text, "\n", " ¶ "))
			}
		}
		var owned *content.Problem
		var free []string
		for _, m := range ms {
			if p := owners[m.at]; p != nil {
				if owned == nil {
					owned = p
				}
				continue
			}
			free = append(free, m.at)
		}
		if owned != nil {
			// A record at a position gets a link to the paper and the other positions; its text does not change.
			if slices.ContainsFunc(owned.Statements, func(s content.Statement) bool { return s.URL == best.url }) {
				continue
			}
			linked++
			appeared += len(free)
			stat(best.at)[3]++
			if !*write {
				continue
			}
			if err := demoLink(*root, owned, best, src, free); err != nil {
				fmt.Fprintf(env.Stderr, "om demo: %s: %v\n", owned.Path, err)
				code = ExitFailure
			}
			continue
		}
		news = append(news, record{ms: ms, text: best})
		s := stat(ms[0].at)
		if allowed && best.conf >= ocr.Threshold {
			s[0]++
		} else {
			s[1]++
			for _, f := range best.flags {
				reasons[f]++
			}
		}
		s[2] += len(ms) - 1
	}
	var cs []string
	for c := range stats {
		cs = append(cs, c)
	}
	slices.Sort(cs)
	for _, c := range cs {
		s := stats[c]
		fmt.Fprintf(env.Stdout, "%s: %d new records with text, %d link-only, %d more positions of them, %d links to existing records\n", c, s[0], s[1], s[2], s[3])
	}
	var rs []string
	for f, n := range reasons {
		rs = append(rs, fmt.Sprintf("%s %d", f, n))
	}
	slices.Sort(rs)
	fmt.Fprintf(env.Stdout, "%d papers, %d problems, %d records, %d links (%d more positions). The flags of the link-only records: %s\n", len(files), len(reads), len(news), linked, appeared, strings.Join(rs, ", "))
	if !*write {
		return code
	}
	tags, err := mintTags(corpus, len(news))
	if err != nil {
		fmt.Fprintf(env.Stderr, "om demo: %v\n", err)
		return ExitFailure
	}
	for i, n := range news {
		if err := demoWrite(*root, corpus, src, allowed, tags[i], n.ms, n.text); err != nil {
			fmt.Fprintf(env.Stderr, "om demo: %s: %v\n", n.ms[0].at, err)
			code = ExitFailure
		}
	}
	var eds []string
	for ed := range papers {
		eds = append(eds, ed)
	}
	slices.Sort(eds)
	for _, ed := range eds {
		if err := demoOfficial(*root, corpus, ed, papers[ed]); err != nil {
			fmt.Fprintf(env.Stderr, "om demo: %s: %v\n", ed, err)
			code = ExitFailure
		}
	}
	return code
}

// demoGet gets a URL through the fetch client, or from the raw store when its capture is young enough.
func demoGet(ctx context.Context, fc *fetch.Client, u string, opt fetch.Options) ([]byte, error) {
	res, err := fc.Get(ctx, demoSource, u, opt)
	if err != nil {
		return nil, err
	}
	rc, err := fc.Store.Get(ctx, res.Capture.Hash)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	var b bytes.Buffer
	_, err = b.ReadFrom(rc)
	return b.Bytes(), err
}

// demoPaper reads the problems of a paper PDF from the runs of text of pdftohtml, and finds its drawings on the pages that pdftoppm renders.
func demoPaper(ctx context.Context, fc *fetch.Client, u string, opt fetch.Options) ([]official.DemoProblem, error) {
	raw, err := demoGet(ctx, fc, u, opt)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "om-demo-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	f := filepath.Join(dir, "paper.pdf")
	if err := os.WriteFile(f, raw, 0o600); err != nil {
		return nil, err
	}
	b, err := figureRunner.Run(ctx, official.RuPDFTool, "-xml", "-i", "-stdout", "-q", f)
	if err != nil {
		return nil, err
	}
	items := official.ParsePDFXML(b)
	return official.DemoPDF(items, demoInk(ctx, f, items)), nil
}

// demoInk finds the ink of a page outside the runs of text: the dark pixels of the page at the scale of pdftohtml that no box of a run of text, a little larger, covers. A page that does not render has ink everywhere, so its problems are links only.
func demoInk(ctx context.Context, file string, items []official.PDFItem) official.DemoInk {
	boxes := map[int][]official.PDFItem{}
	for _, it := range items {
		boxes[it.Page] = append(boxes[it.Page], it)
	}
	rows := map[int][]int{}
	return func(page, top, bottom int) bool {
		r, ok := rows[page]
		if !ok {
			r = demoStray(ctx, file, page, boxes[page])
			rows[page] = r
		}
		if r == nil {
			return true
		}
		n := 0
		for y := max(top, 0); y < min(bottom, len(r)); y++ {
			n += r[y]
		}
		return n >= demoInkMin
	}
}

// demoStray gives the number of dark pixels outside the runs of text in each row of a page, or nil when the page does not render.
func demoStray(ctx context.Context, file string, page int, items []official.PDFItem) []int {
	p := strconv.Itoa(page)
	b, err := figureRunner.Run(ctx, "pdftoppm", "-r", strconv.Itoa(demoDPI), "-gray", "-f", p, "-l", p, "-singlefile", file)
	if err != nil {
		return nil
	}
	w, h, pix, ok := demoPGM(b)
	if !ok {
		return nil
	}
	mask := make([]bool, w*h)
	const pad = 3
	for _, it := range items {
		for y := max(it.Top-pad, 0); y < min(it.Top+it.Height+pad, h); y++ {
			for x := max(it.Left-pad, 0); x < min(it.Right()+pad, w); x++ {
				mask[y*w+x] = true
			}
		}
	}
	out := make([]int, h)
	for i, g := range pix {
		if g < 160 && !mask[i] {
			out[i/w]++
		}
	}
	// A row dark over half the width of the page is the rule under the header or above the footer, not a drawing.
	for y, n := range out {
		if n > w/2 {
			out[y] = 0
		}
	}
	return out
}

// demoPGM reads a binary PGM image with 8 bits for each pixel, as pdftoppm -gray gives it.
func demoPGM(b []byte) (w, h int, pix []byte, ok bool) {
	var fields []int
	i := 2
	if len(b) < 2 || string(b[:2]) != "P5" {
		return 0, 0, nil, false
	}
	for len(fields) < 3 && i < len(b) {
		for i < len(b) && (b[i] == ' ' || b[i] == '\n' || b[i] == '\r' || b[i] == '\t') {
			i++
		}
		j := i
		for j < len(b) && b[j] >= '0' && b[j] <= '9' {
			j++
		}
		n, err := strconv.Atoi(string(b[i:j]))
		if err != nil {
			return 0, 0, nil, false
		}
		fields = append(fields, n)
		i = j
	}
	if len(fields) < 3 || fields[2] != 255 || i >= len(b) {
		return 0, 0, nil, false
	}
	w, h = fields[0], fields[1]
	pix = b[i+1:]
	if w <= 0 || h <= 0 || len(pix) < w*h {
		return 0, 0, nil, false
	}
	return w, h, pix[:w*h], true
}

// demoCandidate is the statement of the paper of r.
func demoCandidate(r *demoRead, src *content.SourceRecord) *canon.Candidate {
	return &canon.Candidate{At: r.at, Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: "de", Name: demoName(r.paper), URL: r.url}
}

// demoWrite writes a new record for the problem of the papers ms, with the text of the paper of text when its confidence is high enough, or a link only.
func demoWrite(root string, corpus *content.Corpus, src *content.SourceRecord, allowed bool, tag string, ms []*demoRead, text *demoRead) error {
	at := ms[0].at
	comp := corpus.Competition(at[:strings.Index(at, "/")])
	s := canon.Scan{Page: text.prob.Page, Text: text.prob.Text, Extraction: "native/" + official.RuPDFTool + " (" + official.DemoVersion + ")", Confidence: text.conf}
	if !allowed || text.conf < ocr.Threshold {
		s = canon.Scan{Page: text.prob.Page}
	}
	_, rule := canon.Rank(&canon.Candidate{Origin: src.Origin, Lang: "de"}, original(comp))
	b, err := canon.RenderScan(at, tag, demoCandidate(text, src), s, rule)
	if err != nil {
		return err
	}
	rec := &canon.Record{Primary: canon.Choice{At: at}}
	var more []string
	for _, m := range ms[1:] {
		more = append(more, m.at)
		rec.Merged = append(rec.Merged, canon.Merge{At: m.at})
	}
	if len(more) > 0 {
		if b, err = canon.AddAppearances(b, more...); err != nil {
			return err
		}
	}
	return writeNew(root, corpus, rec, tag, b)
}

// demoLink adds to an existing record a link to the paper of r, with no text, and the positions free.
func demoLink(root string, p *content.Problem, r *demoRead, src *content.SourceRecord, free []string) error {
	full := filepath.Join(root, filepath.FromSlash(p.Path))
	b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	if b, err = canon.Apply(b, canon.AddLink(p, demoCandidate(r, src), r.prob.Page)); err != nil {
		return err
	}
	if len(free) > 0 {
		if b, err = canon.AddAppearances(b, free...); err != nil {
			return err
		}
	}
	return os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
}

// demoOfficial adds the paper PDFs urls to the official papers of an edition, as comp/year. The PDFs come from the archive, so a paper of de-mo, whose first source is the official site, names the source.
func demoOfficial(root string, corpus *content.Corpus, ed string, urls []string) error {
	if _, err := newEdition(root, corpus, ed+"/p1"); err != nil {
		return err
	}
	file := filepath.Join(root, "editions", filepath.FromSlash(ed)+".yaml")
	list, err := officialOf(file)
	if err != nil {
		return err
	}
	comp := corpus.Competition(ed[:strings.Index(ed, "/")])
	source := ""
	if len(comp.Sources) > 0 && comp.Sources[0] != demoSource {
		source = demoSource
	}
	n := len(list)
	for _, u := range urls {
		if !slices.ContainsFunc(list, func(o content.OfficialPaper) bool { return o.URL == u }) {
			list = append(list, content.OfficialPaper{Lang: "German", Code: "de", URL: u, Source: source})
		}
	}
	if len(list) == n {
		return nil
	}
	return setOfficial(file, list)
}
