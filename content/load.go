package content

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/locator"
	"github.com/tamnd/olymmath-agent/tag"
)

// Load reads the content at root. It returns the corpus and the findings of the files that it cannot read. The corpus is usable when there are findings, but it does not have the files with errors. The audit package checks the rules of spec 09.3 on the corpus.
func Load(root string) (*Corpus, Findings) {
	l := &loader{root: root, corpus: newCorpus()}
	l.sources()
	l.competitions()
	l.editions()
	l.registers()
	l.problems()
	l.solutions()
	l.link()
	return l.corpus, l.findings
}

type loader struct {
	root     string
	corpus   *Corpus
	findings Findings
}

func (l *loader) add(path, rule, format string, args ...any) {
	l.findings = append(l.findings, Finding{Path: path, Rule: rule, Level: Error, Message: fmt.Sprintf(format, args...)})
}

func (l *loader) read(rel string) ([]byte, bool) {
	b, err := os.ReadFile(filepath.Join(l.root, rel)) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		l.add(rel, "", "cannot read the file: %v", err)
		return nil, false
	}
	return b, true
}

func (l *loader) exists(rel string) bool {
	_, err := os.Stat(filepath.Join(l.root, rel))
	return err == nil
}

func (l *loader) sources() {
	const rel = "registers/sources.yaml"
	if !l.exists(rel) {
		return
	}
	b, ok := l.read(rel)
	if !ok {
		return
	}
	var ss []*SourceRecord
	if err := strictYAML(b, &ss); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	for _, s := range ss {
		switch {
		case !locator.ValidCode(s.ID):
			l.add(rel, "", "the source id %q is not valid. Use lower case letters, digits and hyphens", s.ID)
		case l.corpus.bySource[s.ID] != nil:
			l.add(rel, "", "the source id %q is in the register two times", s.ID)
		default:
			l.corpus.bySource[s.ID] = s
			l.corpus.Sources = append(l.corpus.Sources, s)
		}
	}
}

func (l *loader) competitions() {
	const rel = "registers/competitions.yaml"
	b, ok := l.read(rel)
	if !ok {
		return
	}
	var cs []*Competition
	if err := strictYAML(b, &cs); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	for _, c := range cs {
		switch {
		case !locator.ValidCode(c.Code):
			l.add(rel, "", "the code %q is not valid. Use lower case letters, digits and hyphens", c.Code)
		case l.corpus.byCode[c.Code] != nil:
			l.add(rel, "", "the code %q is in the register two times", c.Code)
		case c.Name == "":
			l.add(rel, "", "the competition %q has no name", c.Code)
		default:
			if msg := shapeError(c); msg != "" {
				l.add(rel, "", "the competition %q: %s", c.Code, msg)
			}
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
			l.add(dir, "", "%v", err)
			continue
		}
		for _, en := range entries {
			name := en.Name()
			if en.IsDir() || !strings.HasSuffix(name, ".yaml") {
				continue
			}
			rel := filepath.ToSlash(filepath.Join(dir, name))
			b, ok := l.read(rel)
			if !ok {
				continue
			}
			e := &Edition{ID: strings.TrimSuffix(name, ".yaml"), Path: rel}
			if err := strictYAML(b, e); err != nil {
				l.add(rel, "", "%v", err)
				continue
			}
			if !locator.ValidEdition(e.ID) || !strings.HasPrefix(e.ID, strconv.Itoa(e.Year)) {
				l.add(rel, "", "the name of the file must be the year %d, or the year and a letter or a word", e.Year)
				continue
			}
			for i := range e.Papers {
				if e.Papers[i].ID == "" {
					e.Papers[i].ID = strconv.Itoa(i + 1)
				}
			}
			e.Competition = c
			c.Editions = append(c.Editions, e)
		}
		sort.Slice(c.Editions, func(i, j int) bool { return c.Editions[i].ID > c.Editions[j].ID })
	}
}

// registers reads the tab-separated registers: tags.tsv, aliases.tsv and redirects.tsv. A line that starts with # is a comment.
func (l *loader) registers() {
	l.tsv("registers/tags.tsv", 2, func(f []string) { l.corpus.Tags[f[0]] = f[1] })
	l.tsv("registers/aliases.tsv", 2, func(f []string) { l.corpus.Aliases[f[0]] = f[1] })
	l.tsv("registers/redirects.tsv", 2, func(f []string) { l.corpus.Redirects[f[0]] = f[1] })
}

func (l *loader) tsv(rel string, minCols int, row func([]string)) {
	if !l.exists(rel) {
		return
	}
	b, ok := l.read(rel)
	if !ok {
		return
	}
	sc := bufio.NewScanner(bytes.NewReader(b))
	n := 0
	for sc.Scan() {
		n++
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < minCols {
			l.add(rel, "", "line %d has %d fields, want %d or more, separated by tabs", n, len(f), minCols)
			continue
		}
		row(f)
	}
}

