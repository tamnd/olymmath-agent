// Package content loads the content repository of olymmath: the competition register, the editions and the problem files.
//
// The layout of milestone M1 is simple. registers/competitions.yaml lists the competitions. editions/<competition>/<year>.yaml describes one edition and its papers. problems/<competition>/<year>/<position>.md is one problem: a YAML front matter and the statement in Markdown with TeX math. Milestone M2 changes the problem files to tags.
package content

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Competition is one row of the competition register.
type Competition struct {
	Code        string `yaml:"code"`
	Name        string `yaml:"name"`
	Short       string `yaml:"short"`
	Level       string `yaml:"level"`
	Country     string `yaml:"country"`
	Organizer   string `yaml:"organizer"`
	URL         string `yaml:"url"`
	FirstYear   int    `yaml:"first_year"`
	Description string `yaml:"description"`

	Editions []*Edition `yaml:"-"`
}

// Edition is one year of a competition.
type Edition struct {
	Year     int     `yaml:"year"`
	Number   int     `yaml:"number"`
	Place    string  `yaml:"place"`
	Country  string  `yaml:"country"`
	Dates    string  `yaml:"dates"`
	URL      string  `yaml:"url"`
	Papers   []Paper `yaml:"papers"`
	Language string  `yaml:"language"`

	Competition *Competition `yaml:"-"`
	Problems    []*Problem   `yaml:"-"`
}

// Paper is one sitting of an edition, for example day 1 of the IMO.
type Paper struct {
	Name     string   `yaml:"name"`
	Date     string   `yaml:"date"`
	Time     string   `yaml:"time"`
	Problems []string `yaml:"problems"`
}

// Source says where a text comes from.
type Source struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Page   int    `yaml:"page"`
	Origin string `yaml:"origin"`
	Zone   string `yaml:"zone"`
}

// Problem is one problem file.
type Problem struct {
	Competition string   `yaml:"competition"`
	Year        int      `yaml:"year"`
	Position    string   `yaml:"position"`
	Answer      string   `yaml:"answer"`
	Topics      []string `yaml:"topics"`
	Type        string   `yaml:"type"`
	Proposer    string   `yaml:"proposer"`
	Source      Source   `yaml:"source"`

	// Statement is the body of the file: Markdown with TeX math.
	Statement string `yaml:"-"`
	// Path is the path of the file, relative to the root of the content.
	Path string `yaml:"-"`

	Edition *Edition `yaml:"-"`
	Paper   *Paper   `yaml:"-"`
}

// Number returns the number of the position, for example 3 for p3, or 0 when the position has no number.
func (p *Problem) Number() int {
	n, err := strconv.Atoi(strings.TrimLeft(p.Position, "pP"))
	if err != nil {
		return 0
	}
	return n
}

// Locator returns the locator of the problem, for example imo/2024/p1.
func (p *Problem) Locator() string {
	return fmt.Sprintf("%s/%d/%s", p.Competition, p.Year, p.Position)
}

// Corpus is all the content.
type Corpus struct {
	Competitions []*Competition
	byCode       map[string]*Competition
}

// Competition returns the competition with the code, or nil.
func (c *Corpus) Competition(code string) *Competition { return c.byCode[code] }

// Problems returns all the problems, in the order of competition, year and position.
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
	sort.SliceStable(es, func(i, j int) bool { return es[i].Year > es[j].Year })
	return es
}

// Finding is an error in one file.
type Finding struct {
	Path    string
	Message string
}

func (f Finding) String() string { return f.Path + ": " + f.Message }

// Findings is the list of all the errors that Load found.
type Findings []Finding

func (fs Findings) Error() string {
	lines := make([]string, len(fs))
	for i, f := range fs {
		lines[i] = f.String()
	}
	return strings.Join(lines, "\n")
}

// Topics are the four areas of olympiad mathematics.
var Topics = map[string]string{
	"A": "Algebra",
	"C": "Combinatorics",
	"G": "Geometry",
	"N": "Number theory",
}

// Load reads the content at root. It returns the corpus and the findings. The corpus is usable when there are findings, but it does not have the files with errors.
func Load(root string) (*Corpus, Findings) {
	l := &loader{root: root, corpus: &Corpus{byCode: map[string]*Competition{}}}
	l.competitions()
	l.editions()
	l.problems()
	l.link()
	return l.corpus, l.findings
}

type loader struct {
	root     string
	corpus   *Corpus
	findings Findings
}

func (l *loader) add(path, format string, args ...any) {
	l.findings = append(l.findings, Finding{Path: path, Message: fmt.Sprintf(format, args...)})
}

func (l *loader) read(rel string) ([]byte, bool) {
	b, err := os.ReadFile(filepath.Join(l.root, rel)) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		l.add(rel, "cannot read the file: %v", err)
		return nil, false
	}
	return b, true
}

