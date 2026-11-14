package lane

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestQuota(t *testing.T) {
	said := "ERROR: You’ve hit your usage limit. Upgrade to Pro or try again at Nov 9th, 2026 4:38 PM."
	e := Quota("codex", said)
	if e == nil {
		t.Fatal("no quota error")
	}
	want := time.Date(2026, 11, 9, 16, 38, 0, 0, time.Local)
	if !e.Until.Equal(want) {
		t.Errorf("until %v, want %v", e.Until, want)
	}
	if !errors.Is(e, ErrExhausted) {
		t.Error("the error does not wrap ErrExhausted")
	}
	if d := e.Rest(want.Add(-time.Hour)); d != time.Hour {
		t.Errorf("rest %v", d)
	}
	if Quota("codex", "the context length limit is 200000 tokens") != nil {
		t.Error("a context length error is not a spent allowance")
	}
	if e := Quota("claude", "Claude usage limit reached"); e == nil || e.Rest(time.Now()) != DefaultRest {
		t.Errorf("claude: %v", e)
	}
}

// fake gives a CLI lane whose program gives out and errOut and records the command line.
func fake(t *testing.T, kind, out, errOut string, err error) (*CLI, *[]string) {
	t.Helper()
	// The tests do not read the configuration of codex on this machine.
	t.Setenv("CODEX_HOME", t.TempDir())
	l, e := NewCLI(kind, "/bin/"+kind)
	if e != nil {
		t.Fatal(e)
	}
	var argv []string
	l.run = func(_ context.Context, dir string, a []string) ([]byte, []byte, error) {
		if len(a) == 2 && a[1] == "--version" {
			return []byte("9.8.7 (" + kind + ")\n"), nil, nil
		}
		argv = a
		if kind == "codex" && out != "" {
			if e := os.WriteFile(filepath.Join(dir, "answer.txt"), []byte(out), 0o600); e != nil {
				t.Fatal(e)
			}
			return []byte("banner"), []byte(errOut), err
		}
		return []byte(out), []byte(errOut), err
	}
	return l, &argv
}

func TestCLIKinds(t *testing.T) {
	q := Question{Prompt: "-read the page", Images: []string{"/tmp/x/p1.png"}}
	cases := []struct {
		kind, out, want, model string
		flags                  []string
	}{
		{"codex", "Problem 1. $x$", "Problem 1. $x$", "", []string{"exec", "-i", "/tmp/x/p1.png", "--"}},
		{"gemini", `{"response": "Problem 1.", "stats": {"models": {"gemini-2.5-pro": {}}}}`, "Problem 1.", "gemini-2.5-pro", []string{"--include-directories", "/tmp/x", "-p"}},
		{"claude", `{"result": "Problem 1.", "is_error": false, "modelUsage": {"claude-x": {}}}`, "Problem 1.", "claude-x", []string{"-p", "--add-dir", "/tmp/x", "--allowedTools"}},
	}
	for _, c := range cases {
		l, argv := fake(t, c.kind, c.out, "", nil)
		a, err := l.Ask(context.Background(), q)
		if err != nil {
			t.Fatalf("%s: %v", c.kind, err)
		}
		if a.Text != c.want || a.Model != c.model || a.Lane != c.kind || a.Version != "9.8.7" {
			t.Errorf("%s: answer %+v", c.kind, a)
		}
		for _, f := range c.flags {
			if !slices.Contains(*argv, f) {
				t.Errorf("%s: no %s in %q", c.kind, f, *argv)
			}
		}
		last := (*argv)[len(*argv)-1]
		if !strings.Contains(last, "-read the page") {
			t.Errorf("%s: the prompt is not the last argument: %q", c.kind, last)
		}
		if c.kind != "gemini" && (*argv)[len(*argv)-2] != "--" {
			t.Errorf("%s: the prompt does not come after --", c.kind)
		}
	}
}

