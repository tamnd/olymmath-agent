package fetch

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/store"
)

func TestRobots(t *testing.T) {
	txt := `
User-agent: *
Disallow: /private
Crawl-delay: 5

User-agent: olymmath
User-agent: other
Disallow: /search
Allow: /search/about$
Disallow: /*.pdf$
Crawl-delay: 2.5
`
	r := parseRobots(strings.NewReader(txt), "olymmath/0 (+https://olymmath.tamnd.com/about/)")
	if r.delay != 2500*time.Millisecond {
		t.Errorf("delay %v", r.delay)
	}
	for path, want := range map[string]bool{
		"/":             true,
		"/private":      true,
		"/search?q=1":   false,
		"/search/about": true,
		"/a/b.pdf":      false,
		"/a/b.pdf.html": true,
	} {
		if got := r.allowed(path); got != want {
			t.Errorf("allowed(%q) = %v, want %v", path, got, want)
		}
	}
	star := parseRobots(strings.NewReader(txt), "somebot")
	if star.allowed("/private/x") || star.delay != 5*time.Second {
		t.Errorf("star group: %+v", star)
	}
	if !parseRobots(strings.NewReader(""), "x").allowed("/any") {
		t.Error("an empty robots.txt allows all")
	}
}

func newClient(t *testing.T) *Client {
	t.Helper()
	s, err := store.Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	c := New(s)
	c.Delay = 0
	c.sleep = func(context.Context, time.Duration) error { return nil }
	return c
}

func TestGet(t *testing.T) {
	var hits, fails atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/robots.txt":
			io.WriteString(w, "User-agent: *\nDisallow: /no\n")
		case "/page":
			hits.Add(1)
			if r.Header.Get("User-Agent") != Agent {
				t.Errorf("agent %q", r.Header.Get("User-Agent"))
			}
			if r.Header.Get("If-None-Match") == `"v1"` {
				w.WriteHeader(http.StatusNotModified)
				return
			}
			w.Header().Set("ETag", `"v1"`)
			io.WriteString(w, "page body")
		case "/flaky":
			if fails.Add(1) <= 2 {
				w.Header().Set("Retry-After", "1")
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
			io.WriteString(w, "ok")
		case "/gone":
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	c := newClient(t)
	ctx := context.Background()
	r1, err := c.Get(ctx, "test", srv.URL+"/page", Options{})
	if err != nil || !r1.Changed || r1.Capture.ETag != `"v1"` {
		t.Fatalf("first get: %+v %v", r1, err)
	}
	r2, err := c.Get(ctx, "test", srv.URL+"/page", Options{})
	if err != nil || r2.Changed || r2.Capture.Hash != r1.Capture.Hash {
		t.Fatalf("second get: %+v %v", r2, err)
	}
	if _, err := c.Get(ctx, "test", srv.URL+"/page", Options{Hash: r1.Capture.Hash}); err != nil || hits.Load() != 2 {
		t.Fatalf("a known hash must not send a request: %d hits, %v", hits.Load(), err)
	}
	if _, err := c.Get(ctx, "test", srv.URL+"/no/x", Options{}); !errors.Is(err, ErrRobots) {
		t.Errorf("want ErrRobots, got %v", err)
	}
	if r, err := c.Get(ctx, "test", srv.URL+"/flaky", Options{}); err != nil || fails.Load() != 3 {
		t.Errorf("flaky: %+v %v", r, err)
	}
	if _, err := c.Get(ctx, "test", srv.URL+"/gone", Options{}); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("gone: %v", err)
	}
}

func TestRetriesEnd(t *testing.T) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/robots.txt" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		n.Add(1)
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	c := newClient(t)
	c.Retries = 3
	if _, err := c.Get(context.Background(), "test", srv.URL+"/x", Options{}); err == nil {
		t.Fatal("want an error")
	}
	if n.Load() != 4 {
		t.Errorf("want 4 requests, got %d", n.Load())
	}
}

func TestForbiddenRobots(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	if _, err := newClient(t).Get(context.Background(), "test", srv.URL+"/x", Options{}); !errors.Is(err, ErrRobots) {
		t.Errorf("want ErrRobots, got %v", err)
	}
}

func TestRetryAfter(t *testing.T) {
	if retryAfter("7") != 7*time.Second || retryAfter("") != 0 || retryAfter("soon") != 0 {
		t.Error("retryAfter")
	}
}
