package canon

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

const linkOnly = `---
tag: ABCDEFGHJK
kind: problem
appearances:
  - at: tot/2024-fall/sa3
    primary: true
canonical:
  statement: S1
  rule: curated
statements:
  - id: S1
    lang: ru
    source: problems-ru
    name: problems.ru, problem 67493
    url: https://problems.ru/view_problem_details_new.php?id=67493
    origin: H2
    zone: Z3
    kind: transcription
status:
  statement: sourced
---
`

func TestFill(t *testing.T) {
	c := &Candidate{Source: "hf-aops-wiki", Origin: "D", Zone: "Z2", Lang: "en", Name: "AoPS wiki", URL: "https://artofproblemsolving.com/community/c6h1p1"}
	b, err := Apply([]byte(linkOnly), Additions{Statements: []outStatement{statementOf("S2", c)}})
	if err != nil {
		t.Fatal(err)
	}
	text := "Prove that $n^2+n$ is even for every integer $n$."
	out, err := Fill(b, "S2", text, "derived", "captured")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	for _, want := range []string{"statement: S2\n  rule: derived", "statement: captured", "source: problems-ru", "source: hf-aops-wiki", "---\n" + text + "\n"} {
		if !strings.Contains(s, want) {
			t.Errorf("the filled file has no %q:\n%s", want, s)
		}
	}
	if _, err := Fill(out, "S2", text, "derived", "captured"); err == nil {
		t.Error("a problem with a text got a second text")
	}
	if _, err := Fill(b, "S3", text, "derived", "captured"); err == nil {
		t.Error("a statement that the problem does not have became canonical")
	}
}

func TestAddAppearances(t *testing.T) {
	out, err := AddAppearances([]byte(linkOnly), "ru-moscow/2024/p93", "tot/2024-fall/sa3", "ru-moscow/2024/p93")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "    primary: true\n  - at: ru-moscow/2024/p93\ncanonical:") || strings.Count(s, "- at: ") != 2 || strings.Count(s, "url: ") != 1 {
		t.Errorf("the file:\n%s", s)
	}
	if _, err := AddAppearances([]byte("---\ntag: ABCDEFGHJK\nkind: problem\n---\n"), "tot/2024-fall/sa3"); err == nil {
		t.Errorf("a file with no appearances gave no error")
	}
}

func TestLinkID(t *testing.T) {
	url := "https://artofproblemsolving.com/community/c6h33973p210877"
	p := &content.Problem{Statements: []content.Statement{
		{ID: "S1", Source: "imojp", URL: "https://www.imojp.org/archive/mo2005/jmo2005/problems/jmo15mq.html", Zone: "Z3"},
		{ID: "S2", Source: "hf-aops-wiki", URL: url, Zone: "Z2"},
	}}
	c := &Candidate{Source: "hf-aops-wiki", URL: url, Text: "Let $P(x,y)$ and $Q(x,y)$ be polynomials with integer coefficients."}
	if id := LinkID(p, c); id != "S2" {
		t.Errorf("LinkID = %q, want S2", id)
	}
	// Another page of the same source, another source, or a problem with a text gives no id.
	if id := LinkID(p, &Candidate{Source: "hf-aops-wiki", URL: url + "0"}); id != "" {
		t.Errorf("another page gives %q", id)
	}
	if id := LinkID(p, &Candidate{Source: "hf-ai-mo-aops", URL: url}); id != "" {
		t.Errorf("another source gives %q", id)
	}
	p.Statement = "A text."
	if id := LinkID(p, c); id != "" {
		t.Errorf("a problem with a text gives %q", id)
	}
}
