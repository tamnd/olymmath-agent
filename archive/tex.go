// Package archive reads the problem files of curated archives, for example the TeX files of the Putnam archive, and makes seed rows from them.
package archive

import (
	"fmt"
	"regexp"
	"strings"
)

// displayEnvs maps a display environment of TeX to the environment that KaTeX shows inside $$...$$. An empty value means no environment.
var displayEnvs = map[string]string{
	"align*": "aligned", "align": "aligned", "eqnarray*": "aligned", "eqnarray": "aligned",
	"gather*": "gathered", "gather": "gathered", "equation*": "", "equation": "", "displaymath": "",
}

var (
	commentRE = regexp.MustCompile(`(?m)(^|[^\\])%.*$`)
	envRE     = regexp.MustCompile(`^\\begin\{([a-z]+\*?)\}`)
	itemRE    = regexp.MustCompile(`^\\item(?:\[([^\]]*)\])?\s*`)
	// textMacros are the macros of text that become Markdown.
	emRE     = regexp.MustCompile(`\{\\(?:em|it)\s+([^{}]*)\}`)
	bfRE     = regexp.MustCompile(`\{\\bf\s+([^{}]*)\}`)
	emphRE   = regexp.MustCompile(`\\(?:emph|textit)\{([^{}]*)\}`)
	textbfRE = regexp.MustCompile(`\\textbf\{([^{}]*)\}`)
	dropRE   = regexp.MustCompile(`\\(?:noindent|medskip|bigskip|smallskip|newline|par|vspace\{[^}]*\}|hspace\{[^}]*\}|vfill|hfill)\b\s*`)
	macroRE  = regexp.MustCompile(`\\[a-zA-Z]+`)
	joinRE   = regexp.MustCompile(`[ \t]*\n[ \t]*`)
	paraRE   = regexp.MustCompile(`\n\s*\n+`)
	spacesRE = regexp.MustCompile(`[ \t]+`)
)

var textChars = strings.NewReplacer("``", `"`, "''", `"`, "---", "-", "--", "-", "~", " ", `\%`, "%", `\&`, "&", `\ldots`, "...", `\dots`, "...", `\#`, "#", `\_`, "_")

// Markdown converts the TeX of one problem to Markdown with math. Math stays as written. Lists become paragraphs that start with their labels, and the display environments become $$...$$. It returns an error for TeX that has no Markdown form, for example a figure.
func Markdown(tex string) (string, error) {
	tex = commentRE.ReplaceAllString(tex, "$1")
	var out strings.Builder
	var lists []int
	// closers holds the Markdown that ends each open group of text, for example "*" for \emph{.
	var closers []string
	for i := 0; i < len(tex); {
		rest := tex[i:]
		if prefix, open, closer := group(rest); prefix != "" {
			out.WriteString(open)
			closers = append(closers, closer)
			i += len(prefix)
			continue
		}
		switch {
		case rest[0] == '}' && len(closers) > 0:
			out.WriteString(closers[len(closers)-1])
			closers = closers[:len(closers)-1]
			i++
		case strings.HasPrefix(rest, "$$"):
			j := strings.Index(rest[2:], "$$")
			if j < 0 {
				return "", fmt.Errorf("a $$ with no end")
			}
			out.WriteString("$$" + rest[2:2+j] + "$$")
			i += 4 + j
		case rest[0] == '$':
			j := closing(rest[1:])
			if j < 0 {
				return "", fmt.Errorf("a $ with no end")
			}
			out.WriteString("$" + unnest(rest[1:1+j]) + "$")
			i += 2 + j
		case strings.HasPrefix(rest, `\[`) || strings.HasPrefix(rest, `\(`):
			end := `\]`
			if rest[1] == '(' {
				end = `\)`
			}
			j := strings.Index(rest[2:], end)
			if j < 0 {
				return "", fmt.Errorf("a %s with no end", rest[:2])
			}
			out.WriteString(rest[:2+j+2])
			i += 4 + j
		case strings.HasPrefix(rest, `\begin{`):
			m := envRE.FindStringSubmatch(rest)
			if m == nil {
				return "", fmt.Errorf("an environment with no name")
			}
			name := m[1]
			if inner, ok := displayEnvs[name]; ok {
				end := `\end{` + name + `}`
				j := strings.Index(rest, end)
				if j < 0 {
					return "", fmt.Errorf("%s with no end", name)
				}
				body := strings.TrimSpace(rest[len(m[0]):j])
				if inner != "" {
					body = `\begin{` + inner + `}` + body + `\end{` + inner + `}`
				}
				out.WriteString("\n\n$$" + body + "$$\n\n")
				i += j + len(end)
				continue
			}
			switch name {
			case "enumerate":
				lists = append(lists, 1)
			case "itemize":
				lists = append(lists, 0)
			case "center", "quote", "quotation", "verse":
			default:
				return "", fmt.Errorf("the environment %s has no Markdown form", name)
			}
			out.WriteString("\n\n")
			i += len(m[0])
		case strings.HasPrefix(rest, `\end{`):
			j := strings.Index(rest, "}")
			name := rest[5:j]
			if (name == "enumerate" || name == "itemize") && len(lists) > 0 {
				lists = lists[:len(lists)-1]
			}
			out.WriteString("\n\n")
			i += j + 1
		case strings.HasPrefix(rest, `\item`) && (len(rest) == 5 || !isLetter(rest[5])):
			m := itemRE.FindStringSubmatch(rest)
			label := m[1]
			if m[0] == `\item` || !strings.HasPrefix(m[0], `\item[`) {
				switch {
				case len(lists) > 0 && lists[len(lists)-1] > 0:
					label = fmt.Sprintf("%d.", lists[len(lists)-1])
					lists[len(lists)-1]++
				default:
					label = "-"
				}
			}
			out.WriteString("\n\n" + label + " ")
			i += len(m[0])
		default:
			j := strings.IndexAny(rest[1:], `$\{}`)
			if j < 0 {
				j = len(rest) - 1
			}
			if rest[0] == '\\' {
				// A macro of text, with its argument when it has one.
				k := 1
				for k < len(rest) && isLetter(rest[k]) {
					k++
				}
				if k == 1 && k < len(rest) {
					k++
				}
				j = k - 1
				if k < len(rest) && rest[k] == '{' {
					if e := strings.Index(rest[k:], "}"); e >= 0 {
						j = k + e
					}
				}
			}
			out.WriteString(rest[:1+j])
			i += 1 + j
		}
	}
	return text(out.String())
}

