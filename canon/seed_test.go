package canon

import (
	"maps"
	"strconv"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
)

func TestSeedsLocate(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	text := "Let $n$ be a positive integer. Prove that the sum of the first $n$ odd numbers is a square."
	s := &Seeds{Corpus: corpus, Aliases: al, Locate: func(code string, year int, _ string) string {
		if code == "imo" && year == 2023 {
			return "p4"
		}
		return ""
	}}
	traced := map[string]string{}
	s.Trace = func(r seed.Row, code, at, reason string) {
		traced[strconv.Itoa(r.Year)+"/"+r.Number] = code + " " + at + " " + reason
	}
	s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO"}, Year: 2023, Statement: text})
	s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO"}, Year: 2022, Statement: text})
	s.Add(seed.Row{Source: "imo-official", Author: "evanchen", Labels: []string{"IMO"}, Year: 2023, Number: "5", Statement: text})
	got := s.Candidates()
	if len(got["imo/2023/p4"]) != 1 || len(got["imo/2023/p5"]) != 0 || s.Located != 1 || s.Skipped["no valid position"] != 1 || s.Skipped["text in zone Z3"] != 1 {
		t.Errorf("candidates %v, located %d, skipped %v", got, s.Located, s.Skipped)
	}
	want := map[string]string{"2023/": "imo imo/2023/p4 ", "2022/": "imo  no valid position", "2023/5": "imo  text in zone Z3"}
	if !maps.Equal(traced, want) {
		t.Errorf("traced %v, want %v", traced, want)
	}
}

func TestIsRef(t *testing.T) {
	refs := []string{
		"Kvant M1467 (Sol: 3/1995) IMO 1994 P1 IMO Shortlist 1994 A2",
		"Tuymaada 2002 day 1 problem 1. Author : A. Golovanov.",
		"2013 All Russian Olympiad Final Round grade 10 Day 2 P:6",
		"St. Petersburg Olympiad 2015 2nd Round Grade 9 also known as Yellowstone Permutation",
	}
	for _, s := range refs {
		if !isRef(s) {
			t.Errorf("isRef(%q) = false", s)
		}
	}
	texts := []string{
		"Prove that there exists a triangle which can be cut into 2005 congruent triangles.",
		"Is it possible to find 2005 different positive square numbers such that their sum is also a square number?",
		"On day 1 of 2020, prove that $n$ is odd.",
	}
	for _, s := range texts {
		if isRef(s) {
			t.Errorf("isRef(%q) = true", s)
		}
	}
}

func TestTopics(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	tp := &Topics{Aliases: al}
	url := "https://artofproblemsolving.com/community/c6h"
	// The second IMO paper of 2024 numbers its problems from 1.
	corpus.Competition("imo").Layout = []content.Layout{{Positions: []string{"p1", "p2", "p3"}}, {Positions: []string{"p4", "p5", "p6"}}}
	tp.Corpus = corpus
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2024, Number: "1", Ref: url + "200p1", Statement: "IMO 2024 Day 1 Problem 1"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2024, Number: "1", Ref: url + "201p1", Statement: "IMO 2024 Day 2 Problem 1"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2023, Number: "4", Ref: url + "100p1"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2023, Number: "1", Ref: url + "101p2"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2023, Number: "1", Ref: url + "102p3"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "2", Ref: url + "103p4"})
	// A row of a contest list has no year. Its title gives the year.
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "3", Ref: url + "300p1", Statement: "IMO 2021 #3"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "2", Ref: url + "301p1", Statement: "Germany 2013/5"})
	pos := func(ref string, year int) string {
		p, _ := tp.Position(ref, "imo", year)
		return p
	}
	if got := pos("aops-topic:100", 2023); got != "p4" {
		t.Errorf("topic 100 gives %q", got)
	}
	if a, b := pos("aops-topic:200", 2024), pos("aops-topic:201", 2024); a != "p1" || b != "p4" {
		t.Errorf("the topics of the two papers give %q and %q", a, b)
	}
	if got := pos("aops-topic:100", 2022); got != "" {
		t.Errorf("topic 100 in 2022 gives %q", got)
	}
	if got := pos("aops-topic:300", 2021); got != "p3" {
		t.Errorf("the topic of a title gives %q", got)
	}
	if y := tp.Year("aops-topic:300", "imo"); y != 2021 {
		t.Errorf("the year of topic 300 is %d", y)
	}
	if y := tp.Year("aops-topic:101", "imo") + tp.Year("aops-topic:300", "usamo") + tp.Year("x", "imo"); y != 0 {
		t.Errorf("unknown topics give the year %d", y)
	}
	for _, ref := range []string{"aops-topic:101", "aops-topic:103", "aops-topic:301", "aops-topic:999", "x"} {
		for _, y := range []int{2013, 2023} {
			if got := pos(ref, y); got != "" {
				t.Errorf("%s gives %q", ref, got)
			}
		}
	}
}

