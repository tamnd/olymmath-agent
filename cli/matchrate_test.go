package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/seed"
)

// matchRows writes the seed rows of the test to a dataset.
func matchRows(t *testing.T, dir, source string) {
	t.Helper()
	rows := []seed.Row{
		// The appearance imo/2024/p1 has a record.
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2024, Number: "1", Statement: "Some other text of the problem."},
		// The same text as imo/2024/p1 with no position.
		{Source: "imo-official", Labels: []string{"IMO"}, Statement: "Determine all real numbers $\\alpha$ such that, for every positive integer $n$, the integer $$\\lfloor \\alpha \\rfloor + \\lfloor 2\\alpha \\rfloor + \\cdots + \\lfloor n\\alpha \\rfloor$$ is a multiple of $n$."},
		// A new problem of a known competition.
		{Source: "imo-official", Labels: []string{"IMO"}, Year: 2023, Number: "6", Statement: "Kangaroos jump along hexagonal lattices until purple tulips bloom."},
		// A new problem with no competition.
		{Source: "imo-official", Labels: []string{"Unknown Cup"}, Statement: "Wombats stack copper teacups beside quiet volcanoes."},
	}
	err := seed.WriteFile(dir, source, func(w io.Writer) error {
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
}

func TestMatchrate(t *testing.T) {
	root := copyContent(t)
	dir := t.TempDir()
	matchRows(t, dir, "imo-official")
	code, out, errs := run("matchrate", "-content", root, "-seed", dir, "imo-official")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "imo-official: 4 rows, 2 matched, 0 variants, 0 near, 2 new (50.00%), 2 new problems (5000 for each 10,000 rows), 1 new with a competition") {
		t.Errorf("output: %s", out)
	}
	md, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "match.md"))
	if err != nil || !strings.Contains(string(md), "| imo-official | 4 | 2 | 1 | 0 | 0 | 2 | 50.00% | 2 | 5000 | 1 |") {
		t.Errorf("match.md: %v\n%s", err, md)
	}
	tsv, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "match-new.tsv"))
	if err != nil || !strings.Contains(string(tsv), "imo-official\t2023\t6\tIMO\t") || strings.Contains(string(tsv), "Wombats") {
		t.Errorf("match-new.tsv: %v\n%s", err, tsv)
	}
}

func TestMatchrateDerived(t *testing.T) {
	root := copyContent(t)
	reg := filepath.Join(root, "registers", "sources.yaml")
	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"hf-copy", "hf-unread"} {
		b = append(b, []byte(`- id: `+id+`
  name: A copy
  kind: dataset
  origin: D
  derived: true
  url: https://huggingface.co/datasets/a/`+id+`
  access: hf
  terms: No licence is stated.
  zone: {statement: Z2, solution: Z2}
  ai_train: unknown
  wave: 0
  state: planned
`)...)
	}
	if err := os.WriteFile(reg, b, 0o644); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	matchRows(t, dir, "hf-copy")
	matchRows(t, dir, "imo-official")
	code, out, errs := run("matchrate", "-content", root, "-seed", dir, "-derived")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "hf-copy: 4 rows, 2 matched") || strings.Contains(out, "imo-official:") {
		t.Errorf("output: %s", out)
	}
	cov := filepath.Join(root, "reports", "coverage")
	md, err := os.ReadFile(filepath.Join(cov, "derived.md"))
	for _, want := range []string{"| hf-copy | 4 | 2 | 1 | 0 | 0 | 2 | 50.00% | 2 | 5000 | 1 |", "1 of 1 datasets give 100 or more", "no seed rows yet, so they are not in the table: hf-unread."} {
		if err != nil || !strings.Contains(string(md), want) {
			t.Errorf("derived.md has no %q: %v\n%s", want, err, md)
		}
	}
	for _, f := range []string{"derived.json", "derived-new.tsv"} {
		if _, err := os.Stat(filepath.Join(cov, f)); err != nil {
			t.Error(err)
		}
	}
	if _, err := os.Stat(filepath.Join(cov, "match.md")); err == nil {
		t.Error("om matchrate -derived wrote match.md")
	}
}
