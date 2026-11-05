// Package vision is a client of a vision model that a server of the user serves through the OpenAI chat-completions API, such as vLLM on a GPU host or an MLX server on this Mac. It sends one image and a prompt, and it gives the text of the answer. It sends no API key: the servers that it is for listen on a loopback address of this machine or of a host that the user reaches by ssh.
//
// A Transport carries the bytes. HTTP posts to a base URL. SSH runs curl on the host through ssh and posts to the loopback address of the host, so the server can bind 127.0.0.1 and no port forward is necessary.
package vision

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrUnreachable is a server that does not answer: the host is down, ssh fails, or nothing listens on the port.
	ErrUnreachable = errors.New("the server does not answer")
	// ErrRefused is an answer that is not the text of the image: a refusal, an empty message, or an answer that the token limit cut. A second call with the same image gives the same answer, so the caller does not try again at once.
	ErrRefused = errors.New("the model did not read the image")
)

// Error is an answer with a status that is not 2xx. Message is what the server said, not only its status, because "HTTP 400" is the same for each failure and the message of the server tells what to do.
type Error struct {
	Status  int
	Message string
	// RetryAfter is the wait that the server asked for, or 0.
	RetryAfter time.Duration
}

func (e *Error) Error() string {
	if e.Message == "" {
		return fmt.Sprintf("HTTP %d", e.Status)
	}
	return fmt.Sprintf("HTTP %d: %s", e.Status, e.Message)
}

// Busy tells if the server asks the caller to wait: a rate limit.
func (e *Error) Busy() bool { return e.Status == http.StatusTooManyRequests }

// Down tells if the server has a failure of its own, which is not the fault of the request.
func (e *Error) Down() bool { return e.Status >= 500 }

// Reply is what a transport got back.
type Reply struct {
	Status int
	Body   []byte
	// RetryAfter is the Retry-After header, or 0 when the transport does not see the headers.
	RetryAfter time.Duration
}

// Transport sends one request to the server. path is relative to the base URL, such as "chat/completions". An error from Do wraps ErrUnreachable when the server did not answer.
type Transport interface {
	Do(ctx context.Context, method, path string, body []byte) (Reply, error)
	// Name names the target in a log line, such as 127.0.0.1:8079 or gpc.
	Name() string
}

// Request is one image and its prompt.
type Request struct {
	// Model is the name that the server serves the model under.
	Model  string
	Prompt string
	// Image is the bytes of the image, and MediaType is its type, such as image/png.
	Image     []byte
	MediaType string
	// MaxTokens limits the answer, or 0 for no limit. With no limit the server gives the answer what is left of the context after the prompt and the image. A fixed limit is wrong for a model with a small context, and the finish reason "length" still catches an answer that runs away.
	MaxTokens int
}

// Response is the text of an answer.
type Response struct {
	Text string
	// Model is the model that the server says answered.
	Model string
	// Finish is the finish reason, such as stop.
	Finish string
	// Input and Output are the tokens that the server counted, when it tells them.
	Input, Output int
}

// refusals are the phrases with which a model declines to read an image. They are short and exact, because a page of mathematics can hold almost any sentence, and a loose phrase refuses a page that was read well.
var refusals = []string{
	"i'm sorry, i can't",
	"i cannot assist",
	"i can't help with that",
	"unable to process this image",
}

type textPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type imagePart struct {
	Type     string `json:"type"`
	ImageURL struct {
		URL string `json:"url"`
	} `json:"image_url"`
}

type message struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type chatRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Stream      bool      `json:"stream"`
}

