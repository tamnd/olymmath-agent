package cli

import (
	"context"
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

func TestGrade(t *testing.T) {
	f := fakeModel(t)
	dir := copyContent(t)
	code, out, errs := run("grade", "-dry-run", "-content", dir)
	if code != ExitOK || !strings.Contains(out, "0 solutions to grade") || f.Calls() != 0 {
		t.Fatalf("the dry run gives %d, %q, %q", code, out, errs)
	}
	code, out, errs = run("grade", "-regrade", "-budget", "1", "-content", dir, "-work", t.TempDir(), "RG3MNR0R50")
	if code != ExitOK || !strings.Contains(out, "1 graded") || f.Calls() != 2 {
		t.Fatalf("code %d, %d calls, out %q, errs %q", code, f.Calls(), out, errs)
	}
	corpus, findings := content.Load(dir)
	if findings.Errors() > 0 {
		t.Fatalf("the content has errors after om grade: %v", findings)
	}
	s := corpus.ByTag("0HVHSDNNMY").Solutions[1]
	if s.Tag != "RG3MNR0R50" {
		s = corpus.ByTag("0HVHSDNNMY").Solutions[0]
	}
	g := s.Grade
	if g == nil || g.Score != 7 || g.Bucket != "correct" || g.Rubric != "imo/2024/p1@r2" || len(g.Graders) != 2 || corpus.Runs[g.RunID] == nil {
		t.Fatalf("the grade is %+v", g)
	}
	if code, out, _ := run("audit", "-content", dir); strings.Contains(out, "G01") || strings.Contains(out, "G03") {
		t.Fatalf("the audit gives %d: %s", code, out)
	}
	if code, out, _ := run("fmt", "-check", "-content", dir); code != ExitOK {
		t.Fatalf("the grade is not in the canonical form: %s", out)
	}
}

// cliModel answers the cli: jurors as gemini-2.5-pro would.
type cliModel struct{ f *model.Fake }

func (c cliModel) Complete(ctx context.Context, reqs []model.Request) ([]model.Response, error) {
	out, err := c.f.Complete(ctx, reqs)
	for i := range out {
		out[i].Model = "gemini-2.5-pro"
	}
	return out, err
}

func TestGradeOtherFamily(t *testing.T) {
	api := fakeModel(t)
	cf := &model.Fake{Answer: func(model.Request) string { return "Checkpoint 1 met.\nSCORE: 7" }}
	old := newCLIClient
	newCLIClient = func(string) model.Client { return cliModel{cf} }
	t.Cleanup(func() { newCLIClient = old })
	dir := copyContent(t)
	if code, _, errs := run("grade", "-dry-run", "-jurors", "claude-sonnet-5-5,cli:llama", "-content", dir); code != ExitUsage || !strings.Contains(errs, "cli:llama") {
		t.Fatalf("an unknown client gives %d, %q", code, errs)
	}
	code, out, errs := run("grade", "-regrade", "-budget", "1", "-jurors", "claude-sonnet-5-5,cli:gemini", "-content", dir, "-work", t.TempDir(), "RG3MNR0R50")
	if code != ExitOK || !strings.Contains(out, "1 graded") || api.Calls() != 1 || cf.Calls() != 1 || strings.Contains(out, "Note:") {
		t.Fatalf("code %d, %d and %d calls, out %q, errs %q", code, api.Calls(), cf.Calls(), out, errs)
	}
	corpus, findings := content.Load(dir)
	if findings.Errors() > 0 {
		t.Fatalf("the content has errors after om grade: %v", findings)
	}
	var g *content.Grade
	for _, s := range corpus.ByTag("0HVHSDNNMY").Solutions {
		if s.Tag == "RG3MNR0R50" {
			g = s.Grade
		}
	}
	if g == nil || len(g.Graders) != 2 || g.Graders[1].By != "model:gemini-2.5-pro" || g.Graders[1].Family != "google" || g.Graders[0].Family != "anthropic" {
		t.Fatalf("the grade is %+v", g)
	}
	if code, out, _ := run("fmt", "-check", "-content", dir); code != ExitOK {
		t.Fatalf("the grade is not in the canonical form: %s", out)
	}
	// Only cli: jurors need no API key.
	t.Setenv(model.KeyEnv, "")
	newClient = func(bool) (model.Client, error) { return model.NewAnthropic() }
	code, out, errs = run("grade", "-regrade", "-budget", "1", "-jurors", "cli:gemini,cli:codex", "-content", dir, "-work", t.TempDir(), "RG3MNR0R50")
	if code != ExitOK || !strings.Contains(out, "1 graded") {
		t.Fatalf("with cli: jurors only: code %d, out %q, errs %q", code, out, errs)
	}
}

// TestSolveCLI runs om solve with a generator of a local command-line client. It needs no API key, and the solution names the model that answered.
func TestSolveCLI(t *testing.T) {
	t.Setenv(model.KeyEnv, "")
	f := fakeModel(t)
	newClient = func(bool) (model.Client, error) { return model.NewAnthropic() }
	old := newCLIClient
	newCLIClient = func(string) model.Client { return cliModel{f} }
	t.Cleanup(func() { newCLIClient = old })
	dir := copyContent(t)
	code, out, errs := run("solve", "-budget", "0.01", "-model", "cli:gemini/gemini-2.5-pro", "-content", dir, "-work", t.TempDir(), "-k", "1", "imo/2024/p2")
	if code != ExitOK || !strings.Contains(out, "1 accepted") {
		t.Fatalf("code %d, out %q, errs %q", code, out, errs)
	}
	corpus, findings := content.Load(dir)
	if findings.Errors() > 0 {
		t.Fatalf("the content has errors after om solve: %v", findings)
	}
	s := corpus.ByTag("03JXB69J3N").Solutions[0]
	if s.Run == nil || s.Run.Model != "gemini-2.5-pro" || s.Author.Display != "gemini-2.5-pro" {
		t.Fatalf("the solution is %+v, run %+v", s, s.Run)
	}
	if rec := corpus.Runs[s.Run.RunID]; rec == nil || rec.Items[0].Model != "gemini-2.5-pro" || rec.Items[0].Tokens.Input == 0 {
		t.Fatalf("the run record is %+v", rec)
	}
}
