// Package content loads the content repository of olymmath: the registers, the editions, the problem files and the solution files (spec 04).
//
// registers/competitions.yaml lists the competitions and registers/sources.yaml lists the sources. editions/<competition>/<edition>.yaml describes one edition and its papers. A problem file has a tag, its appearances, its statements and its facts in a YAML front matter, and the statements in Markdown with TeX in the body. The file of a problem is at the locator of its primary appearance, for example problems/imo/2024/p1.md, or at problems/_loose/<shard>/<tag>.md when the problem has no known appearance. A solution file is at solutions/<locator>/<tag>.md.
package content

import (
	"sort"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/locator"
)

// Competition is one row of the competition register (spec 05.2).
type Competition struct {
	Code              string            `yaml:"code"`
	Name              string            `yaml:"name"`
	Short             string            `yaml:"short"`
	Names             map[string]string `yaml:"names"`
	Group             string            `yaml:"group"`
	Level             string            `yaml:"level"`
	Country           string            `yaml:"country"`
	Organizer         string            `yaml:"organizer"`
	URL               string            `yaml:"url"`
	FirstYear         int               `yaml:"first_year"`
	LastYear          int               `yaml:"last_year"`
	EditionForm       string            `yaml:"edition_form"`
	Layout            []Layout          `yaml:"layout"`
	PerYear           int               `yaml:"per_year"`
	OfficialSolutions *bool             `yaml:"official_solutions"`
	Languages         []string          `yaml:"languages"`
	Copyright         string            `yaml:"copyright"`
	Sources           []string          `yaml:"sources"`
	Description       string            `yaml:"description"`

	Editions []*Edition `yaml:"-"`
}

// Layout is one usual paper of an edition of a competition, with its positions.
type Layout struct {
	Paper     string   `yaml:"paper"`
	Name      string   `yaml:"name"`
	Positions []string `yaml:"positions"`
}

// Groups are the groups of the competition register, in the order of the site.
var Groups = []string{"international", "undergraduate", "usa", "national", "journal"}

// Levels are the levels of the competition register.
var Levels = []string{"junior", "high-school", "undergraduate", "open"}

// SourceRecord is one row of the source register (spec 05.3).
type SourceRecord struct {
	ID         string `yaml:"id"`
	Name       string `yaml:"name"`
	Kind       string `yaml:"kind"`
	Origin     string `yaml:"origin"`
	URL        string `yaml:"url"`
	Access     string `yaml:"access"`
	CrawlDelay int    `yaml:"crawl_delay"`
	Terms      string `yaml:"terms"`
	Licence    string `yaml:"licence"`
	Zone       struct {
		Statement string `yaml:"statement"`
		Solution  string `yaml:"solution"`
	} `yaml:"zone"`
	AITrain string `yaml:"ai_train"`
	// DoNotRepublish means that the holder asked that no text of the source is published. Rule X02 checks it.
	DoNotRepublish bool   `yaml:"do_not_republish"`
	Size           string `yaml:"size"`
	Wave           int    `yaml:"wave"`
	State          string `yaml:"state"`
	Notes          string `yaml:"notes"`
}

// Edition is one year or one sitting of a competition.
type Edition struct {
	Year     int     `yaml:"year"`
	Number   int     `yaml:"number"`
	Place    string  `yaml:"place"`
	Country  string  `yaml:"country"`
	Dates    string  `yaml:"dates"`
	URL      string  `yaml:"url"`
	Papers   []Paper `yaml:"papers"`
	Language string  `yaml:"language"`

	// ID is the edition part of the locator: the name of the file, for example 2024 or 2021a.
	ID          string       `yaml:"-"`
	Path        string       `yaml:"-"`
	Competition *Competition `yaml:"-"`
	Problems    []*Problem   `yaml:"-"`
}

// Locator returns the locator of the edition, for example imo/2024.
func (e *Edition) Locator() string { return e.Competition.Code + "/" + e.ID }

// Positions returns each position of the papers of the edition, in order.
func (e *Edition) Positions() []string {
	var ps []string
	for _, p := range e.Papers {
		ps = append(ps, p.Problems...)
	}
	return ps
}

// Paper is one sitting of an edition, for example day 1 of the IMO.
type Paper struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Date     string   `yaml:"date"`
	Time     string   `yaml:"time"`
	Problems []string `yaml:"problems"`
}

// Source says where a text comes from. The site shows it under the text.
type Source struct {
	Name   string
	URL    string
	Page   int
	Origin string
	Zone   string
}

// Appearance is the fact that a problem was set in a paper at a position (spec 04.1).
type Appearance struct {
	At      string `yaml:"at"`
	Primary bool   `yaml:"primary"`
	Note    string `yaml:"note"`

	Locator locator.Locator `yaml:"-"`
}

