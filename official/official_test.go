package official

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

const paperText = `Language: English
Day: 1
Monday, July 15, 2024
Problem 1. Determine all real numbers a such that, for every positive integer n, the integer
x is a multiple of n.
Problem 2. Determine all pairs (a, b) of positive integers for which there exist positive
integers g and N such that gcd(a^n + b, b^n + a) = g holds for all integers n ≥ N.
` + "\f" + `Language: English
Day: 2
Problem 4. Let ABC be a triangle with AB < AC < BC.
Problem 1. Une copie dans une autre langue.
`

func TestSplit(t *testing.T) {
	segs := Split(paperText)
	if len(segs) != 3 {
		t.Fatalf("got %d segments, want 3: %+v", len(segs), segs)
	}
	if segs[0].Position != "p1" || segs[0].Page != 1 || segs[0].Text != "Determine all real numbers a such that, for every positive integer n, the integer x is a multiple of n." {
		t.Errorf("segment 1: %+v", segs[0])
	}
	if segs[2].Position != "p4" || segs[2].Page != 2 || segs[2].Text != "Let ABC be a triangle with AB < AC < BC." {
		t.Errorf("segment 3: %+v", segs[2])
	}
	bare := Split("1. Let p be a prime.\n 1. a list item\n2. Find all n.\n3. Prove it.\n2. another item\n")
	if len(bare) != 3 || bare[1].Text != "Find all n." || bare[2].Text != "Prove it. 2. another item" {
		t.Errorf("bare markers: %+v", bare)
	}
	old := Split("1959/1. Prove that the fraction is irreducible.\n1959/2. For what real values of x")
	if len(old) != 2 || old[1].Position != "p2" {
		t.Errorf("old markers: %+v", old)
	}
}

func TestSignal(t *testing.T) {
	if s := Signal(`Let $\triangle ABC$ be a triangle`, "Let ABC be a triangle."); s != 1 {
		t.Errorf("same words: %v", s)
	}
	if s := Signal("Find all functions", "Prove that the sum is odd"); s != 0 {
		t.Errorf("other words: %v", s)
	}
	if s := Signal("", "words"); s != 0 {
		t.Errorf("empty: %v", s)
	}
}

func TestIMO(t *testing.T) {
	ps := IMO(1979, 1981)
	if len(ps) != 2 || ps[1].Edition != "imo/1981" || ps[1].URL != "https://www.imo-official.org/assets/documents/problems/1981/1981_eng.pdf" {
		t.Errorf("papers: %+v", ps)
	}
}

func TestAnchor(t *testing.T) {
	p1 := &content.Problem{Tag: "a", Appearances: []content.Appearance{{At: "imo/2024/p1"}}, Canonical: content.Canonical{Statement: "S1"},
		Statements: []content.Statement{{ID: "S1", Source: "aops", Text: "Determine all real numbers $a$ such that for every positive integer $n$ the integer $x$ is a multiple of $n$."}}}
	p2 := &content.Problem{Tag: "b", Appearances: []content.Appearance{{At: "imo/2024/p2"}}, Canonical: content.Canonical{Statement: "S1"},
		Statements: []content.Statement{{ID: "S1", Source: "aops", Text: "Find all functions on the reals with a strange property."}}}
	p4 := &content.Problem{Tag: "c", Appearances: []content.Appearance{{At: "imo/2024/p4"}}, Canonical: content.Canonical{Statement: "S1"},
		Statements: []content.Statement{{ID: "S1", Source: "imo-official"}}}
	owners := map[string]*content.Problem{"imo/2024/p1": p1, "imo/2024/p2": p2, "imo/2024/p4": p4}
	segs := append(Split(paperText), Segment{Position: "p5", Page: 2, Text: "x"})
	r := Anchor(owners, "imo-official", Paper{Edition: "imo/2024", URL: "u"}, segs)
	if len(r.Links) != 1 || r.Links[0].Problem != p1 || r.Links[0].Segment.Page != 1 {
		t.Errorf("links: %+v", r.Links)
	}
	if len(r.Items) != 1 || r.Items[0].At != "imo/2024/p2" || r.Items[0].Kind != "official" || len(r.Items[0].ID) != 12 {
		t.Errorf("items: %+v", r.Items)
	}
	if r.Have != 1 || len(r.Missing) != 1 || r.Missing[0] != "imo/2024/p5" {
		t.Errorf("have %d, missing %v", r.Have, r.Missing)
	}
}

