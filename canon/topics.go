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
	// units holds the years of competitions with days or tests, by the start of their appearances, such as "vn-tst/2019/". resolved tells that resolve has run.
	units    map[string]*unitYear
	resolved bool
}

// Add reads one row of a numbered source. When the text of the row starts with a line that names the paper and the problem, the position comes from that line and the layout of the competition, because the source numbers some papers again from 1. A row with no year, no number or no topic link gives nothing, except a row whose text is a title that gives the year, as described at TitleAt. When the text is a title that names the problem, such as "Vietnam MO 1982 P5", the position comes from the title, as described at TitlePosition. When two rows give one topic two different appearances, the topic gets no appearance. When two topics claim one appearance, neither gets it, because the source numbers some papers again from 1 on each day, as the AoPS wiki does for some years of the IMO.
func (t *Topics) Add(r seed.Row) {
	m := topicRE.FindStringSubmatch(r.Ref)
	if m == nil {
		return
	}
	pl, ok := t.place(r)
	if !ok {
		return
	}
	t.resolved = false
	prefix := pl.code + "/" + strconv.Itoa(pl.year) + pl.letter + "/"
	if pl.number > 0 {
		if t.units == nil {
			t.units = map[string]*unitYear{}
		}
		y := t.units[prefix]
		if y == nil {
			y = &unitYear{prefix: prefix, numbers: map[int]map[string]bool{}, fixed: map[string]map[string]bool{}}
			t.units[prefix] = y
		}
		if y.numbers[pl.number] == nil {
			y.numbers[pl.number] = map[string]bool{}
		}
		y.numbers[pl.number][m[1]] = true
		if pl.fixed && !pl.unsure {
			if y.fixed[pl.pos] == nil {
				y.fixed[pl.pos] = map[string]bool{}
			}
			y.fixed[pl.pos][m[1]] = true
		}
		if pl.unit != nil {
			y.perDay = y.perDay || pl.unit.perDay
			if pl.unit.open != nil {
				y.open = append(y.open, unitRow{topic: m[1], open: pl.unit.open, running: pl.unit.running})
				return
			}
		}
	}
	t.claim(prefix+pl.pos, m[1], pl.unsure)
}

// claim records that the topic claims the appearance at. When the topic claims another appearance too, or when unsure is true, the topic gets no appearance.
func (t *Topics) claim(at, topic string, unsure bool) {
	if t.at == nil {
		t.at, t.topics = map[string]string{}, map[string]map[string]bool{}
	}
	if t.topics[at] == nil {
		t.topics[at] = map[string]bool{}
	}
	t.topics[at][topic] = true
	if old, ok := t.at[topic]; unsure || (ok && old != at) {
		at = ""
	}
	t.at[topic] = at
}

// Place gives the year, the letter of the paper and the position of a row of a numbered source, from its number and its title, by the rules of Add. The import uses it for the rows of the AoPS wiki themselves, so that a row gets the appearance that its topic gets. The result ok is false when the row has no position, when its title names another number than the number column, or when another topic claims the same appearance, as in a year where the wiki numbers each day from 1. A topic can have appearances in two competitions, such as the IMO and the IMO Shortlist.
func (t *Topics) Place(r seed.Row) (year int, letter, pos string, ok bool) {
	pl, ok := t.place(r)
	if !ok || pl.unsure {
		return 0, "", "", false
	}
	m := topicRE.FindStringSubmatch(r.Ref)
	if m == nil {
		return 0, "", "", false
	}
	prefix := pl.code + "/" + strconv.Itoa(pl.year) + pl.letter + "/"
	if pl.unit != nil && pl.unit.open != nil {
		// The day of the row is not known from its title, so the row takes the position that resolve gives its topic.
		t.resolve()
		y := t.units[prefix]
		if y == nil || !slices.Contains(pl.unit.open, y.at[m[1]]) {
			return 0, "", "", false
		}
		pl.pos = y.at[m[1]]
	}
	at := prefix + pl.pos
	if len(t.topics[at]) != 1 || !t.topics[at][m[1]] {
		return 0, "", "", false
	}
	return pl.year, pl.letter, pl.pos, true
}

