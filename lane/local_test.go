package lane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/vision"
)

// fakeServer is a vision.Transport. page gives the reply to a page, and health the reply to a health check.
type fakeServer struct {
	name   string
	page   func() (vision.Reply, error)
	health func() (vision.Reply, error)
	pages  int
	checks int
}

func (f *fakeServer) Name() string { return f.name }

func (f *fakeServer) Do(_ context.Context, _, _ string, body []byte) (vision.Reply, error) {
	var req struct {
		MaxTokens int `json:"max_tokens"`
	}
	_ = json.Unmarshal(body, &req)
	if req.MaxTokens == 1 {
		f.checks++
		if f.health != nil {
			return f.health()
		}
		return ok("H", "length")
	}
	f.pages++
	return f.page()
}

func ok(text, finish string) (vision.Reply, error) {
	return vision.Reply{Status: 200, Body: []byte(fmt.Sprintf(`{"model":"m","choices":[{"finish_reason":%q,"message":{"content":%q}}]}`, finish, text))}, nil
}

func status(code int, msg string) func() (vision.Reply, error) {
	return func() (vision.Reply, error) {
		return vision.Reply{Status: code, Body: []byte(fmt.Sprintf(`{"error":{"message":%q}}`, msg))}, nil
	}
}

func unreachable() (vision.Reply, error) {
	return vision.Reply{}, fmt.Errorf("gpu: %w: Operation timed out", vision.ErrUnreachable)
}

func reads(text string) func() (vision.Reply, error) {
	return func() (vision.Reply, error) { return ok(text, "stop") }
}

// image writes a page image and gives the question about it.
func image(t *testing.T) Question {
	t.Helper()
	p := filepath.Join(t.TempDir(), "page.png")
	if err := os.WriteFile(p, []byte("png"), 0o600); err != nil {
		t.Fatal(err)
	}
	return Question{Prompt: "read", Images: []string{p}}
}

// clock is a time that a test moves.
type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func local(c *clock, targets ...*fakeServer) *Local {
	l := &Local{Now: c.now}
	for _, f := range targets {
		model, _, _ := strings.Cut(f.name, "@")
		l.Targets = append(l.Targets, &Target{Model: model, Transport: f})
	}
	return l
}

func TestParseTargets(t *testing.T) {
	ts, err := ParseTargets(DefaultLocal, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 || ts[0].Name() != "GLM-OCR-8bit@127.0.0.1:8079" || ts[1].Name() != "reader-a@gpc" {
		t.Fatalf("targets %v %v", ts[0].Name(), ts[1].Name())
	}
	if s, ok := ts[1].Transport.(*vision.SSH); !ok || s.URL != "http://127.0.0.1:8801/v1" {
		t.Errorf("ssh target %+v", ts[1].Transport)
	}
	for _, bad := range []string{"", "http://x/v1", "m=ftp://x", "m=ssh://gpc", "m=ssh://-oProxyCommand=x/http://a"} {
		if _, err := ParseTargets(bad, nil); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
}

func TestLocalReadsAPage(t *testing.T) {
	c := &clock{time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)}
	a := &fakeServer{name: "m@a", page: reads("**1.** text")}
	l := local(c, a)
	got, err := l.Ask(t.Context(), image(t))
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "**1.** text" || got.Lane != "local" || got.Model != "m" {
		t.Errorf("%+v", got)
	}
	if _, err := l.Ask(t.Context(), image(t)); err != nil {
		t.Fatal(err)
	}
	if a.checks != 1 || a.pages != 2 {
		t.Errorf("%d health checks and %d pages, want 1 and 2", a.checks, a.pages)
	}
}

func TestLocalRoundRobin(t *testing.T) {
	c := &clock{time.Now()}
	a := &fakeServer{name: "m@a", page: reads("a")}
	b := &fakeServer{name: "m@b", page: reads("b")}
	other := &fakeServer{name: "n@c", page: reads("c")}
	l := local(c, a, b, other)
	var got []string
	for range 4 {
		ans, err := l.Ask(t.Context(), image(t))
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, ans.Text)
	}
	if strings.Join(got, "") != "abab" || other.pages != 0 {
		t.Errorf("answers %v, and %d pages of the next model", got, other.pages)
	}
}

