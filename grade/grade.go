// Package grade is the grading jury of spec 07.4. At least two model graders score a solution against the rubric of its problem. When their scores differ by more than one point they read each other's notes and revise once. The jury score is the minimum; a difference that remains sends the solution to the human queue.
package grade

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/model"
)

// Rule is the rule of the jury.
const Rule = "min-of-jury-v1"

// Pipeline is the version of the grader prompts.
const Pipeline = "grade/1"

// Spread is the largest difference of the scores that needs no revision.
const Spread = 1

// DefaultJurors are the graders of the jury before the generator is taken out.
var DefaultJurors = []string{model.Sonnet, model.Haiku}

// Jurors gives the jury for a solution of the model gen: the jurors without gen, since a model does not grade its own output. When fewer than two are left, it adds the first of Opus, Sonnet and Haiku that is not there.
func Jurors(jurors []string, gen string) []string {
	var out []string
	for _, j := range jurors {
		if j != gen && !slices.Contains(out, j) {
			out = append(out, j)
		}
	}
	for _, m := range []string{model.Opus, model.Sonnet, model.Haiku} {
		if len(out) >= 2 {
			break
		}
		if m != gen && !slices.Contains(out, m) {
			out = append(out, m)
		}
	}
	return out
}

const system = `You are a strict olympiad grader. Grade the proposed solution of the problem below on the IMO scale from 0 to 7, with the rubric below.

Rules:
- Follow the rubric. Give the points of a checkpoint only when the solution proves it.
- A correct answer with no proof gets the points that the rubric gives for the answer only.
- Apply the deductions of the rubric.
- Do not reward length or style. A wrong step that the proof needs costs the points of the steps that rest on it.

Write short notes: for each checkpoint, met or not met, with the line of the solution. End with exactly one line "SCORE: N" with N from 0 to 7.`

// GenericRubric is the rubric of a problem that has no rubric file, for the calibration sets.
const GenericRubric = `7: a complete and correct proof.
5 to 6: a correct proof with a small error or omission that is easy to fix.
1 to 4: real progress, such as a key lemma or a correct reduction, but no complete proof.
0: no real progress, or a proof with a fatal error.`

// Task is one solution to grade.
type Task struct {
	// ID names the task, for example the tag of the solution.
	ID        string
	Statement string
	// Rubric is the text of the rubric and Ref its reference, for example imo/2024/p1@r2.
	Rubric string
	Ref    string
	// Reference is a reference solution, when the grader may see one.
	Reference string
	Solution  string
	// Generator is the model that wrote the solution, or empty for a person.
	Generator string
	Jurors    []string
}

// RubricText writes a rubric file as text for the prompt.
func RubricText(r *content.Rubric) string {
	var b strings.Builder
	b.WriteString("Checkpoints:\n")
	for _, c := range r.Checkpoints {
		fmt.Fprintf(&b, "- %d points: %s\n", c.Points, c.Text)
	}
	if len(r.ZeroCredit) > 0 {
		b.WriteString("No credit for:\n")
		for _, z := range r.ZeroCredit {
			fmt.Fprintf(&b, "- %s\n", z)
		}
	}
	if len(r.Deductions) > 0 {
		b.WriteString("Deductions:\n")
		for _, d := range r.Deductions {
			fmt.Fprintf(&b, "- %d points: %s\n", d.Points, d.Text)
		}
	}
	if n := strings.TrimSpace(r.Notes); n != "" {
		b.WriteString("Notes:\n" + n + "\n")
	}
	return b.String()
}

// Verdict is one grader's work on a task.
type Verdict struct {
	Model   string
	Score   int
	Notes   string
	Revised bool
}

// Result is the jury grade of a task.
type Result struct {
	Task     Task
	Verdicts []Verdict
	// Score is the minimum of the scores; Queue is true when they still differ by more than Spread.
	Score  int
	Queue  bool
	Calls  []Call
	Tokens content.Tokens
	Cost   float64
	// Err is set when a grader gave no score.
	Err error
}

