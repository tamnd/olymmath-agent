package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/seed"
)

func TestWriteJournal(t *testing.T) {
	root := copyContent(t)
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the test content has %d errors", f.Errors())
	}
	c := canon.Candidate{Source: "evanchen", Origin: "H2", Zone: "Z3", Lang: "en", Name: "A journal, problem 5171", URL: "https://example.org/issue.pdf"}
	news := []journalLink{{at: "imo/2031/p5171", cand: c, page: 40}}
	if err := writeJournal(root, corpus, news); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "problems", "imo", "2031", "p5171.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"https://example.org/issue.pdf", "A journal, problem 5171"} {
		if !strings.Contains(string(b), want) {
			t.Errorf("the record has no %q:\n%s", want, b)
		}
	}
	corpus, f = content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the content has %d errors after om journal: %v", f.Errors(), f)
	}
	p := official.Owners(corpus)["imo/2031/p5171"]
	if p == nil || strings.TrimSpace(p.Statement) != "" {
		t.Errorf("the link-only record is missing or has a text: %+v", p)
	}
	if code, _, _ := run("journal", "-content", root, "nojournal"); code != ExitUsage {
		t.Errorf("an unknown journal gave %d", code)
	}
}

func TestLinks(t *testing.T) {
	root := copyContent(t)
	dir := t.TempDir()
	rows := []seed.Row{
		{Source: "evanchen", File: "f", Row: 0, ID: "501", Labels: []string{"imo"}, Year: 2032, Number: "p81", Lang: "ru", Statement: "A text that stays out.", Ref: "https://example.org/problem?id=501"},
		{Source: "evanchen", File: "f", Row: 1, ID: "502", Labels: []string{"imo"}, Year: 2032, Number: "p81", Lang: "ru", Statement: "The same position again.", Ref: "https://example.org/problem?id=502"},
		{Source: "evanchen", File: "f", Row: 2, ID: "503", Labels: []string{"no such olympiad"}, Year: 2032, Number: "p1", Ref: "https://example.org/problem?id=503"},
		{Source: "evanchen", File: "f", Row: 3, ID: "504", Labels: []string{"imo"}, Year: 2032, Number: "", Ref: "https://example.org/problem?id=504"},
		// A row that names its edition keeps its exact position. The edition must start with the year of the row.
		{Source: "evanchen", File: "f", Row: 4, ID: "505", Labels: []string{"imo"}, Year: 2032, Edition: "2032-fall", Number: "ja3", Ref: "https://example.org/problem?id=505"},
		{Source: "evanchen", File: "f", Row: 5, ID: "506", Labels: []string{"imo"}, Year: 2032, Edition: "2031-fall", Number: "ja4", Ref: "https://example.org/problem?id=506"},
		{Source: "evanchen", File: "f", Row: 6, ID: "507", Labels: []string{"imo"}, Year: 2032, Edition: "2032-fall", Number: "ja5/p1", Ref: "https://example.org/problem?id=507"},
		// The same page in another paper is one more appearance of the same problem.
		{Source: "evanchen", File: "f", Row: 7, ID: "505", Labels: []string{"imo"}, Year: 2032, Edition: "2032-fall", Number: "sa1", Ref: "https://example.org/problem?id=505"},
	}
	err := seed.WriteFile(dir, "evanchen", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		for _, r := range rows {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if code, _, _ := run("links", "-content", root, "-seed", dir, "imo-official"); code != ExitUsage {
		t.Errorf("a source of zone Z2 gave %d", code)
	}
	code, out, errs := run("links", "-content", root, "-seed", dir, "evanchen")
	if code != ExitOK || !strings.Contains(out, "2 new link-only problems") {
		t.Fatalf("om links gave %d %q %q", code, out, errs)
	}
	b, err := os.ReadFile(filepath.Join(root, "problems", "imo", "2032", "p81.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(b), "https://example.org/problem?id=501") || !strings.Contains(string(b), "problem 501") || strings.Contains(string(b), "stays out") {
		t.Errorf("the record is not a link to problem 501:\n%s", b)
	}
	if corpus, f := content.Load(root); f.Errors() > 0 || official.Owners(corpus)["imo/2032/p81"] == nil || official.Owners(corpus)["imo/2032-fall/ja3"] == nil {
		t.Errorf("the content has %d errors, or no imo/2032/p81 or imo/2032-fall/ja3: %v", f.Errors(), f)
	}
	if b, err := os.ReadFile(filepath.Join(root, "editions", "imo", "2032-fall.yaml")); err != nil || !strings.HasPrefix(string(b), "year: 2032\n") {
		t.Errorf("the edition 2032-fall: %q, %v", b, err)
	}
	corpus, _ := content.Load(root)
	if p := official.Owners(corpus)["imo/2032-fall/sa1"]; p == nil || p.Locator() != "imo/2032-fall/ja3" {
		t.Errorf("the page of two papers is not one problem: %+v", p)
	}
	// A second run adds nothing: the positions are taken, and a page that the content links to is not linked again at another position.
	rows[7].Number = "sa2"
	err = seed.WriteFile(dir, "evanchen", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		for _, r := range rows {
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if code, out, errs := run("links", "-content", root, "-seed", dir, "evanchen"); code != ExitOK || !strings.Contains(out, "0 new link-only problems") || !strings.Contains(out, "problem in the content at another position 1") {
		t.Errorf("a second run gave %d %q %q", code, out, errs)
	}
}
