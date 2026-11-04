package model

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/olymmath-agent/content"
)

// The defaults of the Anthropic client.
const (
	AnthropicURL     = "https://api.anthropic.com"
	AnthropicVersion = "2026-10-21"
	// KeyEnv is the environment variable that holds the API key.
	KeyEnv = "ANTHROPIC_API_KEY"
)

// Anthropic calls the Anthropic Messages API over HTTP. It streams each call, so a long proof does not hit the timeout of a request, or it sends the whole list as one message batch at half the price.
type Anthropic struct {
	Key     string
	BaseURL string
	Version string
	HTTP    *http.Client
	// Batch sends the requests through the Message Batches API and waits for the batch to end.
	Batch bool
	// Poll is the time between two checks of a batch.
	Poll time.Duration
	// Parallel is the number of streamed calls at a time.
	Parallel int
	// Retries is the number of retries of a call that gets 429, 500, 529 or a network error.
	Retries int
}

// NewAnthropic gives a client with the key of the environment. It fails when the key is not set.
func NewAnthropic() (*Anthropic, error) {
	key := strings.TrimSpace(os.Getenv(KeyEnv))
	if key == "" {
		return nil, ErrNoKey
	}
	return &Anthropic{
		Key:      key,
		BaseURL:  AnthropicURL,
		Version:  AnthropicVersion,
		HTTP:     &http.Client{Timeout: 30 * time.Minute},
		Poll:     time.Minute,
		Parallel: 4,
		Retries:  4,
	}, nil
}

// wireBlock is a text block of the API.
type wireBlock struct {
	Type         string         `json:"type"`
	Text         string         `json:"text,omitempty"`
	CacheControl *wireCacheCtrl `json:"cache_control,omitempty"`
}

type wireCacheCtrl struct {
	Type string `json:"type"`
}

type wireMessage struct {
	Role    string      `json:"role"`
	Content []wireBlock `json:"content"`
}

type wireOutputConfig struct {
	Effort string `json:"effort,omitempty"`
}

// wireRequest is the body of a call. It sends no temperature, top_p or top_k: the current models refuse them.
type wireRequest struct {
	Model        string            `json:"model"`
	MaxTokens    int               `json:"max_tokens"`
	System       []wireBlock       `json:"system,omitempty"`
	Messages     []wireMessage     `json:"messages"`
	OutputConfig *wireOutputConfig `json:"output_config,omitempty"`
	Stream       bool              `json:"stream,omitempty"`
}

type wireUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
}

func (u wireUsage) tokens() content.Tokens {
	return content.Tokens{Input: u.InputTokens, Output: u.OutputTokens, CacheRead: u.CacheReadInputTokens, CacheWrite: u.CacheCreationInputTokens}
}

// wireMessageResponse is a message of the API, as a response or as a batch result.
type wireMessageResponse struct {
	Content    []wireBlock `json:"content"`
	StopReason string      `json:"stop_reason"`
	Usage      wireUsage   `json:"usage"`
}

func (m wireMessageResponse) text() string {
	var b strings.Builder
	for _, c := range m.Content {
		// The response also has thinking blocks. Only the text blocks are the answer.
		if c.Type == "text" {
			b.WriteString(c.Text)
		}
	}
	return b.String()
}

func wire(r Request, stream bool) wireRequest {
	w := wireRequest{Model: r.Model, MaxTokens: r.MaxTokens, Stream: stream}
	if r.System != "" {
		w.System = []wireBlock{{Type: "text", Text: r.System, CacheControl: &wireCacheCtrl{Type: "ephemeral"}}}
	}
	for _, m := range r.Messages {
		w.Messages = append(w.Messages, wireMessage{Role: m.Role, Content: []wireBlock{{Type: "text", Text: m.Text}}})
	}
	if r.Effort != "" {
		w.OutputConfig = &wireOutputConfig{Effort: r.Effort}
	}
	return w
}

// Complete sends the requests.
func (a *Anthropic) Complete(ctx context.Context, reqs []Request) ([]Response, error) {
	if a.Key == "" {
		return nil, ErrNoKey
	}
	if a.Batch {
		return a.batch(ctx, reqs)
	}
	out := make([]Response, len(reqs))
	sem := make(chan struct{}, max(a.Parallel, 1))
	var wg sync.WaitGroup
	for i, r := range reqs {
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			out[i] = a.one(ctx, r)
		}()
	}
	wg.Wait()
	return out, nil
}

// one streams one call, with retries.
func (a *Anthropic) one(ctx context.Context, r Request) Response {
	body, err := json.Marshal(wire(r, true))
	if err != nil {
		return Response{ID: r.ID, Err: err}
	}
	var last error
	for try := 0; try <= a.Retries; try++ {
		if try > 0 {
			select {
			case <-ctx.Done():
				return Response{ID: r.ID, Err: ctx.Err()}
			case <-time.After(time.Duration(1<<try) * time.Second):
			}
		}
		res, retry, err := a.stream(ctx, body)
		if err == nil {
			res.ID = r.ID
			if res.StopReason == "refusal" {
				res.Err = errors.New("the model refused the request")
			}
			return res
		}
		last = err
		if !retry {
			break
		}
	}
	return Response{ID: r.ID, Err: last}
}

func (a *Anthropic) post(ctx context.Context, path string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, a.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	a.headers(req)
	req.Header.Set("content-type", "application/json")
	return a.HTTP.Do(req) // #nosec G704 -- the URL is the configured API.
}

func (a *Anthropic) get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	a.headers(req)
	return a.HTTP.Do(req) // #nosec G704 -- the URL is the configured API.
}

