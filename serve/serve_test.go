package serve

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

func load(t *testing.T) *content.Corpus {
	t.Helper()
	corpus, fs := content.Load("../content/testdata/valid")
	if len(fs) != 0 {
		t.Fatal(fs)
	}
	return corpus
}

func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code, w.Body.String()
}

// TestSameAsSite builds the site and wants om serve and the MCP tools to give the bytes of each static API file.
func TestSameAsSite(t *testing.T) {
	corpus := load(t)
	out := t.TempDir()
	if _, err := site.Build(corpus, site.Options{Out: out, URL: "https://example.org", Date: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	s := New(corpus, "https://example.org", "test")
	h := s.Handler()
	n := 0
	err := filepath.WalkDir(filepath.Join(out, "api"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		want, err := os.ReadFile(p) // #nosec G304 -- the path comes from the walk.
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(out, p)
		path := "/" + filepath.ToSlash(rel)
		for _, u := range []string{path, strings.TrimSuffix(path, ".json")} {
			if code, got := get(t, h, u); code != http.StatusOK || got != string(want) {
				t.Errorf("GET %s: %d\n%s\nwant\n%s", u, code, got, want)
			}
		}
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 6 {
		t.Fatalf("only %d API files", n)
	}
	// The MCP tools give the same bytes as the static files.
	tool := func(name, args string) string {
		v, err := s.Call(name, json.RawMessage(args))
		if err != nil {
			t.Fatal(err)
		}
		b, err := site.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	for _, p := range corpus.All() {
		file := func(dir string) string {
			b, _ := os.ReadFile(filepath.Join(out, "api", dir, p.Tag+".json")) // #nosec G304 -- the path is in the test build.
			return string(b)
		}
		if got := tool("get_problem", `{"tag":"`+p.Tag+`"}`); got != file("problem") {
			t.Errorf("get_problem %s differs from the static file", p.Tag)
		}
		if want := file("solutions"); want != "" && tool("get_solutions", `{"tag":"`+p.Tag+`"}`) != want {
			t.Errorf("get_solutions %s differs from the static file", p.Tag)
		}
	}
	want, _ := os.ReadFile(filepath.Join(out, "api/edition/imo/2024.json"))
	if got := tool("list_edition", `{"competition":"imo","edition":"2024"}`); got != string(want) {
		t.Error("list_edition differs from the static file")
	}
}

func TestSearch(t *testing.T) {
	s := New(load(t), "https://example.org", "test")
	if r := s.Search("imo/2024/p2", 0); r.Total == 0 || r.Results[0].Locator != "imo/2024/p2" {
		t.Errorf("locator search: %+v", r)
	}
	if r := s.Search("0hvhsdnnmy", 0); r.Total == 0 || r.Results[0].Tag != "0HVHSDNNMY" {
		t.Errorf("tag search: %+v", r)
	}
	if r := s.Search("no-such-word-anywhere", 0); r.Total != 0 || r.Results == nil {
		t.Errorf("empty search: %+v", r)
	}
	if r := s.Search("imo 2024", 1); r.Total < 2 || len(r.Results) != 1 {
		t.Errorf("limit: %+v", r)
	}
	h := s.Handler()
	if code, body := get(t, h, "/api/search?q=imo+2024"); code != http.StatusOK || !strings.Contains(body, `"total":2`) {
		t.Errorf("GET /api/search: %d %s", code, body)
	}
	if code, body := get(t, h, "/api/problem/0NOSUCHTAG"); code != http.StatusNotFound || !strings.Contains(body, "error") {
		t.Errorf("missing problem: %d %s", code, body)
	}
}

func TestRandom(t *testing.T) {
	s := New(load(t), "https://example.org", "test")
	a, ok := s.Random(Filter{}, 7)
	b, _ := s.Random(Filter{}, 7)
	if !ok || a.Tag != b.Tag {
		t.Errorf("a seed gives two problems: %s %s", a.Tag, b.Tag)
	}
	if p, ok := s.Random(Filter{Competition: "imo", YearFrom: 2024, YearTo: 2024}, 0); !ok || p.Competition != "imo" {
		t.Errorf("filter: %+v", p)
	}
	if _, ok := s.Random(Filter{Competition: "imo", YearFrom: 1990, YearTo: 1991}, 0); ok {
		t.Error("an empty filter gave a problem")
	}
	if !(Filter{Topic: "N"}).Match(&content.Problem{Topics: []string{"N.floor"}}) || (Filter{Topic: "N"}).Match(&content.Problem{Topics: []string{"NT"}}) {
		t.Error("the topic filter is wrong")
	}
	h := s.Handler()
	if code, body := get(t, h, "/api/random?competition=imo&seed=3"); code != http.StatusOK || strings.Contains(body, `"answer":`) {
		t.Errorf("GET /api/random: %d %s", code, body)
	}
	if code, _ := get(t, h, "/api/random?competition=nope"); code != http.StatusNotFound {
		t.Errorf("GET /api/random with no match: %d", code)
	}
}

func TestMCP(t *testing.T) {
	s := New(load(t), "https://example.org", "test")
	in := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"get_problem","arguments":{"tag":"0HVHSDNNMY"}}}`,
		`{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"get_problem","arguments":{"tag":"0NOSUCHTAG"}}}`,
		`{"jsonrpc":"2.0","id":5,"method":"no/such"}`,
		`not json`,
	}, "\n")
	var out bytes.Buffer
	if err := s.ServeMCP(strings.NewReader(in), &out); err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 6 {
		t.Fatalf("want 6 answers, got %d:\n%s", len(lines), out.String())
	}
	for i, want := range []string{`"protocolVersion":"2025-03-26"`, `"name":"random_problem"`, `"structuredContent":{"tag":"0HVHSDNNMY"`, `"isError":true`, `-32601`, `-32700`} {
		if !strings.Contains(lines[i], want) {
			t.Errorf("answer %d has no %s: %s", i+1, want, lines[i])
		}
	}
	if strings.Contains(lines[2], "All even integers") {
		t.Error("get_problem gave the answer")
	}
	// The same server answers at /mcp over HTTP.
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"get_solutions","arguments":{"tag":"0HVHSDNNMY"}}}`)))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "All even integers") {
		t.Errorf("POST /mcp: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)))
	if w.Code != http.StatusAccepted {
		t.Errorf("a notification over HTTP: %d", w.Code)
	}
}