func (l *loader) walk(dir string, file func(rel string)) {
	err := filepath.WalkDir(filepath.Join(l.root, dir), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".md") || d.Name() == "README.md" {
			return nil
		}
		rel, _ := filepath.Rel(l.root, path)
		file(filepath.ToSlash(rel))
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		l.add(dir, "", "%v", err)
	}
}

func (l *loader) problems() {
	l.walk("problems", l.problem)
}

var markerRE = regexp.MustCompile(`(?m)^<!--\s*(S[0-9]+)\b[^>]*-->[ \t]*$`)

// SplitStatements cuts the body into the texts of the statements. The text before the first marker belongs to the canonical statement.
func SplitStatements(body, canonical string) map[string]string {
	texts := map[string]string{}
	locs := markerRE.FindAllStringSubmatchIndex(body, -1)
	start, id := 0, canonical
	for _, m := range locs {
		if t := strings.TrimSpace(body[start:m[0]]); t != "" {
			texts[id] = t
		}
		id = body[m[2]:m[3]]
		start = m[1]
	}
	if t := strings.TrimSpace(body[start:]); t != "" {
		texts[id] = t
	}
	return texts
}

func (l *loader) problem(rel string) {
	b, ok := l.read(rel)
	if !ok {
		return
	}
	front, body, err := splitFront(b)
	if err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	p := &Problem{Path: rel}
	if err := strictYAML(front, p); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	if !tag.IsTag(p.Tag) {
		l.add(rel, "T01", "the tag %q is not valid", p.Tag)
		return
	}
	if q := l.corpus.byTag[p.Tag]; q != nil {
		l.add(rel, "T01", "the tag %s is also the tag of %s", p.Tag, q.Path)
		return
	}
	if p.Kind != "problem" {
		l.add(rel, "", "the kind is %q, want problem", p.Kind)
		return
	}
	for i := range p.Appearances {
		a := &p.Appearances[i]
		loc, err := locator.Parse(a.At)
		if err != nil || loc.Kind() != locator.Appearance {
			l.add(rel, "T03", "the appearance %q is not the locator of an appearance", a.At)
			return
		}
		a.Locator = loc
	}
	want := tag.LoosePath(p.Tag)
	if a := p.Primary(); a != nil {
		want = a.Locator.Path()
		p.Competition = a.Locator.Comp
		p.Year, _ = strconv.Atoi(a.Locator.Edition[:4])
		p.Position = a.Locator.Position
	} else if len(p.Appearances) > 1 {
		l.add(rel, "T04", "the problem has %d appearances and none is primary", len(p.Appearances))
		return
	}
	if rel != want {
		l.add(rel, "T03", "the path must be %s", want)
		return
	}
	texts := SplitStatements(body, p.Canonical.Statement)
	for i := range p.Statements {
		s := &p.Statements[i]
		s.Text = texts[s.ID]
		delete(texts, s.ID)
	}
	for id := range texts {
		l.add(rel, "S01", "the body has a text for %s, but the front matter has no statement %s", id, id)
	}
	if s := p.CanonicalStatement(); s != nil {
		p.Statement = s.Text
		p.Source = l.source(s)
		if s.Traced != "" {
			p.Source.Origin, p.Source.Traced = p.Class(s), true
		}
		if s.Kind != "official" {
			for i := range p.Statements {
				if o := &p.Statements[i]; o.Kind == "official" && o.URL != "" {
					src := l.source(o)
					p.Official = &src
					break
				}
			}
		}
	}
	l.corpus.byTag[p.Tag] = p
	l.corpus.all = append(l.corpus.all, p)
	if p.Primary() == nil {
		l.corpus.Loose = append(l.corpus.Loose, p)
	}
}

// source gives the source of a statement for the site. The register fills the fields that the statement does not give.
func (l *loader) source(s *Statement) Source {
	src := Source{Name: s.Name, URL: s.URL, Page: s.Page, Origin: s.Origin, Zone: s.Zone}
	if r := l.corpus.bySource[s.Source]; r != nil {
		if src.Name == "" {
			src.Name = r.Name
		}
		if src.URL == "" {
			src.URL = r.URL
		}
		if src.Origin == "" {
			src.Origin = r.Origin
		}
		if src.Zone == "" {
			src.Zone = r.Zone.Statement
		}
	}
	return src
}

func (l *loader) solutions() {
	l.walk("solutions", l.solution)
}

