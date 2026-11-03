package site

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// API gives the read records of the corpus (spec 10.8). The site build writes them as static JSON files under /api/, and om serve and the MCP server answer with the same records. The records hold the Markdown of each text, as the machine copies of the pages do.
type API struct {
	corpus *content.Corpus
	url    string
}

// NewAPI returns the records of the corpus. url is the public URL of the site, with no slash at the end. The records link to the pages under it.
func NewAPI(corpus *content.Corpus, url string) *API {
	return &API{corpus: corpus, url: strings.TrimSuffix(url, "/")}
}

// SourceRef is the source of a text in a record.
type SourceRef struct {
	Name   string `json:"name"`
	URL    string `json:"url,omitempty"`
	Page   int    `json:"page,omitempty"`
	Origin string `json:"origin,omitempty"`
	Zone   string `json:"zone,omitempty"`
}

// SolutionRef is one solution in a record. Text is empty when the terms of the source allow only a link.
type SolutionRef struct {
	Origin string `json:"origin"`
	Author string `json:"author,omitempty"`
	Source string `json:"source,omitempty"`
	URL    string `json:"url,omitempty"`
	Zone   string `json:"zone,omitempty"`
	Text   string `json:"text,omitempty"`
}

// ProblemRecord is a problem with no answer and no solution, so that a practice tool can hide them. HasAnswer and Solutions tell what SolutionsRecord holds.
type ProblemRecord struct {
	Tag         string     `json:"tag"`
	Locator     string     `json:"locator,omitempty"`
	URL         string     `json:"url"`
	Title       string     `json:"title"`
	Competition string     `json:"competition,omitempty"`
	Edition     string     `json:"edition,omitempty"`
	Year        int        `json:"year,omitempty"`
	Position    string     `json:"position,omitempty"`
	Type        string     `json:"type,omitempty"`
	Topics      []string   `json:"topics,omitempty"`
	Band        int        `json:"band,omitempty"`
	Statement   string     `json:"statement"`
	Source      SourceRef  `json:"source"`
	Official    *SourceRef `json:"official,omitempty"`
	Status      string     `json:"status,omitempty"`
	// Also are the locators of the other appearances of the problem.
	Also      []string `json:"also,omitempty"`
	HasAnswer bool     `json:"has_answer,omitempty"`
	Solutions int      `json:"solutions"`
}

// SolutionsRecord is the answer and the solutions of a problem.
type SolutionsRecord struct {
	Tag       string        `json:"tag"`
	Answer    string        `json:"answer,omitempty"`
	Solutions []SolutionRef `json:"solutions"`
}

// EditionProblem is one problem in an edition record. Primary is false for a problem whose page is in another edition.
type EditionProblem struct {
	Position string `json:"position"`
	Tag      string `json:"tag"`
	Title    string `json:"title"`
	URL      string `json:"url"`
	Primary  bool   `json:"primary"`
}

// EditionPaper is one paper of an edition record, with the positions of its problems.
type EditionPaper struct {
	ID       string   `json:"id"`
	Name     string   `json:"name,omitempty"`
	Date     string   `json:"date,omitempty"`
	Problems []string `json:"problems"`
}

// EditionRecord is an edition with its papers and its problems, in the order of the positions.
type EditionRecord struct {
	Competition string           `json:"competition"`
	Edition     string           `json:"edition"`
	Title       string           `json:"title"`
	Year        int              `json:"year"`
	URL         string           `json:"url"`
	Papers      []EditionPaper   `json:"papers"`
	Problems    []EditionProblem `json:"problems"`
}

// IndexEdition is one edition in the index.
type IndexEdition struct {
	Edition  string `json:"edition"`
	Year     int    `json:"year"`
	Problems int    `json:"problems"`
}

// IndexCompetition is one competition in the index, with its editions, the newest first.
type IndexCompetition struct {
	Code     string         `json:"code"`
	Name     string         `json:"name"`
	Short    string         `json:"short"`
	Group    string         `json:"group,omitempty"`
	Country  string         `json:"country,omitempty"`
	Editions []IndexEdition `json:"editions"`
}

// IndexRecord lists the competitions and the editions that have records, and the paths of the records.
type IndexRecord struct {
	Problems     int                `json:"problems"`
	Paths        map[string]string  `json:"paths"`
	Competitions []IndexCompetition `json:"competitions"`
}

// Lookup returns the problem with the tag. An old tag of registers/aliases.tsv gives the problem that replaced it.
func (a *API) Lookup(tag string) *content.Problem {
	if p := a.corpus.ByTag(tag); p != nil {
		return p
	}
	if to, ok := a.corpus.Aliases[tag]; ok && to != content.Withdrawn {
		return a.corpus.ByTag(to)
	}
	return nil
}

func sourceRef(s content.Source) SourceRef {
	return SourceRef{Name: s.Name, URL: s.URL, Page: s.Page, Origin: s.Origin, Zone: s.Zone}
}

