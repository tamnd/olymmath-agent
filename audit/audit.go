// Package audit runs the audit rules of spec 09.3 on the content of olymmath.
//
// The loader of the content package already checks the rules that it needs to build the corpus (T01, T03, T04, S01 and L01). This package adds the rules that need the whole corpus, the registers or the renderer.
package audit

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

// Rule is one audit rule.
type Rule struct {
	ID      string
	Level   string
	Summary string
	// Loader is true when the loader of the content package checks the rule.
	Loader bool
	check  func(*Context)
}

// Context is the input of the rules, and it collects their findings.
type Context struct {
	Root     string
	Corpus   *content.Corpus
	Markup   *site.Markup
	findings content.Findings
	rule     *Rule
}

func (c *Context) add(p, format string, args ...any) {
	c.findings = append(c.findings, content.Finding{Path: p, Rule: c.rule.ID, Level: c.rule.Level, Message: fmt.Sprintf(format, args...)})
}

// Rules are the audit rules, in the order of spec 09.3.
var Rules = []*Rule{
	{ID: "T01", Level: content.Error, Summary: "each tag is valid and unique", Loader: true},
	{ID: "T02", Level: content.Error, Summary: "each tag is in registers/tags.tsv, and each removed tag is in registers/aliases.tsv", check: t02},
	{ID: "T03", Level: content.Error, Summary: "each locator is in the competition register, and each redirect goes to a problem", check: t03},
	{ID: "T04", Level: content.Error, Summary: "each appearance belongs to exactly one problem", check: t04},
	{ID: "S01", Level: content.Error, Summary: "each problem has a canonical statement with a valid rule", check: s01},
	{ID: "S02", Level: content.Error, Summary: "the canonical statement is not model-written", check: s02},
	{ID: "S03", Level: content.Error, Summary: "each statement has a source in the register and a licence zone", check: s03},
	{ID: "S04", Level: content.Error, Summary: "the canonical text and the answer render with KaTeX", check: s04},
	{ID: "S05", Level: content.Warning, Summary: "the canonical text comes from a dataset", check: s05},
	{ID: "S06", Level: content.Warning, Summary: "two statements of a problem differ in a number or an operator", check: s06},
	{ID: "S07", Level: content.Error, Summary: "a statement in zone Z3 has no text in the published tree", check: s07},
	{ID: "S08", Level: content.Warning, Summary: "each figure in a text has a file", check: s08},
	{ID: "L01", Level: content.Error, Summary: "each solution points to a problem", Loader: true},
	{ID: "L02", Level: content.Error, Summary: "each AI solution has a complete run block", check: l02},
	{ID: "L03", Level: content.Error, Summary: "no AI solution says that a person wrote it", check: l03},
	{ID: "L04", Level: content.Error, Summary: "each solution has a zone, and its text fits the zone", check: l04},
	{ID: "X01", Level: content.Error, Summary: "each source in use has a licence zone and a terms note", check: x01},
	{ID: "X02", Level: content.Error, Summary: "no text from a do-not-republish source is in the published tree", check: x02},
}

// Lookup returns the rule with the id, or nil.
func Lookup(id string) *Rule {
	for _, r := range Rules {
		if r.ID == id {
			return r
		}
	}
	return nil
}

// Run loads the content at root and runs the rules. With no ids, it runs all the rules. The findings of the loader with no rule are always in the result, because they show a file that cannot be read.
func Run(root string, ids ...string) (*content.Corpus, content.Findings, error) {
	for _, id := range ids {
		if Lookup(id) == nil {
			return nil, nil, fmt.Errorf("audit: no rule %q", id)
		}
	}
	want := func(id string) bool { return len(ids) == 0 || slices.Contains(ids, id) }
	corpus, loaded := content.Load(root)
	m, err := site.NewMarkup(nil)
	if err != nil {
		return nil, nil, err
	}
	c := &Context{Root: root, Corpus: corpus, Markup: m}
	for _, f := range loaded {
		if f.Rule == "" || want(f.Rule) {
			c.findings = append(c.findings, f)
		}
	}
	for _, r := range Rules {
		if r.check != nil && want(r.ID) {
			c.rule = r
			r.check(c)
		}
	}
	sort.SliceStable(c.findings, func(i, j int) bool { return c.findings[i].Path < c.findings[j].Path })
	return corpus, c.findings, nil
}

func t02(c *Context) {
	used := map[string]bool{}
	for _, p := range c.Corpus.All() {
		used[p.Tag] = true
		if k, ok := c.Corpus.Tags[p.Tag]; !ok {
			c.add(p.Path, "the tag %s is not in registers/tags.tsv", p.Tag)
		} else if k != "problem" {
			c.add(p.Path, "registers/tags.tsv gives the tag %s to a %s", p.Tag, k)
		}
		for _, s := range p.Solutions {
			used[s.Tag] = true
			if k, ok := c.Corpus.Tags[s.Tag]; !ok {
				c.add(s.Path, "the tag %s is not in registers/tags.tsv", s.Tag)
			} else if k != "solution" {
				c.add(s.Path, "registers/tags.tsv gives the tag %s to a %s", s.Tag, k)
			}
		}
	}
	for _, t := range sortedKeys(c.Corpus.Tags) {
		if used[t] {
			continue
		}
		to, ok := c.Corpus.Aliases[t]
		switch {
		case !ok:
			c.add("registers/tags.tsv", "the tag %s has no record and is not in registers/aliases.tsv", t)
		case !used[to]:
			c.add("registers/aliases.tsv", "the tag %s points to %s, which has no record", t, to)
		}
	}
}

