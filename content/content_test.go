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
	got := splitStatements("Text one.\n\n<!-- S2 lang=fr -->\nTexte deux.\n", "S1")
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
