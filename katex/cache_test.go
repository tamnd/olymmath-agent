package katex

import (
	"compress/gzip"
	"encoding/gob"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// TestTokenRegexPatch wants the patch of the lexer to find its place in the script. A new release of KaTeX may change it, and then each render compiles the expression again.
func TestTokenRegexPatch(t *testing.T) {
	if strings.Count(script, lexerNew) != 1 {
		t.Fatalf("the script has %d copies of %s, want 1", strings.Count(script, lexerNew), lexerNew)
	}
}

// TestCache saves the formulas, loads them in a new Renderer, and wants the same HTML and errors with no render.
func TestCache(t *testing.T) {
	dir := t.TempDir()
	r, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	formulas := []string{`x^2`, `\frac{a}{b}`, `\nosuchmacro`, `\sqrt{2}`}
	want := map[string]string{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	for _, f := range formulas {
		wg.Go(func() {
			html, err := r.Render(f, true)
			if err != nil {
				html = "error: " + err.Error()
			}
			mu.Lock()
			want[f] = html
			mu.Unlock()
		})
	}
	wg.Wait()
	if err := r.SaveCache(dir); err != nil {
		t.Fatal(err)
	}
	r2, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	r2.LoadCache(dir)
	if len(r2.cache) != len(formulas) {
		t.Fatalf("the cache has %d formulas, want %d", len(r2.cache), len(formulas))
	}
	// The engine is gone, so a formula that is not in the cache would fail.
	r2.idle, r2.slots = make(chan *engine), make(chan struct{})
	for _, f := range formulas {
		html, err := r2.Render(f, true)
		if err != nil {
			var e *Error
			if !errors.As(err, &e) || e.TeX != f {
				t.Errorf("%s gives the error %v", f, err)
			}
			html = "error: " + err.Error()
		}
		if html != want[f] {
			t.Errorf("%s gives %q from the cache, want %q", f, html, want[f])
		}
	}
}

// TestCacheOldFormat loads a cache file of an older om, with one gob value for all the entries, and wants its formulas.
func TestCacheOldFormat(t *testing.T) {
	r, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, r.name))
	if err != nil {
		t.Fatal(err)
	}
	z := gzip.NewWriter(f)
	entries := []cacheEntry{{TeX: "x", HTML: "<x>"}, {TeX: "y", Display: true, HTML: "<y>"}, {TeX: `\bad`, Error: "no", Failed: true}}
	if err := gob.NewEncoder(z).Encode(entries); err != nil {
		t.Fatal(err)
	}
	if err := z.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	r.LoadCache(dir)
	if len(r.cache) != len(entries) {
		t.Fatalf("the cache has %d formulas, want %d", len(r.cache), len(entries))
	}
	if html, err := r.Render("y", true); err != nil || html != "<y>" {
		t.Errorf("y gives %q, %v", html, err)
	}
	if _, err := r.Render(`\bad`, false); err == nil {
		t.Error("the failed formula gives no error")
	}
	// A save writes the new format, and a load of it gives the same formulas.
	if err := r.SaveCache(dir); err != nil {
		t.Fatal(err)
	}
	r2, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	r2.LoadCache(dir)
	if len(r2.cache) != 2 {
		t.Errorf("the saved cache has %d formulas, want the 2 that a render asked for", len(r2.cache))
	}
}

// TestCacheCompressed checks that the cache keeps the HTML compressed and gives back the same HTML as the first render.
func TestCacheCompressed(t *testing.T) {
	r, err := New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.dict) == 0 {
		t.Fatal("the compressor has no dictionary")
	}
	for _, tex := range []string{"x", `\frac{1}{2}`, `\sum_{k=1}^{n} k^2 = \frac{n(n+1)(2n+1)}{6}`, `\begin{pmatrix}a&b\\c&d\end{pmatrix}`} {
		for _, display := range []bool{false, true} {
			first, err := r.Render(tex, display)
			if err != nil {
				t.Fatal(err)
			}
			again, err := r.Render(tex, display)
			if err != nil || again != first {
				t.Errorf("%q from the cache gives %q, %v, want %q", tex, again, err, first)
			}
			res := r.cache[key{tex, display}]
			if res.size != len(first) || len(res.html) >= len(first) {
				t.Errorf("%q keeps %d bytes for %d bytes of HTML (size %d)", tex, len(res.html), len(first), res.size)
			}
		}
	}
}
