package cli

import (
	"fmt"
	"strings"

	"github.com/tamnd/olymmath-agent/audit"
)

func init() {
	register(command{"audit", "run the audit rules of spec 09.3 (om audit -rule T02,S03)", runAudit})
}

func runAudit(env Env, args []string) int {
	fs := newFlags("audit", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	rules := fs.String("rule", "", "the rules to run, separated by commas; empty runs all the rules")
	list := fs.Bool("list", false, "list the rules and stop")
	quiet := fs.Bool("q", false, "print only the findings")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if *list {
		for _, r := range audit.Rules {
			fmt.Fprintf(env.Stdout, "%s  %-7s  %s\n", r.ID, r.Level, r.Summary)
		}
		return ExitOK
	}
	var ids []string
	for _, id := range strings.Split(*rules, ",") {
		if id = strings.TrimSpace(strings.ToUpper(id)); id != "" {
			ids = append(ids, id)
		}
	}
	corpus, findings, err := audit.Run(*root, ids...)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om audit: %v\n", err)
		return ExitUsage
	}
	for _, f := range findings {
		fmt.Fprintln(env.Stdout, f)
	}
	errs := findings.Errors()
	if !*quiet {
		fmt.Fprintf(env.Stdout, "%d problems, %d errors, %d warnings and notes\n", len(corpus.All()), errs, len(findings)-errs)
	}
	if errs > 0 {
		return ExitFindings
	}
	return ExitOK
}
