package grade

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/model"
)

// DraftPipeline is the version of the rubric drafting prompt.
const DraftPipeline = "rubric/1"

const draftSystem = `You write the grading rubric of an olympiad problem on the IMO scale from 0 to 7. Graders use it to score proposed solutions.

Rules:
- Solve the problem first, for yourself. Then name the key steps that a complete proof needs.
- Give 2 to 5 checkpoints. Each checkpoint is one step that a grader can see in a solution, with positive points. The points of the checkpoints add up to exactly 7.
- When other correct approaches exist, write the checkpoints so that they give credit to any correct approach, or name the other approach in the notes.
- Give the items that get no credit, for example a correct answer with no proof, or a check of small cases.
- Give the usual deductions, with negative points.

Write the rubric in exactly this form, one item on each line, and nothing before it:
CHECKPOINT <points>: <text>
ZERO: <text>
DEDUCT <points>: <text>
NOTES:
<short notes for the graders: the answer, the main idea, and the common errors>`

// Draft is a problem that needs a rubric.
type Draft struct {
	// ID names the draft, for example the tag of the problem.
	ID        string
	Statement string
	// Answer is the known answer of the problem, when it has one.
	Answer string
	// References are solutions of the problem that the drafter may read.
	References []string
}

// DraftRequest gives the call that drafts the rubric of d with the model mdl.
func DraftRequest(d Draft, mdl, effort string) model.Request {
	var b strings.Builder
	b.WriteString("Problem:\n\n" + strings.TrimSpace(d.Statement) + "\n")
	if a := strings.TrimSpace(d.Answer); a != "" {
		b.WriteString("\nKnown answer: " + a + "\n")
	}
	for i, r := range d.References {
		fmt.Fprintf(&b, "\nReference solution %d:\n\n%s\n", i+1, strings.TrimSpace(r))
	}
	b.WriteString("\nWrite the rubric.")
	return model.Request{ID: d.ID, Kind: "rubric", Model: mdl, Effort: effort, System: draftSystem, MaxTokens: 16000,
		Messages: []model.Message{{Role: "user", Text: b.String()}}}
}

var (
	checkpointRE = regexp.MustCompile(`^(?:[-*]\s*)?\**CHECKPOINT\**\s*\(?\+?(\d+)\)?\s*(?:points?|pts?)?\s*\**[:.]\**\s*(.+)$`)
	deductRE     = regexp.MustCompile(`^(?:[-*]\s*)?\**DEDUCT(?:ION)?\**\s*\(?(-\d+)\)?\s*(?:points?|pts?)?\s*\**[:.]\**\s*(.+)$`)
	zeroRE       = regexp.MustCompile(`^(?:[-*]\s*)?\**ZERO\**\s*\**[:.]\**\s*(.+)$`)
	notesRE      = regexp.MustCompile(`^\**NOTES\**\s*\**:\**\s*(.*)$`)
)

// ParseDraft reads the rubric that a model wrote. It gives an error when the form is wrong or the checkpoints do not give 7 points.
func ParseDraft(text string) (*content.Rubric, error) {
	r := &content.Rubric{Version: 1, Scale: "imo-7", Source: "drafted", Status: "draft"}
	var notes []string
	inNotes := false
	for line := range strings.SplitSeq(text, "\n") {
		t := strings.TrimSpace(line)
		if inNotes {
			notes = append(notes, strings.TrimRight(line, " \t"))
			continue
		}
		if m := checkpointRE.FindStringSubmatch(t); m != nil {
			p, _ := strconv.Atoi(m[1])
			r.Checkpoints = append(r.Checkpoints, content.Checkpoint{Points: p, Text: clean(m[2])})
		} else if m := deductRE.FindStringSubmatch(t); m != nil {
			p, _ := strconv.Atoi(m[1])
			r.Deductions = append(r.Deductions, content.Checkpoint{Points: p, Text: clean(m[2])})
		} else if m := zeroRE.FindStringSubmatch(t); m != nil {
			r.ZeroCredit = append(r.ZeroCredit, clean(m[1]))
		} else if m := notesRE.FindStringSubmatch(t); m != nil {
			inNotes = true
			if m[1] != "" {
				notes = append(notes, m[1])
			}
		}
	}
	r.Notes = strings.TrimSpace(strings.Join(notes, "\n"))
	if len(r.Checkpoints) == 0 {
		return nil, errors.New("the draft has no checkpoints")
	}
	sum := 0
	for _, c := range r.Checkpoints {
		if c.Points <= 0 {
			return nil, errors.New("a checkpoint of the draft has no points")
		}
		sum += c.Points
	}
	if sum != 7 {
		return nil, fmt.Errorf("the checkpoints of the draft give %d points, want 7", sum)
	}
	return r, nil
}

// clean takes the Markdown bold marks off the ends of an item.
func clean(s string) string {
	return strings.TrimSpace(strings.Trim(strings.TrimSpace(s), "*"))
}