// placement is the appearance that place gives a row.
type placement struct {
	code   string
	year   int
	letter string
	pos    string
	// unsure is true when the title names a part and its number does not agree with the number column.
	unsure bool
	// fixed tells that the title or the first line of the text gives the position, not the number alone.
	fixed bool
	// unit is set for a row of a year with days or tests when unitPosition reads it.
	unit *unitPlace
	// number is the number of a row of a year with days or tests, and 0 for other rows.
	number int
}

// place gives the appearance of a row as Add describes it. The result ok is false when the row gives none.
func (t *Topics) place(r seed.Row) (pl placement, ok bool) {
	pl.code = t.Aliases.Code(r.Labels)
	if pl.code == "" || pl.code == "-" {
		return placement{}, false
	}
	var comp *content.Competition
	if t.Corpus != nil {
		comp = t.Corpus.Competition(pl.code)
	}
	// The title is the source line of the post when the row has one, or else the text of the row, which is only a title in some rows.
	title := r.Title
	if title == "" {
		title = r.Statement
	}
	if comp != nil && comp.EditionForm == "year-season" {
		year, season, pos, ok := SeasonPosition(r.Number, title)
		if !ok {
			return placement{}, false
		}
		return placement{code: pl.code, year: year, letter: "-" + season, pos: pos}, true
	}
	pos := seed.Position(r.Number)
	if pos == "" && !gradeNumberRE.MatchString(strings.TrimSpace(r.Number)) {
		return placement{}, false
	}
	// Agent decision: a title that names the longlist and the number of a problem, such as "IMO LongList 1959 1966 P24", gives the number of the problem in the longlist. The AoPS wiki puts such rows on some shortlist pages with that number, so a row of a shortlist with such a title gives no position. A title that also names the shortlist, such as "IMO ShortList 1988 P15 Iceland 1 P34 of ILL", is read as before.
	if comp != nil && comp.TopicPositions && longlistRE.MatchString(title) && !shortlistRE.MatchString(title) && titleMarkRE.MatchString(title) {
		return placement{}, false
	}
	year := r.Year
	if comp != nil && comp.EditionForm == "year-letter" {
		pl.letter = Letter(r.Labels)
	}
	// Agent decision: in a year with one paper for each grade, the wiki numbers the problems of each grade from 1, so a row gets a position only from its grade, as described at gradePosition.
	y := year
	if ys := titleYearRE.FindAllString(title, -1); y == 0 && len(ys) > 0 && !slices.ContainsFunc(ys, func(s string) bool { return s != ys[0] }) {
		y, _ = strconv.Atoi(ys[0])
	}
	if p, graded := gradePosition(comp, y, r.Number, title); graded {
		pl.year, pl.pos = y, p
		return pl, p != ""
	}
	if pos == "" {
		return placement{}, false
	}
	n, _ := strconv.Atoi(strings.TrimPrefix(pos, "p"))
	// In a year whose layout is a list of days or tests, the title and the number give the position together, as described at unitPosition. A title that no position of the layout agrees with, such as "EGMO 2017 P4" with the number 3, or one that names another year or another competition, is read as in any other year.
	var u unitPlace
	kind := ""
	if year != 0 && strings.HasPrefix(pos, "p") && n > 0 {
		if kind = unitKind(comp, year); kind != "" {
			pl.number = n
			title := strings.TrimSpace(title)
			if isTitle(title) {
				u = unitPosition(comp.Shape(year).Layout, kind, title, year, n)
			} else if _, _, found := PaperPosition(comp, year, title); !found && !unitRE.MatchString(title) && !titleMarkRE.MatchString(title) {
				// A text that is not a title gives the day only by a first line such as "Day 2 P1". A text with no such line and no day or problem mark does not tell the day. A longer text that names one, such as "IMO ShortList 1973 Socialists Republic Of Czechoslovakia 2 IMO 1973 Day 1 P1", is read as in any other year.
				u = unitPosition(comp.Shape(year).Layout, kind, "", year, n)
			}
		}
	}
	switch {
	case year == 0:
		y, p, l, found := TitleAt(comp, title, pos)
		if !found && pl.letter == "" {
			// The number of the title can differ from the number column. TitlePosition then reads the title.
			if ys := titleYearRE.FindAllString(title, -1); len(ys) == 1 {
				y, _ = strconv.Atoi(ys[0])
				p = TitlePosition(comp, t.Aliases, y, pos, title)
				found = p != "" && (comp == nil || comp.EditionForm != "year-letter")
			}
		}
		if !found {
			return placement{}, false
		}
		year, pos, pl.letter = y, p, l
	case u.ok:
		pl.unit, pos, pl.fixed = &u, u.pos, u.pos != ""
	default:
		if p, _, found := PaperPosition(comp, year, title); found {
			pos, pl.fixed = p, true
		} else if p, found := namedPaper(comp, year, title, pos); found {
			pos, pl.fixed = p, true
		} else if p := TitlePosition(comp, t.Aliases, year, pos, title); p != "" {
			pos, pl.fixed = p, true
		} else if tn := titleNumber(title); tn != "" && strings.HasPrefix(pos, "p") && tn != pos {
			// The title names a part, such as "China TST 1986 problem 5", so TitlePosition does not read it. Its number does not agree with the number of the row, so the topic gets no appearance. It still claims the appearance of its number, because the true position is not known.
			pl.unsure = true
		}
		if kind != "" && !pl.fixed && !pl.unsure {
			// Only the number gives the position, and the number does not tell the day, so the row is read as a row with no day.
			if u := unitPosition(comp.Shape(year).Layout, kind, "", year, n); u.ok {
				pl.unit, pos, pl.fixed = &u, u.pos, u.pos != ""
			}
		}
	}
	pl.year, pl.pos = year, pos
	return pl, pos != "" || (pl.unit != nil && pl.unit.open != nil)
}

