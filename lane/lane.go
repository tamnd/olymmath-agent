// Package lane holds the transports to the vision models. Each model call goes through Asker, which has one method. So a test drives the code above it with a function and makes no network call, and no code in this package uses an API key: each lane runs a local command-line client with its own login.
package lane

import (
	"context"
	"errors"
)

// Question is what goes to a lane.
type Question struct {
	// Prompt is the whole message, with the instructions.
	Prompt string
	// Images are the paths of the image files that go with the prompt.
	Images []string
}

// Answer is what comes back.
type Answer struct {
	Text string
	// Lane is the name of the lane that answered, such as codex.
	Lane string
	// Model is the model that answered, when the client tells it.
	Model string
	// Version is the version of the client, such as 2.1.318, when it tells it.
	Version string
	Seconds float64
}

// Asker is the interface between om and a model.
type Asker interface {
	Ask(ctx context.Context, q Question) (Answer, error)
}

// AskerFunc makes a function an Asker. Tests use it.
type AskerFunc func(context.Context, Question) (Answer, error)

// Ask implements Asker.
func (f AskerFunc) Ask(ctx context.Context, q Question) (Answer, error) { return f(ctx, q) }

var (
	// ErrExhausted is an allowance that is spent. A QuotaError wraps it. It does not clear in minutes, so a caller stops and does not try again.
	ErrExhausted = errors.New("the lane has spent its allowance")
	// ErrUnavailable is a lane that cannot answer at all on this machine, for example a client with no login. A chain skips it for the rest of the run.
	ErrUnavailable = errors.New("the lane is not available")
	// ErrEmptyAnswer is a call that succeeded and gave no text. It is an error and not an empty answer, because an empty page that a caller takes as read is a page that is lost.
	ErrEmptyAnswer = errors.New("the lane gave no text")
)
