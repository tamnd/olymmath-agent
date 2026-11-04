package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/model"
	"github.com/tamnd/olymmath-agent/solve"
)

func init() {
	register(command{"solve", "write AI solutions for the problems with no solution (spec 07.3); -dry-run prints the plan and the cost", runSolve})
}

// newClient gives the model client of om solve and om grade. Tests replace it with a fake.
var newClient = func(batch bool) (model.Client, error) {
	a, err := model.NewAnthropic()
	if err != nil {
		return nil, err
	}
	a.Batch = batch
	return a, nil
}

// defaultWork is the directory of the call caches and the transcripts. They stay out of the content repository.
func defaultWork() string {
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "olymmath", "runs")
	}
	return filepath.Join(os.TempDir(), "olymmath-runs")
}

// nextRunID gives the first free id of the form <kind>-YYYYMMDD-NNNN.
func nextRunID(corpus *content.Corpus, root, kind string, now time.Time) string {
	for n := 1; ; n++ {
		id := fmt.Sprintf("%s-%s-%04d", kind, now.Format("20060102"), n)
		if _, ok := corpus.Runs[id]; ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, "runs", id+".json")); err == nil {
			continue
		}
		return id
	}
}

// openRun gives the record of a new run, or of the run to resume.
func openRun(corpus *content.Corpus, root, kind, resume string, budget float64, batch bool, now time.Time) (*content.RunRecord, error) {
	if resume != "" {
		r := corpus.Runs[resume]
		if r == nil {
			return nil, fmt.Errorf("no run record runs/%s.json to resume", resume)
		}
		if r.Kind != kind {
			return nil, fmt.Errorf("the run %s is a %s run", resume, r.Kind)
		}
		r.Budget, r.Status, r.Ended, r.Batch = budget, "running", "", batch
		return r, nil
	}
	return &content.RunRecord{ID: nextRunID(corpus, root, kind, now), Kind: kind, Started: now.UTC().Format(time.RFC3339), Status: "running", Budget: budget, Batch: batch}, nil
}

func runSolve(env Env, args []string) int {
	fs := newFlags("solve", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dry := fs.Bool("dry-run", false, "print the plan and the cost estimate, and call no model")
	budget := fs.Float64("budget", 0, "the most the run may cost, in US dollars. The run stops before a call that could pass it.")
	k := fs.Int("k", 0, "the attempts for each problem (default 2, and 4 from band 8 up)")
	mdl := fs.String("model", "", "the generator for every problem (default "+model.Sonnet+", and "+model.Opus+" from band 8 up)")
	effort := fs.String("effort", "", "the effort of the calls: low, medium, high or max")
	limit := fs.Int("limit", 0, "the most problems, or 0 for all")
	comp := fs.String("comp", "", "only the problems of this competition")
	minBand := fs.Int("min-band", 0, "only the problems of this band or harder")
	maxBand := fs.Int("max-band", 0, "only the problems of this band or easier")
	batch := fs.Bool("batch", true, "use the Batch API, at half the price")
	work := fs.String("work", defaultWork(), "the directory of the call caches and the transcripts")
	resume := fs.String("resume", "", "the id of a run to go on with; the calls that ended come from the cache")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om solve: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	opts := solve.Options{Model: *mdl, K: *k, Effort: *effort}
	items := solve.Select(corpus, solve.Filter{Comp: *comp, Tags: pos, MinBand: *minBand, MaxBand: *maxBand, Limit: *limit}, opts)
	est, err := solve.Plan(items, opts, *batch)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om solve: %v\n", err)
		return ExitUsage
	}
	printPlan(env.Stdout, est, *batch)
	if *dry || len(items) == 0 {
		return ExitOK
	}
	if *budget <= 0 {
		fmt.Fprintln(env.Stderr, "om solve: set -budget in US dollars, or use -dry-run")
		return ExitUsage
	}
	client, err := newClient(*batch)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om solve: %v\n", err)
		return ExitUsage
	}
	now := time.Now()
	rec, err := openRun(corpus, *root, "solve", *resume, *budget, *batch, now)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om solve: %v\n", err)
		return ExitUsage
	}
	rec.Pipeline = solve.Pipeline
	if err := content.WriteRun(*root, rec); err != nil {
		fmt.Fprintf(env.Stderr, "om solve: %v\n", err)
		return ExitFailure
	}
	dir := filepath.Join(*work, rec.ID)
	b := &model.Budget{Client: client, Limit: *budget, Batch: *batch}
	// The calls of the run so far were paid; the cache gives them again at no cost.
	b.Charge(rec.Cost)
	results, err := solve.Run(context.Background(), &model.Cache{Client: b, Dir: filepath.Join(dir, "calls")}, items, opts)
	if err != nil {
		rec.Status = "failed"
		fmt.Fprintf(env.Stderr, "om solve: %v\n", err)
	}
	code := writeSolve(env, corpus, *root, dir, rec, results, now)
	if err != nil {
		return ExitFailure
	}
	return code
}

