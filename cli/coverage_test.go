package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// om coverage reads the gap register and the tiers, and prints C0 against the allowed positions of the tier 1 nationals.
func TestCoverageGaps(t *testing.T) {
	root := copyContent(t)
	reg := filepath.Join(root, "registers", "competitions.yaml")
	b, err := os.ReadFile(reg)
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, []byte(`
- code: xx-mo
  name: X Mathematical Olympiad
  group: national
  level: high-school
  country: CN
  organizer: X
  url: https://example.org/
  first_year: 2020
  per_year: 3
  edition_form: year
  sources: [imo-official]
  description: A test olympiad.
`)...)
	files := map[string][]byte{
		reg: b,
		filepath.Join(root, "registers", "site-tiers.yaml"): []byte("groups: [international]\ncountries: [CN]\ncodes: []\n"),
		filepath.Join(root, "registers", "gaps.yaml"):       []byte("- competition: xx-mo\n  years: [2020-2021]\n  reason: no-source\n  note: No paper of these years was found.\n"),
	}
	for p, b := range files {
		if err := os.WriteFile(p, b, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	code, out, errs := run("coverage", "-content", root, "-seed", t.TempDir(), "-year", "2023")
	if code != ExitOK {
		t.Fatalf("exit %d: %s", code, errs)
	}
	if want := "Tier 1 nationals: C0 0 of 12 (0.0%), 0 of 6 allowed positions (C0 allowed 0.0%). No record: 0 link-only, 0 robots.txt, 6 no allowed source, 6 open."; !strings.Contains(out, want) {
		t.Errorf("output: %s", out)
	}
	md, err := os.ReadFile(filepath.Join(root, "reports", "coverage", "coverage.md"))
	if err != nil || !strings.Contains(string(md), "| xx-mo | 2020-2021 | all | no-source | 6 | No paper of these years was found. |") {
		t.Errorf("coverage.md: %v\n%s", err, md)
	}
	bad := filepath.Join(root, "registers", "gaps.yaml")
	if err := os.WriteFile(bad, []byte("- {competition: nope, reason: robots, note: x}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if code, _, errs := run("coverage", "-content", root, "-seed", t.TempDir()); code == ExitOK || !strings.Contains(errs, "not in the competition register") {
		t.Errorf("a bad gap register gave exit %d: %s", code, errs)
	}
}
