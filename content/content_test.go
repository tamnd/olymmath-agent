package content

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadValid(t *testing.T) {
	c, fs := Load("testdata/valid")
	if len(fs) != 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	imo := c.Competition("imo")
	if imo == nil || len(imo.Editions) != 1 {
		t.Fatalf("imo: %+v", imo)
	}
	e := imo.Editions[0]
	if len(e.Problems) != 2 || e.Problems[0].Position != "p1" {
		t.Fatalf("problems: %+v", e.Problems)
	}
	for _, p := range e.Problems {
		if p.Source.Origin == "H1" && p.Official != nil {
			t.Errorf("%s: an official canonical text has an extra official link %+v", p.Locator(), p.Official)
		}
	}
	p := e.Problems[0]
	if p.Locator() != "imo/2024/p1" || p.Number() != 1 || p.Paper == nil || p.Paper.Name != "Day 1" {
		t.Fatalf("problem: %+v", p)
	}
	if !strings.HasPrefix(p.Statement, "Determine") {
		t.Fatalf("statement: %q", p.Statement)
	}
	if len(c.Problems()) != 2 || len(c.Editions()) != 1 || len(c.All()) != 3 || len(c.Loose) != 1 {
		t.Fatal("the corpus lists are wrong")
	}
	if p.Tag != "0HVHSDNNMY" || c.ByTag(p.Tag) != p || p.Source.Origin != "H1" || p.Source.Zone != "Z2" {
		t.Fatalf("problem: %+v", p)
	}
	if len(p.Solutions) != 2 || p.Solutions[0].Origin != "human" || p.Solutions[1].Origin != "ai" {
		t.Fatalf("solutions: %+v", p.Solutions)
	}
	ai := p.Solutions[1]
	if ai.Run == nil || ai.Run.Cost != 0.66 || ai.Grade == nil || ai.Grade.Score != 6 || len(ai.All()) != 2 || ai.HumanGraded() {
		t.Fatalf("AI solution: %+v", ai)
	}
	if j := ai.All()[0]; j.Rubric != "imo/2024/p1@r2" || j.At != "2026-11-03" || j.Max != 7 {
		t.Fatalf("a grader does not take the rubric and the date of the jury: %+v", j)
	}
	if p.Rubric == nil || p.Rubric.Ref() != "imo/2024/p1@r2" || p.Rubric.Max() != 7 || c.Rubrics["imo/2024/p1"] != p.Rubric {
		t.Fatalf("rubric: %+v", p.Rubric)
	}
	if r := c.Runs["solve-20261103-0001"]; r == nil || len(r.Items) != 1 || r.Items[0].Tokens.Output != 30000 {
		t.Fatalf("runs: %+v", c.Runs)
	}
	if pe := c.People["tamnd"]; pe == nil || !pe.Has("reviewer") || c.People["ann"].Has("reviewer") {
		t.Fatalf("people: %+v", c.People)
	}
	loose := c.Loose[0]
	if loose.Locator() != loose.Tag || loose.Source.Name != "Mathematics Stack Exchange" {
		t.Fatalf("loose: %+v", loose)
	}
	if c.Redirects["imo/2024/q1"] != "imo/2024/p1" || c.Tags["006SZE6NAS"] != "solution" {
		t.Fatal("the registers are wrong")
	}
}

func TestSplitStatements(t *testing.T) {
	got := SplitStatements("Text one.\n\n<!-- S2 lang=fr -->\nTexte deux.\n", "S1")
	if got["S1"] != "Text one." || got["S2"] != "Texte deux." {
		t.Fatalf("got %q", got)
	}
}

// copyTree copies the valid fixtures to a temporary directory, so that a test can break one file.
func copyTree(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := os.CopyFS(dst, os.DirFS("testdata/valid"))
	if err != nil {
		t.Fatal(err)
	}
	return dst
}

