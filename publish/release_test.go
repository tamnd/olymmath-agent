package publish

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

// files reads each file under root by its slash path. A folder with no file does not count, as a host of static assets does not see it.
func files(t *testing.T, root string) map[string]string {
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

// same wants the same files with the same bytes in both maps.
func same(t *testing.T, name string, got, want map[string]string) {
	t.Helper()
	for _, f := range slices.Sorted(maps.Keys(want)) {
		g, ok := got[f]
		if !ok {
			t.Errorf("%s: %s is missing", name, f)
		} else if g != want[f] {
			t.Errorf("%s: %s differs", name, f)
		}
	}
	for f := range got {
		if _, ok := want[f]; !ok {
			t.Errorf("%s: %s is extra", name, f)
		}
	}
}

// TestReleaseSameAsDisk builds the test content on disk with Build, Split, Fit and Pack, and in memory with Render, PlanSplit and Release. It wants the same static assets and the same packs, byte for byte, for each tier and each asset limit.
func TestReleaseSameAsDisk(t *testing.T) {
	corpus, findings := content.Load("../content/testdata/valid")
	if len(findings) != 0 {
		t.Fatal(findings)
	}
	opts := site.Options{URL: "https://example.org", Commit: "0123456789abcdef", Version: "test", Date: time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)}
	mem, _, err := site.Render(corpus, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name  string
		tiers Tiers
	}{
		{"tier 1", Tiers{Groups: []string{"international"}}},
		{"tier 2", Tiers{}},
	} {
		// The limit under the number of static files makes Fit move a competition, and the limit 1 gives the error of Fit.
		all, err := PlanSplit(mem, corpus, tc.tiers, len(mem))
		if err != nil {
			t.Fatal(err)
		}
		static := 0
		for _, place := range all.Places {
			if place == Static {
				static++
			}
		}
		for _, limit := range []int{len(mem), static - 1, 1} {
			t.Run(fmt.Sprintf("%s, limit %d", tc.name, limit), func(t *testing.T) {
				disk := t.TempDir()
				o := opts
				o.Out = filepath.Join(disk, "site")
				if _, err := site.Build(corpus, o); err != nil {
					t.Fatal(err)
				}
				r2 := filepath.Join(disk, "r2")
				deleted, moved, err := Split(o.Out, r2, corpus, tc.tiers)
				if err != nil {
					t.Fatal(err)
				}
				fitted, codes, ferr := Fit(o.Out, r2, corpus, limit)

				plan, perr := PlanSplit(mem, corpus, tc.tiers, limit)
				if (ferr == nil) != (perr == nil) {
					t.Fatalf("Fit gives %v and PlanSplit gives %v", ferr, perr)
				}
				if plan.Deleted != deleted || plan.Moved != moved || plan.Fitted != fitted || !slices.Equal(plan.Fit, codes) {
					t.Fatalf("the plan has %d deleted, %d moved, %d fitted %v; the disk has %d, %d, %d %v", plan.Deleted, plan.Moved, plan.Fitted, plan.Fit, deleted, moved, fitted, codes)
				}
				t.Logf("%d files, %d redirect pages, %d moved, %d fitted %v, %v", len(mem), deleted, moved, fitted, codes, ferr)
				if ferr != nil {
					return
				}
				if plan.Deleted == 0 || plan.Moved == 0 {
					t.Fatalf("the test content has %d redirect pages and %d moved files, want some of each", plan.Deleted, plan.Moved)
				}
				if _, err := Pack(r2, filepath.Join(disk, "pack"), filepath.Join(o.Out, "_pack")); err != nil {
					t.Fatal(err)
				}
				rel := t.TempDir()
				if _, err := Release(mem, plan, filepath.Join(rel, "site"), filepath.Join(rel, "pack")); err != nil {
					t.Fatal(err)
				}
				same(t, "site", files(t, filepath.Join(rel, "site")), files(t, o.Out))
				same(t, "pack", files(t, filepath.Join(rel, "pack")), files(t, filepath.Join(disk, "pack")))
				if len(files(t, filepath.Join(rel, "site", "_pack"))) == 0 {
					t.Error("the release has no pack index")
				}
			})
		}
	}
}

