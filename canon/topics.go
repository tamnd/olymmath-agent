package canon

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
)

// topicRE finds the topic in the URL of an AoPS thread, for example c6h1738331 in https://artofproblemsolving.com/community/c6h1738331p11293588.
var topicRE = regexp.MustCompile(`artofproblemsolving\.com/community/c\d+h(\d+)`)

// Topics maps AoPS topics to the appearances that a numbered source gives them. The AoPS wiki gives the number of each problem with a link to its topic, but it often has only a title for the text. The AoPS dataset has the full text of the topic but no number. Topics joins the two.
type Topics struct {
	Aliases *seed.Aliases
	// Corpus gives the layouts of the competitions. A title such as "2006 IrMO Paper 2 P1" gives a position in the layout. It can be nil.
	Corpus *content.Corpus
	at     map[string]string
	// topics[at] is the set of topics that claim the appearance.
	topics map[string]map[string]bool
}

// Add reads one row of a numbered source. When the text of the row starts with a line that names the paper and the problem, the position comes from that line and the layout of the competition, because the source numbers some papers again from 1. A row with no year, no number or no topic link gives nothing. When two rows give one topic two different appearances, the topic gets no appearance. When two topics claim one appearance, neither gets it, because the source numbers some papers again from 1 on each day, as the AoPS wiki does for some years of the IMO.
func (t *Topics) Add(r seed.Row) {
	m := topicRE.FindStringSubmatch(r.Ref)
	pos := seed.Position(r.Number)
	if m == nil || r.Year == 0 || pos == "" {
		return
	}
	code := t.Aliases.Code(r.Labels)
	if code == "" || code == "-" {
		return
	}
	if t.Corpus != nil {
		if p, _, ok := PaperPosition(t.Corpus.Competition(code), r.Year, r.Statement); ok {
			pos = p
		}
	}
	if pos == "" {
		return
	}
	at := code + "/" + strconv.Itoa(r.Year) + "/" + pos
	if t.at == nil {
		t.at, t.topics = map[string]string{}, map[string]map[string]bool{}
	}
	if t.topics[at] == nil {
		t.topics[at] = map[string]bool{}
	}
	t.topics[at][m[1]] = true
	if old, ok := t.at[m[1]]; ok && old != at {
		at = ""
	}
	t.at[m[1]] = at
}

// Position gives the position of the topic of ref, such as aops-topic:1738331, when its competition and its year are code and year. It gives "" when it does not know the position.
func (t *Topics) Position(ref, code string, year int) string {
	topic, ok := strings.CutPrefix(ref, "aops-topic:")
	if !ok {
		return ""
	}
	at := t.at[topic]
	prefix := code + "/" + strconv.Itoa(year) + "/"
	if !strings.HasPrefix(at, prefix) || len(t.topics[at]) != 1 {
		return ""
	}
	return at[len(prefix):]
}

// paperRE matches a line that names the paper or the day and the problem, for example "2006 IrMO Paper 2 P1" or "Day 1, Problem 3".
var paperRE = regexp.MustCompile(`(?i)^[^\n]{0,40}?\b(?:paper|day)\s*([1-9])[\s,.:-]*(?:p|problem)\s*([0-9]{1,2})[.:)]?[ ]*(?:\n|$)`)

// PaperPosition reads a first line of text that names the paper and the problem, such as "2006 IrMO Paper 2 P1", and gives the position of that problem in the layout of the competition c in the year, and the text after the line. The problem number counts from 1 in its paper, or it is a position of the paper, as in "Day 2 Problem 4". The result ok is false when the text has no such line or c has no layout. The position is "" when the layout has no such problem.
func PaperPosition(c *content.Competition, year int, text string) (pos, rest string, ok bool) {
	m := paperRE.FindStringSubmatchIndex(text)
	if m == nil || c == nil {
		return "", text, false
	}
	layout := c.Shape(year).Layout
	if len(layout) == 0 {
		return "", text, false
	}
	paper, _ := strconv.Atoi(text[m[2]:m[3]])
	n, _ := strconv.Atoi(text[m[4]:m[5]])
	rest = strings.TrimSpace(text[m[1]:])
	if paper > len(layout) || n == 0 {
		return "", rest, true
	}
	ps := layout[paper-1].Positions
	switch p := "p" + strconv.Itoa(n); {
	case n <= len(ps):
		return ps[n-1], rest, true
	case slices.Contains(ps, p):
		return p, rest, true
	}
	return "", rest, true
}
