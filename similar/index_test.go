package similar

import (
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"
)

// TestAboveSameAsFind wants the same matches from an Index of least 2 as from Find.
func TestAboveSameAsFind(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	var vocab []string
	for i := range 200 {
		vocab = append(vocab, "w"+string(rune('a'+i%26))+string(rune('a'+i/26)))
	}
	texts := make([]string, 400)
	for i := range texts {
		ws := make([]string, 3+r.IntN(15))
		for j := range ws {
			ws[j] = vocab[r.IntN(len(vocab))]
		}
		texts[i] = strings.Join(ws, " ")
	}
	want := Find(texts, 5, 0.1)
	s := NewIndex(texts, 2).Searcher()
	for i := range texts {
		if got := s.Above(texts[i], i, 5, 0.1); !reflect.DeepEqual(got, want[i]) {
			t.Fatalf("text %d: got %v, want %v", i, got, want[i])
		}
	}
}

func TestSearcherNearest(t *testing.T) {
	docs := []string{
		"Find all functions f from the reals to the reals with f(x+y)=f(x)+f(y).",
		"Let ABC be a triangle with incircle omega touching BC at D. Prove that BD equals the semiperimeter minus AC.",
		"Let ABC be a triangle with incircle omega touching BC at D. Prove that the circumcircle passes through D.",
	}
	for i := range 20 {
		docs = append(docs, "Some other statement about chess boards and coins number "+string(rune('a'+i)))
	}
	x := NewIndex(docs, 1)
	if x.Len() != len(docs) {
		t.Errorf("Len %d", x.Len())
	}
	s := x.Searcher()
	q := "In triangle ABC the incircle omega touches BC at D. Show that the circumcircle passes through D."
	best, under := s.Nearest(q, 2)
	if best.Doc != 2 || under.Doc != 1 || under.Score >= best.Score {
		t.Errorf("best %+v, under %+v", best, under)
	}
	if best, under := s.Nearest("zzz qqq", 2); best.Doc != -1 || under.Doc != -1 {
		t.Errorf("no word: %+v %+v", best, under)
	}
	// The searcher leaves its buffer empty, so a second query gives the same result.
	if again, _ := s.Nearest(q, 2); again != best {
		t.Errorf("again %+v, want %+v", again, best)
	}
}
