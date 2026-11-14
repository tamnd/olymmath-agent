package grade

import (
	"strings"
	"testing"
)

func TestParseDraft(t *testing.T) {
	text := "Some thinking first.\n**CHECKPOINT 2:** Find the answer $n = 3$.\n- CHECKPOINT 5 points: Prove the bound.\nZERO: The answer with no proof.\nDEDUCT -1: A small gap.\nNOTES:\nThe answer is 3.\nUse induction."
	r, err := ParseDraft(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Checkpoints) != 2 || r.Checkpoints[0].Points != 2 || r.Checkpoints[0].Text != "Find the answer $n = 3$." || r.Checkpoints[1].Points != 5 {
		t.Fatalf("the checkpoints are %+v", r.Checkpoints)
	}
	if len(r.ZeroCredit) != 1 || len(r.Deductions) != 1 || r.Deductions[0].Points != -1 {
		t.Fatalf("the rubric is %+v", r)
	}
	if r.Notes != "The answer is 3.\nUse induction." || r.Status != "draft" || r.Source != "drafted" || r.Version != 1 || r.Scale != "imo-7" {
		t.Fatalf("the rubric is %+v", r)
	}
	for _, bad := range []string{"no rubric here", "CHECKPOINT 3: one\nCHECKPOINT 3: two"} {
		if _, err := ParseDraft(bad); err == nil {
			t.Errorf("ParseDraft(%q) gives no error", bad)
		}
	}
}

func TestDraftRequest(t *testing.T) {
	r := DraftRequest(Draft{ID: "X", Statement: "Find n.", Answer: "3", References: []string{"Since ..."}}, "cli:codex", "high")
	text := r.Messages[0].Text
	if r.Kind != "rubric" || r.Model != "cli:codex" || !strings.Contains(text, "Known answer: 3") || !strings.Contains(text, "Reference solution 1") {
		t.Fatalf("the request is %+v", r)
	}
}
