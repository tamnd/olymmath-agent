package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/publish"
	"github.com/tamnd/olymmath-agent/site"
)

// tree reads each file under root by its slash path.
func tree(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p) // #nosec G304 -- a test file.
		out[filepath.ToSlash(rel)] = string(b)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestSiteReleaseSameAsMemory runs om site release, which keeps the files in a spool on the disk, and wants the same static assets and packs as a release with all the files in memory. It wants no spool file left, and a heap profile with -memprofile.
func TestSiteReleaseSameAsMemory(t *testing.T) {
	// The test content has no tiers, so a copy gets them.
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS("../content/testdata/valid")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "registers", "site-tiers.yaml"), []byte("groups: [international]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	prof := filepath.Join(t.TempDir(), "heap.pprof")
	code, _, errs := run("site", "release", "-content", root, "-out", filepath.Join(dir, "site"), "-pack", filepath.Join(dir, "pack"),
		"-commit", "0123456789abcdef", "-date", "2026-11-03", "-cache", "off", "-memprofile", prof)
	if code != ExitOK {
		t.Fatalf("exit code %d: %s", code, errs)
	}
	if fi, err := os.Stat(prof); err != nil || fi.Size() == 0 {
		t.Errorf("no heap profile: %v", err)
	}
	left, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 2 {
		t.Errorf("the release left %d entries next to the packs, want site and pack", len(left))
	}

	corpus, findings := content.Load(root)
	if len(findings) != 0 {
		t.Fatal(findings)
	}
	tiers, err := publish.LoadTiers(filepath.Join(root, "registers", "site-tiers.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	opts := site.Options{URL: "https://olymmath.tamnd.com", Base: "/", Commit: "0123456789abcdef", Version: "test", Date: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)}
	mem, _, err := site.Render(corpus, opts)
	if err != nil {
		t.Fatal(err)
	}
	plan, err := publish.PlanSplit(mem, corpus, tiers, 19000)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Moved == 0 || plan.Deleted == 0 {
		t.Fatalf("the plan has %d moved files and %d redirect pages, want some of each", plan.Moved, plan.Deleted)
	}
	want := t.TempDir()
	if _, err := publish.Release(mem, plan, filepath.Join(want, "site"), filepath.Join(want, "pack")); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"site", "pack"} {
		got, exp := tree(t, filepath.Join(dir, d)), tree(t, filepath.Join(want, d))
		if len(got) != len(exp) {
			t.Errorf("%s: %d files, want %d", d, len(got), len(exp))
		}
		for f, data := range exp {
			if got[f] != data {
				t.Errorf("%s: %s differs", d, f)
			}
		}
	}
}