func TestLoadFindings(t *testing.T) {
	cases := []struct {
		name, file, old, new, want string
	}{
		{"wrong path", "problems/imo/2024/p2.md", "at: imo/2024/p2", "at: imo/2024/p3", "the path must be problems/imo/2024/p3.md"},
		{"bad tag", "problems/imo/2024/p1.md", "tag: 0HVHSDNNMY", "tag: 0HVHSDNNMZ", "T01 the tag"},
		{"same tag", "problems/imo/2024/p2.md", "tag: 03JXB69J3N", "tag: 0HVHSDNNMY", "is also the tag of"},
		{"bad appearance", "problems/imo/2024/p1.md", "at: imo/2024/p1", "at: imo/2024", "is not the locator of an appearance"},
		{"unknown field", "problems/imo/2024/p1.md", "type: answer", "typ: answer", "field typ not found"},
		{"unknown competition", "registers/competitions.yaml", "code: imo", "code: imx", `the competition "imo" is not in the register`},
		{"no edition", "editions/imo/2024.yaml", "year: 2024", "year: 2023", "must be the year 2023"},
		{"no front matter", "problems/imo/2024/p1.md", "---\ntag", "tag", "does not start with a front matter"},
		{"bad code", "registers/competitions.yaml", "code: imo", "code: IMO", "is not valid"},
		{"orphan text", "problems/imo/2024/p1.md", "is a multiple of $n$.", "is a multiple of $n$.\n\n<!-- S7 -->\nMore.", "no statement S7"},
		{"solution of no problem", "solutions/imo/2024/p1/006SZE6NAS.md", "problem: 0HVHSDNNMY", "problem: 0ZZZZZZZZ0", "L01"},
		{"unplaced position", "problems/_loose/0J8/0J8TP3MFS9.md", "at: imo/2024\n", "at: imo/2024/p1\n", "is not a competition or an edition"},
		{"unplaced competition", "problems/_loose/0J8/0J8TP3MFS9.md", "at: imo/2024\n", "at: imx/2024\n", "names a competition that is not in the register"},
		{"unplaced placed", "problems/imo/2024/p2.md", "kind: problem\n", "kind: problem\nunplaced:\n  - at: imo\n", "has a primary appearance and unplaced entries"},
		{"bad tsv", "registers/redirects.tsv", "imo/2024/q1\timo/2024/p1", "imo/2024/q1", "line 2 has 1 fields"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyTree(t)
			path := filepath.Join(root, tc.file)
			b, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(b), tc.old) {
				t.Fatalf("the fixture has no %q", tc.old)
			}
			if err := os.WriteFile(path, []byte(strings.Replace(string(b), tc.old, tc.new, 1)), 0o600); err != nil {
				t.Fatal(err)
			}
			_, fs := Load(root)
			if !strings.Contains(fs.Error(), tc.want) {
				t.Fatalf("want a finding with %q, got:\n%v", tc.want, fs)
			}
		})
	}
}

