package canon

import (
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
	s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO"}, Year: 2023, Statement: text})
	s.Add(seed.Row{Source: "imo-official", Labels: []string{"IMO"}, Year: 2022, Statement: text})
	s.Add(seed.Row{Source: "imo-official", Author: "evanchen", Labels: []string{"IMO"}, Year: 2023, Number: "5", Statement: text})
	got := s.Candidates()
	if len(got["imo/2023/p4"]) != 1 || len(got["imo/2023/p5"]) != 0 || s.Located != 1 || s.Skipped["no valid position"] != 1 || s.Skipped["text in zone Z3"] != 1 {
		t.Errorf("candidates %v, located %d, skipped %v", got, s.Located, s.Skipped)
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
	if got := tp.Position("aops-topic:100", "imo", 2023); got != "p4" {
		t.Errorf("topic 100 gives %q", got)
	}
	if a, b := tp.Position("aops-topic:200", "imo", 2024), tp.Position("aops-topic:201", "imo", 2024); a != "p1" || b != "p4" {
		t.Errorf("the topics of the two papers give %q and %q", a, b)
	}
	if got := tp.Position("aops-topic:100", "imo", 2022); got != "" {
		t.Errorf("topic 100 in 2022 gives %q", got)
	}
	for _, ref := range []string{"aops-topic:101", "aops-topic:103", "aops-topic:999", "x"} {
		if got := tp.Position(ref, "imo", 2023); got != "" {
			t.Errorf("%s gives %q", ref, got)
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
