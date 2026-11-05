package lane

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/olymmath-agent/vision"
)

// LocalName is the name of the lane of the vision servers of the user.
const LocalName = "local"

// DefaultLocal are the targets of the local lane when the user gives none: GLM-OCR on an MLX server of this Mac, and then olmOCR 2 under vLLM on the GPU host gpc, through ssh.
const DefaultLocal = "GLM-OCR-8bit=http://127.0.0.1:8079/v1,reader-a=ssh://gpc/http://127.0.0.1:8801/v1"

// Rests of a target of the local lane.
const (
	// BusyRest is the rest after a 429 that gives no Retry-After.
	BusyRest = time.Minute
	// DownRest is the rest of a target that does not answer, or whose server fails.
	DownRest = 5 * time.Minute
	// LocalTimeout is the longest time of one page. A server that reads one page at a time puts a page behind the pages of other programs.
	LocalTimeout = 15 * time.Minute
	// HealthTimeout is the longest time of a health check. The check waits behind the pages that the server reads now.
	HealthTimeout = 45 * time.Second
)

// ErrDown is a lane whose servers do not answer now. A RestError wraps it.
var ErrDown = errors.New("no server of the lane answers")

// ErrRefused is a page that the model did not read: a refusal, an empty message, an answer that the token limit cut, or a request that the server rejects, such as an image that is too large for the context. The lane is not at fault, so a chain does not rest it, and the caller goes on to the next page.
var ErrRefused = errors.New("the model did not read the page")

// RestError is a lane all of whose targets rest. Until is when the first of them comes back. A chain rests the lane until then.
type RestError struct {
	Lane    string
	Message string
	Until   time.Time
}

func (e *RestError) Error() string {
	s := fmt.Sprintf("%s: %s until %s", e.Lane, ErrDown, e.Until.Format(time.DateTime))
	if e.Message != "" {
		s += ": " + e.Message
	}
	return s
}

// Unwrap lets errors.Is find ErrDown.
func (e *RestError) Unwrap() error { return ErrDown }

// Rest is how long to stay off the lane after now.
func (e *RestError) Rest(now time.Time) time.Duration { return max(0, e.Until.Sub(now)) }

// Target is one server of the local lane and the model that it serves.
type Target struct {
	// Model is the name that the server serves the model under.
	Model     string
	Transport vision.Transport

	until   time.Time
	checked bool
	// dead is a target whose health check the server rejects, such as a model that the server does not serve. It does not come back in this run.
	dead bool
}

// Name names the target in a log line, such as GLM-OCR-8bit@127.0.0.1:8079.
func (t *Target) Name() string { return t.Model + "@" + t.Transport.Name() }

// ParseTargets reads a comma-separated list of targets. Each target is MODEL=URL. URL is a base URL, such as http://127.0.0.1:8079/v1, or ssh://HOST/URL for a server on the loopback address of an ssh host, such as ssh://gpc/http://127.0.0.1:8801/v1.
func ParseTargets(spec string, run vision.Runner) ([]*Target, error) {
	var out []*Target
	for _, f := range strings.Split(spec, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		model, u, ok := strings.Cut(f, "=")
		model, u = strings.TrimSpace(model), strings.TrimSpace(u)
		if !ok || model == "" || u == "" {
			return nil, fmt.Errorf("lane: target %q is not MODEL=URL", f)
		}
		t := &Target{Model: model}
		switch {
		case strings.HasPrefix(u, "ssh://"):
			host, base, ok := strings.Cut(strings.TrimPrefix(u, "ssh://"), "/")
			if !ok || host == "" || strings.HasPrefix(host, "-") || !strings.HasPrefix(base, "http") {
				return nil, fmt.Errorf("lane: target %q is not ssh://HOST/http://ADDRESS/v1", f)
			}
			t.Transport = &vision.SSH{Host: host, URL: base, Run: run}
		case strings.HasPrefix(u, "http://"), strings.HasPrefix(u, "https://"):
			t.Transport = &vision.HTTP{BaseURL: u}
		default:
			return nil, fmt.Errorf("lane: target %q has no http://, https:// or ssh:// URL", f)
		}
		out = append(out, t)
	}
	if len(out) == 0 {
		return nil, errors.New("lane: the local lane has no target")
	}
	return out, nil
}

// Local is the lane of the vision servers of the user, through the OpenAI chat-completions API. It sends no API key and costs nothing.
//
// The targets of one model share the pages in turns (round robin). The targets of the next model in the list are for when no target of the first model can answer, so the pages of one paper come from one model when they can. A target that answers 429 rests for the time that it asks, or BusyRest. A target that does not answer, or whose server fails, rests for DownRest. Before the first page, and after each rest of DownRest, a target must give one token (vision.Health). When all targets rest, Ask gives a RestError.
type Local struct {
	Targets []*Target
	// Timeout is the longest time of one page. It is LocalTimeout when it is 0.
	Timeout time.Duration
	// Now gives the time. Tests replace it.
	Now func() time.Time

	mu   sync.Mutex
	next map[string]int
}

// NewLocal gives the local lane of a list of targets (ParseTargets).
func NewLocal(spec string) (*Local, error) {
	ts, err := ParseTargets(spec, nil)
	if err != nil {
		return nil, err
	}
	return &Local{Targets: ts}, nil
}

// Name gives LocalName.
func (l *Local) Name() string { return LocalName }

