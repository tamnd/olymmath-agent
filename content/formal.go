package content

import (
	"path"
	"slices"
	"strings"
)

// FormalDir is the directory of the formal and proof records in the content repository. The records of a problem are at formal/<locator>/<tag>.yaml, next to the path of the problem file.
const FormalDir = "formal"

// LeanDir is the Lake package Olymmath in the content repository (spec 08.3).
const LeanDir = "lean"

// Formal is one formal statement of a problem (spec 08.2).
type Formal struct {
	Tag       string `yaml:"tag"`
	Kind      string `yaml:"kind"`
	Problem   string `yaml:"problem"`
	Part      string `yaml:"part,omitempty"`
	System    string `yaml:"system"`
	Toolchain string `yaml:"toolchain"`
	Mathlib   string `yaml:"mathlib"`
	// Path is the Lean file, relative to the content root, for example lean/vendor/compfiles/1745e6ced3e6/Compfiles/Imo2024P1.lean.
	Path string `yaml:"path"`
	// Module is the Lean module of the file, for example Compfiles.Imo2024P1.
	Module string `yaml:"module"`
	// Decl is the full name of the theorem that states the problem.
	Decl   string        `yaml:"decl"`
	Answer *FormalAnswer `yaml:"answer,omitempty"`
	Origin FormalOrigin  `yaml:"origin"`
	// Licence is the SPDX id of the licence of the Lean file.
	Licence string `yaml:"licence"`
	// PortDiff is the diff file of the port to the pinned toolchain, relative to the content root, or empty when the file compiles as it is.
	PortDiff string `yaml:"port_diff,omitempty"`
	// Options are the Lean options that differ from the defaults of the package, for example autoImplicit true for a source that needs it.
	Options map[string]string `yaml:"options,omitempty"`
	// Text is the statement in Lean: the signature of the theorem with the definitions that it needs, without the proof.
	Text         string   `yaml:"text"`
	Check        *Check   `yaml:"check,omitempty"`
	Fidelity     Fidelity `yaml:"fidelity"`
	EquivalentTo []string `yaml:"equivalent_to,omitempty"`
	Rank         int      `yaml:"rank"`
	Proofs       []string `yaml:"proofs,omitempty"`

	// File is the path of the record, relative to the content root.
	File string `yaml:"-"`
	// ProofRecords are the proof records of the statement, in the order of Proofs.
	ProofRecords []*Proof `yaml:"-"`
}

// FormalAnswer says how a formal statement gives the answer of an answer problem.
type FormalAnswer struct {
	// Style is one of AnswerStyles.
	Style string `yaml:"style"`
	Decl  string `yaml:"decl,omitempty"`
}

// AnswerStyles are the ways a formal statement gives its answer (spec 08.2).
var AnswerStyles = []string{"separate-def", "embedded", "none"}

// FormalOrigin says who wrote a formal statement or a proof, and where it comes from.
type FormalOrigin struct {
	// Kind is one of FormalOriginKinds.
	Kind   string `yaml:"kind"`
	Source string `yaml:"source"`
	URL    string `yaml:"url"`
	Commit string `yaml:"commit"`
	// NativeName is the name of the problem in the source, for example Imo2024P1.
	NativeName string `yaml:"native_name"`
}

// FormalOriginKinds are the authors of a formal text.
var FormalOriginKinds = []string{"human", "ai", "ai+human"}

// Check is the result of the last check of a formal statement or a proof at a toolchain (spec 08.5).
type Check struct {
	// Result is pass or fail.
	Result    string `yaml:"result"`
	Toolchain string `yaml:"toolchain"`
	Mathlib   string `yaml:"mathlib"`
	At        string `yaml:"at"`
	// Log is the log of the check, relative to the content root.
	Log string `yaml:"log,omitempty"`
	// Axioms are the axioms that the declaration depends on. A proof may use only AllowedAxioms.
	Axioms []string `yaml:"axioms,omitempty"`
	// Sorry is true when the declaration depends on sorryAx.
	Sorry bool `yaml:"sorry,omitempty"`
	// NativeDecide is true when the proof uses native_decide, so it depends on Lean.ofReduceBool. The site shows the flag.
	NativeDecide bool `yaml:"native_decide,omitempty"`
	// Comparator and Lean4checker are the results of the comparator against the frozen statement and of lean4checker: pass, fail or not-run.
	Comparator   string `yaml:"comparator,omitempty"`
	Lean4checker string `yaml:"lean4checker,omitempty"`
	Note         string `yaml:"note,omitempty"`
}

// The results of a check.
const (
	Pass   = "pass"
	Fail   = "fail"
	NotRun = "not-run"
)

// AllowedAxioms are the axioms that a proof may depend on (spec 08.5).
var AllowedAxioms = []string{"propext", "Quot.sound", "Classical.choice"}

// NativeAxioms are the axioms that native_decide adds. A proof may depend on them only with the native_decide flag.
var NativeAxioms = []string{"Lean.ofReduceBool", "Lean.trustCompiler"}

// Fidelity records how far a formal statement is known to say the same as the problem (spec 08.4).
type Fidelity struct {
	// Status is one of FidelityStatuses.
	Status string `yaml:"status"`
	// Reviews are the ids of the review records that back the status.
	Reviews []string `yaml:"reviews,omitempty"`
	Defects []Defect `yaml:"defects,omitempty"`
	// Steps record the steps of the pipeline of spec 08.4 that ran, with their results.
	Steps []FidelityStep `yaml:"steps,omitempty"`
}

// FidelityStep is the result of one step of the fidelity pipeline.
type FidelityStep struct {
	Step   string `yaml:"step"`
	Result string `yaml:"result"`
	At     string `yaml:"at"`
	By     string `yaml:"by"`
	Note   string `yaml:"note,omitempty"`
}