func t03(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, a := range p.Appearances {
			if a.Locator.Comp == "" {
				continue
			}
			comp := c.Corpus.Competition(a.Locator.Comp)
			if comp == nil {
				c.add(p.Path, "the competition of %s is not in the register", a.At)
			} else if a.Locator.Edition != "" && comp.FirstYear > 0 && editionYear(a.Locator.Edition) > 0 && editionYear(a.Locator.Edition) < comp.FirstYear {
				c.add(p.Path, "the edition of %s is before the first year %d of %s", a.At, comp.FirstYear, comp.Code)
			}
		}
	}
	located := map[string]bool{}
	for _, p := range c.Corpus.All() {
		for _, a := range p.Appearances {
			located[a.At] = true
		}
	}
	for _, old := range sortedKeys(c.Corpus.Redirects) {
		if to := c.Corpus.Redirects[old]; !located[to] {
			c.add("registers/redirects.tsv", "%s goes to %s, which is not an appearance of a problem", old, to)
		}
		if located[old] {
			c.add("registers/redirects.tsv", "%s is a redirect and also an appearance of a problem", old)
		}
	}
}

// editionYear returns the year at the start of an edition id, for example 2024 for 2024 or 2024-spring, or 0.
func editionYear(id string) int {
	y := 0
	for i := 0; i < len(id) && i < 4; i++ {
		if id[i] < '0' || id[i] > '9' {
			return 0
		}
		y = y*10 + int(id[i]-'0')
	}
	if len(id) < 4 {
		return 0
	}
	return y
}

func t04(c *Context) {
	owner := map[string]string{}
	for _, p := range c.Corpus.All() {
		for _, a := range p.Appearances {
			if prev, ok := owner[a.At]; ok && prev != p.Path {
				c.add(p.Path, "the appearance %s also belongs to %s", a.At, prev)
				continue
			}
			owner[a.At] = p.Path
		}
	}
}

func s01(c *Context) {
	for _, p := range c.Corpus.All() {
		switch {
		case p.CanonicalStatement() == nil:
			c.add(p.Path, "the canonical statement %q is not in the list of statements", p.Canonical.Statement)
		case !slices.Contains(content.Rules, p.Canonical.Rule):
			c.add(p.Path, "the canonical rule %q is not one of %s", p.Canonical.Rule, strings.Join(content.Rules, ", "))
		}
	}
}

func s02(c *Context) {
	for _, p := range c.Corpus.All() {
		if s := p.CanonicalStatement(); s != nil && s.Origin == "M" {
			c.add(p.Path, "the canonical statement %s is model-written", s.ID)
		}
	}
}

func s03(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Statements {
			switch {
			case s.Source == "":
				c.add(p.Path, "the statement %s has no source", s.ID)
			case c.Corpus.Source(s.Source) == nil:
				c.add(p.Path, "the source %q of the statement %s is not in registers/sources.yaml", s.Source, s.ID)
			}
			if !slices.Contains(content.Zones, s.Zone) {
				c.add(p.Path, "the statement %s has the zone %q, not one of Z1, Z2 and Z3", s.ID, s.Zone)
			}
			if !slices.Contains(content.Origins, s.Origin) {
				c.add(p.Path, "the statement %s has the origin %q, not one of %s", s.ID, s.Origin, strings.Join(content.Origins, ", "))
			}
		}
	}
}

func s04(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, text := range []string{p.Statement, p.Answer} {
			_, errs := c.Markup.HTML(text)
			for _, e := range errs {
				c.add(p.Path, "a formula does not render: %v", e)
			}
		}
	}
}

func s05(c *Context) {
	for _, p := range c.Corpus.All() {
		if s := p.CanonicalStatement(); s != nil && s.Origin == "D" {
			c.add(p.Path, "the canonical statement %s comes from a dataset; find the original text", s.ID)
		}
	}
}

// tokens matches the numbers and the operators of a text, which rule S06 compares.
var tokens = regexp.MustCompile(`\d+(?:\.\d+)?|\\(?:leq?|geq?|neq|lt|gt|le|ge)\b|[<>=+\-*/^]`)

func signature(s string) []string {
	t := tokens.FindAllString(s, -1)
	sort.Strings(t)
	return t
}

