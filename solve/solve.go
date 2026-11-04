// Package solve is the solver agent of spec 07.3. It picks the problems that need an AI solution, estimates the cost, and runs the stages S0 to S5 for many problems at once, so each round of calls can go out as one batch.
package solve

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/audit"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/model"
)

// HardBand is the least difficulty band that gets the strong model and four attempts (spec 07.3).
const HardBand = 8

// Options are the knobs of a run. A zero field takes its default.
type Options struct {
	// Model is the generator of every problem. Empty means Sonnet, and Opus from HardBand up.
	Model string
	// K is the number of attempts of every problem. 0 means 2, and 4 from HardBand up.
	K int
	// Effort is the effort of the calls, or empty for the default of the model.
	Effort string
	// Rounds is the most revisions of an attempt (S3). 0 means 5.
	Rounds int
	// Passes is the number of verifier passes in a row that accept an attempt. 0 means 3.
	Passes int
	// GenTokens and VerifyTokens are the max_tokens of the calls. They include the thinking of the model.
	GenTokens    int
	VerifyTokens int
}

func (o Options) withDefaults() Options {
	if o.Rounds == 0 {
		o.Rounds = 5
	}
	if o.Passes == 0 {
		o.Passes = 3
	}
	if o.GenTokens == 0 {
		o.GenTokens = 32000
	}
	if o.VerifyTokens == 0 {
		o.VerifyTokens = 16000
	}
	return o
}

// Item is one problem of a run.
type Item struct {
	Problem *content.Problem
	Band    int
	Model   string
	K       int
}

// Statement gives the text that the solver sees: the canonical statement only (S0).
func (it Item) Statement() string {
	if s := it.Problem.CanonicalStatement(); s != nil {
		return strings.TrimSpace(s.Text)
	}
	return ""
}

// Filter chooses the problems of a run.
type Filter struct {
	// Comp keeps the problems of one competition, or all when empty.
	Comp string
	// Tags keeps the problems with these tags or locators, when not empty. A named problem is kept even when it has a human solution.
	Tags []string
	// MinBand and MaxBand keep the problems with a band in the range. 0 means no bound.
	MinBand, MaxBand int
	// Limit is the most problems, or 0 for all.
	Limit int
}

// Select gives the problems of priority P1 of spec 07.2: no solution at all, and a statement with a text. They come in the order of their locators.
func Select(c *content.Corpus, f Filter, o Options) []Item {
	named := map[string]bool{}
	for _, t := range f.Tags {
		named[strings.Trim(t, "/")] = true
	}
	var out []Item
	for _, p := range c.All() {
		switch {
		case len(named) > 0 && !named[p.Tag] && !named[p.Locator()]:
			continue
		case len(named) == 0 && len(p.Solutions) > 0:
			continue
		case f.Comp != "" && p.Competition != f.Comp:
			continue
		}
		it := Item{Problem: p, Band: p.Band()}
		if it.Statement() == "" || (f.MinBand > 0 && it.Band < f.MinBand) || (f.MaxBand > 0 && it.Band > f.MaxBand) {
			continue
		}
		it.Model, it.K = o.Model, o.K
		if it.Model == "" {
			it.Model = model.Sonnet
			if it.Band >= HardBand {
				it.Model = model.Opus
			}
		}
		if it.K == 0 {
			it.K = 2
			if it.Band >= HardBand {
				it.K = 4
			}
		}
		out = append(out, it)
	}
	slices.SortFunc(out, func(a, b Item) int { return cmp.Compare(a.Problem.Locator(), b.Problem.Locator()) })
	if f.Limit > 0 && len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out
}

// The expected sizes of the calls, for the estimate. The pilot of 500 problems measures them (spec 07.6).
const (
	// ExpectGen is the expected output of a generation or a revision, with the thinking.
	ExpectGen = 8000
	// ExpectVerify is the expected output of a verifier call.
	ExpectVerify = 3000
	// ExpectRevisions is the expected number of revisions of an attempt.
	ExpectRevisions = 1
)

// Estimate is the expected cost of a plan.
type Estimate struct {
	// Bands has one row for each difficulty band, the band 0 for the problems with no band.
	Bands []BandEstimate
	Calls int
	Cost  float64
}

