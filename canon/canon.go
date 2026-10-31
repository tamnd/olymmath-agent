// Package canon makes problem records from candidates (spec 06.1 to 06.3). It groups the candidates of each appearance, chooses the canonical statement by the human-first rule, sends doubtful candidates to the review queue, and merges the appearances that the known duplicate patterns and the matcher say are one problem. The functions are pure: the same candidates give the same plan.
package canon

import (
	"cmp"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/normalize"
)

// Version names the rules of this package and of the packages it uses.
const Version = "canon/1+" + normalize.Version + "+" + match.Version

// Candidate is one text of a problem at one appearance, from one source.
type Candidate struct {
	// At is the locator of the appearance, for example aime/2021a/p3.
	At     string
	Source string
	// Origin and Zone come from the source register.
	Origin string
	Zone   string
	Lang   string
	// Name names the text at the source, for example "AIME 2021 I problem 3".
	Name string
	URL  string
	// Ref identifies the text inside the source, for example the file and the row of a dataset.
	Ref    string
	Text   string
	Answer string
	// Key is the strength of the locator of the candidate: strong when it comes from H1 or H2 metadata, weak when it comes from the free text of a dataset.
	Key match.Key

	sh []uint64
}

// Shingles returns the shingles of the text. It computes them once.
func (c *Candidate) Shingles() []uint64 {
	if c.sh == nil {
		c.sh = match.Shingles(c.Text)
	}
	return c.sh
}

// Rank gives the precedence of a candidate by the rule of spec 06.3, and the name of the rule. A lower rank wins. A model-written candidate gives -1: it never becomes a statement.
func Rank(c *Candidate, original string) (int, string) {
	switch c.Origin {
	case "H1":
		if original == "" || c.Lang == original {
			return 0, "official-original"
		}
		return 1, "official-translation"
	case "H2":
		return 2, "curated"
	case "H3":
		return 3, "community"
	case "D":
		return 4, "derived"
	}
	return -1, ""
}

// Curated is the order of the curated sources inside the rule "curated" (spec 06.3). A source that is not in the list goes after the listed sources.
var Curated = []string{"imo-compendium", "usa-tst-archive", "problems-ru", "kedlaya-putnam", "evanchen", "aops-wiki", "aops-collections"}

func priority(src string) int {
	if i := slices.Index(Curated, src); i >= 0 {
		return i
	}
	return len(Curated)
}

// Choice is the result of Choose for one appearance.
type Choice struct {
	At string
	// Best is the canonical candidate, and Rule is the rule that chose it.
	Best *Candidate
	Rule string
	// Kept are the other candidates that agree with Best, one for each source.
	Kept []*Candidate
	// Doubts are the candidates that locate to the appearance but whose text does not agree with Best. They go to the review queue.
	Doubts []Doubt
	// Answer is the answer that all candidates with an answer agree on, or "".
	Answer string
	// Disputed is true when two candidates give different answers.
	Disputed bool
}

// Doubt is a candidate that the review queue must decide on.
type Doubt struct {
	C *Candidate
	T float64
	A match.Answer
}

// Choose chooses the canonical statement of one appearance. original is the original language of the competition. Inside one rank, a source with a higher priority wins, and then the candidate whose text agrees most with the other candidates. Choose returns false when no candidate can be canonical.
func Choose(at string, cands []*Candidate, original string) (Choice, bool) {
	ch := Choice{At: at}
	var ok []*Candidate
	for _, c := range cands {
		if r, _ := Rank(c, original); r >= 0 && c.Text != "" {
			ok = append(ok, c)
		}
	}
	if len(ok) == 0 {
		return ch, false
	}
	agree := map[*Candidate]float64{}
	for _, a := range ok {
		for _, b := range ok {
			if a != b {
				agree[a] += match.Jaccard(a.Shingles(), b.Shingles())
			}
		}
	}
	slices.SortStableFunc(ok, func(a, b *Candidate) int {
		ra, _ := Rank(a, original)
		rb, _ := Rank(b, original)
		return cmp.Or(
			cmp.Compare(ra, rb),
			cmp.Compare(priority(a.Source), priority(b.Source)),
			cmp.Compare(agree[b], agree[a]),
			strings.Compare(a.Source, b.Source),
			strings.Compare(a.Ref, b.Ref),
		)
	})
	ch.Best = ok[0]
	_, ch.Rule = Rank(ch.Best, original)
	seen := map[string]bool{ch.Best.Source: true}
	answers := []string{}
	if ch.Best.Answer != "" {
		answers = append(answers, ch.Best.Answer)
	}
	for _, c := range ok[1:] {
		t := match.Jaccard(ch.Best.Shingles(), c.Shingles())
		a := match.Answers(ch.Best.Answer, c.Answer)
		if match.Decide(match.Signals{K: c.Key, T: t, A: a}) == match.Review {
			ch.Doubts = append(ch.Doubts, Doubt{C: c, T: t, A: a})
			if a == match.AnswerMismatch && t >= match.Same {
				ch.Disputed = true
			}
			continue
		}
		if c.Answer != "" {
			answers = append(answers, c.Answer)
		}
		if !seen[c.Source] {
			seen[c.Source] = true
			ch.Kept = append(ch.Kept, c)
		}
	}
	if len(answers) > 0 {
		ch.Answer = answers[0]
		for _, a := range answers[1:] {
			if match.Answers(ch.Answer, a) == match.AnswerMismatch {
				ch.Answer = ""
				ch.Disputed = true
				break
			}
		}
	}
	if ch.Disputed {
		ch.Answer = ""
	}
	return ch, true
}

// Status gives the status of the statement of a choice (spec 09.1).
func (ch Choice) Status() string {
	switch {
	case ch.Disputed:
		return "disputed"
	case ch.Best.Origin == "H1" || ch.Best.Origin == "H2":
		return "sourced"
	}
	return "captured"
}
