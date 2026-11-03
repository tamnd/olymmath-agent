package site

import (
	"net/url"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// formalView is the formal section of a problem page (spec 08.7): the statement of the best rank, its fidelity badge, its answer, its proof status, its source and the links to read it in Lean.
type formalView struct {
	*content.Formal
	// Fidelity is the status that the review records back, and Badge is its label. Only two-human is shown as faithful.
	Fidelity string
	Badge    string
	Faithful bool
	// SourceName is the name of the source in the register.
	SourceName string
	// Proof is the label of the proof status, and NativeDecide flags a proof that uses native_decide.
	Proof        string
	Proved       bool
	NativeDecide bool
	// File is the Lean file in the content repository, Upstream is the file in its source, and Live opens the statement in the Lean web editor.
	File     string
	Upstream string
	Live     string
	// Others is the number of the other formal statements of the problem.
	Others int
}

// fidelityLabels are the labels of the fidelity statuses on the site.
var fidelityLabels = map[string]string{
	content.TwoHuman:     "Faithful: checked by two people",
	content.OneHuman:     "Reviewed once",
	content.LLMTriaged:   "Triaged by models, not reviewed by a person",
	content.CheckerClean: "Passed the checks, not reviewed by a person",
	content.Unreviewed:   "Not reviewed",
	content.KnownWrong:   "Known wrong",
}

// FidelityLabel returns the label of a fidelity status.
func FidelityLabel(status string) string {
	if l, ok := fidelityLabels[status]; ok {
		return l
	}
	return fidelityLabels[content.Unreviewed]
}

// liveURL opens a statement in the Lean web editor with Mathlib.
func liveURL(text string) string {
	return "https://live.lean-lang.org/#code=" + url.QueryEscape("import Mathlib\n\n"+strings.TrimSpace(text)+" := by\n  sorry\n")
}

func (b *builder) formalView(p *content.Problem) *formalView {
	f := p.Top()
	if f == nil {
		return nil
	}
	st := b.corpus.ShownFidelity(f)
	v := &formalView{
		Formal:     f,
		Fidelity:   st,
		Badge:      FidelityLabel(st),
		Faithful:   b.corpus.Faithful(f),
		SourceName: f.Origin.Source,
		File:       b.opts.Repo + "/blob/main/" + f.Path,
		Upstream:   f.Origin.URL,
		Live:       liveURL(f.Text),
		Others:     len(p.Formals) - 1,
		Proof:      "No proof yet",
	}
	if r := b.corpus.Source(f.Origin.Source); r != nil {
		v.SourceName = r.Name
	}
	for _, pr := range f.ProofRecords {
		if pr.Check != nil && pr.Check.Result == content.Pass && !pr.Check.Sorry {
			v.Proved = true
			v.NativeDecide = pr.Check.NativeDecide
			v.Proof = "Proof checked at Lean " + strings.TrimPrefix(pr.Check.Toolchain, "leanprover/lean4:") + " and Mathlib " + pr.Check.Mathlib
			break
		}
	}
	return v
}
