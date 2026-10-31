package site

import (
	"bytes"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"

	"github.com/tamnd/olymmath-agent/katex"
)

// mathSpan is one formula of a text.
type mathSpan struct {
	tex     string
	display bool
}

// splitMath replaces each formula of src with a token and returns the text and the formulas. It knows $...$, $$...$$, \(...\) and \[...\]. A dollar sign after a backslash is not a delimiter.
func splitMath(src string) (string, []mathSpan) {
	var out strings.Builder
	var spans []mathSpan
	i := 0
	for i < len(src) {
		open, close, display := "", "", false
		switch {
		case strings.HasPrefix(src[i:], `\$`):
			out.WriteString(`\$`)
			i += 2
			continue
		case strings.HasPrefix(src[i:], "$$"):
			open, close, display = "$$", "$$", true
		case strings.HasPrefix(src[i:], `\[`):
			open, close, display = `\[`, `\]`, true
		case strings.HasPrefix(src[i:], `\(`):
			open, close = `\(`, `\)`
		case src[i] == '$':
			open, close = "$", "$"
		}
		if open == "" {
			out.WriteByte(src[i])
			i++
			continue
		}
		end := indexClose(src[i+len(open):], close)
		if end < 0 {
			out.WriteString(open)
			i += len(open)
			continue
		}
		tex := strings.TrimSpace(src[i+len(open) : i+len(open)+end])
		fmt.Fprintf(&out, "@@MATH%d@@", len(spans))
		spans = append(spans, mathSpan{tex: tex, display: display})
		i += len(open) + end + len(close)
	}
	return out.String(), spans
}

// indexClose finds the closing delimiter, and skips a delimiter after a backslash.
func indexClose(s, close string) int {
	for j := 0; j+len(close) <= len(s); j++ {
		if s[j] == '\\' && close == "$" {
			j++
			continue
		}
		if strings.HasPrefix(s[j:], close) {
			return j
		}
	}
	return -1
}

var md = goldmark.New(goldmark.WithExtensions(extension.Table, extension.Strikethrough))

// Markup renders Markdown with TeX math to HTML. It returns the HTML and the formulas that KaTeX cannot parse. A formula with an error shows as its source in a code element.
type Markup struct {
	tex *katex.Renderer
}

// NewMarkup returns a Markup with the macros of the site.
func NewMarkup(macros map[string]string) (*Markup, error) {
	r, err := katex.New(macros)
	if err != nil {
		return nil, err
	}
	return &Markup{tex: r}, nil
}

// HTML renders src. Each error is a formula that KaTeX cannot parse.
func (m *Markup) HTML(src string) (string, []error) {
	text, spans := splitMath(src)
	var buf bytes.Buffer
	if err := md.Convert([]byte(text), &buf); err != nil {
		return "<p>" + html.EscapeString(src) + "</p>", []error{err}
	}
	out := buf.String()
	var errs []error
	for i := len(spans) - 1; i >= 0; i-- {
		s := spans[i]
		rendered, err := m.tex.Render(s.tex, s.display)
		if err != nil {
			errs = append(errs, err)
			rendered = `<code class="math-error" title="` + html.EscapeString(err.Error()) + `">` + html.EscapeString(s.tex) + `</code>`
		}
		out = strings.ReplaceAll(out, "@@MATH"+strconv.Itoa(i)+"@@", rendered)
	}
	// A display formula alone in a paragraph does not need the paragraph.
	return out, errs
}

// Plain returns the text of src for search and for descriptions: the formulas keep their TeX source, and the Markdown marks go.
func Plain(src string) string {
	text, spans := splitMath(src)
	r := strings.NewReplacer("*", "", "_", " ", "`", "", "#", "", "\n", " ")
	text = r.Replace(text)
	for i := len(spans) - 1; i >= 0; i-- {
		text = strings.ReplaceAll(text, "@@MATH"+strconv.Itoa(i)+"@@", spans[i].tex)
	}
	return strings.Join(strings.Fields(text), " ")
}
