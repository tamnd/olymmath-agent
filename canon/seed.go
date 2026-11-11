package canon

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/normalize"
	"github.com/tamnd/olymmath-agent/seed"
)

// MinText is the shortest statement, in characters, that an import accepts. A shorter text is a title or a reference, for example "USAMO 2022/1".
const MinText = 40

// MinWords is the smallest number of words of a statement with no formula. A shorter text is a title, for example "IMO LongList 1967 The Democratic Republic Of Germany 1". A text with no formula and no punctuation that ends a sentence is also a title, and so is a short list of references with no formula, such as "Romanian IMO TST 1999 problem 12; 17th Iranian Math. Olympiad".
const MinWords = 10

var (
	letterRE   = regexp.MustCompile(`(?i)(?:^|[\s\d])(ii|i|a|b)\s*$`)
	positionRE = regexp.MustCompile(`^[a-z]{1,2}[0-9]{1,4}$`)
	// titleRE matches a text that names a problem instead of stating it.
	// refRE matches a short text with no formula that names the source of a problem, for example "Tuymaada 2002 day 1 problem 1. Author : A. Golovanov." or "Kvant M1467 (Sol: 3/1995) IMO 1994 P1 IMO Shortlist 1994 A2".
	refRE   = regexp.MustCompile(`(?i)\b(19|20)[0-9]{2}\b.*\b(problem|shortlist|olympiad|tst|authors?|grade|round|day|kvant)\b|\b(problem|shortlist|olympiad|tst|authors?|grade|round|day|kvant)\b.*\b(19|20)[0-9]{2}\b`)
	titleRE = regexp.MustCompile(`(?i)^[\w .'-]*\b(19|20)[0-9]{2}\b[\w .'-]*(/|#|problem\s*)[a-z]?[0-9]{1,2}\b[\w ./#-]*$`)
)

// isRef tells if a text names the source of a problem instead of stating it: a text with no formula, a year, a word such as "problem" or "shortlist", fewer than 20 words and no question mark at the end.
func isRef(text string) bool {
	return !strings.Contains(text, "$") && len(strings.Fields(text)) < 20 && !strings.HasSuffix(text, "?") && refRE.MatchString(text)
}

// Letter gives the letter of the edition of a competition with two papers each year, from the labels of a row: "AIME I" gives a, "AIME II" gives b and "AMC 10B" gives b. It returns "" when no label has a letter.
func Letter(labels []string) string {
	for _, l := range labels {
		m := letterRE.FindStringSubmatch(strings.TrimSpace(l))
		if m == nil {
			continue
		}
		switch strings.ToLower(m[1]) {
		case "a", "i":
			return "a"
		case "b", "ii":
			return "b"
		}
	}
	return ""
}

// Paper gives the part and the letter of the paper of a row of a competition with two papers each year, from the labels of the row. A label that names a season, such as "2021 AMC 10 B Fall" or "Fall 2021 AMC 10 B", gives the part of that season when the register lists papers of that part for the year, as the fall of 2021 of the AMC 10. A spring label gives the main part. The result ok is false when a label names a season that is not a part of the year, because the paper is not known. The letter is "" when no label names one.
func Paper(c *content.Competition, year int, labels []string) (part, letter string, ok bool) {
	plain := make([]string, len(labels))
	for i, l := range labels {
		if m := seasonRE.FindString(l); m != "" && part == "" {
			part = strings.ToLower(m)
		}
		plain[i] = strings.TrimSpace(seasonLabelRE.ReplaceAllString(l, " "))
	}
	if part == "autumn" {
		part = "fall"
	}
	if part == "spring" {
		part = ""
	}
	if part != "" && (c == nil || c.PartLetters(year, part) == nil) {
		return "", "", false
	}
	return part, Letter(plain), true
}

// seasonLabelRE matches a season word in a label, with the brackets around it, as in "2021 AMC 10A (Fall)".
var seasonLabelRE = regexp.MustCompile(`(?i)\(?\b(?:fall|spring|autumn)\b\)?`)

