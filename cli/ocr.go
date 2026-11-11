package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/lane"
	"github.com/tamnd/olymmath-agent/locator"
	"github.com/tamnd/olymmath-agent/ocr"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/site"
)

func init() {
	register(command{"ocr", "read an official paper with a vision model, or its text layer with -lanes native, and make a problem for each position of the edition that has no problem (om ocr -source om-sem pl-om/1997 URL)", runOCR})
}

// ocrOptions are the flags of om ocr.
type ocrOptions struct {
	root, source, paper, lang, file, pages, work string
	images                                       string
	lanes, dpi, local                            string
	limit                                        int
	dry, sequence, check                         bool
}

func runOCR(env Env, args []string) int {
	fs := newFlags("ocr", env.Stderr)
	var o ocrOptions
	fs.StringVar(&o.root, "content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	fs.StringVar(&o.source, "source", "", "the source of the paper in registers/sources.yaml. It is the source of the official paper of the edition when it is not given")
	fs.StringVar(&o.paper, "paper", "", "the paper of the layout of the competition that the PDF holds, such as day1. The problems of a paper that numbers from 1 get the positions of that paper")
	fs.StringVar(&o.lang, "lang", "", "the language code of the paper. It is the code of the official paper of the edition when it is not given")
	fs.StringVar(&o.file, "file", "", "a local copy of the PDF at the URL. Then om ocr does not get the URL")
	fs.StringVar(&o.images, "images", "", "a directory of image files to read instead of a PDF, in the order of their names. Each image is a part of a page that holds one problem, as the images of a dataset. The name of each file without its extension names the problem in the source, such as the id of the row")
	fs.StringVar(&o.pages, "pages", "", "the pages to read, such as 1-2,4. All the pages up to -limit when it is not given")
	fs.IntVar(&o.limit, "limit", ocr.DefaultLimit, "the largest number of pages to read when -pages is not given")
	fs.StringVar(&o.work, "work", filepath.Join(os.TempDir(), "om-ocr"), "the directory for the PDF and the page images. om ocr deletes them when it ends")
	fs.StringVar(&o.lanes, "lanes", "codex,gemini,claude", "the model clients to ask, in order, or local for the vision servers of -local, or native for the text layer of a born-digital PDF, with its math put into TeX (no model)")
	fs.StringVar(&o.local, "local", envOr("OLYMMATH_LOCAL", lane.DefaultLocal), "the targets of the local lane, as a comma-separated list of MODEL=URL. URL is a base URL such as http://127.0.0.1:8079/v1, or ssh://HOST/URL for a server on the loopback address of an ssh host. $OLYMMATH_LOCAL sets the default")
	fs.BoolVar(&o.check, "check", false, "ask each target of -local for one token, print which targets answer, and read no page")
	fs.StringVar(&o.dpi, "dpi", "200,300", "the resolutions of the page images. A page whose text has a flag is read again at the next one")
	fs.BoolVar(&o.dry, "dry", false, "print the problems and write no file")
	fs.BoolVar(&o.sequence, "sequence", false, "number the problems 1, 2, 3 in the order of the paper, for a paper whose days each start at problem 1")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if o.check {
		return checkLocal(env, o.local)
	}
	if len(pos) < 1 || len(pos) > 2 {
		fmt.Fprintln(env.Stderr, "usage: om ocr [-source SOURCE] [-paper PAPER] [-pages PAGES] [-file FILE | -images DIR] [-lanes LANES] [-local TARGETS] [-dry] EDITION [URL]\n       om ocr -check [-local TARGETS]\nEDITION is an edition such as pl-om/1997. URL is the address of the official paper. It is the official paper of the edition from the source when it is not given, or the address of the source for the images of a dataset.\nLANES is codex,gemini,claude by default. -lanes local reads with the vision servers of -local, which by default are "+lane.DefaultLocal+". -lanes native reads the text layer of the PDF and puts its math into TeX, with no model.")
		return ExitUsage
	}
	if o.images != "" && (o.file != "" || o.pages != "") {
		fmt.Fprintln(env.Stderr, "om ocr: -images reads image files, so it takes no -file and no -pages")
		return ExitUsage
	}
	native := strings.TrimSpace(o.lanes) == ocr.NativeLane
	if native && o.images != "" {
		fmt.Fprintln(env.Stderr, "om ocr: -lanes native reads the text layer of a PDF, so it takes no -images")
		return ExitUsage
	}
	corpus, findings := content.Load(o.root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om ocr: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	e, err := findEdition(corpus, pos[0])
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitUsage
	}
	if s := corpus.Source(o.source); o.images != "" && len(pos) == 1 && s != nil && s.Kind == "dataset" {
		pos = append(pos, s.URL)
	}
	paper, err := ocrPaper(e, o, pos[1:])
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitUsage
	}
	src := corpus.Source(paper.Source)
	if src == nil {
		fmt.Fprintf(env.Stderr, "om ocr: %q is not in the source register\n", paper.Source)
		return ExitUsage
	}
	pages, err := parsePages(o.pages)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitUsage
	}
	dpi, err := parseInts(o.dpi)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: -dpi: %v\n", err)
		return ExitUsage
	}
	ctx := context.Background()
	var read []ocr.Page
	if native {
		n := &ocr.Native{Version: ocr.PdftotextVersion(ctx), Limit: o.limit, Log: env.Stderr}
		read, err = readPDF(ctx, env, o, *dir, *remote, src.ID, paper.URL, pages, n.Read)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om ocr: %s: %v\n", e.Locator(), err)
			return ExitFailure
		}
		return finishOCR(env, o, corpus, e, src, paper, read)
	}
	lanes, variants, err := ocrLanes(o)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitUsage
	}
	cache, err := ocr.DefaultCache()
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitFailure
	}
	chain := &lane.Chain{Lanes: lanes, State: filepath.Join(cache.Dir, "lanes.json")}
	if o.images != "" {
		files, ferr := imageFiles(o.images)
		if ferr != nil {
			fmt.Fprintf(env.Stderr, "om ocr: -images: %v\n", ferr)
			return ExitUsage
		}
		r := &ocr.Reader{Asker: chain, Cache: cache, Log: env.Stderr, Variants: variants}
		read, err = r.ReadImages(ctx, files)
	} else {
		r := &ocr.Reader{Asker: chain, Cache: cache, DPI: dpi, Limit: o.limit, Log: env.Stderr, Variants: variants}
		read, err = readPDF(ctx, env, o, *dir, *remote, src.ID, paper.URL, pages, func(ctx context.Context, file string, pages []int) ([]ocr.Page, error) {
			r.Work = filepath.Dir(file)
			return r.Read(ctx, file, pages)
		})
	}
	if errors.Is(err, lane.ErrExhausted) {
		// The batch stops here. The pages that were read are in the cache, so the next run asks only for the others.
		fmt.Fprintf(env.Stderr, "om ocr: %s: %v\nThe pages read so far are in the cache. Run again when a lane has its allowance.\n", e.Locator(), err)
		return ExitLater
	}
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %s: %v\n", e.Locator(), err)
		return ExitFailure
	}
	return finishOCR(env, o, corpus, e, src, paper, read)
}