func TestLoadPaperEditions(t *testing.T) {
	root := copyTree(t)
	reg := filepath.Join(root, "registers", "competitions.yaml")
	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, []byte(`
- code: amc10
  name: AMC 10
  group: usa
  level: high-school
  first_year: 2000
  edition_form: year-letter
  papers:
    - {from: 2002, letters: [a, b]}
    - {from: 2021, to: 2021, part: fall, letters: [a, b]}
  per_year: 25
  sources: [imo-official]
  description: A test competition.
`)...)
	if err := os.WriteFile(reg, b, 0o600); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(root, "editions", "amc10")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"2001", "2021a", "2021-fall-b", "2022-fall-a", "2003c"} {
		if err := os.WriteFile(filepath.Join(dir, id+".yaml"), []byte("year: "+id[:4]+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	c, fs := Load(root)
	got := fs.Error()
	for _, want := range []string{"2022-fall-a is not one of them", "2003c is not one of them"} {
		if !strings.Contains(got, want) {
			t.Errorf("want a finding with %q, got:\n%v", want, got)
		}
	}
	if len(fs) != 2 {
		t.Errorf("findings:\n%v", got)
	}
	var ids []string
	for _, e := range c.Competition("amc10").Editions {
		ids = append(ids, e.ID)
	}
	if strings.Join(ids, " ") != "2021a 2021-fall-b 2001" {
		t.Errorf("editions %q", ids)
	}
}

func TestShape(t *testing.T) {
	c := &Competition{
		Layout:    []Layout{{Positions: []string{"a1", "c1"}}},
		Eras:      []Era{{From: 1959, To: 2005, PerYear: 3}, {From: 2015, To: 2015}, {From: 2020, Layout: []Layout{{Positions: []string{"a1"}}, {Positions: []string{"g1"}}}}},
		Overrides: map[int][]string{1990: {"p1", "p2"}},
	}
	for _, tc := range []struct {
		year int
		want string
	}{
		{1959, "p1 p2 p3"}, {1990, "p1 p2"}, {2005, "p1 p2 p3"}, {2010, "a1 c1"}, {2030, "a1 g1"}, {0, "a1 c1"}, {2015, ""},
	} {
		if got := strings.Join(c.Shape(tc.year).Positions(), " "); got != tc.want {
			t.Errorf("Shape(%d) = %q, want %q", tc.year, got, tc.want)
		}
	}
	if !c.HasShape() || (&Competition{}).HasShape() || !(&Competition{Eras: []Era{{From: 2000, PerYear: 4}}}).HasShape() {
		t.Error("HasShape is wrong")
	}
}

func TestLetters(t *testing.T) {
	c := &Competition{FirstYear: 2000, Gaps: []int{2010}, EditionForm: "year-letter", Papers: []Papers{{From: 2002, To: 2020, Letters: []string{"a", "b"}}}}
	for _, tc := range []struct {
		year int
		want string
	}{{1999, "<nil>"}, {2001, ""}, {2002, "a b"}, {2010, "<nil>"}, {2020, "a b"}, {2021, ""}} {
		got := "<nil>"
		if ls := c.Letters(tc.year); ls != nil {
			got = strings.Join(ls, " ")
		}
		if got != tc.want {
			t.Errorf("Letters(%d) = %q, want %q", tc.year, got, tc.want)
		}
	}
}

func TestEditionIDs(t *testing.T) {
	c := &Competition{FirstYear: 2000, Gaps: []int{2010}, EditionForm: "year-letter", Papers: []Papers{
		{From: 2002, Letters: []string{"a", "b"}},
		{From: 2021, To: 2021, Part: "fall", Letters: []string{"a", "b"}},
	}}
	for _, tc := range []struct {
		year int
		want string
	}{{1999, ""}, {2001, "2001"}, {2002, "2002a 2002b"}, {2010, ""}, {2021, "2021a 2021b 2021-fall-a 2021-fall-b"}, {2022, "2022a 2022b"}} {
		if got := strings.Join(c.EditionIDs(tc.year), " "); got != tc.want {
			t.Errorf("EditionIDs(%d) = %q, want %q", tc.year, got, tc.want)
		}
	}
	if got := c.PartLetters(2022, "fall"); got != nil {
		t.Errorf("PartLetters(2022, fall) = %q", got)
	}
	if got := strings.Join(c.Parts(2021), " "); got != "fall" {
		t.Errorf("Parts(2021) = %q", got)
	}
	if got := strings.Join(c.Letters(2021), " "); got != "a b" {
		t.Errorf("Letters(2021) = %q", got)
	}
}

func TestShapeError(t *testing.T) {
	ok := Layout{Positions: []string{"p1", "p2"}}
	for _, tc := range []struct {
		c    Competition
		want string
	}{
		{Competition{Layout: []Layout{ok}, Eras: []Era{{From: 1990, To: 1999, PerYear: 4}, {From: 2000, PerYear: 5}}, Overrides: map[int][]string{1995: {"p1"}}}, ""},
		{Competition{Layout: []Layout{{Positions: []string{"p1", "p1"}}}}, "two times"},
		{Competition{Layout: []Layout{{Positions: []string{"P 1"}}}}, "not valid"},
		{Competition{Eras: []Era{{To: 1990, PerYear: 4}}}, "no from year"},
		{Competition{Eras: []Era{{From: 1990, To: 1980, PerYear: 4}}}, "ends before"},
		{Competition{Eras: []Era{{From: 1990}}}, ""},
		{Competition{Eras: []Era{{From: 1990, To: 2000, PerYear: 4}, {From: 2000, PerYear: 5}}}, "overlaps"},
		{Competition{Eras: []Era{{From: 1990, PerYear: 4}, {From: 1980, To: 1985, PerYear: 5}, {From: 1995, PerYear: 5}}}, "era 3 overlaps era 1"},
		{Competition{Overrides: map[int][]string{1990: {}}}, "no positions"},
		{Competition{EditionForm: "year-letter", Papers: []Papers{{From: 2002, Letters: []string{"a", "b"}}}}, ""},
		{Competition{Papers: []Papers{{From: 2002, Letters: []string{"a", "b"}}}}, "year-letter"},
		{Competition{EditionForm: "year-letter", Papers: []Papers{{From: 2002, Letters: []string{"A"}}}}, "not one small letter"},
		{Competition{EditionForm: "year-letter", Papers: []Papers{{From: 2002}}}, "no letters"},
		{Competition{EditionForm: "year-letter", Papers: []Papers{{From: 2002, Letters: []string{"a", "b"}}, {From: 2021, To: 2021, Part: "fall", Letters: []string{"a", "b"}}}}, ""},
		{Competition{EditionForm: "year-letter", Papers: []Papers{{From: 2002, Letters: []string{"a", "b"}}, {From: 2021, To: 2021, Part: "Fall 2", Letters: []string{"a"}}}}, "not one word"},
		{Competition{EditionForm: "year-letter", Papers: []Papers{{From: 2002, Letters: []string{"a", "b"}}, {From: 2021, To: 2021, Letters: []string{"c"}}}}, "papers 2 overlaps papers 1"},
	} {
		got := shapeError(&tc.c)
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("shapeError(%+v) = %q, want %q", tc.c, got, tc.want)
		}
	}
}

func TestBucket(t *testing.T) {
	for _, tc := range []struct {
		score, max int
		want       string
	}{{7, 7, "correct"}, {6, 7, "almost"}, {5, 7, "almost"}, {4, 7, "partial"}, {1, 7, "partial"}, {0, 7, "incorrect"}, {9, 10, "almost"}, {10, 10, "correct"}, {1, 10, "incorrect"}} {
		if got := Bucket(tc.score, tc.max); got != tc.want {
			t.Errorf("Bucket(%d, %d) = %s, want %s", tc.score, tc.max, got, tc.want)
		}
	}
}

func TestParseRubricRef(t *testing.T) {
	loc, v, err := ParseRubricRef("imo/2024/p1@r12")
	if err != nil || loc != "imo/2024/p1" || v != 12 {
		t.Fatalf("got %q %d %v", loc, v, err)
	}
	for _, bad := range []string{"imo/2024/p1", "imo/2024/p1@r0", "@r1", "imo/2024/p1@2"} {
		if _, _, err := ParseRubricRef(bad); err == nil {
			t.Errorf("%q: want an error", bad)
		}
	}
}

func TestRubricError(t *testing.T) {
	ok := Rubric{Version: 1, Scale: "imo-7", Source: "written", Status: "draft", Checkpoints: []Checkpoint{{Points: 7, Text: "all"}}}
	if msg := rubricError(&ok); msg != "" {
		t.Fatal(msg)
	}
	for name, edit := range map[string]func(r *Rubric){
		"sum":      func(r *Rubric) { r.Checkpoints[0].Points = 6 },
		"scale":    func(r *Rubric) { r.Scale = "ten" },
		"reviewed": func(r *Rubric) { r.Status = "reviewed" },
		"drafted":  func(r *Rubric) { r.Source = "drafted" },
		"deduct":   func(r *Rubric) { r.Deductions = []Checkpoint{{Points: 1, Text: "x"}} },
	} {
		r := ok
		r.Checkpoints = []Checkpoint{ok.Checkpoints[0]}
		edit(&r)
		if rubricError(&r) == "" {
			t.Errorf("%s: want an error", name)
		}
	}
}
