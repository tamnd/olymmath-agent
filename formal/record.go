package formal

import (
	"bufio"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// CheckScript is the script that builds a library of the package and runs the checks on the Lean host.
//
//go:embed check.sh
var CheckScript string

// Results are the outputs of the check script for one library.
type Results struct {
	// Built are the modules that compiled.
	Built map[string]bool
	// Axioms gives the axioms of each declaration that #print axioms reached.
	Axioms map[string][]string
	// Flags gives the checks that proved a statement, by declaration: a sign of a defect.
	Flags map[string][]string
	// Checked are the declarations that the static checks ran on.
	Checked map[string]bool
	// Errors are the first error lines of the build, by module.
	Errors map[string]string
}

// ReadResults reads the outputs of the check script for the library lib in dir.
func ReadResults(dir, lib string) (*Results, error) {
	r := &Results{Built: map[string]bool{}, Axioms: map[string][]string{}, Flags: map[string][]string{}, Checked: map[string]bool{}, Errors: map[string]string{}}
	built, err := os.ReadFile(filepath.Join(dir, lib+".built")) // #nosec G304 -- the path is the results directory that the user gives.
	if err != nil {
		return nil, err
	}
	for _, m := range strings.Fields(string(built)) {
		r.Built[m] = true
	}
	if b, err := os.ReadFile(filepath.Join(dir, lib+".axioms.out")); err == nil { // #nosec G304 -- as above.
		r.Axioms = ParseAxioms(string(b))
	}
	src, err1 := os.ReadFile(filepath.Join(dir, lib+".checks.lean")) // #nosec G304 -- as above.
	out, err2 := os.ReadFile(filepath.Join(dir, lib+".checks.out"))  // #nosec G304 -- as above.
	if err1 == nil && err2 == nil {
		r.Flags, r.Checked = ParseChecks(string(src), string(out))
	}
	if b, err := os.ReadFile(filepath.Join(dir, lib+".build.log")); err == nil { // #nosec G304 -- as above.
		r.Errors = ParseBuildErrors(string(b))
	}
	return r, nil
}

var axiomsRE = regexp.MustCompile(`(?s)'([^']+)' (depends on axioms: \[([^\]]*)\]|does not depend on any axioms)`)

// privateRE finds the prefix of a private name, as in _private.Archive.Imo.Imo2019Q2.0.imo2019_q2. The check script imports a file of the module system with import all, and then #print axioms names a private declaration in this form. A part of a module name that is a number is in « and », so the first part that is a number ends the prefix.
var privateRE = regexp.MustCompile(`^_private(?:\.[^.]+)*?\.[0-9]+\.`)

// ParseAxioms reads the output of #print axioms. A private declaration gets its name without the private prefix, which is the name in the targets.
func ParseAxioms(out string) map[string][]string {
	m := map[string][]string{}
	for _, g := range axiomsRE.FindAllStringSubmatch(out, -1) {
		var as []string
		for _, a := range strings.Split(g[3], ",") {
			if a = strings.TrimSpace(a); a != "" {
				as = append(as, a)
			}
		}
		sort.Strings(as)
		m[privateRE.ReplaceAllString(g[1], "")] = as
	}
	return m
}

var checkLineRE = regexp.MustCompile(`-- om-check (\S+) (\S+)$`)
var errLineRE = regexp.MustCompile(`(?m)^[^\s:]+\.lean:(\d+):\d+: error`)

// ParseChecks reads the file of static checks and its output. A check whose line has no error proved the statement, so it is a flag.
func ParseChecks(src, out string) (flags map[string][]string, checked map[string]bool) {
	bad := map[int]bool{}
	last := 0
	for _, m := range errLineRE.FindAllStringSubmatch(out, -1) {
		n, _ := strconv.Atoi(m[1])
		bad[n] = true
		last = max(last, n)
	}
	// Lean stops at its limit of errors. Then the checks after the last error did not run, and they count as not checked.
	stopped := strings.Contains(out, "maximum number of errors")
	flags, checked = map[string][]string{}, map[string]bool{}
	sc := bufio.NewScanner(strings.NewReader(src))
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	n := 0
	for sc.Scan() {
		n++
		m := checkLineRE.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		if stopped && n >= last {
			continue
		}
		checked[m[1]] = true
		if !bad[n] {
			flags[m[1]] = append(flags[m[1]], m[2])
		}
	}
	// A file that did not run at all has no error lines and would flag every check. Then no result counts.
	if len(bad) == 0 && strings.Contains(out, "error") {
		return map[string][]string{}, map[string]bool{}
	}
	return flags, checked
}

var buildErrRE = regexp.MustCompile(`(?m)^error: (?:\./)?(?:\S*/)?([^/\s]+)\.lean:\d+:\d+: (.*)$`)

// ParseBuildErrors gives the first error of each file in a build log, by file name without .lean.
func ParseBuildErrors(log string) map[string]string {
	m := map[string]string{}
	for _, g := range buildErrRE.FindAllStringSubmatch(log, -1) {
		if _, ok := m[g[1]]; !ok {
			m[g[1]] = g[2]
		}
	}
	return m
}

// Record writes the results of the checks into the formal and proof records of a source. A statement passes when its module compiled. A proof passes when its module compiled and #print axioms reached it.
//
// The date of a check is the date of the last run that changed its result (spec 08.5). When the new check of a record is the same as the old check, apart from the date, Record does not write the file, so the record keeps its old date. Changed counts the records that Record wrote.
func Record(root, source string, r *Results, date string) (pass, fail, proofs, changed int, err error) {
	corpus, fs := content.Load(root)
	if fs.Errors() > 0 {
		return 0, 0, 0, 0, fmt.Errorf("the content has errors:\n%v", fs)
	}
	for _, f := range corpus.Formals {
		if f.Origin.Source != source {
			continue
		}
		ch := &content.Check{Result: content.Fail, Toolchain: Toolchain, Mathlib: Mathlib, At: date}
		if r.Built[f.Module] {
			ch.Result = content.Pass
			pass++
		} else {
			fail++
			if e, ok := r.Errors[path.Base(strings.ReplaceAll(f.Module, ".", "/"))]; ok {
				ch.Note = "First error: " + e
			}
		}
		if v := f.Options["autoImplicit"]; v == "true" {
			ch.Note = strings.TrimSpace(ch.Note + " The source needs autoImplicit true.")
		}
		for _, pr := range f.ProofRecords {
			pc := &content.Check{Result: content.Fail, Toolchain: Toolchain, Mathlib: Mathlib, At: date, Comparator: content.NotRun, Lean4checker: content.NotRun}
			as, ok := r.Axioms[pr.Decl]
			if r.Built[pr.Module] && ok {
				pc.Axioms = as
				pc.Sorry = slices.Contains(as, "sorryAx")
				pc.NativeDecide = slices.Contains(as, "Lean.ofReduceBool")
				if !pc.Sorry {
					pc.Result = content.Pass
					proofs++
				}
				pc.Note = "The proof is in the same declaration as the statement, so it cannot change the statement."
			}
			if SameCheck(pr.Check, pc) {
				continue
			}
			pr.Check = pc
			if err := writeYAML(root, pr.File, pr); err != nil {
				return 0, 0, 0, 0, err
			}
			changed++
		}
		if SameCheck(f.Check, ch) {
			continue
		}
		f.Check = ch
		if err := writeYAML(root, f.File, f); err != nil {
			return 0, 0, 0, 0, err
		}
		changed++
	}
	return pass, fail, proofs, changed, nil
}

// SameCheck tells if the new check b gives the same result as the old check a. The date is not part of the result. A record with no old check has a new result.
func SameCheck(a, b *content.Check) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Result == b.Result && a.Toolchain == b.Toolchain && a.Mathlib == b.Mathlib && a.Log == b.Log &&
		slices.Equal(a.Axioms, b.Axioms) && a.Sorry == b.Sorry && a.NativeDecide == b.NativeDecide &&
		a.Comparator == b.Comparator && a.Lean4checker == b.Lean4checker && a.Note == b.Note
}