func TestLocalRestsOn429(t *testing.T) {
	c := &clock{time.Now()}
	a := &fakeServer{name: "m@a", page: func() (vision.Reply, error) {
		return vision.Reply{Status: 429, Body: []byte(`{"error":{"message":"busy"}}`), RetryAfter: 30 * time.Second}, nil
	}}
	b := &fakeServer{name: "m@b", page: reads("b")}
	l := local(c, a, b)
	ans, err := l.Ask(t.Context(), image(t))
	if err != nil || ans.Text != "b" {
		t.Fatalf("%+v %v", ans, err)
	}
	// a rests, so b reads the next pages too.
	for range 2 {
		if ans, err := l.Ask(t.Context(), image(t)); err != nil || ans.Text != "b" {
			t.Fatalf("%+v %v", ans, err)
		}
	}
	if a.pages != 1 {
		t.Errorf("a got %d pages while it rests", a.pages)
	}
	if want := c.t.Add(30 * time.Second); !l.Targets[0].until.Equal(want) {
		t.Errorf("a rests until %v, want %v", l.Targets[0].until, want)
	}
	// When b answers 429 too, with no Retry-After, the lane rests until a comes back.
	b.page = status(429, "busy")
	_, err = l.Ask(t.Context(), image(t))
	var re *RestError
	if !errors.As(err, &re) || !errors.Is(err, ErrDown) || !re.Until.Equal(c.t.Add(30*time.Second)) {
		t.Fatalf("error %v", err)
	}
	if !l.Targets[1].until.Equal(c.t.Add(BusyRest)) {
		t.Errorf("b rests until %v", l.Targets[1].until)
	}
	c.t = c.t.Add(31 * time.Second)
	a.page = reads("a")
	if ans, err := l.Ask(t.Context(), image(t)); err != nil || ans.Text != "a" {
		t.Fatalf("%+v %v", ans, err)
	}
	if a.checks != 1 {
		t.Errorf("a was checked %d times. A 429 is not a server that is down, so it needs no new check", a.checks)
	}
}

func TestLocalDownTarget(t *testing.T) {
	c := &clock{time.Now()}
	glm := &fakeServer{name: "glm@mac", page: unreachable}
	gpu := &fakeServer{name: "olm@gpu", page: reads("olm"), health: unreachable}
	l := local(c, glm, gpu)
	_, err := l.Ask(t.Context(), image(t))
	var re *RestError
	if !errors.As(err, &re) || !re.Until.Equal(c.t.Add(DownRest)) {
		t.Fatalf("error %v", err)
	}
	if !strings.Contains(err.Error(), "glm@mac") || !strings.Contains(err.Error(), "olm@gpu: the health check") {
		t.Errorf("the error does not tell which target failed: %v", err)
	}
	if gpu.pages != 0 {
		t.Error("a target that failed its health check got a page")
	}
	// A chain rests the lane until the first target comes back, and tells the caller that no lane can answer.
	ch := &Chain{Lanes: []Named{l}, Now: c.now}
	if _, err := ch.Ask(t.Context(), image(t)); !errors.Is(err, ErrExhausted) {
		t.Fatalf("chain error %v", err)
	}
	if glm.pages != 1 {
		t.Errorf("glm got %d pages, want 1: each target rests, so the lane asks no server", glm.pages)
	}
	if got := ch.Ready(); len(got) != 0 {
		t.Errorf("ready %v while the lane rests", got)
	}
	// After the rest, each target is checked again, and the second model reads the page when the first is still down.
	c.t = c.t.Add(DownRest)
	gpu.health = nil
	ans, err := l.Ask(t.Context(), image(t))
	if err != nil || ans.Text != "olm" || ans.Model != "olm" {
		t.Fatalf("%+v %v", ans, err)
	}
	if glm.checks != 2 || gpu.checks != 2 {
		t.Errorf("checks %d and %d, want 2 each", glm.checks, gpu.checks)
	}
}

func TestLocalRefusalDoesNotRest(t *testing.T) {
	for name, page := range map[string]func() (vision.Reply, error){
		"refusal phrase": reads("I'm sorry, I can't help with that."),
		"token limit":    func() (vision.Reply, error) { return ok("2.5.1.1. 2.5.1.1.", "length") },
		"HTTP 400":       status(400, "input length 18303 exceeds the maximum context length 16384"),
	} {
		t.Run(name, func(t *testing.T) {
			c := &clock{time.Now()}
			a := &fakeServer{name: "m@a", page: page}
			b := &fakeServer{name: "m@b", page: reads("b")}
			l := local(c, a, b)
			_, err := l.Ask(t.Context(), image(t))
			if !errors.Is(err, ErrRefused) || errors.Is(err, ErrDown) {
				t.Fatalf("error %v, want ErrRefused", err)
			}
			if name == "HTTP 400" && !strings.Contains(err.Error(), "18303") {
				t.Errorf("the error has not the message of the server: %v", err)
			}
			if b.pages != 0 || !l.Targets[0].until.IsZero() {
				t.Error("a refused page went to another target, or the target rests")
			}
			// A chain gives the error to the caller and does not rest the lane.
			b.page = page
			ch := &Chain{Lanes: []Named{l}, Now: c.now}
			if _, err := ch.Ask(t.Context(), image(t)); !errors.Is(err, ErrRefused) || errors.Is(err, ErrExhausted) {
				t.Errorf("chain error %v", err)
			}
		})
	}
}

func TestLocalHealthRejected(t *testing.T) {
	c := &clock{time.Now()}
	a := &fakeServer{name: "wrong@a", page: reads("a"), health: status(404, "The model `wrong` does not exist.")}
	l := local(c, a)
	_, err := l.Ask(t.Context(), image(t))
	if !errors.Is(err, ErrUnavailable) || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("error %v, want ErrUnavailable", err)
	}
	if a.pages != 0 {
		t.Error("a target with no model got a page")
	}
	errs := l.Check(t.Context())
	if len(errs) != 1 || errs[0] == nil {
		t.Errorf("check %v", errs)
	}
}
