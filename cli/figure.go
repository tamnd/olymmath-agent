package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/figure"
	"github.com/tamnd/olymmath-agent/pdfsrc"
)

func init() {
	register(command{"figure", "crop the figure of a problem from its official PDF and save it next to the problem (om figure -after \"2.\" -before \"3.\" pt-opm/2008/p2)", runFigure})
}

// figureRunner runs the poppler tools for om figure. A test puts a fake in its place.
var figureRunner pdfsrc.Runner = pdfsrc.Exec

// figureOptions are the flags of om figure.
type figureOptions struct {
	root, file, after, before, box, format, alt, work string
	page, n, dpi                                      int
	insert, dry, erase                                bool
}

const figureUsage = `usage: om figure [-file PDF] [-page N] [-after TEXT] [-before TEXT] [-box X0,Y0,X1,Y1] [-erase] [-format auto|svg|png] [-n N] [-insert] [-dry] LOCATOR
LOCATOR is a problem such as pt-opm/2008/p2. The PDF is the URL and the page of the canonical statement when -file and -page are not given.
Without -box, om figure looks for the drawing between the line where the text -after starts and the line where the text -before starts: the ink of the page that no word of the text layer covers, with the short labels next to it.
The figure is saved as p2.fig1.svg next to the problem, or as a PNG when the figure is a raster image. With -insert, the statement gets the Markdown image after its text.`

func runFigure(env Env, args []string) int {
	fs := newFlags("figure", env.Stderr)
	var o figureOptions
	fs.StringVar(&o.root, "content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	fs.StringVar(&o.file, "file", "", "a local copy of the PDF. Then om figure does not get the URL of the statement")
	fs.IntVar(&o.page, "page", 0, "the page of the PDF with the figure. It is the page of the canonical statement when it is 0")
	fs.StringVar(&o.after, "after", "", "a text of the page at the top of the band where the figure is, such as the start of the problem")
	fs.StringVar(&o.before, "before", "", "a text of the page under the figure, such as the start of the next problem. The band ends at the bottom of the page when it is not given")
	fs.StringVar(&o.box, "box", "", "the box of the figure in points from the top left corner of the page, as x0,y0,x1,y1. It replaces the search of -after and -before")
	fs.StringVar(&o.format, "format", figure.Auto, "the format of the file: auto (SVG for a drawing, PNG for a raster image), svg or png")
	fs.IntVar(&o.dpi, "dpi", 200, "the resolution of a PNG figure")
	fs.IntVar(&o.n, "n", 0, "the number of the figure, as in p2.fig2.svg. It is the next free number when it is 0")
	fs.StringVar(&o.alt, "alt", "", "the alternative text of the image. It is \"Figure N\" when it is not given")
	fs.BoolVar(&o.insert, "insert", false, "add the Markdown image to the end of the canonical statement when it does not have it")
	fs.BoolVar(&o.dry, "dry", false, "find the box and print it, and write no file")
	fs.BoolVar(&o.erase, "erase", false, "take the words of the text of the problem out of the figure: the words in the box on lines that are at least a third of the page wide. The labels of the figure are short lines, so they stay")
	fs.StringVar(&o.work, "work", filepath.Join(os.TempDir(), "om-figure"), "the directory for the PDF of a URL. om figure deletes the PDF when it ends")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 {
		fmt.Fprintln(env.Stderr, figureUsage)
		return ExitUsage
	}
	if o.format != figure.Auto && o.format != figure.SVG && o.format != figure.PNG {
		fmt.Fprintf(env.Stderr, "om figure: -format %q is not auto, svg or png\n", o.format)
		return ExitUsage
	}
	corpus, findings := content.Load(o.root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om figure: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	p := problemAt(corpus, pos[0])
	if p == nil {
		fmt.Fprintf(env.Stderr, "om figure: %s is not a problem\n", pos[0])
		return ExitUsage
	}
	st := p.CanonicalStatement()
	if st == nil {
		fmt.Fprintf(env.Stderr, "om figure: %s has no canonical statement\n", p.Path)
		return ExitUsage
	}
	// A figure is part of the statement, so it needs a source whose terms allow the statement.
	src := corpus.Source(st.Source)
	if src == nil || !textAllowed(src) {
		fmt.Fprintf(env.Stderr, "om figure: the terms of the source %q do not allow the statement in the content, so they do not allow its figure\n", st.Source)
		return ExitFindings
	}
	page := o.page
	if page == 0 {
		page = max(st.Page, 1)
	}
	ctx := context.Background()
	pdf := o.file
	if pdf == "" {
		if st.URL == "" {
			fmt.Fprintf(env.Stderr, "om figure: the statement of %s has no URL. Give -file\n", p.Path)
			return ExitUsage
		}
		data, err := ocrBytes(ctx, env, ocrOptions{}, *dir, *remote, st.Source, st.URL)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
			return ExitFailure
		}
		if err := os.MkdirAll(o.work, 0o750); err != nil {
			fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
			return ExitFailure
		}
		f, err := os.CreateTemp(o.work, "paper-*.pdf")
		if err != nil {
			fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
			return ExitFailure
		}
		pdf = f.Name()
		defer func() { _ = os.Remove(pdf) }()
		_, werr := f.Write(data)
		if cerr := f.Close(); werr == nil {
			werr = cerr
		}
		if werr != nil {
			fmt.Fprintf(env.Stderr, "om figure: %v\n", werr)
			return ExitFailure
		}
	}
	var box figure.Box
	if o.box != "" {
		box, err = figure.ParseBox(o.box)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
			return ExitUsage
		}
	} else {
		box, err = figure.Locate(ctx, figureRunner, pdf, page, o.after, o.before, figure.Options{})
		if err != nil {
			fmt.Fprintf(env.Stderr, "om figure: page %d: %v. Give -box\n", page, err)
			return ExitFindings
		}
	}
	if o.dry {
		fmt.Fprintf(env.Stdout, "%s: page %d, box %s (%.0f x %.0f pt). -dry: no file written.\n", p.Path, page, box, box.W(), box.H())
		return ExitOK
	}
	var erase []figure.Box
	if o.erase {
		erase, err = textWords(ctx, pdf, page, box)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
			return ExitFailure
		}
	}
	fig, err := figure.Crop(ctx, figureRunner, pdf, page, box, o.format, o.dpi, erase)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
		return ExitFailure
	}
	rel, err := saveFigure(o.root, p, o.n, fig)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
		return ExitFailure
	}
	n := figureNumber(rel)
	alt := o.alt
	if alt == "" {
		alt = "Figure " + strconv.Itoa(n)
	}
	ref := fmt.Sprintf("![%s](%s)", alt, path.Base(rel))
	fmt.Fprintf(env.Stdout, "%s: page %d, box %s, %s, %d bytes\n%s\n", rel, page, fig.Box, fig.Format, len(fig.Data), ref)
	if o.insert {
		changed, err := insertFigure(o.root, p, ref)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om figure: %v\n", err)
			return ExitFailure
		}
		if changed {
			fmt.Fprintf(env.Stdout, "%s: the statement shows the figure now\n", p.Path)
		}
	}
	return ExitOK
}