// TestTopicsLonglistTitle checks that a row of a shortlist whose title gives the number of a longlist problem gives no position.
func TestTopicsLonglistTitle(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	corpus.Competition("imo").TopicPositions = true
	tp := &Topics{Aliases: al, Corpus: corpus}
	url := "https://artofproblemsolving.com/community/c6h"
	rows := []seed.Row{
		{Labels: []string{"IMO"}, Year: 1966, Number: "24", Ref: url + "400p1", Title: "IMO LongList 1959 1966 P24"},
		{Labels: []string{"IMO"}, Year: 1970, Number: "27", Ref: url + "401p1", Title: "ILL 1970   P27."},
		{Labels: []string{"IMO"}, Year: 1988, Number: "15", Ref: url + "402p1", Title: "IMO ShortList 1988 P15 Iceland 1 P34 of ILL"},
		{Labels: []string{"IMO"}, Year: 1983, Number: "11", Ref: url + "403p1", Title: "IMO Longlist 1983"},
	}
	for _, r := range rows {
		tp.Add(r)
	}
	want := []string{"", "", "p15", "p11"}
	for i, r := range rows {
		_, _, got, _ := tp.Place(r)
		if got != want[i] {
			t.Errorf("Place(%q) = %q, want %q", r.Title, got, want[i])
		}
	}
}

func TestTopicsLetter(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	c := corpus.Competition("imo")
	c.EditionForm = "year-letter"
	c.Layout = nil
	tp := &Topics{Aliases: al, Corpus: corpus}
	url := "https://artofproblemsolving.com/community/c5h"
	// Problem 1 of the A paper and problem 1 of the B paper are two appearances.
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "1", Ref: url + "1p1", Statement: "2014 AMC 10A #1/ 12A #1"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "1", Ref: url + "2p1", Statement: "AMC 10B 2014 P1"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "2", Ref: url + "3p1", Statement: "2021 AMC 10A/12A (Fall) #2"})
	if p, l := tp.Position("aops-topic:1", "imo", 2014); p != "p1" || l != "a" {
		t.Errorf("topic 1 gives %q %q", p, l)
	}
	if p, l := tp.Position("aops-topic:2", "imo", 2014); p != "p1" || l != "b" {
		t.Errorf("topic 2 gives %q %q", p, l)
	}
	if p, _ := tp.Position("aops-topic:3", "imo", 2021); p != "" {
		t.Errorf("a title with a season gives %q", p)
	}
	text := "What is the value of $1+2+3+4$, the sum of the first four positive integers?"
	s := &Seeds{Corpus: corpus, Aliases: al, Topics: tp}
	s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO"}, Year: 2014, Ref: "aops-topic:2", Statement: text})
	s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO", "2014 IMO Spring"}, Year: 2014, Ref: "aops-topic:1", Statement: text})
	if got := s.Candidates(); len(got["imo/2014b/p1"]) != 1 || len(got["imo/2014a/p1"]) != 0 || s.Skipped["no edition letter"] != 1 {
		t.Errorf("candidates %v, skipped %v", got, s.Skipped)
	}
}

func TestTitleAt(t *testing.T) {
	c := &content.Competition{Layout: []content.Layout{
		{Paper: "junior", Name: "Junior", Positions: []string{"j1", "j2", "j3", "j4"}},
		{Paper: "senior", Name: "Senior", Positions: []string{"s1", "s2", "s3", "s4"}},
	}}
	cases := []struct {
		title, pos string
		year       int
		at         string
		ok         bool
	}{
		{"2017 AMC 8 #4", "p4", 2017, "p4", true},
		{"Bundeswettbewerb Mathematik 2023 Round 1 P1", "p1", 2023, "p1", true},
		{"USEMO 2019/2", "p2", 2019, "p2", true},
		{"2023 Olympic Revenge problem 2", "p2", 2023, "p2", true},
		{"P3 Francophone MO Junior 2023", "p3", 2023, "j3", true},
		{"2021 Francophone MO Seniors p4", "p4", 2021, "s4", true},
		// The number column and the title do not agree.
		{"Germany 2013/5", "p2", 0, "", false},
		// No number, no year, two years, a later round, a statement.
		{"2015 Brazilian Olympic Revenge", "p1", 0, "", false},
		{"25th Olympic Revenge", "p1", 0, "", false},
		{"ITAMO 1998 p2 Bundeswettbewerb Mathematik 1975 R1 p2", "p2", 0, "", false},
		{"Bundeswettbewerb Mathematik 2023 Round 2 P1", "p1", 0, "", false},
		{"Find all $n$ such that $2023 \\mid n$ #1", "p1", 0, "", false},
	}
	for _, tc := range cases {
		year, at, _, ok := TitleAt(c, tc.title, tc.pos)
		if year != tc.year || at != tc.at || ok != tc.ok {
			t.Errorf("TitleAt(%q, %q) = %d, %q, %v, want %d, %q, %v", tc.title, tc.pos, year, at, ok, tc.year, tc.at, tc.ok)
		}
	}
}

