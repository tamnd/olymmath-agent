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

func TestWrongList(t *testing.T) {
	ibero := &content.Competition{Code: "ibero", Name: "Iberoamerican Mathematical Olympiad"}
	isl := &content.Competition{Code: "isl", Name: "IMO Shortlist"}
	cases := []struct {
		comp  *content.Competition
		label string
		wrong bool
		rng   bool
	}{
		{ibero, "IberoAmerican Shortlist 1992 - 2010 - Geometry", true, true},
		{ibero, "IberoAmerican Shortlist 2011+ Geometry", true, true},
		{ibero, "2000 IberoAmerican", false, false},
		{isl, "2007 IMO Shortlist", false, false},
		{isl, "IMO Shortlist 1959-1966", false, true},
	}
	for _, c := range cases {
		if got := wrongList(c.comp, c.label); got != c.wrong {
			t.Errorf("wrongList(%s, %q) = %v", c.comp.Code, c.label, got)
		}
		if got := yearRangeRE.MatchString(c.label); got != c.rng {
			t.Errorf("year range in %q: %v", c.label, got)
		}
	}
}

func TestImportLoosePolish(t *testing.T) {
	text := "Prove that for every positive integer $n$ the number $n^{5}-n$ is divisible by $30$, and find all $n$ for which it is divisible by $60$."
	rows := []seed.Row{
		{Labels: []string{"Other"}, Statement: "XXXVIII OM - I - Problem 11\n\n" + text}, // new, unplaced pl-om/1987
		{Labels: []string{"Other"}, Statement: "XXXVIII OM - III - Problem 4\n\nLet $a, b, c$ be positive reals with $a+b+c=1$. Prove that $a^2+b^2+c^2 \\ge 1/3$ and find when it is equal."},
	}
	root, dir := looseFixture(t, rows)
	reg := "\n- code: pl-om\n  name: Polish Mathematical Olympiad\n  group: national\n  level: high-school\n  country: PL\n  url: https://om.sem.edu.pl/problems/\n  first_year: 1950\n  per_year: 6\n  sources: [hf-test]\n  description: The Polish olympiad.\n"
	f, err := os.OpenFile(filepath.Join(root, "registers", "competitions.yaml"), os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(reg); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	code, _, errs := run("import", "-loose", "-content", root, "-seed", dir, "-sources", "hf-test")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(errs, "1 new problems from 1 rows") || !strings.Contains(errs, "has a position") {
		t.Errorf("the counts:\n%s", errs)
	}
	c, fs := content.Load(root)
	if fs.Errors() > 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	var p *content.Problem
	for _, x := range c.Loose {
		if x.Statement == text {
			p = x
		}
	}
	if p == nil {
		t.Fatalf("no loose problem with the text; loose: %d", len(c.Loose))
	}
	if len(p.Unplaced) != 1 || p.Unplaced[0].At != "pl-om/1987" || p.Unplaced[0].Label != "XXXVIII OM - I - Problem 11" {
		t.Errorf("unplaced: %+v", p.Unplaced)
	}
}

func TestImportLooseBlockLink(t *testing.T) {
	floor := "Determine all real numbers $\\alpha$ such that, for every positive integer $n$, the integer\n$$\\lfloor \\alpha \\rfloor + \\lfloor 2\\alpha \\rfloor + \\cdots + \\lfloor n\\alpha \\rfloor$$\nis a multiple of $n$."
	root, dir := looseFixture(t, []seed.Row{{Labels: []string{"Some forum"}, Statement: "Problem 3. " + floor}})
	code, _, errs := run("import", "-loose", "-content", root, "-seed", dir, "-sources", "hf-test", "-block", "imo")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(errs, "dropped, same as a problem of a link-only competition: 1") || !strings.Contains(errs, "0 rows link 0 problems") {
		t.Errorf("the counts:\n%s", errs)
	}
	c, fs := content.Load(root)
	if fs.Errors() > 0 {
		t.Fatalf("findings:\n%v", fs)
	}
	if p := c.ByTag("0HVHSDNNMY"); p == nil || len(p.Statements) != 1 {
		t.Errorf("imo/2024/p1 got a statement: %+v", p)
	}
}

func TestImportLooseRemoved(t *testing.T) {
	text := "Prove that for any integer $n$, among the numbers $n, n+1, n+2, n+3, n+4$ there is at least one number that is coprime with the other four of these numbers."
	root, dir := looseFixture(t, []seed.Row{{Labels: []string{"Some forum"}, Statement: text}, {Labels: []string{"Some forum"}, Statement: strings.Replace(text, "five", "5", 1) + " Find all such $n$."}})
	reg := "# capture, reason\nhf-test/train.parquet#row=0\tnot one clear problem\n"
	if err := os.WriteFile(filepath.Join(root, RemovedFile), []byte(reg), 0o600); err != nil {
		t.Fatal(err)
	}
	code, _, errs := run("import", "-loose", "-content", root, "-seed", dir, "-sources", "hf-test", "-dry")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if !strings.Contains(errs, "dropped, removed by a review: 1") || !strings.Contains(errs, "1 new problems from 1 rows") {
		t.Errorf("the counts:\n%s", errs)
	}
	if err := os.WriteFile(filepath.Join(root, RemovedFile), []byte("hf-test/train.parquet#row=0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := run("import", "-loose", "-content", root, "-seed", dir, "-sources", "hf-test", "-dry"); code == ExitOK || !strings.Contains(errs, "want a capture, a tab and a reason") {
		t.Errorf("a line with no reason: exit %d: %s", code, errs)
	}
}