// BandEstimate is the estimate of one band.
type BandEstimate struct {
	Band     int
	Problems int
	Calls    int
	Cost     float64
}

// Plan estimates the cost of the items: all K attempts, ExpectRevisions revisions each, and the verifier passes after the first draft and after each revision. The prefix of each call is a cache write the first time and a cache read after.
func Plan(items []Item, o Options, batch bool) (Estimate, error) {
	o = o.withDefaults()
	by := map[int]*BandEstimate{}
	var e Estimate
	for _, it := range items {
		st := model.Tokens(it.Statement())
		gen := model.Tokens(generateSystem) + st
		ver := model.Tokens(verifySystem) + st
		drafts := 1 + ExpectRevisions
		var t content.Tokens
		// The first attempt writes the cache; the other calls read it.
		t.CacheWrite += gen + ver
		t.CacheRead += gen*(it.K*drafts-1) + ver*(it.K*drafts*o.Passes-1)
		t.Input += it.K * ExpectRevisions * (ExpectGen + ExpectVerify)
		t.Input += it.K * drafts * o.Passes * ExpectGen
		t.Output += it.K * (drafts*ExpectGen + drafts*o.Passes*ExpectVerify)
		c, err := model.Cost(it.Model, t, batch)
		if err != nil {
			return e, err
		}
		calls := it.K * drafts * (1 + o.Passes)
		b := by[it.Band]
		if b == nil {
			b = &BandEstimate{Band: it.Band}
			by[it.Band] = b
		}
		b.Problems++
		b.Calls += calls
		b.Cost += c
		e.Calls += calls
		e.Cost += c
	}
	for _, b := range by {
		e.Bands = append(e.Bands, *b)
	}
	slices.SortFunc(e.Bands, func(a, b BandEstimate) int { return cmp.Compare(a.Band, b.Band) })
	return e, nil
}

// The outcomes of a problem.
const (
	// Accepted means that an attempt passed the verifier and the answer check.
	Accepted = "accepted"
	// Unsolved means that no attempt passed.
	Unsolved = "unsolved"
	// Mismatch means that the only attempts that passed the verifier have another answer than the problem.
	Mismatch = "mismatch"
	// OverBudget means that the run stopped at its budget before the problem ended.
	OverBudget = "budget"
	// Failed means that the calls failed.
	Failed = "failed"
)

// Call is one model call of a problem, for the transcript.
type Call struct {
	Stage   string          `json:"stage"`
	Attempt int             `json:"attempt"`
	Model   string          `json:"model"`
	Key     string          `json:"key"`
	Input   []model.Message `json:"input"`
	Output  string          `json:"output"`
	Usage   content.Tokens  `json:"usage"`
	Cost    float64         `json:"cost_usd"`
	Cached  bool            `json:"cached,omitempty"`
	Error   string          `json:"error,omitempty"`
}

// Result is the end of one problem.
type Result struct {
	Item
	Outcome string
	// Stage is the last stage that the problem reached.
	Stage string
	// Text is the accepted solution, and Attempt its number.
	Text    string
	Attempt int
	// Answer is the boxed answer of the solution, and AnswerCheck the result of S4: match, mismatch, none or n/a.
	Answer      string
	AnswerCheck string
	// Follows is the tag of the human solution that the solution closely follows (S5).
	Follows string
	Note    string
	Tokens  content.Tokens
	Cost    float64
	Calls   []Call
}

// Transcript gives the calls as JSON and its SHA-256.
func (r *Result) Transcript() ([]byte, string) {
	b, _ := json.MarshalIndent(r.Calls, "", "  ")
	h := sha256.Sum256(b)
	return b, hex.EncodeToString(h[:])
}

// RunItem gives the line of the run record.
func (r *Result) RunItem(solution string) content.RunItem {
	_, h := r.Transcript()
	return content.RunItem{
		Problem: r.Problem.Tag, Solution: solution, Band: r.Band, Model: r.Model,
		Stage: r.Stage, Outcome: r.Outcome, Tokens: r.Tokens, Cost: r.Cost, Calls: len(r.Calls),
		Transcript: h, Note: r.Note,
	}
}

