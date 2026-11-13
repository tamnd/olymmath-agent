package coverage

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseGaps(t *testing.T) {
	known := func(code string) bool { return code == "imo" || code == "amc10" }
	good := `
- competition: amc10
  years: [2001, "2003-2005"]
  positions: [p1]
  reason: robots
  note: robots.txt blocks the archive.
- competition: amc10
  years: [2002]
  reason: no-source
  note: No paper was found.
`
	g, err := parseGaps([]byte(good), known)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		year int
		pos  string
		want string
	}{
		{2001, "p1", Robots},
		{2004, "p1", Robots},
		{2004, "p2", ""},
		{2002, "p2", NoSource},
		{2006, "p1", ""},
	}
	for _, tc := range cases {
		got := ""
		if e := g.entry("amc10", tc.year, tc.pos); e != nil {
			got = e.Reason
		}
		if got != tc.want {
			t.Errorf("entry(%d, %s) = %q, want %q", tc.year, tc.pos, got, tc.want)
		}
	}
	if g, err := parseGaps(nil, known); err != nil || len(g.List) != 0 {
		t.Errorf("an empty register gave %v, %v", g, err)
	}
	bad := map[string]string{
		"- {competition: nope, reason: robots, note: x}":                    "not in the competition register",
		"- {competition: imo, reason: maybe, note: x}":                      "is not one of",
		"- {competition: imo, reason: robots}":                              "the note",
		"- {competition: imo, years: [1999-1990], reason: robots, note: x}": "not a year",
		"- {competition: imo, years: [abc], reason: robots, note: x}":       "not a year",
		"- {competition: imo, reason: robots, note: x, extra: 1}":           "extra",
		"- {competition: imo, years: [2000-2010], reason: robots, note: x}\n" +
			"- {competition: imo, years: [2005], positions: [p2], reason: no-source, note: y}": "an earlier entry",
	}
	for in, want := range bad {
		if _, err := parseGaps([]byte(in), known); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q gave %v, want %q", in, err, want)
		}
	}
	// Entries for other positions of one year do not overlap.
	two := "- {competition: imo, years: [2005], positions: [p1], reason: robots, note: x}\n- {competition: imo, years: [2005], positions: [p2], reason: robots, note: y}"
	if _, err := parseGaps([]byte(two), known); err != nil {
		t.Errorf("two positions gave %v", err)
	}
}

func TestMeasureGaps(t *testing.T) {
	c := amc()
	g, err := parseGaps([]byte(`
- competition: amc10
  years: ["2000-2001"]
  reason: link-only
  note: The rule says link only.
- competition: amc10
  years: [2010]
  reason: no-source
  note: No paper was found.
- competition: amc10
  years: [2021]
  positions: [p1]
  reason: robots
  note: robots.txt blocks the archive.
`), func(string) bool { return true })
	if err != nil {
		t.Fatal(err)
	}
	cand := map[string]map[string]bool{"2001": {"p1": true}, "2010b": {"p2": true}}
	out := measure(c, cand, nil, nil, 0, 2026, g)
	// 2000 and 2001: 4 link-only positions, the candidate too. 2010a and 2010b: 4 positions with no source, less the candidate of 2010b. 2021: p1 of four papers.
	if out.LinkOnly != 4 || out.NoSource != 3 || out.Robots != 4 {
		t.Errorf("link-only %d, no source %d, robots %d", out.LinkOnly, out.NoSource, out.Robots)
	}
	if out.Open != out.Expected-out.C0-11 || out.Allowed() != out.Expected-11 {
		t.Errorf("open %d, allowed %d, expected %d", out.Open, out.Allowed(), out.Expected)
	}
	if g.List[0].Count != 4 || g.List[1].Count != 3 || g.List[2].Count != 4 {
		t.Errorf("counts %d %d %d", g.List[0].Count, g.List[1].Count, g.List[2].Count)
	}
	for _, y := range out.Years {
		if y.Edition == "2010b" && (y.Reasons["p1"] != NoSource || y.Reasons["p2"] != "") {
			t.Errorf("2010b reasons %v", y.Reasons)
		}
	}
	rep := &Report{Competitions: []*Comp{out}, Gaps: g.List}
	rep.Totals = rep.Sum(func(*Comp) bool { return true })
	out.Tier1 = true
	t1 := rep.Sum(func(c *Comp) bool { return c.Tier1 })
	rep.Tier1 = &t1
	var md bytes.Buffer
	if err := rep.Markdown(&md, 0); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"## Tier 1 nationals", "C0 allowed", "4 link-only by rule, 4 blocked by robots.txt, 3 with no allowed source", "| amc10 | 2010 | all | no-source | 3 | No paper was found. |", "| amc10 | 92 | 0.0% | 81 | 0.0% | 2 | 4 | 4 | 3 | 81 |"} {
		if !strings.Contains(md.String(), want) {
			t.Errorf("the Markdown has no %q", want)
		}
	}
}
