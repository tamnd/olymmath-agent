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

// Add reads one row of a numbered source. When the text of the row starts with a line that names the paper and the problem, the position comes from that line and the layout of the competition, because the source numbers some papers again from 1. A row with no year, no number or no topic link gives nothing, except a row whose text is a title that gives the year, as described at TitleAt. When the text is a title that names the problem, such as "Vietnam MO 1982 P5", the position comes from the title, as described at TitlePosition. When two rows give one topic two different appearances, the topic gets no appearance. When two topics claim one appearance, neither gets it, because the source numbers some papers again from 1 on each day, as the AoPS wiki does for some years of the IMO.
func (t *Topics) Add(r seed.Row) {
	m := topicRE.FindStringSubmatch(r.Ref)
	pos := seed.Position(r.Number)
	if m == nil || pos == "" {
		return
	}
	code := t.Aliases.Code(r.Labels)
	if code == "" || code == "-" {
		return
	}
	var comp *content.Competition
	if t.Corpus != nil {
		comp = t.Corpus.Competition(code)
	}
	year, letter := r.Year, ""
	if comp != nil && comp.EditionForm == "year-letter" {
		letter = Letter(r.Labels)
	}
	if year == 0 {
		y, p, l, ok := TitleAt(comp, r.Statement, pos)
		if !ok && letter == "" {
			// The number of the title can differ from the number column. TitlePosition then reads the title.
			if ys := titleYearRE.FindAllString(r.Statement, -1); len(ys) == 1 {
				y, _ = strconv.Atoi(ys[0])
				p = TitlePosition(comp, t.Aliases, y, pos, r.Statement)
				ok = p != "" && (comp == nil || comp.EditionForm != "year-letter")
			}
		}
		if !ok {
			return
		}
		year, pos, letter = y, p, l
	} else if p, _, ok := PaperPosition(comp, year, r.Statement); ok {
		pos = p
	} else if p, ok := namedPaper(comp, year, r.Statement, pos); ok {
		pos = p
	} else if p := TitlePosition(comp, t.Aliases, year, pos, r.Statement); p != "" {
		pos = p
	}
	if pos == "" {
		return
	}
	at := code + "/" + strconv.Itoa(year) + letter + "/" + pos
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

// Position gives the position of the topic of ref, such as aops-topic:1738331, when its competition and its year are code and year, and the letter of its paper, such as "a" for the AMC 10A, when the numbered source gives it. It gives "" when it does not know the position.
func (t *Topics) Position(ref, code string, year int) (pos, letter string) {
	topic, ok := strings.CutPrefix(ref, "aops-topic:")
	if !ok {
		return "", ""
	}
	at := t.at[topic]
	rest, ok := strings.CutPrefix(at, code+"/"+strconv.Itoa(year))
	if !ok || len(t.topics[at]) != 1 {
		return "", ""
	}
	letter, pos, ok = strings.Cut(rest, "/")
	if !ok || (letter != "" && letter != "a" && letter != "b") {
		return "", ""
	}
	return pos, letter
}

var (
	titleYearRE   = regexp.MustCompile(`\b(?:19|20)[0-9]{2}\b`)
	titleNumRE    = regexp.MustCompile(`(?i)(?:#|/|\bp#?|\bproblem\s*|\bpb\s*)([0-9]{1,2})\b`)
	titleLetterRE = regexp.MustCompile(`(?i)\b(?:1[02]\s?([ab])|aime\s*(ii|i))\b`)
	laterPaperRE  = regexp.MustCompile(`(?i)\b(?:round|day|paper|part|exam)\s*(?:[2-9]|ii)\b`)
	seasonRE      = regexp.MustCompile(`(?i)\b(?:fall|spring|autumn)\b`)
)

// isTitle tells if a text is a short title of one line with no formula, such as "2017 AMC 8 #4".
func isTitle(text string) bool {
	return text != "" && !strings.ContainsAny(text, "$\n") && len(strings.Fields(text)) <= 12
}

// TitleAt reads a title such as "2017 AMC 8 #4" or "Germany 2016 P3". The AoPS wiki lists the problems of a contest with such a title, a number and a topic link, but with no year. It gives the year of the title, the position and the letter of the paper for a competition with two papers each year. The title must have one year only and a problem number that agrees with the position pos of the number column. A title that names a paper or a day after the first must give the position through the layout. For a competition with two papers each year, a title that names a season, such as "2021 AMC 10A (Fall) #2", is not read, because that year had four papers. The result ok is false when the title does not give all of these.
func TitleAt(c *content.Competition, title, pos string) (year int, at, letter string, ok bool) {
	title = strings.TrimSpace(title)
	if !isTitle(title) {
		return 0, "", "", false
	}
	ys := titleYearRE.FindAllString(title, -1)
	if len(ys) == 0 {
		return 0, "", "", false
	}
	for _, y := range ys[1:] {
		if y != ys[0] {
			return 0, "", "", false
		}
	}
	year, _ = strconv.Atoi(ys[0])
	if c != nil && c.EditionForm == "year-letter" {
		if seasonRE.MatchString(title) {
			return 0, "", "", false
		}
		for _, m := range titleLetterRE.FindAllStringSubmatch(title, -1) {
			l := Letter([]string{m[1] + m[2]})
			if letter != "" && l != letter {
				return 0, "", "", false
			}
			letter = l
		}
	}
	if p, _, ok := PaperPosition(c, year, title); ok {
		return year, p, letter, p != ""
	}
	nums := titleNumRE.FindAllStringSubmatch(title, -1)
	if len(nums) == 0 || laterPaperRE.MatchString(title) {
		return 0, "", "", false
	}
	for _, n := range nums {
		if "p"+strings.TrimLeft(n[1], "0") != pos {
			return 0, "", "", false
		}
	}
	if p, ok := namedPaper(c, year, title, pos); ok {
		return year, p, letter, p != ""
	}
	return year, pos, letter, true
}

// namedPaper reads a title that names one paper of the layout of c in the year, such as "P2 Francophone MO Junior 2023" for the paper Junior. It gives the position of problem pos of that paper, or "" when the paper has no such problem. The result ok is false when the title is not a short title or does not name exactly one paper. A paper with a name such as "Day 1" or "Paper" is not named, because PaperPosition reads these.
func namedPaper(c *content.Competition, year int, title, pos string) (string, bool) {
	if c == nil || !isTitle(title) {
		return "", false
	}
	layout := c.Shape(year).Layout
	if len(layout) < 2 {
		return "", false
	}
	words := strings.Fields(strings.ToLower(title))
	var found *content.Layout
	for i := range layout {
		name := strings.ToLower(layout[i].Name)
		if name == "" || strings.ContainsAny(name, " 0123456789") || name == "paper" {
			continue
		}
		if !slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, name) }) {
			continue
		}
		if found != nil {
			return "", false
		}
		found = &layout[i]
	}
	if found == nil {
		return "", false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(pos, "p"))
	if err != nil || n < 1 || n > len(found.Positions) {
		return "", true
	}
	return found.Positions[n-1], true
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

