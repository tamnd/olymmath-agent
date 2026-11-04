package site

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// aiView is one AI solution in the AI section of a problem page (spec 07.5).
type aiView struct {
	solutionView
	// Grade is the score and the bucket, for example "6 of 7, almost", or "" when no grade is there.
	Grade string
	// Graders names each grader with its score. A model grader is an agent grade.
	Graders []graderView
	// Verified is true when a person graded the solution. Otherwise the grade is a model grade.
	Verified bool
	// Rubric is the link to the rubric file of the grade on GitHub, or "".
	Rubric string
	// Contaminated is true when the problem is older than the training cutoff of the model, so the model may have seen it.
	Contaminated bool
	// Followers counts the AI solutions that closely follow this one and are not shown.
	Followers int
}

// graderView is one grader of an AI solution.
type graderView struct {
	Name  string
	Agent bool
	Score int
	Max   int
}

// aiSection splits the AI solutions from the others. A solution that follows another one (stage S5) is not shown, and the solution that it follows counts it.
func (b *builder) aiSection(p *content.Problem, all []solutionView) (others []solutionView, ai []aiView) {
	follows := map[string]int{}
	for _, s := range all {
		if s.Origin == "ai" && s.Follows != "" {
			follows[s.Follows]++
		}
	}
	for _, s := range all {
		switch {
		case s.Origin != "ai":
			others = append(others, s)
		case s.Follows == "":
			ai = append(ai, b.aiView(p, s, follows[s.Tag]))
		}
	}
	return others, ai
}

func (b *builder) aiView(p *content.Problem, s solutionView, followers int) aiView {
	v := aiView{solutionView: s, Verified: s.HumanGraded(), Followers: followers}
	if s.Run != nil {
		v.Contaminated = olderThan(p, s.Run.Cutoff)
	}
	if g := s.Grade; g != nil {
		top := content.Scales[g.Scale]
		v.Grade = fmt.Sprintf("%d of %d, %s", g.Score, top, g.Bucket)
		if loc, _, err := content.ParseRubricRef(g.Rubric); err == nil {
			if r := b.corpus.Rubrics[loc]; r != nil {
				v.Rubric = b.opts.Repo + "/blob/main/" + r.Path
			}
		}
	}
	for _, j := range s.All() {
		gv := graderView{Name: j.By, Score: j.Score, Max: j.Max}
		if gv.Max == 0 {
			gv.Max = 7
		}
		if m, ok := strings.CutPrefix(j.By, "model:"); ok {
			gv.Name, gv.Agent = m, true
		} else if id := content.HumanGrader(j.By); id != "" {
			gv.Name = id
			if per := b.corpus.People[id]; per != nil && per.Name != "" {
				gv.Name = per.Name
			}
		}
		v.Graders = append(v.Graders, gv)
	}
	return v
}

// olderThan tells if the problem was set before a training cutoff such as 2026-01. A problem of the year of the cutoff counts as older unless the dates of its edition come after the cutoff.
func olderThan(p *content.Problem, cutoff string) bool {
	var year, month int
	if p.Year == 0 || cutoff == "" {
		return false
	}
	if _, err := fmt.Sscanf(cutoff, "%d-%d", &year, &month); err != nil {
		return false
	}
	switch {
	case p.Year < year:
		return true
	case p.Year > year:
		return false
	}
	if p.Edition != nil && len(p.Edition.Dates) >= 7 {
		return p.Edition.Dates[:7] < cutoff
	}
	return true
}

// aiDashboard is the data of the AI page: the AI solutions, their grades and the runs that made them.
type aiDashboard struct {
	Solutions, Shown, Followers, Graded, Verified, Queue int
	Match, Mismatch                                      int
	Buckets                                              []bucketCount
	Models                                               []modelRow
	Runs                                                 []*content.RunRecord
	Cost                                                 float64
	// Waiting are the AI solutions whose graders still differ, so a person must grade them.
	Waiting []alsoView
}

type bucketCount struct {
	Name  string
	Count int
}

// modelRow counts the AI solutions of one model.
type modelRow struct {
	Model                                string
	Solutions, Graded, Correct, Verified int
	Cost                                 float64
}

// ai writes the AI page (spec 07.5): how many AI solutions olymmath has, how the jury graded them, and the cost of each run.
func (b *builder) ai() error {
	v := aiDashboard{}
	buckets := map[string]int{}
	models := map[string]*modelRow{}
	for _, p := range b.corpus.All() {
		for _, s := range p.Solutions {
			if s.Origin != "ai" {
				continue
			}
			v.Solutions++
			if s.Follows != "" {
				v.Followers++
			} else {
				v.Shown++
			}
			switch s.AnswerCheck {
			case "match":
				v.Match++
			case "mismatch":
				v.Mismatch++
			}
			name := "unknown"
			if s.Run != nil && s.Run.Model != "" {
				name = s.Run.Model
			}
			m := models[name]
			if m == nil {
				m = &modelRow{Model: name}
				models[name] = m
			}
			m.Solutions++
			if s.Run != nil {
				m.Cost += s.Run.Cost
			}
			if s.HumanGraded() {
				v.Verified++
				m.Verified++
			}
			if g := s.Grade; g != nil {
				v.Graded++
				m.Graded++
				buckets[g.Bucket]++
				if g.Bucket == "correct" {
					m.Correct++
				}
				if g.Queue {
					v.Queue++
					v.Waiting = append(v.Waiting, alsoView{Title: problemTitle(p), Link: b.link(problemPath(p))})
				}
			}
		}
	}
	for _, name := range content.Buckets {
		v.Buckets = append(v.Buckets, bucketCount{Name: name, Count: buckets[name]})
	}
	for _, m := range models {
		v.Models = append(v.Models, *m)
	}
	sort.Slice(v.Models, func(i, j int) bool { return v.Models[i].Model < v.Models[j].Model })
	for _, r := range b.corpus.Runs {
		v.Runs = append(v.Runs, r)
		v.Cost += r.Cost
	}
	slices.SortFunc(v.Runs, func(a, c *content.RunRecord) int { return strings.Compare(c.ID, a.ID) })
	return b.render("ai.html", page{
		Title:       "AI solutions",
		Description: "The AI solutions on olymmath, how a jury of models graded them, and the cost of each run.",
		Path:        "/ai/",
		Section:     "ai",
		Data:        v,
	})
}
