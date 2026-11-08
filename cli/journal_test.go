package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/official"
)

func TestWriteJournal(t *testing.T) {
	root := copyContent(t)
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the test content has %d errors", f.Errors())
	}
	c := canon.Candidate{Source: "evanchen", Origin: "H2", Zone: "Z3", Lang: "en", Name: "A journal, problem 5171", URL: "https://example.org/issue.pdf"}
	news := []journalLink{{at: "imo/2031/p5171", cand: c, page: 40}}
	if err := writeJournal(root, corpus, "en", news); err != nil {
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
