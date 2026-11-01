package canon

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"

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
	year, letter, unsure := r.Year, "", false
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
	} else if n := titleNumber(r.Statement); n != "" && strings.HasPrefix(pos, "p") && n != pos {
		// The title names a part, such as "China TST 1986 problem 5", so TitlePosition does not read it. Its number does not agree with the number of the row, so the topic gets no appearance. It still claims the appearance of its number, because the true position is not known.
		unsure = true
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
	if old, ok := t.at[m[1]]; unsure || (ok && old != at) {
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

// namedPaper reads a title that names one paper of the layout of c in the year, such as "P2 Francophone MO Junior 2023" for the paper Junior or "Middle European MO 2013 T 5" for the position t5. It gives the position of problem pos of that paper, or "" when the paper has no such problem. The result ok is false when the title is not a short title, or when it names no position and not exactly one paper. A paper with a name such as "Day 1" or "Paper" is not named, because PaperPosition reads these. A word of the title can end at a punctuation mark, as in "(Advanced)" or "2016,Advanced". When the title ends with a problem number that is not pos, the number counts in the paper or across the papers, and the title gives no position when the two numbers do not agree. For a paper of several days, a title that names a day counts the number from 1 in that day, as described at dayPosition. A number above the size of the paper counts across the papers in the order of the layout: problem 10 of a title that names the paper Team after a paper of four problems is problem 6 of Team.
func namedPaper(c *content.Competition, year int, title, pos string) (string, bool) {
	if c == nil || !isTitle(title) {
		return "", false
	}
	layout := c.Shape(year).Layout
	if len(layout) < 2 {
		return "", false
	}
	if p := layoutToken(layout, title); p != "" {
		return p, true
	}
	i := namedIndex(layout, title)
	if i < 0 {
		return "", false
	}
	n, err := strconv.Atoi(strings.TrimPrefix(pos, "p"))
	if err != nil || n < 1 {
		return "", true
	}
	offset := 0
	for _, l := range layout[:i] {
		offset += len(l.Positions)
	}
	// Agent decision: in a paper of several days, the number of the title counts in the day or in the paper, and pos counts in the day, in the paper or across the days, as in "Tuymaada 2014 Day 2 P1 Junior League" with pos p5. The number of the title is used when one of these counts explains pos.
	if d := titleDay(title); d > 0 && layout[i].Days > 1 {
		t := titleMark(title)
		if t == 0 {
			return dayPosition(layout[i], d, n), true
		}
		before := (d - 1) * (len(layout[i].Positions) / layout[i].Days)
		if t != n && t+before != n && t-before != n {
			return "", true
		}
		return dayPosition(layout[i], d, t), true
	}
	// Agent decision: the number of the title and pos can count in two ways, from 1 in the paper or across the papers. When the offset of the paper explains the difference, as in "MEMO 2008 Team P5" for problem 1 of Team, or in "MEMO 2009 problem 1 team competition" with pos p5, the number in the paper is used. Else the true number is not known.
	if t := titleMark(title); t > 0 && t != n {
		switch {
		case t+offset == n:
			n = t
		case t-offset != n:
			return "", true
		}
	}
	ps := layout[i].Positions
	if n <= len(ps) {
		return ps[n-1], true
	}
	if n > offset && n <= offset+len(ps) {
		return ps[n-offset-1], true
	}
	return "", true
}

// namedIndex gives the index of the one paper of the layout that the text names, -1 when it names none and -2 when it names more than one. A paper with a name that has a space or a digit, or the name "Paper", is never named.
func namedIndex(layout []content.Layout, text string) int {
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) })
	found := -1
	for i := range layout {
		name := strings.ToLower(layout[i].Name)
		if name == "" || strings.ContainsAny(name, " 0123456789") || name == "paper" {
			continue
		}
		if !slices.ContainsFunc(words, func(w string) bool { return strings.HasPrefix(w, name) }) {
			continue
		}
		if found >= 0 {
			return -2
		}
		found = i
	}
	return found
}

// tokenRE finds a position with a letter in a title, such as "T 5" in "Middle European MO 2013 T 5", "J1" in "Tuymaada 2021 / J1", "I-3" or "PI 3".
var tokenRE = regexp.MustCompile(`(?i)(?:^|[^a-z0-9])p?([a-z])\s*-?\s*([0-9]{1,2})(?:[^0-9]|$)`)

// layoutToken gives the position that a title names with its letter, such as t5 for "Middle European MO 2013 T 5", when that position is in the layout and its letter is not p. It gives "" when the title names no such position or more than one.
func layoutToken(layout []content.Layout, title string) string {
	var all []string
	for _, l := range layout {
		all = append(all, l.Positions...)
	}
	found := ""
	for _, m := range tokenRE.FindAllStringSubmatch(title, -1) {
		letter := strings.ToLower(m[1])
		n, _ := strconv.Atoi(m[2])
		p := letter + strconv.Itoa(n)
		if letter == "p" || !slices.Contains(all, p) {
			continue
		}
		if found != "" && found != p {
			return ""
		}
		found = p
	}
	return found
}

