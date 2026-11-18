package canon

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

func TestAddScan(t *testing.T) {
	p := &content.Problem{Statements: []content.Statement{{ID: "S1", Source: "problems-ru"}}}
	c := &Candidate{Source: "problems-ru", Origin: "H2", Zone: "Z2", Lang: "ru", Name: "problems.ru, problem 67493", URL: "https://problems.ru/view_problem_details_new.php?id=67493"}
	if _, err := AddScan(p, c, Scan{Page: 2, Text: "x"}); err == nil {
		t.Error("a scanned text with no extraction was added")
	}
	add, err := AddScan(p, c, Scan{Page: 2, Extraction: "vision/claude@2.1.325 (claude-opus-5-5)", Confidence: 0.899})
	if err != nil {
		t.Fatal(err)
	}
	if add.ID(c) != "S2" {
		t.Fatalf("the new statement is %q", add.ID(c))
	}
	b, err := Apply([]byte(linkOnly), add)
	if err != nil {
		t.Fatal(err)
	}
	text := "Докажите, что $n^2+n$ чётно."
	out, err := Fill(b, "S2", text, "official-original", "captured")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"id: S1", "id: S2", "page: 2", "extraction: vision/claude@2.1.325 (claude-opus-5-5)", "confidence: 0.9\n", "statement: S2\n  rule: official-original", "statement: captured", "---\n" + text + "\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("the file has no %q:\n%s", want, s)
		}
	}
}