// Fidelity runs the steps of the fidelity pipeline that an agent may run (spec 08.4 steps 1 and 2) and records them. A statement that compiled with autoImplicit false and that no static check flagged becomes checker-clean, backed by one agent review for the source. A flagged statement stays unreviewed and gets a suspected defect. The function never sets a status above checker-clean, and it does not lower a status that a person set.
func Fidelity(root, source string, r *Results, date string) (clean, flagged int, err error) {
	corpus, fs := content.Load(root)
	if fs.Errors() > 0 {
		return 0, 0, fmt.Errorf("the content has errors:\n%v", fs)
	}
	var cleanTags, flagTags []string
	var todo []*content.Formal
	for _, f := range corpus.Formals {
		if f.Origin.Source != source {
			continue
		}
		if st := f.Fidelity.Status; st == content.OneHuman || st == content.TwoHuman || st == content.KnownWrong {
			continue
		}
		todo = append(todo, f)
	}
	cleanID := reviewID(source, content.DecisionClean, date)
	flagID := reviewID(source, content.DecisionFlagged, date)
	for _, f := range todo {
		compiled := f.Check != nil && f.Check.Result == content.Pass
		strict := f.Options["autoImplicit"] != "true"
		s1 := content.FidelityStep{Step: "compile", At: date, By: "agent", Result: content.Fail}
		switch {
		case compiled && strict:
			s1.Result = content.Pass
		case compiled:
			s1.Result = content.Fail
			s1.Note = "The file compiles only with autoImplicit true, so a name in the statement can be bound by accident."
		}
		s2 := content.FidelityStep{Step: "static-checks", At: date, By: "agent", Result: content.NotRun}
		flags := r.Flags[f.Decl]
		if r.Checked[f.Decl] {
			s2.Result = content.Pass
			if len(flags) > 0 {
				s2.Result = content.Fail
				s2.Note = "The tactics of the checks " + strings.Join(flags, " and ") + " prove the statement."
			}
		}
		steps := []content.FidelityStep{s1, s2}
		for _, s := range []string{"counterexample-search", "back-translation", "equivalence", "human-review"} {
			steps = append(steps, content.FidelityStep{Step: s, Result: content.NotRun, At: date, By: "agent"})
		}
		f.Fidelity.Steps = steps
		f.Fidelity.Reviews = slices.DeleteFunc(f.Fidelity.Reviews, func(id string) bool {
			rv := corpus.FidelityReviews[id]
			return rv == nil || rv.ByAgent()
		})
		f.Fidelity.Defects = slices.DeleteFunc(f.Fidelity.Defects, func(d content.Defect) bool { return d.By == "agent" })
		switch {
		case len(flags) > 0:
			for _, fl := range flags {
				f.Fidelity.Defects = append(f.Fidelity.Defects, content.Defect{Kind: fl, By: "agent", Note: content.AgentNote + " The tactics of the check " + fl + " prove the statement. A person must decide if the statement is wrong."})
			}
			f.Fidelity.Status = content.Unreviewed
			f.Fidelity.Reviews = append(f.Fidelity.Reviews, flagID)
			flagTags = append(flagTags, f.Tag)
			flagged++
		case s1.Result == content.Pass && s2.Result == content.Pass:
			f.Fidelity.Status = content.CheckerClean
			f.Fidelity.Reviews = append(f.Fidelity.Reviews, cleanID)
			cleanTags = append(cleanTags, f.Tag)
			clean++
		default:
			f.Fidelity.Status = content.Unreviewed
		}
		if err := writeYAML(root, f.File, f); err != nil {
			return 0, 0, err
		}
	}
	if len(cleanTags) > 0 {
		note := content.AgentNote + " The statements of " + source + " compiled at " + Toolchain + " and Mathlib " + Mathlib + " with autoImplicit false, and the static checks (" + strings.Join(checkNames(), ", ") + ") did not prove them. No person has compared them with the problems."
		if err := writeReview(root, cleanID, content.DecisionClean, date, note, cleanTags); err != nil {
			return 0, 0, err
		}
	}
	if len(flagTags) > 0 {
		note := content.AgentNote + " A static check proved each of these statements of " + source + ". A trivial proof is a sign of a defect, but it is not proof of one. A person must decide."
		if err := writeReview(root, flagID, content.DecisionFlagged, date, note, flagTags); err != nil {
			return 0, 0, err
		}
	}
	return clean, flagged, nil
}

// CheckNames are the static checks of check.sh.
var CheckNames = []string{"contradictory", "trivial-goal"}

func checkNames() []string { return CheckNames }

func reviewID(source, decision, date string) string {
	h := sha256.Sum256([]byte("fidelity\x00" + source + "\x00" + decision + "\x00" + date))
	return hex.EncodeToString(h[:])[:12]
}

type reviewRecord struct {
	ID       string   `yaml:"id"`
	Kind     string   `yaml:"kind"`
	Records  []string `yaml:"records,flow"`
	Decision string   `yaml:"decision"`
	By       string   `yaml:"by"`
	Date     string   `yaml:"date"`
	Note     string   `yaml:"note"`
}

func writeReview(root, id, decision, date, note string, tags []string) error {
	sort.Strings(tags)
	r := reviewRecord{ID: id, Kind: "fidelity", Records: tags, Decision: decision, By: "agent", Date: date, Note: note}
	return writeYAML(root, path.Join("reviews", date[:4], date[5:7], id+".yaml"), r)
}
