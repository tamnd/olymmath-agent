package model

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/lane"
)

// CLIPrefix starts the id of a model that a local command-line client answers through a lane: cli:<client> for the default model of the client, or cli:<client>/<model>. Examples are cli:gemini, cli:codex and cli:gemini/gemini-2.5-pro.
const CLIPrefix = "cli:"

// The model families. Spec 07.4 asks for a juror of another family than the generator.
const (
	FamilyAnthropic = "anthropic"
	FamilyOpenAI    = "openai"
	FamilyGoogle    = "google"
)

// clientFamily gives the family of the models of each command-line client.
var clientFamily = map[string]string{"codex": FamilyOpenAI, "gemini": FamilyGoogle, "claude": FamilyAnthropic}

// IsCLI reports whether a local command-line client answers the model id.
func IsCLI(id string) bool { return strings.HasPrefix(id, CLIPrefix) }

// ParseCLI splits a cli: model id into the client, such as gemini, and the model to ask, or "" for the default of the client.
func ParseCLI(id string) (client, name string, err error) {
	rest, ok := strings.CutPrefix(id, CLIPrefix)
	if !ok {
		return "", "", fmt.Errorf("the model %s is not of the form cli:<client>", id)
	}
	client, name, _ = strings.Cut(rest, "/")
	if _, ok := clientFamily[client]; !ok {
		return "", "", fmt.Errorf("the model %s names no known client; use one of cli:codex, cli:gemini or cli:claude", id)
	}
	return client, name, nil
}

// Family gives the family of a model id, or "" when it is not known. A cli: id gets the family of its client. The real model of a cli: answer, such as gpt-5.5 or gemini-2.5-pro, also has a family.
func Family(id string) string {
	if client, name, err := ParseCLI(id); err == nil {
		if name != "" {
			if f := Family(name); f != "" {
				return f
			}
		}
		return clientFamily[client]
	}
	switch id = strings.ToLower(id); {
	case strings.HasPrefix(id, "claude"):
		return FamilyAnthropic
	case strings.HasPrefix(id, "gemini"):
		return FamilyGoogle
	case strings.HasPrefix(id, "gpt"), strings.HasPrefix(id, "codex"), len(id) > 1 && id[0] == 'o' && id[1] >= '0' && id[1] <= '9':
		return FamilyOpenAI
	}
	return ""
}

// CLI is a Client that sends each request to a local command-line client, such as codex or gemini, through a lane of the lane package. Each client runs with its own login, so CLI sends no API key. The calls cost nothing here, because the plan of the client pays for them. The usage is the token count that the client gives (codex and claude), or else an estimate of four bytes a token. A client whose allowance is spent rests until the time that it gave, and a client with no login is skipped for the rest of the run (lane.Chain). The other requests of the list still go.
//
// A command-line client takes one prompt and keeps no conversation, so CLI writes the system text and the messages as one prompt. The effort and the output limit of a request do not reach the client.
type CLI struct {
	// Lane gives the lane of a client and a model ("" for the default model). NewCLI sets it to the clients on this machine. Tests give a fake.
	Lane func(client, name string) (lane.Named, error)
	// StateDir is the directory of the files that keep the rest of each lane between runs, or "" for none. So a run does not spend a call to learn again that an allowance is spent.
	StateDir string
	// Parallel is the most calls at the same time. It is 1 when it is 0.
	Parallel int
	// Timeout is the longest time of one call. NewCLI sets it to CLITimeout.
	Timeout time.Duration

	mu     sync.Mutex
	chains map[string]*lane.Chain
}

// CLITimeout is the longest time of one call of a CLI client. A solver or a grader can think for many minutes on a hard problem, so it is longer than lane.DefaultTimeout.
const CLITimeout = 30 * time.Minute

// NewCLI gives a CLI client for the command-line clients on this machine (lane.Defaults). stateDir is the directory of the rests of the lanes, or "".
func NewCLI(stateDir string) *CLI {
	c := &CLI{StateDir: stateDir, Timeout: CLITimeout}
	c.Lane = func(client, name string) (lane.Named, error) {
		for _, l := range lane.Defaults() {
			if l.Name() != client {
				continue
			}
			c, ok := l.(*lane.CLI)
			if !ok {
				return l, nil
			}
			out, err := lane.NewCLI(client, c.Path)
			if err != nil {
				return nil, err
			}
			out.Model = name
			if c.Timeout > 0 {
				out.Timeout = c.Timeout
			}
			return out, nil
		}
		return nil, fmt.Errorf("the client %s is not on this machine", client)
	}
	return c
}