// finishOCR cuts the pages into problems, prints them, and writes the new problems unless -dry.
func finishOCR(env Env, o ocrOptions, corpus *content.Corpus, e *content.Edition, src *content.SourceRecord, paper content.OfficialPaper, read []ocr.Page) int {
	markup, err := site.NewMarkup(nil)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitFailure
	}
	news := planScans(corpus, e, paper, read, o.sequence, textAllowed(src), markup)
	if o.images != "" {
		// A crop is not a page of the paper, so the statement gives no page.
		for i := range news {
			news[i].scan.Page = 0
		}
	}
	for _, n := range news {
		where := fmt.Sprintf("page %d", n.scan.Page)
		if n.ref != "" {
			where = n.ref
		}
		fmt.Fprintf(env.Stdout, "%s: %s, %s\n", n.at, where, n.why)
	}
	if o.dry {
		for _, n := range news {
			if n.scan.Text != "" {
				fmt.Fprintf(env.Stdout, "\n--- %s\n%s\n", n.at, n.scan.Text)
			}
		}
		fmt.Fprintf(env.Stdout, "%d new problems. -dry: no file written.\n", len(news))
		return ExitOK
	}
	if err := writeScans(o.root, corpus, e, src, paper, news); err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d new problems.\n", len(news))
	return ExitOK
}

