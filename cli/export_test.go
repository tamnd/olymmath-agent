package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExportHF(t *testing.T) {
	out := filepath.Join(t.TempDir(), "hf")
	code, stdout, errs := run("export", "hf", "-content", "../content/testdata/valid", "-out", out)
	if code != ExitOK || !strings.Contains(stdout, "z1-text 1, z2-facts 2, z3-links 0") {
		t.Fatalf("%d %s %s", code, stdout, errs)
	}
	for _, rel := range []string{"README.md", "z1-text/train.parquet", "z2-facts/train.parquet", "z3-links/train.parquet"} {
		if _, err := os.Stat(filepath.Join(out, rel)); err != nil {
			t.Errorf("no %s", rel)
		}
	}
}

// TestPublishHFRefuses wants om publish hf to push nothing without -confirm and without HF_TOKEN.
func TestPublishHFRefuses(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the fake hf program is a shell script")
	}
	dir := filepath.Join(t.TempDir(), "hf")
	if code, _, errs := run("export", "hf", "-content", "../content/testdata/valid", "-out", dir); code != ExitOK {
		t.Fatal(errs)
	}
	bin := t.TempDir()
	log := filepath.Join(bin, "args")
	script := "#!/bin/sh\necho \"$@\" > " + log + "\n"
	if err := os.WriteFile(filepath.Join(bin, "hf"), []byte(script), 0o700); err != nil { // #nosec G306 -- the test needs a program.
		t.Fatal(err)
	}
	old := hfCommand
	hfCommand = filepath.Join(bin, "hf")
	t.Cleanup(func() { hfCommand = old })

	t.Setenv("HF_TOKEN", "fake")
	if code, _, errs := run("publish", "hf", "-dir", dir); code != ExitUsage || !strings.Contains(errs, "-confirm") {
		t.Errorf("no -confirm: %d %s", code, errs)
	}
	t.Setenv("HF_TOKEN", "")
	if code, _, errs := run("publish", "hf", "-dir", dir, "-confirm"); code != ExitUsage || !strings.Contains(errs, "HF_TOKEN") {
		t.Errorf("no token: %d %s", code, errs)
	}
	if _, err := os.Stat(log); err == nil {
		t.Fatal("om publish hf ran the upload without -confirm and a token")
	}
	if code, _, errs := run("publish", "hf", "-dir", filepath.Join(bin, "none"), "-confirm"); code != ExitUsage || !strings.Contains(errs, "no export") {
		t.Errorf("no export: %d %s", code, errs)
	}
	t.Setenv("HF_TOKEN", "fake")
	if code, _, errs := run("publish", "hf", "-dir", dir, "-confirm", "-repo", "someone/test"); code != ExitOK {
		t.Fatalf("with -confirm and a token: %d %s", code, errs)
	}
	got, err := os.ReadFile(log) // #nosec G304 -- the path is in the test.
	if err != nil || !strings.HasPrefix(string(got), "upload someone/test "+dir+" . --repo-type dataset") || strings.Contains(string(got), "fake") {
		t.Errorf("the upload got %q %v", got, err)
	}
}
