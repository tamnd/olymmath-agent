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
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2023, Number: "4", Ref: url + "100p1"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2023, Number: "1", Ref: url + "101p2"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Year: 2023, Number: "1", Ref: url + "102p3"})
	tp.Add(seed.Row{Labels: []string{"IMO"}, Number: "2", Ref: url + "103p4"})
	if got := tp.Position("aops-topic:100", "imo", 2023); got != "p4" {
		t.Errorf("topic 100 gives %q", got)
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
