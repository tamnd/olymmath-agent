// Package fetch gets web pages and files for the raw store. Each request uses the same identity, obeys robots.txt, waits between two requests to the same host, retries with back-off, and uses conditional requests when the store has an earlier capture.
package fetch

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/tamnd/olymmath-agent/store"
)

// Agent is the User-Agent of each request when the client has no other one.
const Agent = "olymmath/0 (+https://olymmath.tamnd.com/about/)"

// ErrRobots is the error when robots.txt does not let the client get a URL.
var ErrRobots = errors.New("robots.txt does not allow the URL")

// Client gets URLs into a store.
type Client struct {
	Store   *store.Store
	HTTP    *http.Client
	Agent   string
	Delay   time.Duration // the minimum time between two requests to the same host
	Retries int           // the number of retries after a failure that can go away
	MaxWait time.Duration // the longest back-off

	mu     sync.Mutex
	robots map[string]*robots
	next   map[string]time.Time
	sleep  func(context.Context, time.Duration) error
}

// New returns a client with the default values.
func New(s *store.Store) *Client {
	return &Client{Store: s, HTTP: &http.Client{Timeout: 10 * time.Minute}, Agent: Agent, Delay: time.Second, Retries: 4, MaxWait: 2 * time.Minute}
}

func (c *Client) agent() string {
	if c.Agent == "" {
		return Agent
	}
	return c.Agent
}

func (c *Client) wait(ctx context.Context, d time.Duration) error {
	if c.sleep != nil {
		return c.sleep(ctx, d)
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// rules returns the robots.txt rules of the host of u. A robots.txt that does not exist allows all. A robots.txt that the client cannot get also allows all, but a 401 or 403 response blocks all, as RFC 9309 tells.
func (c *Client) rules(ctx context.Context, u *url.URL) (*robots, error) {
	key := u.Scheme + "://" + u.Host
	c.mu.Lock()
	r, ok := c.robots[key]
	c.mu.Unlock()
	if ok {
		return r, nil
	}
	if err := c.pace(ctx, u.Host, 0); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, key+"/robots.txt", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.agent())
	resp, err := c.HTTP.Do(req)
	switch {
	case err != nil:
		r = &robots{}
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		r = &robots{disallow: []string{"/"}}
	case resp.StatusCode/100 == 2:
		r = parseRobots(io.LimitReader(resp.Body, 500<<10), c.agent())
	default:
		r = &robots{}
	}
	if resp != nil {
		_ = resp.Body.Close()
	}
	c.mu.Lock()
	if c.robots == nil {
		c.robots = map[string]*robots{}
	}
	c.robots[key] = r
	c.mu.Unlock()
	return r, nil
}

// pace waits until the client can send the next request to host.
func (c *Client) pace(ctx context.Context, host string, delay time.Duration) error {
	delay = max(delay, c.Delay)
	c.mu.Lock()
	if c.next == nil {
		c.next = map[string]time.Time{}
	}
	now := time.Now()
	at := c.next[host]
	if at.Before(now) {
		at = now
	}
	c.next[host] = at.Add(delay)
	c.mu.Unlock()
	if d := at.Sub(now); d > 0 {
		return c.wait(ctx, d)
	}
	return nil
}

// Result is the outcome of a Get.
type Result struct {
	Capture store.Capture
	// Changed is false when the server told that the earlier capture is still current.
	Changed bool
}

// Options change one Get.
type Options struct {
	// Hash is the expected SHA-256 hash of the body. When it is set and the store has the bytes, Get sends no request.
	Hash string
	// Remote sends the body to the remote store and not to the local disk. It needs Hash.
	Remote bool
	// Licence goes into the capture.
	Licence string
	// Fresh sends a request with no condition.
	Fresh bool
}

// Get gets u for source and puts the body in the store.
func (c *Client) Get(ctx context.Context, source, u string, opt Options) (Result, error) {
	pu, err := url.Parse(u)
	if err != nil {
		return Result{}, err
	}
	prev, have, err := c.Store.Latest(source, u)
	if err != nil {
		return Result{}, err
	}
	if opt.Hash != "" && have && prev.Hash == opt.Hash && (prev.Local || prev.Remote) {
		return Result{Capture: prev}, nil
	}
	rules, err := c.rules(ctx, pu)
	if err != nil {
		return Result{}, err
	}
	if !rules.allowed(pu.EscapedPath()) {
		return Result{}, fmt.Errorf("%s: %w", u, ErrRobots)
	}
	backoff := 2 * time.Second
	for attempt := 0; ; attempt++ {
		if err := c.pace(ctx, pu.Host, rules.delay); err != nil {
			return Result{}, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return Result{}, err
		}
		req.Header.Set("User-Agent", c.agent())
		if have && !opt.Fresh {
			if prev.ETag != "" {
				req.Header.Set("If-None-Match", prev.ETag)
			}
			if prev.LastModified != "" {
				req.Header.Set("If-Modified-Since", prev.LastModified)
			}
		}
		resp, err := c.HTTP.Do(req)
		retry := err != nil
		var wait time.Duration
		if err == nil {
			switch {
			case resp.StatusCode == http.StatusNotModified && have:
				_ = resp.Body.Close()
				return Result{Capture: prev}, nil
			case resp.StatusCode/100 == 2:
				res, err := c.save(ctx, resp, source, u, opt)
				_ = resp.Body.Close()
				return res, err
			case resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode/100 == 5:
				retry = true
				wait = retryAfter(resp.Header.Get("Retry-After"))
				err = fmt.Errorf("%s: %s", u, resp.Status)
			default:
				err = fmt.Errorf("%s: %s", u, resp.Status)
			}
			_ = resp.Body.Close()
		}
		if !retry || attempt >= c.Retries {
			return Result{}, err
		}
		if wait == 0 {
			wait = backoff + rand.N(backoff/2) // #nosec G404 -- the jitter of a retry needs no strong random numbers.
			backoff *= 2
		}
		if c.MaxWait > 0 {
			wait = min(wait, c.MaxWait)
		}
		if err := c.wait(ctx, wait); err != nil {
			return Result{}, err
		}
	}
}

func (c *Client) save(ctx context.Context, resp *http.Response, source, u string, opt Options) (Result, error) {
	cp := store.Capture{
		Hash:         opt.Hash,
		Source:       source,
		URL:          u,
		Type:         resp.Header.Get("Content-Type"),
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
		Licence:      opt.Licence,
		Fetched:      time.Now().UTC(),
	}
	var err error
	if opt.Remote {
		cp, err = c.Store.PutRemote(ctx, resp.Body, resp.ContentLength, cp)
	} else {
		cp, err = c.Store.Put(resp.Body, cp)
	}
	if err != nil {
		return Result{}, fmt.Errorf("%s: %w", u, err)
	}
	return Result{Capture: cp, Changed: true}, nil
}

func retryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if n, err := strconv.Atoi(v); err == nil && n >= 0 {
		return time.Duration(n) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		return max(time.Until(t), 0)
	}
	return 0
}
