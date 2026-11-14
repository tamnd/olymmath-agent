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
	// A cli: juror that names the generator does not grade its solution.
	if got := Jurors([]string{"cli:codex", "cli:claude/claude-sonnet-5-5", "cli:claude"}, model.Sonnet); !slices.Equal(got, []string{"cli:codex", "cli:claude"}) {
		t.Fatalf("Jurors of cli: models for Sonnet = %v", got)
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

func TestOtherFamily(t *testing.T) {
	if OtherFamily(Jurors(DefaultJurors, model.Opus), model.Opus) {
		t.Fatal("two Claude jurors are of another family than Opus")
	}
	if !OtherFamily([]string{model.Sonnet, "cli:gemini"}, model.Opus) || !OtherFamily([]string{model.Sonnet, model.Haiku}, "") {
		t.Fatal("no other family")
	}
}

// cliJury answers the Claude jurors with the fake and the cli: jurors with a fake model that tells its real name.
type cliJury struct{ api *model.Fake }

func (c cliJury) Complete(ctx context.Context, reqs []model.Request) ([]model.Response, error) {
	out, err := c.api.Complete(ctx, reqs)
	for i, r := range reqs {
		if model.IsCLI(r.Model) {
			out[i].Model = "gemini-2.5-pro"
		}
	}
	return out, err
}

func TestRunOtherFamily(t *testing.T) {
	f := jury(map[string]int{model.Sonnet: 7, "cli:gemini": 7}, nil)
	tk := task()
	tk.Jurors = Jurors([]string{model.Sonnet, "cli:gemini"}, model.Opus)
	got, err := Run(context.Background(), cliJury{f}, []Task{tk}, Options{})
	if err != nil || got[0].Err != nil {
		t.Fatal(err, got[0].Err)
	}
	r := got[0]
	if r.Calls[1].Cost != 0 || r.Calls[0].Cost == 0 || r.Calls[1].Model != "cli:gemini" || r.Calls[1].Answered != "gemini-2.5-pro" || r.Calls[0].Answered != "" {
		t.Fatalf("calls %+v", r.Calls)
	}
	g := r.Grade("2026-11-05", "grade-20261105-0001")
	if g.Graders[0].By != "model:"+model.Sonnet || g.Graders[0].Family != model.FamilyAnthropic || g.Graders[1].By != "model:gemini-2.5-pro" || g.Graders[1].Family != model.FamilyGoogle {
		t.Fatalf("the graders are %+v", g.Graders)
	}
	if est, calls, err := Estimate([]Task{tk}, Options{}, true); err != nil || calls != 2 || est <= 0 {
		t.Fatalf("Estimate = %v, %d, %v", est, calls, err)
	}
}
