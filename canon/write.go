package canon

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
)

// MaxTexts is the largest number of statement texts in one new problem file. The canonical text counts. A candidate whose text is the same as the canonical text (signal T at or above match.Same) is kept as a link with no text.
const MaxTexts = 3

type outAppearance struct {
	At      string `yaml:"at"`
	Primary bool   `yaml:"primary,omitempty"`
	Note    string `yaml:"note,omitempty"`
}

type outStatement struct {
	ID     string `yaml:"id"`
	Lang   string `yaml:"lang"`
	Source string `yaml:"source"`
	Name   string `yaml:"name,omitempty"`
	URL    string `yaml:"url,omitempty"`
	Page   int    `yaml:"page,omitempty"`
	Origin string `yaml:"origin"`
	Zone   string `yaml:"zone"`
	Kind   string `yaml:"kind"`
}

type outProblem struct {
	Tag         string          `yaml:"tag"`
	Kind        string          `yaml:"kind"`
	Appearances []outAppearance `yaml:"appearances"`
	Type        string          `yaml:"type,omitempty"`
	Answer      string          `yaml:"answer,omitempty"`
	Canonical   struct {
		Statement string `yaml:"statement"`
		Rule      string `yaml:"rule"`
	} `yaml:"canonical"`
	Statements []outStatement `yaml:"statements"`
	Status     struct {
		Statement string `yaml:"statement"`
	} `yaml:"status"`
}

func statementOf(id string, c *Candidate) outStatement {
	kind := "transcription"
	if c.Origin == "H1" {
		kind = "official"
	}
	lang := c.Lang
	if lang == "" {
		lang = "en"
	}
	return outStatement{ID: id, Lang: lang, Source: c.Source, Name: c.Name, URL: c.URL, Origin: c.Origin, Zone: c.Zone, Kind: kind}
}