// Statement is one text of a problem, in one language, from one source.
type Statement struct {
	ID       string `yaml:"id"`
	Lang     string `yaml:"lang"`
	Source   string `yaml:"source"`
	Name     string `yaml:"name"`
	URL      string `yaml:"url"`
	Page     int    `yaml:"page"`
	Capture  string `yaml:"capture"`
	Kind     string `yaml:"kind"`
	Origin   string `yaml:"origin"`
	Zone     string `yaml:"zone"`
	DiffFrom string `yaml:"diff_from"`

	// Text is the text of the statement from the body of the file. It is empty when the statement is only a diff or a link.
	Text string `yaml:"-"`
}

// Canonical names the canonical statement and the rule that chose it (spec 06.3).
type Canonical struct {
	Statement string `yaml:"statement"`
	Rule      string `yaml:"rule"`
}

// Rules are the precedence rules for the canonical statement, best first (spec 06.3).
var Rules = []string{"official-original", "official-translation", "curated", "community", "derived"}

// Origins are the origin classes of a text (spec 06.2). M is model-written text.
var Origins = []string{"H1", "H2", "H3", "D", "M"}

// Zones are the licence zones (spec 11.2).
var Zones = []string{"Z1", "Z2", "Z3"}

// Author is a person who has a role for a problem, for example the proposer.
type Author struct {
	Name    string `yaml:"name"`
	Person  string `yaml:"person"`
	Role    string `yaml:"role"`
	Country string `yaml:"country"`
}

// Link is a typed edge from a problem to a record (spec 04.7).
type Link struct {
	Type string `yaml:"type"`
	To   string `yaml:"to"`
}

// LinkTypes are the types of the edges of spec 04.7.
var LinkTypes = []string{"variant_of", "generalizes", "special_case_of", "uses", "same_idea", "answers_at", "formalized_as"}

// Problem is one problem file.
type Problem struct {
	Tag         string            `yaml:"tag"`
	Kind        string            `yaml:"kind"`
	Appearances []Appearance      `yaml:"appearances"`
	Title       string            `yaml:"title"`
	Type        string            `yaml:"type"`
	Topics      []string          `yaml:"topics"`
	Answer      string            `yaml:"answer"`
	Authors     []Author          `yaml:"authors"`
	Canonical   Canonical         `yaml:"canonical"`
	Statements  []Statement       `yaml:"statements"`
	Xref        map[string]string `yaml:"xref"`
	Links       []Link            `yaml:"links"`

	// Statement is the canonical statement: Markdown with TeX math.
	Statement string `yaml:"-"`
	// Path is the path of the file, relative to the root of the content.
	Path string `yaml:"-"`
	// Competition, Year and Position come from the primary appearance. They are empty for a loose problem.
	Competition string `yaml:"-"`
	Year        int    `yaml:"-"`
	Position    string `yaml:"-"`
	// Source is the source of the canonical statement.
	Source    Source      `yaml:"-"`
	Edition   *Edition    `yaml:"-"`
	Paper     *Paper      `yaml:"-"`
	Solutions []*Solution `yaml:"-"`
}

// Types are the problem types of spec 04.3.
var Types = []string{"proof", "answer", "answer+proof", "construction", "multiple-choice"}

// Primary returns the primary appearance, or nil for a loose problem.
func (p *Problem) Primary() *Appearance {
	for i := range p.Appearances {
		if p.Appearances[i].Primary {
			return &p.Appearances[i]
		}
	}
	if len(p.Appearances) == 1 {
		return &p.Appearances[0]
	}
	return nil
}

// CanonicalStatement returns the canonical statement, or nil.
func (p *Problem) CanonicalStatement() *Statement {
	for i := range p.Statements {
		if p.Statements[i].ID == p.Canonical.Statement {
			return &p.Statements[i]
		}
	}
	return nil
}

// Proposer returns the name of the proposer, or an empty string.
func (p *Problem) Proposer() string {
	for _, a := range p.Authors {
		if a.Role == "proposer" {
			return a.Name
		}
	}
	return ""
}

// Number returns the number of the position, for example 3 for p3, or 0 when the position has no number.
func (p *Problem) Number() int {
	n, err := strconv.Atoi(strings.TrimLeft(p.Position, "abcdefghijklmnopqrstuvwxyz"))
	if err != nil {
		return 0
	}
	return n
}

// Locator returns the locator of the primary appearance, for example imo/2024/p1, or the tag for a loose problem.
func (p *Problem) Locator() string {
	if a := p.Primary(); a != nil {
		return a.Locator.String()
	}
	return p.Tag
}

