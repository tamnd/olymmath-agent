package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// calibrationWork puts the IMO-GradingBench fixture where om calibrate reads it.
func calibrationWork(t *testing.T) string {
	t.Helper()
	work := t.TempDir()
	b, err := os.ReadFile("../calibrate/testdata/gradingbench.csv")
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(work, "calibration", "gradingbench")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gradingbench.csv"), b, 0o600); err != nil {
		t.Fatal(err)
	}
	return work
}

func TestCalibrateGrades(t *testing.T) {
	work := calibrationWork(t)
	grades := filepath.Join(work, "grades.jsonl")
	lines := `{"id": "gradingbench/GB-0001", "score": 7}
{"id": "gradingbench/GB-0002", "score": 0}
{"id": "gradingbench/GB-0003", "score": 5}
{"id": "gradingbench/GB-0004", "score": 3}
`
	if err := os.WriteFile(grades, []byte(lines), 0o600); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(work, "calibration.md")
	code, _, errs := run("calibrate", "-set", "gradingbench", "-work", work, "-grades", grades, "-out", out)
	if code != ExitOK {
		t.Fatalf("code %d, errs %q", code, errs)
	}
	b, _ := os.ReadFile(out)
	if !strings.Contains(string(b), "| IMO-GradingBench | CC-BY-4.0 | 4 | 50.0% | 1.000 |") {
		t.Fatalf("the report is\n%s", b)
	}
	if code, _, errs := run("calibrate", "-set", "proofbench", "-work", work, "-grades", grades); code != ExitUsage || !strings.Contains(errs, "om calibrate fetch") {
		t.Fatalf("a missing set gives %d, %q", code, errs)
	}
}

func TestCalibrateJury(t *testing.T) {
	f := fakeModel(t)
	work := calibrationWork(t)
	code, out, _ := run("calibrate", "-set", "gradingbench", "-work", work, "-dry-run")
	if code != ExitOK || !strings.Contains(out, "4 proofs to grade, about 8 calls") || f.Calls() != 0 {
		t.Fatalf("the dry run gives %d, %q", code, out)
	}
	// The fake jury gives 7 to every proof, so it does not reach the target.
	code, out, errs := run("calibrate", "-set", "gradingbench", "-work", work, "-budget", "1")
	if code != ExitFindings || f.Calls() != 8 || !strings.Contains(out, "4 proofs graded") || !strings.Contains(out, "| no |") {
		t.Fatalf("code %d, %d calls, out %q, errs %q", code, f.Calls(), out, errs)
	}
}
