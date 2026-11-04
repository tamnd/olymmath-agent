package model

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/lane"
)

// fakeLane is a lane that answers with a function and keeps the prompts.
type fakeLane struct {
	name string
	ask  func(lane.Question) (lane.Answer, error)

	mu      sync.Mutex
	prompts []string
}

func (f *fakeLane) Name() string { return f.name }

func (f *fakeLane) Ask(_ context.Context, q lane.Question) (lane.Answer, error) {
	f.mu.Lock()
	f.prompts = append(f.prompts, q.Prompt)
	f.mu.Unlock()
	return f.ask(q)
}

func (f *fakeLane) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.prompts)
}

// fakeCLI gives a CLI client whose lanes are the fakes, by client name. It records the model that each lane was made for.
func fakeCLI(lanes map[string]*fakeLane, made *[]string) *CLI {
	return &CLI{Lane: func(client, name string) (lane.Named, error) {
		*made = append(*made, client+"/"+name)
		l, ok := lanes[client]
		if !ok {
			return nil, errors.New("no client " + client)
		}
		return l, nil
	}}
}

func TestParseCLIAndFamily(t *testing.T) {
	if c, n, err := ParseCLI("cli:gemini/gemini-2.5-pro"); err != nil || c != "gemini" || n != "gemini-2.5-pro" {
		t.Fatalf("ParseCLI = %q, %q, %v", c, n, err)
	}
	if c, n, err := ParseCLI("cli:codex"); err != nil || c != "codex" || n != "" {
		t.Fatalf("ParseCLI = %q, %q, %v", c, n, err)
	}
	if _, _, err := ParseCLI("cli:llama"); err == nil {
		t.Fatal("an unknown client is parsed")
	}
	for id, want := range map[string]string{
		Opus: FamilyAnthropic, Haiku: FamilyAnthropic, "cli:claude": FamilyAnthropic,
		"cli:codex": FamilyOpenAI, "gpt-5.5": FamilyOpenAI, "o3": FamilyOpenAI,
		"cli:gemini": FamilyGoogle, "gemini-2.5-pro": FamilyGoogle, "cli:codex/gpt-5.5": FamilyOpenAI,
		"deepseek-r1": "", "": "",
	} {
		if got := Family(id); got != want {
			t.Errorf("Family(%q) = %q, want %q", id, got, want)
		}
	}
	if c, err := Cost("cli:gemini", content.Tokens{Input: 1e6, Output: 1e6}, false); err != nil || c != 0 {
		t.Fatalf("Cost of a cli: model = %v, %v", c, err)
	}
	if c, err := Estimate(Request{Model: "cli:codex", System: "s", MaxTokens: 1000}, true); err != nil || c != 0 {
		t.Fatalf("Estimate of a cli: model = %v, %v", c, err)
	}
}

func TestPrompt(t *testing.T) {
	p := Prompt(Request{System: "Grade it.", Messages: []Message{{"user", "Solution."}, {"assistant", "SCORE: 3"}, {"user", "Grade once more."}}})
	want := "Grade it.\n\nSolution.\n\nYour answer before this message:\n\nSCORE: 3\n\n(The end of your answer before this message.)\n\nGrade once more."
	if p != want {
		t.Fatalf("Prompt = %q", p)
	}
}