func (a *Anthropic) headers(req *http.Request) {
	req.Header.Set("x-api-key", a.Key)
	req.Header.Set("anthropic-version", a.Version)
}

// retryable reports whether a status is worth a retry.
func retryable(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

func apiError(resp *http.Response) error {
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return fmt.Errorf("the API answered %s: %s", resp.Status, strings.TrimSpace(string(b)))
}

// stream sends one streamed call and reads its events.
func (a *Anthropic) stream(ctx context.Context, body []byte) (Response, bool, error) {
	resp, err := a.post(ctx, "/v1/messages", body)
	if err != nil {
		return Response{}, true, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return Response{}, retryable(resp.StatusCode), apiError(resp)
	}
	return readStream(resp.Body)
}

// streamEvent holds the fields of the stream events that om reads.
type streamEvent struct {
	Type    string `json:"type"`
	Message struct {
		Usage wireUsage `json:"usage"`
	} `json:"message"`
	Delta struct {
		Type       string `json:"type"`
		Text       string `json:"text"`
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage *wireUsage `json:"usage"`
	Error *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

// readStream reads the server-sent events of a streamed message: the text deltas, the stop reason and the usage.
func readStream(r io.Reader) (Response, bool, error) {
	var res Response
	var text strings.Builder
	var usage wireUsage
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	done := false
	for sc.Scan() {
		line := sc.Text()
		data, ok := strings.CutPrefix(line, "data:")
		if !ok {
			continue
		}
		var ev streamEvent
		if err := json.Unmarshal([]byte(strings.TrimSpace(data)), &ev); err != nil {
			return res, false, fmt.Errorf("a stream event is not JSON: %w", err)
		}
		switch ev.Type {
		case "message_start":
			usage = ev.Message.Usage
		case "content_block_delta":
			if ev.Delta.Type == "text_delta" {
				text.WriteString(ev.Delta.Text)
			}
		case "message_delta":
			if ev.Delta.StopReason != "" {
				res.StopReason = ev.Delta.StopReason
			}
			if ev.Usage != nil {
				// The usage of message_delta is the running total of the output tokens.
				usage.OutputTokens = ev.Usage.OutputTokens
			}
		case "message_stop":
			done = true
		case "error":
			msg := "an error in the stream"
			retry := false
			if ev.Error != nil {
				msg = ev.Error.Type + ": " + ev.Error.Message
				retry = ev.Error.Type == "overloaded_error" || ev.Error.Type == "api_error"
			}
			return res, retry, errors.New(msg)
		}
	}
	if err := sc.Err(); err != nil {
		return res, true, err
	}
	if !done {
		return res, true, errors.New("the stream ended before message_stop")
	}
	res.Text = text.String()
	res.Usage = usage.tokens()
	return res, false, nil
}

// batch sends the requests as one message batch, waits for it and reads its results.
func (a *Anthropic) batch(ctx context.Context, reqs []Request) ([]Response, error) {
	type item struct {
		CustomID string      `json:"custom_id"`
		Params   wireRequest `json:"params"`
	}
	in := struct {
		Requests []item `json:"requests"`
	}{}
	at := map[string]int{}
	for i, r := range reqs {
		id := fmt.Sprintf("r%d", i)
		at[id] = i
		in.Requests = append(in.Requests, item{id, wire(r, false)})
	}
	body, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	resp, err := a.post(ctx, "/v1/messages/batches", body)
	if err != nil {
		return nil, err
	}
	var b struct {
		ID               string `json:"id"`
		ProcessingStatus string `json:"processing_status"`
		ResultsURL       string `json:"results_url"`
	}
	err = decode(resp, &b)
	if err != nil {
		return nil, err
	}
	for b.ProcessingStatus != "ended" {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("the batch %s did not end: %w", b.ID, ctx.Err())
		case <-time.After(a.Poll):
		}
		resp, err := a.get(ctx, a.BaseURL+"/v1/messages/batches/"+b.ID)
		if err != nil {
			continue
		}
		if err := decode(resp, &b); err != nil {
			return nil, err
		}
	}
	resp, err = a.get(ctx, b.ResultsURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, apiError(resp)
	}
	out := make([]Response, len(reqs))
	for i, r := range reqs {
		out[i] = Response{ID: r.ID, Err: errors.New("the batch has no result for the request")}
	}
	sc := bufio.NewScanner(resp.Body)
	sc.Buffer(make([]byte, 1<<20), 1<<26)
	for sc.Scan() {
		var line struct {
			CustomID string `json:"custom_id"`
			Result   struct {
				Type    string              `json:"type"`
				Message wireMessageResponse `json:"message"`
				Error   json.RawMessage     `json:"error"`
			} `json:"result"`
		}
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			return nil, fmt.Errorf("a batch result is not JSON: %w", err)
		}
		i, ok := at[line.CustomID]
		if !ok {
			continue
		}
		res := Response{ID: reqs[i].ID, Batch: true}
		switch line.Result.Type {
		case "succeeded":
			m := line.Result.Message
			res.Text, res.StopReason, res.Usage = m.text(), m.StopReason, m.Usage.tokens()
			if res.StopReason == "refusal" {
				res.Err = errors.New("the model refused the request")
			}
		default:
			res.Err = fmt.Errorf("the batch request %s: %s", line.Result.Type, strings.TrimSpace(string(line.Result.Error)))
		}
		out[i] = res
	}
	return out, sc.Err()
}

func decode(resp *http.Response, v any) error {
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return apiError(resp)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}
