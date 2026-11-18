package official

import (
	"reflect"
	"strings"
	"testing"
)

func TestNabojArchive(t *testing.T) {
	page := `<h2 class="font-bold">Náboj Math 2026</h2><a href=/gb/en/archive/29/problems/>Problems</a><a href=/gb/en/archive/29/results/>Results</a>
<h2 class="font-bold">Náboj Math 2010</h2><a href=/gb/en/archive/13/results/>Results</a>
<h2>Náboj Junior 2021</h2><a href="/sk/sk/archive/9/problems">Úlohy</a>`
	want := []NabojYear{{2026, "/gb/en/archive/29/problems/"}, {2010, ""}, {2021, "/sk/sk/archive/9/problems"}}
	if got := NabojArchive(page); !reflect.DeepEqual(got, want) {
		t.Errorf("NabojArchive = %v, want %v", got, want)
	}
	if got := NabojPDF(`<a href=/uploads/competitions/14/problems-en.pdf>PDF</a>`); got != "/uploads/competitions/14/problems-en.pdf" {
		t.Errorf("NabojPDF = %q", got)
	}
}

func TestNabojProblems(t *testing.T) {
	page := `<article class="my-8"><h2 class="font-bold text-lg">Problem 1</h2><div class="prose prose-archive"><p class=noindent>He has <math display=inline xmlns=http://www.w3.org/1998/Math/MathML><mn>5</mn></math> trunks and pays $2.<table class=equation-star><tr><td><math display=block><mrow><msup><mrow><mn>2</mn></mrow><mrow><mi>n</mi></mrow></msup></mrow></math></table><ul><li>one<li>two</ul><details><summary>Solution</summary>The answer is 7.</details></div></article>
<article><h2 class="font-bold text-lg">Úloha 2</h2><div class="prose prose-archive"><p style=margin-top:0;margin-bottom:0><span style=font-weight:700>Zbieranie kamienkov</span><br><p style=margin-top:0;margin-bottom:0>Alica dala <math display=inline><mn>11</mn></math> kamienkov.<img src=x.svg><details><summary>Riešenie</summary></details></div></article>
<article><h2>Problem 3</h2><div class="prose prose-archive"><p>Fill the table<table class=tabular><tr><td><math><mrow></mrow></math><td><math><mn>6</mn></math></table><details></details></div></article>`
	got := NabojProblems(page)
	if len(got) != 3 {
		t.Fatalf("NabojProblems read %d problems, want 3", len(got))
	}
	if want := "He has $5$ trunks and pays \\$2.\n\n$$2^n$$\n\n- one\n\n- two"; got[0].N != 1 || got[0].Text != want || got[0].Figure || len(got[0].Flags) != 0 {
		t.Errorf("problem 1 = %d %q %v %v, want the text %q", got[0].N, got[0].Text, got[0].Figure, got[0].Flags, want)
	}
	// The bold title of a junior problem is no part of its text, and its picture makes it a figure.
	if want := "Alica dala $11$ kamienkov."; got[1].N != 2 || got[1].Text != want || !got[1].Figure {
		t.Errorf("problem 2 = %d %q %v, want the text %q with a figure", got[1].N, got[1].Text, got[1].Figure, want)
	}
	if strings.Contains(got[2].Text, "$$") || !reflect.DeepEqual(got[2].Flags, []string{"layout"}) {
		t.Errorf("problem 3 = %q %v, want no empty formula and the flag layout", got[2].Text, got[2].Flags)
	}
}
