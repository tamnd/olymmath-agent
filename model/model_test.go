package model

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
)

func TestCost(t *testing.T) {
	c, err := Cost(Opus, content.Tokens{Input: 1e6, Output: 1e6}, false)
	if err != nil || c != 24 {
		t.Fatalf("Cost = %v, %v, want 24", c, err)
	}
	c, _ = Cost(Sonnet, content.Tokens{Input: 1e6, Output: 1e6, CacheRead: 1e6, CacheWrite: 1e6}, true)
	if want := (2 + 10 + 0.2 + 2.5) / 2; math.Abs(c-want) > 1e-9 {
		t.Fatalf("Cost = %v, want %v", c, want)
	}
	if _, err := Cost("gpt", content.Tokens{}, false); err == nil {
		t.Fatal("a model with no price has a cost")
	}
}

func TestNoKey(t *testing.T) {
	t.Setenv(KeyEnv, "")
	if _, err := NewAnthropic(); !errors.Is(err, ErrNoKey) {
		t.Fatalf("NewAnthropic = %v, want ErrNoKey", err)
	}
	a := &Anthropic{}
	if _, err := a.Complete(context.Background(), []Request{{Model: Haiku}}); !errors.Is(err, ErrNoKey) {
		t.Fatalf("Complete = %v, want ErrNoKey", err)
	}
}

func TestKey(t *testing.T) {
	r := Request{ID: "a", Model: Haiku, System: "s", Messages: []Message{{"user", "x"}}}
	q := r
	q.ID = "b"
	if r.Key() != q.Key() {
		t.Fatal("the key depends on the id")
	}
	q.Messages = []Message{{"user", "y"}}
	if r.Key() == q.Key() {
		t.Fatal("two requests have one key")
	}
}

func TestCache(t *testing.T) {
	f := &Fake{Answer: func(r Request) string { return "answer to " + r.Messages[0].Text }}
	c := &Cache{Client: f, Dir: t.TempDir()}
	reqs := []Request{{ID: "1", Model: Haiku, Messages: []Message{{"user", "a"}}}, {ID: "2", Model: Haiku, Messages: []Message{{"user", "b"}}}}
	if _, err := c.Complete(context.Background(), reqs); err != nil {
		t.Fatal(err)
	}
	got, err := c.Complete(context.Background(), append(reqs, Request{ID: "3", Model: Haiku, Messages: []Message{{"user", "c"}}}))
	if err != nil {
		t.Fatal(err)
	}
	if f.Calls() != 3 {
		t.Fatalf("the client got %d calls, want 3", f.Calls())
	}
	if !got[0].Cached || got[0].Text != "answer to a" || got[0].ID != "1" || got[2].Cached {
		t.Fatalf("the responses are %+v", got)
	}
}

// stopAfter answers n calls and then fails, as a run that is stopped.
type stopAfter struct {
	f *Fake
	n int
}

func (s *stopAfter) Complete(ctx context.Context, reqs []Request) ([]Response, error) {
	if s.f.Calls()+len(reqs) > s.n {
		return nil, errors.New("stopped")
	}
	return s.f.Complete(ctx, reqs)
}

func TestCacheEach(t *testing.T) {
	f := &Fake{Answer: func(r Request) string { return "answer to " + r.Messages[0].Text }}
	dir := t.TempDir()
	reqs := []Request{{ID: "1", Model: Haiku, Messages: []Message{{"user", "a"}}}, {ID: "2", Model: Haiku, Messages: []Message{{"user", "b"}}}}
	c := &Cache{Client: &stopAfter{f, 1}, Dir: dir, Each: true}
	if _, err := c.Complete(context.Background(), reqs); err == nil {
		t.Fatal("the stopped run gives no error")
	}
	// The first response was kept before the run stopped.
	if !c.Has(reqs[0]) || c.Has(reqs[1]) {
		t.Fatal("the cache does not hold the first response only")
	}
	c = &Cache{Client: f, Dir: dir, Each: true}
	got, err := c.Complete(context.Background(), reqs)
	if err != nil || !got[0].Cached || got[1].Cached || got[1].Text != "answer to b" || f.Calls() != 2 {
		t.Fatalf("the resumed run gives %+v, %v, %d calls", got, err, f.Calls())
	}
}

func TestBudget(t *testing.T) {
	f := &Fake{Answer: func(Request) string { return strings.Repeat("x", 4000) }}
	// A Haiku call with 1,000,000 output tokens can cost $0.50.
	b := &Budget{Client: f, Limit: 1.2}
	r := Request{Model: Haiku, MaxTokens: 1e6}
	got, err := b.Complete(context.Background(), []Request{r, r, r})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Err != nil || got[1].Err != nil || !errors.Is(got[2].Err, ErrBudget) {
		t.Fatalf("the errors are %v, %v, %v", got[0].Err, got[1].Err, got[2].Err)
	}
	if want := 2 * 1000 * 0.5 / 1e6; math.Abs(b.Spent()-want) > 1e-12 {
		t.Fatalf("Spent = %v, want %v", b.Spent(), want)
	}
	b.Charge(1.2)
	got, _ = b.Complete(context.Background(), []Request{r})
	if !errors.Is(got[0].Err, ErrBudget) || f.Calls() != 2 {
		t.Fatal("a spent budget sends a call")
	}
}