// Defect is one known or suspected defect of a formal statement.
type Defect struct {
	Kind string `yaml:"kind"`
	By   string `yaml:"by"`
	Note string `yaml:"note"`
}

// The fidelity statuses, from the weakest to the strongest, and known-wrong (spec 08.4).
const (
	Unreviewed   = "unreviewed"
	CheckerClean = "checker-clean"
	LLMTriaged   = "llm-triaged"
	OneHuman     = "one-human"
	TwoHuman     = "two-human"
	KnownWrong   = "known-wrong"
)

// FidelityStatuses are the fidelity statuses of spec 08.2.
var FidelityStatuses = []string{Unreviewed, CheckerClean, LLMTriaged, OneHuman, TwoHuman, KnownWrong}

// Proof is one formal proof of one formal statement (spec 04.1, 08.5).
type Proof struct {
	Tag    string       `yaml:"tag"`
	Kind   string       `yaml:"kind"`
	Formal string       `yaml:"formal"`
	Origin FormalOrigin `yaml:"origin"`
	Path   string       `yaml:"path"`
	Module string       `yaml:"module"`
	Decl   string       `yaml:"decl"`
	Check  *Check       `yaml:"check,omitempty"`

	File string `yaml:"-"`
}

// FidelityReview is a review record of kind fidelity (spec 09.5). The content package reads only these fields.
type FidelityReview struct {
	ID       string   `yaml:"id"`
	Kind     string   `yaml:"kind"`
	Records  []string `yaml:"records"`
	Decision string   `yaml:"decision"`
	By       string   `yaml:"by"`
	Date     string   `yaml:"date"`
	Note     string   `yaml:"note"`
}

// AgentNote is the start of the note of each review that an agent writes. Such a review is never a person's review.
const AgentNote = "Agent decision, not a person's review."

// The decisions of a fidelity review. A person decides faithful or wrong. An agent check records clean or flagged.
const (
	DecisionFaithful = "faithful"
	DecisionWrong    = "wrong"
	DecisionClean    = "clean"
	DecisionFlagged  = "flagged"
)

// ByAgent reports whether a review is by an agent and not by a person.
func (r *FidelityReview) ByAgent() bool {
	by := strings.ToLower(r.By)
	return by == "agent" || strings.HasPrefix(by, "agent:") || strings.HasPrefix(by, "ai:") || strings.HasPrefix(r.Note, AgentNote)
}

// Names reports whether the review names the record with the tag.
func (r *FidelityReview) Names(tag string) bool { return slices.Contains(r.Records, tag) }

// HumanSignOffs returns the people who reviewed the formal statement and found it faithful, each once. A review by an agent does not count.
func (c *Corpus) HumanSignOffs(f *Formal) []string {
	var people []string
	for _, id := range f.Fidelity.Reviews {
		r := c.FidelityReviews[id]
		if r == nil || r.ByAgent() || r.Decision != DecisionFaithful || !r.Names(f.Tag) {
			continue
		}
		if !slices.Contains(people, r.By) {
			people = append(people, r.By)
		}
	}
	return people
}

// ShownFidelity returns the fidelity status that the site may show for a formal statement. It is the recorded status, lowered to what the review records back. Only two sign-offs by two people give two-human, which the site calls faithful (spec 08.4 step 6).
func (c *Corpus) ShownFidelity(f *Formal) string {
	st := f.Fidelity.Status
	if !slices.Contains(FidelityStatuses, st) {
		return Unreviewed
	}
	n := len(c.HumanSignOffs(f))
	switch st {
	case TwoHuman:
		if n >= 2 {
			return TwoHuman
		}
		if n == 1 {
			return OneHuman
		}
		return c.agentLevel(f)
	case OneHuman:
		if n >= 1 {
			return OneHuman
		}
		return c.agentLevel(f)
	case CheckerClean, LLMTriaged:
		if l := c.agentLevel(f); l != Unreviewed {
			return st
		}
		return Unreviewed
	case KnownWrong:
		if len(f.Fidelity.Defects) > 0 {
			return KnownWrong
		}
		return Unreviewed
	}
	return Unreviewed
}

// agentLevel returns checker-clean when a review record of the statement says that the checks found nothing, and unreviewed otherwise. A model never gives more than llm-triaged, and this function never gives more than checker-clean.
func (c *Corpus) agentLevel(f *Formal) string {
	for _, id := range f.Fidelity.Reviews {
		if r := c.FidelityReviews[id]; r != nil && r.Names(f.Tag) && r.Decision == DecisionClean {
			return CheckerClean
		}
	}
	return Unreviewed
}

// Faithful reports whether the site may show the formal statement as faithful.
func (c *Corpus) Faithful(f *Formal) bool { return c.ShownFidelity(f) == TwoHuman }

// FormalPath returns the path of the record with the tag for a problem, for example formal/imo/2024/p1/QX2M9R27K3.yaml.
func FormalPath(p *Problem, tag string) string {
	return path.Join(FormalDir, strings.TrimSuffix(strings.TrimPrefix(p.Path, "problems/"), ".md"), tag+".yaml")
}

// Top returns the formal statement of the best rank, or nil.
func (p *Problem) Top() *Formal {
	if len(p.Formals) == 0 {
		return nil
	}
	return p.Formals[0]
}

// Proved reports whether a proof of the statement passed its check.
func (f *Formal) Proved() bool {
	for _, pr := range f.ProofRecords {
		if pr.Check != nil && pr.Check.Result == Pass && !pr.Check.Sorry {
			return true
		}
	}
	return false
}