// findEdition gives the edition of a locator such as pl-om/1997.
func findEdition(corpus *content.Corpus, at string) (*content.Edition, error) {
	l, err := locator.Parse(strings.Trim(at, "/"))
	if err != nil {
		return nil, err
	}
	c := corpus.Competition(l.Comp)
	if c == nil {
		return nil, fmt.Errorf("%q is not a competition", l.Comp)
	}
	for _, e := range c.Editions {
		if e.ID == l.Edition {
			return e, nil
		}
	}
	return nil, fmt.Errorf("%s has no edition file. Make it first", at)
}

// ocrPaper gives the official paper to read: the URL of the arguments, or the official paper of the edition. The flags fill the fields that the edition does not give.
func ocrPaper(e *content.Edition, o ocrOptions, urls []string) (content.OfficialPaper, error) {
	var p content.OfficialPaper
	for _, q := range e.Official {
		if (len(urls) == 0 && (o.source == "" || q.Source == o.source) && q.Paper == o.paper) || (len(urls) > 0 && q.URL == urls[0]) {
			p = q
			break
		}
	}
	if len(urls) > 0 {
		p.URL = urls[0]
	}
	if p.URL == "" {
		return p, fmt.Errorf("%s has no official paper%s. Give its URL", e.Locator(), map[bool]string{true: " " + o.paper, false: ""}[o.paper != ""])
	}
	if o.source != "" {
		p.Source = o.source
	}
	if o.paper != "" {
		p.Paper = o.paper
	}
	if o.lang != "" {
		p.Code = o.lang
	}
	if p.Source == "" {
		if len(e.Competition.Sources) == 0 {
			return p, fmt.Errorf("give the source of %s with -source", p.URL)
		}
		p.Source = e.Competition.Sources[0]
	}
	if p.Code == "" {
		p.Code = "en"
	}
	if p.Lang == "" {
		p.Lang = languageNames[p.Code]
	}
	return p, nil
}

// languageNames are the names of the languages of the papers that om ocr reads most. A paper in another language gets its name from the edition, or has none.
var languageNames = map[string]string{"en": "English", "pl": "Polish", "de": "German", "sr": "Serbian", "cs": "Czech", "sk": "Slovak", "hu": "Hungarian", "ro": "Romanian", "ru": "Russian", "uk": "Ukrainian", "fr": "French", "es": "Spanish", "it": "Italian", "pt": "Portuguese", "bg": "Bulgarian", "hr": "Croatian", "sl": "Slovenian", "tr": "Turkish", "vi": "Vietnamese", "nl": "Dutch", "da": "Danish", "sv": "Swedish", "no": "Norwegian", "fi": "Finnish", "et": "Estonian", "lv": "Latvian", "lt": "Lithuanian", "ga": "Irish"}

