package cli

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/lane"
	"github.com/tamnd/olymmath-agent/ocr"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/site"
)

func TestParsePages(t *testing.T) {
	got, err := parsePages("1-3, 5")
	if err != nil || !slices.Equal(got, []int{1, 2, 3, 5}) {
		t.Errorf("got %v %v", got, err)
	}
	for _, bad := range []string{"0", "3-1", "x"} {
		if _, err := parsePages(bad); err == nil {
			t.Errorf("%q gave no error", bad)
		}
	}
}

func TestTextAllowed(t *testing.T) {
	src := &content.SourceRecord{Terms: "No licence stated."}
	src.Zone.Statement = "Z2"
	if !textAllowed(src) {
		t.Error("a Z2 source with no licence allows no text")
	}
	src.Terms = "So the content links to the papers and holds no text of them."
	if textAllowed(src) {
		t.Error("a source whose terms say that the content holds no text allows a text")
	}
	src.Terms, src.DoNotRepublish = "", true
	if textAllowed(src) {
		t.Error("a source that asks that no text is published allows a text")
	}
	src.DoNotRepublish, src.Zone.Statement = false, "Z3"
	if textAllowed(src) {
		t.Error("a Z3 source allows a text")
	}
}

func TestScans(t *testing.T) {
	root := copyContent(t)
	reg := filepath.Join(root, "registers", "competitions.yaml")
	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), "  first_year: 1959\n", "  first_year: 1959\n  per_year: 6\n", 1))
	if err := os.WriteFile(reg, b, 0o600); err != nil {
		t.Fatal(err)
	}
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the test content has %d errors", f.Errors())
	}
	e, err := findEdition(corpus, "imo/2024")
	if err != nil {
		t.Fatal(err)
	}
	paper, err := ocrPaper(e, ocrOptions{source: "imo-official"}, []string{"https://www.imo-official.org/problems/2024/eng.pdf"})
	if err != nil {
		t.Fatal(err)
	}
	pages := []ocr.Page{{N: 1, Lane: "claude", Version: "2.1.318", Model: "m", Text: "# Day 2\n\n**1.** A text that is not new, because p1 has a problem.\n\n**3.** Let $n \\ge 1$ be an integer. Prove that $n^2 \\ge n$.\n\n**4.** Find all $x$ with $x [?] 2$ here."}}
	markup, err := site.NewMarkup(nil)
	if err != nil {
		t.Fatal(err)
	}
	news := planScans(corpus, e, paper, pages, false, true, markup)
	if len(news) != 2 || news[0].at != "imo/2024/p3" || news[1].at != "imo/2024/p4" {
		t.Fatalf("news = %+v", news)
	}
	if news[0].scan.Text == "" || news[0].scan.Extraction != "vision/claude@2.1.318 (m)" || news[1].scan.Text != "" || news[1].scan.Extraction != "" {
		t.Errorf("news = %+v", news)
	}
	if got := planScans(corpus, e, paper, pages, false, false, markup); got[0].scan.Text != "" {
		t.Errorf("a source with no text allowed gave a text: %+v", got[0])
	}
	if err := writeScans(root, corpus, e, corpus.Source("imo-official"), paper, news); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(root, "problems", "imo", "2024", "p3.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"statement: captured", "extraction: vision/claude@2.1.318 (m)", "confidence: 0.9", "Prove that $n^2 \\ge n$."} {
		if !strings.Contains(string(b), want) {
			t.Errorf("p3 has no %q:\n%s", want, b)
		}
	}
	if b, err = os.ReadFile(filepath.Join(root, "editions", "imo", "2024.yaml")); err != nil || !strings.Contains(string(b), "eng.pdf") {
		t.Errorf("the edition does not list the paper:\n%s %v", b, err)
	}
	corpus, f = content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the content has %d errors after om ocr: %v", f.Errors(), f)
	}
	p := official.Owners(corpus)["imo/2024/p3"]
	if p == nil || p.Source.Extraction == "" || p.Statement == "" {
		t.Errorf("the loaded p3 has no extraction: %+v", p)
	}
}

