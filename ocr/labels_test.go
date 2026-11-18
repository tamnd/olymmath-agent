package ocr

import (
	"slices"
	"testing"
)

func TestLabels(t *testing.T) {
	pages := []Page{
		{N: 1, Text: "# Županijsko natjecanje\n\n## 1. razred\n\n**1.** Neka su $x$ i $y$ realni brojevi. Odredi $x+y$.\n\n**2.** Dokaži da je $n^2+n$ paran broj za svaki $n$."},
		{N: 2, Text: "**3.** Na slici je trokut $ABC$. Odredi njegovu površinu.\n\n## 2. razred\n\n**1.** Riješi jednadžbu $x^2=4$ u skupu realnih brojeva.\n\n**2.** Kratko."},
	}
	got := Labels(pages)
	var labels []string
	var pageOf []int
	for _, l := range got {
		labels = append(labels, l.Label)
		pageOf = append(pageOf, l.Page)
	}
	if !slices.Equal(labels, []string{"1", "2", "3", "1", "2"}) || !slices.Equal(pageOf, []int{1, 1, 2, 2, 2}) {
		t.Fatalf("labels %v on pages %v", labels, pageOf)
	}
	if got[0].Text != "Neka su $x$ i $y$ realni brojevi. Odredi $x+y$." || got[0].Flags != nil {
		t.Errorf("the first problem: %q %v", got[0].Text, got[0].Flags)
	}
	if !slices.Contains(got[2].Flags, FlagFigure) {
		t.Errorf("a problem that names a figure has the flags %v", got[2].Flags)
	}
	if !slices.Contains(got[4].Flags, FlagShort) {
		t.Errorf("a short problem has the flags %v", got[4].Flags)
	}
}

func TestLabelsLong(t *testing.T) {
	pages := []Page{{N: 2, Text: "**Aufgabe 011131:** Man beweise, dass $a^2 \\ge 0$ für jede reelle Zahl $a$ gilt.\n\n**Aufgabe 011135 = 011234:** Man bestimme alle Zahlen $n$ mit $n^2 = n$.\n\nArbeitszeit: 4 Stunden"}}
	got := Labels(pages)
	if len(got) != 2 || got[0].Label != "011131" || got[1].Label != "011135" {
		t.Fatalf("labels %+v", got)
	}
	if got[1].Digit() != 5 || got[0].Digit() != 1 {
		t.Errorf("digits %d %d", got[0].Digit(), got[1].Digit())
	}
}

func TestLabelsWord(t *testing.T) {
	pages := []Page{{N: 1, Text: "Problem 1. Find all primes $p$ such that $p+2$ is prime.\nProblem 2. Find the sum of the digits of $2^{10}$."}}
	got := Labels(pages)
	if len(got) != 2 || got[1].Label != "2" || got[1].Text != "Find the sum of the digits of $2^{10}$." {
		t.Fatalf("labels %+v", got)
	}
	if Labels([]Page{{N: 1, Text: "No problems here."}}) != nil {
		t.Error("a text with no labels has problems")
	}
}

func TestPositionDigit(t *testing.T) {
	for pos, want := range map[string]int{"p126a": 6, "a12": 2, "h30": 0, "p115": 5, "x": -1} {
		if got := PositionDigit(pos); got != want {
			t.Errorf("PositionDigit(%q) = %d, want %d", pos, got, want)
		}
	}
	if got := (Labeled{Label: "011246A"}).Digit(); got != 6 {
		t.Errorf("the digit of 011246A is %d", got)
	}
}
