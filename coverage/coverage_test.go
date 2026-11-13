package coverage

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
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
	if err := os.MkdirAll(filepath.Join(root, "registers"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, seed.AliasFile), []byte("Some Olympiad\tnope\n"), 0o600); err != nil {
		t.Fatal(err)
	}
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
	rep, err := Build(c, a, nil, 2026, func(fn func(seed.Row) error) error {
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

// A problem that also appears in another edition fills its position there, so the position is not counted as missing.
func TestBuildAlso(t *testing.T) {
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../content/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	ed := "year: 2023\npapers:\n  - name: Day 1\n    problems: [p1, p2, p3]\n"
	if err := os.WriteFile(filepath.Join(root, "editions/imo/2023.yaml"), []byte(ed), 0o600); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, "problems/imo/2024/p1.md")
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	b = []byte(strings.Replace(string(b), "    primary: true\n", "    primary: true\n  - at: imo/2023/p3\n", 1))
	if err := os.WriteFile(p, b, 0o600); err != nil {
		t.Fatal(err)
	}
	c, fs := content.Load(root)
	if len(fs) > 0 {
		t.Fatalf("load: %v", fs)
	}
	rep, err := Build(c, aliases(t, c, ""), nil, 2026, func(func(seed.Row) error) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	for _, y := range rep.Competitions[0].Years {
		if y.Year == 2023 && (len(y.Record) != 1 || y.Record[0] != "p3") {
			t.Errorf("2023 %+v", y)
		}
	}
}

// amc is a competition with two papers from 2002 and a fall part in 2021, as the AMC 10.
func amc() *content.Competition {
	return &content.Competition{
		Code: "amc10", Name: "AMC 10", FirstYear: 2000, LastYear: 2022, PerYear: 2,
		EditionForm: "year-letter",
		Papers: []content.Papers{
			{From: 2002, Letters: []string{"a", "b"}},
			{From: 2021, To: 2021, Part: "fall", Letters: []string{"a", "b"}},
		},
	}
}

func TestPaperOf(t *testing.T) {
	c := amc()
	cases := []struct {
		year   int
		labels []string
		want   string
		known  bool
	}{
		{2001, []string{"AMC 10"}, "2001", true},
		{2010, []string{"AMC 10A"}, "2010a", true},
		{2010, []string{"AMC 10 B"}, "2010b", true},
		{2010, []string{"AMC 10"}, "", false},
		{2021, []string{"AMC 10 B", "Fall"}, "2021-fall-b", true},
		{2021, []string{"AMC 10A"}, "2021a", true},
		{2010, []string{"AMC 10 A", "Fall"}, "", false},
		{2001, []string{"AMC 10"}, "2001", true},
		{2010, []string{"AMC 10C"}, "", false},
	}
	for _, tc := range cases {
		got, known := paperOf(c, tc.year, tc.labels)
		if got != tc.want || known != tc.known {
			t.Errorf("paperOf(%d, %q) = %q, %v, want %q, %v", tc.year, tc.labels, got, known, tc.want, tc.known)
		}
	}
	if got, known := paperOf(&content.Competition{Code: "imo"}, 2024, nil); got != "2024" || !known {
		t.Errorf("one paper gave %q, %v", got, known)
	}
}

func TestMeasurePapers(t *testing.T) {
	c := amc()
	cand := map[string]map[string]bool{
		"2001":        {"p1": true},
		"2010b":       {"p2": true},
		"2021-fall-a": {"p1": true},
	}
	seen := map[string]int{"2001": 1, "2010b": 1, "2021-fall-a": 3}
	out := measure(c, cand, seen, map[int]int{2010: 4}, 9, 2026, nil)
	// Two years with one paper, 21 years with the papers a and b, and the two fall papers of 2021: 2 + 42 + 2 papers of 2 positions.
	if out.Expected != 92 || out.Candidate != 3 || out.NoCandidate != 89 {
		t.Errorf("expected %d, candidate %d, none %d", out.Expected, out.Candidate, out.NoCandidate)
	}
	var eds []string
	years := map[string]*Year{}
	for _, y := range out.Years {
		if y.Year == 2001 || y.Year == 2010 || y.Year == 2021 {
			eds = append(eds, y.Edition)
		}
		years[strconv.Itoa(y.Year)+y.Edition] = y
	}
	if got := strings.Join(eds, " "); got != " 2010a 2010b 2021a 2021b 2021-fall-a 2021-fall-b" {
		t.Errorf("editions %q", got)
	}
	if y := years["20102010a"]; y.Rows != 4 || len(y.Candidate) != 0 {
		t.Errorf("2010a %+v", y)
	}
	if y := years["20102010b"]; y.Rows != 1 || len(y.Candidate) != 1 || y.Candidate[0] != "p2" {
		t.Errorf("2010b %+v", y)
	}
	if y := years["20212021-fall-a"]; y.Rows != 3 || len(y.Candidate) != 1 {
		t.Errorf("2021-fall-a %+v", y)
	}
}