// Year gives the year of the topic of ref when the numbered source gives the topic one position of competition code. It gives 0 when it does not know the year.
func (t *Topics) Year(ref, code string) int {
	t.resolve()
	topic, ok := strings.CutPrefix(ref, "aops-topic:")
	if !ok {
		return 0
	}
	at := t.at[topic]
	rest, ok := strings.CutPrefix(at, code+"/")
	if !ok || len(t.topics[at]) != 1 || len(rest) < 4 {
		return 0
	}
	y, err := strconv.Atoi(rest[:4])
	if err != nil {
		return 0
	}
	return y
}

// Position gives the position of the topic of ref, such as aops-topic:1738331, when its competition and its year are code and year, and the letter of its paper, such as "a" for the AMC 10A, when the numbered source gives it. For a competition with seasons, the letter is the season of the edition, such as "-fall" for tot/2024-fall. It gives "" when it does not know the position.
func (t *Topics) Position(ref, code string, year int) (pos, letter string) {
	t.resolve()
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
	if !ok || (letter != "" && letter != "a" && letter != "b" && !seasonLetterRE.MatchString(letter)) {
		return "", ""
	}
	return pos, letter
}

var (
	titleYearRE = regexp.MustCompile(`\b(?:19|20)[0-9]{2}\b`)
	titleNumRE  = regexp.MustCompile(`(?:#|/|(?i:\bp#?|\bproblem\s*|\bpb\s*|\bq\.?\s*)|Q\.)([0-9]{1,2})\b`)
	// titleBareRE finds a problem number with no mark at the end of a title, as in "Japan MO Finals 2020 1" or "2019 Japan Junior MO (JJMO) Finals 1". The word before the number must be the year or "Finals". After other words, such as "Algebra 1", "Level 2" or "Day 2", the number is often not the number of a problem in the paper of the competition.
	titleBareRE = regexp.MustCompile(`(?i)(?:^|\s)(?:(?:19|20)[0-9]{2}|finals?)\s+([0-9]{1,2})$`)
	// prelimRE finds a title that names a preliminary round, such as "2016 JMO Preliminary #4" (the wiki also misspells it, with "mi" and "li" swapped). That paper is not the final round, so its number gives no position of the final round.
	prelimRE      = regexp.MustCompile(`(?i)\bpre(?:liminary|milinary)\b|\bprelims?\b`)
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
	if prelimRE.MatchString(title) {
		return 0, "", "", false
	}
	nums := titleNumRE.FindAllStringSubmatch(title, -1)
	if m := titleBareRE.FindStringSubmatch(title); len(nums) == 0 && m != nil {
		// Agent decision: a title with no mark can end with the problem number, as in "Japan Junior MO Finals 2018 1". It must agree with the number column like a marked number.
		nums = [][]string{{m[1], m[1]}}
	}
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

// PaperPosition reads a first line of text that names the paper and the problem, such as "2006 IrMO Paper 2 P1", and gives the position of that problem in the layout of the competition c in the year, and the text after the line. The problem number counts from 1 in its paper, or it is a position of the paper, as in "Day 2 Problem 4". When both readings give a position and the two positions differ, the position is "". The result ok is false when the text has no such line or c has no layout. The position is "" when the layout has no such problem.
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
	p := "p" + strconv.Itoa(n)
	switch in, named := n <= len(ps), slices.Contains(ps, p); {
	case in && named && ps[n-1] != p:
		// The paper has more problems than the papers before it, as Day 2 of the IMO 1960 has p4 to p7. Then "Day 2 P4" can be the fourth problem of the day or the position p4, so the line gives no position.
		return "", rest, true
	case in:
		return ps[n-1], rest, true
	case named:
		return p, rest, true
	}
	return "", rest, true
}

