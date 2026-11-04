package grade

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/model"
)

func TestJurors(t *testing.T) {
	if got := Jurors(DefaultJurors, model.Opus); !slices.Equal(got, []string{model.Sonnet, model.Haiku}) {
		t.Fatalf("Jurors for Opus = %v", got)
	}
	if got := Jurors(DefaultJurors, model.Sonnet); !slices.Equal(got, []string{model.Haiku, model.Opus}) {
		t.Fatalf("Jurors for Sonnet = %v", got)
	}
}

func TestScore(t *testing.T) {
	if s, ok := Score("SCORE: 3\nmore\nSCORE: 6"); !ok || s != 6 {
		t.Fatalf("Score = %d, %v", s, ok)
	}
	if _, ok := Score("SCORE: 9"); ok {
		t.Fatal("a score of 9 is read")
	}
}

// jury answers with fixed scores by model, and with revised scores in the second round.
func jury(first, second map[string]int) *model.Fake {
	return &model.Fake{Answer: func(r model.Request) string {
		s := first[r.Model]
		if len(r.Messages) > 1 {
			s = second[r.Model]
		}
		return "Notes.\nSCORE: " + string(rune('0'+s))
	}}
}

func task() Task {
	return Task{ID: "S", Statement: "Prove it.", Solution: "Proof.", Generator: model.Opus, Ref: "imo/2024/p1@r2", Jurors: Jurors(DefaultJurors, model.Opus)}
}

func TestRunAgree(t *testing.T) {
	f := jury(map[string]int{model.Sonnet: 7, model.Haiku: 6}, nil)
	got, err := Run(context.Background(), f, []Task{task()}, Options{})
	if err != nil {
		t.Fatal(err)
	}
	r := got[0]
	if r.Err != nil || r.Score != 6 || r.Queue || f.Calls() != 2 {
		t.Fatalf("score %d, queue %v, %d calls, err %v", r.Score, r.Queue, f.Calls(), r.Err)
	}
	g := r.Grade("2026-11-04", "grade-20261104-0001")
	if g.Bucket != "almost" || g.Rule != Rule || len(g.Graders) != 2 || g.Graders[0].By != "model:"+model.Sonnet || len(g.Graders[0].NotesHash) != 64 {
		t.Fatalf("the grade is %+v", g)
	}
}

func TestRunRevise(t *testing.T) {
	f := jury(map[string]int{model.Sonnet: 7, model.Haiku: 2}, map[string]int{model.Sonnet: 5, model.Haiku: 4})
	got, _ := Run(context.Background(), f, []Task{task()}, Options{})
	r := got[0]
	if r.Score != 4 || r.Queue || f.Calls() != 4 || !r.Verdicts[0].Revised {
		t.Fatalf("score %d, queue %v, %d calls", r.Score, r.Queue, f.Calls())
	}
	if !strings.Contains(f.Requests[2].Messages[2].Text, "Grader 2 gave 2") {
		t.Fatal("the revision does not show the notes of the other grader")
	}
}

func TestRunQueue(t *testing.T) {
	f := jury(map[string]int{model.Sonnet: 7, model.Haiku: 1}, map[string]int{model.Sonnet: 7, model.Haiku: 3})
	got, _ := Run(context.Background(), f, []Task{task()}, Options{})
	if r := got[0]; r.Score != 3 || !r.Queue {
		t.Fatalf("score %d, queue %v", r.Score, r.Queue)
	}
}

func TestRunSelf(t *testing.T) {
	tk := task()
	tk.Jurors = []string{model.Sonnet, model.Opus}
	f := jury(nil, nil)
	got, _ := Run(context.Background(), f, []Task{tk}, Options{})
	if got[0].Err == nil || f.Calls() != 0 {
		t.Fatal("a model graded its own solution")
	}
}

func TestEstimate(t *testing.T) {
	c, n, err := Estimate([]Task{task()}, Options{}, true)
	if err != nil || n != 2 || c <= 0 {
		t.Fatalf("Estimate = %v, %d, %v", c, n, err)
	}
}
