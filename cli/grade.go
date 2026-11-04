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

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/grade"
	"github.com/tamnd/olymmath-agent/model"
)

func init() {
	register(command{"grade", "grade the AI solutions with a jury of models and the rubric of the problem (spec 07.4); -dry-run prints the plan and the cost", runGrade})
}

// gradeJob is a solution to grade, with its file.
type gradeJob struct {
	sol  *content.Solution
	task grade.Task
}

func runGrade(env Env, args []string) int {
	fs := newFlags("grade", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dry := fs.Bool("dry-run", false, "print the plan and the cost estimate, and call no model")
	budget := fs.Float64("budget", 0, "the most the run may cost, in US dollars")
	jurors := fs.String("jurors", strings.Join(grade.DefaultJurors, ","), "the models of the jury, by commas. The model that wrote a solution is taken out.")
	batch := fs.Bool("batch", true, "use the Batch API, at half the price")
	work := fs.String("work", defaultWork(), "the directory of the call caches and the transcripts")
	limit := fs.Int("limit", 0, "the most solutions, or 0 for all")
	draft := fs.Bool("draft-ok", false, "also use rubrics with the status draft. A draft rubric is for a trial, not for grades at scale.")
	regrade := fs.Bool("regrade", false, "also grade the solutions that have a jury grade")
	human := fs.Bool("human", false, "also grade the human solutions of the named problems or solutions")
	effort := fs.String("effort", "", "the effort of the calls: low, medium, high or max")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om grade: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	jobs, skipped := gradeJobs(corpus, pos, strings.Split(*jurors, ","), *draft, *regrade, *human)
	if *limit > 0 && len(jobs) > *limit {
		jobs = jobs[:*limit]
	}
	tasks := make([]grade.Task, len(jobs))
	for i, j := range jobs {
		tasks[i] = j.task
	}
	opts := grade.Options{Effort: *effort}
	cost, calls, err := grade.Estimate(tasks, opts, *batch)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om grade: %v\n", err)
		return ExitUsage
	}
	fmt.Fprintf(env.Stdout, "%d solutions to grade, about %d calls and $%.2f. %d have no rubric that can be used.\n", len(jobs), calls, cost, skipped)
	if *dry || len(jobs) == 0 {
		return ExitOK
	}
	if *budget <= 0 {
		fmt.Fprintln(env.Stderr, "om grade: set -budget in US dollars, or use -dry-run")
		return ExitUsage
	}
	client, err := newClient(*batch)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om grade: %v\n", err)
		return ExitUsage
	}
	now := time.Now()
	rec, _ := openRun(corpus, *root, "grade", "", *budget, *batch, now)
	rec.Pipeline = grade.Pipeline
	if err := content.WriteRun(*root, rec); err != nil {
		fmt.Fprintf(env.Stderr, "om grade: %v\n", err)
		return ExitFailure
	}
	dir := filepath.Join(*work, rec.ID)
	b := &model.Budget{Client: client, Limit: *budget, Batch: *batch}
	results, err := grade.Run(context.Background(), &model.Cache{Client: b, Dir: filepath.Join(dir, "calls")}, tasks, opts)
	if err != nil {
		rec.Status = "failed"
		fmt.Fprintf(env.Stderr, "om grade: %v\n", err)
	}
	code := ExitOK
	graded, queued, failed := 0, 0, 0
	overBudget := false
	for i, r := range results {
		s := jobs[i].sol
		item := content.RunItem{Problem: s.Problem, Solution: s.Tag, Model: strings.Join(r.Task.Jurors, ","), Tokens: r.Tokens, Cost: r.Cost, Calls: len(r.Calls)}
		if tr, err := json.MarshalIndent(r.Calls, "", "  "); err == nil {
			item.Transcript = grade.Hash(string(tr))
			if os.MkdirAll(dir, 0o750) == nil {
				_ = os.WriteFile(filepath.Join(dir, s.Tag+".json"), tr, 0o600)
			}
		}
		switch {
		case r.Err != nil:
			item.Outcome, item.Note = "failed", r.Err.Error()
			failed++
			overBudget = overBudget || errors.Is(r.Err, model.ErrBudget)
		default:
			item.Outcome = "graded"
			if r.Queue {
				item.Outcome = "queued"
				queued++
			}
			graded++
			if err := writeGrade(*root, s, r.Grade(now.UTC().Format(time.DateOnly), rec.ID)); err != nil {
				fmt.Fprintf(env.Stderr, "om grade: %s: %v\n", s.Path, err)
				code = ExitFailure
			}
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
		fmt.Fprintf(env.Stderr, "om grade: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "run %s: %d graded, %d of them for the human queue, %d failed; $%.2f.\n", rec.ID, graded, queued, failed, rec.Cost)
	if err != nil {
		return ExitFailure
	}
	return code
}

// gradeJobs gives the solutions to grade: the AI solutions with no jury grade whose problem has a reviewed rubric, or the named solutions and the solutions of the named problems. It also counts the solutions that have no usable rubric.
func gradeJobs(corpus *content.Corpus, names []string, jurors []string, draftOK, regrade, human bool) ([]gradeJob, int) {
	named := map[string]bool{}
	for _, n := range names {
		named[strings.Trim(n, "/")] = true
	}
	var jobs []gradeJob
	skipped := 0
	for _, p := range corpus.All() {
		for _, s := range p.Solutions {
			switch {
			case len(named) > 0 && !named[s.Tag] && !named[p.Tag] && !named[p.Locator()]:
				continue
			case s.Origin != "ai" && (!human || len(named) == 0):
				continue
			case s.Grade != nil && !regrade:
				continue
			case strings.TrimSpace(s.Body) == "":
				continue
			}
			r := p.Rubric
			if r == nil || (r.Status != "reviewed" && !draftOK) {
				skipped++
				continue
			}
			gen := ""
			if s.Run != nil {
				gen = s.Run.Model
			}
			st := ""
			if c := p.CanonicalStatement(); c != nil {
				st = c.Text
			}
			jobs = append(jobs, gradeJob{s, grade.Task{
				ID: s.Tag, Statement: st, Rubric: grade.RubricText(r), Ref: r.Ref(),
				Solution: s.Body, Generator: gen, Jurors: grade.Jurors(jurors, gen),
			}})
		}
	}
	return jobs, skipped
}

// writeGrade sets the jury grade in the file of a solution.
func writeGrade(root string, s *content.Solution, g *content.Grade) error {
	path := filepath.Join(root, s.Path)
	b, err := os.ReadFile(path) // #nosec G304 -- a solution file inside the content root.
	if err != nil {
		return err
	}
	out, err := content.SetGrade(b, g)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644) // #nosec G306 G703 -- content files are public.
}