// Seeds makes candidates from seed rows. It accepts a row when the alias table resolves its labels to a competition of comps (all competitions when comps is empty), when its year and its position are valid for the competition, and when its text is a statement. The result maps each appearance to its candidates.
type Seeds struct {
	Corpus  *content.Corpus
	Aliases *seed.Aliases
	Comps   []string
	// ThisYear is the last valid year.
	ThisYear int
	// Accept, when it is not nil, can refuse a text, for example a text with a formula that does not render.
	Accept func(text string) bool
	// Locate, when it is not nil, gives the position of a row that has no number, for example from the official paper of the edition. It gives "" when it does not know the position.
	Locate func(code string, year int, text string) string

	// Confirm, when it is not nil, can refuse the position that a row got from Topics. It gives false when the official paper of the edition has another problem at that position.
	Confirm func(code string, year int, text, pos string) bool

	// Matched, when it is not nil, gives the appearance of a row that the content match placed (package keymatch), by the reference of the row (source|file#row). Such a row takes that appearance and no other rule reads its labels, its number or its title.
	Matched map[string]string
	// Matches counts the rows that got an appearance from Matched.
	Matches int

	// Topics, when it is not nil, gives the numbers of AoPS topics. A row with no number and the reference of a topic takes the number of the topic when the topic has the same competition and year, and the letter of the paper when the row has none.
	Topics *Topics

	pending []pendingRow
	// lettered[code][year] is true when a row of the year has an edition letter.
	lettered map[string]map[int]bool
	// sections[code][year] is true when a row of the year has a position with a section letter, such as c3 of the shortlist.
	sections map[string]map[int]bool
	// Skipped counts the rows that were not accepted, by reason.
	Skipped map[string]int
	// Located counts the rows that got a position from Locate.
	Located int
	// Numbered counts the rows that got a position from Topics.
	Numbered int
	// Led counts the rows that got a position from the problem mark at the start of their text, as described at LeadNumber and leadSure.
	Led int
	// collections counts the rows of each collection, and leads gives the numbers of the lead marks of each collection.
	collections map[string]int
	leads       map[string][]int
	// Trace, when it is not nil, gets each row with the code of its competition and the reason why it was skipped. The reason is "" for a row that gives a candidate, and at is the appearance of the candidate. Candidates also calls it for a candidate that it drops.
	Trace func(r seed.Row, code, at, reason string)
}

type pendingRow struct {
	row    seed.Row
	c      *Candidate
	comp   *content.Competition
	year   int
	letter string
	pos    string
	// lead is not nil for a row that got its position from the problem mark at the start of its text.
	lead *leadRow
	k    int
	one  bool
	// matched is true for a row that the content match placed. Its position is the appearance of a record, so the numbering of other rows does not apply to it.
	matched bool
}

func (s *Seeds) skip(r seed.Row, code, reason string) {
	if s.Skipped == nil {
		s.Skipped = map[string]int{}
	}
	s.Skipped[reason]++
	if s.Trace != nil {
		s.Trace(r, code, "", reason)
	}
}

// MatchRef gives the reference of a row in Seeds.Matched.
func MatchRef(r seed.Row) string { return r.Source + "|" + r.File + "#" + strconv.Itoa(r.Row) }

