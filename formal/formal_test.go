package formal

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

const compfilesSrc = `module

public import ProblemExtraction

/-!
# IMO 2024 Problem 1
-/

@[expose] public section

namespace Imo2024P1

-- a comment with problem in it
determine solutionSet : Set ℝ := { α : ℝ | ∃ m : ℤ, α = 2 * m }

problem imo2024_p1 (α : ℝ) :
    α ∈ solutionSet ↔ ∀ n : ℕ, 0 < n → (n : ℤ) ∣ ∑ i ∈ Finset.Icc 1 n, ⌊i * α⌋ := by
  constructor
  · simp

end Imo2024P1
`

func TestCompfilesItems(t *testing.T) {
	its := compfilesItems("Compfiles/Imo2024P1.lean", compfilesSrc)
	if len(its) != 1 {
		t.Fatalf("got %d items", len(its))
	}
	it := its[0]
	if it.Locator != "imo/2024/p1" || it.Decl.Name != "Imo2024P1.imo2024_p1" || it.Answer == nil || it.Answer.Name != "Imo2024P1.solutionSet" || !it.Proved || it.Module != "Compfiles.Imo2024P1" {
		t.Fatalf("bad item %+v", it)
	}
	if strings.Contains(it.Decl.Text, ":= by") || !strings.HasPrefix(Standalone(it.Decl.Text), "theorem imo2024_p1") {
		t.Fatalf("bad text %q", it.Decl.Text)
	}
	for name, want := range map[string]string{"Usa1985P1": "usamo/1985/p1", "UK2020R1P3": "uk-bmo1/2020/p3", "Bulgaria1998P1": "", "UK2020P3": ""} {
		its := compfilesItems("Compfiles/"+name+".lean", "problem foo : True := by sorry\n")
		if len(its) != 1 || its[0].Locator != want || its[0].Proved {
			t.Errorf("%s: got %+v, want locator %q and no proof", name, its, want)
		}
	}
}

func TestOtherSources(t *testing.T) {
	its := putnamItems("src/putnam_2001_a1.lean", "import Mathlib\n\nabbrev putnam_2001_a1_solution : ℕ := sorry\n\ntheorem putnam_2001_a1 : putnam_2001_a1_solution = 1 :=\nsorry\n")
	if len(its) != 1 || its[0].Locator != "putnam/2001/a1" || its[0].Answer == nil || its[0].Proved {
		t.Fatalf("putnam: %+v", its)
	}
	its = imoslItems("IMOSLLean4/Main/IMO2018/A2/A2.lean", "namespace IMOSL\nnamespace IMO2018A2\n\ntheorem final_solution_part1 : True := trivial\n\ntheorem final_solution_part2 : True := trivial\n\nend IMO2018A2\nend IMOSL\n")
	if len(its) != 2 || its[0].Locator != "isl/2018/a2" || its[0].Part != "1" || its[1].Decl.Name != "IMOSL.IMO2018A2.final_solution_part2" {
		t.Fatalf("imosl: %+v", its)
	}
	if its := imoslItems("IMOSLLean4/Main/IMO2020/A4.lean", "theorem final_solution : True := trivial\n"); len(its) != 1 || its[0].Locator != "isl/2020/a4" {
		t.Fatalf("imosl flat layout: %+v", its)
	}
	its = archiveItems("Archive/Imo/Imo2024Q1.lean", "namespace Imo2024Q1\n\ntheorem helper : True := trivial\n\nend Imo2024Q1\n\ntheorem result : True := trivial\n")
	if len(its) != 1 || its[0].Decl.Name != "result" || its[0].Locator != "imo/2024/p1" {
		t.Fatalf("archive: %+v", its)
	}
	its = formalImoItems("Imo/2005/P3.lean", "theorem imo_2005_p3.parts.a : True := by\n  sorry\n\ntheorem imo_2005_p3.parts.b : True := by\n  sorry\n")
	if len(its) != 2 || its[1].Part != "b" || its[0].Proved || its[0].Module != "Imo.«2005».P3" {
		t.Fatalf("formal-imo: %+v", its)
	}
}

