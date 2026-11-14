package cli

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/model"
)

func TestRubricDraft(t *testing.T) {
	t.Setenv(model.KeyEnv, "")
	f := &model.Fake{Answer: func(model.Request) string {
		return "CHECKPOINT 3: Find the pairs.\nCHECKPOINT 4: Prove that no other pair works.\nZERO: Small cases only.\nDEDUCT -1: A small gap.\nNOTES:\nThe answer is $(1, 1)$."
	}}
	old := newCLIClient
	newCLIClient = func(string) model.Client { return cliModel{f} }
	t.Cleanup(func() { newCLIClient = old })
	dir := copyContent(t)
	if code, out, _ := run("rubric", "-dry-run", "-model", "cli:gemini", "-content", dir, "imo/2024/p1", "imo/2024/p2"); code != ExitOK || !strings.Contains(out, "1 problems need a rubric") {
		t.Fatalf("the dry run gives %d, %q", code, out)
	}
	code, out, errs := run("rubric", "-budget", "0.01", "-model", "cli:gemini", "-content", dir, "-work", t.TempDir(), "imo/2024/p2")
	if code != ExitOK || !strings.Contains(out, "1 rubrics drafted") || f.Calls() != 1 {
		t.Fatalf("code %d, out %q, errs %q", code, out, errs)
	}
	corpus, findings := content.Load(dir)
	if findings.Errors() > 0 {
		t.Fatalf("the content has errors after om rubric: %v", findings)
	}
	r := corpus.ByTag("03JXB69J3N").Rubric
	if r == nil || r.Status != "draft" || r.DraftedBy != "model:gemini-2.5-pro" || len(r.Checkpoints) != 2 || r.Notes != "The answer is $(1, 1)$." {
		t.Fatalf("the rubric is %+v", r)
	}
	var rec *content.RunRecord
	for _, x := range corpus.Runs {
		if x.Kind == "rubric" {
			rec = x
		}
	}
	if rec == nil || rec.Status != "done" || rec.Items[0].Outcome != "drafted" || rec.Items[0].Model != "gemini-2.5-pro" {
		t.Fatalf("the run record is %+v", rec)
	}
}