// Call is one model call of a task, for the transcript.
type Call struct {
	Round  int            `json:"round"`
	Model  string         `json:"model"`
	Output string         `json:"output"`
	Usage  content.Tokens `json:"usage"`
	Cost   float64        `json:"cost_usd"`
	Cached bool           `json:"cached,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// Grade gives the content record of the result.
func (r *Result) Grade(at, runID string) *content.Grade {
	g := &content.Grade{Scale: "imo-7", Score: r.Score, Bucket: content.Bucket(r.Score, 7), Rubric: r.Task.Ref, At: at, Rule: Rule, Queue: r.Queue, RunID: runID}
	for _, v := range r.Verdicts {
		g.Graders = append(g.Graders, content.Judgement{By: "model:" + v.Model, Score: v.Score, NotesHash: Hash(v.Notes), Revised: v.Revised})
	}
	return g
}

// Hash gives the SHA-256 of a text.
func Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

var scoreRE = regexp.MustCompile(`SCORE:\s*([0-7])\b`)

// Score reads the last SCORE line of the notes of a grader.
func Score(notes string) (int, bool) {
	m := scoreRE.FindAllStringSubmatch(notes, -1)
	if m == nil {
		return 0, false
	}
	n, _ := strconv.Atoi(m[len(m)-1][1])
	return n, true
}

// Options are the knobs of the jury.
type Options struct {
	MaxTokens int
	Effort    string
}

func prefix(t Task) string {
	var b strings.Builder
	b.WriteString(system)
	b.WriteString("\n\nProblem:\n" + strings.TrimSpace(t.Statement))
	rub := t.Rubric
	if strings.TrimSpace(rub) == "" {
		rub = GenericRubric
	}
	b.WriteString("\n\nRubric:\n" + strings.TrimSpace(rub))
	if t.Reference != "" {
		b.WriteString("\n\nReference solution (one correct way; other correct ways get full marks):\n" + strings.TrimSpace(t.Reference))
	}
	return b.String()
}

func ask(t Task) string { return "Proposed solution:\n\n" + strings.TrimSpace(t.Solution) }

// Run grades the tasks. The first round sends every grader of every task in one list; the second round sends the revisions of the tasks whose scores differ by more than Spread.
func Run(ctx context.Context, client model.Client, tasks []Task, o Options) ([]*Result, error) {
	if o.MaxTokens == 0 {
		o.MaxTokens = 16000
	}
	results := make([]*Result, len(tasks))
	var reqs []model.Request
	type slot struct{ r, v int }
	var at []slot
	for i, t := range tasks {
		results[i] = &Result{Task: t}
		switch {
		case len(t.Jurors) < 2:
			results[i].Err = errors.New("the jury needs two graders")
			continue
		case t.Generator != "" && slices.Contains(t.Jurors, t.Generator):
			results[i].Err = fmt.Errorf("the model %s cannot grade its own solution", t.Generator)
			continue
		}
		for j, m := range t.Jurors {
			results[i].Verdicts = append(results[i].Verdicts, Verdict{Model: m})
			reqs = append(reqs, model.Request{ID: t.ID + "/" + m, Kind: "grade", Model: m, System: prefix(t), MaxTokens: o.MaxTokens, Effort: o.Effort,
				Messages: []model.Message{{Role: "user", Text: ask(t)}}})
			at = append(at, slot{i, j})
		}
	}
	got, err := client.Complete(ctx, reqs)
	if err != nil {
		return results, err
	}
	for k, res := range got {
		take(results[at[k].r], at[k].v, 1, reqs[k], res)
	}
	// Round 2: the graders of a split jury read the notes of the others and revise once.
	reqs, at = nil, nil
	for i, r := range results {
		if r.Err != nil || spread(r) <= Spread {
			continue
		}
		for j, v := range r.Verdicts {
			var others strings.Builder
			for k, w := range r.Verdicts {
				if k != j {
					fmt.Fprintf(&others, "Grader %d gave %d:\n%s\n\n", k+1, w.Score, strings.TrimSpace(w.Notes))
				}
			}
			reqs = append(reqs, model.Request{ID: r.Task.ID + "/" + v.Model + "/2", Kind: "grade", Model: v.Model, System: prefix(r.Task), MaxTokens: o.MaxTokens, Effort: o.Effort,
				Messages: []model.Message{
					{Role: "user", Text: ask(r.Task)},
					{Role: "assistant", Text: v.Notes},
					{Role: "user", Text: "The other graders of the jury wrote the notes below. Read them, check the points where you differ against the solution and the rubric, and grade once more. Keep your score if you still think it is right. End with one line \"SCORE: N\".\n\n" + others.String()},
				}})
			at = append(at, slot{i, j})
		}
	}
	if len(reqs) > 0 {
		got, err = client.Complete(ctx, reqs)
		if err != nil {
			return results, err
		}
		for k, res := range got {
			take(results[at[k].r], at[k].v, 2, reqs[k], res)
		}
	}
	for _, r := range results {
		if r.Err != nil || len(r.Verdicts) == 0 {
			continue
		}
		r.Score = r.Verdicts[0].Score
		for _, v := range r.Verdicts {
			r.Score = min(r.Score, v.Score)
		}
		r.Queue = spread(r) > Spread
	}
	return results, nil
}

func take(r *Result, v, round int, req model.Request, res model.Response) {
	c := Call{Round: round, Model: req.Model, Output: res.Text, Usage: res.Usage, Cached: res.Cached}
	if res.Err == nil {
		c.Cost, _ = model.Cost(req.Model, res.Usage, res.Batch)
	} else {
		c.Error = res.Err.Error()
	}
	r.Calls = append(r.Calls, c)
	r.Tokens.Add(res.Usage)
	r.Cost += c.Cost
	if res.Err != nil {
		if r.Err == nil {
			r.Err = res.Err
		}
		return
	}
	s, ok := Score(res.Text)
	if !ok {
		if r.Err == nil {
			r.Err = fmt.Errorf("the grader %s gave no score", req.Model)
		}
		return
	}
	vd := &r.Verdicts[v]
	if round == 2 {
		vd.Revised = s != vd.Score
	}
	vd.Score, vd.Notes = s, res.Text
}

func spread(r *Result) int {
	if len(r.Verdicts) == 0 {
		return 0
	}
	lo, hi := 7, 0
	for _, v := range r.Verdicts {
		lo, hi = min(lo, v.Score), max(hi, v.Score)
	}
	return hi - lo
}

// Estimate gives the expected cost and number of calls of one round of the jury on the tasks. A split jury adds a second round.
func Estimate(tasks []Task, o Options, batch bool) (float64, int, error) {
	if o.MaxTokens == 0 {
		o.MaxTokens = 16000
	}
	cost, calls := 0.0, 0
	for _, t := range tasks {
		for _, m := range t.Jurors {
			in := model.Tokens(prefix(t)) + model.Tokens(ask(t))
			// The expected notes are short; the thinking takes most of the output.
			c, err := model.Cost(m, content.Tokens{Input: in, Output: 4000}, batch)
			if err != nil {
				return 0, 0, err
			}
			cost += c
			calls++
		}
	}
	return cost, calls, nil
}