// The states of an attempt.
const (
	needGen = iota
	needVerify
	needRevise
	accepted
	rejected
)

type attempt struct {
	res    *Result
	n      int
	state  int
	text   string
	report string
	passes int
	checks int
	rounds int
	// mismatch is true when the attempt passed the verifier with another answer than the problem.
	mismatch bool
}

func (a *attempt) genSystem() string {
	p := a.res.Problem
	var b strings.Builder
	b.WriteString(generateSystem)
	b.WriteString("\n\nProblem:\n")
	b.WriteString(a.res.Statement())
	if p.Answer != "" {
		fmt.Fprintf(&b, "\n\nThe answer is known: %s. Your solution must prove it.", p.Answer)
	}
	return b.String()
}

func (a *attempt) firstAsk() string {
	return fmt.Sprintf("Write your solution. This is attempt %d.", a.n)
}

func (a *attempt) request(o Options) model.Request {
	r := model.Request{ID: fmt.Sprintf("%s/%d", a.res.Problem.Tag, a.n), Model: a.res.Model, Effort: o.Effort}
	switch a.state {
	case needGen:
		r.Kind, r.System, r.MaxTokens = "generate", a.genSystem(), o.GenTokens
		r.Messages = []model.Message{{Role: "user", Text: a.firstAsk()}}
	case needVerify:
		r.Kind, r.MaxTokens = "verify", o.VerifyTokens
		r.System = verifySystem + "\n\nProblem:\n" + a.res.Statement()
		r.Messages = []model.Message{{Role: "user", Text: fmt.Sprintf("Proposed solution:\n\n%s\n\nThis is check %d.", a.text, a.checks+1)}}
	case needRevise:
		r.Kind, r.System, r.MaxTokens = "revise", a.genSystem(), o.GenTokens
		r.Messages = []model.Message{
			{Role: "user", Text: a.firstAsk()},
			{Role: "assistant", Text: a.text},
			{Role: "user", Text: reviseInstruction + "\n\nBug report:\n\n" + a.report},
		}
	}
	return r
}

var stageOf = map[string]string{"generate": "S1", "verify": "S2", "revise": "S3"}

// Run runs the stages S0 to S5 for the items. Each round sends the next call of every open attempt in one list, so a batch client sends one batch a round. An attempt is accepted when the verifier passes it Passes times in a row; a problem ends at its first accepted attempt.
func Run(ctx context.Context, client model.Client, items []Item, o Options) ([]*Result, error) {
	o = o.withDefaults()
	var results []*Result
	var atts []*attempt
	for _, it := range items {
		r := &Result{Item: it, Stage: "S0"}
		results = append(results, r)
		for n := 1; n <= it.K; n++ {
			atts = append(atts, &attempt{res: r, n: n})
		}
	}
	for {
		var reqs []model.Request
		var who []*attempt
		for _, a := range atts {
			if a.res.Outcome != "" || a.state == accepted || a.state == rejected {
				continue
			}
			reqs = append(reqs, a.request(o))
			who = append(who, a)
		}
		if len(reqs) == 0 {
			break
		}
		got, err := client.Complete(ctx, reqs)
		if err != nil {
			return results, err
		}
		for i, res := range got {
			step(who[i], reqs[i], res, o)
		}
		for _, r := range results {
			end(r, atts)
		}
	}
	for _, r := range results {
		end(r, atts)
	}
	return results, nil
}

