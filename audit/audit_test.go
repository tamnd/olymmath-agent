package audit

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

const valid = "../content/testdata/valid"

// fixture copies the valid test content to a new directory and applies the edits. An edit maps a file to a list of pairs: each old text is replaced with the new text. A file with the old text "" gets the new text as its full content.
func fixture(t *testing.T, edits map[string][][2]string) string {
	t.Helper()
	dir := t.TempDir()
	err := filepath.WalkDir(valid, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(valid, p)
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
	for rel, pairs := range edits {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		b, _ := os.ReadFile(full)
		s := string(b)
		for _, pr := range pairs {
			if pr[0] == "" {
				s = pr[1]
				continue
			}
			if !strings.Contains(s, pr[0]) {
				t.Fatalf("%s has no %q", rel, pr[0])
			}
			s = strings.Replace(s, pr[0], pr[1], 1)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(s), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

func rules(fs content.Findings) map[string]bool {
	m := map[string]bool{}
	for _, f := range fs {
		m[f.Rule] = true
	}
	return m
}

func TestValidHasNoFindings(t *testing.T) {
	_, fs, err := Run(valid)
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Fatal(fs)
	}
}

const p2 = "problems/imo/2024/p2.md"
const sol = "solutions/imo/2024/p1/006SZE6NAS.md"

func TestRules(t *testing.T) {
	cases := []struct {
		rule  string
		edits map[string][][2]string
	}{
		{"T02", map[string][][2]string{"registers/tags.tsv": {{"03JXB69J3N\tproblem\n", ""}}}},
		{"T02", map[string][][2]string{"registers/tags.tsv": {{"006SZE6NAS\tsolution", "006SZE6NAS\tsolution\n0000000000\tproblem"}}}},
		{"T03", map[string][][2]string{"registers/redirects.tsv": {{"imo/2024/q1\timo/2024/p1", "imo/2024/q1\timo/2024/p9"}}}},
		{"T04", map[string][][2]string{p2: {{"  - at: imo/2024/p2\n    primary: true", "  - at: imo/2024/p2\n    primary: true\n  - at: imo/2024/p1"}}}},
		{"S01", map[string][][2]string{p2: {{"rule: official-original", "rule: best"}}}},
		{"S02", map[string][][2]string{p2: {{"origin: H1", "origin: M"}}}},
		{"S03", map[string][][2]string{p2: {{"source: imo-official", "source: nowhere"}}}},
		{"S04", map[string][][2]string{p2: {{`\lfloor 2\alpha \rfloor`, `\frac{2\alpha`}}}},
		{"S05", map[string][][2]string{p2: {{"origin: H1", "origin: D"}}}},
		{"S06", map[string][][2]string{p2: {
			{"    kind: official\n", "    kind: official\n  - id: S2\n    lang: en\n    source: imo-official\n    origin: H1\n    zone: Z2\n    kind: transcription\n"},
			{"is a multiple of $n$.", "is a multiple of $n$.\n<!-- S2 -->\nDetermine all real numbers $\\alpha$ such that $\\lfloor 3\\alpha \\rfloor$ is a multiple of $n$."},
		}}},
		{"S07", map[string][][2]string{p2: {{"zone: Z2", "zone: Z3"}}}},
		{"S08", map[string][][2]string{p2: {{"is a multiple of $n$.", "is a multiple of $n$.\n\n![The figure](figure.svg)"}}}},
		{"L02", map[string][][2]string{sol: {{"origin: human", "origin: ai"}}}},
		{"L03", map[string][][2]string{sol: {{"complete: true", "complete: true\nrun:\n  model: claude-opus-5-5"}}}},
		{"L04", map[string][][2]string{sol: {{"zone: Z1", "zone: Z3"}}}},
		{"X01", map[string][][2]string{"registers/sources.yaml": {{"  terms: Copyright of the International Mathematical Olympiad. No licence is stated.\n", ""}}}},
		{"X02", map[string][][2]string{"registers/sources.yaml": {{"  state: active", "  state: active\n  do_not_republish: true"}}}},
	}
	for _, tc := range cases {
		t.Run(tc.rule, func(t *testing.T) {
			_, fs, err := Run(fixture(t, tc.edits))
			if err != nil {
				t.Fatal(err)
			}
			if !rules(fs)[tc.rule] {
				t.Errorf("want a finding of %s, got %v", tc.rule, fs)
			}
		})
	}
}

func TestOneRule(t *testing.T) {
	dir := fixture(t, map[string][][2]string{p2: {{"origin: H1", "origin: D"}}})
	_, fs, err := Run(dir, "S02")
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 0 {
		t.Errorf("S02 alone: want no finding, got %v", fs)
	}
	if _, _, err := Run(dir, "Z99"); err == nil {
		t.Error("want an error for an unknown rule")
	}
}

func TestWarningIsNotError(t *testing.T) {
	_, fs, err := Run(fixture(t, map[string][][2]string{p2: {{"origin: H1", "origin: D"}}}))
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) == 0 || fs.Errors() != 0 {
		t.Errorf("want only warnings, got %v", fs)
	}
}

func TestEditionYear(t *testing.T) {
	for id, want := range map[string]int{"2024": 2024, "2024-spring": 2024, "24": 0, "x2024": 0} {
		if got := editionYear(id); got != want {
			t.Errorf("editionYear(%q) = %d, want %d", id, got, want)
		}
	}
}