// chain gives the chain of one lane for a model id, so that the rests of the lane apply to each call.
func (c *CLI) chain(id string) (*lane.Chain, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if ch := c.chains[id]; ch != nil {
		return ch, nil
	}
	client, name, err := ParseCLI(id)
	if err != nil {
		return nil, err
	}
	l, err := c.Lane(client, name)
	if err != nil {
		return nil, err
	}
	ch := &lane.Chain{Lanes: []lane.Named{l}}
	if c.StateDir != "" {
		// One file for each model id, so a spent allowance of one model does not stop another model of the same client.
		ch.State = filepath.Join(c.StateDir, strings.NewReplacer(":", "-", "/", "-").Replace(id)+".json")
	}
	if c.chains == nil {
		c.chains = map[string]*lane.Chain{}
	}
	c.chains[id] = ch
	return ch, nil
}

// Prompt writes a request as the one prompt of a command-line client.
func Prompt(r Request) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(r.System))
	for _, m := range r.Messages {
		b.WriteString("\n\n")
		if m.Role == "assistant" {
			b.WriteString("Your answer before this message:\n\n" + strings.TrimSpace(m.Text) + "\n\n(The end of your answer before this message.)")
			continue
		}
		b.WriteString(strings.TrimSpace(m.Text))
	}
	return b.String()
}

// Complete sends each request to the lane of its model. An error of one request is in its response.
func (c *CLI) Complete(ctx context.Context, reqs []Request) ([]Response, error) {
	out := make([]Response, len(reqs))
	n := max(c.Parallel, 1)
	sem := make(chan struct{}, n)
	var wg sync.WaitGroup
	for i, r := range reqs {
		ch, err := c.chain(r.Model)
		if err != nil {
			out[i] = Response{ID: r.ID, Err: err}
			continue
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			out[i] = c.one(ctx, ch, r)
		}()
	}
	wg.Wait()
	return out, nil
}

func (c *CLI) one(ctx context.Context, ch *lane.Chain, r Request) Response {
	p := Prompt(r)
	res := Response{ID: r.ID}
	a, err := ch.Ask(ctx, lane.Question{Prompt: p})
	if err != nil {
		res.Err = fmt.Errorf("%s: %w", r.Model, err)
		if errors.Is(err, lane.ErrExhausted) {
			res.StopReason = "quota"
		}
		return res
	}
	res.Text, res.StopReason, res.Model, res.Seconds = a.Text, "end_turn", a.Model, a.Seconds
	res.Usage = content.Tokens{Input: Tokens(p), Output: Tokens(a.Text)}
	if u := a.Usage; u.Known() {
		res.Usage = content.Tokens{Input: u.Input, Output: u.Output, CacheRead: u.CacheRead, CacheWrite: u.CacheWrite}
	}
	return res
}

// Split sends the requests of cli: models to CLI and the other requests to API.
type Split struct {
	// API answers the models of the Messages API. When it is nil, a request for one of them gets ErrNoKey.
	API Client
	CLI Client
}

// Complete implements Client.
func (s *Split) Complete(ctx context.Context, reqs []Request) ([]Response, error) {
	out := make([]Response, len(reqs))
	var api, cli []Request
	var atAPI, atCLI []int
	for i, r := range reqs {
		switch {
		case IsCLI(r.Model):
			cli, atCLI = append(cli, r), append(atCLI, i)
		case s.API == nil:
			out[i] = Response{ID: r.ID, Err: ErrNoKey}
		default:
			api, atAPI = append(api, r), append(atAPI, i)
		}
	}
	for _, part := range []struct {
		c    Client
		reqs []Request
		at   []int
	}{{s.API, api, atAPI}, {s.CLI, cli, atCLI}} {
		if len(part.reqs) == 0 {
			continue
		}
		if part.c == nil {
			for _, i := range part.at {
				out[i] = Response{ID: reqs[i].ID, Err: errors.New("no command-line client is set")}
			}
			continue
		}
		got, err := part.c.Complete(ctx, part.reqs)
		if err != nil {
			return nil, err
		}
		for j, res := range got {
			out[part.at[j]] = res
		}
	}
	return out, nil
}
