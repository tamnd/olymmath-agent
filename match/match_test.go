package match

import (
	"fmt"
	"testing"
)

const p1 = "Determine all real numbers $\\alpha$ such that, for every positive integer $n$, the integer $\\lfloor \\alpha \\rfloor + \\lfloor 2\\alpha \\rfloor + \\cdots + \\lfloor n\\alpha \\rfloor$ is a multiple of $n$."

func TestJaccard(t *testing.T) {
	a := Shingles(p1)
	if j := Jaccard(a, a); j != 1 {
		t.Errorf("self %v", j)
	}
	b := Shingles("Determine all real numbers $\\alpha$ such that, for every positive integer $n$, the integer $\\lfloor \\alpha \\rfloor + \\lfloor 2\\alpha \\rfloor + \\dots + \\lfloor n\\alpha \\rfloor$ is a multiple of $n$.")
	if j := Jaccard(a, b); j < 0.7 || j == 1 {
		t.Errorf("near copy %v", j)
	}
	c := Shingles("Let $ABC$ be a triangle with incentre $I$. Prove that $AI$ bisects the angle $BAC$.")
	if j := Jaccard(a, c); j > 0.1 {
		t.Errorf("other problem %v", j)
	}
	if len(Shingles("x+y")) != 1 {
		t.Error("a short text gives one shingle")
	}
}

func TestDecide(t *testing.T) {
	cases := []struct {
		s    Signals
		want Decision
	}{
		{Signals{K: StrongKey, T: 0.9}, SameAuto},
		{Signals{K: StrongKey}, SameAuto},
		{Signals{K: StrongKey, T: 0.2}, Review},
		{Signals{T: 0.85}, SameAuto},
		{Signals{T: 0.85, A: AnswerMismatch}, Review},
		{Signals{K: WeakKey, A: AnswerMismatch}, Review},
		{Signals{T: 0.3}, New},
		{Signals{}, New},
	}
	for _, c := range cases {
		if got := Decide(c.s); got != c.want {
			t.Errorf("Decide(%+v) = %v, want %v", c.s, got, c.want)
		}
	}
	if !Strict(Signals{K: StrongKey, T: 0.9}) || Strict(Signals{K: StrongKey}) {
		t.Error("Strict")
	}
	// E counts as a text match only with a strong key.
	if !Strict(Signals{K: StrongKey, T: 0.2, E: 0.8}) || Strict(Signals{K: StrongKey, E: 0.6}) || Strict(Signals{A: AnswerMatch, E: 0.9}) {
		t.Error("Strict with E")
	}
}

func TestAnswers(t *testing.T) {
	if Answers("$042$", "42.") != AnswerMatch || Answers("\\boxed{\\dfrac12}", "\\frac12") != AnswerMatch || Answers("5", "6") != AnswerMismatch || Answers("", "6") != NoAnswer {
		t.Error("Answers")
	}
}

func TestIndex(t *testing.T) {
	x := NewIndex()
	x.Add("a", Shingles(p1))
	x.Add("b", Shingles(p1+" Justify your answer."))
	for i := range 50 {
		x.Add(fmt.Sprint("o", i), Shingles(fmt.Sprintf("Find the number of subsets of a set with %d elements whose sum is divisible by %d.", i+3, i+7)))
	}
	hits := x.Near(Shingles(p1), 0.8)
	if len(hits) != 2 {
		t.Errorf("hits %v", hits)
	}
	n := 0
	x.Pairs(0.8, func(a, b string, _ float64) {
		if (a == "a" && b == "b") || (a == "b" && b == "a") {
			n++
		}
	})
	if n != 1 {
		t.Errorf("pair found %d times", n)
	}
}
