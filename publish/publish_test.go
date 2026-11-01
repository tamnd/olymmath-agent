package publish

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
)

// fakeR2 keeps the objects of one bucket in memory.
type fakeR2 struct {
	mu      sync.Mutex
	objects map[string]string
	types   map[string]string
	puts    int
	// fail is a key that the fake refuses to write.
	fail string
}

func (f *fakeR2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if r.Header.Get("Authorization") != "Bearer token" {
		http.Error(w, "no", http.StatusForbidden)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/accounts/acc/r2/buckets/site/objects/")
	switch r.Method {
	case http.MethodPut:
		if key == f.fail {
			http.Error(w, "no", http.StatusForbidden)
			return
		}
		b, _ := io.ReadAll(r.Body)
		f.objects[key] = string(b)
		f.types[key] = r.Header.Get("Content-Type")
		f.puts++
	case http.MethodGet:
		b, ok := f.objects[key]
		if !ok {
			http.NotFound(w, r)
			return
		}
		_, _ = io.WriteString(w, b)
	case http.MethodDelete:
		delete(f.objects, key)
	}
}

func write(t *testing.T, dir, rel, text string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSync(t *testing.T) {
	f := &fakeR2{objects: map[string]string{}, types: map[string]string{}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	r := &R2{Account: "acc", Bucket: "site", Token: "token", Base: srv.URL, Rate: -1}
	dir := t.TempDir()
	write(t, dir, "nz/2020/p1.html", "one")
	write(t, dir, "nz/2020/p2.md", "two")
	ctx := context.Background()
	res, err := Sync(ctx, r, dir, 4)
	if err != nil || res.Put != 2 || res.Files != 2 {
		t.Fatalf("%+v %v", res, err)
	}
	if f.objects["nz/2020/p1.html"] != "one" || f.types["nz/2020/p2.md"] != "text/markdown; charset=utf-8" || f.objects[ManifestKey] == "" {
		t.Errorf("%v %v", f.objects, f.types)
	}
	write(t, dir, "nz/2020/p1.html", "new")
	if err := os.Remove(filepath.Join(dir, "nz", "2020", "p2.md")); err != nil {
		t.Fatal(err)
	}
	res, err = Sync(ctx, r, dir, 4)
	if err != nil || res.Put != 1 || res.Deleted != 1 {
		t.Fatalf("%+v %v", res, err)
	}
	if _, ok := f.objects["nz/2020/p2.md"]; ok || f.objects["nz/2020/p1.html"] != "new" {
		t.Errorf("%v", f.objects)
	}
	if res, _ = Sync(ctx, r, dir, 4); res.Put != 0 || res.Deleted != 0 {
		t.Errorf("a sync with no change did %+v", res)
	}
}

func TestSyncUntil(t *testing.T) {
	f := &fakeR2{objects: map[string]string{}, types: map[string]string{}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	r := &R2{Account: "acc", Bucket: "site", Token: "token", Base: srv.URL, Rate: -1}
	dir := t.TempDir()
	write(t, dir, "a.html", "a")
	write(t, dir, "b.html", "b")
	ctx := context.Background()
	// A time in the past sends nothing and leaves both files for the next run.
	res, err := SyncUntil(ctx, r, dir, 1, time.Now().Add(-time.Second))
	if err != nil || res.Put != 0 || res.Left != 2 {
		t.Fatalf("%+v %v", res, err)
	}
	if res, err = Sync(ctx, r, dir, 1); err != nil || res.Put != 2 || res.Left != 0 {
		t.Fatalf("the next run did %+v %v", res, err)
	}
}

func TestSplit(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	write(t, dir, "redirects.json", `{"/p/ABC":"/imo/2024/p1","/isl/2024/n1":"/imo/2024/p1"}`)
	write(t, dir, "p/ABC.html", "moved")
	write(t, dir, "isl/2024/n1.html", "moved")
	write(t, dir, "imo/2024/p1.html", "imo")
	write(t, dir, "imo/2024/p1.md", "imo")
	write(t, dir, "imo/2024/index.json", "{}")
	write(t, dir, "p/XYZ.json", "{}")
	write(t, dir, "nz/index.html", "nz")
	write(t, dir, "nz/2020/index.html", "edition")
	write(t, dir, "nz/2020/p1.html", "problem")
	write(t, dir, "nz/2020/p1.json", "{}")
	corpus := &content.Corpus{Competitions: []*content.Competition{
		{Code: "imo", Group: "international"},
		{Code: "isl", Group: "international"},
		{Code: "nz", Group: "national", Country: "NZ"},
	}}
	tiers := Tiers{Groups: []string{"international"}, Countries: []string{"US"}}
	deleted, moved, err := Split(dir, out, corpus, tiers)
	if err != nil || deleted != 2 || moved != 4 {
		t.Fatalf("%d %d %v", deleted, moved, err)
	}
	for _, rel := range []string{"imo/2024/p1.html", "imo/2024/index.json", "nz/index.html", "nz/2020/index.html"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
	for _, rel := range []string{"nz/2020/p1.html", "nz/2020/p1.json", "imo/2024/p1.md", "p/XYZ.json"} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("%s did not move: %v", rel, err)
		}
	}
}

func TestFit(t *testing.T) {
	dir, out := t.TempDir(), t.TempDir()
	for _, rel := range []string{"index.html", "imo/index.html", "imo/2024/p1.html", "imo/2024/p2.html", "imo/2024/p3.html", "aime/index.html", "aime/2024/p1.html"} {
		write(t, dir, rel, "x")
	}
	corpus := &content.Corpus{Competitions: []*content.Competition{{Code: "aime"}, {Code: "imo"}}}
	if n, codes, err := Fit(dir, out, corpus, 10); n != 0 || codes != nil || err != nil {
		t.Fatalf("a site under the limit changed: %d %v %v", n, codes, err)
	}
	n, codes, err := Fit(dir, out, corpus, 5)
	if err != nil || n != 3 || !slices.Equal(codes, []string{"imo"}) {
		t.Fatalf("%d %v %v", n, codes, err)
	}
	for _, rel := range []string{"imo/index.html", "aime/2024/p1.html"} {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Errorf("%s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(out, "imo/2024/p2.html")); err != nil {
		t.Errorf("imo/2024/p2.html did not move: %v", err)
	}
	if _, _, err := Fit(dir, out, corpus, 1); err == nil {
		t.Error("a site over the limit after the move gave no error")
	}
}

func TestLoadTiers(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "t.yaml", "groups: [international]\ncountries: [RU]\ncodes: [aime]\n")
	tiers, err := LoadTiers(filepath.Join(dir, "t.yaml"))
	if err != nil || !tiers.First(&content.Competition{Code: "ru-spb", Country: "RU"}) || tiers.First(&content.Competition{Code: "nz-nzmo", Country: "NZ"}) || !tiers.First(&content.Competition{Code: "aime"}) {
		t.Errorf("%+v %v", tiers, err)
	}
	write(t, dir, "bad.yaml", "group: [x]\n")
	if _, err := LoadTiers(filepath.Join(dir, "bad.yaml")); err == nil {
		t.Error("an unknown key was accepted")
	}
}

func TestSyncResume(t *testing.T) {
	f := &fakeR2{objects: map[string]string{}, types: map[string]string{}, fail: "b.html"}
	srv := httptest.NewServer(f)
	defer srv.Close()
	r := &R2{Account: "acc", Bucket: "site", Token: "token", Base: srv.URL, Rate: -1}
	dir := t.TempDir()
	for _, n := range []string{"a.html", "b.html", "c.html"} {
		if err := os.WriteFile(filepath.Join(dir, n), []byte(n), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	ctx := context.Background()
	if _, err := Sync(ctx, r, dir, 1); err == nil {
		t.Fatal("no error for a refused put")
	}
	f.fail = ""
	res, err := Sync(ctx, r, dir, 1)
	if err != nil || res.Put != 1 {
		t.Errorf("second sync put %d files, error %v; want 1", res.Put, err)
	}
}

func TestPack(t *testing.T) {
	in, out, index := t.TempDir(), t.TempDir(), t.TempDir()
	write(t, in, "nz/2020/p1.html", "one")
	write(t, in, "nz/2020/p1.md", "two")
	write(t, in, "ca/2021/p2.html", "three")
	res, err := Pack(in, out, index)
	if err != nil || res.Files != 3 || res.Packs != 2 {
		t.Fatalf("%+v %v", res, err)
	}
	b, err := os.ReadFile(filepath.Join(index, "nz.json"))
	if err != nil {
		t.Fatal(err)
	}
	var ix PackIndex
	if err := json.Unmarshal(b, &ix); err != nil {
		t.Fatal(err)
	}
	pack, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(ix.Pack)))
	if err != nil {
		t.Fatal(err)
	}
	for f, want := range map[string]string{"2020/p1.html": "one", "2020/p1.md": "two"} {
		e := ix.Files[f]
		if got := string(pack[e.Offset : e.Offset+e.Length]); got != want || len(e.Hash) != 16 {
			t.Errorf("%s gives %q and hash %q", f, got, e.Hash)
		}
	}
	// The same files give the same pack, and a change gives a new pack name.
	first := ix.Pack
	if _, err := Pack(in, t.TempDir(), index); err != nil {
		t.Fatal(err)
	}
	if b2, _ := os.ReadFile(filepath.Join(index, "nz.json")); string(b2) != string(b) {
		t.Error("the same files give another index")
	}
	write(t, in, "nz/2020/p1.md", "new")
	if _, err := Pack(in, t.TempDir(), index); err != nil {
		t.Fatal(err)
	}
	b, _ = os.ReadFile(filepath.Join(index, "nz.json"))
	if err := json.Unmarshal(b, &ix); err != nil || ix.Pack == first {
		t.Errorf("a changed file keeps the pack %s", ix.Pack)
	}
	write(t, in, "top.html", "x")
	if _, err := Pack(in, t.TempDir(), t.TempDir()); err == nil {
		t.Error("a file that is not in a folder is packed")
	}
}

func TestSyncKeep(t *testing.T) {
	f := &fakeR2{objects: map[string]string{}, types: map[string]string{}}
	srv := httptest.NewServer(f)
	defer srv.Close()
	r := &R2{Account: "acc", Bucket: "site", Token: "token", Base: srv.URL, Rate: -1}
	dir := t.TempDir()
	write(t, dir, "pack/a.1.pack", "a")
	ctx := context.Background()
	if _, err := Sync(ctx, r, dir, 1); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "pack", "a.1.pack")); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "pack/a.2.pack", "b")
	res, err := SyncWith(ctx, r, dir, SyncOptions{Workers: 1, Keep: true})
	if err != nil || res.Put != 1 || res.Deleted != 0 || f.objects["pack/a.1.pack"] != "a" {
		t.Fatalf("a sync with keep did %+v %v", res, err)
	}
	// The next sync with no keep deletes the old pack, because the manifest still lists it.
	if res, err = Sync(ctx, r, dir, 1); err != nil || res.Put != 0 || res.Deleted != 1 {
		t.Fatalf("the sync after keep did %+v %v", res, err)
	}
}