func TestParse(t *testing.T) {
	ax := ParseAxioms("'A.b' depends on axioms: [propext,\n Classical.choice, Quot.sound]\n'C' does not depend on any axioms\n'D' depends on axioms: [sorryAx]\n")
	if strings.Join(ax["A.b"], " ") != "Classical.choice Quot.sound propext" || len(ax["C"]) != 0 || ax["D"][0] != "sorryAx" {
		t.Fatalf("axioms: %v", ax)
	}
	if _, ok := ax["C"]; !ok {
		t.Fatal("C is missing")
	}
	src := "import X\n\nexample : True := by trivial -- om-check A trivial-goal\nexample : False := by simp -- om-check A contradictory\n"
	out := "/tmp/x/Compfiles.checks.lean:4:22: error: simp made no progress\n"
	flags, checked := ParseChecks(src, out)
	if !checked["A"] || len(flags["A"]) != 1 || flags["A"][0] != "trivial-goal" {
		t.Fatalf("checks: %v %v", flags, checked)
	}
	if f, c := ParseChecks(src, "error: unknown package 'X'\n"); len(f) != 0 || len(c) != 0 {
		t.Fatalf("a file that did not run gave %v %v", f, c)
	}
	errs := ParseBuildErrors("error: ./././vendor/compfiles/abc/Compfiles/Foo.lean:3:4: unknown identifier 'x'\nerror: ./././vendor/compfiles/abc/Compfiles/Foo.lean:9:4: other\n")
	if errs["Foo"] != "unknown identifier 'x'" {
		t.Fatalf("build errors: %v", errs)
	}
}

func TestLakefile(t *testing.T) {
	lf := Lakefile([]Vendored{{Source: Sources["compfiles"], Dir: "vendor/compfiles/1745e6ced3e6", Roots: []string{"ProblemExtraction"}}, {Source: Sources["imosllean4"], Dir: "vendor/imosllean4/883b0326e585"}}, nil)
	for _, want := range []string{`rev = "v4.35.0"`, `srcDir = "vendor/compfiles/1745e6ced3e6"`, `roots = ["ProblemExtraction"]`, `leanOptions = {autoImplicit = true}`, `autoImplicit = false`} {
		if !strings.Contains(lf, want) {
			t.Errorf("the lakefile has no %q:\n%s", want, lf)
		}
	}
}

