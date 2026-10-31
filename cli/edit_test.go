package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// copyContent copies the valid test content to a new directory.
func copyContent(t *testing.T) string {
	t.Helper()
	src := "../content/testdata/valid"
	dir := t.TempDir()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		out := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestNewProblemAndSolution(t *testing.T) {
	dir := copyContent(t)
	code, out, errs := run("new", "problem", "imo/2027/p1", "-content", dir, "-source", "imo-official")
	if code != ExitOK {
		t.Fatalf("new problem: %d %s", code, errs)
	}
	if !strings.Contains(out, "problems/imo/2027/p1.md") || !strings.Contains(out, "editions/imo/2027.yaml") {
		t.Errorf("new problem wrote %q", out)
	}
	code, out, errs = run("new", "solution", "imo/2027/p1", "-origin", "human", "-content", dir)
	if code != ExitOK || !strings.HasPrefix(out, "solutions/imo/2027/p1/") {
		t.Fatalf("new solution: %d %q %s", code, out, errs)
	}
	code, out, _ = run("audit", "-content", dir, "-q")
	if code != ExitOK {
		t.Errorf("the new files fail the audit:\n%s", out)
	}
	code, out, _ = run("fmt", "-check", "-content", dir)
	if code != ExitOK {
		t.Errorf("the new files are not in canonical form:\n%s", out)
	}
	tags, _ := os.ReadFile(filepath.Join(dir, "registers", "tags.tsv"))
	if strings.Count(string(tags), "\n") < 6 {
		t.Errorf("tags.tsv has no new tags:\n%s", tags)
	}
}

func TestNewErrors(t *testing.T) {
	dir := copyContent(t)
	for _, args := range [][]string{
		{"new", "problem", "nope/2027/p1", "-source", "imo-official"},
		{"new", "problem", "imo/2027", "-source", "imo-official"},
		{"new", "problem", "imo/2027/p1"},
		{"new", "problem", "imo/2024/p1", "-source", "imo-official"},
		{"new", "solution", "imo/2024/p1", "-origin", "robot"},
		{"new", "solution", "imo/1900/p1"},
		{"new", "thing"},
	} {
		if code, _, _ := run(append(args, "-content", dir)...); code == ExitOK {
			t.Errorf("%v: want an error", args)
		}
	}
}

func TestFmt(t *testing.T) {
	dir := copyContent(t)
	p := filepath.Join(dir, "problems", "imo", "2024", "p2.md")
	b, _ := os.ReadFile(p)
	moved := strings.Replace(string(b), "tag: 03JXB69J3N\nkind: problem\n", "kind: problem\ntag: 03JXB69J3N\n", 1) + "\n\n"
	if err := os.WriteFile(p, []byte(moved), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := run("fmt", "-check", "-content", dir); code != ExitFindings || !strings.Contains(out, "p2.md") {
		t.Fatalf("fmt -check: %d %q", code, out)
	}
	if code, _, _ := run("fmt", "-content", dir); code != ExitOK {
		t.Fatal("fmt failed")
	}
	got, _ := os.ReadFile(p)
	if string(got) != string(b) {
		t.Errorf("fmt did not restore the file:\n%s", got)
	}
}

func TestTag(t *testing.T) {
	dir := copyContent(t)
	code, out, _ := run("tag", "new", "-n", "3", "-content", dir)
	if code != ExitOK || len(strings.Fields(out)) != 3 {
		t.Fatalf("tag new: %d %q", code, out)
	}
	if code, _, _ := run("tag", "check", "0hvh-sdnnmy"); code != ExitOK {
		t.Error("a valid tag fails the check")
	}
	if code, _, _ := run("tag", "check", "0HVHSDNNMZ"); code != ExitFindings {
		t.Error("a tag with a wrong check character passes")
	}
}

func TestAuditList(t *testing.T) {
	code, out, _ := run("audit", "-list")
	if code != ExitOK || !strings.Contains(out, "X02") {
		t.Fatalf("audit -list: %d %q", code, out)
	}
	if code, _, _ := run("audit", "-rule", "Z99", "-content", "../content/testdata/valid"); code != ExitUsage {
		t.Error("an unknown rule is not a usage error")
	}
}
