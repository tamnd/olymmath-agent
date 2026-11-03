package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

func TestAPIFiles(t *testing.T) {
	out := build(t, "/")
	var p ProblemRecord
	if err := json.Unmarshal([]byte(read(t, out, "api/problem/0HVHSDNNMY.json")), &p); err != nil {
		t.Fatal(err)
	}
	if p.Locator != "imo/2024/p1" || p.Edition != "2024" || p.URL != "https://example.org/imo/2024/p1" || p.Title != "IMO 2024 Problem 1" || !p.HasAnswer || p.Solutions != 1 || p.Statement == "" {
		t.Errorf("problem record: %+v", p)
	}
	// The problem record hides the answer and the solutions.
	if raw := read(t, out, "api/problem/0HVHSDNNMY.json"); strings.Contains(raw, "All even integers") || strings.Contains(raw, `"text"`) {
		t.Errorf("the problem record has the answer or a solution: %s", raw)
	}
	var s SolutionsRecord
	if err := json.Unmarshal([]byte(read(t, out, "api/solutions/0HVHSDNNMY.json")), &s); err != nil {
		t.Fatal(err)
	}
	if s.Answer != "All even integers $\\alpha$." || len(s.Solutions) != 1 || s.Solutions[0].Text == "" {
		t.Errorf("solutions record: %+v", s)
	}
	// The loose problem has no answer and no solution, so it has no solutions file.
	if _, err := os.Stat(filepath.Join(out, "api/solutions/0J8TP3MFS9.json")); err == nil {
		t.Error("the loose problem has a solutions file")
	}
	read(t, out, "api/problem/0J8TP3MFS9.json")
	var e EditionRecord
	if err := json.Unmarshal([]byte(read(t, out, "api/edition/imo/2024.json")), &e); err != nil {
		t.Fatal(err)
	}
	if e.Title != "IMO 2024" || len(e.Problems) < 2 || e.Problems[0].Position != "p1" || e.Problems[0].Tag != "0HVHSDNNMY" || len(e.Papers) == 0 {
		t.Errorf("edition record: %+v", e)
	}
	var ix IndexRecord
	if err := json.Unmarshal([]byte(read(t, out, "api/index.json")), &ix); err != nil {
		t.Fatal(err)
	}
	if ix.Problems != 3 || len(ix.Competitions) != 1 || ix.Competitions[0].Code != "imo" || ix.Competitions[0].Editions[0].Edition != "2024" {
		t.Errorf("index record: %+v", ix)
	}
	if !strings.Contains(read(t, out, "llms.txt"), "https://example.org/api/index.json") {
		t.Error("llms.txt has no link to the API")
	}
}

// TestAPISameAsCopy wants the API records and the JSON copy of the problem page to give the same facts and texts.
func TestAPISameAsCopy(t *testing.T) {
	out := build(t, "/")
	corpus, _ := content.Load("../content/testdata/valid")
	a := NewAPI(corpus, "https://example.org/")
	for _, q := range corpus.All() {
		var c problemCopy
		page := strings.TrimPrefix(problemPath(q), "/") + ".json"
		if err := json.Unmarshal([]byte(read(t, out, page)), &c); err != nil {
			t.Fatal(err)
		}
		r, s := a.Problem(q), a.Solutions(q)
		if c.Tag != r.Tag || c.Locator != r.Locator || c.URL != r.URL || c.Title != r.Title || c.Statement != r.Statement || c.Source != r.Source || c.Answer != s.Answer || len(c.Solutions) != len(s.Solutions) {
			t.Errorf("%s: the copy and the records differ", q.Tag)
		}
		want, err := Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		if got := read(t, out, "api/problem/"+q.Tag+".json"); got != string(want) {
			t.Errorf("%s: the static file is not the record:\n%s\n%s", q.Tag, got, want)
		}
	}
}

func TestAPILookup(t *testing.T) {
	corpus, _ := content.Load("../content/testdata/valid")
	corpus.Aliases["0OLDTAG000"] = "0HVHSDNNMY"
	a := NewAPI(corpus, "https://example.org")
	if p := a.Lookup("0OLDTAG000"); p == nil || p.Tag != "0HVHSDNNMY" {
		t.Errorf("alias: %v", p)
	}
	if a.Lookup("0NOSUCHTAG") != nil {
		t.Error("an unknown tag gives a problem")
	}
	if a.Edition("imo", "2024") == nil || a.Edition("imo", "1900") != nil || a.Edition("nope", "2024") != nil {
		t.Error("edition lookup is wrong")
	}
}
