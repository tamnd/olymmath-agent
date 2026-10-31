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

func TestMatchrate(t *testing.T) {
	root := copyContent(t)
	dir := t.TempDir()
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
	err := seed.WriteFile(dir, "imo-official", func(w io.Writer) error {
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
	code, out, errs := run("matchrate", "-content", root, "-seed", dir, "imo-official")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "4 rows, 2 matched, 0 near, 2 new (50.00%), 1 new with a competition") {
		t.Errorf("output: %s", out)
	}
	md, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "match.md"))
	if err != nil || !strings.Contains(string(md), "| imo-official | 4 | 2 | 1 | 0 | 2 | 50.00% | 1 |") {
		t.Errorf("match.md: %v\n%s", err, md)
	}
	tsv, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "match-new.tsv"))
	if err != nil || !strings.Contains(string(tsv), "imo-official\t2023\t6\tIMO\t") || strings.Contains(string(tsv), "Wombats") {
		t.Errorf("match-new.tsv: %v\n%s", err, tsv)
	}
}