func TestCLIComplete(t *testing.T) {
	gem := &fakeLane{name: "gemini", ask: func(q lane.Question) (lane.Answer, error) {
		return lane.Answer{Text: "Notes.\nSCORE: 6", Lane: "gemini", Model: "gemini-2.5-pro"}, nil
	}}
	var made []string
	c := fakeCLI(map[string]*fakeLane{"gemini": gem}, &made)
	c.Parallel = 2
	reqs := []Request{
		{ID: "a", Model: "cli:gemini", System: "Grade it.", Messages: []Message{{"user", "Proof A."}}},
		{ID: "b", Model: "cli:gemini", System: "Grade it.", Messages: []Message{{"user", "Proof B."}}},
		{ID: "c", Model: "cli:codex", Messages: []Message{{"user", "x"}}},
		{ID: "d", Model: "cli:llama", Messages: []Message{{"user", "x"}}},
	}
	got, err := c.Complete(context.Background(), reqs)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range []int{0, 1} {
		r := got[i]
		if r.Err != nil || r.ID != reqs[i].ID || r.Text != "Notes.\nSCORE: 6" || r.Model != "gemini-2.5-pro" || r.StopReason != "end_turn" || r.Usage.Input == 0 || r.Usage.Output == 0 {
			t.Fatalf("response %d: %+v", i, r)
		}
	}
	if got[2].Err == nil || got[3].Err == nil {
		t.Fatalf("a request for a missing client succeeded: %+v %+v", got[2], got[3])
	}
	if gem.calls() != 2 || !strings.Contains(gem.prompts[0], "Grade it.\n\nProof") {
		t.Fatalf("the lane got %q", gem.prompts)
	}
	// The lane of a model id is made once.
	if n := strings.Count(strings.Join(made, " "), "gemini/"); n != 1 {
		t.Fatalf("the gemini lane was made %d times: %v", n, made)
	}
}

func TestCLIQuota(t *testing.T) {
	codex := &fakeLane{name: "codex", ask: func(lane.Question) (lane.Answer, error) {
		return lane.Answer{Lane: "codex"}, &lane.QuotaError{Lane: "codex", Message: "you've hit your usage limit"}
	}}
	var made []string
	c := fakeCLI(map[string]*fakeLane{"codex": codex}, &made)
	c.StateDir = t.TempDir()
	reqs := []Request{{ID: "a", Model: "cli:codex", Messages: []Message{{"user", "x"}}}, {ID: "b", Model: "cli:codex", Messages: []Message{{"user", "y"}}}}
	got, err := c.Complete(context.Background(), reqs)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range got {
		if !errors.Is(r.Err, lane.ErrExhausted) || r.StopReason != "quota" {
			t.Fatalf("response %+v", r)
		}
	}
	// The chain rests the lane after the first spent allowance, so the second request makes no call.
	if codex.calls() != 1 {
		t.Fatalf("the lane got %d calls, want 1", codex.calls())
	}
	// A new client reads the rest from the state file and makes no call.
	c2 := fakeCLI(map[string]*fakeLane{"codex": codex}, &made)
	c2.StateDir = c.StateDir
	if got, _ := c2.Complete(context.Background(), reqs[:1]); !errors.Is(got[0].Err, lane.ErrExhausted) || codex.calls() != 1 {
		t.Fatalf("after a restart: %+v, %d calls", got[0], codex.calls())
	}
}

func TestSplit(t *testing.T) {
	api := &Fake{Answer: func(r Request) string { return "api " + r.Model }}
	gem := &fakeLane{name: "gemini", ask: func(lane.Question) (lane.Answer, error) {
		return lane.Answer{Text: "cli", Model: "gemini-2.5-pro"}, nil
	}}
	var made []string
	s := &Split{API: api, CLI: fakeCLI(map[string]*fakeLane{"gemini": gem}, &made)}
	reqs := []Request{{ID: "1", Model: Sonnet}, {ID: "2", Model: "cli:gemini"}, {ID: "3", Model: Haiku}}
	got, err := s.Complete(context.Background(), reqs)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Text != "api "+Sonnet || got[1].Text != "cli" || got[1].Model != "gemini-2.5-pro" || got[2].Text != "api "+Haiku || api.Calls() != 2 || gem.calls() != 1 {
		t.Fatalf("responses %+v", got)
	}
	s.API = nil
	got, _ = s.Complete(context.Background(), reqs)
	if !errors.Is(got[0].Err, ErrNoKey) || got[1].Err != nil {
		t.Fatalf("with no API client: %+v", got)
	}
}