// Add reads one seed row.
func (s *Seeds) Add(r seed.Row) {
	if at, ok := s.Matched[MatchRef(r)]; ok {
		s.addMatched(r, at)
		return
	}
	code := s.Aliases.Code(r.Labels)
	if code == "" || code == "-" {
		s.skip(r, code, "no competition")
		return
	}
	if len(s.Comps) > 0 && !slices.Contains(s.Comps, code) {
		s.skip(r, code, "other competition")
		return
	}
	comp := s.Corpus.Competition(code)
	if comp == nil {
		s.skip(r, code, "no competition")
		return
	}
	if comp.EditionForm == "year-season" && r.Number == "" && r.Title == "" && s.Topics != nil {
		// The year of a collection of a competition with seasons, such as "2020 Tournament of Towns", can be the year of either edition of a tournament that runs from fall to spring. The title that the numbered source gives the topic names the edition, so its year is used.
		if y := s.Topics.Year(r.Ref, code); y != 0 {
			r.Year = y
		}
	} else if r.Year == 0 && r.Number == "" && s.Topics != nil {
		// A collection such as the AoPS AHSME set has no year in its labels. The topic of the row gives the year when the numbered source knows the topic.
		r.Year = s.Topics.Year(r.Ref, code)
	}
	pos := seed.Position(r.Number)
	topicLetter := ""
	if s.Topics != nil && (r.Title != "" || (r.Number != "" && topicRE.MatchString(r.Ref))) {
		// A row of the AoPS wiki takes the place that its title and its number give, as its topic does in Topics, because the wiki numbers some papers from 1. A row with no title is read the same way, so that its number alone does not take an appearance that another topic claims.
		y, l, p, ok := s.Topics.Place(r)
		if !ok {
			s.skip(r, code, "no position from the title")
			return
		}
		r.Year, pos, topicLetter = y, p, l
	}
	if r.Year == 0 || (comp.FirstYear > 0 && r.Year < comp.FirstYear) || (s.ThisYear > 0 && r.Year > s.ThisYear) {
		s.skip(r, code, "no valid year")
		return
	}
	// A line such as "Paper 2 P1" at the start of the text is surer than the number, because some sources number each paper from 1.
	if p, rest, ok := PaperPosition(comp, r.Year, r.Statement); ok && r.Title == "" {
		pos, r.Statement = p, rest
		if pos == "" {
			s.skip(r, code, "no position in the paper")
			return
		}
	}
	if pos == "" && r.Number == "" && s.Topics != nil {
		if pos, topicLetter = s.Topics.Position(r.Ref, code, r.Year); pos != "" {
			if s.Confirm != nil && !s.Confirm(code, r.Year, normalize.Statement(r.Statement, ""), pos) {
				s.skip(r, code, "not the problem of the official paper")
				return
			}
			s.Numbered++
		}
	}
	collection := r.Source + "|" + code + "/" + strconv.Itoa(r.Year) + "|" + strings.Join(r.Labels, "\x1f")
	if r.Number == "" && r.Title == "" {
		if s.collections == nil {
			s.collections = map[string]int{}
		}
		s.collections[collection]++
	}
	var lead *leadRow
	var leadK int
	if pos == "" && r.Number == "" && r.Title == "" && !topicRE.MatchString(r.Ref) && (comp.EditionForm == "" || comp.EditionForm == "year") {
		// Agent decision: a row with no number whose text starts with a problem mark, such as "Problem 4.", takes the number of the mark, but only when the whole collection agrees, as described at leadSure, and when no topic of the numbered source claims the appearance or gives the row another one.
		if n, rest, ok := LeadNumber(r.Statement); ok {
			p, k := leadPosition(comp, r.Year, n)
			at := code + "/" + strconv.Itoa(r.Year) + "/" + p
			switch {
			case p == "":
				s.skip(r, code, "no valid position")
				return
			case s.Topics.Claimed(at, r.Ref) || s.Topics.Elsewhere(at, r.Ref):
				s.skip(r, code, "lead number claimed by another topic")
				return
			}
			pos, r.Statement, lead, leadK = p, rest, &leadRow{collection: collection, n: n}, k
			if s.leads == nil {
				s.leads = map[string][]int{}
			}
			s.leads[collection] = append(s.leads[collection], n)
		}
	}
	if pos == "" && r.Number == "" && s.Locate != nil {
		if pos = s.Locate(code, r.Year, normalize.Statement(r.Statement, "")); pos != "" {
			s.Located++
		}
	}
	if pos == "" || !validPosition(comp, r.Year, pos) {
		s.skip(r, code, "no valid position")
		return
	}
	c, ok := s.candidate(r, code, pos)
	if !ok {
		return
	}
	letter := ""
	switch comp.EditionForm {
	case "", "year":
	case "year-season":
		// The season of the edition comes only from the title that the numbered source gives the row or its topic, as described at SeasonPosition.
		if letter = topicLetter; !seasonLetterRE.MatchString(letter) {
			s.skip(r, code, "no season")
			return
		}
	case "year-letter":
		part, l, ok := Paper(comp, r.Year, r.Labels)
		letter = l
		// A row with no letter, such as a row of the AoPS collection "2009 AMC 10" that holds both papers, takes the letter that the AoPS wiki gives its topic. A row of a collection that names a season, such as "2021 AMC 10 Fall", does not, because that year had four papers.
		if letter == "" && !seasonRE.MatchString(strings.Join(r.Labels, " ")) {
			letter = topicLetter
		}
		// A paper of a part of the year, such as the fall papers of 2021, has the edition 2021-fall-a. A row whose season is not a part of the year has no paper.
		if !ok {
			s.skip(r, code, "no paper of the season")
			return
		}
		if part != "" && letter != "" {
			letter = "-" + part + "-" + letter
		}
		if letter != "" {
			if s.lettered == nil {
				s.lettered = map[string]map[int]bool{}
			}
			if s.lettered[code] == nil {
				s.lettered[code] = map[int]bool{}
			}
			s.lettered[code][r.Year] = true
		}
	default:
		s.skip(r, code, "edition form not supported")
		return
	}
	s.pend(r, c, comp, pos, letter, lead, leadK)
}