func TestCLIUsage(t *testing.T) {
	events := `{"type":"thread.started","thread_id":"x"}
{"type":"item.completed","item":{"id":"item_0","type":"agent_message","text":"SCORE: 3"}}
{"type":"turn.completed","usage":{"input_tokens":15025,"cached_input_tokens":11776,"cache_write_input_tokens":0,"output_tokens":5,"reasoning_output_tokens":0}}
`
	l, argv := fake(t, "codex", "SCORE: 3", "", nil)
	l.run = func(_ context.Context, dir string, a []string) ([]byte, []byte, error) {
		if len(a) == 2 {
			return []byte("codex-cli 0.167.0"), nil, nil
		}
		*argv = a
		return []byte(events), []byte("Reading additional input from stdin..."), os.WriteFile(filepath.Join(dir, "answer.txt"), []byte("SCORE: 3"), 0o600)
	}
	if err := os.WriteFile(filepath.Join(os.Getenv("CODEX_HOME"), "config.toml"), []byte("model = \"gpt-x\"\n[projects.\"/a\"]\nmodel = \"other\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	a, err := l.Ask(context.Background(), Question{Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if want := (Usage{Input: 3249, CacheRead: 11776, Output: 5}); a.Usage != want || a.Model != "gpt-x" || a.Text != "SCORE: 3" {
		t.Errorf("codex: %+v", a)
	}
	if !slices.Contains(*argv, "--json") {
		t.Errorf("codex: no --json in %q", *argv)
	}

	l, _ = fake(t, "claude", `{"result": "OK", "is_error": false, "modelUsage": {"claude-x": {}}, "usage": {"input_tokens": 2, "cache_creation_input_tokens": 7, "cache_read_input_tokens": 15677, "output_tokens": 4}}`, "", nil)
	a, err = l.Ask(context.Background(), Question{Prompt: "p"})
	if err != nil {
		t.Fatal(err)
	}
	if want := (Usage{Input: 2, Output: 4, CacheRead: 15677, CacheWrite: 7}); a.Usage != want || !a.Usage.Known() {
		t.Errorf("claude: %+v", a.Usage)
	}
	if (Usage{}).Known() {
		t.Error("a zero usage is known")
	}
}

func TestCLIFailures(t *testing.T) {
	l, _ := fake(t, "codex", "", "ERROR: You've hit your usage limit. try again at Nov 9th, 2026 4:38 PM", errors.New("exit 1"))
	if _, err := l.Ask(context.Background(), Question{Prompt: "p"}); !errors.Is(err, ErrExhausted) {
		t.Errorf("codex quota: %v", err)
	}
	l, _ = fake(t, "gemini", "", "Please set an Auth method in your settings.json", errors.New("exit 41"))
	if _, err := l.Ask(context.Background(), Question{Prompt: "p"}); !errors.Is(err, ErrUnavailable) {
		t.Errorf("gemini login: %v", err)
	}
	l, _ = fake(t, "claude", `{"result": "  ", "is_error": false}`, "", nil)
	if _, err := l.Ask(context.Background(), Question{Prompt: "p"}); !errors.Is(err, ErrEmptyAnswer) {
		t.Errorf("claude empty: %v", err)
	}
	l, _ = fake(t, "claude", `{"result": "boom", "is_error": true}`, "", nil)
	if _, err := l.Ask(context.Background(), Question{Prompt: "p"}); err == nil {
		t.Error("claude error: no error")
	}
	if _, err := NewCLI("other", "/bin/other"); err == nil {
		t.Error("an unknown client is not an error")
	}
}

type named struct {
	name  string
	calls int
	err   error
}

func (n *named) Name() string { return n.name }
func (n *named) Ask(context.Context, Question) (Answer, error) {
	n.calls++
	if n.err != nil {
		return Answer{Lane: n.name}, n.err
	}
	return Answer{Text: "ok " + n.name, Lane: n.name}, nil
}

func TestChain(t *testing.T) {
	state := filepath.Join(t.TempDir(), "lanes.json")
	now := time.Date(2026, 11, 4, 10, 0, 0, 0, time.Local)
	codex := &named{name: "codex", err: &QuotaError{Lane: "codex", Until: now.Add(48 * time.Hour)}}
	gemini := &named{name: "gemini", err: ErrUnavailable}
	claude := &named{name: "claude"}
	c := &Chain{Lanes: []Named{codex, gemini, claude}, State: state, Now: func() time.Time { return now }}
	for range 3 {
		a, err := c.Ask(context.Background(), Question{})
		if err != nil || a.Lane != "claude" {
			t.Fatalf("answer %+v, %v", a, err)
		}
	}
	if codex.calls != 1 || gemini.calls != 1 || claude.calls != 3 {
		t.Errorf("calls %d %d %d", codex.calls, gemini.calls, claude.calls)
	}
	if got := c.Ready(); !slices.Equal(got, []string{"claude"}) {
		t.Errorf("ready %v", got)
	}
	// A new chain reads the rest of codex from the state file and does not ask it.
	codex2 := &named{name: "codex"}
	c2 := &Chain{Lanes: []Named{codex2, claude}, State: state, Now: func() time.Time { return now }}
	if _, err := c2.Ask(context.Background(), Question{}); err != nil || codex2.calls != 0 {
		t.Errorf("codex was asked again: %d, %v", codex2.calls, err)
	}
	// When each lane is spent, the chain stops with ErrExhausted.
	claude.err = &QuotaError{Lane: "claude"}
	if _, err := c2.Ask(context.Background(), Question{}); !errors.Is(err, ErrExhausted) {
		t.Errorf("all spent: %v", err)
	}
	// Any other error goes to the caller, and the lane does not rest.
	other := &named{name: "codex", err: errors.New("boom")}
	c3 := &Chain{Lanes: []Named{other}}
	if _, err := c3.Ask(context.Background(), Question{}); err == nil || errors.Is(err, ErrExhausted) {
		t.Errorf("other error: %v", err)
	}
}

func TestSelect(t *testing.T) {
	all := []Named{&named{name: "codex"}, &named{name: "claude"}}
	got, err := Select(all, "claude,codex")
	if err != nil || got[0].Name() != "claude" || got[1].Name() != "codex" {
		t.Errorf("select %v %v", got, err)
	}
	if _, err := Select(all, "gemini"); err == nil {
		t.Error("a missing client is not an error")
	}
	if got, _ := Select(all, ""); len(got) != 2 {
		t.Error("an empty list does not give all")
	}
}

func TestCodexModel(t *testing.T) {
	l, _ := fake(t, "codex", "SCORE: 7", "OpenAI Codex v0.166.0\n--------\nworkdir: /tmp/x\nmodel: gpt-5.5\nprovider: openai\n", nil)
	a, err := l.Ask(context.Background(), Question{Prompt: "grade"})
	if err != nil || a.Model != "gpt-5.5" || a.Text != "SCORE: 7" {
		t.Fatalf("answer %+v, %v", a, err)
	}
}
