package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/official"
)

const linkOnly = `---
tag: DAWMSWVRAX
kind: problem
appearances:
  - at: ru-moscow/2003/p83
    primary: true
canonical:
  statement: S1
  rule: curated
statements:
  - id: S1
    lang: ru
    source: problems-ru
    name: problems.ru, problem 105146
    url: https://problems.ru/view_problem_details_new.php?id=105146
    origin: H2
    zone: Z3
    kind: transcription
status:
  statement: sourced
---
`

func TestFillOne(t *testing.T) {
	root := t.TempDir()
	rel := "problems/ru-moscow/2003/p83.md"
	full := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(linkOnly), 0o600); err != nil {
		t.Fatal(err)
	}
	p := &content.Problem{Path: rel, Statements: []content.Statement{{ID: "S1", Source: "problems-ru", URL: "https://problems.ru/view_problem_details_new.php?id=105146"}}}
	comp := &content.Competition{Name: "Moscow Mathematical Olympiad", Languages: []string{"ru"}}
	src := &content.SourceRecord{ID: "mccme-mmo", Origin: "H1"}
	src.Zone.Statement = "Z2"
	o := official.RuProblem{Edition: "2003", Position: "p83", Text: "Можно ли раскрасить доску $8 \\times 8$?", URL: "https://mmo.mccme.ru/2003/mmo2003.htm"}
	if err := fillOne(root, p, "ru-moscow/2003/p83", comp, src, o, 0.9); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(full) // #nosec G304 -- a test file.
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"statement: S2\n  rule: official-original", "source: mccme-mmo", "name: Moscow Mathematical Olympiad 2003 official page", "url: https://mmo.mccme.ru/2003/mmo2003.htm", "kind: official", "extraction: native/html (om-rutex/1)", "confidence: 0.9", "statement: captured", "---\nМожно ли раскрасить доску $8 \\times 8$?\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("the record has no %q:\n%s", want, got)
		}
	}
	// A record with a text is never filled again.
	if err := fillOne(root, p, "ru-moscow/2003/p83", comp, src, o, 0.9); err == nil {
		t.Error("a second fill gave no error")
	}
}

func TestWriteFillReview(t *testing.T) {
	root := t.TempDir()
	row := func(at string) string { return at + "\tTAG\t1\thttps://x\t0.10\t-\t0.00\twhy" }
	if err := writeFillReview(root, "ru-moscow", []string{"2003", "2004"}, []string{row("ru-moscow/2004/p81"), row("ru-moscow/2003/p82")}); err != nil {
		t.Fatal(err)
	}
	// A second run of 2004 alone keeps the rows of 2003.
	if err := writeFillReview(root, "ru-moscow", []string{"2004"}, []string{row("ru-moscow/2004/p92")}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, fillReview("ru-moscow"))) // #nosec G304 -- a test file.
	if err != nil {
		t.Fatal(err)
	}
	want := fillHeader + row("ru-moscow/2003/p82") + "\n" + row("ru-moscow/2004/p92") + "\n"
	if string(b) != want {
		t.Errorf("review file:\n%s\nwant:\n%s", b, want)
	}
}

func TestPRSameYear(t *testing.T) {
	if !prSameYear("2003 год", "2003") || prSameYear("2004 год", "2003") || prSameYear("8 класс", "2003") {
		t.Error("prSameYear")
	}
}

func TestFillTexts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "texts.jsonl")
	data := `{"at":"ru-moscow/2013/p81","figure":true,"text":" Угол $\\angle A$. "}

{"at":"ru-moscow/2013/d111","figure":false,"text":"Скан.","url":"https://example.org/2den.pdf","page":1}
{"at":"ru-moscow/2014/d111","text":"Другой год.","url":"https://example.org/x.pdf","page":1}
`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	texts, err := readFillTexts(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := texts["ru-moscow/2013/p81"]; got.Text != `Угол $\angle A$.` || !got.Figure {
		t.Errorf("p81: %+v", got)
	}
	probs := []official.RuProblem{
		{Edition: "2013", Position: "p81", Text: "layer", Flags: []string{"layout"}, Page: 1},
		{Edition: "2013", Position: "p82", Text: "kept", Page: 1},
	}
	probs = withFillTexts(probs, texts, "ru-moscow", "2013", "claude-opus-5-5")
	if len(probs) != 3 {
		t.Fatalf("got %d problems", len(probs))
	}
	if p := probs[0]; p.Text != `Угол $\angle A$.` || len(p.Flags) != 0 || p.Model != "claude-opus-5-5" || !p.Figure {
		t.Errorf("p81: %+v", p)
	}
	if p := probs[1]; p.Text != "kept" || p.Model != "" {
		t.Errorf("p82: %+v", p)
	}
	if p := probs[2]; p.Position != "d111" || p.URL != "https://example.org/2den.pdf" || p.Page != 1 || p.Model != "claude-opus-5-5" {
		t.Errorf("d111: %+v", p)
	}
	for _, bad := range []string{`{"at":"x"}`, `{"at":"x","text":"a"}` + "\n" + `{"at":"x","text":"b"}`, `{`} {
		if err := os.WriteFile(path, []byte(bad), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := readFillTexts(path); err == nil {
			t.Errorf("no error for %q", bad)
		}
	}
}

func TestTotHolds(t *testing.T) {
	for _, c := range []struct {
		name, edition string
		want          bool
	}{
		{"30 турнир (2008/2009 год)", "2008-fall", true},
		{"30 турнир (2008/2009 год)", "2009-spring", true},
		{"30 турнир (2008/2009 год)", "2009-fall", false},
		{"30 турнир (2008/2009 год)", "2008-spring", false},
		{"30 турнир (2008/2009 год)", "2008", false},
	} {
		if got := totHolds(c.name, c.edition); got != c.want {
			t.Errorf("totHolds(%q, %q) = %v, want %v", c.name, c.edition, got, c.want)
		}
	}
}

func TestTotTour(t *testing.T) {
	for _, c := range []struct {
		edition, prefix string
		n               int
	}{
		{"2008-fall", "os", 30},
		{"2009-spring", "vs", 30},
		{"2026-spring", "vs", 47},
	} {
		n, prefix, ok := totTour(c.edition)
		if !ok || n != c.n || prefix != c.prefix {
			t.Errorf("totTour(%q) = %d %q %v, want %d %q", c.edition, n, prefix, ok, c.n, c.prefix)
		}
	}
	for _, name := range []string{"os-36-baz-avt.pdf", "os32os-avt.pdf", "os31sl.pdf", "vs-35-sl-avt.pdf"} {
		if !totPDFFile.MatchString(name) {
			t.Errorf("%s is not a paper PDF", name)
		}
	}
	for _, name := range []string{"vs-30-ustn-avt.pdf", "os-43-sol.pdf", "sol_os30baz.pdf"} {
		if totPDFFile.MatchString(name) {
			t.Errorf("%s is a paper PDF", name)
		}
	}
	if !totTeXFile.MatchString("os30os.tex") || totTeXFile.MatchString("sol_os30baz.tex") {
		t.Error("totTeXFile")
	}
}
