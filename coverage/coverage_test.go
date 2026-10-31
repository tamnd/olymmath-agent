package coverage

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
)

func corpus(t *testing.T) *content.Corpus {
	t.Helper()
	c, fs := content.Load("../content/testdata/valid")
	if len(fs) > 0 {
		t.Fatalf("load: %v", fs)
	}
	return c
}

func aliases(t *testing.T, c *content.Corpus, table string) *seed.Aliases {
	t.Helper()
	root := t.TempDir()
	if table != "" {
		p := filepath.Join(root, seed.AliasFile)
		if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(table), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	a, err := seed.LoadAliases(root, c)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func TestAliases(t *testing.T) {
	c := corpus(t)
	a := aliases(t, c, "# comment\nIMO-SL\t-\nInternationale Mathematik-Olympiade\timo\n")
	cases := map[string]string{
		"International Mathematical Olympiad 2024": "imo",
		"IMO":                                 "imo",
		"Internationale Mathematik-Olympiade": "imo",
		"IMO SL":                              "-",
		"Baltic Way":                          "",
	}
	for l, want := range cases {
		if got := a.Code([]string{l}); got != want {
			t.Errorf("Code(%q) = %q, want %q", l, got, want)
		}
	}
	root := t.TempDir()
	os.MkdirAll(filepath.Join(root, "registers"), 0o750)
	os.WriteFile(filepath.Join(root, seed.AliasFile), []byte("Some Olympiad\tnope\n"), 0o600)
	if _, err := seed.LoadAliases(root, c); err == nil || !strings.Contains(err.Error(), "not in the competition register") {
		t.Errorf("an unknown code gave %v", err)
	}
}

func TestBuild(t *testing.T) {
	c := corpus(t)
	a := aliases(t, c, "numina:cn_k12\t-\n")
	rows := []seed.Row{
		{Source: "a", Labels: []string{"IMO"}, Year: 2024, Number: "P2", Statement: "x"},
		{Source: "a", Labels: []string{"IMO"}, Year: 2023, Number: "3", Statement: "x"},
		{Source: "a", Labels: []string{"IMO"}, Statement: "no year"},
		{Source: "b", Labels: []string{"Baltic Way"}, Year: 2001, Statement: "x"},
		{Source: "b", Labels: []string{"Baltic Way"}, Year: 2002, Statement: "x"},
		{Source: "b", Labels: []string{"numina:cn_k12"}, Statement: "x"},
	}
	rep, err := Build(c, a, 2026, func(fn func(seed.Row) error) error {
		for _, r := range rows {
			if err := fn(r); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Totals.Expected != 2 || rep.Totals.C0 != 2 || rep.Totals.Rows != 3 {
		t.Errorf("totals %+v", rep.Totals)
	}
	if len(rep.Unknown) != 1 || rep.Unknown[0].Label != "Baltic Way" || rep.Unknown[0].Rows != 2 {
		t.Errorf("unknown %+v", rep.Unknown)
	}
	src := map[string]*Source{}
	for _, s := range rep.Sources {
		src[s.ID] = s
	}
	if src["a"].Located != 3 || src["a"].Positioned != 2 || src["b"].Excluded != 1 {
		t.Errorf("sources a %+v, b %+v", src["a"], src["b"])
	}
	var y2023 *Year
	for _, y := range rep.Competitions[0].Years {
		if y.Year == 2023 {
			y2023 = y
		}
	}
	if y2023 == nil || len(y2023.Candidate) != 1 || y2023.Candidate[0] != "p3" {
		t.Errorf("2023 %+v", y2023)
	}
	var md, js, tsv bytes.Buffer
	if err := rep.Markdown(&md, 10); err != nil || !strings.Contains(md.String(), "International Mathematical Olympiad") {
		t.Errorf("markdown %v", err)
	}
	if err := rep.JSON(&js); err != nil || !strings.Contains(js.String(), `"c0": 2`) && !strings.Contains(js.String(), `"c0":2`) {
		t.Errorf("json %v", err)
	}
	if err := rep.UnknownTSV(&tsv); err != nil || !strings.Contains(tsv.String(), "Baltic Way") {
		t.Errorf("tsv %v %q", err, tsv.String())
	}
}
