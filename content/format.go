package content

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// keyOrder gives the order of the keys of each kind of front matter, and of the records inside it. A key that is not in the list goes after the known keys, in its old order. The loader reports such a key.
var keyOrder = map[string][]string{
	"problem":    {"tag", "kind", "appearances", "title", "type", "topics", "answer", "authors", "canonical", "statements", "xref", "links", "status"},
	"solution":   {"tag", "kind", "problem", "origin", "author", "source", "url", "capture", "zone", "lang", "method", "complete", "answer_claimed", "run", "translated_by"},
	"appearance": {"at", "primary", "note"},
	"statement":  {"id", "lang", "source", "name", "url", "page", "capture", "origin", "zone", "kind", "diff_from"},
	"canonical":  {"statement", "rule"},
	"author":     {"name", "person", "display", "role", "country"},
	"run":        {"model", "prompt", "date", "run_id", "transcript", "cost"},
	"link":       {"type", "to"},
}

// children names the order list of the records under a key.
var children = map[string]string{
	"appearances":   "appearance",
	"statements":    "statement",
	"canonical":     "canonical",
	"authors":       "author",
	"author":        "author",
	"run":           "run",
	"translated_by": "run",
	"links":         "link",
}

// Format returns a content file in its canonical form: the keys of the front matter in a fixed order, an indent of two spaces, no space at the end of a line and one newline at the end of the file. The text of the body does not change in any other way.
func Format(b []byte) ([]byte, error) {
	front, body, err := splitFront(b)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return nil, fmt.Errorf("YAML: %w", err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("the front matter is not a mapping")
	}
	top := doc.Content[0]
	kind := "problem"
	if v := value(top, "kind"); v != nil && v.Value == "solution" {
		kind = "solution"
	}
	order(top, kind)
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(top); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	lines := strings.Split(strings.TrimRight(body, "\n\t "), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t")
	}
	text := strings.TrimLeft(strings.Join(lines, "\n"), "\n")
	if text != "" {
		buf.WriteString(text)
		buf.WriteString("\n")
	}
	return buf.Bytes(), nil
}

func value(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// order sorts the keys of the mapping m with the order list of kind, and then the records under it.
func order(m *yaml.Node, kind string) {
	keys := keyOrder[kind]
	type pair struct{ k, v *yaml.Node }
	pairs := make([]pair, 0, len(m.Content)/2)
	for i := 0; i+1 < len(m.Content); i += 2 {
		pairs = append(pairs, pair{m.Content[i], m.Content[i+1]})
	}
	rank := func(k string) int {
		if i := slices.Index(keys, k); i >= 0 {
			return i
		}
		return len(keys)
	}
	slices.SortStableFunc(pairs, func(a, b pair) int { return rank(a.k.Value) - rank(b.k.Value) })
	m.Content = m.Content[:0]
	for _, p := range pairs {
		m.Content = append(m.Content, p.k, p.v)
		child, ok := children[p.k.Value]
		if !ok {
			continue
		}
		switch p.v.Kind {
		case yaml.MappingNode:
			order(p.v, child)
		case yaml.SequenceNode:
			for _, item := range p.v.Content {
				if item.Kind == yaml.MappingNode {
					order(item, child)
				}
			}
		}
	}
}
