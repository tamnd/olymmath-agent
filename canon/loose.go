package canon

import (
	"bytes"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
)

type outLoose struct {
	Tag       string `yaml:"tag"`
	Kind      string `yaml:"kind"`
	Type      string `yaml:"type,omitempty"`
	Answer    string `yaml:"answer,omitempty"`
	Canonical struct {
		Statement string `yaml:"statement"`
		Rule      string `yaml:"rule"`
	} `yaml:"canonical"`
	Statements []outStatement     `yaml:"statements"`
	Status     map[string]string  `yaml:"status"`
	Unplaced   []content.Unplaced `yaml:"unplaced,omitempty"`
}

// RenderLoose writes the file of a new loose problem (spec 04.3) from its candidates: the first is the canonical statement, and each other candidate of another source is a statement too. A candidate keeps its text when the text is not the same as the canonical text (signal T below match.Same), up to MaxTexts texts. typ is the problem type, or "" when it is not known.
func RenderLoose(tag, typ, answer string, cands []*Candidate, unplaced []content.Unplaced) ([]byte, error) {
	if len(cands) == 0 {
		return nil, fmt.Errorf("a loose problem needs a candidate")
	}
	best := cands[0]
	p := outLoose{Tag: tag, Kind: "problem", Type: typ, Answer: answer, Status: map[string]string{"statement": "captured"}, Unplaced: unplaced}
	p.Canonical.Statement = "S1"
	_, p.Canonical.Rule = Rank(best, "")
	p.Statements = append(p.Statements, statementOf("S1", best))
	var body strings.Builder
	body.WriteString(best.Text)
	body.WriteString("\n")
	texts := 1
	seen := map[string]bool{best.Source: true}
	for _, c := range cands[1:] {
		if seen[c.Source] {
			continue
		}
		seen[c.Source] = true
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

// AddUnplaced adds the unplaced entries us to an existing problem file. An entry with the same competition or edition and the same source as an entry of the file is left out. The body does not change.
func AddUnplaced(file []byte, us []content.Unplaced) ([]byte, error) {
	if len(us) == 0 {
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
		if m.Content[i].Value == "unplaced" {
			list = m.Content[i+1]
		}
	}
	have := map[string]bool{}
	if list == nil {
		list = &yaml.Node{Kind: yaml.SequenceNode}
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "unplaced"}, list)
	} else {
		var old []content.Unplaced
		if err := list.Decode(&old); err != nil {
			return nil, err
		}
		for _, u := range old {
			have[u.At+"\x00"+u.Source] = true
		}
	}
	for _, u := range us {
		if have[u.At+"\x00"+u.Source] {
			continue
		}
		have[u.At+"\x00"+u.Source] = true
		var n yaml.Node
		if err := n.Encode(u); err != nil {
			return nil, err
		}
		list.Content = append(list.Content, &n)
	}
	return writeFront(&doc, body)
}
