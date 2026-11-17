package cli

import (
	"slices"
	"testing"

	"github.com/tamnd/olymmath-agent/official"
)

func TestUkmtPaperOf(t *testing.T) {
	for _, c := range []struct {
		slug    string
		want    hrPaper
		matches bool
	}{
		{"junior-mathematical-challenge-2024", hrPaper{kind: "uk-jmc", year: 2024}, true},
		{"senior-mathematical-challenge-2009", hrPaper{kind: "uk-smc", year: 2009}, true},
		{"junior-mathematical-olympiad-2023-2", hrPaper{kind: "uk-jmo", year: 2023}, true},
		{"cayley-mathematical-olympiad-2025", hrPaper{kind: "uk-imoc", variant: "cayley", year: 2025}, true},
		{"cayley-olympiad-2023-no-files", hrPaper{kind: "uk-imoc", variant: "cayley", year: 2023}, true},
		{"cayley-2022", hrPaper{kind: "uk-imoc", variant: "cayley", year: 2022}, true},
		{"mathematical-olympiad-for-girls-2014-2", hrPaper{kind: "uk-mog", year: 2014}, true},
		{"grey-kangaroo-2024", hrPaper{}, false},
		{"british-mathematical-olympiad-1-2023", hrPaper{}, false},
		{"senior-mathematical-challenge-sample-answer-sheet", hrPaper{}, false},
	} {
		got, ok := ukmtPaperOf(c.slug)
		if ok != c.matches || got != c.want {
			t.Errorf("ukmtPaperOf(%q) = %+v, %v; want %+v, %v", c.slug, got, ok, c.want, c.matches)
		}
	}
}

func TestUkmtPaperPDF(t *testing.T) {
	u := "https://ukmt.org.uk/wp-content/uploads/2023/08/"
	page := `<a href="` + u + `imc-2019-extended.pdf">x</a><a href="` + u + `imc-2019-q.pdf">x</a><a href="` + u + `imc-2019-s.pdf">x</a><a href="` + u + `imc-2019-q.pdf">x</a>`
	if got := ukmtPaperPDF(page); got != u+"imc-2019-q.pdf" {
		t.Errorf("ukmtPaperPDF = %q", got)
	}
	for _, page := range []string{
		`<a href="` + u + `SMC-2015-sol-short.pdf">x</a>`,
		`<img src="/wp-content/uploads/individual-problems/jmc/2008/Pic1.gif">`,
		`<a href="` + u + `a.pdf">x</a><a href="` + u + `b.pdf">x</a>`,
	} {
		if got := ukmtPaperPDF(page); got != "" {
			t.Errorf("ukmtPaperPDF(%q) = %q, want none", page, got)
		}
	}
}

func TestUkmtPlace(t *testing.T) {
	probs := []official.HrProblem{{N: 1, Section: "A"}, {N: 2}}
	probs[0].Code = "A1"
	probs[1].Code = "2"
	for _, c := range []struct {
		p    hrPaper
		want []string
	}{
		{hrPaper{kind: "uk-jmc", year: 2024}, []string{"uk-jmc/2024/p1", "uk-jmc/2024/p2"}},
		{hrPaper{kind: "uk-imoc", variant: "hamilton", year: 2017}, []string{"uk-imoc/2017/h1", "uk-imoc/2017/h2"}},
		{hrPaper{kind: "uk-jmo", year: 2016}, []string{"uk-jmo/2016/a1", ""}},
		{hrPaper{kind: "uk-jmo", year: 2025}, []string{"uk-jmo/2025/p1", "uk-jmo/2025/p2"}},
	} {
		ats, names := ukmtPlace(c.p, probs)
		if !slices.Equal(ats, c.want) || len(names) != 2 {
			t.Errorf("ukmtPlace(%+v) = %v, %v; want %v", c.p, ats, names, c.want)
		}
	}
	if _, names := ukmtPlace(hrPaper{kind: "uk-imoc", variant: "cayley", year: 2024}, probs[:1]); names[0] != "UKMT, Intermediate Mathematical Olympiad, Cayley paper 2024" {
		t.Errorf("name = %q", names[0])
	}
}