func TestPaperPosition(t *testing.T) {
	c := &content.Competition{Layout: []content.Layout{{Positions: []string{"p1", "p2", "p3"}}, {Positions: []string{"p4", "p5", "p6"}}}}
	cases := []struct {
		text, pos, rest string
		ok              bool
	}{
		{"2006 IrMO Paper 2 P1", "p4", "", true},
		{"1990 IrMO Paper 1 P3\nLet $n$ be odd.", "p3", "Let $n$ be odd.", true},
		{"IMO 2023 Day 2, Problem 5", "p5", "", true},
		{"Paper 1 P6", "", "", true},
		{"Paper 3 P1", "", "", true},
		{"Tuymaada 2009 Senior League Second Day P2", "p5", "", true},
		{"Zhautykov 2009 day 2 problem 4.", "p4", "", true},
		{"Let $n$ be odd.", "", "Let $n$ be odd.", false},
		{"On day 1 problem 2 of the test, Ana wrote $n$.", "", "On day 1 problem 2 of the test, Ana wrote $n$.", false},
	}
	for _, tc := range cases {
		pos, rest, ok := PaperPosition(c, 2006, tc.text)
		if pos != tc.pos || rest != tc.rest || ok != tc.ok {
			t.Errorf("PaperPosition(%q) = %q, %q, %v, want %q, %q, %v", tc.text, pos, rest, ok, tc.pos, tc.rest, tc.ok)
		}
	}
	if _, _, ok := PaperPosition(&content.Competition{}, 2006, "Paper 1 P1"); ok {
		t.Error("a competition with no layout reads the paper line")
	}
	c.Eras = []content.Era{{From: 1990, To: 1999, Layout: []content.Layout{{Positions: []string{"p1", "p2", "p3", "p4"}}, {Positions: []string{"p5", "p6", "p7", "p8"}}}}}
	if pos, _, _ := PaperPosition(c, 1995, "Paper 2 P1"); pos != "p5" {
		t.Errorf("in an era, paper 2 P1 gives %q", pos)
	}
	if pos, _, _ := PaperPosition(c, 2000, "Paper 2 P1"); pos != "p4" {
		t.Errorf("after the era, paper 2 P1 gives %q", pos)
	}
	// Day 2 of the IMO 1960 has four problems, so "Day 2 P4" can be p4 or p7.
	c.Eras = append(c.Eras, content.Era{From: 1960, To: 1960, Layout: []content.Layout{{Positions: []string{"p1", "p2", "p3"}}, {Positions: []string{"p4", "p5", "p6", "p7"}}}})
	for text, want := range map[string]string{"Day 2 P1": "p4", "Day 2 P4": "", "Day 2 P5": "p5", "Day 2 P7": "p7"} {
		if pos, _, ok := PaperPosition(c, 1960, text); pos != want || !ok {
			t.Errorf("in 1960, %q gives %q, %v, want %q", text, pos, ok, want)
		}
	}
}

