package audit

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// Pin is the toolchain and the Mathlib version that the Lake package of the content pins (spec 08.3).
type Pin struct {
	Toolchain string
	Mathlib   string
}

var mathlibRevRE = regexp.MustCompile(`(?s)name\s*=\s*"mathlib".*?rev\s*=\s*"([^"]+)"`)

// ReadPin reads lean/lean-toolchain and the Mathlib rev of lean/lakefile.toml under root. A missing file gives an empty field.
func ReadPin(root string) Pin {
	var p Pin
	if b, err := os.ReadFile(filepath.Join(root, content.LeanDir, "lean-toolchain")); err == nil { // #nosec G304 -- the path is inside the content root.
		p.Toolchain = strings.TrimSpace(string(b))
	}
	if b, err := os.ReadFile(filepath.Join(root, content.LeanDir, "lakefile.toml")); err == nil { // #nosec G304 -- the path is inside the content root.
		if m := mathlibRevRE.FindSubmatch(b); m != nil {
			p.Mathlib = string(m[1])
		}
	}
	return p
}

func (c *Context) pin() Pin {
	if c.pinned == nil {
		p := ReadPin(c.Root)
		c.pinned = &p
	}
	return *c.pinned
}

func (c *Context) fileExists(rel string) bool {
	_, err := os.Stat(filepath.Join(c.Root, filepath.FromSlash(rel)))
	return err == nil
}

// f01 checks that each formal statement is a complete record and that its last check passed at the pinned toolchain.
func f01(c *Context) {
	if len(c.Corpus.Formals) == 0 {
		return
	}
	pin := c.pin()
	if pin.Toolchain == "" || pin.Mathlib == "" {
		c.add(content.LeanDir, "the Lake package has no lean-toolchain or no Mathlib rev in lakefile.toml")
		return
	}
	for _, f := range c.Corpus.Formals {
		switch {
		case f.System != "lean4":
			c.add(f.File, "the system is %q, want lean4", f.System)
		case f.Toolchain != pin.Toolchain || f.Mathlib != pin.Mathlib:
			c.add(f.File, "the statement is for %s and Mathlib %s, but the package pins %s and Mathlib %s", f.Toolchain, f.Mathlib, pin.Toolchain, pin.Mathlib)
		}
		if !strings.HasPrefix(f.Path, content.LeanDir+"/") || !strings.HasSuffix(f.Path, ".lean") || !c.fileExists(f.Path) {
			c.add(f.File, "the Lean file %q is not in the package", f.Path)
		}
		if f.PortDiff != "" && !c.fileExists(f.PortDiff) {
			c.add(f.File, "the port diff %q does not exist", f.PortDiff)
		}
		if f.Module == "" || f.Decl == "" || strings.TrimSpace(f.Text) == "" {
			c.add(f.File, "the record needs a module, a declaration and the text of the statement")
		}
		if !slices.Contains(content.FormalOriginKinds, f.Origin.Kind) {
			c.add(f.File, "the origin kind %q is not one of %s", f.Origin.Kind, strings.Join(content.FormalOriginKinds, ", "))
		}
		if c.Corpus.Source(f.Origin.Source) == nil {
			c.add(f.File, "the source %q is not in registers/sources.yaml", f.Origin.Source)
		}
		if f.Licence == "" {
			c.add(f.File, "the record has no licence")
		}
		if f.Answer != nil && !slices.Contains(content.AnswerStyles, f.Answer.Style) {
			c.add(f.File, "the answer style %q is not one of %s", f.Answer.Style, strings.Join(content.AnswerStyles, ", "))
		}
		if f.Rank < 1 {
			c.add(f.File, "the rank must be 1 or more")
		}
		switch ch := f.Check; {
		case ch == nil:
			c.add(f.File, "the statement has no check record, so it is not known to compile")
		case ch.Result != content.Pass:
			c.add(f.File, "the statement does not compile: the check result is %q", ch.Result)
		case ch.Toolchain != pin.Toolchain || ch.Mathlib != pin.Mathlib:
			c.add(f.File, "the statement last compiled at %s and Mathlib %s, not at the pinned toolchain", ch.Toolchain, ch.Mathlib)
		}
	}
}

