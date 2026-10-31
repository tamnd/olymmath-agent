package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
)

func build(t *testing.T, base string) string {
	t.Helper()
	corpus, fs := content.Load("../content/testdata/valid")
	if len(fs) != 0 {
		t.Fatal(fs)
	}
	out := t.TempDir()
	stats, err := Build(corpus, Options{Out: out, URL: "https://example.org", Base: base, Commit: "0123456789abcdef", Version: "test", Date: time.Date(2026, 10, 31, 0, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	if len(stats.Errors) != 0 || stats.Problems != 3 || stats.Redirects != 3 {
		t.Fatalf("stats: %+v", stats)
	}
	return out
}

func read(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestBuildPages(t *testing.T) {
	out := build(t, "/")
	for _, rel := range []string{
		"index.html", "c/index.html", "c/imo/index.html", "imo/2024/index.html", "imo/2024/p1.html", "imo/2024/p2.html",
		"about/index.html", "sources/index.html", "search/index.html", "404.html",
		"p/0J8TP3MFS9.html", "p/0HVHSDNNMY.html", "imo/2024/q1.html",
		"search.json", "sitemap.xml", "robots.txt", "llms.txt", "_headers",
		"assets/site.css", "assets/site.js", "assets/theme.js", "assets/favicon.svg", "assets/katex/katex.min.css",
	} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("no %s", rel)
		}
	}
	p1 := read(t, out, "imo/2024/p1.html")
	for _, want := range []string{
		`<link rel="canonical" href="https://example.org/imo/2024/p1">`,
		`class="katex"`,
		`data-next="/imo/2024/p2"`,
		"Official text",
		"Show the answer",
		"Day 1",
		"Number theory",
	} {
		if !strings.Contains(p1, want) {
			t.Errorf("imo/2024/p1.html has no %q", want)
		}
	}
	if !strings.Contains(read(t, out, "sitemap.xml"), "<loc>https://example.org/imo/2024/p1</loc>") {
		t.Error("the sitemap has no problem page")
	}
	if strings.Contains(read(t, out, "sitemap.xml"), "404") {
		t.Error("the sitemap has the 404 page")
	}
}

func TestBuildTagsAndRedirects(t *testing.T) {
	out := build(t, "/")
	loose := read(t, out, "p/0J8TP3MFS9.html")
	if !strings.Contains(loose, `<link rel="canonical" href="https://example.org/p/0J8TP3MFS9">`) || !strings.Contains(loose, "Problems with no known competition") {
		t.Error("the page of the loose problem is wrong")
	}
	for rel, want := range map[string]string{
		"p/0HVHSDNNMY.html": `content="0; url=/imo/2024/p1"`,
		"imo/2024/q1.html":  `href="https://example.org/imo/2024/p1"`,
	} {
		if !strings.Contains(read(t, out, rel), want) {
			t.Errorf("%s has no %q", rel, want)
		}
	}
	if strings.Contains(read(t, out, "sitemap.xml"), "/p/0HVHSDNNMY") {
		t.Error("the sitemap has a redirect page")
	}
	if !strings.Contains(read(t, out, "search.json"), `"u":"/p/0J8TP3MFS9"`) {
		t.Error("the search index has no loose problem")
	}
	p1 := read(t, out, "imo/2024/p1.html")
	if !strings.Contains(p1, "Human solution") || strings.Contains(p1, "No solution is on olymmath yet") {
		t.Error("the solution of imo/2024/p1 is not on its page")
	}
}

func TestCompetitionsPage(t *testing.T) {
	out := build(t, "/")
	c := read(t, out, "c/index.html")
	for _, want := range []string{`id="international"`, "1 of them has problems", "data-filter", `href="/c/imo/"`} {
		if !strings.Contains(c, want) {
			t.Errorf("c/index.html has no %q", want)
		}
	}
}

func TestYearGridAndCoverage(t *testing.T) {
	out := build(t, "/")
	c := read(t, out, "c/imo/index.html")
	for _, want := range []string{`class="cell cell-`, `href="/imo/2024/p1"`, "C0", "human solution"} {
		if !strings.Contains(c, want) {
			t.Errorf("c/imo/index.html has no %q", want)
		}
	}
	cov := read(t, out, "coverage/index.html")
	for _, want := range []string{"<h1>Coverage</h1>", `href="/c/imo/"`, "Expected positions"} {
		if !strings.Contains(cov, want) {
			t.Errorf("coverage/index.html has no %q", want)
		}
	}
}

func TestCountryName(t *testing.T) {
	if countryName("VN") != "Vietnam" || countryName("ZZ") != "ZZ" {
		t.Error("countryName is wrong")
	}
}

func TestBuildBase(t *testing.T) {
	out := build(t, "/olymmath/")
	p1 := read(t, out, "imo/2024/p1.html")
	if !strings.Contains(p1, `href="/olymmath/c/"`) || strings.Contains(p1, `href="/c/"`) {
		t.Error("the links do not have the base prefix")
	}
	if !strings.Contains(read(t, out, "search.json"), `"u":"/olymmath/imo/2024/p1"`) {
		t.Error("the search index does not have the base prefix")
	}
}

func TestBuildIsDeterministic(t *testing.T) {
	a, b := build(t, "/"), build(t, "/")
	if read(t, a, "imo/2024/p1.html") != read(t, b, "imo/2024/p1.html") {
		t.Error("two builds of the same input differ")
	}
}

func TestBadBase(t *testing.T) {
	_, err := Build(&content.Corpus{}, Options{Out: t.TempDir(), Base: "olymmath"})
	if err == nil {
		t.Fatal("want an error for a base with no slashes")
	}
}

func TestOrdinal(t *testing.T) {
	for n, want := range map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th", 11: "11th", 12: "12th", 13: "13th", 21: "21st", 65: "65th", 66: "66th", 67: "67th", 111: "111th"} {
		if got := ordinal(n); got != want {
			t.Errorf("ordinal(%d) = %s, want %s", n, got, want)
		}
	}
}