var (
	// sectionRE finds a problem of a shortlist with sections in a title, for example "G2", "Algebra 2" or "number theory problem 4".
	sectionRE = regexp.MustCompile(`(?i)(?:^|[^a-z])(algebra|combinatorics|geometry|number theory|[acgn])\s*(?:problem\s*)?#?\s*([0-9]{1,2})\b`)
	// shortlistRE tells that a title names a shortlist.
	shortlistRE = regexp.MustCompile(`(?i)short\s*list|\bisl\b`)
	// numberRE matches a title that ends with the number of the problem, for example "Vietnam MO 1982 P5" or "Iberoamerican 2004 problem 6". The name before the number has one year and no other digit.
	numberRE = regexp.MustCompile(`(?i)^([^0-9]*?)\b((?:19|20)[0-9]{2})\b([^0-9]*?)(?:\bp|\bproblem|\bq|#)\s*\.?\s*([0-9]{1,2})\s*[.:]?$`)
	// partRE finds a word that names a part of an edition, such as a day, a grade or a level. The number after the word "P" can then count from 1 in that part.
	partRE = regexp.MustCompile(`(?i)\b(short\s*list|long\s*list|day|grade|class|level|junior|senior|test|exam|round|paper|quiz|tst|team|individual|part|and)\b|&`)
)

// TitlePosition reads the title of a problem, such as "Vietnam MO 1982 P5" or "IMO Shortlist 2011 G2", and gives the position that it names in the competition c in the year. It gives "" when the title does not name exactly one position. The AoPS wiki gives each topic the number of the problem in its day or in its section, so that number is not the position in many editions. Agent decision: the number in the title is the position, because the title names it and the wiki number does not. For a competition with sections, such as the IMO Shortlist, the title must name the shortlist and one section, and the position is the section letter and the number. For other competitions, the title replaces only a plain number pos. A title that names a day, a grade or another part does not give a position, because the parts of the edition are not known, and the number then counts from 1 in the part. A title that names another competition does not give a position.
func TitlePosition(c *content.Competition, al *seed.Aliases, year int, pos, text string) string {
	text = strings.TrimSpace(text)
	if c == nil || text == "" || strings.ContainsAny(text, "\n$") || len(text) > 80 || !strings.Contains(text, strconv.Itoa(year)) {
		return ""
	}
	if c.TopicPositions {
		ms := sectionRE.FindAllStringSubmatch(text, -1)
		if len(ms) != 1 || !shortlistRE.MatchString(text) {
			return ""
		}
		n, _ := strconv.Atoi(ms[0][2])
		if n == 0 {
			return ""
		}
		return strings.ToLower(ms[0][1][:1]) + strconv.Itoa(n)
	}
	if pos != "" && pos[0] != 'p' {
		return ""
	}
	m := numberRE.FindStringSubmatch(text)
	if m == nil || m[2] != strconv.Itoa(year) || partRE.MatchString(m[1]+" "+m[3]) {
		return ""
	}
	if name := strings.TrimSpace(m[1] + " " + m[3]); al != nil && name != "" {
		if code := al.Code([]string{name}); code != "" && code != c.Code {
			return ""
		}
	}
	n, _ := strconv.Atoi(m[4])
	if n == 0 {
		return ""
	}
	return "p" + strconv.Itoa(n)
}
