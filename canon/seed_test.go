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
	for _, ref := range []string{"aops-topic:101", "aops-topic:103", "aops-topic:301", "aops-topic:999", "x"} {
		for _, y := range []int{2013, 2023} {
			if got := pos(ref, y); got != "" {
				t.Errorf("%s gives %q", ref, got)
			}
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
