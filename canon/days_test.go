package canon

import (
	"slices"
	"strconv"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
)

// dayLayout gives a layout of days named "Day 1", "Day 2" and so on, with the given number of problems on each day.
func dayLayout(name string, sizes ...int) []content.Layout {
	var ls []content.Layout
	n := 0
	for d, k := range sizes {
		l := content.Layout{Paper: "day" + strconv.Itoa(d+1), Name: name + " " + strconv.Itoa(d+1)}
		for range k {
			n++
			l.Positions = append(l.Positions, "p"+strconv.Itoa(n))
		}
		ls = append(ls, l)
	}
	return ls
}

func TestTitleUnits(t *testing.T) {
	cases := []struct {
		title     string
		day, test int
	}{
		{"Vietnam TST 2019 Day 2 P1", 2, 0},
		{"Vietnam TST 2019 Day II Problem 1", 2, 0},
		{"China south east mathematical olympiad 2006 day2 problem 5", 2, 0},
		{"Vietnam MO 2018 2nd day 1st problem", 2, 0},
		{"Vietnam MO 1991 #4 second day", 2, 0},
		{"Problem 5 (Day 2)", 2, 0},
		{"Vietnam MO 2022 problem 5 day 2", 2, 0},
		{"TST 3 Problem 2", 0, 3},
		{"TST3,day1,P2", 1, 3},
		{"Iran TST 2015 exam 2 day 1 problem 2", 1, 2},
		{"Iranian TST 2018 first exam day 1 problem 1", 1, 1},
		{"Romania TST 2010 Test 4 Problem 1", 0, 4},
		{"Vietnam TST 2019", 0, 0},
		{"Vietnam TST 2019 P4", 0, 0},
		{"Iran TST 2013 Day 1 P2 Day 2", -1, 0},
		{"Tests 2 day in the morning", 0, 0},
	}
	for _, tc := range cases {
		if d, ts := titleUnits(tc.title); d != tc.day || ts != tc.test {
			t.Errorf("titleUnits(%q) = %d, %d, want %d, %d", tc.title, d, ts, tc.day, tc.test)
		}
	}
}

func TestUnitKind(t *testing.T) {
	cases := []struct {
		layout []content.Layout
		kind   string
	}{
		{dayLayout("Day", 3, 3), "day"},
		{dayLayout("Day", 4, 3), "day"},
		{dayLayout("Test", 3, 3, 3), "test"},
		{dayLayout("Day", 6), ""},
		{dayLayout("Paper", 3, 3), ""},
		{[]content.Layout{{Positions: []string{"p1", "p2", "p3"}}, {Positions: []string{"p4", "p5", "p6"}}}, ""},
		{[]content.Layout{{Name: "Day 2", Positions: []string{"p1"}}, {Name: "Day 1", Positions: []string{"p2"}}}, ""},
		{[]content.Layout{{Name: "Day 1", Positions: []string{"p1"}}, {Name: "Test 2", Positions: []string{"p2"}}}, ""},
		{[]content.Layout{{Name: "Day 1", Days: 2, Positions: []string{"p1", "p2"}}, {Name: "Day 2", Positions: []string{"p3"}}}, ""},
	}
	for i, tc := range cases {
		c := &content.Competition{Layout: tc.layout}
		if k := unitKind(c, 2020); k != tc.kind {
			t.Errorf("case %d gives %q, want %q", i, k, tc.kind)
		}
	}
	if k := unitKind(&content.Competition{Layout: dayLayout("Day", 3, 3), Gaps: []int{2020}}, 2020); k != "" {
		t.Errorf("a year with no edition gives %q", k)
	}
	if k := unitKind(nil, 2020); k != "" {
		t.Errorf("no competition gives %q", k)
	}
}

