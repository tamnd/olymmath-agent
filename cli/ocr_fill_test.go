package cli

import (
	"slices"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/locator"
	"github.com/tamnd/olymmath-agent/ocr"
)

// fillProblem gives a problem at the positions of ats, the first one primary, with a link to page of url.
func fillProblem(url string, page int, text string, ats ...string) *content.Problem {
	p := &content.Problem{Statement: text}
	for i, at := range ats {
		l, err := locator.Parse(at)
		if err != nil {
			panic(err)
		}
		p.Appearances = append(p.Appearances, content.Appearance{At: at, Primary: i == 0, Locator: l})
	}
	if url != "" {
		p.Statements = []content.Statement{{ID: "S1", Source: "olympiade-mathematik", URL: url, Page: page, Zone: "Z2"}}
	}
	return p
}

func TestPaperGroups(t *testing.T) {
	a, b := "https://example.org/01113_a.pdf", "https://example.org/01123_a.pdf"
	e := &content.Edition{ID: "1962", Competition: &content.Competition{Code: "de-mo-r3"}, Official: []content.OfficialPaper{{URL: a}, {URL: b}, {URL: a}}}
	e.Problems = []*content.Problem{
		fillProblem(a, 3, "", "de-mo-r3/1962/p113"),
		fillProblem(a, 2, "Man beweise den Satz.", "de-mo-r3/1962/p111"),
		fillProblem(a, 2, "", "de-mo-r3/1962/p112"),
		fillProblem(b, 3, "", "de-mo-r3/1962/p124", "de-mo-r3/1962/p115"),
		fillProblem(b, 2, "", "de-mo-r3/1962/p121"),
		fillProblem("https://example.org/other.pdf", 1, "", "de-mo-r3/1962/p71"),
		fillProblem(a, 2, "", "de-mo/1962/p1", "de-mo-r3/1962/p110"),
	}
	groups, urls := paperGroups(e)
	if !slices.Equal(urls, []string{a, b}) {
		t.Fatalf("papers %v", urls)
	}
	var got []string
	for _, r := range groups[a] {
		s := r.pos
		if r.shared {
			s += "*"
		}
		if r.link {
			s += "-"
		}
		got = append(got, s)
	}
	if want := []string{"p110*", "p111", "p112-", "p113-", "p115*"}; !slices.Equal(got, want) {
		t.Errorf("the paper %s has %v, want %v", a, got, want)
	}
	if len(groups[b]) != 2 || groups[b][0].pos != "p121" || groups[b][1].pos != "p124" {
		t.Errorf("the paper %s has %+v", b, groups[b])
	}
	if p := pagesToRead(groups[a], 3); !slices.Equal(p, []int{2, 3}) {
		t.Errorf("the pages to read are %v", p)
	}
	if p := pagesToRead(groups[b], 4); !slices.Equal(p, []int{2, 3, 4}) {
		t.Errorf("the pages to read are %v", p)
	}
}

func TestAlignPaper(t *testing.T) {
	text := "Man beweise, dass für jede natürliche Zahl n die Zahl n hoch zwei plus n gerade ist."
	g := []fillRec{
		{p: &content.Problem{Statement: text}, pos: "p111", page: 2},
		{p: &content.Problem{}, pos: "p112", page: 2, link: true},
		{p: &content.Problem{}, pos: "p115", shared: true},
	}
	probs := []ocr.Labeled{{Label: "011131", Page: 2, Text: text}, {Label: "011132", Page: 2, Text: "Man bestimme alle x."}, {Label: "011135", Page: 3, Text: "Man zeige y."}}
	if why := alignPaper(g, probs, 0.3); why != "" {
		t.Errorf("a paper that lines up: %s", why)
	}
	cases := map[string][]ocr.Labeled{
		"read 2 problems": probs[:2],
		"is on page 2":    {probs[0], {Label: "011132", Page: 3}, probs[2]},
		"digit of p112":   {probs[0], {Label: "011133", Page: 2}, probs[2]},
		"does not agree":  {{Label: "011131", Page: 2, Text: "Ein ganz anderer Text über Dreiecke und Kreise in der Ebene."}, probs[1], probs[2]},
	}
	for want, p := range cases {
		if why := alignPaper(g, p, 0.3); !strings.Contains(why, want) {
			t.Errorf("got %q, want %q", why, want)
		}
	}
}

func TestComparePositions(t *testing.T) {
	ps := []string{"h10", "m2", "h2", "h1", "p126a", "p126", "p12"}
	slices.SortFunc(ps, comparePositions)
	if want := []string{"h1", "h2", "h10", "m2", "p12", "p126", "p126a"}; !slices.Equal(ps, want) {
		t.Errorf("sorted %v, want %v", ps, want)
	}
	for pos, want := range map[string]string{"p115": "p11", "p126a": "p12", "h3": "h", "x": "x"} {
		if got := positionPrefix(pos); got != want {
			t.Errorf("positionPrefix(%q) = %q, want %q", pos, got, want)
		}
	}
}
