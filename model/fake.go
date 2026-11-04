package model

import (
	"context"
	"sync"

	"github.com/tamnd/olymmath-agent/content"
)

// Fake answers requests with a function, for tests. It counts the calls and keeps the requests.
type Fake struct {
	// Answer gives the text of the response to a request.
	Answer func(Request) string
	// Batch marks the responses as Batch API responses.
	Batch bool

	mu       sync.Mutex
	Requests []Request
}

// Complete answers each request. The usage counts four bytes a token.
func (f *Fake) Complete(_ context.Context, reqs []Request) ([]Response, error) {
	out := make([]Response, len(reqs))
	for i, r := range reqs {
		text := ""
		if f.Answer != nil {
			text = f.Answer(r)
		}
		in := Tokens(r.System)
		for _, m := range r.Messages {
			in += Tokens(m.Text)
		}
		out[i] = Response{ID: r.ID, Text: text, StopReason: "end_turn", Usage: content.Tokens{Input: in, Output: Tokens(text)}, Batch: f.Batch}
	}
	f.mu.Lock()
	f.Requests = append(f.Requests, reqs...)
	f.mu.Unlock()
	return out, nil
}

// Calls gives the number of requests so far.
func (f *Fake) Calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Requests)
}
