package site

import (
	"fmt"
	"hash/fnv"
	"html"
	"regexp"
	"strings"

	"github.com/tamnd/olymmath-agent/asy"
)

var asyBlock = regexp.MustCompile(`(?is)\[asy\](.*?)\[/asy\]`)

// sizeCommands are TeX font commands in labels. The figure sets the font size, so they go.
var sizeCommands = regexp.MustCompile(`\\(?:tiny|scriptsize|footnotesize|small|normalsize|large|Large|LARGE|huge|Huge)\b\s*`)

// splitFigures replaces each Asymptote block of src with a token in a paragraph of its own, and returns the text and the blocks.
func splitFigures(src string) (string, []string) {
	var codes []string
	text := asyBlock.ReplaceAllStringFunc(src, func(m string) string {
		codes = append(codes, asyBlock.FindStringSubmatch(m)[1])
		return fmt.Sprintf("\n\n@@ASY%d@@\n\n", len(codes)-1)
	})
	return text, codes
}

// figure draws an Asymptote program as SVG with its labels over it. When the program is not in the subset, the figure shows its source.
func (m *Markup) figure(code string) (string, error) {
	h := fnv.New32a()
	_, _ = h.Write([]byte(code))
	id := fmt.Sprintf("asy%08x", h.Sum32())
	pic, err := asy.Run(code)
	var f *asy.Figure
	if err == nil {
		f, err = pic.Layout(id)
	}
	if err != nil {
		return `<pre class="asy-source"><code>` + html.EscapeString(strings.TrimSpace(code)) + `</code></pre>`, err
	}
	var b strings.Builder
	fmt.Fprintf(&b, `<figure class="asy" style="width:%.0fpx">%s`, f.Width, f.SVG)
	for _, l := range f.Labels {
		style := fmt.Sprintf("left:%.2f%%;top:%.2f%%;font-size:%.2fcqw;transform:translate(calc(%.0f%% + %.2fem),calc(%.0f%% + %.2fem))", l.Left, l.Top, l.Size, 100*l.DX, l.MX, 100*l.DY, l.MY)
		if l.Color != "" {
			style += ";color:" + l.Color
		}
		fmt.Fprintf(&b, `<span style="%s">%s</span>`, style, m.label(l.Text))
	}
	b.WriteString(`</figure>`)
	return b.String(), nil
}

// label renders the TeX of a label: the formulas with KaTeX and the rest as text.
func (m *Markup) label(s string) string {
	s = sizeCommands.ReplaceAllString(s, "")
	text, spans := splitMath(s)
	out := html.EscapeString(text)
	for i := len(spans) - 1; i >= 0; i-- {
		r, err := m.tex.Render(spans[i].tex, false)
		if err != nil {
			r = html.EscapeString(spans[i].tex)
		}
		out = strings.ReplaceAll(out, fmt.Sprintf("@@MATH%d@@", i), r)
	}
	return out
}