// ocrBytes gives the bytes of the PDF: the local file, or the URL through the fetch client, which keeps robots.txt and the crawl delay and puts the PDF in the raw store.
func ocrBytes(ctx context.Context, env Env, o ocrOptions, dir string, remote bool, source, url string) ([]byte, error) {
	if o.file != "" {
		return os.ReadFile(o.file)
	}
	st, err := openStore(dir, remote)
	if err != nil {
		return nil, err
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	res, err := fc.Get(ctx, source, url, fetch.Options{})
	if err != nil {
		return nil, err
	}
	rc, err := st.Get(ctx, res.Capture.Hash)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

// textAllowed tells if the content may hold the text of a source: the zone of its statements is Z1 or Z2, the holder did not ask that no text is published, and the terms of the register do not say that the content holds no text of it.
func textAllowed(src *content.SourceRecord) bool {
	z := src.Zone.Statement
	return (z == "Z1" || z == "Z2") && !src.DoNotRepublish && !strings.Contains(strings.ToLower(src.Terms), "holds no text")
}

// newScan is a new problem from a scanned paper.
type newScan struct {
	at   string
	scan canon.Scan
	why  string
	// ref names the problem in the source: the name of its image file without the extension. It is "" for a page of a PDF.
	ref string
}

// planScans gives a new problem for each problem of the pages at a position of the edition that has no problem. A problem holds its text when the source allows it, the checks find nothing, and the text renders. Otherwise it only links to its page.
func planScans(corpus *content.Corpus, e *content.Edition, paper content.OfficialPaper, pages []ocr.Page, sequence, allowed bool, markup *site.Markup) []newScan {
	probs := ocr.Split(pages)
	if sequence {
		probs = ocr.Sequence(pages)
	}
	segs := make([]official.Segment, len(probs))
	for i, p := range probs {
		segs[i] = official.Segment{Position: "p" + strconv.Itoa(p.Number), Page: p.Page}
	}
	shape := e.Competition.Shape(e.Year)
	official.Place(shape, paper.Paper, segs)
	positions := shape.Positions()
	owners := official.Owners(corpus)
	byPage := map[int]ocr.Page{}
	for _, p := range pages {
		byPage[p.N] = p
	}
	var out []newScan
	for i, p := range probs {
		pos := segs[i].Position
		at := e.Locator() + "/" + pos
		if pos == "" || !slices.Contains(positions, pos) || owners[at] != nil {
			continue
		}
		pg := byPage[p.Page]
		flags := slices.Clone(p.Flags)
		for _, f := range pg.Flags {
			// A refusal and a broken font are flags of the whole page, so each problem of the page has them.
			if (f == ocr.FlagRefusal || f == ocr.FlagEncoding) && !slices.Contains(flags, f) {
				flags = append(flags, f)
			}
		}
		if _, errs := markup.HTML(p.Text); len(errs) > 0 {
			flags = append(flags, "render")
		}
		s := canon.Scan{Page: p.Page, Text: p.Text, Extraction: pg.Extraction(), Confidence: ocr.Confidence(flags)}
		why := fmt.Sprintf("text, confidence %.2f", s.Confidence)
		switch {
		case !allowed:
			s.Text, why = "", "a link only: the source allows no text"
		case s.Confidence < ocr.Threshold:
			s.Text, why = "", fmt.Sprintf("a link only: the checks found %s", strings.Join(flags, ", "))
		}
		if s.Text == "" {
			s.Extraction, s.Confidence = "", 0
		}
		ref := ""
		if pg.File != "" {
			ref = strings.TrimSuffix(filepath.Base(pg.File), filepath.Ext(pg.File))
		}
		out = append(out, newScan{at: at, scan: s, why: why, ref: ref})
	}
	return out
}

// writeScans writes the new problems, and it adds the paper to the official papers of the edition when the edition does not list it.
func writeScans(root string, corpus *content.Corpus, e *content.Edition, src *content.SourceRecord, paper content.OfficialPaper, news []newScan) error {
	if len(news) == 0 {
		return nil
	}
	tags, err := mintTags(corpus, len(news))
	if err != nil {
		return err
	}
	c := e.Competition
	name := fmt.Sprintf("%s %d official paper", c.Name, e.Year)
	dataset := src.Kind == "dataset"
	for i, n := range news {
		cd := &canon.Candidate{Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: paper.Code, Name: name, URL: paper.URL}
		if dataset {
			// The text is a row of a dataset, not the official paper: the statement names the dataset and the row.
			cd.Name, cd.Ref = src.Name, n.ref
			if n.ref != "" {
				cd.Name = src.Name + ", " + n.ref
			}
		}
		_, rule := canon.Rank(&canon.Candidate{Origin: src.Origin, Lang: paper.Code}, original(c))
		b, err := canon.RenderScan(n.at, tags[i], cd, n.scan, rule)
		if err != nil {
			return err
		}
		if err := writeNew(root, corpus, &canon.Record{Primary: canon.Choice{At: n.at}}, tags[i], b); err != nil {
			return fmt.Errorf("%s: %w", n.at, err)
		}
	}
	if dataset || slices.ContainsFunc(e.Official, func(p content.OfficialPaper) bool { return p.URL == paper.URL }) {
		return nil
	}
	file := filepath.Join(root, "editions", c.Code, e.ID+".yaml")
	return setOfficial(file, append(slices.Clone(e.Official), paper))
}

// parsePages reads a list of pages such as 1-2,4.
func parsePages(s string) ([]int, error) {
	var out []int
	for _, f := range strings.Split(s, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		a, b, isRange := strings.Cut(f, "-")
		lo, err := strconv.Atoi(a)
		hi := lo
		if err == nil && isRange {
			hi, err = strconv.Atoi(b)
		}
		if err != nil || lo < 1 || hi < lo {
			return nil, fmt.Errorf("%q is not a page or a range of pages", f)
		}
		for n := lo; n <= hi; n++ {
			out = append(out, n)
		}
	}
	return out, nil
}

func parseInts(s string) ([]int, error) {
	var out []int
	for _, f := range strings.Split(s, ",") {
		n, err := strconv.Atoi(strings.TrimSpace(f))
		if err != nil {
			return nil, fmt.Errorf("%q is not a number", f)
		}
		out = append(out, n)
	}
	return out, nil
}

// readPDF gets the PDF, from the local file or the URL, writes it into a new work directory, and reads its pages with read.
func readPDF(ctx context.Context, env Env, o ocrOptions, dir string, remote bool, source, url string, pages []int, read func(ctx context.Context, file string, pages []int) ([]ocr.Page, error)) ([]ocr.Page, error) {
	pdf, err := ocrBytes(ctx, env, o, dir, remote, source, url)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", url, err)
	}
	if err := os.MkdirAll(o.work, 0o750); err != nil {
		return nil, err
	}
	work, err := os.MkdirTemp(o.work, "run-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(work) }()
	file := filepath.Join(work, "paper.pdf")
	if err := os.WriteFile(file, pdf, 0o600); err != nil {
		return nil, err
	}
	return read(ctx, file, pages)
}

// imageExts are the extensions of the image files that -images reads.
var imageExts = []string{".png", ".jpg", ".jpeg", ".webp"}

// imageFiles gives the image files of the directory, in the order of their names.
func imageFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && slices.Contains(imageExts, strings.ToLower(filepath.Ext(e.Name()))) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%s has no image file", dir)
	}
	return out, nil
}