func TestNamedPaper(t *testing.T) {
	memo := &content.Competition{Layout: []content.Layout{
		{Paper: "individual", Name: "Individual", Positions: []string{"i1", "i2", "i3", "i4"}},
		{Paper: "team", Name: "Team", Positions: []string{"t1", "t2", "t3", "t4", "t5", "t6", "t7", "t8"}},
	}}
	tuymaada := &content.Competition{Layout: []content.Layout{
		{Paper: "junior", Name: "Junior", Days: 2, Positions: []string{"j1", "j2", "j3", "j4", "j5", "j6", "j7", "j8"}},
		{Paper: "senior", Name: "Senior", Days: 2, Positions: []string{"s1", "s2", "s3", "s4", "s5", "s6", "s7", "s8"}},
	}}
	cases := []struct {
		c          *content.Competition
		title, pos string
		at         string
		ok         bool
	}{
		{memo, "Middle European MO 2013 T 5", "p5", "t5", true},
		{memo, "2019 MEMO PI 3", "p3", "i3", true},
		{memo, "MEMO 2016,Team P2", "p2", "t2", true},
		{memo, "MEMO 2008 (Team) P5", "p1", "t1", true},
		{memo, "MEMO 2009 problem 1 team competition", "p5", "t1", true},
		{memo, "MEMO 2010 Team P5", "p2", "", true},
		{memo, "MEMO 2010", "p2", "", false},
		{tuymaada, "Tuymaada 2014 Day 2 P1 Junior League", "p5", "j5", true},
		{tuymaada, "Tuymaada 2012 P5 Day 2 Juniors", "p1", "j5", true},
		{tuymaada, "Tuymaada 2017 Seniors Q8 (day 2 P4)", "p8", "s8", true},
		{tuymaada, "Tuymaada 2015 Day 1 P6 Junior League", "p6", "", true},
		{tuymaada, "Tuymaada 2016 Senior P2 Junior P3", "p2", "", false},
	}
	for _, tc := range cases {
		at, ok := namedPaper(tc.c, 2020, tc.title, tc.pos)
		if at != tc.at || ok != tc.ok {
			t.Errorf("namedPaper(%q, %q) = %q, %v, want %q, %v", tc.title, tc.pos, at, ok, tc.at, tc.ok)
		}
	}
	for _, tc := range []struct{ text, pos string }{
		{"Tuymaada 2009 Senior League Second Day P2", "s6"},
		{"Tuymaada 2009 Junior League Day 1 P3", "j3"},
		{"Tuymaada 2009 Junior League Day 2 P7", "j7"},
		{"Tuymaada 2009 Day 2 P1", ""},
	} {
		if pos, _, _ := PaperPosition(tuymaada, 2009, tc.text); pos != tc.pos {
			t.Errorf("PaperPosition(%q) = %q, want %q", tc.text, pos, tc.pos)
		}
	}
}

func TestValidPositionEra(t *testing.T) {
	c := &content.Competition{
		Eras:    []content.Era{{From: 1959, To: 2005, PerYear: 30}, {From: 2006, To: 2006}},
		Layout:  []content.Layout{{Positions: []string{"a1", "c1", "g1", "n1"}}},
		PerYear: 4,
	}
	for _, tc := range []struct {
		year int
		pos  string
		want bool
	}{
		{1990, "p30", true}, {1990, "a1", false}, {2010, "a1", true}, {2010, "p1", false}, {2006, "p99", true},
	} {
		if got := validPosition(c, tc.year, tc.pos); got != tc.want {
			t.Errorf("validPosition(%d, %s) = %v", tc.year, tc.pos, got)
		}
	}
}