func TestUnitPosition(t *testing.T) {
	three := dayLayout("Day", 3, 3)
	vmo := dayLayout("Day", 4, 3)
	tests := dayLayout("Test", 3, 3)
	cases := []struct {
		layout []content.Layout
		kind   string
		title  string
		n      int
		pos    string
		open   []string
		perDay bool
		ok     bool
	}{
		// The title counts across the days and the number in the day.
		{three, "day", "Vietnam TST 2019 P4", 1, "p4", nil, true, true},
		{three, "day", "Vietnam TST 2019 for the 60th IMO problem 6", 3, "p6", nil, true, true},
		{three, "day", "Vietnam TST 2019 Day 2 P1", 1, "p4", nil, true, true},
		{three, "day", "Vietnam TST 2019 Day II Problem 1", 4, "p4", nil, false, true},
		{three, "day", "Problem 5 (Day 2)", 2, "p5", nil, true, true},
		{three, "day", "P4 (Day 2)", 4, "p4", nil, false, true},
		{three, "day", "CMO 2019 4", 1, "p4", nil, true, true},
		{three, "day", "Vietnam TST 2019 Day 2", 2, "p5", nil, true, true},
		{three, "day", "Vietnam TST 2019", 5, "p5", nil, false, true},
		// The day is not known.
		{three, "day", "Vietnam TST 2019 P1", 1, "", []string{"p1", "p4"}, false, true},
		{three, "day", "Vietnam TST 2019", 2, "", []string{"p2", "p5"}, false, true},
		{three, "day", "", 3, "", []string{"p3", "p6"}, false, true},
		// The title disagrees with the number, or names another part.
		{three, "day", "Vietnam TST 2019 P5", 1, "", nil, false, false},
		{three, "day", "Vietnam TST 2019 Day 1 P4", 1, "", nil, false, false},
		{three, "day", "Vietnam TST 2019 Day 3 P1", 1, "", nil, false, false},
		{three, "day", "Vietnam TST 2019 TST 2 P1", 1, "", nil, false, false},
		{three, "day", "Vietnam TST 2018 P4", 1, "", nil, false, false},
		{three, "day", "Taiwan MO 2019 Indian TST 2011", 1, "", nil, false, false},
		{three, "day", "IMO Shortlist 2019 G4", 1, "", nil, false, false},
		{three, "day", "Vietnam TST 2019 Day 1 P1 Day 2 P1", 1, "", nil, false, false},
		{three, "day", "Indonesia MO 2019 P4 (INAMO 2019/4)", 4, "p4", nil, false, true},
		{three, "day", "Indonesia MO 2019 P2 (INAMO 2019/2)", 2, "", []string{"p2", "p5"}, false, true},
		{three, "day", "INAMO 2019 Day 1 P3 or APMO 2019 P1", 3, "", nil, false, false},
		{three, "day", "Vietnam TST 2019", 7, "", nil, false, false},
		// Days of different sizes.
		{vmo, "day", "Vietnam MO 2019 problem 5 day 2", 1, "p5", nil, true, true},
		{vmo, "day", "Vietnam MO 2019 Day 2", 3, "p7", nil, true, true},
		{vmo, "day", "Vietnam MO 2019 P4", 4, "p4", nil, false, true},
		{vmo, "day", "Vietnam MO 2019 P1", 1, "", []string{"p1", "p5"}, false, true},
		{vmo, "day", "Vietnam National MO 2019 P7 Day 1", 3, "", nil, false, false},
		// Tests.
		{tests, "test", "Romania TST 2019 Test 2 Problem 1", 1, "p4", nil, true, true},
		{tests, "test", "TST 2 Problem 3", 6, "p6", nil, false, true},
		{tests, "test", "Romania TST 2019 Test 2 Day 1 P1", 1, "", nil, false, false},
	}
	for _, tc := range cases {
		u := unitPosition(tc.layout, tc.kind, tc.title, 2019, tc.n)
		if u.pos != tc.pos || !slices.Equal(u.open, tc.open) || u.perDay != tc.perDay || u.ok != tc.ok {
			t.Errorf("unitPosition(%q, %d) = %q, %v, %v, %v, want %q, %v, %v, %v", tc.title, tc.n, u.pos, u.open, u.perDay, u.ok, tc.pos, tc.open, tc.perDay, tc.ok)
		}
	}
}