func (l *loader) competitions() {
	const rel = "registers/competitions.yaml"
	b, ok := l.read(rel)
	if !ok {
		return
	}
	var cs []*Competition
	if err := strictYAML(b, &cs); err != nil {
		l.add(rel, "%v", err)
		return
	}
	for _, c := range cs {
		switch {
		case !validCode(c.Code):
			l.add(rel, "the code %q is not valid. Use lower case letters, digits and hyphens", c.Code)
		case l.corpus.byCode[c.Code] != nil:
			l.add(rel, "the code %q is in the register two times", c.Code)
		case c.Name == "":
			l.add(rel, "the competition %q has no name", c.Code)
		default:
			if c.Short == "" {
				c.Short = strings.ToUpper(c.Code)
			}
			l.corpus.byCode[c.Code] = c
			l.corpus.Competitions = append(l.corpus.Competitions, c)
		}
	}
}

func (l *loader) editions() {
	for _, c := range l.corpus.Competitions {
		dir := filepath.Join("editions", c.Code)
		entries, err := os.ReadDir(filepath.Join(l.root, dir))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			l.add(dir, "%v", err)
			continue
		}
		for _, en := range entries {
			name := en.Name()
			if en.IsDir() || !strings.HasSuffix(name, ".yaml") {
				continue
			}
			rel := filepath.Join(dir, name)
			b, ok := l.read(rel)
			if !ok {
				continue
			}
			e := &Edition{}
			if err := strictYAML(b, e); err != nil {
				l.add(rel, "%v", err)
				continue
			}
			if strconv.Itoa(e.Year) != strings.TrimSuffix(name, ".yaml") {
				l.add(rel, "the year %d is not the name of the file", e.Year)
				continue
			}
			e.Competition = c
			c.Editions = append(c.Editions, e)
		}
		sort.Slice(c.Editions, func(i, j int) bool { return c.Editions[i].Year > c.Editions[j].Year })
	}
}

func (l *loader) problems() {
	dir := filepath.Join(l.root, "problems")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(l.root, path)
		l.problem(filepath.ToSlash(rel))
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		l.add("problems", "%v", err)
	}
}

func (l *loader) problem(rel string) {
	b, ok := l.read(rel)
	if !ok {
		return
	}
	front, body, err := splitFront(b)
	if err != nil {
		l.add(rel, "%v", err)
		return
	}
	p := &Problem{Path: rel, Statement: strings.TrimSpace(body)}
	if err := strictYAML(front, p); err != nil {
		l.add(rel, "%v", err)
		return
	}
	if want := "problems/" + p.Locator() + ".md"; rel != want {
		l.add(rel, "the front matter gives the locator %s, thus the path must be %s", p.Locator(), want)
		return
	}
	if p.Statement == "" {
		l.add(rel, "the statement is empty")
		return
	}
	for _, t := range p.Topics {
		if Topics[t] == "" {
			l.add(rel, "the topic %q is not one of A, C, G and N", t)
		}
	}
	if p.Source.Name == "" || p.Source.URL == "" {
		l.add(rel, "the source has no name or no URL")
	}
	switch p.Source.Zone {
	case "Z1", "Z2", "Z3":
	default:
		l.add(rel, "the licence zone %q is not one of Z1, Z2 and Z3", p.Source.Zone)
	}
	c := l.corpus.byCode[p.Competition]
	if c == nil {
		l.add(rel, "the competition %q is not in the register", p.Competition)
		return
	}
	for _, e := range c.Editions {
		if e.Year == p.Year {
			p.Edition = e
			e.Problems = append(e.Problems, p)
			return
		}
	}
	l.add(rel, "the edition %s %d has no file in editions/", c.Code, p.Year)
}

// link puts the problems of each edition in the order of the papers, and checks that each position of a paper has a problem.
func (l *loader) link() {
	for _, c := range l.corpus.Competitions {
		for _, e := range c.Editions {
			sort.Slice(e.Problems, func(i, j int) bool {
				a, b := e.Problems[i], e.Problems[j]
				if a.Number() != b.Number() {
					return a.Number() < b.Number()
				}
				return a.Position < b.Position
			})
			byPos := map[string]*Problem{}
			for _, p := range e.Problems {
				byPos[p.Position] = p
			}
			for i := range e.Papers {
				paper := &e.Papers[i]
				for _, pos := range paper.Problems {
					if p := byPos[pos]; p != nil {
						p.Paper = paper
					}
				}
			}
		}
	}
}

func splitFront(b []byte) (front []byte, body string, err error) {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	if !bytes.HasPrefix(b, []byte("---\n")) {
		return nil, "", errors.New("the file does not start with a front matter (---)")
	}
	rest := b[4:]
	end := bytes.Index(rest, []byte("\n---\n"))
	if end < 0 {
		return nil, "", errors.New("the front matter has no end (---)")
	}
	return rest[:end], string(rest[end+5:]), nil
}

func strictYAML(b []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("YAML: %w", err)
	}
	return nil
}

func validCode(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}
