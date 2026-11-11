package canon

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

var (
	// leadRE matches a problem mark at the start of a text, such as "Problem 4.", "**Problem 3.**", "P6.", "7." or "2)". The mark must end with a point, a colon or a parenthesis and a space, so that "14 students" and "9.2" do not match.
	leadRE = regexp.MustCompile(`(?i)^\s*(?:\*\*|__)?\s*(?:(?:problem|task|question)\s*-?\s*|[pq]\s*\.?\s*)?([1-9][0-9]?)\s*(?:\.\)|[.:)])\s*(?:\*\*|__)?(?:\s|$)`)
	// moreRE finds a second problem mark in the text after the lead mark, at the start of a line or in bold, as in a row that holds a whole paper ("**p1.** ... **p2.** ...") or a problem with numbered parts.
	moreRE = regexp.MustCompile(`(?i)(?:^|\n)\s*(?:\*\*|__)?\s*(?:problem\s*|[pq]\s*)?[0-9]{1,2}\s*(?:\.\)|[.:)])(?:\s|\*\*|$)|\*\*\s*(?:problem\s*|[pq]\s*)?[0-9]{1,2}\s*[.:)]?\s*\*\*`)
)

// LeadNumber reads the problem mark at the start of a text, such as "Problem 4." or "7.", and gives the number and the text after the mark. Some rows of the AoPS dataset start with the number of the problem in its paper. The result ok is false when the text has no such mark, or when the text after it has a second mark, because the row then holds more than one problem or a problem with numbered parts.
func LeadNumber(text string) (n int, rest string, ok bool) {
	m := leadRE.FindStringSubmatchIndex(text)
	if m == nil {
		return 0, text, false
	}
	n, _ = strconv.Atoi(text[m[2]:m[3]])
	rest = strings.TrimSpace(text[m[1]:])
	if rest == "" || moreRE.MatchString(rest) {
		return 0, text, false
	}
	return n, rest, true
}

// leadPosition gives the position of problem n of a year whose positions are p1 to pK in this order, and the number K. It gives "" when the year has other positions, such as the positions of a shortlist or of a paper for each grade, or when n is above K.
func leadPosition(c *content.Competition, year, n int) (pos string, k int) {
	ps := c.Shape(year).Positions()
	for i, p := range ps {
		if p != "p"+strconv.Itoa(i+1) {
			return "", 0
		}
	}
	if n < 1 || n > len(ps) {
		return "", len(ps)
	}
	return ps[n-1], len(ps)
}

// onePaper tells if the register gives the year one paper. A year of an override has the shape of one paper, but the register does not say how many papers it had.
func onePaper(c *content.Competition, year int) bool {
	if _, ok := c.Overrides[year]; ok {
		return false
	}
	return len(c.Shape(year).Layout) == 1
}

// leadRow is a row that got its position from the problem mark at the start of its text.
type leadRow struct {
	// collection is the source, the competition, the year and the labels of the row: the rows of one AoPS contest page.
	collection string
	n          int
}

// leadSure tells if the numbers of the rows of one collection give their positions. Agent decision: a number alone gives the position only when the source numbers the whole edition from 1. The numbers must be different, the collection must have no more rows than the year has positions, and the year must have one paper, or a row of the collection must have the number of the last position. A source that numbers each paper or each day from 1 never has that number, so its rows get no position.
func leadSure(numbers []int, rows, k int, one bool) bool {
	if len(numbers) == 0 || rows > k {
		return false
	}
	seen := map[int]bool{}
	for _, n := range numbers {
		if seen[n] {
			return false
		}
		seen[n] = true
	}
	return one || slices.Contains(numbers, k)
}

// Claimed tells if a topic other than the topic of ref claims the appearance at, such as imo/2004/p1. The reference of a row of the AoPS dataset is aops-topic: and the topic.
func (t *Topics) Claimed(at, ref string) bool {
	if t == nil {
		return false
	}
	t.resolve()
	topic := strings.TrimPrefix(ref, "aops-topic:")
	for o := range t.topics[at] {
		if o != topic {
			return true
		}
	}
	return false
}

// Elsewhere tells if the numbered source gives the topic of ref another appearance than at, or claims it for two appearances.
func (t *Topics) Elsewhere(at, ref string) bool {
	if t == nil {
		return false
	}
	t.resolve()
	topic, ok := strings.CutPrefix(ref, "aops-topic:")
	if !ok {
		return false
	}
	old, ok := t.at[topic]
	return ok && old != at
}
