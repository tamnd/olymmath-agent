package canon

import (
	"bytes"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
)

// MergeFiles joins the problem file from into the problem file into, for two records of one problem that two imports made. The primary appearance, the tag and the canonical statement of into stay. The appearances of from join into, and its statements join the statements of into with new ids. A text of from goes into the body when it is not the same as the canonical text (signal T below match.Same) and the body has fewer than MaxTexts texts. The topics join, and the answer of from is kept when into has none. Two answers that disagree are an error, because then the two records are not one problem.
func MergeFiles(into, from []byte) ([]byte, error) {
	iFront, iBody, err := content.SplitFront(into)
	if err != nil {
		return nil, err
	}
	fFront, fBody, err := content.SplitFront(from)
	if err != nil {
		return nil, err
	}
	var iDoc, fDoc yaml.Node
	if err := yaml.Unmarshal(iFront, &iDoc); err != nil {
		return nil, err
	}
	if err := yaml.Unmarshal(fFront, &fDoc); err != nil {
		return nil, err
	}
	im, fm := iDoc.Content[0], fDoc.Content[0]
	if a, b := scalar(im, "answer"), scalar(fm, "answer"); a != "" && b != "" && match.Answers(a, b) == match.AnswerMismatch {
		return nil, fmt.Errorf("the answers %q and %q disagree", a, b)
	}

	apps := child(im, "appearances")
	if apps == nil || len(apps.Content) == 0 {
		return nil, fmt.Errorf("the problem has no appearances")
	}
	if !slices.ContainsFunc(apps.Content, func(n *yaml.Node) bool { return scalar(n, "primary") == "true" }) {
		set(apps.Content[0], "primary", "true", yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: "true"})
	}
	if fa := child(fm, "appearances"); fa != nil {
		for _, a := range fa.Content {
			apps.Content = append(apps.Content, without(a, "primary"))
		}
	}

	list := child(im, "statements")
	if list == nil {
		return nil, fmt.Errorf("the problem has no statements")
	}
	ids := map[string]string{}
	if fs := child(fm, "statements"); fs != nil {
		for _, s := range fs.Content {
			id := "S" + strconv.Itoa(len(list.Content)+1)
			ids[scalar(s, "id")] = id
			s = without(s, "")
			set(s, "id", id, yaml.Node{})
			list.Content = append(list.Content, s)
		}
	}

	icanon := scalar(child(im, "canonical"), "statement")
	texts := content.SplitStatements(iBody, icanon)
	best := match.Shingles(texts[icanon])
	ftexts := content.SplitStatements(fBody, scalar(child(fm, "canonical"), "statement"))
	var body strings.Builder
	body.WriteString(strings.TrimRight(iBody, "\n"))
	body.WriteString("\n")
	n := len(texts)
	keys := make([]string, 0, len(ftexts))
	for old := range ftexts {
		if ids[old] != "" {
			keys = append(keys, old)
		}
	}
	slices.SortFunc(keys, func(a, b string) int { return cmpID(ids[a], ids[b]) })
	for _, old := range keys {
		t := ftexts[old]
		if n >= MaxTexts || match.Jaccard(best, match.Shingles(t)) >= match.Same {
			continue
		}
		n++
		fmt.Fprintf(&body, "\n<!-- %s -->\n%s\n", ids[old], t)
	}

	if scalar(im, "answer") == "" && scalar(fm, "answer") != "" {
		copyKey(im, fm, "answer")
		if scalar(im, "type") == "" {
			copyKey(im, fm, "type")
		}
	}
	if ft := child(fm, "topics"); ft != nil {
		it := child(im, "topics")
		if it == nil {
			copyKey(im, fm, "topics")
		} else {
			for _, t := range ft.Content {
				if !slices.ContainsFunc(it.Content, func(n *yaml.Node) bool { return n.Value == t.Value }) {
					it.Content = append(it.Content, t)
				}
			}
		}
	}

	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&iDoc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.WriteString(body.String())
	return content.Format(buf.Bytes())
}

func cmpID(a, b string) int {
	x, _ := strconv.Atoi(strings.TrimPrefix(a, "S"))
	y, _ := strconv.Atoi(strings.TrimPrefix(b, "S"))
	return x - y
}

// child gives the value of the key in the mapping m, or nil.
func child(m *yaml.Node, key string) *yaml.Node {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

func scalar(m *yaml.Node, key string) string {
	if n := child(m, key); n != nil && n.Kind == yaml.ScalarNode {
		return n.Value
	}
	return ""
}

// set gives the key of the mapping m the value v. The node n is the value when it has a kind.
func set(m *yaml.Node, key, v string, n yaml.Node) {
	if n.Kind == 0 {
		n = yaml.Node{Kind: yaml.ScalarNode, Value: v}
	}
	if c := child(m, key); c != nil {
		*c = n
		return
	}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, &n)
}

// without gives a copy of the mapping m with no key named key.
func without(m *yaml.Node, key string) *yaml.Node {
	out := *m
	out.Content = nil
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value != key {
			out.Content = append(out.Content, m.Content[i], m.Content[i+1])
		}
	}
	return &out
}

func copyKey(dst, src *yaml.Node, key string) {
	if v := child(src, key); v != nil {
		dst.Content = append(dst.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
	}
}