// groups are the groups of text that become Markdown, with the Markdown that opens and closes them. A plain group has no Markdown.
var groups = []struct{ prefix, open, close string }{
	{`\emph{`, "*", "*"}, {`\textit{`, "*", "*"}, {`\textbf{`, "**", "**"},
	{`{\em `, "*", "*"}, {`{\it `, "*", "*"}, {`{\bf `, "**", "**"},
	{`\textsuperscript{`, "", ""}, {`\text{`, "", ""}, {`\footnote{`, " (", ")"}, {`{`, "", ""},
}

// group gives the group of text at the start of s, or an empty prefix. A prefix that ends with a space also matches a line break.
func group(s string) (prefix, open, end string) {
	for _, g := range groups {
		p := strings.TrimSuffix(g.prefix, " ")
		if strings.HasPrefix(s, p) && (p == g.prefix || len(s) > len(p) && (s[len(p)] == ' ' || s[len(p)] == '\n')) {
			return s[:len(g.prefix)], g.open, g.close
		}
	}
	return "", "", ""
}

// text converts the parts of s outside math and joins the lines of each paragraph.
func text(s string) (string, error) {
	var b strings.Builder
	parts := splitMath(s)
	for k, p := range parts {
		if k%2 == 1 {
			b.WriteString(p)
			continue
		}
		p = emRE.ReplaceAllString(p, "*$1*")
		p = emphRE.ReplaceAllString(p, "*$1*")
		p = bfRE.ReplaceAllString(p, "**$1**")
		p = textbfRE.ReplaceAllString(p, "**$1**")
		p = dropRE.ReplaceAllString(p, "")
		p = textChars.Replace(p)
		p = strings.ReplaceAll(p, `\\`, "\n")
		if m := macroRE.FindString(p); m != "" {
			return "", fmt.Errorf("the macro %s is not known in text", m)
		}
		p = strings.NewReplacer("{", "", "}", "").Replace(p)
		b.WriteString(p)
	}
	s = paraRE.ReplaceAllString(b.String(), "\x00")
	s = joinRE.ReplaceAllString(s, " ")
	s = spacesRE.ReplaceAllString(s, " ")
	paras := strings.Split(s, "\x00")
	var keep []string
	for _, p := range paras {
		if p = strings.TrimSpace(p); p != "" {
			keep = append(keep, p)
		}
	}
	return strings.Join(keep, "\n\n"), nil
}

// splitMath splits s into text and math: the parts with an odd index are math, with their delimiters.
func splitMath(s string) []string {
	var parts []string
	last := 0
	for i := 0; i < len(s); {
		var end string
		switch {
		case strings.HasPrefix(s[i:], "$$"):
			end = "$$"
		case s[i] == '$':
			end = "$"
		case strings.HasPrefix(s[i:], `\[`):
			end = `\]`
		case strings.HasPrefix(s[i:], `\(`):
			end = `\)`
		case s[i] == '\\':
			i += 2
			continue
		default:
			i++
			continue
		}
		open := len(end)
		if end == `\]` || end == `\)` {
			open = 2
		}
		j := strings.Index(s[i+open:], end)
		if end == "$" {
			j = closing(s[i+1:])
		}
		if j < 0 {
			i++
			continue
		}
		parts = append(parts, s[last:i], s[i:i+open+j+len(end)])
		i += open + j + len(end)
		last = i
	}
	return append(parts, s[last:])
}

// nestedRE matches a \text group with math inside, for example \text{the $k$th turn}.
var nestedRE = regexp.MustCompile(`\\text\{([^{}]*\$[^{}]*)\}`)

// unnest takes the math out of each \text group of a formula, so that the formula has no $ inside: \text{the $k$th turn} becomes \text{the }{k}\text{th turn}.
func unnest(math string) string {
	return nestedRE.ReplaceAllStringFunc(math, func(g string) string {
		parts := strings.Split(g[len(`\text{`):len(g)-1], "$")
		var b strings.Builder
		for k, p := range parts {
			switch {
			case k%2 == 1:
				b.WriteString("{" + p + "}")
			case p != "":
				b.WriteString(`\text{` + p + `}`)
			}
		}
		return b.String()
	})
}

// closing gives the index of the first $ in s that is not escaped and not inside braces, or -1. A $ inside braces opens math inside a text group, for example in \text{the $k$th turn}.
func closing(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\\':
			i++
		case '{':
			depth++
		case '}':
			depth--
		case '$':
			if depth <= 0 {
				return i
			}
		}
	}
	return -1
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