// step takes the response of one call of an attempt.
func step(a *attempt, req model.Request, res model.Response, o Options) {
	r := a.res
	c := Call{Stage: stageOf[req.Kind], Attempt: a.n, Model: req.Model, Key: req.Key(), Input: req.Messages, Output: res.Text, Usage: res.Usage, Cached: res.Cached}
	if res.Err == nil {
		c.Cost, _ = model.Cost(req.Model, res.Usage, res.Batch)
	} else {
		c.Error = res.Err.Error()
	}
	r.Calls = append(r.Calls, c)
	r.Tokens.Add(res.Usage)
	r.Cost += c.Cost
	if later(c.Stage, r.Stage) {
		r.Stage = c.Stage
	}
	switch {
	case errors.Is(res.Err, model.ErrBudget):
		r.Outcome, r.Note = OverBudget, "the run stopped at its budget"
		return
	case res.Err != nil:
		a.state = rejected
		r.Note = res.Err.Error()
		return
	}
	switch req.Kind {
	case "generate", "revise":
		if req.Kind == "revise" {
			a.rounds++
		}
		a.text, a.passes, a.state = strings.TrimSpace(res.Text), 0, needVerify
		if res.StopReason == "max_tokens" {
			a.report = "The solution was cut at the token limit. Write it again, shorter."
			a.state = needRevise
			if a.rounds >= o.Rounds {
				a.state = rejected
			}
		}
	case "verify":
		a.checks++
		if Verdict(res.Text) {
			a.passes++
			if a.passes >= o.Passes {
				a.accept()
			}
			return
		}
		a.passes, a.report = 0, strings.TrimSpace(res.Text)
		a.state = needRevise
		if a.rounds >= o.Rounds {
			a.state = rejected
		}
	}
}

// accept runs the answer check (S4) on an attempt that passed the verifier.
func (a *attempt) accept() {
	r := a.res
	p := r.Problem
	r.Stage = "S4"
	ans := Boxed(a.text)
	if p.Answer != "" && ans != "" && match.Answers(p.Answer, ans) == match.AnswerMismatch {
		a.state, a.mismatch = rejected, true
		return
	}
	a.state = accepted
}

// end closes a problem when an attempt is accepted or when all are rejected. It runs the dedupe (S5) on the accepted solution.
func end(r *Result, atts []*attempt) {
	if r.Outcome != "" {
		return
	}
	open, mismatch := false, false
	for _, a := range atts {
		if a.res != r {
			continue
		}
		switch a.state {
		case accepted:
			r.Outcome, r.Text, r.Attempt = Accepted, a.text, a.n
			r.Answer = Boxed(a.text)
			r.AnswerCheck = check(r.Problem, r.Answer)
			r.Follows = follows(r.Problem, a.text)
			r.Stage = "S5"
			return
		case rejected:
			mismatch = mismatch || a.mismatch
		default:
			open = true
		}
	}
	switch {
	case open:
	case mismatch:
		r.Outcome, r.Note = Mismatch, "the attempts that passed the verifier have another answer than the problem"
	case r.Note != "":
		r.Outcome = Failed
	default:
		r.Outcome = Unsolved
	}
}

// check is stage S4 for a claimed answer.
func check(p *content.Problem, ans string) string {
	switch {
	case p.Type != "answer" && p.Type != "answer+proof":
		return "n/a"
	case p.Answer == "" || ans == "":
		return "none"
	case match.Answers(p.Answer, ans) == match.AnswerMatch:
		return "match"
	}
	return "mismatch"
}

// follows is stage S5: the tag of the human solution that the text is a near copy of, or "".
func follows(p *content.Problem, text string) string {
	sh := match.Shingles(text)
	best, tag := 0.0, ""
	for _, s := range p.Solutions {
		if s.Origin == "ai" || len(s.Body) == 0 {
			continue
		}
		if t := match.Jaccard(sh, match.Shingles(s.Body)); t >= audit.NearCopy && t > best {
			best, tag = t, s.Tag
		}
	}
	return tag
}

func later(a, b string) bool { return a > b }

// Verdict reads the last VERDICT line of a bug report. It is true for PASS.
func Verdict(report string) bool {
	i := strings.LastIndex(report, "VERDICT:")
	if i < 0 {
		return false
	}
	v := strings.TrimSpace(report[i+len("VERDICT:"):])
	return strings.HasPrefix(strings.ToUpper(v), "PASS")
}

// Boxed gives the content of the last \boxed{...} of a text, or "".
func Boxed(text string) string {
	i := strings.LastIndex(text, `\boxed{`)
	if i < 0 {
		return ""
	}
	s := text[i+len(`\boxed{`):]
	depth := 1
	for j, c := range s {
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return strings.TrimSpace(s[:j])
			}
		}
	}
	return ""
}