// ocrLanes gives the lanes of -lanes, and the variants of the cache for them. The local lane runs alone: its answers are kept apart in the cache under each model (ocr.Reader.Variants), and the key of a page must be known before a lane is asked.
func ocrLanes(o ocrOptions) ([]lane.Named, []string, error) {
	names := strings.Split(o.lanes, ",")
	for i := range names {
		names[i] = strings.TrimSpace(names[i])
	}
	if !slices.Contains(names, lane.LocalName) {
		l, err := lane.Select(lane.Defaults(), o.lanes)
		return l, nil, err
	}
	if len(names) > 1 {
		return nil, nil, fmt.Errorf("-lanes: the local lane runs alone, so run the other lanes in another run")
	}
	l, err := lane.NewLocal(o.local)
	if err != nil {
		return nil, nil, err
	}
	var variants []string
	for _, m := range l.Models() {
		variants = append(variants, lane.LocalName+"/"+m)
	}
	return []lane.Named{l}, variants, nil
}

// checkLocal runs the health check of each target of the local lane.
func checkLocal(env Env, spec string) int {
	l, err := lane.NewLocal(spec)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om ocr: %v\n", err)
		return ExitUsage
	}
	code := ExitFailure
	for i, err := range l.Check(context.Background()) {
		if err != nil {
			fmt.Fprintf(env.Stdout, "%s: %v\n", l.Targets[i].Name(), err)
			continue
		}
		fmt.Fprintf(env.Stdout, "%s: answers\n", l.Targets[i].Name())
		code = ExitOK
	}
	return code
}

// envOr gives the value of the environment variable, or def when it is not set.
func envOr(name, def string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return def
}
