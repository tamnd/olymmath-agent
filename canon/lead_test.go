package canon

import (
	"strconv"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
)

func TestLeadNumber(t *testing.T) {
	cases := []struct {
		text string
		n    int
		rest string
		ok   bool
	}{
		{"1. Let $n$ be odd.", 1, "Let $n$ be odd.", true},
		{"**Problem 3.** Let $n$ be odd.", 3, "Let $n$ be odd.", true},
		{"Problem-2:  Given a circle.", 2, "Given a circle.", true},
		{"P6. It is given the integer $Y$.", 6, "It is given the integer $Y$.", true},
		{"10. Let $n \\ge 5$ be odd.", 10, "Let $n \\ge 5$ be odd.", true},
		{"4) A board has $k$ black cells.", 4, "A board has $k$ black cells.", true},
		{"1.) In the plane let $S$ be a set.", 1, "In the plane let $S$ be a set.", true},
		// A number that starts the sentence, a grade and a number, or a mark with no text are not marks.
		{"14 students attend the camp.", 0, "", false},
		{"9.2 Given 19 cards.", 0, "", false},
		{"4.4. Prove that $X$ exists.", 0, "", false},
		{"Problem 3.", 0, "", false},
		{"Let $n$ be odd.", 0, "", false},
		// A row with a second mark holds a whole paper or numbered parts.
		{"**p1.** Find $x$. **p2.** Find $y$.", 0, "", false},
		{"1. Find $x$.\n2. Find $y$.", 0, "", false},
	}
	for _, tc := range cases {
		n, rest, ok := LeadNumber(tc.text)
		if !tc.ok {
			rest = ""
		}
		if n != tc.n || rest != tc.rest || ok != tc.ok {
			t.Errorf("LeadNumber(%q) = %d, %q, %v, want %d, %q, %v", tc.text, n, rest, ok, tc.n, tc.rest, tc.ok)
		}
	}
}

func TestLeadSure(t *testing.T) {
	cases := []struct {
		numbers []int
		rows, k int
		one     bool
		want    bool
	}{
		{[]int{1, 2, 4}, 4, 4, false, true},
		{[]int{1, 2, 4, 5, 6, 7, 8, 9, 10}, 10, 10, false, true},
		{[]int{2}, 1, 4, true, true},
		// Two papers that the source numbers from 1 each: the last number is not there, or a number comes twice.
		{[]int{1, 2, 3}, 3, 6, false, false},
		{[]int{1, 2, 3, 1}, 4, 6, false, false},
		// More rows than positions: the collection has another part of the edition.
		{[]int{1, 6}, 7, 6, false, false},
		{nil, 1, 6, true, false},
	}
	for _, tc := range cases {
		if got := leadSure(tc.numbers, tc.rows, tc.k, tc.one); got != tc.want {
			t.Errorf("leadSure(%v, %d, %d, %v) = %v, want %v", tc.numbers, tc.rows, tc.k, tc.one, got, tc.want)
		}
	}
}

func TestLeadPosition(t *testing.T) {
	c := &content.Competition{
		Layout: []content.Layout{{Positions: []string{"p1", "p2", "p3"}}, {Positions: []string{"p4", "p5", "p6"}}},
		Eras:   []content.Era{{From: 2019, To: 2019, Layout: []content.Layout{{Name: "Grade 9", Positions: []string{"p91", "p92"}}, {Name: "Grade 10", Positions: []string{"p101", "p102"}}}}},
	}
	for _, tc := range []struct {
		year, n int
		pos     string
	}{{2020, 4, "p4"}, {2020, 7, ""}, {2019, 1, ""}} {
		if pos, _ := leadPosition(c, tc.year, tc.n); pos != tc.pos {
			t.Errorf("leadPosition(%d, %d) = %q, want %q", tc.year, tc.n, pos, tc.pos)
		}
	}
	if onePaper(c, 2020) || !onePaper(&content.Competition{Layout: c.Layout[:1]}, 2020) || onePaper(&content.Competition{Layout: c.Layout[:1], Overrides: map[int][]string{2020: {"p1"}}}, 2020) {
		t.Error("onePaper gives the wrong answer")
	}
}

// TestSeedsLead checks the rows of the AoPS dataset whose text starts with a problem mark, in a competition with two papers of three problems.
func TestSeedsLead(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	imo := corpus.Competition("imo")
	imo.Layout = []content.Layout{{Positions: []string{"p1", "p2", "p3"}}, {Positions: []string{"p4", "p5", "p6"}}}
	imo.PerYear = 6
	tp := &Topics{Aliases: al, Corpus: corpus}
	// The numbered source gives topic 900 the position imo/2024/p1.
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2024, Number: "1", Ref: "https://artofproblemsolving.com/community/c6h900p1", Statement: "IMO 2024 P1"})
	s := &Seeds{Corpus: corpus, Aliases: al, Topics: tp}
	text := " Let $n$ be a positive integer. Prove that the sum of the first $n$ odd numbers is a square."
	topic := 1000
	add := func(year int, marks ...string) {
		for _, m := range marks {
			topic++
			s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO"}, Year: year, Ref: "aops-topic:" + strconv.Itoa(topic), Statement: m + text})
		}
	}
	add(2020, "1.", "**Problem 2.**", "", "6)")
	add(2021, "1.", "2.", "3.", "1.")
	add(2022, "1.", "2.", "3.")
	add(2023, "1.", "2.", "3.", "4.", "5.", "6.", "")
	add(2024, "1.", "6.")
	got := s.Candidates()
	for _, at := range []string{"imo/2020/p1", "imo/2020/p2", "imo/2020/p6", "imo/2024/p6"} {
		if len(got[at]) != 1 {
			t.Errorf("no candidate at %s", at)
		}
	}
	if c := got["imo/2020/p1"]; len(c) == 1 && c[0].Text[:7] != "Let $n$" {
		t.Errorf("the mark is in the text: %q", c[0].Text)
	}
	if len(got) != 4 || s.Led != 4 || s.Skipped["lead number not sure"] != 13 || s.Skipped["lead number claimed by another topic"] != 1 {
		t.Errorf("candidates %v, led %d, skipped %v", got, s.Led, s.Skipped)
	}
}
