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
	if len(c.Problems()) != 2 || len(c.Editions()) != 1 {
		t.Fatal("the corpus lists are wrong")
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
		{"wrong path", "problems/imo/2024/p2.md", "position: p2", "position: p3", "thus the path must be"},
		{"bad topic", "problems/imo/2024/p1.md", "topics: [N]", "topics: [X]", `the topic "X"`},
		{"bad zone", "problems/imo/2024/p1.md", "zone: Z2", "zone: Z9", `zone "Z9"`},
		{"unknown field", "problems/imo/2024/p1.md", "type: answer", "kind: answer", "field kind not found"},
		{"unknown competition", "problems/imo/2024/p1.md", "competition: imo", "competition: xyz", "thus the path must be"},
		{"year is not the file name", "editions/imo/2024.yaml", "year: 2024", "year: 2023", "is not the name of the file"},
		{"no front matter", "problems/imo/2024/p1.md", "---\ncompetition", "competition", "does not start with a front matter"},
		{"bad code", "registers/competitions.yaml", "code: imo", "code: IMO", "is not valid"},
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