func (l *loader) solution(rel string) {
	b, ok := l.read(rel)
	if !ok {
		return
	}
	front, body, err := splitFront(b)
	if err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	s := &Solution{Path: rel, Body: strings.TrimSpace(body)}
	if err := strictYAML(front, s); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	if !tag.IsTag(s.Tag) || s.Kind != "solution" {
		l.add(rel, "T01", "a solution needs a valid tag and the kind solution")
		return
	}
	p := l.corpus.byTag[s.Problem]
	if p == nil {
		l.add(rel, "L01", "the problem %q does not exist", s.Problem)
		return
	}
	if want := "solutions/" + strings.TrimSuffix(strings.TrimPrefix(p.Path, "problems/"), ".md") + "/" + s.Tag + ".md"; rel != want {
		l.add(rel, "T03", "the path must be %s", want)
		return
	}
	p.Solutions = append(p.Solutions, s)
}

// link attaches each problem to its edition, puts the problems of each edition in the order of the papers, and orders the solutions.
func (l *loader) link() {
	for _, p := range l.corpus.all {
		a := p.Primary()
		if a == nil {
			continue
		}
		c := l.corpus.byCode[a.Locator.Comp]
		if c == nil {
			l.add(p.Path, "T03", "the competition %q is not in the register", a.Locator.Comp)
			continue
		}
		found := false
		for _, e := range c.Editions {
			if e.ID == a.Locator.Edition {
				p.Edition = e
				e.Problems = append(e.Problems, p)
				found = true
				break
			}
		}
		if !found {
			l.add(p.Path, "T03", "the edition %s has no file in editions/", a.Locator.EditionOf())
		}
		for _, o := range p.Appearances {
			if o.Primary {
				continue
			}
			if e := l.corpus.edition(o.Locator.Comp, o.Locator.Edition); e != nil {
				e.Also = append(e.Also, p)
			}
		}
	}
	for _, c := range l.corpus.Competitions {
		for _, e := range c.Editions {
			order := map[string]int{}
			for i, pos := range e.Positions() {
				order[pos] = i + 1
			}
			sort.Slice(e.Problems, func(i, j int) bool {
				a, b := e.Problems[i], e.Problems[j]
				if order[a.Position] != order[b.Position] {
					return order[a.Position] < order[b.Position]
				}
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
	rank := map[string]int{}
	for i, o := range SolutionOrigins {
		rank[o] = i
	}
	for _, p := range l.corpus.all {
		sort.SliceStable(p.Solutions, func(i, j int) bool { return rank[p.Solutions[i].Origin] < rank[p.Solutions[j].Origin] })
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
		if bytes.HasSuffix(rest, []byte("\n---")) {
			return rest[:len(rest)-4], "", nil
		}
		return nil, "", errors.New("the front matter has no end (---)")
	}
	return rest[:end], string(rest[end+5:]), nil
}

// SplitFront cuts a file into its front matter and its body.
func SplitFront(b []byte) (front []byte, body string, err error) { return splitFront(b) }

func strictYAML(b []byte, v any) error {
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(v); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("YAML: %w", err)
	}
	return nil
}

// shapeError gives the first error of the layout, the eras and the overrides of a competition, or "".
func shapeError(c *Competition) string {
	if msg := layoutError(c.Layout, c.PerYear); msg != "" {
		return msg
	}
	for i, e := range c.Eras {
		switch {
		case e.From <= 0:
			return fmt.Sprintf("era %d has no from year", i+1)
		case e.To != 0 && e.To < e.From:
			return fmt.Sprintf("era %d ends before it starts", i+1)
		}
		if msg := layoutError(e.Layout, e.PerYear); msg != "" {
			return fmt.Sprintf("era %d: %s", i+1, msg)
		}
		for j, o := range c.Eras[:i] {
			if (o.To == 0 || e.From <= o.To) && (e.To == 0 || o.From <= e.To) {
				return fmt.Sprintf("era %d overlaps era %d", i+1, j+1)
			}
		}
	}
	for y, ps := range c.Overrides {
		if y <= 0 {
			return fmt.Sprintf("the override year %d is not valid", y)
		}
		if len(ps) == 0 {
			return fmt.Sprintf("the override of %d has no positions", y)
		}
		if msg := layoutError([]Layout{{Positions: ps}}, 0); msg != "" {
			return fmt.Sprintf("the override of %d: %s", y, msg)
		}
	}
	return ""
}

// layoutError gives the first error of a layout, or "". A position must be valid and must not be in the layout two times.
func layoutError(ls []Layout, perYear int) string {
	if perYear < 0 {
		return fmt.Sprintf("per_year %d is negative", perYear)
	}
	seen := map[string]bool{}
	for _, l := range ls {
		for _, p := range l.Positions {
			if _, err := locator.Parse("x/2000/" + p); err != nil {
				return fmt.Sprintf("the position %q is not valid", p)
			}
			if seen[p] {
				return fmt.Sprintf("the position %q is in the layout two times", p)
			}
			seen[p] = true
		}
	}
	return ""
}
