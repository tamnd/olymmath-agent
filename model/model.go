// Package model calls language models for the solver and the grader of spec 07. A Client sends a list of requests and gives one response for each. The Anthropic client calls the Messages API; the Fake client answers in tests and in dry runs.
package model

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"

	"github.com/tamnd/olymmath-agent/content"
)

// The default models of spec 07.3 and 07.4.
const (
	Opus   = "claude-opus-5-5"
	Sonnet = "claude-sonnet-5-5"
	Haiku  = "claude-haiku-5-5"
)

// Message is one turn of a conversation.
type Message struct {
	// Role is user or assistant.
	Role string `json:"role"`
	Text string `json:"text"`
}

// Request is one model call.
type Request struct {
	// ID names the request inside a list. It is the custom id of a batch request.
	ID string `json:"id"`
	// Kind is the stage of the call, for example generate, verify, revise or grade. It is for the logs and the fake client; the API does not see it.
	Kind  string `json:"kind"`
	Model string `json:"model"`
	// System is the shared prefix of the calls: the instructions and the problem. The client caches it.
	System    string    `json:"system"`
	Messages  []Message `json:"messages"`
	MaxTokens int       `json:"max_tokens"`
	// Effort is low, medium, high or max, or empty for the default of the model.
	Effort string `json:"effort,omitempty"`
}

// Key is the SHA-256 of the request, without its id. Two equal requests have the same key, so a resumed run finds the response of a request in the cache.
func (r Request) Key() string {
	r.ID = ""
	b, _ := json.Marshal(r)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

// Response is the answer to one request.
type Response struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// StopReason is end_turn, max_tokens, refusal or another stop reason of the API.
	StopReason string         `json:"stop_reason"`
	Usage      content.Tokens `json:"usage"`
	// Batch is true when the call went through the Batch API, at half the price.
	Batch bool `json:"batch,omitempty"`
	// Cached is true when the response came from the cache of a resumed run. It cost nothing this time.
	Cached bool `json:"-"`
	// Err is the error of this request. The other requests of the list can still succeed.
	Err error `json:"-"`
}

// Client sends model requests.
type Client interface {
	// Complete sends the requests and gives the responses in the same order. An error means that no request was sent; the error of one request is in its response.
	Complete(ctx context.Context, reqs []Request) ([]Response, error)
}

// ErrNoKey means that the environment has no API key.
var ErrNoKey = errors.New("ANTHROPIC_API_KEY is not set; om does not call a model without it")

// ErrBudget means that the next calls would pass the budget of the run.
var ErrBudget = errors.New("the budget of the run is spent")