// candidate makes the candidate of a row at the position pos, or gives false when the source of the row or its text is not usable.
func (s *Seeds) candidate(r seed.Row, code, pos string) (*Candidate, bool) {
	src := s.Corpus.Source(r.Source)
	if src == nil || src.Origin == "M" || src.DoNotRepublish {
		s.skip(r, code, "source not usable")
		return nil, false
	}
	// A text in zone Z3 is a link only (spec 11.2), so the import does not take it.
	if a := s.Corpus.Source(r.Author); src.Zone.Statement == "Z3" || (a != nil && a.Zone.Statement == "Z3") {
		s.skip(r, code, "text in zone Z3")
		return nil, false
	}
	text := normalize.Statement(r.Statement, pos)
	if utf8.RuneCountInString(text) < MinText || titleRE.MatchString(text) || isRef(text) || (len(strings.Fields(text)) < MinWords && !strings.Contains(text, "$")) || !strings.ContainsAny(text, "$.?!:") || (strings.Contains(text, ";") && !strings.Contains(text, "$") && len(strings.Fields(text)) < 30) {
		s.skip(r, code, "not a statement")
		return nil, false
	}
	if s.Accept != nil && !s.Accept(text) {
		s.skip(r, code, "does not render")
		return nil, false
	}
	key := match.WeakKey
	if src.Origin == "H1" || src.Origin == "H2" {
		key = match.StrongKey
	}
	url := src.URL
	if strings.HasPrefix(r.Ref, "https://") {
		url = r.Ref
	}
	lang := r.Lang
	if lang == "" {
		lang = "en"
	}
	return &Candidate{
		Source: r.Source, Origin: src.Origin, Zone: src.Zone.Statement, Lang: lang,
		Name: src.Name, URL: url, Ref: r.File + "#" + strconv.Itoa(r.Row),
		Text: text, Answer: Answer(r.Answer), Key: key, Reworded: src.Reworded,
	}, true
}

// pend keeps the candidate of a row for Candidates.
func (s *Seeds) pend(r seed.Row, c *Candidate, comp *content.Competition, pos, letter string, lead *leadRow, leadK int) {
	code := comp.Code
	if len(comp.Shape(r.Year).Positions()) == 0 && pos[0] != 'p' {
		if s.sections == nil {
			s.sections = map[string]map[int]bool{}
		}
		if s.sections[code] == nil {
			s.sections[code] = map[int]bool{}
		}
		s.sections[code][r.Year] = true
	}
	s.pending = append(s.pending, pendingRow{row: r, c: c, comp: comp, year: r.Year, letter: letter, pos: pos, lead: lead, k: leadK, one: onePaper(comp, r.Year)})
}