// Models gives the models of the targets, each one time, in the order of the list.
func (l *Local) Models() []string {
	var out []string
	for _, t := range l.Targets {
		if !slices.Contains(out, t.Model) {
			out = append(out, t.Model)
		}
	}
	return out
}

func (l *Local) now() time.Time {
	if l.Now != nil {
		return l.Now()
	}
	return time.Now()
}

// Check runs the health check of each target and gives an error for each target, or nil for a target that answers.
func (l *Local) Check(ctx context.Context) []error {
	out := make([]error, len(l.Targets))
	for i, t := range l.Targets {
		cctx, cancel := context.WithTimeout(ctx, HealthTimeout)
		out[i] = vision.Health(cctx, t.Transport, t.Model)
		cancel()
	}
	return out
}

// Ask reads the one image of the question.
func (l *Local) Ask(ctx context.Context, q Question) (Answer, error) {
	if len(q.Images) != 1 {
		return Answer{Lane: LocalName}, fmt.Errorf("%s: a question must have one image, not %d", LocalName, len(q.Images))
	}
	img, err := os.ReadFile(q.Images[0]) // #nosec G304 -- a page image of the caller.
	if err != nil {
		return Answer{Lane: LocalName}, err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.next == nil {
		l.next = map[string]int{}
	}
	var why []string
	for _, model := range l.Models() {
		var group []*Target
		for _, t := range l.Targets {
			if t.Model == model {
				group = append(group, t)
			}
		}
		first := l.next[model] % len(group)
		for i := range group {
			t := group[(first+i)%len(group)]
			a, done, err := l.try(ctx, t, q.Prompt, img, filepath.Ext(q.Images[0]))
			if done {
				l.next[model] = (first + i + 1) % len(group)
				return a, err
			}
			if err != nil {
				why = append(why, err.Error())
			}
		}
	}
	return Answer{Lane: LocalName}, l.restError(why)
}

// try asks one target. done is false when the target rests or did not answer, so Ask asks the next target.
func (l *Local) try(ctx context.Context, t *Target, prompt string, img []byte, ext string) (Answer, bool, error) {
	if t.dead || l.now().Before(t.until) {
		return Answer{}, false, nil
	}
	if !t.checked {
		hctx, cancel := context.WithTimeout(ctx, HealthTimeout)
		err := vision.Health(hctx, t.Transport, t.Model)
		cancel()
		if ctx.Err() != nil {
			return Answer{Lane: LocalName}, true, ctx.Err()
		}
		var ve *vision.Error
		switch {
		case err == nil:
			t.checked = true
		case errors.As(err, &ve) && !ve.Busy() && !ve.Down():
			t.dead = true
			return Answer{}, false, fmt.Errorf("%s: the health check: %w", t.Name(), err)
		default:
			l.rest(t, err)
			return Answer{}, false, fmt.Errorf("%s: the health check: %w", t.Name(), err)
		}
	}
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = LocalTimeout
	}
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	start := time.Now()
	res, err := vision.Read(cctx, t.Transport, vision.Request{Model: t.Model, Prompt: prompt, Image: img, MediaType: mediaType(ext)})
	a := Answer{Lane: LocalName, Model: t.Model, Seconds: time.Since(start).Seconds()}
	var ve *vision.Error
	switch {
	case err == nil:
		a.Text = res.Text
		return a, true, nil
	case ctx.Err() != nil:
		return a, true, ctx.Err()
	case errors.Is(err, vision.ErrRefused):
		return a, true, fmt.Errorf("%s: %s: %w: %w", LocalName, t.Name(), ErrRefused, err)
	case errors.As(err, &ve) && !ve.Busy() && !ve.Down():
		// A 4xx other than 429 is about this request, such as an image that is too large for the context. Another target gives the same answer.
		return a, true, fmt.Errorf("%s: %s: %w: %w", LocalName, t.Name(), ErrRefused, err)
	case errors.As(err, &ve), errors.Is(err, vision.ErrUnreachable), errors.Is(err, context.DeadlineExceeded):
		l.rest(t, err)
		return a, false, fmt.Errorf("%s: %w", t.Name(), err)
	default:
		return a, true, fmt.Errorf("%s: %s: %w", LocalName, t.Name(), err)
	}
}

// rest takes a target out of the turns: for the time that a 429 asks, or for DownRest. A target that did not answer must pass the health check again when it comes back.
func (l *Local) rest(t *Target, err error) {
	var ve *vision.Error
	if errors.As(err, &ve) && ve.Busy() {
		d := ve.RetryAfter
		if d <= 0 {
			d = BusyRest
		}
		t.until = l.now().Add(d)
		return
	}
	t.until = l.now().Add(DownRest)
	t.checked = false
}

// restError gives the error of a lane with no target that can answer: a RestError until the first target comes back, or ErrUnavailable when no target comes back in this run.
func (l *Local) restError(why []string) error {
	var wake time.Time
	for _, t := range l.Targets {
		if !t.dead && (wake.IsZero() || t.until.Before(wake)) {
			wake = t.until
		}
	}
	msg := strings.Join(why, "; ")
	if msg == "" {
		msg = "each target rests"
	}
	if wake.IsZero() {
		return fmt.Errorf("%s: %w: %s", LocalName, ErrUnavailable, msg)
	}
	return &RestError{Lane: LocalName, Message: msg, Until: wake}
}

// mediaType gives the type of an image from the extension of its file.
func mediaType(ext string) string {
	switch strings.ToLower(ext) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	}
	return "image/png"
}
