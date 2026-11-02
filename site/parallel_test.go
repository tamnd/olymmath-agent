package site

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
)

// files reads each file under root, by its relative path.
func files(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p) // #nosec G304 -- the path comes from the walk.
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		out[rel] = b
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestParallelSameAsSerial builds the site with one worker, with many workers, and with a warm formula cache, and wants the same files each time.
func TestParallelSameAsSerial(t *testing.T) {
	corpus, fs := content.Load("../content/testdata/valid")
	if len(fs) != 0 {
		t.Fatal(fs)
	}
	cache := t.TempDir()
	run := func(workers int, cacheDir string) (map[string][]byte, Stats) {
		out := t.TempDir()
		stats, err := Build(corpus, Options{Out: out, URL: "https://example.org", Commit: "0123456789abcdef", Version: "test", Date: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC), Workers: workers, Cache: cacheDir})
		if err != nil {
			t.Fatal(err)
		}
		return files(t, out), stats
	}
	serial, st := run(1, "")
	if len(serial) == 0 || st.Files != len(serial) {
		t.Fatalf("the serial build has %d files, and its stats say %d", len(serial), st.Files)
	}
	for _, c := range []struct {
		name    string
		workers int
		cache   string
	}{
		{"8 workers", 8, ""},
		{"a cold cache", 8, cache},
		{"a warm cache", 8, cache},
	} {
		got, stats := run(c.workers, c.cache)
		if stats.Files != st.Files || stats.Pages != st.Pages || len(stats.Errors) != len(st.Errors) {
			t.Errorf("%s: the stats are %+v, want %+v", c.name, stats, st)
		}
		if len(got) != len(serial) {
			t.Errorf("%s: %d files, want %d", c.name, len(got), len(serial))
		}
		for rel, want := range serial {
			if !bytes.Equal(got[rel], want) {
				t.Errorf("%s: %s is not the same as in the serial build", c.name, rel)
			}
		}
	}
	if m, _ := filepath.Glob(filepath.Join(cache, "katex-*.gob.gz")); len(m) != 1 {
		t.Errorf("the cache has %v, want one file", m)
	}
}

// TestWriterKeepsLastWrite writes one path many times and wants the last data, as with one worker.
func TestWriterKeepsLastWrite(t *testing.T) {
	dir := t.TempDir()
	w := newWriter(4)
	p := filepath.Join(dir, "a", "b.html")
	for i := range 100 {
		if err := w.write(p, []byte{byte(i)}); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.close(); err != nil {
		t.Fatal(err)
	}
	if b, err := os.ReadFile(p); err != nil || len(b) != 1 || b[0] != 99 { // #nosec G304 -- a test file.
		t.Errorf("the file has %v, %v, want [99]", b, err)
	}
}