// addMatched reads a row that the content match placed at the appearance at. The edition of at gives the year and, for a competition with seasons, the season, as 2024-fall.
func (s *Seeds) addMatched(r seed.Row, at string) {
	parts := strings.Split(at, "/")
	if len(parts) != 3 || len(parts[1]) < 4 {
		s.skip(r, "", "no valid position")
		return
	}
	code, pos := parts[0], parts[2]
	comp := s.Corpus.Competition(code)
	year, err := strconv.Atoi(parts[1][:4])
	if comp == nil || err != nil {
		s.skip(r, code, "no competition")
		return
	}
	letter := parts[1][4:]
	switch comp.EditionForm {
	case "", "year":
		if letter != "" {
			s.skip(r, code, "no valid position")
			return
		}
	case "year-season":
		if !seasonLetterRE.MatchString(letter) {
			s.skip(r, code, "no season")
			return
		}
	default:
		s.skip(r, code, "edition form not supported")
		return
	}
	r.Year = year
	if !validPosition(comp, year, pos) {
		s.skip(r, code, "no valid position")
		return
	}
	r.Statement = CleanHead(r.Statement)
	c, ok := s.candidate(r, code, pos)
	if !ok {
		return
	}
	s.Matches++
	// The row takes the appearance of a record, so it does not mark its year as one with sections: a second day such as d113 sits next to p101 in the same year.
	s.pending = append(s.pending, pendingRow{row: r, c: c, comp: comp, year: r.Year, letter: letter, pos: pos, one: onePaper(comp, r.Year), matched: true})
}

// Candidates returns the candidates of each appearance. A position such as p3 is dropped when another row of the same year has a position with a section letter, such as c3, because the two numberings do not agree. A row that the content match placed is not dropped, because its position is the appearance of a record. For a competition with two papers each year, a row with no letter is dropped when a row of the same year or of an earlier year has a letter, because the paper of the row is not known.
func (s *Seeds) Candidates() map[string][]*Candidate {
	first := map[string]int{}
	for code, ys := range s.lettered {
		for y := range ys {
			if f, ok := first[code]; !ok || y < f {
				first[code] = y
			}
		}
	}
	out := map[string][]*Candidate{}
	for _, p := range s.pending {
		if p.lead != nil {
			if !leadSure(s.leads[p.lead.collection], s.collections[p.lead.collection], p.k, p.one) {
				s.skip(p.row, p.comp.Code, "lead number not sure")
				continue
			}
			s.Led++
		}
		if p.pos[0] == 'p' && !p.matched && s.sections[p.comp.Code][p.year] {
			s.skip(p.row, p.comp.Code, "numbered position in a year with sections")
			continue
		}
		ed := strconv.Itoa(p.year)
		if p.comp.EditionForm == "year-letter" {
			if f, ok := first[p.comp.Code]; p.letter == "" && ok && p.year >= f {
				s.skip(p.row, p.comp.Code, "no edition letter")
				continue
			}
			ed += p.letter
		}
		if p.comp.EditionForm == "year-season" {
			ed += p.letter
		}
		p.c.At = p.comp.Code + "/" + ed + "/" + p.pos
		if s.Trace != nil {
			s.Trace(p.row, p.comp.Code, p.c.At, "")
		}
		out[p.c.At] = append(out[p.c.At], p.c)
	}
	return out
}

func validPosition(c *content.Competition, year int, pos string) bool {
	if all := c.Shape(year).Positions(); len(all) > 0 {
		return slices.Contains(all, pos)
	}
	return positionRE.MatchString(pos)
}

// placeholders are answer values that some datasets give to a problem with no answer.
var placeholders = []string{"", "proof", "not found", "none", "n/a", "na", "null", "-", "see solution", "no answer"}

// Answer cleans the answer of a seed row. A placeholder, such as "proof" for a proof problem, gives "".
func Answer(s string) string {
	s = strings.TrimSpace(s)
	if slices.Contains(placeholders, strings.ToLower(strings.Trim(s, ".$ "))) {
		return ""
	}
	return s
}

// TypeOf gives the problem type from the competition and the answer, or "" when it is not known.
func TypeOf(comp, answer string) string {
	switch comp {
	case "amc8", "amc10", "amc12", "ahsme":
		return "multiple-choice"
	case "aime":
		return "answer"
	}
	if answer != "" {
		return "answer"
	}
	return ""
}
