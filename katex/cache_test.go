package katex

import (
	"errors"
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
