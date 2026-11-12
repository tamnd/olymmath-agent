package cli

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
)

// looseFixture copies the valid content, adds a dataset source and writes its seed rows.
func looseFixture(t *testing.T, rows []seed.Row) (root, dir string) {
	t.Helper()
	root, dir = t.TempDir(), t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../content/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	src := "\n- id: hf-test\n  name: A test dataset\n  kind: dataset\n  origin: D\n  url: https://huggingface.co/datasets/test\n  access: hf\n  zone: {statement: Z2, solution: Z2}\n  ai_train: allowed\n  wave: 1\n  state: active\n"
	f, err := os.OpenFile(filepath.Join(root, "registers", "sources.yaml"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(src); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := os.Create(seed.File(dir, "hf-test"))
	if err != nil {
		t.Fatal(err)
	}
	gz := gzip.NewWriter(out)
	enc := json.NewEncoder(gz)
	for i, r := range rows {
		r.Source, r.File, r.Row = "hf-test", "train.parquet", i
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	if err := out.Close(); err != nil {
		t.Fatal(err)
	}
	return root, dir
}

func TestImportLoose(t *testing.T) {
	floor := "Determine all real numbers $\\alpha$ such that, for every positive integer $n$, the integer\n$$\\lfloor \\alpha \\rfloor + \\lfloor 2\\alpha \\rfloor + \\cdots + \\lfloor n\\alpha \\rfloor$$\nis a multiple of $n$."
	newText := "Let $p$ be an odd prime. Prove that the sum $1^{p-2} + 2^{p-2} + \\cdots + (p-1)^{p-2}$ is divisible by $p$ for every such $p$."
	rows := []seed.Row{
		{Labels: []string{"IMO"}, Year: 2024, Number: "1", Statement: floor},                      // 0: has a position
		{Labels: []string{"Some forum"}, Statement: "Problem 3. " + floor, Ref: "aops-topic:123"}, // 1: links imo/2024/p1
		{Labels: []string{"IMO"}, Year: 2023, Statement: newText, Lang: "en"},                     // 2: new, unplaced imo/2023
		{Labels: []string{"Other"}, Statement: newText},                                           // 3: the same as row 2
		{Labels: []string{"Other"}, Statement: "Solution. By Fermat we are done, as the sum of the inverses is zero."},
		{Labels: []string{"Other"}, Statement: "In the figure below, $ABCD$ is a square of side $1$. Find the area of the shaded region."},
	}
	root, dir := looseFixture(t, rows)
	code, _, errs := run("import", "-loose", "-content", root, "-seed", dir, "-sources", "hf-test")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(errs, "1 new problems from 2 rows, 1 rows link 1 problems") {
		t.Errorf("the counts:\n%s", errs)
	}
	c, fs := content.Load(root)
	if fs.Errors() > 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	p1 := c.ByTag("0HVHSDNNMY")
	if p1 == nil || len(p1.Statements) != 2 || p1.Statements[1].Source != "hf-test" || p1.Statements[1].Capture != "hf-test/train.parquet#row=1" || p1.Statements[1].URL != "https://artofproblemsolving.com/community/c6h123" {
		t.Fatalf("imo/2024/p1: %+v", p1)
	}
	var got *content.Problem
	for _, p := range c.Loose {
		if p.Statement == newText {
			got = p
		}
	}
	if got == nil {
		t.Fatalf("no new loose problem with the text; loose: %d", len(c.Loose))
	}
	if len(got.Unplaced) != 1 || got.Unplaced[0].At != "imo/2023" || got.Unplaced[0].Label != "IMO" {
		t.Errorf("unplaced: %+v", got.Unplaced)
	}
	if len(got.Statements) != 1 || got.Statements[0].Capture != "hf-test/train.parquet#row=2" || got.Canonical.Rule != "derived" {
		t.Errorf("statements: %+v %+v", got.Statements, got.Canonical)
	}
	// A second run takes no row again.
	code, _, errs = run("import", "-loose", "-content", root, "-seed", dir, "-sources", "hf-test")
	if code != ExitOK || !strings.Contains(errs, "2 already imported") || !strings.Contains(errs, "0 new problems") {
		t.Errorf("second run, exit %d:\n%s", code, errs)
	}
}