// f02 checks that each proof passed its check, uses no sorry and uses only the allowed axioms.
func f02(c *Context) {
	pin := c.pin()
	for _, pr := range c.Corpus.Proofs {
		ch := pr.Check
		if ch == nil {
			c.add(pr.File, "the proof has no check record")
			continue
		}
		if ch.Result != content.Pass {
			c.add(pr.File, "the proof does not check: the result is %q", ch.Result)
		}
		if ch.Toolchain != pin.Toolchain || ch.Mathlib != pin.Mathlib {
			c.add(pr.File, "the proof last checked at %s and Mathlib %s, not at the pinned toolchain", ch.Toolchain, ch.Mathlib)
		}
		if ch.Sorry || slices.Contains(ch.Axioms, "sorryAx") {
			c.add(pr.File, "the proof uses sorry")
		}
		for _, a := range ch.Axioms {
			switch {
			case a == "sorryAx" || slices.Contains(content.AllowedAxioms, a):
			case slices.Contains(content.NativeAxioms, a):
				if !ch.NativeDecide {
					c.add(pr.File, "the proof depends on %s, so it needs the native_decide flag", a)
				}
			default:
				c.add(pr.File, "the proof depends on the axiom %s, which is not allowed", a)
			}
		}
	}
}

// f03 checks the fidelity record of each formal statement. A status above unreviewed needs review records that back it, and only people can give one-human and two-human.
func f03(c *Context) {
	for _, r := range sortedKeys(c.Corpus.FidelityReviews) {
		rv := c.Corpus.FidelityReviews[r]
		if rv.ByAgent() {
			if rv.By != "agent" || !strings.HasPrefix(rv.Note, content.AgentNote) {
				c.add(reviewFile(rv), "an agent review must have by: agent and a note that starts %q", content.AgentNote)
			}
			if rv.Decision == content.DecisionFaithful || rv.Decision == content.DecisionWrong {
				c.add(reviewFile(rv), "an agent review cannot decide %s; only a person can", rv.Decision)
			}
		}
	}
	for _, f := range c.Corpus.Formals {
		fd := f.Fidelity
		if !slices.Contains(content.FidelityStatuses, fd.Status) {
			c.add(f.File, "the fidelity status %q is not one of %s", fd.Status, strings.Join(content.FidelityStatuses, ", "))
			continue
		}
		for _, id := range fd.Reviews {
			r := c.Corpus.FidelityReviews[id]
			switch {
			case r == nil:
				c.add(f.File, "the fidelity review %s is not in reviews/ with kind fidelity", id)
			case !r.Names(f.Tag):
				c.add(f.File, "the fidelity review %s does not name %s", id, f.Tag)
			}
		}
		people := len(c.Corpus.HumanSignOffs(f))
		switch fd.Status {
		case content.TwoHuman:
			if people < 2 {
				c.add(f.File, "the status two-human needs faithful reviews by two people, and it has %d", people)
			}
		case content.OneHuman:
			if people < 1 {
				c.add(f.File, "the status one-human needs a faithful review by a person")
			}
		case content.CheckerClean, content.LLMTriaged:
			if len(fd.Reviews) == 0 {
				c.add(f.File, "the status %s needs a review record of the checks", fd.Status)
			}
		case content.KnownWrong:
			if len(fd.Defects) == 0 {
				c.add(f.File, "a known-wrong statement needs its defect and reason")
			}
		}
		if got := c.Corpus.ShownFidelity(f); got != fd.Status {
			c.add(f.File, "the fidelity status is %s, but the review records back only %s", fd.Status, got)
		}
	}
}

// formalTags returns the tags and kinds of the formal and proof records, with their files, for rule T02.
func formalTags(corpus *content.Corpus) [][3]string {
	var out [][3]string
	for _, f := range corpus.Formals {
		out = append(out, [3]string{f.Tag, "formal", f.File})
	}
	for _, p := range corpus.Proofs {
		out = append(out, [3]string{p.Tag, "proof", p.File})
	}
	return out
}

// reviewFile gives the file of a review record, relative to the content root.
func reviewFile(r *content.FidelityReview) string {
	if len(r.Date) >= 7 {
		return "reviews/" + r.Date[:4] + "/" + r.Date[5:7] + "/" + r.ID + ".yaml"
	}
	return "reviews/" + r.ID + ".yaml"
}
