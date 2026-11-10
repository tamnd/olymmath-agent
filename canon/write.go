package canon

import (
	"bytes"
	"fmt"
	"math"
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
	// Extraction and Confidence are set for a text that a model read from a scan.
	Extraction string  `yaml:"extraction,omitempty"`
	Confidence float64 `yaml:"confidence,omitempty"`
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
	switch {
	case c.Origin == "H1":
		kind = "official"
	case c.Reworded:
		kind = "rewording"
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
		cur, _ = Rank(&Candidate{Origin: p.Class(s), Lang: s.Lang}, original)
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
		// A reworded text never replaces a canonical statement (spec 06.3).
		if rk, _ := Rank(c, original); rk >= 0 && rk < cur && a.Better == nil && !c.Reworded {
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
	return writeFront(&doc, body)
}

// AddAppearances adds the positions ats as appearances of an existing problem file that is not primary in any of them, for example a position of another olympiad that used the problem. A position that the problem has already is left out. The body and the statements do not change.
func AddAppearances(file []byte, ats ...string) ([]byte, error) {
	front, body, err := content.SplitFront(file)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return nil, err
	}
	list := child(doc.Content[0], "appearances")
	if list == nil || list.Kind != yaml.SequenceNode {
		return nil, fmt.Errorf("the problem has no appearances")
	}
	have := map[string]bool{}
	for _, a := range list.Content {
		have[scalar(a, "at")] = true
	}
	for _, at := range ats {
		if have[at] {
			continue
		}
		have[at] = true
		var n yaml.Node
		if err := n.Encode(outAppearance{At: at}); err != nil {
			return nil, err
		}
		list.Content = append(list.Content, &n)
	}
	return writeFront(&doc, body)
}

// writeFront gives a problem file with the front matter doc and the body.
func writeFront(doc *yaml.Node, body string) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(doc); err != nil {
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

// Fill gives a text to an existing problem that has none, such as a problem with only a link to a page whose terms do not allow a copy of the text. The statement id, which Apply has added, becomes the canonical statement with the rule and the status of the statement, and text becomes the body. The other statements stay as links. It fails when the problem has a body already, because the canonical text of a problem never changes by import.
func Fill(file []byte, id, text, rule, status string) ([]byte, error) {
	front, body, err := content.SplitFront(file)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(body) != "" {
		return nil, fmt.Errorf("the problem has a text already")
	}
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("the text is empty")
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
	canonical, status0 := field(m, "canonical"), field(m, "status")
	if field(canonical, "statement") == nil || field(canonical, "rule") == nil || field(status0, "statement") == nil {
		return nil, fmt.Errorf("the problem has no canonical statement, rule or status")
	}
	found := false
	for _, st := range field(m, "statements").Content {
		if f := field(st, "id"); f != nil && f.Value == id {
			found = true
		}
	}
	if !found {
		return nil, fmt.Errorf("the problem has no statement %s", id)
	}
	field(canonical, "statement").Value = id
	field(canonical, "rule").Value = rule
	field(status0, "statement").Value = status
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
	buf.WriteString(strings.TrimSpace(text) + "\n")
	return content.Format(buf.Bytes())
}

// Trace marks the statement id of an existing problem file as traced to the official statement official, and makes it the canonical statement with the rule. The text of id moves to the top of the body, and the other texts keep their markers.
func Trace(file []byte, id, official, rule string) ([]byte, error) {
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
	canonical, status, list := field(m, "canonical"), field(m, "status"), field(m, "statements")
	if field(canonical, "statement") == nil || field(canonical, "rule") == nil || field(status, "statement") == nil || list == nil {
		return nil, fmt.Errorf("the problem has no canonical statement, rule, status or statements")
	}
	var target *yaml.Node
	var ids []string
	found := false
	for _, s := range list.Content {
		sid := field(s, "id")
		if sid == nil {
			continue
		}
		ids = append(ids, sid.Value)
		switch sid.Value {
		case id:
			target = s
		case official:
			found = true
		}
	}
	if target == nil || !found || id == official {
		return nil, fmt.Errorf("the problem has no statements %s and %s", id, official)
	}
	if t := field(target, "traced"); t != nil {
		t.Value = official
	} else {
		target.Content = append(target.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "traced"}, &yaml.Node{Kind: yaml.ScalarNode, Value: official})
	}
	old := field(canonical, "statement").Value
	texts := content.SplitStatements(body, old)
	if texts[id] == "" {
		return nil, fmt.Errorf("the statement %s has no text", id)
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
	buf.WriteString(texts[id] + "\n")
	for _, sid := range ids {
		if sid != id && texts[sid] != "" {
			buf.WriteString("\n<!-- " + sid + " -->\n" + texts[sid] + "\n")
		}
	}
	return content.Format(buf.Bytes())
}

// RenderLink writes the file of a new problem whose only statement is a link with no text to page of the official paper of c. rule is the rule that makes the link canonical. A problem such as this comes from a paper whose text the content may not hold. The problem appears at at, its primary appearance, and at each locator of also.
func RenderLink(at, tag string, c *Candidate, page int, rule string, also ...string) ([]byte, error) {
	p := outProblem{Tag: tag, Kind: "problem"}
	p.Appearances = []outAppearance{{At: at, Primary: true}}
	for _, a := range also {
		p.Appearances = append(p.Appearances, outAppearance{At: a})
	}
	p.Canonical.Statement = "S1"
	p.Canonical.Rule = rule
	p.Status.Statement = "sourced"
	s := statementOf("S1", c)
	s.Page = page
	p.Statements = []outStatement{s}
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
	return content.Format(buf.Bytes())
}

// Scan is the text of a problem that a model read from a page image of an official paper.
type Scan struct {
	Page int
	// Text is the Markdown text with TeX math. It is "" when the content may not hold the text of the source: the file then only links to the page.
	Text string
	// Extraction is <lane>/<tool>@<version>, as in vision/claude@2.1.318.
	Extraction string
	// Confidence is from 0 to 1.
	Confidence float64
}

// RenderScan writes the file of a new problem whose only statement is the official paper of c at the page of s. When s has a text, the file holds it, with the extraction and the confidence, and the status is captured: a model read the words and no person has checked them (spec 09.1). When s has no text, the file is the same as the file of RenderLink.
func RenderScan(at, tag string, c *Candidate, s Scan, rule string) ([]byte, error) {
	if strings.TrimSpace(s.Text) == "" {
		return RenderLink(at, tag, c, s.Page, rule)
	}
	if s.Extraction == "" {
		return nil, fmt.Errorf("%s: a scanned text needs its extraction", at)
	}
	p := outProblem{Tag: tag, Kind: "problem"}
	p.Appearances = []outAppearance{{At: at, Primary: true}}
	p.Canonical.Statement = "S1"
	p.Canonical.Rule = rule
	p.Status.Statement = "captured"
	st := statementOf("S1", c)
	st.Page = s.Page
	st.Extraction = s.Extraction
	st.Confidence = math.Round(s.Confidence*100) / 100
	p.Statements = []outStatement{st}
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
	buf.WriteString(strings.TrimSpace(s.Text) + "\n")
	return content.Format(buf.Bytes())
}
