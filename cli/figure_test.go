package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakePoppler map[string]string

func (f fakePoppler) Run(_ context.Context, name string, _ ...string) ([]byte, error) {
	if o, ok := f[name]; ok {
		return []byte(o), nil
	}
	return nil, errors.New(name + ": no output")
}

func TestFigure(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../content/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="40pt" height="30pt"><path d="M0 0L40 30"/></svg>`
	old := figureRunner
	figureRunner = fakePoppler{"pdftocairo": svg}
	defer func() { figureRunner = old }()
	pdf := filepath.Join(root, "paper.pdf")
	if err := os.WriteFile(pdf, []byte("%PDF"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "problems", "imo", "2024")

	code, out, errs := run("figure", "-content", root, "-file", pdf, "-box", "10,10,50,40", "-dry", "imo/2024/p2")
	if code != ExitOK || !strings.Contains(out, "box 10.0,10.0,50.0,40.0") {
		t.Fatalf("-dry: %d %q %q", code, out, errs)
	}
	if _, err := os.Stat(filepath.Join(dir, "p2.fig1.svg")); err == nil {
		t.Fatal("-dry wrote the figure")
	}

	code, out, errs = run("figure", "-content", root, "-file", pdf, "-box", "10,10,50,40", "-insert", "imo/2024/p2")
	if code != ExitOK || !strings.Contains(out, "![Figure 1](p2.fig1.svg)") {
		t.Fatalf("%d %q %q", code, out, errs)
	}
	if b, err := os.ReadFile(filepath.Join(dir, "p2.fig1.svg")); err != nil || string(b) != svg {
		t.Fatalf("the figure: %q %v", b, err)
	}
	md, err := os.ReadFile(filepath.Join(dir, "p2.md"))
	if err != nil || !strings.HasSuffix(string(md), "holds for all integers $n \\geq N$.\n\n![Figure 1](p2.fig1.svg)\n") {
		t.Fatalf("the problem: %s %v", md, err)
	}
	// The same figure again as number 1 replaces the file, and the statement keeps one ref.
	if code, _, errs := run("figure", "-content", root, "-file", pdf, "-box", "10,10,50,40", "-n", "1", "-insert", "03JXB69J3N"); code != ExitOK {
		t.Fatalf("%d %q", code, errs)
	}
	if md2, _ := os.ReadFile(filepath.Join(dir, "p2.md")); string(md2) != string(md) {
		t.Errorf("the problem changed: %s", md2)
	}
	// A new figure takes the next number.
	if code, out, _ := run("figure", "-content", root, "-file", pdf, "-box", "10,10,50,40", "imo/2024/p2"); code != ExitOK || !strings.Contains(out, "p2.fig2.svg") {
		t.Errorf("the second figure: %d %q", code, out)
	}
	if code, _, _ := run("figure", "-content", root, "-file", pdf, "-box", "1,2,3", "imo/2024/p2"); code != ExitUsage {
		t.Errorf("a bad box gives %d", code)
	}
	if code, _, _ := run("figure", "-content", root, "-file", pdf, "-box", "10,10,50,40", "imo/2024/p9"); code != ExitUsage {
		t.Errorf("a problem that is not there gives %d", code)
	}
}

func TestFigureTerms(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../content/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "registers", "sources.yaml")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), "terms: Copyright of the International Mathematical Olympiad. No licence is stated.", "terms: Copyright of the IMO. The register holds no text of it.", 1))
	if err := os.WriteFile(file, b, 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errs := run("figure", "-content", root, "-file", "x.pdf", "-box", "10,10,50,40", "imo/2024/p2")
	if code != ExitFindings || !strings.Contains(errs, "do not allow") {
		t.Errorf("%d %q", code, errs)
	}
}
