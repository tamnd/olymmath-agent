package official

import (
	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
)

// Link is a problem whose canonical text agrees with a segment of a paper.
type Link struct {
	Problem *content.Problem
	Segment Segment
	T       float64
}

// Result is the outcome of the anchor of one paper.
type Result struct {
	Links []Link
	// Items are the review items of kind official: a segment that does not agree with the canonical text of its problem.
	Items []canon.Item
	// Have counts the problems that already link to the source.
	Have int
	// Missing are the appearances of the paper that have no problem in the content.
	Missing []string
}

// Owners maps each appearance of the corpus to its problem.
func Owners(c *content.Corpus) map[string]*content.Problem {
	out := map[string]*content.Problem{}
	for _, p := range c.All() {
		for _, a := range p.Appearances {
			out[a.At] = p
		}
	}
	return out
}

// Anchor compares the segments of a paper with the problems of its edition. It does not change a problem.
func Anchor(owners map[string]*content.Problem, source string, p Paper, segs []Segment) Result {
	var r Result
	for _, sg := range segs {
		at := p.Edition + "/" + sg.Position
		pr := owners[at]
		if pr == nil {
			r.Missing = append(r.Missing, at)
			continue
		}
		if hasSource(pr, source) {
			r.Have++
			continue
		}
		text := ""
		if s := pr.CanonicalStatement(); s != nil {
			text = s.Text
		}
		t := Signal(sg.Text, text)
		if t >= Agree {
			r.Links = append(r.Links, Link{Problem: pr, Segment: sg, T: t})
			continue
		}
		r.Items = append(r.Items, canon.Item{ID: canon.ItemID("official", at, source), Kind: "official", At: at, Source: source, Ref: p.URL, T: round(t), Text: text, Other: sg.Text})
	}
	return r
}

func hasSource(p *content.Problem, source string) bool {
	for _, s := range p.Statements {
		if s.Source == source {
			return true
		}
	}
	return false
}

func round(f float64) float64 { return float64(int(f*1000+0.5)) / 1000 }
