package cli

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/tag"
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
		// The text of the loose record 0J8TP3MFS9: new against the placed records, matched against all the records.
		{Source: "imo-official", Labels: []string{"Unknown Cup"}, Statement: "Prove that $a^2 + b^2 \\geq 2ab$ for all real numbers $a$ and $b$."},
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
	if !strings.Contains(out, "imo-official: 5 rows. Against the placed records: 2 matched (1 by key, 0 by link), 0 variants, 0 near, 3 new (60.00%), 3 new problems (6000 for each 10,000 rows), 1 new with a competition. Against all the records: 3 matched (1 by key, 0 by link), 0 variants, 0 near, 2 new (40.00%), 2 new problems (4000 for each 10,000 rows), 1 new with a competition.") {
		t.Errorf("output: %s", out)
	}
	md, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "match.md"))
	for _, want := range []string{"| imo-official | 5 | 2 | 1 | 0 | 0 | 0 | 3 | 60.00% | 3 | 6000 | 1 |", "| imo-official | 5 | 3 | 1 | 0 | 0 | 0 | 2 | 40.00% | 2 | 4000 | 1 |", "| imo-official | 5 | 6000 | 4000 |", "| dataset not read by the loose import | 2 | 2 |"} {
		if err != nil || !strings.Contains(string(md), want) {
			t.Errorf("match.md has no %q: %v\n%s", want, err, md)
		}
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
	if !strings.Contains(out, "hf-copy: 5 rows. Against the placed records: 2 matched") || strings.Contains(out, "imo-official:") {
		t.Errorf("output: %s", out)
	}
	cov := filepath.Join(root, "reports", "coverage")
	md, err := os.ReadFile(filepath.Join(cov, "derived.md"))
	for _, want := range []string{"| hf-copy | 5 | 3 | 1 | 0 | 0 | 0 | 2 | 40.00% | 2 | 4000 | 1 |", "1 of 1 datasets give 100 or more new problems for each 10,000 rows against all the records", "no seed rows yet, so they are not in the tables: hf-unread."} {
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

// TestMatchrateLoose covers a row that an import linked to a loose record, and the reasons of the new rows of a source of the loose import.
func TestMatchrateLoose(t *testing.T) {
	root := copyContent(t)
	reg := filepath.Join(root, "registers", "sources.yaml")
	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, []byte(`- id: hf-numinamath-1-5
  name: NuminaMath
  kind: dataset
  origin: D
  derived: true
  url: https://huggingface.co/datasets/a/numina
  access: hf
  terms: No licence is stated.
  zone: {statement: Z2, solution: Z2}
  ai_train: unknown
  wave: 0
  state: planned
`)...)
	if err := os.WriteFile(reg, b, 0o644); err != nil {
		t.Fatal(err)
	}
	// A loose record that an import made from row 7. Its text is not the text of the row, so only the link matches them.
	tg := "1AB2C3D4E" + string(tag.Check("1AB2C3D4E"))
	loose := filepath.Join(root, "problems", "_loose", tg[:3], tg+".md")
	if err := os.MkdirAll(filepath.Dir(loose), 0o750); err != nil {
		t.Fatal(err)
	}
	rec := `---
tag: ` + tg + `
kind: problem
canonical:
  statement: S1
  rule: derived
statements:
  - id: S1
    lang: en
    source: hf-numinamath-1-5
    url: https://huggingface.co/datasets/a/numina
    capture: hf-numinamath-1-5/data/train.parquet#row=7
    origin: D
    zone: Z2
    kind: transcription
status:
  statement: captured
---
Find all primes $p$ such that $p^2 + 2$ is also a prime number.
`
	if err := os.WriteFile(loose, []byte(rec), 0o600); err != nil {
		t.Fatal(err)
	}
	removed := "# capture\treason\nhf-numinamath-1-5/data/train.parquet#row=9\tnot one clear problem\n"
	if err := os.WriteFile(filepath.Join(root, "registers", "loose-removed.tsv"), []byte(removed), 0o600); err != nil {
		t.Fatal(err)
	}
	rows := []seed.Row{
		{Row: 7, Labels: []string{"numina:olympiads"}, Statement: "Determine every prime whose square plus two is again a prime."},
		{Row: 8, Labels: []string{"numina:cn_k12"}, Statement: "Kangaroos jump along hexagonal lattices until purple tulips bloom in the meadow."},
		{Row: 9, Labels: []string{"numina:olympiads"}, Statement: "Wombats stack copper teacups beside quiet volcanoes at night."},
		{Row: 10, Labels: []string{"numina:olympiads"}, Statement: "Let $x$ and $y$ be positive reals with $x + y = 1$. Prove that $x y \\le 1/4$ and find when equality holds."},
	}
	dir := t.TempDir()
	err = seed.WriteFile(dir, "hf-numinamath-1-5", func(w io.Writer) error {
		enc := json.NewEncoder(w)
		for _, r := range rows {
			r.Source, r.File = "hf-numinamath-1-5", "data/train.parquet"
			if err := enc.Encode(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	code, out, errs := run("matchrate", "-content", root, "-seed", dir, "-derived", "hf-numinamath-1-5")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(out, "Against the placed records: 0 matched (0 by key, 0 by link), 0 variants, 0 near, 4 new") || !strings.Contains(out, "Against all the records: 1 matched (0 by key, 1 by link), 0 variants, 0 near, 3 new") {
		t.Errorf("output: %s", out)
	}
	md, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "derived.md"))
	for _, want := range []string{"| not an olympiad set | 1 | 1 |", "| removed by a review | 1 | 1 |", "| passed the checks | 1 | 1 |"} {
		if err != nil || !strings.Contains(string(md), want) {
			t.Errorf("derived.md has no %q: %v\n%s", want, err, md)
		}
	}
	var rates []MatchRate
	js, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "derived.json"))
	if err == nil {
		err = json.Unmarshal(js, &rates)
	}
	if err != nil || len(rates) != 1 || rates[0].All.ByLink != 1 || rates[0].Placed.New != 4 || rates[0].All.Why["passed the checks"] != 1 {
		t.Errorf("derived.json: %v %+v", err, rates)
	}
}

func TestShownText(t *testing.T) {
	corpus, _ := content.Load(copyContent(t))
	row := seed.Row{Source: "imo-official", Statement: "Find all n.\nShow it."}
	if got := shownText(corpus, "imo", row); got != oneLine(row.Statement) {
		t.Errorf("imo: %q", got)
	}
	if got := shownText(corpus, "kr-kmo", row); got != "-" {
		t.Errorf("kr-kmo: %q", got)
	}
	if got := shownText(corpus, "jp-jmo", row); got != "-" {
		t.Errorf("jp-jmo: %q", got)
	}
	bmo := seed.Row{Source: "imo-official", Statement: "Find all n. (1994 British Mathematical Olympiad)"}
	if got := shownText(corpus, "", bmo); got != "-" {
		t.Errorf("bmo: %q", got)
	}
	ru := seed.Row{Source: problemsRu, Statement: "x"}
	if got := shownText(corpus, "", ru); got != "-" {
		t.Errorf("problems-ru: %q", got)
	}
}
