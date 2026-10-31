package canon

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/seed"
)

// topicRE finds the topic in the URL of an AoPS thread, for example c6h1738331 in https://artofproblemsolving.com/community/c6h1738331p11293588.
var topicRE = regexp.MustCompile(`artofproblemsolving\.com/community/c\d+h(\d+)`)

// Topics maps AoPS topics to the appearances that a numbered source gives them. The AoPS wiki gives the number of each problem with a link to its topic, but it often has only a title for the text. The AoPS dataset has the full text of the topic but no number. Topics joins the two.
type Topics struct {
	Aliases *seed.Aliases
	at      map[string]string
	// topics[at] is the set of topics that claim the appearance.
	topics map[string]map[string]bool
}

// Add reads one row of a numbered source. A row with no year, no number or no topic link gives nothing. When two rows give one topic two different appearances, the topic gets no appearance. When two topics claim one appearance, neither gets it, because the source numbers some papers again from 1 on each day, as the AoPS wiki does for some years of the IMO.
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