func TestImageScans(t *testing.T) {
	root := copyContent(t)
	reg := filepath.Join(root, "registers", "competitions.yaml")
	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), "  first_year: 1959\n", "  first_year: 1959\n  per_year: 6\n", 1))
	if err := os.WriteFile(reg, b, 0o600); err != nil {
		t.Fatal(err)
	}
	srcs := filepath.Join(root, "registers", "sources.yaml")
	b, err = os.ReadFile(srcs)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, []byte("- id: hf-test\n  name: Test set\n  kind: dataset\n  origin: D\n  url: https://huggingface.co/datasets/test/set\n  access: hf\n  terms: No licence stated.\n  zone: {statement: Z2, solution: Z2}\n  ai_train: unknown\n  wave: 1\n  state: active\n")...)
	if err := os.WriteFile(srcs, b, 0o600); err != nil {
		t.Fatal(err)
	}
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the test content has %d errors: %v", f.Errors(), f)
	}
	e, err := findEdition(corpus, "imo/2024")
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "editions", "imo", "2024.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	src := corpus.Source("hf-test")
	paper, err := ocrPaper(e, ocrOptions{source: "hf-test", lang: "en"}, []string{src.URL})
	if err != nil {
		t.Fatal(err)
	}
	pages := []ocr.Page{{N: 1, File: "/x/imo_2024_3.png", Lane: "claude", Version: "2.1.318", Text: "**3.** Let $n \\ge 1$ be an integer. Prove that $n^2 \\ge n$."}}
	markup, err := site.NewMarkup(nil)
	if err != nil {
		t.Fatal(err)
	}
	news := planScans(corpus, e, paper, pages, false, true, markup)
	if len(news) != 1 || news[0].at != "imo/2024/p3" || news[0].ref != "imo_2024_3" {
		t.Fatalf("news = %+v", news)
	}
	news[0].scan.Page = 0
	if err := writeScans(root, corpus, e, src, paper, news); err != nil {
		t.Fatal(err)
	}
	b, err = os.ReadFile(filepath.Join(root, "problems", "imo", "2024", "p3.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"source: hf-test", "name: Test set, imo_2024_3", "kind: transcription", "origin: D", "url: https://huggingface.co/datasets/test/set"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("p3 has no %q:\n%s", want, b)
		}
	}
	if strings.Contains(string(b), "page:") {
		t.Errorf("a crop gave a page:\n%s", b)
	}
	// A dataset is not an official paper, so the edition does not list it.
	if after, err := os.ReadFile(filepath.Join(root, "editions", "imo", "2024.yaml")); err != nil || string(after) != string(before) {
		t.Errorf("the edition changed:\n%s %v", after, err)
	}
	if _, f = content.Load(root); f.Errors() > 0 {
		t.Fatalf("the content has %d errors after om ocr: %v", f.Errors(), f)
	}
}

func TestImageFiles(t *testing.T) {
	dir := t.TempDir()
	for _, n := range []string{"b.png", "a.JPG", "c.txt", "d.webp"} {
		if err := os.WriteFile(filepath.Join(dir, n), nil, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	got, err := imageFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, g := range got {
		names = append(names, filepath.Base(g))
	}
	if !slices.Equal(names, []string{"a.JPG", "b.png", "d.webp"}) {
		t.Errorf("files = %v", names)
	}
	if _, err := imageFiles(t.TempDir()); err == nil {
		t.Error("an empty directory gave no error")
	}
}

func TestOCRLanes(t *testing.T) {
	lanes, variants, err := ocrLanes(ocrOptions{lanes: "local", local: lane.DefaultLocal})
	if err != nil || len(lanes) != 1 || lanes[0].Name() != "local" {
		t.Fatalf("%v %v", lanes, err)
	}
	if !slices.Equal(variants, []string{"local/GLM-OCR-8bit", "local/reader-a"}) {
		t.Errorf("variants %v", variants)
	}
	if _, _, err := ocrLanes(ocrOptions{lanes: "local,codex", local: lane.DefaultLocal}); err == nil {
		t.Error("the local lane ran with another lane")
	}
	if _, _, err := ocrLanes(ocrOptions{lanes: "local", local: "http://127.0.0.1:8079/v1"}); err == nil {
		t.Error("a target with no model")
	}
}
