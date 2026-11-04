package audit

import (
	"slices"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
)

// NearCopy is the Jaccard similarity of the shingles of two solutions at which rule L06 calls one a near copy of the other.
const NearCopy = 0.8

// minCopyText is the shortest solution text, in bytes, that rule L06 compares. A shorter text is an excerpt or a sketch.
const minCopyText = 300

func l05(c *Context) {
	for _, p := range c.Corpus.All() {
		if p.Type != "answer" && p.Type != "answer+proof" {
			continue
		}
		for _, s := range p.Solutions {
			switch {
			case s.AnswerCheck == "mismatch":
				c.add(s.Path, "the claimed answer %q does not match the answer of the problem", s.Answer)
			case s.AnswerCheck == "" && s.Origin == "ai" && p.Answer != "" && s.Answer != "" && match.Answers(p.Answer, s.Answer) == match.AnswerMismatch:
				c.add(s.Path, "the claimed answer %q does not look like the answer %q of the problem; check it and set answer_check", s.Answer, p.Answer)
			}
		}
	}
}

func l06(c *Context) {
	for _, p := range c.Corpus.All() {
		type text struct {
			s  *content.Solution
			sh []uint64
		}
		var ts []text
		for _, s := range p.Solutions {
			if len(s.Body) >= minCopyText {
				ts = append(ts, text{s, match.Shingles(s.Body)})
			}
		}
		for i := range ts {
			for j := i + 1; j < len(ts); j++ {
				a, b := ts[i].s, ts[j].s
				if a.Follows == b.Tag || b.Follows == a.Tag {
					continue
				}
				if t := match.Jaccard(ts[i].sh, ts[j].sh); t >= NearCopy {
					c.add(b.Path, "the solution is a near copy of the solution %s (T %.3f); set follows: %s or remove it", a.Tag, t, a.Tag)
				}
			}
		}
	}
}

func g01(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Solutions {
			for _, j := range s.All() {
				g01Judgement(c, s, j)
			}
			g := s.Grade
			if g == nil {
				continue
			}
			top, ok := content.Scales[g.Scale]
			switch {
			case !ok:
				c.add(s.Path, "the grade has the scale %q, not imo-7", g.Scale)
				continue
			case !slices.Contains(content.GradeRules, g.Rule):
				c.add(s.Path, "the grade has the rule %q, not one of %s", g.Rule, strings.Join(content.GradeRules, ", "))
			case len(g.Graders) == 0:
				c.add(s.Path, "the grade has no graders")
			case g.Score < 0 || g.Score > top:
				c.add(s.Path, "the grade has the score %d, not from 0 to %d", g.Score, top)
			case g.Bucket != content.Bucket(g.Score, top):
				c.add(s.Path, "the grade has the bucket %q, but the score %d is %s", g.Bucket, g.Score, content.Bucket(g.Score, top))
			}
			if g.Rule == "min-of-jury-v1" && len(g.Graders) > 0 && !g.Queue {
				low := g.Graders[0].Score
				for _, j := range g.Graders {
					low = min(low, j.Score)
				}
				if g.Score != low {
					c.add(s.Path, "the grade has the score %d, but the minimum of the graders is %d", g.Score, low)
				}
			}
		}
	}
}

func g01Judgement(c *Context, s *content.Solution, j content.Judgement) {
	switch {
	case j.By == "":
		c.add(s.Path, "a grade does not name its grader")
	case !content.IsModel(j.By) && !strings.HasPrefix(j.By, "human:"):
		c.add(s.Path, "the grader %q is not model:<id> or human:reviewer/<id>", j.By)
	}
	if _, err := time.Parse(time.DateOnly, j.At); err != nil {
		c.add(s.Path, "the grade by %s has no date of the form 2026-12-01", j.By)
	}
	top := j.Max
	if top == 0 {
		top = 7
	}
	if j.Score < 0 || j.Score > top {
		c.add(s.Path, "the grade by %s has the score %d, not from 0 to %d", j.By, j.Score, top)
	}
	if content.IsModel(j.By) && s.Run != nil && "model:"+s.Run.Model == j.By {
		c.add(s.Path, "the model %s graded its own solution", s.Run.Model)
	}
	if j.Rubric == "" {
		c.add(s.Path, "the grade by %s does not name its rubric", j.By)
		return
	}
	loc, v, err := content.ParseRubricRef(j.Rubric)
	if err != nil {
		c.add(s.Path, "%v", err)
		return
	}
	r := c.Corpus.Rubrics[loc]
	switch {
	case r == nil:
		c.add(s.Path, "the rubric %s has no file rubrics/%s.md", j.Rubric, loc)
	case v > r.Version:
		c.add(s.Path, "the grade uses the version %d of the rubric %s, which is at version %d", v, loc, r.Version)
	case r.Problem != s.Problem:
		c.add(s.Path, "the rubric %s is for another problem", j.Rubric)
	}
}

func g02(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Solutions {
			for _, j := range s.All() {
				if !strings.HasPrefix(j.By, "human:") {
					continue
				}
				id := content.HumanGrader(j.By)
				person := c.Corpus.People[id]
				switch {
				case id == "":
					c.add(s.Path, "the human grader %q is not of the form human:reviewer/<id>", j.By)
				case person == nil:
					c.add(s.Path, "the grader %s is not in registers/people.yaml", id)
				case !person.Has("reviewer"):
					c.add(s.Path, "the grader %s does not have the role reviewer in registers/people.yaml", id)
				}
			}
		}
	}
}

func g03(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Solutions {
			if s.HumanGraded() {
				continue
			}
			lo, hi, n := 8, -1, 0
			for _, j := range s.All() {
				if !content.IsModel(j.By) {
					continue
				}
				top := j.Max
				if top == 0 {
					top = 7
				}
				v := content.Seven(j.Score, top)
				lo, hi, n = min(lo, v), max(hi, v), n+1
			}
			if n >= 2 && hi-lo > 2 {
				c.add(s.Path, "the model grades differ by %d points out of 7 and no person graded the solution; send it to the human queue", hi-lo)
			}
		}
	}
}
