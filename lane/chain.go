package lane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Named is an Asker with a name.
type Named interface {
	Asker
	Name() string
}

// Chain asks its lanes in order. A lane whose allowance is spent rests until the time that its provider gave, and a lane that is not available on this machine is skipped for the rest of the run. Then the chain asks the next lane. When no lane can answer, Ask gives an error that wraps ErrExhausted, so that a caller stops the batch and does not loop.
type Chain struct {
	Lanes []Named
	// State is the file that keeps the rest of each lane between runs, or "" for none. So a run does not spend a call to learn again that an allowance is spent.
	State string
	// Now gives the time. Tests replace it.
	Now func() time.Time

	mu    sync.Mutex
	rest  map[string]time.Time
	skip  map[string]bool
	ready bool
}

// Name gives the names of the lanes of the chain.
func (c *Chain) Name() string {
	var names []string
	for _, l := range c.Lanes {
		names = append(names, l.Name())
	}
	return strings.Join(names, ",")
}

func (c *Chain) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// load reads the state file once.
func (c *Chain) load() {
	if c.ready {
		return
	}
	c.ready = true
	c.rest, c.skip = map[string]time.Time{}, map[string]bool{}
	if c.State == "" {
		return
	}
	b, err := os.ReadFile(c.State)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, &c.rest)
}

// save writes the state file. A failure to write it costs only a call on the next run, so it is not an error.
func (c *Chain) save() {
	if c.State == "" {
		return
	}
	b, err := json.MarshalIndent(c.rest, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(c.State), 0o750)
	_ = os.WriteFile(c.State, b, 0o600)
}

// Ready gives the names of the lanes that can be asked now.
func (c *Chain) Ready() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.load()
	var out []string
	for _, l := range c.Lanes {
		if !c.skip[l.Name()] && !c.now().Before(c.rest[l.Name()]) {
			out = append(out, l.Name())
		}
	}
	return out
}

// Ask asks the first lane that can answer.
func (c *Chain) Ask(ctx context.Context, q Question) (Answer, error) {
	var why []string
	for _, l := range c.Lanes {
		name := l.Name()
		c.mu.Lock()
		c.load()
		until, skip := c.rest[name], c.skip[name]
		c.mu.Unlock()
		if skip {
			why = append(why, name+" is not available")
			continue
		}
		if c.now().Before(until) {
			why = append(why, name+" rests until "+until.Format(time.DateTime))
			continue
		}
		a, err := l.Ask(ctx, q)
		var qe *QuotaError
		switch {
		case err == nil:
			return a, nil
		case errors.As(err, &qe):
			c.mu.Lock()
			c.rest[name] = c.now().Add(qe.Rest(c.now()))
			c.save()
			c.mu.Unlock()
			why = append(why, err.Error())
		case errors.Is(err, ErrExhausted):
			c.mu.Lock()
			c.rest[name] = c.now().Add(DefaultRest)
			c.save()
			c.mu.Unlock()
			why = append(why, err.Error())
		case errors.Is(err, ErrUnavailable):
			c.mu.Lock()
			c.skip[name] = true
			c.mu.Unlock()
			why = append(why, err.Error())
		default:
			return a, err
		}
	}
	return Answer{}, fmt.Errorf("no lane can answer (%s): %w", strings.Join(why, "; "), ErrExhausted)
}

// Defaults gives the lanes of the clients on this machine, in the order to ask them: codex, then gemini, then claude as the last choice. A client is first looked for at its usual path on a Mac, and then in the PATH. A client that is not found is left out.
func Defaults() []Named {
	home, _ := os.UserHomeDir()
	places := []struct{ kind, path string }{
		{"codex", "/opt/homebrew/bin/codex"},
		{"gemini", "/opt/homebrew/bin/gemini"},
		{"claude", filepath.Join(home, ".local", "bin", "claude")},
	}
	var out []Named
	for _, p := range places {
		path := p.path
		if _, err := os.Stat(path); err != nil {
			if path, err = exec.LookPath(p.kind); err != nil {
				continue
			}
		}
		l, err := NewCLI(p.kind, path)
		if err == nil {
			out = append(out, l)
		}
	}
	return out
}

// Select gives the lanes of all whose names are in the comma-separated list names, in the order of the list. An empty list gives all.
func Select(all []Named, names string) ([]Named, error) {
	if strings.TrimSpace(names) == "" {
		return all, nil
	}
	var out []Named
	for _, n := range strings.Split(names, ",") {
		n = strings.TrimSpace(n)
		found := false
		for _, l := range all {
			if l.Name() == n {
				out = append(out, l)
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("lane: no client %q on this machine", n)
		}
	}
	return out, nil
}