// Render writes the file of a new problem. typ is the problem type, or "" when it is not known.
func Render(r *Record, tag, typ string) ([]byte, error) {
	p := outProblem{Tag: tag, Kind: "problem", Type: typ, Answer: r.Primary.Answer}
	p.Appearances = append(p.Appearances, outAppearance{At: r.Primary.At, Primary: len(r.Merged) > 0})
	if len(r.Merged) == 0 {
		p.Appearances[0].Primary = true
	}
	for _, m := range r.Merged {
		p.Appearances = append(p.Appearances, outAppearance{At: m.At})
	}
	best := r.Primary.Best
	p.Canonical.Statement = "S1"
	p.Canonical.Rule = r.Primary.Rule
	p.Status.Statement = r.Primary.Status()
	p.Statements = append(p.Statements, statementOf("S1", best))
	var body strings.Builder
	body.WriteString(best.Text)
	body.WriteString("\n")
	texts := 1
	seen := map[string]bool{best.Source + "\x00" + best.At: true}
	others := append([]*Candidate{}, r.Primary.Kept...)
	for _, m := range r.Merged {
		others = append(others, m.Choice.Best)
		others = append(others, m.Choice.Kept...)
	}
	for _, c := range others {
		k := c.Source + "\x00" + c.At
		if seen[k] {
			continue
		}
		seen[k] = true
		id := fmt.Sprintf("S%d", len(p.Statements)+1)
		p.Statements = append(p.Statements, statementOf(id, c))
		if texts < MaxTexts && c.Text != "" && match.Jaccard(best.Shingles(), c.Shingles()) < match.Same {
			texts++
			fmt.Fprintf(&body, "\n<!-- %s -->\n%s\n", id, c.Text)
		}
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(p); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.WriteString(body.String())
	return content.Format(buf.Bytes())
}

// Additions are the statements that an import adds to an existing problem: one link with no text for each source that the problem does not have yet. The canonical statement of an existing problem never changes by import. When a candidate ranks above the canonical statement, Better is true, and a person decides in review.
type Additions struct {
	Statements []outStatement
	Better     *Candidate
}

// Add gives the additions for an existing problem from the record of its primary appearance.
func Add(p *content.Problem, r *Record, original string) Additions {
	var a Additions
	have := map[string]bool{}
	for _, s := range p.Statements {
		have[s.Source] = true
	}
	cur := 99
	if s := p.CanonicalStatement(); s != nil {
		cur, _ = Rank(&Candidate{Origin: s.Origin, Lang: s.Lang}, original)
	}
	all := append([]*Candidate{r.Primary.Best}, r.Primary.Kept...)
	n := len(p.Statements)
	for _, c := range all {
		if have[c.Source] {
			continue
		}
		have[c.Source] = true
		n++
		a.Statements = append(a.Statements, statementOf(fmt.Sprintf("S%d", n), c))
		if rk, _ := Rank(c, original); rk >= 0 && rk < cur && a.Better == nil {
			a.Better = c
		}
	}
	return a
}

// ID gives the id of the statement of the source of c in a, or "".
func (a Additions) ID(c *Candidate) string {
	for _, s := range a.Statements {
		if s.Source == c.Source {
			return s.ID
		}
	}
	return ""
}

// Agrees tells if the text of c is the problem p: the signal T of the two texts is at least match.Doubt, and the answers do not disagree.
func Agrees(c *Candidate, p *content.Problem) bool {
	if p.Statement == "" || c.Text == "" {
		return false
	}
	return match.Jaccard(c.Shingles(), match.Shingles(p.Statement)) >= match.Doubt && match.Answers(c.Answer, p.Answer) != match.AnswerMismatch
}

// AddLink gives the addition of one link with no text to an existing problem, for example a link to the page of an official paper where the problem is.
func AddLink(p *content.Problem, c *Candidate, page int) Additions {
	s := statementOf(fmt.Sprintf("S%d", len(p.Statements)+1), c)
	s.Page = page
	return Additions{Statements: []outStatement{s}}
}

// Apply adds the statements of a to the front matter of an existing problem file. The body does not change, so each added statement is a link with no text.
func Apply(file []byte, a Additions) ([]byte, error) {
	if len(a.Statements) == 0 {
		return file, nil
	}
	front, body, err := content.SplitFront(file)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return nil, err
	}
	m := doc.Content[0]
	var list *yaml.Node
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == "statements" {
			list = m.Content[i+1]
		}
	}
	if list == nil {
		return nil, fmt.Errorf("the problem has no statements")
	}
	for _, s := range a.Statements {
		var n yaml.Node
		if err := n.Encode(s); err != nil {
			return nil, err
		}
		list.Content = append(list.Content, &n)
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.WriteString(body)
	return content.Format(buf.Bytes())
}

// Promote makes the statement id the canonical statement of an existing problem file, with text as its text. It is for a candidate of class H1 or H2 that ranks above the canonical statement old and whose text agrees with it. The statement id must be in the front matter with no text, for example as a link that Apply added. The text of old stays in the body under its own marker.
func Promote(file []byte, old, id, text, rule string) ([]byte, error) {
	front, body, err := content.SplitFront(file)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return nil, err
	}
	field := func(n *yaml.Node, key string) *yaml.Node {
		for i := 0; n != nil && i+1 < len(n.Content); i += 2 {
			if n.Content[i].Value == key {
				return n.Content[i+1]
			}
		}
		return nil
	}
	m := doc.Content[0]
	canonical, status := field(m, "canonical"), field(m, "status")
	if field(canonical, "statement") == nil || field(canonical, "rule") == nil || field(status, "statement") == nil {
		return nil, fmt.Errorf("the problem has no canonical statement, rule or status")
	}
	if field(canonical, "statement").Value != old || old == id {
		return nil, fmt.Errorf("the canonical statement is %s, not %s", field(canonical, "statement").Value, old)
	}
	if strings.Contains(body, "<!-- "+id+" -->") {
		return nil, fmt.Errorf("the statement %s has a text already", id)
	}
	field(canonical, "statement").Value = id
	field(canonical, "rule").Value = rule
	field(status, "statement").Value = "sourced"
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.WriteString(text + "\n\n<!-- " + old + " -->\n" + strings.TrimLeft(body, "\n"))
	return content.Format(buf.Bytes())
}