const sse = `event: message_start
data: {"type":"message_start","message":{"usage":{"input_tokens":10,"output_tokens":1,"cache_read_input_tokens":100,"cache_creation_input_tokens":5}}}

event: content_block_start
data: {"type":"content_block_start","index":0,"content_block":{"type":"thinking","thinking":""}}

event: content_block_delta
data: {"type":"content_block_delta","index":0,"delta":{"type":"thinking_delta","thinking":"hmm"}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"The answer is "}}

event: content_block_delta
data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"$\\boxed{2}$."}}

event: message_delta
data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":42}}

event: message_stop
data: {"type":"message_stop"}
`

func TestAnthropicStream(t *testing.T) {
	var body wireRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-api-key") != "k" || r.Header.Get("anthropic-version") != AnthropicVersion || r.URL.Path != "/v1/messages" {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		b, _ := io.ReadAll(r.Body)
		if strings.Contains(string(b), "temperature") {
			http.Error(w, "temperature", http.StatusBadRequest)
			return
		}
		_ = json.Unmarshal(b, &body)
		w.Header().Set("content-type", "text/event-stream")
		_, _ = io.WriteString(w, sse)
	}))
	defer srv.Close()
	a := &Anthropic{Key: "k", BaseURL: srv.URL, Version: AnthropicVersion, HTTP: srv.Client(), Parallel: 2}
	got, err := a.Complete(context.Background(), []Request{{ID: "x", Model: Opus, System: "sys", Messages: []Message{{"user", "solve"}}, MaxTokens: 100, Effort: "high"}})
	if err != nil {
		t.Fatal(err)
	}
	r := got[0]
	if r.Err != nil || r.Text != `The answer is $\boxed{2}$.` || r.StopReason != "end_turn" || r.ID != "x" {
		t.Fatalf("the response is %+v", r)
	}
	if r.Usage != (content.Tokens{Input: 10, Output: 42, CacheRead: 100, CacheWrite: 5}) {
		t.Fatalf("the usage is %+v", r.Usage)
	}
	if !body.Stream || body.System[0].CacheControl == nil || body.OutputConfig.Effort != "high" || body.Messages[0].Content[0].Text != "solve" {
		t.Fatalf("the body is %+v", body)
	}
}

func TestAnthropicRetry(t *testing.T) {
	n := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		n++
		if n == 1 {
			http.Error(w, "overloaded", 529)
			return
		}
		if n == 2 {
			http.Error(w, "bad", http.StatusBadRequest)
			return
		}
		_, _ = io.WriteString(w, sse)
	}))
	defer srv.Close()
	a := &Anthropic{Key: "k", BaseURL: srv.URL, HTTP: srv.Client(), Retries: 3}
	got, _ := a.Complete(context.Background(), []Request{{Model: Haiku}})
	if got[0].Err == nil || !strings.Contains(got[0].Err.Error(), "400") || n != 2 {
		t.Fatalf("the error is %v after %d calls; want the 400 after a retry", got[0].Err, n)
	}
}

func TestAnthropicBatch(t *testing.T) {
	polls := 0
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/v1/messages/batches":
			var in struct {
				Requests []struct {
					CustomID string      `json:"custom_id"`
					Params   wireRequest `json:"params"`
				} `json:"requests"`
			}
			_ = json.NewDecoder(r.Body).Decode(&in)
			if len(in.Requests) != 2 || in.Requests[1].CustomID != "r1" || in.Requests[0].Params.Stream {
				http.Error(w, "bad batch", http.StatusBadRequest)
				return
			}
			_, _ = io.WriteString(w, `{"id":"b1","processing_status":"in_progress"}`)
		case r.URL.Path == "/v1/messages/batches/b1":
			polls++
			fmt.Fprintf(w, `{"id":"b1","processing_status":"ended","results_url":"%s/results/b1"}`, srv.URL)
		case r.URL.Path == "/results/b1":
			_, _ = io.WriteString(w, `{"custom_id":"r1","result":{"type":"errored","error":{"type":"invalid_request_error"}}}
{"custom_id":"r0","result":{"type":"succeeded","message":{"content":[{"type":"thinking","thinking":"t"},{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":4}}}}
`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	a := &Anthropic{Key: "k", BaseURL: srv.URL, HTTP: srv.Client(), Batch: true, Poll: time.Millisecond}
	got, err := a.Complete(context.Background(), []Request{{ID: "a", Model: Haiku}, {ID: "b", Model: Haiku}})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Text != "ok" || !got[0].Batch || got[0].Usage.Output != 4 || got[0].ID != "a" || got[1].Err == nil || polls != 1 {
		t.Fatalf("the responses are %+v after %d polls", got, polls)
	}
}