func TestTitlePosition(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	imo := corpus.Competition("imo")
	isl := &content.Competition{Code: "isl", TopicPositions: true}
	cases := []struct {
		c    *content.Competition
		year int
		pos  string
		text string
		want string
	}{
		{imo, 1982, "p2", "IMO 1982 P5", "p5"},
		{imo, 2004, "p2", "International Mathematical Olympiad 2004 problem 6.", "p6"},
		{imo, 2004, "p2", "IMO 2004 #3", "p3"},
		{imo, 2004, "", "IMO 2004 P3", "p3"},
		{imo, 2004, "p2", "IMO 2005 P3", ""},
		{imo, 2004, "p2", "IMO 2004 Day 2 P3", ""},
		{imo, 2004, "p2", "IMO Shortlist 2003 G4 IMO 2004 P3", ""},
		{imo, 2004, "p2", "IMO 2004 Junior P3", ""},
		{imo, 2004, "p2", "Practice Test 1 2004 P3", ""},
		{imo, 2004, "p2", "IMO 2004", ""},
		{imo, 2004, "p2", "IMO 2004 P3\nLet $n$ be odd.", ""},
		{imo, 2004, "a2", "IMO 2004 P3", ""},
		{isl, 2011, "2", "IMO Shortlist 2011 G2", "g2"},
		{isl, 2004, "3", "Moldova TST 2005 IMO Shortlist 2004 geometry problem 3", "g3"},
		{isl, 2011, "2", "IMO Shortlist 2011 Algebra 2", "a2"},
		{isl, 1994, "5", "IMO 1994 P3 IMO Shortlist 1994 N5", "n5"},
		{isl, 2006, "2", "IMO Shortlist 2006 G4 A2", ""},
		{isl, 2011, "2", "Romania 2011 G2", ""},
		{isl, 1990, "12", "IMO ShortList 1990 P12 (IRE 1)", ""},
	}
	for _, tc := range cases {
		if got := TitlePosition(tc.c, al, tc.year, tc.pos, tc.text); got != tc.want {
			t.Errorf("TitlePosition(%q) = %q, want %q", tc.text, got, tc.want)
		}
	}
	// The wiki numbers each day from 1, so two topics claim P2. The titles give P2 and P5.
	tp := &Topics{Aliases: al, Corpus: corpus}
	url := "https://artofproblemsolving.com/community/c6h"
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 1982, Number: "2", Ref: url + "300p1", Statement: "IMO 1982 P2"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 1982, Number: "2", Ref: url + "301p1", Statement: "IMO 1982 P5"})
	// A wiki row with no year takes the year of its title.
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "1", Ref: url + "302p1", Statement: "IMO 1983 P4"})
	a, _ := tp.Position("aops-topic:300", "imo", 1982)
	b, _ := tp.Position("aops-topic:301", "imo", 1982)
	c, _ := tp.Position("aops-topic:302", "imo", 1983)
	if a != "p2" || b != "p5" || c != "p4" {
		t.Errorf("the topics give %q, %q and %q", a, b, c)
	}
	// A title that names a part gives no position. When its number does not agree with the number of the row, the topic gets no position.
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2011, Number: "1", Ref: url + "303p1", Statement: "IMO 2011 TST problem 5"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2011, Number: "2", Ref: url + "304p1", Statement: "IMO 2011 TST problem 2"})
	d, _ := tp.Position("aops-topic:303", "imo", 2011)
	e, _ := tp.Position("aops-topic:304", "imo", 2011)
	if d != "" || e != "p2" {
		t.Errorf("the topics with a part give %q and %q", d, e)
	}
}

// A row of the AoPS wiki with a title takes the place of its title, as its topic does. A row whose title names another number gives no candidate, and a row whose topic claims an appearance with another topic gives no candidate.
func TestSeedsTitle(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	s := &Seeds{Corpus: corpus, Aliases: al, Topics: &Topics{Aliases: al, Corpus: corpus}}
	text := "Let $n$ be a positive integer. Prove that the sum of the first $n$ odd numbers is a square."
	url := "https://artofproblemsolving.com/community/c6h"
	rows := []seed.Row{
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 1982, Number: "2", Ref: url + "301p1", Title: "IMO 1982 P5", Statement: text},
		{Source: "imo-official", Labels: []string{"IMO"}, Number: "1", Ref: url + "302p1", Title: "IMO 1983 P4", Statement: text},
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2011, Number: "1", Ref: url + "303p1", Title: "IMO 2011 TST problem 5", Statement: text},
		// Two topics with the number 1 in one year, as when the wiki numbers each day from 1, give no candidate.
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 1984, Number: "1", Ref: url + "304p1", Title: "IMO 1984", Statement: text},
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 1984, Number: "1", Ref: url + "305p1", Title: "IMO 1984", Statement: text},
		// A row with no title and a number that another topic claims gives no candidate.
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 1985, Number: "1", Ref: url + "306p1", Title: "IMO 1985 P1", Statement: text},
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 1985, Number: "1", Ref: url + "307p1", Statement: text},
		// A row with no title and a number that no other topic claims gives a candidate.
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 1985, Number: "2", Ref: url + "308p1", Statement: text},
	}
	for _, r := range rows {
		s.Topics.Add(r)
	}
	for _, r := range rows {
		s.Add(r)
	}
	got := s.Candidates()
	if len(got["imo/1982/p5"]) != 1 || len(got["imo/1982/p2"]) != 0 || len(got["imo/1983/p4"]) != 1 || len(got["imo/2011/p1"]) != 0 || len(got["imo/1984/p1"]) != 0 || len(got["imo/1985/p1"]) != 0 || len(got["imo/1985/p2"]) != 1 || s.Skipped["no position from the title"] != 5 {
		t.Errorf("candidates %v, skipped %v", got, s.Skipped)
	}
}

