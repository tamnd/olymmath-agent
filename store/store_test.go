package store

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type memRemote struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func (m *memRemote) Put(_ context.Context, key string, r io.Reader, _ int64) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.objs == nil {
		m.objs = map[string][]byte{}
	}
	m.objs[key] = b
	return nil
}

func (m *memRemote) Get(_ context.Context, key string) (io.ReadCloser, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	b, ok := m.objs[key]
	if !ok {
		return nil, ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *memRemote) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objs, key)
	return nil
}

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

func read(t *testing.T, s *Store, hash string) string {
	t.Helper()
	rc, err := s.Get(context.Background(), hash)
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	return string(b)
}

func TestPutAndGet(t *testing.T) {
	s, err := Open(t.TempDir(), nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := s.PutBytes([]byte("hello"), Capture{Source: "hf:a/b", URL: "https://example.org/x"})
	if err != nil {
		t.Fatal(err)
	}
	if c.Hash != sum("hello") || c.Size != 5 || !c.Local {
		t.Fatalf("got %+v", c)
	}
	if got := read(t, s, c.Hash); got != "hello" {
		t.Fatalf("got %q", got)
	}
	if _, err := s.PutBytes([]byte("hello"), Capture{Source: "hf:a/b", URL: "https://example.org/x", ETag: `"2"`}); err != nil {
		t.Fatal(err)
	}
	cs, _ := s.Captures("hf:a/b")
	if len(cs) != 2 {
		t.Fatalf("want 2 captures, got %d", len(cs))
	}
	l, ok, _ := s.Latest("hf:a/b", "https://example.org/x")
	if !ok || l.ETag != `"2"` {
		t.Fatalf("latest: %+v", l)
	}
	src, _ := s.Sources()
	if len(src) != 1 || src[0] != "hf:a/b" {
		t.Fatalf("sources: %v", src)
	}
	if _, err := s.PutBytes([]byte("x"), Capture{Source: "s", Hash: sum("y")}); err == nil {
		t.Error("want a hash mismatch")
	}
	if _, err := s.Get(context.Background(), sum("nothing")); err == nil {
		t.Error("want an error for a missing hash")
	}
}

func TestRemote(t *testing.T) {
	m := &memRemote{}
	s, _ := Open(t.TempDir(), m)
	c, err := s.PutRemote(context.Background(), strings.NewReader("big file"), 8, Capture{Source: "hf:a/b", Hash: sum("big file")})
	if err != nil {
		t.Fatal(err)
	}
	if !c.Remote || c.Local || s.HasLocal(c.Hash) {
		t.Fatalf("got %+v", c)
	}
	if got := read(t, s, c.Hash); got != "big file" {
		t.Fatalf("got %q", got)
	}
	if _, err := s.PutRemote(context.Background(), strings.NewReader("other"), 5, Capture{Source: "s", Hash: sum("big file2")}); err == nil {
		t.Fatal("want a hash mismatch")
	}
	if _, ok := m.objs[Key(sum("big file2"))]; ok {
		t.Error("the bad object is still in the remote")
	}
	l, _ := s.PutBytes([]byte("local"), Capture{Source: "s"})
	if err := s.Push(context.Background(), l.Hash); err != nil {
		t.Fatal(err)
	}
	if err := s.Evict(l.Hash); err != nil {
		t.Fatal(err)
	}
	if got := read(t, s, l.Hash); got != "local" {
		t.Fatalf("after evict: %q", got)
	}
}

func TestR2(t *testing.T) {
	objs := map[string]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer tok" {
			w.WriteHeader(http.StatusForbidden)
			io.WriteString(w, `{"success":false,"errors":[{"code":10000,"message":"Authentication error"}]}`)
			return
		}
		switch r.Method {
		case http.MethodPut:
			b, _ := io.ReadAll(r.Body)
			objs[r.URL.Path] = string(b)
			io.WriteString(w, `{"success":true}`)
		case http.MethodGet:
			b, ok := objs[r.URL.Path]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			io.WriteString(w, b)
		case http.MethodDelete:
			delete(objs, r.URL.Path)
		}
	}))
	defer srv.Close()
	r := &R2{Account: "acct", Token: "tok", Bucket: Bucket, Base: srv.URL}
	ctx := context.Background()
	if err := r.Put(ctx, "sha256/ab/abc", strings.NewReader("data"), 4); err != nil {
		t.Fatal(err)
	}
	if _, ok := objs["/accounts/acct/r2/buckets/olymmath-raw/objects/sha256/ab/abc"]; !ok {
		t.Fatalf("objects: %v", objs)
	}
	rc, err := r.Get(ctx, "sha256/ab/abc")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(rc)
	rc.Close()
	if string(b) != "data" {
		t.Fatalf("got %q", b)
	}
	if err := r.Delete(ctx, "sha256/ab/abc"); err != nil {
		t.Fatal(err)
	}
	if err := r.Delete(ctx, "sha256/ab/abc"); err != nil {
		t.Fatal(err)
	}
	r.Token = "bad"
	if err := r.Put(ctx, "k", strings.NewReader("x"), 1); err == nil || !strings.Contains(err.Error(), "Authentication error") {
		t.Fatalf("want the API error, got %v", err)
	}
}

func TestValidHash(t *testing.T) {
	if !ValidHash(sum("a")) || ValidHash("abc") || ValidHash(strings.ToUpper(sum("a"))) {
		t.Error("ValidHash")
	}
}

func TestCursor(t *testing.T) {
	s, _ := Open(t.TempDir(), nil)
	if c, err := s.Cursor("aops:c13"); c != "" || err != nil {
		t.Fatalf("empty: %q %v", c, err)
	}
	if err := s.SetCursor("aops:c13", "page 4"); err != nil {
		t.Fatal(err)
	}
	if c, _ := s.Cursor("aops:c13"); c != "page 4" {
		t.Fatalf("got %q", c)
	}
}