// TestTopicsDays checks the rows of the years of a competition with days: the title gives the day, the number column counts in the day or across the days, and a row with no day gets a position only when its year is numbered across the days or by elimination.
func TestTopicsDays(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	c := corpus.Competition("imo")
	c.Eras = append(c.Eras, content.Era{From: 2031, To: 2039, Layout: dayLayout("Day", 3, 3)})
	tp := &Topics{Aliases: al, Corpus: corpus}
	url := "https://artofproblemsolving.com/community/c6h"
	type row struct {
		year   int
		number string
		topic  int
		title  string
		want   string
	}
	rows := []row{
		// 2031 is numbered by day and the titles count across the days. The rows of Day 1 get their positions by elimination.
		{2031, "1", 1, "IMO 2031 P1", "p1"},
		{2031, "1", 2, "IMO 2031 P4", "p4"},
		{2031, "2", 3, "IMO 2031 P2", "p2"},
		{2031, "2", 4, "IMO 2031 Day 2 P2", "p5"},
		{2031, "3", 5, "", "p3"},
		{2031, "3", 6, "Problem 6 (Day 2)", "p6"},
		// 2032 is numbered by day and no title names the day or counts across the days, so no row gets a position.
		{2032, "1", 11, "IMO 2032", ""},
		{2032, "1", 12, "IMO 2032", ""},
		{2032, "2", 13, "IMO 2032 P2", ""},
		{2032, "2", 14, "IMO 2032 P2", ""},
		// 2033 is numbered across the days.
		{2033, "1", 21, "", "p1"},
		{2033, "2", 22, "IMO 2033", "p2"},
		{2033, "5", 23, "IMO 2033 P5", "p5"},
		{2033, "6", 24, "", "p6"},
		// 2034: a title shows that the year is numbered by day. The row of number 2 can be p2 or p5 and no other topic takes either, so it gets none.
		{2034, "1", 31, "IMO 2034 Day 2 P1", "p4"},
		{2034, "2", 32, "IMO 2034", ""},
		// 2035: two rows with no day are left the same position, so neither gets it.
		{2035, "1", 41, "IMO 2035 P4", "p4"},
		{2035, "1", 42, "IMO 2035", ""},
		{2035, "1", 43, "IMO 2035 P1", ""},
		// 2036: the title names another part and disagrees with the number, so the topic gets nothing and still claims p3. The other row of number 3 is then left no position.
		{2036, "3", 51, "IMO Shortlist 2036 problem 5", ""},
		{2036, "3", 52, "IMO 2036 P6", "p6"},
		{2036, "3", 53, "IMO 2036", ""},
		// 2037: no position of the layout agrees with the title and the number, so the title is read as in a year with no days.
		{2037, "3", 61, "IMO 2037 P5", "p5"},
		// 2038: the titles that also name the shortlist are not read, and their numbers do not tell the day, so neither row of number 1 is placed.
		{2038, "1", 71, "IMO 2038 P1 IMO Shortlist 2038 P6", ""},
		{2038, "1", 72, "IMO 2038 Day 2 P4 IMO ShortList 2038 P10", ""},
		{2038, "2", 73, "IMO 2038 Day 1 P2", "p2"},
		// 2039: a title that names another year gives only its number, and the other position of the number is taken.
		{2039, "3", 81, "IMO 2039 Day 2 P3", "p6"},
		{2039, "3", 82, "IMO Nanjing 2038", "p3"},
	}
	for _, r := range rows {
		tp.Add(seed.Row{Labels: []string{"IMO"}, Year: r.year, Number: r.number, Ref: url + strconv.Itoa(r.topic) + "p1", Title: r.title})
	}
	for _, r := range rows {
		year, _, pos, ok := tp.Place(seed.Row{Labels: []string{"IMO"}, Year: r.year, Number: r.number, Ref: url + strconv.Itoa(r.topic) + "p1", Title: r.title})
		if pos != r.want || ok != (r.want != "") || (ok && year != r.year) {
			t.Errorf("%d %s %q gives %d %q %v, want %q", r.year, r.number, r.title, year, pos, ok, r.want)
		}
		if p, _ := tp.Position("aops-topic:"+strconv.Itoa(r.topic), "imo", r.year); p != r.want {
			t.Errorf("topic %d gives %q, want %q", r.topic, p, r.want)
		}
	}
	// A topic with an appearance in another year, as a problem of the IMO has in the shortlist, has no single appearance, but its row of 2033 still takes p1.
	other := seed.Row{Labels: []string{"IMO"}, Year: 2033, Number: "1", Ref: url + "21p1"}
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2040, Number: "3", Ref: url + "21p1", Title: "IMO 2040 P3"})
	if _, _, pos, ok := tp.Place(other); !ok || pos != "p1" {
		t.Errorf("the row of a topic with two appearances gives %q %v", pos, ok)
	}
	if p, _ := tp.Position("aops-topic:21", "imo", 2033); p != "" {
		t.Errorf("a topic with two appearances gives %q", p)
	}
	if tp.Claimed("imo/2031/p1", "aops-topic:1") || !tp.Claimed("imo/2031/p1", "aops-topic:2") {
		t.Error("the claims of the resolved rows are wrong")
	}
	if y := tp.Year("aops-topic:5", "imo"); y != 2031 {
		t.Errorf("the year of a resolved topic is %d", y)
	}
}