func copyTree(t *testing.T, from, to string) {
	t.Helper()
	err := filepath.WalkDir(from, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(from, p)
		out := filepath.Join(to, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o750)
		}
		b, err := os.ReadFile(p) // #nosec G304 -- test data.
		if err != nil {
			return err
		}
		return os.WriteFile(out, b, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestImport(t *testing.T) {
	root := t.TempDir()
	copyTree(t, "../content/testdata/valid", root)
	up := t.TempDir()
	if err := os.MkdirAll(filepath.Join(up, "Compfiles"), 0o750); err != nil {
		t.Fatal(err)
	}
	for name, src := range map[string]string{"Imo2024P1.lean": compfilesSrc, "Imo2024P2.lean": "problem imo2024_p2 : True := by\n  sorry\n", "Bulgaria1998P1.lean": "problem b : True := trivial\n"} {
		if err := os.WriteFile(filepath.Join(up, "Compfiles", name), []byte(src), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	commit := "1745e6ced3e671bd74417548494b034eb2c87da8"
	rep, err := Import(root, Sources["compfiles"], up, commit, false)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Mapped != 2 || rep.Proofs != 1 || rep.NewTags != 0 || len(rep.Rows) != 3 {
		t.Fatalf("dry import: %+v", rep)
	}
	if _, err := os.Stat(filepath.Join(root, "lean/vendor/compfiles/1745e6ced3e6/Compfiles/Imo2024P2.lean")); err != nil {
		t.Fatal(err)
	}
	rep, err = Import(root, Sources["compfiles"], up, commit, true)
	if err != nil {
		t.Fatal(err)
	}
	// The fixture has the record and the proof of imo2024_p1, so they keep their tags. Only imo2024_p2 gets a new tag.
	if rep.Mapped != 2 || rep.Proofs != 1 || rep.NewTags != 1 || rep.Rows[1].Tag != "X70GFNJBRT" {
		t.Fatalf("import: %+v", rep)
	}
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("the content has errors after the import:\n%v", f)
	}
	n := 0
	for _, fm := range corpus.Formals {
		if fm.Decl == "Imo2024P1.imo2024_p1" && fm.Answer.Decl == "Imo2024P1.solutionSet" && len(fm.ProofRecords) == 1 {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("the import gave %d records of imo2024_p1 with a proof", n)
	}
}

func TestRecordAndFidelity(t *testing.T) {
	root := t.TempDir()
	copyTree(t, "../content/testdata/valid", root)
	r := &Results{
		Built:   map[string]bool{"Compfiles.Imo2024P1": true},
		Axioms:  map[string][]string{"Imo2024P1.imo2024_p1": {"Classical.choice", "Quot.sound", "propext"}},
		Flags:   map[string][]string{},
		Checked: map[string]bool{"Imo2024P1.imo2024_p1": true},
	}
	pass, fail, proofs, err := Record(root, "compfiles", r, "2026-11-03")
	if err != nil || pass != 1 || fail != 0 || proofs != 1 {
		t.Fatalf("record: %d %d %d %v", pass, fail, proofs, err)
	}
	clean, flagged, err := Fidelity(root, "compfiles", r, "2026-11-03")
	if err != nil || clean != 1 || flagged != 0 {
		t.Fatalf("fidelity: %d %d %v", clean, flagged, err)
	}
	corpus, f := content.Load(root)
	if f.Errors() > 0 {
		t.Fatalf("errors:\n%v", f)
	}
	fm := corpus.Formals[0]
	if fm.Fidelity.Status != content.CheckerClean || corpus.ShownFidelity(fm) != content.CheckerClean || corpus.Faithful(fm) {
		t.Fatalf("status %q, shown %q", fm.Fidelity.Status, corpus.ShownFidelity(fm))
	}
	rv := corpus.FidelityReviews[fm.Fidelity.Reviews[len(fm.Fidelity.Reviews)-1]]
	if rv == nil || rv.By != "agent" || !strings.HasPrefix(rv.Note, content.AgentNote) || rv.Decision != content.DecisionClean {
		t.Fatalf("review %+v", rv)
	}
	r.Flags["Imo2024P1.imo2024_p1"] = []string{"trivial-goal"}
	if _, flagged, err := Fidelity(root, "compfiles", r, "2026-11-04"); err != nil || flagged != 1 {
		t.Fatalf("flagged %d %v", flagged, err)
	}
	corpus, _ = content.Load(root)
	fm = corpus.Formals[0]
	if fm.Fidelity.Status != content.Unreviewed || len(fm.Fidelity.Defects) != 1 {
		t.Fatalf("after a flag: %+v", fm.Fidelity)
	}
}

func TestPortPatch(t *testing.T) {
	src := "import Mathlib\nimport FormalConjectures.Util.Answer\nimport Imo.ProblemImports.Attributes\n"
	got := Sources["formal-imo"].Patches["Imo/ProblemImports.lean"](src)
	if !strings.Contains(got, "import FormalConjecturesUtil.Answer\n") || strings.Contains(got, "FormalConjectures.Util") {
		t.Fatalf("patch: %q", got)
	}
	want := "--- a/x.lean\n+++ b/x.lean\n@@ -2 +2 @@\n-import FormalConjectures.Util.Answer\n+import FormalConjecturesUtil.Answer\n"
	if d := Diff("x.lean", src, got); d != want {
		t.Fatalf("diff:\n%s", d)
	}
	if Diff("x", "a", "a") != "" {
		t.Fatal("a diff of equal texts")
	}
	sym := "@[simp] lemma fst_out_mk_self : (Quot.out s(a, a)).1 = a := by simpa using out_fst_mem s(a, a)\n@[simp] lemma snd_out_mk_self : (Quot.out s(a, a)).2 = a := by simpa using out_snd_mem s(a, a)\n"
	if got := Sources["formal-conjectures"].Patches["FormalConjecturesForMathlib/Data/Sym/Sym2.lean"](sym); got != "-- Mathlib v4.35.0 has Sym2.fst_out_mk_self.\n-- Mathlib v4.35.0 has Sym2.snd_out_mk_self.\n" {
		t.Fatalf("sym2 patch: %q", got)
	}
	if !Sources["formal-conjectures"].Keep("FormalConjecturesForMathlib/Geometry/2d.lean") || Sources["formal-conjectures"].Keep("FormalConjectures/Paper/X.lean") {
		t.Fatal("keep")
	}
}

func TestParseChecksStopped(t *testing.T) {
	src := "import A\n\nexample : True := by trivial -- om-check A.x trivial-goal\nexample : True := by trivial -- om-check A.y trivial-goal\nexample : True := by trivial -- om-check A.z trivial-goal\n"
	out := "f.lean:3:1: error: unsolved goals\nf.lean:4:1: error: maximum number of errors (100; from option `maxErrors`) reached, exiting\n"
	flags, checked := ParseChecks(src, out)
	if !checked["A.x"] || checked["A.y"] || checked["A.z"] || len(flags) != 0 {
		t.Fatalf("flags %v checked %v", flags, checked)
	}
}

func TestDeclsSectionModifiers(t *testing.T) {
	src := "@[expose] public section\nnamespace Imo1962P4\nnoncomputable section\ndef f : Nat := 1\nend\nproblem imo1962_p4 : True := trivial\nend Imo1962P4\nend\n"
	ds := Decls(src)
	if len(ds) != 2 || ds[0].Name != "Imo1962P4.f" || ds[1].Name != "Imo1962P4.imo1962_p4" {
		t.Fatalf("decls: %+v", ds)
	}
}