// Problem gives the record of a problem.
func (a *API) Problem(p *content.Problem) ProblemRecord {
	r := ProblemRecord{
		Tag: p.Tag, Locator: p.Locator(), URL: a.url + problemPath(p), Title: problemTitle(p),
		Competition: p.Competition, Year: p.Year, Position: p.Position, Type: p.Type,
		Topics: p.Topics, Band: p.Band(), Statement: p.Statement, Source: sourceRef(p.Source),
		Status: p.Status.Statement, HasAnswer: p.Answer != "", Solutions: len(p.Solutions),
	}
	if p.Edition != nil {
		r.Edition = p.Edition.ID
	}
	if p.Official != nil {
		o := sourceRef(*p.Official)
		r.Official = &o
	}
	for _, ap := range p.Appearances {
		if !ap.Primary {
			r.Also = append(r.Also, ap.Locator.String())
		}
	}
	return r
}

// Solutions gives the answer and the solutions of a problem.
func (a *API) Solutions(p *content.Problem) SolutionsRecord {
	r := SolutionsRecord{Tag: p.Tag, Answer: p.Answer, Solutions: []SolutionRef{}}
	for _, s := range p.Solutions {
		name := s.Source
		if src := a.corpus.Source(s.Source); src != nil {
			name = src.Name
		}
		r.Solutions = append(r.Solutions, SolutionRef{Origin: s.Origin, Author: s.Author.Display, Source: name, URL: s.URL, Zone: s.Zone, Text: s.Body})
	}
	return r
}

// Edition returns the edition of a competition, or nil.
func (a *API) Edition(comp, id string) *content.Edition {
	c := a.corpus.Competition(comp)
	if c == nil {
		return nil
	}
	for _, e := range c.Editions {
		if e.ID == id {
			return e
		}
	}
	return nil
}

// EditionRecord gives the record of an edition.
func (a *API) EditionRecord(e *content.Edition) EditionRecord {
	c := e.Competition
	r := EditionRecord{Competition: c.Code, Edition: e.ID, Title: c.Short + " " + e.ID, Year: e.Year, URL: a.url + "/" + e.Locator() + "/", Papers: []EditionPaper{}, Problems: []EditionProblem{}}
	for _, paper := range e.Papers {
		r.Papers = append(r.Papers, EditionPaper{ID: paper.ID, Name: paper.Name, Date: paper.Date, Problems: append([]string{}, paper.Problems...)})
	}
	seen := map[string]bool{}
	for _, p := range e.Problems {
		seen[p.Position] = true
		r.Problems = append(r.Problems, EditionProblem{Position: p.Position, Tag: p.Tag, Title: problemTitle(p), URL: a.url + problemPath(p), Primary: true})
	}
	for _, p := range e.Also {
		pos := p.PositionIn(e)
		if seen[pos] {
			continue
		}
		seen[pos] = true
		r.Problems = append(r.Problems, EditionProblem{Position: pos, Tag: p.Tag, Title: appearanceTitle(c, e.ID, pos), URL: a.url + problemPath(p)})
	}
	sort.SliceStable(r.Problems, func(i, j int) bool { return comparePositions(r.Problems[i].Position, r.Problems[j].Position) < 0 })
	return r
}

// Index gives the index of the records.
func (a *API) Index() IndexRecord {
	r := IndexRecord{
		Problems: len(a.corpus.All()),
		Paths: map[string]string{
			"problem":   "/api/problem/<tag>.json",
			"solutions": "/api/solutions/<tag>.json",
			"edition":   "/api/edition/<competition>/<edition>.json",
		},
		Competitions: []IndexCompetition{},
	}
	for _, c := range a.corpus.Competitions {
		if len(c.Editions) == 0 {
			continue
		}
		ic := IndexCompetition{Code: c.Code, Name: c.Name, Short: c.Short, Group: c.Group, Country: c.Country}
		for _, e := range c.Editions {
			ic.Editions = append(ic.Editions, IndexEdition{Edition: e.ID, Year: e.Year, Problems: len(e.Problems)})
		}
		sort.SliceStable(ic.Editions, func(i, j int) bool { return ic.Editions[i].Edition > ic.Editions[j].Edition })
		r.Competitions = append(r.Competitions, ic)
	}
	return r
}

// Marshal gives the JSON of a record, with no indent and a newline at the end. The static files and om serve use it, so they give the same bytes.
func Marshal(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// HasSolutions tells if the problem has an answer or a solution. Only such a problem has a solutions file.
func HasSolutions(p *content.Problem) bool { return p.Answer != "" || len(p.Solutions) > 0 }

// api writes the records under /api/: the index, one file for each problem, one solutions file for each problem with an answer or a solution, and one file for each edition.
func (b *builder) api() error {
	write := func(rel string, v any) error {
		data, err := Marshal(v)
		if err != nil {
			return err
		}
		return b.write(rel, data)
	}
	if err := write("api/index.json", b.records.Index()); err != nil {
		return err
	}
	for _, p := range b.corpus.All() {
		if err := write("api/problem/"+p.Tag+".json", b.records.Problem(p)); err != nil {
			return err
		}
		if HasSolutions(p) {
			if err := write("api/solutions/"+p.Tag+".json", b.records.Solutions(p)); err != nil {
				return err
			}
		}
	}
	for _, c := range b.corpus.Competitions {
		for _, e := range c.Editions {
			if err := write("api/edition/"+e.Locator()+".json", b.records.EditionRecord(e)); err != nil {
				return err
			}
		}
	}
	return nil
}