func TestLocate(t *testing.T) {
	ps := Papers{"imo/2024": Split(paperText)}
	if got := ps.Locate("imo/2024", "Determine all pairs $(a,b)$ of positive integers for which there exist positive integers $g$ and $N$ such that $\\gcd(a^n+b,b^n+a)=g$ holds for all integers $n\\ge N$."); got != "p2" {
		t.Errorf("locate p2: %q", got)
	}
	if got := ps.Locate("imo/2024", "Find all functions on the reals."); got != "" {
		t.Errorf("locate other: %q", got)
	}
	if got := ps.Locate("imo/2023", "Let ABC be a triangle with AB < AC < BC."); got != "" {
		t.Errorf("locate no paper: %q", got)
	}
	text := "Prove that every positive integer greater than one has a prime divisor."
	twice := Papers{"usamo/2001": {{Position: "p1", Text: text}, {Position: "p1", Text: text}, {Position: "p2", Text: "Find all real numbers with a property."}}}
	if got := twice.Locate("usamo/2001", text); got != "p1" {
		t.Errorf("locate with two copies: %q", got)
	}
}

func TestTrace(t *testing.T) {
	paper := "Let n be a positive integer. A Japanese triangle consists of 1 + 2 + ... + n circles arranged in an equilateral triangular shape."
	same := "Let $n$ be a positive integer. A *Japanese triangle* consists of $1 + 2 + \\dots + n$ circles arranged in an equilateral triangular shape."
	if !Trace(paper, same) {
		t.Errorf("a transcription was not traced (order %.3f)", Order(paper, same))
	}
	if Trace(paper, strings.Replace(same, "1 + 2", "1 + 3", 1)) {
		t.Error("a changed number was traced")
	}
	if Trace(paper, "Let $n$ be a positive integer. Find all $1 + 2 + \\dots + n$ shapes.") {
		t.Error("a different text was traced")
	}
	if !Trace("Points A, B and O lie on a circle with centre O 0 1 2.", "Points $A$, $B$ and $O$ lie on a circle with centre $O$ 1 2.") {
		t.Error("the point O of an old paper was counted as a number")
	}
}

func TestSplitBooklet(t *testing.T) {
	head := "Shortlisted problems                    %d\n"
	var b strings.Builder
	for pg := 1; pg <= 6; pg++ {
		b.WriteString(strings.Replace(head, "%d", strings.Repeat("1", pg), 1))
		switch pg {
		case 1:
			b.WriteString("Problems\nAlgebra\nA1. Let n be a positive integer.\n 1 1\nProve that n > 0.\n(Ukraine)\n")
		case 2:
			b.WriteString("A2\n   A2\n   Find all reals x.\n")
		case 3:
			b.WriteString("Solutions\nA1. Let n be a positive integer. Prove that n > 0.\nSolution. Easy.\n")
		case 4:
			b.WriteString("G 3  CZE  (Czech Republic)\nLet ABC be a triangle.\nAnswer: none.\n")
		}
		b.WriteString("\n17\n\f")
	}
	segs := SplitBooklet(b.String())
	want := map[string]string{"a1": "Let n be a positive integer. 1 1 Prove that n > 0.", "a2": "Find all reals x.", "g3": "Let ABC be a triangle."}
	if len(segs) != len(want) {
		t.Fatalf("%+v", segs)
	}
	for _, s := range segs {
		if want[s.Position] != s.Text {
			t.Errorf("%s: %q", s.Position, s.Text)
		}
	}
	if segs[0].Page != 1 || segs[2].Page != 4 {
		t.Errorf("pages %d %d", segs[0].Page, segs[2].Page)
	}
}

func TestShortlist(t *testing.T) {
	ps := Shortlist(2000, 2007)
	if len(ps) != 2 || ps[0].Edition != "isl/2006" || !ps[1].Booklet || ps[1].URL != "https://www.imo-official.org/assets/documents/problems/2007/IMO2007SL.pdf" {
		t.Errorf("%+v", ps)
	}
}

func TestLanguages(t *testing.T) {
	page := `<select data-settings-problems-language><option value="afr">Afrikaans</option></select>
<select data-download-widget="language-select">
<option
  value="/assets/documents/problems/2023/2023_eng.pdf"
  data-lang-code="eng"
  selected
>
  English
</option>
<option value="/assets/documents/problems/2023/2023_alk.pdf" data-lang-code="alk">Albanian (Kosovo)</option>
<option value="/assets/documents/problems/2023/2023_alk.pdf" data-lang-code="alk">Albanian (Kosovo)</option>
</select>`
	ls := Languages(page)
	if len(ls) != 2 || ls[0].Lang != "English" || ls[0].Code != "eng" || ls[1].Lang != "Albanian (Kosovo)" || ls[1].URL != "https://www.imo-official.org/assets/documents/problems/2023/2023_alk.pdf" {
		t.Errorf("%+v", ls)
	}
}
