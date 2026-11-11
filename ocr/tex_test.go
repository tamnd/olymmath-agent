package ocr

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

func TestTeX(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Let x ≤ y be reals.", "Let $x \\le y$ be reals."},
		{"Prove that a² + b² ≥ 2ab.", "Prove that $a^2 + b^2 \\ge 2ab$."},
		{"Find all n ≥ 2 such that 2ⁿ⁺¹ − 1 is prime.", "Find all $n \\ge 2$ such that $2^{n+1} - 1$ is prime."},
		{"The sum x₁ + x₂ + … + xₙ equals ½.", "The sum $x_1 + x_2 + \\ldots + x_n$ equals $\\frac{1}{2}$."},
		{"Let α, β be the angles with ∠ABC = 60°.", "Let $\\alpha$, $\\beta$ be the angles with $\\angle ABC = 60^\\circ$."},
		{"Show that √(n+1) − √n < 1⁄2.", "Show that $\\sqrt{n+1} - \\sqrt{n} < \\frac{1}{2}$."},
		{"f : ℝ → ℝ satisfies f(x+y) = f(x)f(y).", "f : $\\mathbb{R} \\to \\mathbb{R}$ satisfies $f(x+y) = f(x)f(y)$."},
		{"A 5 × 5 board.", "A $5 \\times 5$ board."},
		{"Let 𝑥 and 𝑦 be integers.", "Let $x$ and $y$ be integers."},
		{"αx = 1", "$\\alpha x = 1$"},
		// Prose stays as it is.
		{"In 2024, there were 3 problems and so on…", "In 2024, there were 3 problems and so on…"},
		{"The students and/or teachers.", "The students and/or teachers."},
		{"Η άσκηση είναι εύκολη.", "Η άσκηση είναι εύκολη."},
		{"Αν α ≤ β, τότε", "Αν $α \\le β$, τότε"},
		// Math between dollars stays.
		{"Let $x \\le 1$ and y ≥ 0.", "Let $x \\le 1$ and $y \\ge 0$."},
		{"$$\na² ≤ b\n$$\nso c ≤ d", "$$\na² ≤ b\n$$\nso $c \\le d$"},
		{"Only one $ here ≤", "Only one $ here ≤"},
		{"It costs 50% more.", "It costs 50% more."},
	}
	for _, c := range cases {
		if got := TeX(c.in); got != c.want {
			t.Errorf("TeX(%q)\n got %q\nwant %q", c.in, got, c.want)
		}
	}
}

func TestLayout(t *testing.T) {
	in := "   1.   Find   the ﬁrst  number.\r\n\n\n\n   It is in\u00adside.  "
	if got, want := Layout(in), "1. Find the first number.\n\nIt is inside."; got != want {
		t.Errorf("Layout: got %q, want %q", got, want)
	}
}

func TestEncodingFlag(t *testing.T) {
	if encodingFlag("Find all x.\n\fNext page") {
		t.Error("a clean text has a flag")
	}
	if !encodingFlag("Find \uF06C x") || !encodingFlag("a \uFFFD b") {
		t.Error("a broken font has no flag")
	}
}

// layer is a Runner that gives a born-digital PDF of three pages with a text layer. Page 2 has a font with no Unicode map, and page 3 has an exponent that the text layer writes on the line.
type layer struct{}

// box gives the layout of a page of one line, with a word of each height.
func box(heights ...float64) []byte {
	var b strings.Builder
	b.WriteString(`<html><body><doc><page width="595" height="842"><flow><block><line xMin="0" yMin="0" xMax="500" yMax="12">`)
	for i, h := range heights {
		fmt.Fprintf(&b, `<word xMin="%d" yMin="%g" xMax="%d" yMax="12">w</word>`, 10*i, 12-h, 10*i+8)
	}
	b.WriteString(`</line></block></flow></page></doc></body></html>`)
	return []byte(b.String())
}

func (layer) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	switch name {
	case "pdfinfo":
		return []byte("Pages: 3\nPage size: 595 x 842 pts (A4)\n"), nil
	case "pdftotext":
		page := args[2]
		if args[0] == "-bbox-layout" {
			if page == "3" {
				return box(10, 10, 7, 10), nil
			}
			return box(10, 10, 10, 9.5), nil
		}
		switch page {
		case "1":
			return []byte("      Problem 1.   Prove that a² + b² ≥ 2ab\n   for all reals a, b.\n\f"), nil
		case "2":
			return []byte("Problem 2. Find all  such that\f"), nil
		}
		return []byte("Problem 3. Prove that A−1 is the inverse of A.\f"), nil
	}
	return nil, errors.New("no " + name)
}

func TestNative(t *testing.T) {
	n := &Native{PDF: layer{}, Version: "26.10.0"}
	pages, err := n.Read(context.Background(), "a.pdf", nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(pages) != 3 {
		t.Fatalf("%d pages", len(pages))
	}
	if want := "Problem 1. Prove that $a^2 + b^2 \\ge 2ab$\nfor all reals a, b."; pages[0].Text != want || len(pages[0].Flags) != 0 {
		t.Errorf("page 1 = %q %v, want %q", pages[0].Text, pages[0].Flags, want)
	}
	if got := pages[0].Extraction(); got != "native/pdftotext@26.10.0 (om-tex/1)" {
		t.Errorf("extraction = %q", got)
	}
	if got := pages[1].Extraction(); got != "native/pdftotext@26.10.0" || !slices.Contains(pages[1].Flags, FlagEncoding) {
		t.Errorf("page 2: %q, flags %v", got, pages[1].Flags)
	}
	if !slices.Contains(pages[2].Flags, FlagLayout) || Confidence(pages[2].Flags) >= Threshold {
		t.Errorf("page 3 has flags %v", pages[2].Flags)
	}
	probs := Split(pages)
	if len(probs) != 3 || probs[0].Number != 1 || Confidence(probs[0].Flags) < Threshold {
		t.Errorf("split: %+v", probs)
	}
}