// Solution is one solution file (spec 04.4).
type Solution struct {
	Tag     string `yaml:"tag"`
	Kind    string `yaml:"kind"`
	Problem string `yaml:"problem"`
	Origin  string `yaml:"origin"`
	Author  struct {
		Person  string `yaml:"person"`
		Display string `yaml:"display"`
	} `yaml:"author"`
	Source   string   `yaml:"source"`
	URL      string   `yaml:"url"`
	Capture  string   `yaml:"capture"`
	Zone     string   `yaml:"zone"`
	Lang     string   `yaml:"lang"`
	Method   []string `yaml:"method"`
	Complete bool     `yaml:"complete"`
	Answer   string   `yaml:"answer_claimed"`
	Run      *Run     `yaml:"run"`
	// TranslatedBy names the model that translated a human solution.
	TranslatedBy *Run `yaml:"translated_by"`

	Body string `yaml:"-"`
	Path string `yaml:"-"`
}

// Run records how a model made a text (spec 07).
type Run struct {
	Model      string `yaml:"model"`
	Prompt     string `yaml:"prompt"`
	Date       string `yaml:"date"`
	RunID      string `yaml:"run_id"`
	Transcript string `yaml:"transcript"`
	Cost       string `yaml:"cost"`
}

// SolutionOrigins are the origins of a solution, in the order of the site (spec 07.1).
var SolutionOrigins = []string{"official", "human", "human+ai", "ai"}

// Corpus is all the content.
type Corpus struct {
	Competitions []*Competition
	Sources      []*SourceRecord
	// Loose are the problems with no known appearance.
	Loose []*Problem
	// Redirects maps an old locator to its new locator (registers/redirects.tsv).
	Redirects map[string]string
	// Aliases maps an old tag to the tag that replaced it (registers/aliases.tsv).
	Aliases map[string]string
	// Tags maps each tag that was ever given to its kind (registers/tags.tsv).
	Tags map[string]string

	byCode   map[string]*Competition
	bySource map[string]*SourceRecord
	byTag    map[string]*Problem
	all      []*Problem
}

func newCorpus() *Corpus {
	return &Corpus{
		Redirects: map[string]string{},
		Aliases:   map[string]string{},
		Tags:      map[string]string{},
		byCode:    map[string]*Competition{},
		bySource:  map[string]*SourceRecord{},
		byTag:     map[string]*Problem{},
	}
}

// Competition returns the competition with the code, or nil.
func (c *Corpus) Competition(code string) *Competition { return c.byCode[code] }

// Source returns the source with the id, or nil.
func (c *Corpus) Source(id string) *SourceRecord { return c.bySource[id] }

// ByTag returns the problem with the tag, or nil.
func (c *Corpus) ByTag(t string) *Problem { return c.byTag[t] }

// All returns every problem file that loaded, with the loose problems, in the order of their paths.
func (c *Corpus) All() []*Problem { return c.all }

// Problems returns the problems with an appearance, in the order of competition, edition and position.
func (c *Corpus) Problems() []*Problem {
	var ps []*Problem
	for _, comp := range c.Competitions {
		for _, e := range comp.Editions {
			ps = append(ps, e.Problems...)
		}
	}
	return ps
}

// Editions returns all the editions, the newest first.
func (c *Corpus) Editions() []*Edition {
	var es []*Edition
	for _, comp := range c.Competitions {
		es = append(es, comp.Editions...)
	}
	sort.SliceStable(es, func(i, j int) bool { return es[i].ID > es[j].ID })
	return es
}

// Finding is an error or a warning in one file.
type Finding struct {
	Path    string
	Rule    string
	Level   string
	Message string
}

// The levels of a finding.
const (
	Error   = "error"
	Warning = "warning"
	Info    = "info"
)

func (f Finding) String() string {
	s := f.Path + ": "
	if f.Rule != "" {
		s += f.Rule + " "
	}
	if f.Level != "" && f.Level != Error {
		s += f.Level + ": "
	}
	return s + f.Message
}

// Findings is a list of findings.
type Findings []Finding

func (fs Findings) Error() string {
	lines := make([]string, len(fs))
	for i, f := range fs {
		lines[i] = f.String()
	}
	return strings.Join(lines, "\n")
}

// Errors returns the number of findings of level error.
func (fs Findings) Errors() int {
	n := 0
	for _, f := range fs {
		if f.Level == "" || f.Level == Error {
			n++
		}
	}
	return n
}

// Topics are the four areas of olympiad mathematics.
var Topics = map[string]string{
	"A": "Algebra",
	"C": "Combinatorics",
	"G": "Geometry",
	"N": "Number theory",
}

// TopicRoot returns the root of a topic, for example N for N.diophantine.pell.
func TopicRoot(t string) string {
	root, _, _ := strings.Cut(t, ".")
	return root
}

func in(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Valid reports whether s is in the list. It is for the rules of the audit.
func Valid(list []string, s string) bool { return in(list, s) }