// TestGradePosition checks the rows of a competition with one paper for each grade, where the wiki numbers the problems of each grade from 1.
func TestGradePosition(t *testing.T) {
	grade := func(g, n, days int) content.Layout {
		l := content.Layout{Paper: "grade" + strconv.Itoa(g), Name: "Grade " + strconv.Itoa(g), Days: days}
		for i := 1; i <= n; i++ {
			l.Positions = append(l.Positions, "p"+strconv.Itoa(g)+strconv.Itoa(i))
		}
		return l
	}
	c := &content.Competition{
		Code: "xk-mo",
		Eras: []content.Era{
			{From: 2019, To: 2019, Layout: []content.Layout{grade(9, 5, 0), grade(10, 5, 0), grade(11, 5, 0), grade(12, 5, 0)}},
			{From: 2002, To: 2002, Layout: []content.Layout{grade(7, 8, 2), grade(11, 8, 2)}},
		},
	}
	corr := content.Layout{Paper: "correspondence", Name: "Correspondence"}
	for i := 1; i <= 24; i++ {
		corr.Positions = append(corr.Positions, "p"+strconv.Itoa(i))
	}
	c.Eras = append(c.Eras, content.Era{From: 2017, To: 2017, Layout: []content.Layout{corr, grade(8, 8, 2), grade(9, 8, 2), grade(10, 8, 2)}})
	cases := []struct {
		year          int
		number, title string
		pos           string
		graded        bool
	}{
		// A title that names the day counts its number in the day or in the paper, and the number column must agree.
		{2017, "8", "Sharygin 2017 Finals Day 2 Grade 10 P4", "p108", true},
		{2017, "4", "Sharygin 2017 Finals Day 2 Grade 10 P4", "p108", true},
		{2017, "8", "Sharygin 2017 Day 2 P10.8 Grade 10", "p108", true},
		{2017, "7", "Sharygin 2017 Grade 9 Day 2 P9.7", "p97", true},
		{2017, "5", "Sharygin contest 2017. Grade 10. Second day. P5", "p105", true},
		{2017, "6", "Sharygin 2017 Finals Day 2 Grade 10 P4", "", true},
		{2017, "4", "Sharygin 2017 Finals Day 1 Grade 10 P5", "", true},
		// The mark of "P9.8" names the grade, and a number of the grade that counts in the day agrees with the number column.
		{2017, "8", "Sharygin Finals 2017 P9.8", "p98", true},
		{2017, "1", "Sharygin Finals 2017 9.5", "p95", true},
		{2017, "8", "Sharygin Finals 2017 P9.7", "", true},
		// A title that names another paper and no grade is read by the other rules. A title that names no paper gives no position.
		{2017, "20", "2017 Sharygin Geometry Olympiad Correspondence Round P20", "", false},
		{2017, "4", "Sharygin final round 2017", "", true},
		{2017, "14", "Sharygin Geometry Olympiad Correspondence Round 2017 P 14 (Grade 9 11)", "", false},
		{2017, "6", "Sharygin Geometry Olympiad Correspondence round 2017 P 6 Grade 8", "", false},
		// A grade in brackets is the grade that a problem of the other paper is meant for.
		{2017, "1", "Sharygin 2017   P1 (Grade 8)", "", true},
		// A number that no paper of a grade has is a problem of the other paper.
		{2017, "23", "Sharygin 2017   P23 (Grade 10 11)", "", false},
		{2017, "24", "Sharygin Geometry Olympiad 2017    P24", "", false},
		{2017, "24", "Sharygin Geometry Olympiad 2017    P4", "", true},
		{2017, "8", "Sharygin 2017   P8 (Grade (a)8 9 (b)10 11)", "", true},
		{2017, "3", "Sharygin 2017 Final Round Grade 8 P3", "p83", true},
		// The number after a mark is not read as a grade, so "problem 1 grade 8" names grade 8.
		{2017, "1", "2017 Sharygin Geometry Olympiad Final Round problem 1 grade 8", "p81", true},
		{2017, "6", "Sharygin Geometry Olympiad 2017 Final Round problem 6 grade 10", "p106", true},
		{2017, "2", "2017 Sharygin Geometry Olympiad Final Round problem 1 grade 8", "", true},
		// The first round is the other paper, so its grade is the grade that the problem is meant for.
		{2017, "P1", "Sharygin Geometry Olympiad 2017 First Round P1  grade8", "", false},
		{2019, "1", "Kosovo MO 2019 Grade 10 P1", "p101", true},
		{2019, "5", "Kosovo MO 2019 Grade 9 P5", "p95", true},
		{2019, "3", "Kosovo MO 2019 grade XI problem 3", "p113", true},
		{2019, "2", "Kosovo MO 2019 12th Grade P2", "p122", true},
		{2019, "10.3", "2019 Kosovo MO 10.3", "p103", true},
		{2019, "10.3", "2019 Kosovo MO", "p103", true},
		// A row with no grade, a grade with no paper, two grades or a number that does not agree gives no position.
		{2019, "3", "", "", true},
		{2019, "5", "Kosovo MO 2019 P5", "", true},
		{2019, "1", "Kosovo MO 2019 Grade 8 P1", "", true},
		{2019, "1", "Kosovo MO 2019 Grade 9 P1 Grade 10 P1", "", true},
		{2019, "2", "Kosovo MO 2019 Grade 9 P3", "", true},
		{2019, "6", "Kosovo MO 2019 Grade 9 P6", "", true},
		{2019, "10.3", "2019 Kosovo MO 11.3", "", true},
		{2019, "10.3", "2019 Kosovo MO Grade 11", "", true},
		{2019, "1", "Kosovo MO 2019 Grade 9 P1\n$x$", "", true},
		// In a paper of two days, the number column can count from 1 in each day.
		{2002, "1", "Moldova NMO 2002 grade 7 problem nr.5", "p75", true},
		{2002, "4", "Moldova NMO 2002 grade 11 problem nr.8", "p118", true},
		{2002, "3", "Moldova NMO 2002 grade 7 problem nr.3", "p73", true},
		{2002, "2", "Moldova NMO 2002 grade 7 problem nr.5", "", true},
		// A year with no paper of a grade is read by the other rules.
		{2020, "1", "Kosovo MO 2020 Grade 10 P1", "", false},
	}
	for _, tc := range cases {
		pos, graded := gradePosition(c, tc.year, tc.number, tc.title)
		if pos != tc.pos || graded != tc.graded {
			t.Errorf("gradePosition(%d, %q, %q) = %q, %v, want %q, %v", tc.year, tc.number, tc.title, pos, graded, tc.pos, tc.graded)
		}
	}
	if pos, graded := gradePosition(nil, 2019, "1", "Kosovo MO 2019 Grade 10 P1"); pos != "" || graded {
		t.Errorf("a nil competition gives %q, %v", pos, graded)
	}
	if g := titleGrade("Moldova National MO XI Class 2007"); g != 11 {
		t.Errorf("titleGrade gives %d", g)
	}
}