var (
	// sectionRE finds a problem of a shortlist with sections in a title, for example "G2", "Algebra 2" or "number theory problem 4".
	sectionRE = regexp.MustCompile(`(?i)(?:^|[^a-z])(algebra|combinatorics|geometry|number theory|[acgn])\s*(?:problem\s*)?#?\s*([0-9]{1,2})\b`)
	// shortlistRE tells that a title names a shortlist.
	shortlistRE = regexp.MustCompile(`(?i)short\s*list|\bisl\b`)
	// longlistRE tells that a title names a longlist.
	longlistRE = regexp.MustCompile(`(?i)long\s*list|\bill\b`)
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

var (
	// gradeRE finds a grade in a title, such as "Grade 10", "grade XI", "9th Grade", "XI Class" or "Class 11".
	gradeRE = regexp.MustCompile(`(?i)\b(?:(?:grade|class)\s*([0-9]{1,2}|[ivx]{1,4})|([0-9]{1,2})(?:st|nd|rd|th)?\s*(?:grade|class)|([ivx]{1,4})\s*(?:grade|class))\b`)
	// gradeNumberRE matches a number that names the grade and the problem, such as "10.3" for problem 3 of grade 10.
	gradeNumberRE = regexp.MustCompile(`^([0-9]{1,2})\.([0-9]{1,2})$`)
	// gradeTokenRE finds a number such as "10.3" in a title, as in "2018 Belarusian NMO 10.3".
	gradeTokenRE = regexp.MustCompile(`(?:^|[^0-9.])([0-9]{1,2})\.([0-9]{1,2})(?:[^0-9.]|$)`)
	// gradeMarkRE finds the number after a mark of a problem number in a title, as titleMarkNumRE does, and also after "problem nr.", as in "Moldova NMO 2002 grade 7 problem nr.5".
	gradeMarkRE = regexp.MustCompile(`(?i)(?:\bp|#|\bproblem(?:\s*nr)?|\bq|\bpb|/)\s*\.?\s*([0-9]{1,2})\b`)
	// firstRoundRE finds a title that names the first round, such as "Sharygin Geometry Olympiad 2016 First Round P1 grade8". In a layout with papers of the grades and one other paper, the first round is the other paper.
	firstRoundRE = regexp.MustCompile(`(?i)\b(?:first|1st)\s+round\b`)
	// roundRE finds a title that names a numbered round, such as "St. Petersburg MO 2009 round II grade 9 P2", "Olympiad 2009 2nd Round Grade 10" or "Round 3".
	roundRE = regexp.MustCompile(`(?i)\b(?:(?:first|second|third|fourth|1st|2nd|3rd|4th|[1-4])\s+round|round\s+(?:[1-4]|i{1,3}|iv))\b`)
	// bracketRE finds a part of a title in brackets with no bracket inside, such as "(Grade 8 9)".
	bracketRE = regexp.MustCompile(`\([^()]*\)`)
	// gradePaperRE matches the name of a paper of one grade, such as "Grade 10".
	gradePaperRE = regexp.MustCompile(`(?i)^grade ([0-9]{1,2})$`)
)

// romans gives the value of a roman numeral of a grade, from I to XII.
var romans = map[string]int{"i": 1, "ii": 2, "iii": 3, "iv": 4, "v": 5, "vi": 6, "vii": 7, "viii": 8, "ix": 9, "x": 10, "xi": 11, "xii": 12}

// titleGrade gives the grade that a title names, such as 10 for "Kosovo MO 2019 Grade 10 P1". It gives 0 when the title names no grade, and -1 when it names two different grades.
func titleGrade(title string) int {
	g := 0
	for _, m := range gradeRE.FindAllStringSubmatch(title, -1) {
		s := strings.ToLower(m[1] + m[2] + m[3])
		n, err := strconv.Atoi(s)
		if err != nil {
			n = romans[s]
		}
		if n == 0 || (g != 0 && n != g) {
			return -1
		}
		g = n
	}
	return g
}

// gradePosition reads a row of a competition that has one paper for each grade in the year, such as the Kosovo MO, where the wiki numbers the problems of each grade from 1. The name of such a paper is "Grade G". The grade comes from the title, such as "Kosovo MO 2019 Grade 10 P1", or from a number such as "10.3", in the number column or in the title, as in "2018 Belarusian NMO 10.3". The number of the problem comes from the number column, and a number of the title must agree with it. In a paper of several days, the number column can count from 1 in each day, as in the Moldova NMO 2002, where problem 5 of a grade has the number 1, and a title that names the day, such as "Sharygin 2019 Finals Day 2 Grade 10 P4", counts its number in the day or in the paper. In a number such as "P9.8" the mark names the grade, so the problem is 8. It gives the position of the problem in the paper of its grade. graded is false when the layout of the year has no paper of a grade, or when the title names another paper of the layout, such as the correspondence round of the Sharygin Olympiad. In such a layout a grade in brackets does not name a paper. Then the row is read by the other rules. The position is "" when the row does not name one grade and one problem of that grade, so that a row with no grade is never placed by its number alone.
func gradePosition(c *content.Competition, year int, number, title string) (pos string, graded bool) {
	if c == nil || year == 0 {
		return "", false
	}
	layout := c.Shape(year).Layout
	grades := map[int]int{}
	for i, l := range layout {
		if m := gradePaperRE.FindStringSubmatch(l.Name); m != nil {
			g, _ := strconv.Atoi(m[1])
			grades[g] = i
		}
	}
	if len(grades) == 0 {
		return "", false
	}
	// Each source of the grade and of the number must agree. 0 means not known.
	g, n := 0, 0
	agree := func(old *int, v int) bool {
		if *old != 0 && *old != v {
			return false
		}
		*old = v
		return true
	}
	number = strings.TrimSpace(number)
	if m := gradeNumberRE.FindStringSubmatch(number); m != nil {
		g, _ = strconv.Atoi(m[1])
		n, _ = strconv.Atoi(m[2])
	} else if p := seed.Position(number); strings.HasPrefix(p, "p") {
		n, _ = strconv.Atoi(p[1:])
	}
	if n == 0 {
		return "", true
	}
	title = strings.TrimSpace(title)
	if !isTitle(title) {
		return "", true
	}
	// other tells that the layout has a paper that is not a paper of a grade, such as the correspondence round of the Sharygin Olympiad.
	other := len(grades) < len(layout)
	if i := namedIndex(layout, title); other && ((i >= 0 && gradePaperRE.FindStringSubmatch(layout[i].Name) == nil) || firstRoundRE.MatchString(title)) {
		// Agent decision: a title that names the other paper, such as "Sharygin Geometry Olympiad Correspondence round 2016 P 6 Grade 8" or "Sharygin Geometry Olympiad 2016 First Round P1 grade8", is read by the other rules, because a grade in such a title is the grade that the problem is meant for, not a paper.
		return "", false
	}
	if !other && roundRE.MatchString(title) {
		// Agent decision: when every paper of the layout is a paper of a grade, a title that names a round, such as "St. Petersburg MO 2009 round II grade 9 P2", can be a row of an earlier round that numbers its problems from 1 in each grade too, so it gives no position.
		return "", true
	}
	// t is the number of the problem that the title names, 0 when it names none.
	t := 0
	ms := gradeTokenRE.FindAllStringSubmatchIndex(title, -1)
	switch {
	case len(ms) > 1:
		return "", true
	case len(ms) == 1:
		tg, _ := strconv.Atoi(title[ms[0][2]:ms[0][3]])
		t, _ = strconv.Atoi(title[ms[0][4]:ms[0][5]])
		if !agree(&g, tg) {
			return "", true
		}
		// The mark of a number such as "P9.8" names the grade, not the problem, so the number is taken out before the marks and the grade words are read.
		title = title[:ms[0][2]] + " " + title[ms[0][5]:]
	}
	// The day is also taken out of the grade words, so that "Day 2 Grade 10" does not read as grade 2.
	words := dayRE.ReplaceAllString(title, " ")
	// The marks of the problem number are taken out too, so that "problem 1 grade 8" does not read as grade 1. The marks are read from the title below.
	words = gradeMarkRE.ReplaceAllString(words, " ")
	if other {
		// Agent decision: when the layout has another paper, a grade in brackets, as in "Sharygin 2023 P1 (Grade 8)", is the grade that a problem of that paper is meant for, so it does not name a paper.
		for w := ""; w != words; {
			w, words = words, bracketRE.ReplaceAllString(words, " ")
		}
	}
	tg := titleGrade(words)
	if tg < 0 || (tg > 0 && !agree(&g, tg)) {
		return "", true
	}
	if g == 0 && other && len(layout)-len(grades) == 1 && titleDay(title) == 0 {
		// Agent decision: a title that names no grade, such as "Sharygin Geometry Olympiad 2012 P24", gives a problem of the other paper when its number is above the size of each paper of a grade, because only that paper has such a number. A smaller number gives no position, because it can be a problem of a grade.
		most := 0
		for _, l := range layout {
			if gradePaperRE.MatchString(l.Name) {
				most = max(most, len(l.Positions))
			}
		}
		t2 := t
		if marks := gradeMarkRE.FindAllStringSubmatch(title, -1); len(marks) == 1 {
			t2, _ = strconv.Atoi(marks[0][1])
		}
		if n > most && (t2 == 0 || t2 > most) {
			return "", false
		}
	}
	i, ok := grades[g]
	if g == 0 || !ok {
		return "", true
	}
	ps := layout[i].Positions
	if marks := gradeMarkRE.FindAllStringSubmatch(title, -1); len(marks) > 1 {
		return "", true
	} else if len(marks) == 1 {
		m, _ := strconv.Atoi(marks[0][1])
		if t != 0 && m != t {
			return "", true
		}
		t = m
	}
	days := layout[i].Days
	per := len(ps)
	if days > 1 && len(ps)%days == 0 {
		per = len(ps) / days
	}
	// Agent decision: in a paper of several days, a title that names the day, such as "Sharygin 2019 Finals Day 2 Grade 10 P4", counts its number from 1 in the day or in the paper, as described at dayPosition, and the number column must give the same problem, counted in the paper or in the day.
	if d := titleDay(title); d > 0 && days > 1 {
		if t == 0 {
			t = n
		}
		p := dayPosition(layout[i], d, t)
		k := slices.Index(ps, p)
		if p == "" || (n != k+1 && n != k+1-(d-1)*per) {
			return "", true
		}
		return p, true
	}
	if t == 0 {
		t = n
	}
	if t < 1 || t > len(ps) {
		return "", true
	}
	// Agent decision: the number column counts in the paper, or from 1 in the day of the problem.
	if t != n && (days < 2 || per == len(ps) || t-(t-1)/per*per != n) {
		return "", true
	}
	return ps[t-1], true
}