// textWords gives the words of the text of the problem in the box on the page of the PDF.
func textWords(ctx context.Context, pdf string, page int, box figure.Box) ([]figure.Box, error) {
	layout, err := pdfsrc.Layout(ctx, figureRunner, pdf, page)
	if err != nil {
		return nil, err
	}
	p, err := figure.ParseLayout(layout)
	if err != nil {
		return nil, err
	}
	return p.TextWords(box.Out()), nil
}

// problemAt gives the problem at a locator such as pt-opm/2008/p2, or with a tag.
func problemAt(corpus *content.Corpus, at string) *content.Problem {
	at = strings.Trim(at, "/")
	if p := corpus.ByTag(at); p != nil {
		return p
	}
	for _, p := range corpus.All() {
		if p.Locator() == at || p.Path == "problems/"+at+".md" {
			return p
		}
	}
	return nil
}

// figureName gives the path from the root of the content of figure n of the problem in the format, as problems/pt-opm/2008/p2.fig1.svg.
func figureName(p *content.Problem, n int, format string) string {
	return strings.TrimSuffix(p.Path, ".md") + ".fig" + strconv.Itoa(n) + "." + format
}

// figureNumber gives the number of a figure file from its name.
func figureNumber(rel string) int {
	name := strings.TrimSuffix(path.Base(rel), path.Ext(rel))
	n, _ := strconv.Atoi(name[strings.LastIndex(name, ".fig")+4:])
	return n
}

// saveFigure writes the figure next to the problem as figure n, or as the next free number when n is 0. A file of figure n in the other format goes, so a figure has one file.
func saveFigure(root string, p *content.Problem, n int, fig figure.Figure) (string, error) {
	exists := func(rel string) bool {
		_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
		return err == nil
	}
	if n <= 0 {
		n = 1
		for exists(figureName(p, n, figure.SVG)) || exists(figureName(p, n, figure.PNG)) {
			n++
		}
	}
	for _, f := range []string{figure.SVG, figure.PNG} {
		if rel := figureName(p, n, f); f != fig.Format && exists(rel) {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
				return "", err
			}
		}
	}
	rel := figureName(p, n, fig.Format)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(rel)), fig.Data, 0o644); err != nil { // #nosec G306 -- the content is public.
		return "", err
	}
	return rel, nil
}

// insertFigure adds the Markdown image ref to the end of the canonical statement of the problem file, before the other statements, when the statement does not show the file yet. It tells if the file changed.
func insertFigure(root string, p *content.Problem, ref string) (bool, error) {
	file := filepath.Join(root, filepath.FromSlash(p.Path))
	b, err := os.ReadFile(file) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return false, err
	}
	target := ref[strings.LastIndex(ref, "](")+2 : len(ref)-1]
	for _, r := range content.FigureRefs(p.Statement) {
		if path.Base(r) == target {
			return false, nil
		}
	}
	out, err := content.AddToStatement(b, p.Canonical.Statement, p.Canonical.Statement, ref)
	if err != nil {
		return false, fmt.Errorf("%s: %w", p.Path, err)
	}
	return true, os.WriteFile(file, out, 0o644) // #nosec G306 G703 -- the content is public, and the path is inside the content root.
}
