package seed

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	// aopsTagRE finds a tag of an AoPS post. An attribute value in quotes can hold ">".
	aopsTagRE = regexp.MustCompile(`<(/?)([a-zA-Z0-9]+)((?:[^>"']|"[^"]*"|'[^']*')*)>`)
	aopsAltRE = regexp.MustCompile(`\balt=(?:"([^"]*)"|'([^']*)')`)
	aopsClsRE = regexp.MustCompile(`\bclass=(?:"([^"]*)"|'([^']*)')`)
	// aopsUnsure finds the parts of a post that make its text unsure: a hidden block, which can hold a hint or an answer, TeX that AoPS did not render, and a picture that is not a formula or an Asymptote figure.
	aopsUnsure = regexp.MustCompile(`cmty-hide|aopscode-error|bbcode_img`)
)

// aopsHTML gives the text of the HTML of an AoPS post, as the column problem_html of hf-aops-wiki has it. A formula is a picture with its TeX in the alt text, and an Asymptote figure is a picture with its source in the alt text. A line break, a block and an item of a list start a new line. The HTML was pretty-printed, so each text and tag is on its own line, and the space before a punctuation mark and after an opening bracket is not the text's. It gives false for a post with a hidden block, a TeX error or another picture, because then the text is not sure to be the full statement.
func aopsHTML(h string) (string, bool) {
	if strings.TrimSpace(h) == "" || aopsUnsure.MatchString(h) {
		return "", false
	}
	var b strings.Builder
	// glue is true when the next piece follows with no space, as after a new line or an opening bracket.
	glue := true
	add := func(s string) {
		if s == "" {
			return
		}
		if !glue && !strings.ContainsAny(s[:1], ".,;:!?)]}") {
			b.WriteByte(' ')
		}
		b.WriteString(s)
		glue = strings.ContainsAny(s[len(s)-1:], "([{\n")
	}
	// line starts a new line, and br adds a line break, so that two br give a blank line between paragraphs.
	line := func() {
		if b.Len() > 0 && !strings.HasSuffix(b.String(), "\n") {
			b.WriteByte('\n')
		}
		glue = true
	}
	br := func() {
		b.WriteByte('\n')
		glue = true
	}
	// lists holds the item number of each open list, or -1 for a list with no numbers.
	var lists []int
	at := 0
	for _, m := range aopsTagRE.FindAllStringSubmatchIndex(h, -1) {
		add(strings.Join(strings.Fields(html.UnescapeString(h[at:m[0]])), " "))
		at = m[1]
		closing, name, attrs := h[m[2]:m[3]] == "/", strings.ToLower(h[m[4]:m[5]]), h[m[6]:m[7]]
		switch name {
		case "img":
			cls := attr(aopsClsRE, attrs)
			alt := strings.TrimSpace(html.UnescapeString(attr(aopsAltRE, attrs)))
			switch {
			case strings.Contains(cls, "bbcode_smiley"):
			case strings.Contains(cls, "asy-image"):
				line()
				add(alt)
				line()
			case strings.Contains(cls, "latexcenter"):
				line()
				add(alt)
				line()
			case strings.Contains(cls, "latex"):
				add(alt)
			default:
				return "", false
			}
		case "br":
			br()
		case "hr":
			line()
		case "div", "p", "pre":
			line()
		case "ul", "ol":
			if closing {
				if len(lists) > 0 {
					lists = lists[:len(lists)-1]
				}
			} else if name == "ol" {
				lists = append(lists, 0)
			} else {
				lists = append(lists, -1)
			}
			line()
		case "li":
			line()
			if closing || len(lists) == 0 {
				break
			}
			if n := lists[len(lists)-1]; n >= 0 {
				lists[len(lists)-1] = n + 1
				add(strconv.Itoa(n+1) + ".")
			} else {
				add("-")
			}
		}
	}
	add(strings.Join(strings.Fields(html.UnescapeString(h[at:])), " "))
	lines := strings.Split(b.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	s := strings.TrimSpace(blankRE.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
	return s, s != ""
}

// blankRE finds two or more line breaks with only spaces between them.
var blankRE = regexp.MustCompile(`\n(?:[ ]*\n)+`)

func attr(re *regexp.Regexp, attrs string) string {
	m := re.FindStringSubmatch(attrs)
	if m == nil {
		return ""
	}
	return m[1] + m[2]
}