// titleMark gives the number after the one mark of a problem number in a title, such as 5 for "MEMO 2008 Team P5" or 1 for "MEMO 2009 problem 1 team competition". It gives 0 when the title has no such mark or more than one.
func titleMark(title string) int {
	ms := titleMarkNumRE.FindAllStringSubmatch(title, -1)
	if len(ms) != 1 || len(titleMarkRE.FindAllString(title, -1)) != 1 {
		return 0
	}
	n, _ := strconv.Atoi(ms[0][1])
	return n
}

// titleMarkNumRE finds the number after a mark of a problem number in a title.
var titleMarkNumRE = regexp.MustCompile(`(?i)(?:\bp|#|\bproblem|\bq|\bpb|/)\s*\.?\s*([0-9]{1,2})\b`)

// dayRE finds the day in a title, such as "Day 2" or "Second Day".
var dayRE = regexp.MustCompile(`(?i)\b(?:day\s*([1-9])|(first|second|third)\s+day)\b`)

// titleDay gives the day that a text names, or 0 when it names no day or more than one.
func titleDay(text string) int {
	ms := dayRE.FindAllStringSubmatch(text, -1)
	if len(ms) != 1 {
		return 0
	}
	if ms[0][1] != "" {
		d, _ := strconv.Atoi(ms[0][1])
		return d
	}
	return ordinals[strings.ToLower(ms[0][2])]
}

// dayPosition gives the position of problem n of day d of the paper l, which has l.Days days with the same number of problems each. The number counts from 1 in the day, as in "Senior League Second Day P1", or it is a number of the paper, as in "Day 2 P5". It gives "" when the paper has no such problem.
func dayPosition(l content.Layout, d, n int) string {
	per := len(l.Positions) / l.Days
	switch {
	case d < 1 || d > l.Days || n < 1:
		return ""
	case n <= per:
		return l.Positions[(d-1)*per+n-1]
	case n > (d-1)*per && n <= d*per:
		return l.Positions[n-1]
	}
	return ""
}

var (
	// titleEndRE finds the problem number at the end of a title, for example "P6" in "IMC 2013 P6" or "#4" in "2016 JMO Preliminary #4".
	titleEndRE = regexp.MustCompile(`(?i)(?:\bp|#|\bproblem|\bq|\bpb|/)\s*\.?\s*([0-9]{1,2})\.?$`)
	// titleMarkRE finds each mark of a problem number in a title. A title with two marks, such as "Tuymaada 2014 Day 1 P2 Juniors P4 Seniors", gives no number.
	titleMarkRE = regexp.MustCompile(`(?i)(?:\bp|#|\bproblem|\bq|\bpb|/)\s*\.?\s*[0-9]`)
)

// titleNumber gives the position that the final problem number of a title gives, for example "p5" for "China TST 1986 problem 5". It gives "" when the text is not a short title, has no final number or has more than one number mark.
func titleNumber(title string) string {
	title = strings.TrimSpace(title)
	if !isTitle(title) {
		return ""
	}
	m := titleEndRE.FindStringSubmatch(title)
	if m == nil || len(titleMarkRE.FindAllString(title, -1)) != 1 {
		return ""
	}
	return seed.Position(m[1])
}

// paperRE matches a line that names the paper or the day and the problem, for example "2006 IrMO Paper 2 P1", "Day 1, Problem 3" or "Second Day P2".
var paperRE = regexp.MustCompile(`(?i)^[^\n]{0,40}?\b(?:(?:paper|day)\s*([1-9])|(first|second|third)\s+(?:paper|day))[\s,.:-]*(?:p|problem)\s*([0-9]{1,2})[.:)]?[ ]*(?:\n|$)`)

// ordinals gives the number of an ordinal word in a paper line, such as "Second Day".
var ordinals = map[string]int{"first": 1, "second": 2, "third": 3}

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
	var paper int
	if m[2] >= 0 {
		paper, _ = strconv.Atoi(text[m[2]:m[3]])
	} else {
		paper = ordinals[strings.ToLower(text[m[4]:m[5]])]
	}
	n, _ := strconv.Atoi(text[m[6]:m[7]])
	rest = strings.TrimSpace(text[m[1]:])
	// In a layout of papers of several days, such as the leagues of the Tuymaada Olympiad, the number of the line is a day of the paper that the line names.
	if slices.ContainsFunc(layout, func(l content.Layout) bool { return l.Days > 1 }) {
		i := namedIndex(layout, text[:m[1]])
		if i < 0 || layout[i].Days < 2 {
			return "", rest, true
		}
		return dayPosition(layout[i], paper, n), rest, true
	}
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