// Payload gives the body of the request: one user message with the image as a data URL and then the prompt. The temperature is 0, because a transcription has nothing to choose, and at a higher temperature a hard formula comes out different on each call.
func Payload(r Request) ([]byte, error) {
	if strings.TrimSpace(r.Model) == "" {
		return nil, errors.New("vision: no model")
	}
	var parts []any
	if len(r.Image) > 0 {
		t := r.MediaType
		if t == "" {
			t = "image/png"
		}
		var p imagePart
		p.Type = "image_url"
		p.ImageURL.URL = "data:" + t + ";base64," + base64.StdEncoding.EncodeToString(r.Image)
		parts = append(parts, p)
	}
	parts = append(parts, textPart{Type: "text", Text: r.Prompt})
	return json.Marshal(chatRequest{
		Model:     r.Model,
		Messages:  []message{{Role: "user", Content: parts}},
		MaxTokens: r.MaxTokens,
	})
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		FinishReason string `json:"finish_reason"`
		Message      struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

// Read asks for the text of one image.
func Read(ctx context.Context, t Transport, r Request) (Response, error) {
	if len(r.Image) == 0 {
		return Response{}, errors.New("vision: no image")
	}
	body, err := Payload(r)
	if err != nil {
		return Response{}, err
	}
	reply, err := t.Do(ctx, http.MethodPost, "chat/completions", body)
	if err != nil {
		return Response{}, err
	}
	cr, err := decode(reply)
	if err != nil {
		return Response{}, err
	}
	if len(cr.Choices) == 0 {
		return Response{}, fmt.Errorf("%w: the server gave no choices", ErrRefused)
	}
	c := cr.Choices[0]
	out := Response{Text: strings.TrimSpace(c.Message.Content), Model: cr.Model, Finish: c.FinishReason, Input: cr.Usage.PromptTokens, Output: cr.Usage.CompletionTokens}
	if out.Text == "" {
		return out, fmt.Errorf("%w: the server gave an empty message", ErrRefused)
	}
	low := strings.ToLower(strings.ReplaceAll(out.Text, "’", "'"))
	for _, p := range refusals {
		if strings.Contains(low, p) {
			first, _, _ := strings.Cut(out.Text, "\n")
			return out, fmt.Errorf("%w: %s", ErrRefused, oneLine(first, 120))
		}
	}
	if c.FinishReason == "length" {
		return out, fmt.Errorf("%w: the answer reached the token limit, so it is cut", ErrRefused)
	}
	return out, nil
}

// Health asks the model for one token. A list of the models is not a test: a server whose engine is stuck still answers GET /v1/models, because the list does not go through the engine. Only a token that comes back shows that the engine works.
func Health(ctx context.Context, t Transport, model string) error {
	body, err := json.Marshal(chatRequest{
		Model:     model,
		Messages:  []message{{Role: "user", Content: "hi"}},
		MaxTokens: 1,
	})
	if err != nil {
		return err
	}
	reply, err := t.Do(ctx, http.MethodPost, "chat/completions", body)
	if err != nil {
		return err
	}
	cr, err := decode(reply)
	if err != nil {
		return err
	}
	if len(cr.Choices) == 0 {
		return fmt.Errorf("%w: the server gave no choices", ErrUnreachable)
	}
	return nil
}

// decode reads the body of a reply, or gives the error that the server described.
func decode(r Reply) (chatResponse, error) {
	var cr chatResponse
	if r.Status < 200 || r.Status >= 300 {
		return cr, &Error{Status: r.Status, Message: errorMessage(r.Body), RetryAfter: r.RetryAfter}
	}
	if err := json.Unmarshal(r.Body, &cr); err != nil {
		return cr, fmt.Errorf("vision: the answer is not JSON: %w: %s", err, oneLine(string(r.Body), 120))
	}
	return cr, nil
}

// errorMessage gives the message in the body of an error: the message of an OpenAI error, the detail of a FastAPI error, or else the body.
func errorMessage(body []byte) string {
	var env struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
		Detail  any             `json:"detail"`
	}
	if json.Unmarshal(body, &env) == nil {
		var e struct {
			Message string `json:"message"`
		}
		var s string
		switch {
		case json.Unmarshal(env.Error, &e) == nil && e.Message != "":
			return oneLine(e.Message, 400)
		case json.Unmarshal(env.Error, &s) == nil && s != "":
			return oneLine(s, 400)
		case env.Message != "":
			return oneLine(env.Message, 400)
		case env.Detail != nil:
			return oneLine(fmt.Sprint(env.Detail), 400)
		}
	}
	return oneLine(string(body), 400)
}

// retryAfter reads a Retry-After header in seconds or as an HTTP date.
func retryAfter(v string) time.Duration {
	v = strings.TrimSpace(v)
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(0, time.Until(t))
	}
	return 0
}

// oneLine keeps at most n bytes of a message, on one line.
func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > n {
		s = s[:n] + "..."
	}
	return s
}
