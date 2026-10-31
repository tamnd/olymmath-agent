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

// Traced is a problem with a transcription that agrees with the text of a paper (see Trace).
type Traced struct {
	Problem *content.Problem
	// Statement is the id of the transcription.
	Statement string
}

// Result is the outcome of the anchor of one paper.
type Result struct {
	Links []Link
	// Items are the review items of kind official: a segment that does not agree with the canonical text of its problem.
	Items []canon.Item
	// Have counts the problems that already link to the source.
	Have int
	// Traced are the problems whose canonical class is below H2 and that have a transcription that agrees with its segment.
	Traced []Traced
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
		linked := hasSource(pr, source)
		if id := transcription(pr, p.Lang, sg.Text); id != "" {
			r.Traced = append(r.Traced, Traced{Problem: pr, Statement: id})
		}
		if linked {
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

// transcription gives the id of the statement of p that is a transcription of the text of a paper in the language lang, with the best order signal. It gives "" when the canonical statement has the class H1 or H2 already, or when no statement agrees.
func transcription(p *content.Problem, lang, paper string) string {
	if s := p.CanonicalStatement(); s != nil {
		if c := p.Class(s); c == "H1" || c == "H2" {
			return ""
		}
	}
	id, best := "", 0.0
	for _, s := range p.Statements {
		if s.Text == "" || s.Lang != lang || s.Kind == "official" || (s.Origin != "D" && s.Origin != "H3") {
			continue
		}
		if t := Order(paper, s.Text); t > best && Trace(paper, s.Text) {
			id, best = s.ID, t
		}
	}
	return id
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
