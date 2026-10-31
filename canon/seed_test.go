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
	got := s.Candidates()
	if len(got["imo/2023/p4"]) != 1 || s.Located != 1 || s.Skipped["no valid position"] != 1 {
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