// TestTopicsGrade checks that the rows of the grades of one year get the positions of their grades, also when the row has no year and its title gives it.
func TestTopicsGrade(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	c := corpus.Competition("imo")
	c.Eras = append(c.Eras, content.Era{From: 2030, To: 2030, Layout: []content.Layout{
		{Paper: "grade9", Name: "Grade 9", Positions: []string{"p91", "p92"}},
		{Paper: "grade10", Name: "Grade 10", Positions: []string{"p101", "p102"}},
	}})
	tp := &Topics{Aliases: al, Corpus: corpus}
	url := "https://artofproblemsolving.com/community/c6h"
	rows := []seed.Row{
		{Labels: []string{"IMO"}, Year: 2030, Number: "1", Ref: url + "1p1", Title: "IMO 2030 Grade 9 P1"},
		{Labels: []string{"IMO"}, Number: "1", Ref: url + "1p1", Title: "IMO 2030 Grade 9 P1"},
		{Labels: []string{"IMO"}, Year: 2030, Number: "1", Ref: url + "2p1", Title: "IMO 2030 Grade 10 P1"},
		{Labels: []string{"IMO"}, Year: 2030, Number: "10.2", Ref: url + "3p1", Title: "IMO 2030 10.2"},
		{Labels: []string{"IMO"}, Year: 2030, Number: "2", Ref: url + "4p1"},
	}
	for _, r := range rows {
		tp.Add(r)
	}
	want := []string{"p91", "p91", "p101", "p102", ""}
	for i, r := range rows {
		year, _, pos, ok := tp.Place(r)
		if pos != want[i] || ok != (want[i] != "") || (ok && year != 2030) {
			t.Errorf("row %d gives %d %q %v, want %q", i, year, pos, ok, want[i])
		}
	}
	if p, _ := tp.Position("aops-topic:1", "imo", 2030); p != "p91" {
		t.Errorf("topic 1 gives %q", p)
	}
}