// TestPlanSplitSameAsDisk runs Split and Fit on a site with several competitions, and wants PlanSplit to put each file at the same place.
func TestPlanSplitSameAsDisk(t *testing.T) {
	mem := map[string][]byte{
		"redirects.json": []byte(`{"/p/ABC":"/imo/2024/p1","/isl/2024/n1":"/imo/2024/p1","/nz/old":"/nz/2020/p1"}`),
		"index.html":     []byte("home"),
		"nz.html":        []byte("not in a folder"),
	}
	for _, rel := range []string{
		"p/ABC.html", "isl/2024/n1.html", "nz/old.html",
		"imo/index.html", "imo/2024/index.html", "imo/2024/p1.html", "imo/2024/p1.md", "imo/2024/p1.json", "imo/2024/p2.html", "imo/2024/p3.html",
		"isl/2024/index.html", "isl/2024/g1.html", "isl/2024/g1.md",
		"p/XYZ.html", "p/XYZ.json", "p/XYZ.md",
		"nz/index.html", "nz/2020/index.html", "nz/2020/p1.html", "nz/2020/p1.json",
		"api/index.json", "api/problem/ABC.json", "api/edition/imo/2024.json",
		"assets/site.css",
	} {
		mem[rel] = []byte(rel)
	}
	corpus := &content.Corpus{Competitions: []*content.Competition{
		{Code: "imo", Group: "international"},
		{Code: "isl", Group: "international"},
		{Code: "nz", Group: "national", Country: "NZ"},
	}}
	tiers := Tiers{Groups: []string{"international"}}
	for limit := range 20 {
		dir, out := t.TempDir(), t.TempDir()
		for rel, data := range mem {
			write(t, dir, rel, string(data))
		}
		deleted, moved, err := Split(dir, out, corpus, tiers)
		if err != nil {
			t.Fatal(err)
		}
		fitted, codes, ferr := Fit(dir, out, corpus, limit)
		plan, perr := PlanSplit(mem, corpus, tiers, limit)
		if (ferr == nil) != (perr == nil) || plan.Deleted != deleted || plan.Moved != moved || plan.Fitted != fitted || !slices.Equal(plan.Fit, codes) {
			t.Fatalf("limit %d: the plan has %d, %d, %d %v %v; the disk has %d, %d, %d %v %v", limit, plan.Deleted, plan.Moved, plan.Fitted, plan.Fit, perr, deleted, moved, fitted, codes, ferr)
		}
		want := map[string]string{}
		for f, s := range files(t, dir) {
			want[f] = "static " + s
		}
		for f, s := range files(t, out) {
			want[f] = "r2 " + s
		}
		got := map[string]string{}
		for f, place := range plan.Places {
			switch place {
			case Static:
				got[f] = "static " + string(mem[f])
			case Bucket:
				got[f] = "r2 " + string(mem[f])
			}
		}
		same(t, fmt.Sprintf("limit %d", limit), got, want)
	}
}

// TestPackFilesSameAsPack wants the same packs and indexes from files in memory as from the same files on disk, with a big folder in shards and a small one in one pack.
func TestPackFilesSameAsPack(t *testing.T) {
	in := t.TempDir()
	mem := map[string][]byte{}
	for i := range ShardFiles + 5 {
		mem[fmt.Sprintf("api/problem/%d.json", i)] = fmt.Appendf(nil, "{%d}", i)
	}
	mem["nz/2020/p1.html"] = []byte("p1")
	mem["nz/index.html"] = []byte("nz")
	for rel, data := range mem {
		write(t, in, rel, string(data))
	}
	a, b := t.TempDir(), t.TempDir()
	ra, err := Pack(in, filepath.Join(a, "pack"), filepath.Join(a, "index"))
	if err != nil {
		t.Fatal(err)
	}
	rb, err := PackFiles(mem, filepath.Join(b, "pack"), filepath.Join(b, "index"))
	if err != nil {
		t.Fatal(err)
	}
	if ra != rb || ra.Packs != Shards+1 {
		t.Fatalf("Pack gives %+v and PackFiles gives %+v", ra, rb)
	}
	same(t, "packs", files(t, b), files(t, a))
	if _, err := PackFiles(map[string][]byte{"top.html": nil}, b, b); err == nil {
		t.Error("a file that is not in a folder gave no error")
	}
}
