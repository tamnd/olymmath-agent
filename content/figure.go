package content

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// figureRef matches a Markdown image and gives its target, as in ![The figure](p1.fig1.svg).
var figureRef = regexp.MustCompile(`!\[[^\]]*\]\(([^)\s]+)`)

// figureName is the name of a figure file next to a problem (spec 4.3): the base name of the problem, .fig, a number and the format, as p1.fig1.svg.
var figureName = regexp.MustCompile(`^[^/]+\.fig[1-9][0-9]*\.(svg|png)$`)

// FigureRefs gives the target of each Markdown image of text, in the order of the text. It keeps the targets with a scheme, such as https://, so a caller can tell them apart.
func FigureRefs(text string) []string {
	var refs []string
	for _, m := range figureRef.FindAllStringSubmatch(text, -1) {
		refs = append(refs, m[1])
	}
	return refs
}

// IsFigureName tells if name is the name of a figure file next to a problem, such as p1.fig1.svg.
func IsFigureName(name string) bool { return figureName.MatchString(name) }

// FigurePath gives the path from the root of the content of the figure ref in the text of the problem file at file, or "" for a ref with a scheme. A ref with a leading slash is from the root, and any other ref is next to the problem file.
func FigurePath(file, ref string) string {
	if strings.Contains(ref, "://") {
		return ""
	}
	if strings.HasPrefix(ref, "/") {
		return path.Clean(strings.TrimPrefix(ref, "/"))
	}
	return path.Join(path.Dir(file), ref)
}

// Figures gives the paths from the root of the content of the figures that the canonical statement of p shows, once each, in the order of the text. A figure with a scheme is not a file, so it is not in the list.
func (p *Problem) Figures() []string {
	var out []string
	seen := map[string]bool{}
	for _, ref := range FigureRefs(p.Statement) {
		f := FigurePath(p.Path, ref)
		if f != "" && !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return out
}

// AddToStatement puts text in its own paragraph at the end of the statement id in the body of a problem file b, and gives the new file. The text before the first marker belongs to the canonical statement canonical, as in SplitStatements.
func AddToStatement(b []byte, id, canonical, text string) ([]byte, error) {
	front, body, err := splitFront(b)
	if err != nil {
		return nil, err
	}
	// Each part of the body runs from its start to the next marker: the first part is the canonical statement, and each marker starts a part.
	type part struct {
		id         string
		start, end int
	}
	parts := []part{{id: canonical}}
	for _, m := range markerRE.FindAllStringSubmatchIndex(body, -1) {
		parts[len(parts)-1].end = m[0]
		parts = append(parts, part{id: body[m[2]:m[3]], start: m[1]})
	}
	parts[len(parts)-1].end = len(body)
	// The text goes into the last part of the statement with some text, or into its marker part.
	at := -1
	for i, p := range parts {
		if p.id == id && (at < 0 || strings.TrimSpace(body[p.start:p.end]) != "") {
			at = i
		}
	}
	if at < 0 {
		return nil, fmt.Errorf("the body has no statement %s", id)
	}
	p := parts[at]
	seg := strings.TrimRight(body[p.start:p.end], "\n")
	if strings.TrimSpace(seg) != "" {
		seg += "\n\n"
	} else if p.start > 0 {
		seg = "\n"
	} else {
		seg = ""
	}
	seg += text + "\n"
	if p.end < len(body) {
		seg += "\n"
	}
	var out strings.Builder
	out.WriteString("---\n")
	out.Write(front)
	out.WriteString("\n---\n")
	out.WriteString(body[:p.start])
	out.WriteString(seg)
	out.WriteString(body[p.end:])
	return []byte(out.String()), nil
}

// MapFigures gives text with the target of each Markdown image replaced by f of the target.
func MapFigures(text string, f func(ref string) string) string {
	ms := figureRef.FindAllStringSubmatchIndex(text, -1)
	if len(ms) == 0 {
		return text
	}
	var out strings.Builder
	last := 0
	for _, m := range ms {
		out.WriteString(text[last:m[2]])
		out.WriteString(f(text[m[2]:m[3]]))
		last = m[3]
	}
	out.WriteString(text[last:])
	return out.String()
}
