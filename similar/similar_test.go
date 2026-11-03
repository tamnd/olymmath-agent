package similar

import (
	"math/rand/v2"
	"reflect"
	"sort"
	"strings"
	"testing"
)

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

// bestMap is best as it was with a new map for each text. TestBestSameAsMap compares the two.
func bestMap(i int, words []string, vec map[string]float64, index map[string][]posting, n int, least float64) []Match {
	score := map[int]float64{}
	for _, w := range words {
		x := vec[w]
		for _, p := range index[w] {
			if p.doc != i {
				score[p.doc] += x * p.w
			}
		}
	}
	var ms []Match
	for d, s := range score {
		if s >= least {
			ms = append(ms, Match{d, s})
		}
	}
	sort.Slice(ms, func(a, b int) bool {
		if ms[a].Score != ms[b].Score {
			return ms[a].Score > ms[b].Score
		}
		return ms[a].Doc < ms[b].Doc
	})
	if len(ms) > n {
		ms = ms[:n]
	}
	return ms
}

// TestBestSameAsMap wants the same matches from the shared buffer as from a new map for each text. The texts come from a small set of words, so they share many words and have equal scores.
func TestBestSameAsMap(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	var vocab []string
	for i := range 300 {
		vocab = append(vocab, "w"+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	texts := make([]string, 600)
	for i := range texts {
		ws := make([]string, 3+r.IntN(20))
		for j := range ws {
			ws[j] = vocab[r.IntN(len(vocab))]
		}
		// Some texts are the same, so that some scores are equal.
		if i%7 == 0 && i > 0 {
			texts[i] = texts[i-1]
			continue
		}
		texts[i] = strings.Join(ws, " ")
	}
	sorted, vecs, index := weigh(texts)
	for _, least := range []float64{0, 0.1, 0.3} {
		for _, n := range []int{1, 5, 1000} {
			// One buffer for all the texts, as in a worker of Find.
			s := newScores(len(texts))
			for i := range texts {
				got := best(i, sorted[i], vecs[i], index, n, least, s)
				want := bestMap(i, sorted[i], vecs[i], index, n, least)
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("least %v, n %d, text %d: got %v, want %v", least, n, i, got, want)
				}
			}
		}
	}
	all := Find(texts, 5, 0.1)
	for i := range texts {
		if want := bestMap(i, sorted[i], vecs[i], index, 5, 0.1); !reflect.DeepEqual(all[i], want) {
			t.Fatalf("Find, text %d: got %v, want %v", i, all[i], want)
		}
	}
}
