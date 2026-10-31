package similar

import "testing"

func TestFind(t *testing.T) {
	texts := []string{
		"Let ABC be a triangle with incircle omega touching BC at D. Prove that the circumcircle passes through D.",
		"In triangle ABC the incircle omega touches BC at D. Show that the circumcircle of the triangle passes through D.",
		"Find all functions f from the reals to the reals with f(x+y)=f(x)+f(y).",
		"Find all functions f such that f(xy)=f(x)f(y) for all reals.",
		"Prove that there are infinitely many primes p such that p divides 2^n+1.",
	}
	for i := 5; i < 30; i++ {
		texts = append(texts, "Some other unrelated statement about chess boards and coins number "+string(rune('a'+i)))
	}
	got := Find(texts, 2, 0.2)
	if len(got[0]) == 0 || got[0][0].Doc != 1 {
		t.Errorf("0: %+v", got[0])
	}
	if len(got[2]) == 0 || got[2][0].Doc != 3 {
		t.Errorf("2: %+v", got[2])
	}
	for _, m := range got[4] {
		if m.Doc < 5 {
			t.Errorf("4: %+v", got[4])
		}
	}
}

func TestWords(t *testing.T) {
	if got := Words("Let $x$ be a REAL number, and x2 > 0."); len(got) != 1 || got[0] != "x2" {
		t.Errorf("%q", got)
	}
}

func TestNearest(t *testing.T) {
	docs := []string{
		"Let ABC be a triangle with incircle omega touching BC at D. Prove that the circumcircle passes through D.",
		"Find all functions f from the reals to the reals with f(x+y)=f(x)+f(y).",
		"Prove that there are infinitely many primes p such that p divides 2^n+1.",
	}
	for i := range 20 {
		docs = append(docs, "Some other statement about chess boards and coins number "+string(rune('a'+i)))
	}
	got := Nearest(docs, []string{
		"In triangle ABC the incircle omega touches BC at D. Show that the circumcircle passes through D.",
		"Infinitely many primes p divide 2^n+1. Prove it.",
		"zzz qqq",
	})
	if got[0].Doc != 0 || got[0].Score < 0.5 {
		t.Errorf("0: %+v", got[0])
	}
	if got[1].Doc != 2 {
		t.Errorf("1: %+v", got[1])
	}
	if got[2].Doc != -1 {
		t.Errorf("2: %+v", got[2])
	}
}
