package loose

import (
	"regexp"
	"strings"
)

var (
	// byRE is the line of the author at the end of a text, such as "*Proposed by John Cuya, Peru*".
	byRE = regexp.MustCompile(`(?im)^\s*[*_(]*\s*(?:proposed by|author:|by)\s[^\n]{0,80}$`)
	// texSame are TeX commands with the same meaning, and spacing that has none.
	texSame = strings.NewReplacer(
		`\left`, "", `\right`, "", `\dfrac`, `\frac`, `\tfrac`, `\frac`,
		`\geqslant`, `\ge`, `\leqslant`, `\le`, `\geq`, `\ge`, `\leq`, `\le`, `\neq`, `\ne`,
		`^{\circ}`, "°", `^\circ`, "°", `^{o}`, "°", `^o`, "°", `...`, `\dots`, `\cdots`, `\dots`, `\ldots`, `\dots`, `\dotsc`, `\dots`, `\dotsb`, `\dots`,
		`\,`, "", `\;`, "", `\!`, "", `\:`, "", `\ `, "", `\displaystyle`, "",
		`\textbf`, `\text`, `\textrm`, `\text`, `\mathrm`, `\text`,
	)
	// oneRE is a group of one letter or digit after _ or ^, such as x_{1}.
	oneRE = regexp.MustCompile(`([_^])\{([A-Za-z0-9])\}`)
	numRE = regexp.MustCompile(`^-?[0-9]+(?:[.,][0-9]+)?$`)
)

// matchText gives the form of a text that the match compares. Two copies of one problem often differ only in the spacing of their formulas, as $A B C$ and $ABC$, in TeX commands with the same meaning, as \geq and \ge, in a number in a formula or not, or in the line of the author. The form has none of these differences. It is used only to compare; the text that is kept does not change.
func matchText(text string) string {
	text = byRE.ReplaceAllString(text, "")
	return mathRE.ReplaceAllStringFunc(text, func(m string) string {
		body := strings.Trim(m, "$")
		body = texSame.Replace(body)
		body = strings.Join(strings.Fields(body), "")
		body = oneRE.ReplaceAllString(body, "$1$2")
		if numRE.MatchString(body) {
			return body
		}
		return "$" + body + "$"
	})
}
