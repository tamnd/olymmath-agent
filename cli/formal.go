package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/formal"
)

func init() {
	register(command{"formal", "import and check formal statements (om formal import, lakefile, targets, script, record, fidelity)", runFormal})
}

const formalUsage = `usage: om formal SUB [flags]

  import    copy a Lean source to lean/vendor and write its formal and proof records
  lakefile  write lean/lakefile.toml and lean/lean-toolchain for the vendored sources
  targets   print the declarations of a source that the checks look at
  script    print the check script that runs on the Lean host
  record    write the results of the check script into the records of a source whose result changed
  fidelity  run the agent steps of the fidelity pipeline on a source and write the agent reviews
`

func runFormal(env Env, args []string) int {
	if len(args) == 0 || strings.HasPrefix(args[0], "-") {
		fmt.Fprint(env.Stderr, formalUsage)
		return ExitUsage
	}
	sub, args := args[0], args[1:]
	fs := newFlags("formal "+sub, env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	src := fs.String("source", "", "the id of the source: "+strings.Join(sourceIDs(), ", "))
	up := fs.String("upstream", "", "import: the root of a clone of the source")
	commit := fs.String("commit", "", "import: the commit of the clone")
	results := fs.String("results", "", "record and fidelity: the directory of the outputs of the check script")
	records := fs.Bool("records", true, "import: write the formal and proof records; with -records=false, copy the files and write the report only")
	date := fs.String("date", time.Now().UTC().Format("2006-01-02"), "the date of the records")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	need := func() (*formal.Source, bool) {
		s := formal.Sources[*src]
		if s == nil {
			fmt.Fprintf(env.Stderr, "om formal %s: give -source, one of %s.\n", sub, strings.Join(sourceIDs(), ", "))
		}
		return s, s != nil
	}
	fail := func(err error) int {
		fmt.Fprintf(env.Stderr, "om formal %s: %v\n", sub, err)
		return ExitFailure
	}
	switch sub {
	case "import":
		s, ok := need()
		if !ok || *up == "" || *commit == "" {
			fmt.Fprintln(env.Stderr, "om formal import: give -source, -upstream and -commit.")
			return ExitUsage
		}
		rep, err := formal.Import(*root, s, *up, *commit, *records)
		if err != nil {
			return fail(err)
		}
		by := map[string]int{}
		for _, r := range rep.Rows {
			by[r.Status]++
		}
		fmt.Fprintf(env.Stdout, "%s at %s: %d files, %d statements, %d mapped, %d with no locator, %d with no problem, %d proofs, %d new tags. The report is %s.\n", s.ID, *commit, rep.Files, len(rep.Rows), rep.Mapped, by["no-locator"], by["no-problem"], rep.Proofs, rep.NewTags, formal.ReportPath(s.ID))
	case "lakefile":
		vs, err := formal.FindVendored(*root)
		if err != nil {
			return fail(err)
		}
		dir := filepath.Join(*root, content.LeanDir)
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lakefile.toml"), []byte(formal.Lakefile(vs, nil)), 0o600); err != nil {
			return fail(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "lean-toolchain"), []byte(formal.Toolchain+"\n"), 0o600); err != nil {
			return fail(err)
		}
		fmt.Fprintf(env.Stdout, "wrote lean/lakefile.toml with %d vendored sources\n", len(vs))
	case "targets":
		if _, ok := need(); !ok {
			return ExitUsage
		}
		corpus, f := content.Load(*root)
		if f.Errors() > 0 {
			return fail(fmt.Errorf("the content has %d errors", f.Errors()))
		}
		fmt.Fprint(env.Stdout, formal.TargetsFile(formal.Targets(corpus, *src)))
	case "script":
		fmt.Fprint(env.Stdout, formal.CheckScript)
	case "record", "fidelity":
		s, ok := need()
		if !ok || *results == "" {
			fmt.Fprintf(env.Stderr, "om formal %s: give -source and -results.\n", sub)
			return ExitUsage
		}
		r, err := formal.ReadResults(*results, s.Lib)
		if err != nil {
			return fail(err)
		}
		if sub == "record" {
			pass, nfail, proofs, changed, err := formal.Record(*root, s.ID, r, *date)
			if err != nil {
				return fail(err)
			}
			fmt.Fprintf(env.Stdout, "%s: %d statements compiled, %d did not, %d proofs passed, %d records changed\n", s.ID, pass, nfail, proofs, changed)
			return ExitOK
		}
		clean, flagged, err := formal.Fidelity(*root, s.ID, r, *date)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(env.Stdout, "%s: %d checker-clean, %d flagged by a static check\n", s.ID, clean, flagged)
	default:
		fmt.Fprint(env.Stderr, formalUsage)
		return ExitUsage
	}
	return ExitOK
}

func sourceIDs() []string {
	var out []string
	for id := range formal.Sources {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