func printPlan(w io.Writer, e solve.Estimate, batch bool) {
	api := "the Messages API"
	if batch {
		api = "the Batch API"
	}
	fmt.Fprintf(w, "band\tproblems\tcalls\tcost (USD)\n")
	n := 0
	for _, b := range e.Bands {
		band := fmt.Sprint(b.Band)
		if b.Band == 0 {
			band = "none"
		}
		fmt.Fprintf(w, "%s\t%d\t%d\t%.2f\n", band, b.Problems, b.Calls, b.Cost)
		n += b.Problems
	}
	fmt.Fprintf(w, "%d problems, about %d calls and $%.2f with %s. The estimate takes all attempts and one revision each; the pilot measures the real cost.\n", n, e.Calls, e.Cost, api)
}

// writeSolve writes the accepted solutions, the transcripts and the run record.
func writeSolve(env Env, corpus *content.Corpus, root, dir string, rec *content.RunRecord, results []*solve.Result, now time.Time) int {
	byProblem := map[string]int{}
	for i, it := range rec.Items {
		byProblem[it.Problem] = i
	}
	counts := map[string]int{}
	code := ExitOK
	for _, r := range results {
		counts[r.Outcome]++
		solTag := ""
		if r.Outcome == solve.Accepted {
			t, err := writeAISolution(corpus, root, rec.ID, r, now)
			if err != nil {
				fmt.Fprintf(env.Stderr, "om solve: %s: %v\n", r.Problem.Locator(), err)
				code = ExitFailure
			}
			solTag = t
		}
		b, _ := r.Transcript()
		if err := os.MkdirAll(dir, 0o750); err == nil {
			_ = os.WriteFile(filepath.Join(dir, r.Problem.Tag+".json"), b, 0o600)
		}
		item := r.RunItem(solTag)
		if i, ok := byProblem[item.Problem]; ok {
			rec.Items[i] = item
		} else {
			byProblem[item.Problem] = len(rec.Items)
			rec.Items = append(rec.Items, item)
		}
	}
	rec.Cost, rec.Tokens = 0, content.Tokens{}
	for _, it := range rec.Items {
		rec.Cost += it.Cost
		rec.Tokens.Add(it.Tokens)
	}
	if rec.Status == "running" {
		rec.Status = "done"
		if counts[solve.OverBudget] > 0 {
			rec.Status = "budget"
		}
	}
	rec.Ended = time.Now().UTC().Format(time.RFC3339)
	if err := content.WriteRun(root, rec); err != nil {
		fmt.Fprintf(env.Stderr, "om solve: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "run %s: %d accepted, %d unsolved, %d mismatch, %d over budget, %d failed; $%.2f. The transcripts are in %s.\n",
		rec.ID, counts[solve.Accepted], counts[solve.Unsolved], counts[solve.Mismatch], counts[solve.OverBudget], counts[solve.Failed], rec.Cost, dir)
	if rec.Status == "budget" {
		fmt.Fprintf(env.Stdout, "The run stopped at its budget. Go on with om solve -resume %s -budget N.\n", rec.ID)
	}
	return code
}

// writeAISolution writes the file of an accepted solution and registers its tag.
func writeAISolution(corpus *content.Corpus, root, runID string, r *solve.Result, now time.Time) (string, error) {
	p := r.Problem
	if dup := aiSolutionOf(p, runID); dup != "" {
		return dup, nil
	}
	tags, err := mintTags(corpus, 1)
	if err != nil {
		return "", err
	}
	t := tags[0]
	_, hash := r.Transcript()
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntag: %s\nkind: solution\nproblem: %s\norigin: ai\nauthor:\n  display: %s\nzone: Z1\nlang: en\ncomplete: true\n", t, p.Tag, r.Model)
	if r.Answer != "" {
		fmt.Fprintf(&b, "answer_claimed: %s\n", yamlString(r.Answer))
	}
	if r.AnswerCheck != "" {
		fmt.Fprintf(&b, "answer_check: %s\n", r.AnswerCheck)
	}
	if r.Follows != "" {
		fmt.Fprintf(&b, "follows: %s\n", r.Follows)
	}
	fmt.Fprintf(&b, "run:\n  model: %s\n  prompt: %s\n  pipeline: %s\n  date: %s\n  run_id: %s\n  transcript: %s\n  tokens: {input: %d, output: %d, cache_read: %d, cache_write: %d}\n  cost: %.4f\n",
		r.Model, solve.Pipeline, solve.Pipeline, now.UTC().Format(time.DateOnly), runID, hash, r.Tokens.Input, r.Tokens.Output, r.Tokens.CacheRead, r.Tokens.CacheWrite, r.Cost)
	if c := model.Cutoffs[r.Model]; c != "" {
		fmt.Fprintf(&b, "  cutoff: %s\n", c)
	}
	fmt.Fprintf(&b, "---\n%s\n", r.Text)
	out, err := content.Format([]byte(b.String()))
	if err != nil {
		return "", err
	}
	rel := filepath.Join("solutions", strings.TrimSuffix(strings.TrimPrefix(p.Path, "problems/"), ".md"), t+".md")
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil { // #nosec G306 G703 -- content files are public.
		return "", err
	}
	if err := addTag(root, t, "solution"); err != nil {
		return "", err
	}
	corpus.Tags[t] = "solution"
	return t, nil
}

// aiSolutionOf gives the tag of the solution that a run already wrote for a problem, so a resumed run does not write it twice.
func aiSolutionOf(p *content.Problem, runID string) string {
	for _, s := range p.Solutions {
		if s.Run != nil && s.Run.RunID == runID {
			return s.Tag
		}
	}
	return ""
}

// yamlString quotes a value for a YAML line.
func yamlString(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}
