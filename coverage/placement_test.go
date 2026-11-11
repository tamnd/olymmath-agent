package coverage

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/locator"
)

func TestExpected(t *testing.T) {
	c := &content.Competition{Code: "imo", FirstYear: 1959, PerYear: 6, Overrides: map[int][]string{2000: {"p1", "p2", "p3"}}}
	if got := strings.Join(Expected(c, 2024, nil), " "); got != "p1 p2 p3 p4 p5 p6" {
		t.Errorf("shape %q", got)
	}
	if got := strings.Join(Expected(c, 2000, nil), " "); got != "p1 p2 p3" {
		t.Errorf("override %q", got)
	}
	e := &content.Edition{Papers: []content.Paper{{Problems: []string{"p1", "p2"}}, {Problems: []string{"p3", "p4", "p5", "p6", "p7"}}}}
	if got := strings.Join(Expected(c, 2024, e), " "); got != "p1 p2 p3 p4 p5 p6 p7" {
		t.Errorf("edition file %q", got)
	}
	if got := Expected(&content.Competition{Code: "isl"}, 2024, &content.Edition{}); got != nil {
		t.Errorf("no shape %q", got)
	}
}

func TestUnexpected(t *testing.T) {
	imo := &content.Competition{Code: "imo", FirstYear: 1959, Gaps: []int{1980}, PerYear: 6}
	extra := &content.Edition{ID: "2008", Papers: []content.Paper{{Problems: []string{"p1", "p2", "p3", "p4", "p5", "p6", "p7"}}}}
	memo := &content.Competition{Code: "memo", FirstYear: 2007, LastYear: 2020, Layout: []content.Layout{{Positions: []string{"i1", "i2"}}, {Positions: []string{"t1", "t2"}}}}
	free := &content.Competition{Code: "isl", FirstYear: 1959}
	season := &content.Competition{Code: "tot", FirstYear: 1980, EditionForm: "year-season", PerYear: 5}
	cases := []struct {
		c    *content.Competition
		e    *content.Edition
		at   string
		want string
	}{
		{imo, nil, "imo/2024/p6", ""},
		{imo, nil, "imo/1980/p6", "the register marks 1980 as a year with no imo"},
		{imo, nil, "imo/1958/p1", "1958 is before the first year 1959 of imo"},
		{imo, nil, "imo/2027/p1", "2027 is after the last year 2026 of imo"},
		{imo, nil, "imo/2024/p7", "imo/2024 has the positions p1 to p6, and p7 is not one of them"},
		{imo, extra, "imo/2008/p7", ""},
		{imo, nil, "imo/2024a/p1", "the register gives one edition a year, 2024, and no edition 2024a"},
		{memo, nil, "memo/2021/i1", "2021 is after the last year 2020 of memo"},
		{memo, nil, "memo/2010/t2", ""},
		{memo, nil, "memo/2010/p5", "memo/2010 has the positions i1, i2, t1, t2, and p5 is not one of them"},
		{free, nil, "isl/2024/c9", ""},
		{amc(), nil, "amc10/2010a/p2", ""},
		{amc(), nil, "amc10/2010/p2", "the register gives the papers 2010a, 2010b for 2010, and 2010 is not one of them"},
		{amc(), nil, "amc10/2001/p1", ""},
		{amc(), nil, "amc10/2021-fall-b/p2", ""},
		{amc(), nil, "amc10/2010a/p3", "amc10/2010a has the positions p1, p2, and p3 is not one of them"},
		{season, nil, "tot/2024-fall/p5", ""},
		{season, nil, "tot/2024-fall/p6", "tot/2024-fall has the positions p1 to p5, and p6 is not one of them"},
	}
	for _, tc := range cases {
		l, err := locator.Parse(tc.at)
		if err != nil {
			t.Fatal(err)
		}
		if got := Unexpected(tc.c, tc.e, l, 2026); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.at, got, tc.want)
		}
	}
}
