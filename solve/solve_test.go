package solve

import (
	"context"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/model"
)

func corpus(t *testing.T) *content.Corpus {
	t.Helper()
	c, f := content.Load("../content/testdata/valid")
	if f.Errors() > 0 {
		t.Fatalf("the fixture has errors: %v", f)
	}
	return c
}

func TestSelect(t *testing.T) {
	c := corpus(t)
	items := Select(c, Filter{}, Options{})
	if len(items) == 0 {
		t.Fatal("no problem selected")
	}
	for _, it := range items {
		if len(it.Problem.Solutions) > 0 {
			t.Errorf("%s has a solution but is selected", it.Problem.Locator())
		}
		if it.Model == "" || it.K == 0 {
			t.Errorf("%s has no model or no k", it.Problem.Locator())
		}
	}
	named := Select(c, Filter{Tags: []string{"imo/2024/p1"}}, Options{Model: model.Haiku, K: 3})
	if len(named) != 1 || named[0].Model != model.Haiku || named[0].K != 3 {
		t.Fatalf("Select of a named problem gives %+v", named)
	}
	if got := Select(c, Filter{Limit: 1}, Options{}); len(got) != 1 {
		t.Fatalf("Limit 1 gives %d", len(got))
	}
}

func TestPlan(t *testing.T) {
	items := Select(corpus(t), Filter{}, Options{})
	e, err := Plan(items, Options{}, false)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := Plan(items, Options{}, true)
	if e.Cost <= 0 || e.Calls == 0 || b.Cost >= e.Cost || len(e.Bands) == 0 {
		t.Fatalf("the plan is %+v, with batch %+v", e, b)
	}
}

func TestBoxedVerdict(t *testing.T) {
	if got := Boxed(`so $\boxed{1}$ and finally $\boxed{\frac{1}{2}}$.`); got != `\frac{1}{2}` {
		t.Fatalf("Boxed = %q", got)
	}
	if Boxed("no answer") != "" || Boxed(`\boxed{open`) != "" {
		t.Fatal("Boxed finds an answer that is not there")
	}
	if !Verdict("issues: none\nVERDICT: PASS") || Verdict("VERDICT: PASS\nlater VERDICT: FAIL") || Verdict("fine") {
		t.Fatal("Verdict reads the wrong line")
	}
}

// script is a fake model: the generator writes a solution with an answer, and the verifier fails the first draft of attempt 1 once.
func script(answer string) func(model.Request) string {
	return func(r model.Request) string {
		switch r.Kind {
		case "generate":
			return "We prove it. " + strings.Repeat("Each step holds. ", 30) + `So the answer is $\boxed{` + answer + `}$.`
		case "revise":
			return "We prove it again, fixed. So the answer is $\\boxed{" + answer + "}$."
		}
		text := r.Messages[0].Text
		if strings.Contains(text, "We prove it. ") && strings.Contains(r.ID, "/1") && strings.HasSuffix(text, "check 1.") {
			return "Critical error: \"Each step holds.\" is not proved.\nVERDICT: FAIL"
		}
		return "No issues.\nVERDICT: PASS"
	}
}

func TestRun(t *testing.T) {
	c := corpus(t)
	items := Select(c, Filter{Tags: []string{"imo/2024/p2"}}, Options{K: 2, Model: model.Sonnet})
	f := &model.Fake{Answer: script("(a, b) = (1, 1)")}
	got, err := Run(context.Background(), f, items, Options{})
	if err != nil {
		t.Fatal(err)
	}
	r := got[0]
	if r.Outcome != Accepted || r.Stage != "S5" || r.AnswerCheck != "match" || r.Answer != "(a, b) = (1, 1)" {
		t.Fatalf("the result is %+v", r)
	}
	// Attempt 2 passes three checks first, while attempt 1 is revised; the problem ends there.
	if r.Attempt != 2 || r.Cost <= 0 || len(r.Calls) != f.Calls() {
		t.Fatalf("attempt %d, cost %v, %d calls of %d", r.Attempt, r.Cost, len(r.Calls), f.Calls())
	}
	if _, h := r.Transcript(); len(h) != 64 {
		t.Fatalf("the transcript hash is %q", h)
	}
	for _, req := range f.Requests {
		if strings.Contains(req.System, "Write the solution here") {
			t.Fatal("the solver saw a human solution")
		}
	}
}

func TestRunMismatch(t *testing.T) {
	items := Select(corpus(t), Filter{Tags: []string{"imo/2024/p2"}}, Options{K: 1, Model: model.Sonnet})
	got, err := Run(context.Background(), &model.Fake{Answer: script("(2, 2)")}, items, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Outcome != Mismatch {
		t.Fatalf("the outcome is %s, want mismatch", got[0].Outcome)
	}
}

func TestRunUnsolved(t *testing.T) {
	items := Select(corpus(t), Filter{Tags: []string{"imo/2024/p2"}}, Options{K: 1, Model: model.Haiku})
	f := &model.Fake{Answer: func(r model.Request) string {
		if r.Kind == "verify" {
			return "VERDICT: FAIL"
		}
		return "a try"
	}}
	got, _ := Run(context.Background(), f, items, Options{Rounds: 2})
	// One draft, two revisions, and a check after each.
	if got[0].Outcome != Unsolved || f.Calls() != 6 {
		t.Fatalf("the outcome is %s after %d calls", got[0].Outcome, f.Calls())
	}
}

func TestRunBudget(t *testing.T) {
	items := Select(corpus(t), Filter{Tags: []string{"imo/2024/p2"}}, Options{K: 2, Model: model.Opus})
	f := &model.Fake{Answer: script("(a, b) = (1, 1)")}
	b := &model.Budget{Client: f, Limit: 0.01}
	got, err := Run(context.Background(), b, items, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Outcome != OverBudget || f.Calls() != 0 || b.Spent() > 0.01 {
		t.Fatalf("the outcome is %s after %d calls and $%v", got[0].Outcome, f.Calls(), b.Spent())
	}
}

func TestResume(t *testing.T) {
	items := Select(corpus(t), Filter{Tags: []string{"imo/2024/p2"}}, Options{K: 2, Model: model.Sonnet})
	dir := t.TempDir()
	f := &model.Fake{Answer: script("(a, b) = (1, 1)")}
	// The first run stops on a small budget; the second, with more, goes on from the cache.
	first, _ := Run(context.Background(), &model.Cache{Client: &model.Budget{Client: f, Limit: 0.5}, Dir: dir}, items, Options{})
	n := f.Calls()
	second, err := Run(context.Background(), &model.Cache{Client: &model.Budget{Client: f, Limit: 10}, Dir: dir}, items, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if first[0].Outcome != OverBudget || second[0].Outcome != Accepted || n == 0 {
		t.Fatalf("the outcomes are %s and %s after %d calls", first[0].Outcome, second[0].Outcome, n)
	}
	cached := 0
	for _, c := range second[0].Calls {
		if c.Cached {
			cached++
		}
	}
	if cached != n {
		t.Fatalf("the second run read %d calls from the cache, want %d", cached, n)
	}
}