func s06(c *Context) {
	for _, p := range c.Corpus.All() {
		var base *content.Statement
		for i := range p.Statements {
			s := &p.Statements[i]
			if s.Text == "" || s.Lang != "en" {
				continue
			}
			if base == nil {
				base = s
				continue
			}
			if !slices.Equal(signature(base.Text), signature(s.Text)) {
				c.add(p.Path, "the statements %s and %s differ in a number or an operator; the problem may be disputed", base.ID, s.ID)
			}
		}
	}
}

func s07(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Statements {
			if s.Zone == "Z3" && strings.TrimSpace(s.Text) != "" {
				c.add(p.Path, "the statement %s is in zone Z3, so its text must not be in the published tree", s.ID)
			}
		}
	}
}

// figures matches the images of a Markdown text.
var figures = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)`)

func s08(c *Context) {
	for _, p := range c.Corpus.All() {
		texts := []string{p.Statement, p.Answer}
		for _, s := range p.Statements {
			texts = append(texts, s.Text)
		}
		for _, text := range texts {
			for _, m := range figures.FindAllStringSubmatch(text, -1) {
				ref := m[1]
				if strings.Contains(ref, "://") {
					continue
				}
				var rel string
				if strings.HasPrefix(ref, "/") {
					rel = strings.TrimPrefix(ref, "/")
				} else {
					rel = path.Join(path.Dir(p.Path), ref)
				}
				if _, err := os.Stat(filepath.Join(c.Root, filepath.FromSlash(rel))); err != nil {
					c.add(p.Path, "the figure %s has no file", ref)
				}
			}
		}
	}
}

func l02(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Solutions {
			if s.Origin != "ai" && s.Origin != "human+ai" {
				continue
			}
			if s.Run == nil {
				c.add(s.Path, "the %s solution has no run block", s.Origin)
				continue
			}
			for _, f := range [][2]string{{"model", s.Run.Model}, {"prompt", s.Run.Prompt}, {"date", s.Run.Date}, {"run_id", s.Run.RunID}, {"transcript", s.Run.Transcript}} {
				if f[1] == "" {
					c.add(s.Path, "the run block has no %s", f[0])
				}
			}
		}
	}
}

func l03(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Solutions {
			if (s.Origin == "human" || s.Origin == "official") && s.Run != nil {
				c.add(s.Path, "the solution has a run block, so a model wrote it; its origin cannot be %s", s.Origin)
			}
			if s.TranslatedBy != nil && s.TranslatedBy.Model == "" {
				c.add(s.Path, "the translated_by block does not name the model")
			}
		}
	}
}

func l04(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Solutions {
			body := strings.TrimSpace(s.Body)
			switch s.Zone {
			case "Z1":
			case "Z2":
				if n := strings.Count(body, "\n") + 1; body != "" && n > 2 {
					c.add(s.Path, "the solution is in zone Z2, so the published text is an excerpt of at most two lines, not %d", n)
				}
				if s.URL == "" {
					c.add(s.Path, "the solution is in zone Z2 and has no link to its source")
				}
			case "Z3":
				if body != "" {
					c.add(s.Path, "the solution is in zone Z3, so it is a link only and has no text")
				}
				if s.URL == "" {
					c.add(s.Path, "the solution is in zone Z3 and has no link")
				}
			default:
				c.add(s.Path, "the solution has the zone %q, not one of Z1, Z2 and Z3", s.Zone)
			}
		}
	}
}

// inUse returns the ids of the sources that a statement or a solution names, with the first file that names each.
func inUse(c *Context) map[string]string {
	use := map[string]string{}
	for _, p := range c.Corpus.All() {
		for _, s := range p.Statements {
			if _, ok := use[s.Source]; !ok && s.Source != "" {
				use[s.Source] = p.Path
			}
		}
		for _, s := range p.Solutions {
			if _, ok := use[s.Source]; !ok && s.Source != "" {
				use[s.Source] = s.Path
			}
		}
	}
	return use
}

func x01(c *Context) {
	use := inUse(c)
	for _, id := range sortedKeys(use) {
		src := c.Corpus.Source(id)
		if src == nil {
			continue // S03 reports it.
		}
		if !slices.Contains(content.Zones, src.Zone.Statement) || !slices.Contains(content.Zones, src.Zone.Solution) {
			c.add("registers/sources.yaml", "the source %s is in use (%s) and has no licence zone for statements and solutions", id, use[id])
		}
		if strings.TrimSpace(src.Terms) == "" {
			c.add("registers/sources.yaml", "the source %s is in use (%s) and has no terms note", id, use[id])
		}
	}
}

func x02(c *Context) {
	for _, p := range c.Corpus.All() {
		for _, s := range p.Statements {
			if src := c.Corpus.Source(s.Source); src != nil && src.DoNotRepublish && strings.TrimSpace(s.Text) != "" {
				c.add(p.Path, "the statement %s comes from %s, which asks that its text is not published", s.ID, s.Source)
			}
		}
		for _, s := range p.Solutions {
			if src := c.Corpus.Source(s.Source); src != nil && src.DoNotRepublish && strings.TrimSpace(s.Body) != "" {
				c.add(s.Path, "the solution comes from %s, which asks that its text is not published", s.Source)
			}
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
