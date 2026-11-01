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
	if len(p.Solutions) != 1 || p.Solutions[0].Origin != "human" {
		t.Fatalf("solutions: %+v", p.Solutions)
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
	} {
		got := shapeError(&tc.c)
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("shapeError(%+v) = %q, want %q", tc.c, got, tc.want)
		}
	}
}
