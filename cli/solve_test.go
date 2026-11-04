package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/model"
)

// fakeModel answers as a solver that proves (a, b) = (1, 1) and as a verifier that passes everything.
func fakeModel(t *testing.T) *model.Fake {
	t.Helper()
	f := &model.Fake{Batch: true, Answer: func(r model.Request) string {
		switch r.Kind {
		case "generate", "revise":
			return "Take $a = b = 1$. " + strings.Repeat("Each step holds. ", 10) + `So $\boxed{(a, b) = (1, 1)}$.`
		case "grade":
			return "Checkpoint 1 met.\nSCORE: 7"
		}
		return "No issues.\nVERDICT: PASS"
	}}
	old := newClient
	newClient = func(bool) (model.Client, error) { return f, nil }
	t.Cleanup(func() { newClient = old })
	return f
}

func TestSolveDryRun(t *testing.T) {
	f := fakeModel(t)
	code, out, errs := run("solve", "-dry-run", "-content", "../content/testdata/valid")
	if code != ExitOK || !strings.Contains(out, "problems, about") || f.Calls() != 0 {
		t.Fatalf("code %d, calls %d, out %q, errs %q", code, f.Calls(), out, errs)
	}
}

func TestSolveNoKey(t *testing.T) {
	t.Setenv(model.KeyEnv, "")
	dir := copyContent(t)
	code, _, errs := run("solve", "-budget", "1", "-content", dir, "-work", t.TempDir(), "imo/2024/p2")
	if code != ExitUsage || !strings.Contains(errs, model.KeyEnv) {
		t.Fatalf("code %d, errs %q", code, errs)
	}
	if _, err := os.Stat(filepath.Join(dir, "runs", "solve-")); err == nil {
		t.Fatal("a run record was written with no key")
	}
}

func TestSolve(t *testing.T) {
	f := fakeModel(t)
	dir := copyContent(t)
	work := t.TempDir()
	code, out, errs := run("solve", "-budget", "5", "-content", dir, "-work", work, "-k", "1", "imo/2024/p2")
	if code != ExitOK || !strings.Contains(out, "1 accepted") {
		t.Fatalf("code %d, out %q, errs %q", code, out, errs)
	}
	corpus, findings := content.Load(dir)
	if findings.Errors() > 0 {
		t.Fatalf("the content has errors after om solve: %v", findings)
	}
	p := corpus.ByTag("03JXB69J3N")
	if len(p.Solutions) != 1 {
		t.Fatalf("the problem has %d solutions", len(p.Solutions))
	}
	s := p.Solutions[0]
	if s.Origin != "ai" || s.Run == nil || s.Run.Model != model.Sonnet || s.AnswerCheck != "match" || s.Run.Cost <= 0 {
		t.Fatalf("the solution is %+v, run %+v", s, s.Run)
	}
	rec := corpus.Runs[s.Run.RunID]
	if rec == nil || rec.Status != "done" || len(rec.Items) != 1 || rec.Items[0].Solution != s.Tag || rec.Cost != rec.Items[0].Cost {
		t.Fatalf("the run record is %+v", rec)
	}
	if _, err := os.Stat(filepath.Join(work, rec.ID, "03JXB69J3N.json")); err != nil {
		t.Fatal("no transcript")
	}
	if code, out, _ := run("audit", "-content", dir); code == ExitFindings && strings.Contains(out, " error") {
		t.Fatalf("the audit finds errors: %s", out)
	}
	// A resumed run reads every call from the cache and writes no second solution.
	n := f.Calls()
	code, _, errs = run("solve", "-budget", "5", "-content", dir, "-work", work, "-k", "1", "-resume", rec.ID, "imo/2024/p2")
	if code != ExitOK || f.Calls() != n {
		t.Fatalf("the resume gives %d and %d new calls: %s", code, f.Calls()-n, errs)
	}
	corpus, _ = content.Load(dir)
	if len(corpus.ByTag("03JXB69J3N").Solutions) != 1 {
		t.Fatal("the resume wrote a second solution")
	}
}
