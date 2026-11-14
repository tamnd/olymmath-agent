package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/grade"
	"github.com/tamnd/olymmath-agent/model"
)

func init() {
	register(command{"rubric", "draft the rubric of a problem with a model, as a draft that a person reviews (spec 07.4); -dry-run prints the plan and the cost", runRubric})
}

func runRubric(env Env, args []string) int {
	fs := newFlags("rubric", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dry := fs.Bool("dry-run", false, "print the plan and the cost estimate, and call no model")
	budget := fs.Float64("budget", 0, "the most the run may cost, in US dollars")
	mdl := fs.String("model", model.Opus, "the model that drafts the rubrics. A model cli:claude, cli:codex or cli:gemini, or cli:<client>/<model>, runs a local command-line client with its own login.")
	effort := fs.String("effort", "", "the effort of the calls: low, medium, high or max")
	batch := fs.Bool("batch", true, "use the Batch API, at half the price")
	work := fs.String("work", defaultWork(), "the directory of the call caches and the transcripts")
	limit := fs.Int("limit", 0, "the most problems, or 0 for all")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if err := checkJurors([]string{*mdl}); err != nil {
		fmt.Fprintf(env.Stderr, "om rubric: %v\n", err)
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om rubric: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	probs := rubricProblems(corpus, pos)
	if *limit > 0 && len(probs) > *limit {
		probs = probs[:*limit]
	}
	reqs := make([]model.Request, len(probs))
	cost := 0.0
	for i, p := range probs {
		reqs[i] = grade.DraftRequest(rubricDraft(p), *mdl, *effort)
		c, err := model.Estimate(reqs[i], *batch)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om rubric: %v\n", err)
			return ExitUsage
		}
		cost += c
	}
	fmt.Fprintf(env.Stdout, "%d problems need a rubric, about %d calls and $%.2f.\n", len(probs), len(probs), cost)
	if *dry || len(probs) == 0 {
		return ExitOK
	}
	if *budget <= 0 {
		fmt.Fprintln(env.Stderr, "om rubric: set -budget in US dollars, or use -dry-run")
		return ExitUsage
	}
	client, err := juryClient([]string{*mdl}, *batch, *work)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om rubric: %v\n", err)
		return ExitUsage
	}
	now := time.Now()
	rec, _ := openRun(corpus, *root, "rubric", "", *budget, *batch, now)
	rec.Pipeline = grade.DraftPipeline
	if err := content.WriteRun(*root, rec); err != nil {
		fmt.Fprintf(env.Stderr, "om rubric: %v\n", err)
		return ExitFailure
	}
	dir := filepath.Join(*work, rec.ID)
	b := &model.Budget{Client: client, Limit: *budget, Batch: *batch}
	res, err := (&model.Cache{Client: b, Dir: filepath.Join(dir, "calls"), Each: allCLI([]string{*mdl})}).Complete(context.Background(), reqs)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om rubric: %v\n", err)
		rec.Status = "failed"
	}
	code := ExitOK
	drafted, failed := 0, 0
	overBudget := false
	for i, r := range res {
		p := probs[i]
		writer := r.Model
		if writer == "" {
			writer = cliName(*mdl)
		}
		item := content.RunItem{Problem: p.Tag, Model: writer, Tokens: r.Usage, Calls: 1, Seconds: r.Seconds}
		if tr, err := json.MarshalIndent(r, "", "  "); err == nil {
			item.Transcript = grade.Hash(string(tr))
			if os.MkdirAll(dir, 0o750) == nil {
				_ = os.WriteFile(filepath.Join(dir, p.Tag+".json"), tr, 0o600)
			}
		}
		var rub *content.Rubric
		if r.Err == nil {
			item.Cost, _ = model.Cost(*mdl, r.Usage, r.Batch)
			rub, r.Err = grade.ParseDraft(r.Text)
		}
		if r.Err == nil {
			rub.Problem, rub.DraftedBy = p.Tag, "model:"+writer
			if err := writeRubric(*root, p, rub); err != nil {
				fmt.Fprintf(env.Stderr, "om rubric: %s: %v\n", p.Locator(), err)
				code = ExitFailure
				r.Err = err
			}
		}
		if r.Err != nil {
			item.Outcome, item.Note = "failed", r.Err.Error()
			failed++
			overBudget = overBudget || errors.Is(r.Err, model.ErrBudget)
		} else {
			item.Outcome = "drafted"
			drafted++
		}
		rec.Items = append(rec.Items, item)
		rec.Cost += item.Cost
		rec.Tokens.Add(item.Tokens)
	}
	if rec.Status == "running" {
		rec.Status = "done"
		if overBudget {
			rec.Status = "budget"
		}
	}
	rec.Ended = time.Now().UTC().Format(time.RFC3339)
	if err := content.WriteRun(*root, rec); err != nil {
		fmt.Fprintf(env.Stderr, "om rubric: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "run %s: %d rubrics drafted, %d failed; $%.2f. The rubrics have the status draft until a person reviews them.\n", rec.ID, drafted, failed, rec.Cost)
	if err != nil {
		return ExitFailure
	}
	return code
}

// rubricProblems gives the problems that need a rubric: the named problems, or else the problems with an AI solution that has no grade. A problem that has a rubric is left out.
func rubricProblems(corpus *content.Corpus, names []string) []*content.Problem {
	named := map[string]bool{}
	for _, n := range names {
		named[strings.Trim(n, "/")] = true
	}
	var out []*content.Problem
	for _, p := range corpus.All() {
		if p.Rubric != nil || p.CanonicalStatement() == nil {
			continue
		}
		if len(named) > 0 {
			if named[p.Tag] || named[p.Locator()] {
				out = append(out, p)
			}
			continue
		}
		for _, s := range p.Solutions {
			if s.Origin == "ai" && s.Grade == nil {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

// rubricDraft gives the input of the drafter: the statement, the answer and the human solutions of the problem.
func rubricDraft(p *content.Problem) grade.Draft {
	d := grade.Draft{ID: p.Tag, Answer: p.Answer}
	if s := p.CanonicalStatement(); s != nil {
		d.Statement = s.Text
	}
	for _, s := range p.Solutions {
		if s.Origin != "ai" && strings.TrimSpace(s.Body) != "" {
			d.References = append(d.References, s.Body)
		}
	}
	return d
}

// cliName gives the model that a cli:<client>/<model> name names, or the name itself.
func cliName(m string) string {
	if model.IsCLI(m) {
		if _, name, err := model.ParseCLI(m); err == nil && name != "" {
			return name
		}
	}
	return m
}

// writeRubric writes rubrics/<locator>.md.
func writeRubric(root string, p *content.Problem, r *content.Rubric) error {
	var b strings.Builder
	b.WriteString("---\n")
	e := yaml.NewEncoder(&b)
	e.SetIndent(2)
	if err := e.Encode(r); err != nil {
		return err
	}
	if err := e.Close(); err != nil {
		return err
	}
	b.WriteString("---\n")
	if r.Notes != "" {
		b.WriteString(r.Notes + "\n")
	}
	path := filepath.Join(root, "rubrics", p.Locator()+".md")
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644) // #nosec G306 G703 -- content files are public.
}
