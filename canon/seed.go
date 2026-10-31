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
	positionRE = regexp.MustCompile(`^[a-z][0-9]{1,2}$`)
	// titleRE matches a text that names a problem instead of stating it.
	titleRE = regexp.MustCompile(`(?i)^[\w .'-]*\b(19|20)[0-9]{2}\b[\w .'-]*(/|#|problem\s*)[a-z]?[0-9]{1,2}\b[\w ./#-]*$`)
)

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

// Seeds makes candidates from seed rows. It accepts a row when the alias table resolves its labels to a competition of comps (all competitions when comps is empty), when its year and its position are valid for the competition, and when its text is a statement. The result maps each appearance to its candidates.
type Seeds struct {
	Corpus  *content.Corpus
	Aliases *seed.Aliases
	Comps   []string
	// ThisYear is the last valid year.
	ThisYear int
	// Accept, when it is not nil, can refuse a text, for example a text with a formula that does not render.
	Accept func(text string) bool

	pending []pendingRow
	// lettered[code][year] is true when a row of the year has an edition letter.
	lettered map[string]map[int]bool
	// sections[code][year] is true when a row of the year has a position with a section letter, such as c3 of the shortlist.
	sections map[string]map[int]bool
	// Skipped counts the rows that were not accepted, by reason.
	Skipped map[string]int
}

type pendingRow struct {
	c      *Candidate
	comp   *content.Competition
	year   int
	letter string
	pos    string
}

func (s *Seeds) skip(reason string) {
	if s.Skipped == nil {
		s.Skipped = map[string]int{}
	}
	s.Skipped[reason]++
}

// Add reads one seed row.
func (s *Seeds) Add(r seed.Row) {
	code := s.Aliases.Code(r.Labels)
	if code == "" || code == "-" {
		s.skip("no competition")
		return
	}
	if len(s.Comps) > 0 && !slices.Contains(s.Comps, code) {
		s.skip("other competition")
		return
	}
	comp := s.Corpus.Competition(code)
	if comp == nil {
		s.skip("no competition")
		return
	}
	if r.Year == 0 || (comp.FirstYear > 0 && r.Year < comp.FirstYear) || (s.ThisYear > 0 && r.Year > s.ThisYear) {
		s.skip("no valid year")
		return
	}
	pos := seed.Position(r.Number)
	if pos == "" || !validPosition(comp, pos) {
		s.skip("no valid position")
		return
	}
	src := s.Corpus.Source(r.Source)
	if src == nil || src.Origin == "M" || src.DoNotRepublish {
		s.skip("source not usable")
		return
	}
	text := normalize.Statement(r.Statement, pos)
	if utf8.RuneCountInString(text) < MinText || titleRE.MatchString(text) || (len(strings.Fields(text)) < MinWords && !strings.Contains(text, "$")) || !strings.ContainsAny(text, "$.?!:") || (strings.Contains(text, ";") && !strings.Contains(text, "$") && len(strings.Fields(text)) < 30) {
		s.skip("not a statement")
		return
	}
	if s.Accept != nil && !s.Accept(text) {
		s.skip("does not render")
		return
	}
	letter := ""
	switch comp.EditionForm {
	case "", "year":
	case "year-letter":
		letter = Letter(r.Labels)
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
		s.skip("edition form not supported")
		return
	}
	if len(comp.Layout) == 0 && comp.PerYear == 0 && pos[0] != 'p' {
		if s.sections == nil {
			s.sections = map[string]map[int]bool{}
		}
		if s.sections[code] == nil {
			s.sections[code] = map[int]bool{}
		}
		s.sections[code][r.Year] = true
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
	c := &Candidate{
		Source: r.Source, Origin: src.Origin, Zone: src.Zone.Statement, Lang: lang,
		Name: src.Name, URL: url, Ref: r.File + "#" + strconv.Itoa(r.Row),
		Text: text, Answer: strings.TrimSpace(r.Answer), Key: key,
	}
	s.pending = append(s.pending, pendingRow{c: c, comp: comp, year: r.Year, letter: letter, pos: pos})
}

// Candidates returns the candidates of each appearance. A position such as p3 is dropped when another row of the same year has a position with a section letter, such as c3, because the two numberings do not agree. For a competition with two papers each year, a row with no letter is dropped when a row of the same year or of an earlier year has a letter, because the paper of the row is not known.
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
		if p.pos[0] == 'p' && s.sections[p.comp.Code][p.year] {
			s.skip("numbered position in a year with sections")
			continue
		}
		ed := strconv.Itoa(p.year)
		if p.comp.EditionForm == "year-letter" {
			if f, ok := first[p.comp.Code]; p.letter == "" && ok && p.year >= f {
				s.skip("no edition letter")
				continue
			}
			ed += p.letter
		}
		p.c.At = p.comp.Code + "/" + ed + "/" + p.pos
		out[p.c.At] = append(out[p.c.At], p.c)
	}
	return out
}

func validPosition(c *content.Competition, pos string) bool {
	var all []string
	for _, l := range c.Layout {
		all = append(all, l.Positions...)
	}
	if len(all) == 0 && c.PerYear > 0 {
		for i := 1; i <= c.PerYear; i++ {
			all = append(all, "p"+strconv.Itoa(i))
		}
	}
	if len(all) > 0 {
		return slices.Contains(all, pos)
	}
	return positionRE.MatchString(pos)
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
