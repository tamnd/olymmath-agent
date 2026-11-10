package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
)

// TestBuildFigures builds a copy of the test content where imo/2024/p2 has a figure next to it.
func TestBuildFigures(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../content/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "problems", "imo", "2024", "p2.md")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	b, err = content.AddToStatement(b, "S1", "S1", "![Figure 1](p2.fig1.svg)")
	if err != nil {
		t.Fatal(err)
	}
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="10pt" height="10pt" viewBox="0 0 10 10"><path d="M0 0L10 10"/></svg>`
	for name, data := range map[string]string{"p2.md": string(b), "p2.fig1.svg": svg} {
		if err := os.WriteFile(filepath.Join(filepath.Dir(file), name), []byte(data), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	corpus, fs := content.Load(root)
	if len(fs) != 0 {
		t.Fatal(fs)
	}
	out := t.TempDir()
	stats, err := Build(corpus, Options{Out: out, URL: "https://example.org", Base: "/", Date: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Errors) != 0 {
		t.Fatal(stats.Errors)
	}
	if got := read(t, out, "imo/2024/p2.fig1.svg"); got != svg {
		t.Errorf("the figure in the site is %q", got)
	}
	// The page of the problem and the page of its edition both link to the figure by its full path.
	for _, rel := range []string{"imo/2024/p2.html", "imo/2024/index.html"} {
		if page := read(t, out, rel); !strings.Contains(page, `<img src="/imo/2024/p2.fig1.svg" alt="Figure 1"`) {
			t.Errorf("%s does not show the figure", rel)
		}
	}
	if !strings.Contains(read(t, out, "imo/2024/p2.json"), `"https://example.org/imo/2024/p2.fig1.svg"`) {
		t.Error("the JSON copy has no URL of the figure")
	}
	if !strings.Contains(read(t, out, "imo/2024/p2.md"), "![Figure 1](p2.fig1.svg)") {
		t.Error("the Markdown copy does not show the figure next to it")
	}
	if strings.Contains(read(t, out, "search.json"), "fig1") {
		t.Error("the search text has the image")
	}
}

func TestBuildFigureWithNoFile(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../content/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "problems", "imo", "2024", "p2.md")
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if b, err = content.AddToStatement(b, "S1", "S1", "![Figure 1](p2.fig1.svg)"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, b, 0o600); err != nil {
		t.Fatal(err)
	}
	corpus, _ := content.Load(root)
	stats, err := Build(corpus, Options{Out: t.TempDir(), URL: "https://example.org", Base: "/"})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Errors) != 1 || !strings.Contains(stats.Errors[0].Error(), "has no file") {
		t.Errorf("errors: %v", stats.Errors)
	}
}