func TestSeasonPosition(t *testing.T) {
	cases := []struct {
		number, title string
		year          int
		season, pos   string
	}{
		{"3", "Tournament of Towns 2024 Fall Senior A-Level P3", 2024, "fall", "sa3"},
		{"5", "Tournament of Towns Fall 2002 Junior A Level P5", 2002, "fall", "ja5"},
		{"7", "Spring 2006 Tournament of Towns Junior A Level #7", 2006, "spring", "ja7"},
		{"4", "Tournament Of Towns Spring 2000 Junior 0 Level p4", 2000, "spring", "jo4"},
		{"P7", "42nd International Tournament of Towns Senior A Level P7 Spring 2021", 2021, "spring", "sa7"},
		{"5", "ToT Junior O Level Autumn 1998", 1998, "fall", "jo5"},
		{"7", "Tournament of Towns 2016 Fall Tour A Senior P#7", 2016, "fall", "sa7"},
		{"7.", "Tournament of Towns Spring 2015 Senior A level", 2015, "spring", "sa7"},
		{"", "Tournament of Towns Junior A Level  Fall 2019 p7", 2019, "fall", "ja7"},
		// The title must name one season, one tier, one level and one number that agrees with the number column.
		{"4", "Tournament of Towns 2007   Spring   Junior A Level   P7", 0, "", ""},
		{"7", "ToT 2003 JA 7 SA 5", 0, "", ""},
		{"7", "Tournament of towns Spring 2018 A level P7", 0, "", ""},
		{"7", "Tournament of Towns Spring 2016", 0, "", ""},
		{"1", "Tournament of Towns 2020 oral p1 (15 March 2020)", 0, "", ""},
		{"5", "Fall 2007 Tournament of Towns Junior P Level #5", 0, "", ""},
		{"5", "Tournament of towns Junior B Level paper Fall 2004", 0, "", ""},
		{"5", "Tournament of Towns Junior A Level", 0, "", ""},
		{"5", "Tournament of Towns 2020/2021 Junior A Level Fall", 0, "", ""},
		{"", "Tournament of Towns Fall 2002 Junior A Level", 0, "", ""},
		{"2", "Tournament of Towns Fall 2002 Junior A Level P2 Senior A Level P1", 0, "", ""},
	}
	for _, tc := range cases {
		year, season, pos, ok := SeasonPosition(tc.number, tc.title)
		if year != tc.year || season != tc.season || pos != tc.pos || ok != (tc.pos != "") {
			t.Errorf("SeasonPosition(%q, %q) = %d %q %q %v, want %d %q %q", tc.number, tc.title, year, season, pos, ok, tc.year, tc.season, tc.pos)
		}
	}
}

func TestTopicsSeason(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if fs.Errors() > 0 {
		t.Fatal(fs)
	}
	al, err := seed.LoadAliases("../content/testdata/valid", corpus)
	if err != nil {
		t.Fatal(err)
	}
	c := corpus.Competition("imo")
	c.EditionForm = "year-season"
	tp := &Topics{Aliases: al, Corpus: corpus}
	url := "https://artofproblemsolving.com/community/c6h"
	text := "Prove that among any $n+1$ of the numbers $1, 2, \\ldots, 2n$ there are two numbers such that one divides the other."
	rows := []seed.Row{
		// The collection year 2020 is the fall of the tournament, and the title names the spring of 2021.
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2020, Number: "P3", Ref: url + "1p1", Title: "42nd International Tournament of Towns Senior A Level P3 Spring 2021", Statement: text},
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2009, Number: "4", Ref: url + "2p1", Title: "Tournament of Towns 2009 Fall Junior O Level P4", Statement: text + " Find all $n$."},
		// A row with no season and two topics that claim one appearance give nothing.
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2009, Number: "5", Ref: url + "3p1", Statement: text + " Find $n$."},
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2010, Number: "1", Ref: url + "4p1", Title: "Tournament of Towns Spring 2010 Senior A Level P1", Statement: text + " Is it true?"},
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2010, Number: "1", Ref: url + "5p1", Title: "ToT Spring 2010 Senior A Level #1", Statement: text + " Is it so?"},
	}
	for _, r := range rows {
		tp.Add(r)
	}
	if p, l := tp.Position("aops-topic:1", "imo", 2021); p != "sa3" || l != "-spring" {
		t.Errorf("topic 1 gives %q %q", p, l)
	}
	s := &Seeds{Corpus: corpus, Aliases: al, Topics: tp}
	for _, r := range rows {
		s.Add(r)
	}
	// A row of the dataset with no number joins its topic, and the year of the title is used.
	s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO"}, Year: 2020, Ref: "aops-topic:1", Statement: text + " Show it."})
	got := s.Candidates()
	if len(got["imo/2021-spring/sa3"]) != 2 || len(got["imo/2009-fall/jo4"]) != 1 || len(got) != 2 {
		t.Errorf("candidates %v, skipped %v", got, s.Skipped)
	}
}
