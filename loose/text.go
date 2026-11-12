package loose

import (
	"regexp"
	"strings"

	"github.com/tamnd/olymmath-agent/canon"
)

var (
	// byRE is the line of the author at the end of a text, such as "*Proposed by John Cuya, Peru*".
	byRE = regexp.MustCompile(`(?im)^\s*[*_(]*\s*(?:proposed by|author:|by)\s[^\n]{0,80}$`)
	// texSame are TeX commands with the same meaning, and spacing that has none. \langle and \rangle are < and >, and an empty group, as in $75_{}^{}$, is nothing.
	texSame = strings.NewReplacer(
		`\left`, "", `\right`, "", `\dfrac`, `\frac`, `\tfrac`, `\frac`,
		`\geqslant`, `\ge`, `\leqslant`, `\le`, `\geq`, `\ge`, `\leq`, `\le`, `\neq`, `\ne`,
		`^{\circ}`, "°", `^\circ`, "°", `^{o}`, "°", `^o`, "°", `...`, `\dots`, `\cdots`, `\dots`, `\ldots`, `\dots`, `\dotsc`, `\dots`, `\dotsb`, `\dots`,
		`\,`, "", `\;`, "", `\!`, "", `\:`, "", `\ `, "", `\displaystyle`, "",
		`\textbf`, `\text`, `\textrm`, `\text`, `\mathrm`, `\text`,
		`\measuredangle`, `\angle`, `\vartriangle`, `\triangle`, `\bigtriangleup`, `\triangle`,
		`\langle`, `<`, `\rangle`, `>`, `^{}`, "", `_{}`, "",
	)
	// decoRE is a segment or an angle written with a mark around its letters, as \overline{AB}, \angle{ABC} and \triangle{ABC}. One copy of a problem has the mark and another has not, and the grams of the marks outweighed the problem.
	decoRE = regexp.MustCompile(`\\(overline|angle|triangle)\s*\{([A-Za-z0-9']*)\}`)
	// oneRE is a group of one letter or digit after _ or ^, such as x_{1}.
	oneRE = regexp.MustCompile(`([_^])\{([A-Za-z0-9])\}`)
	numRE = regexp.MustCompile(`^-?[0-9]+(?:[.,][0-9]+)?$`)
	// cmdEndRE is a TeX command at the end of a word of a formula.
	cmdEndRE = regexp.MustCompile(`\\[A-Za-z]+$`)
)

// matchText gives the form of a text that the match compares. Two copies of one problem often differ only in the spacing of their formulas, as $A B C$ and $ABC$, in TeX commands with the same meaning, as \geq and \ge, in a number in a formula or not, or in the line of the author. The form has none of these differences. It is used only to compare; the text that is kept does not change.
func matchText(text string) string {
	text = byRE.ReplaceAllString(text, "")
	return mathRE.ReplaceAllStringFunc(text, func(m string) string {
		body := strings.Trim(m, "$")
		body = texSame.Replace(body)
		body = decoRE.ReplaceAllStringFunc(body, func(m string) string {
			g := decoRE.FindStringSubmatch(m)
			if g[1] == "overline" {
				return g[2]
			}
			return `\` + g[1] + " " + g[2]
		})
		body = joinFields(body)
		body = oneRE.ReplaceAllString(body, "$1$2")
		if numRE.MatchString(body) {
			return body
		}
		return "$" + body + "$"
	})
}

// joinFields gives a formula with no spaces, but for one space between a command and a letter: "0 \le AB" and "0\le AB" are "0\le AB", and not the command "\leAB".
func joinFields(body string) string {
	var b strings.Builder
	prev := ""
	for _, f := range strings.Fields(body) {
		if prev != "" && (f[0] >= 'a' && f[0] <= 'z' || f[0] >= 'A' && f[0] <= 'Z') && cmdEndRE.MatchString(prev) {
			b.WriteByte(' ')
		}
		b.WriteString(f)
		prev = f
	}
	return b.String()
}

// choicesRE finds the start of the answer choices of a multiple choice problem: "\\textbf{(A)}", "\\mathbf{(A)}", "\\mathrm{(A)}", "(\\mathrm{A})", "\\qquad (A)" or "(A)" at the start of a line. The content writes the letter in the command and the parentheses outside it, as "(\\mathrm {A})", which the match form makes "(\\text{A})"; a row of the same problem has "\\textbf{(A)}". When only one of the two loses its choices, their texts are far apart.
var choicesRE = regexp.MustCompile(`(?:\\(?:textbf|mathbf|mathrm|textrm|text|mbox)\s*\{\s*\(A\)|\(\s*\\(?:textbf|mathbf|mathrm|textrm|text|mbox)\s*\{\s*A\s*\}\s*\)|\\qquad\s*\(A\)|(?m:^)\s*\(A\)\s)`)

// stem gives a match text without the answer choices. The choices of two problems of one contest are often the same numbers, such as 3, 4, 5, 6 and 7, so they make two different problems look alike.
func stem(form string) string {
	m := choicesRE.FindStringIndex(form)
	if m == nil || m[0] < canon.MinText {
		return form
	}
	st := strings.TrimSpace(form[:m[0]])
	// The choices often start a formula: "$\textbf{(A)}". The "$" that opens it is not part of the stem, but the "$" that closes a formula before the choices is.
	if strings.Count(st, "$")%2 == 1 && strings.HasSuffix(st, "$") {
		st = strings.TrimSpace(strings.TrimSuffix(st, "$"))
	}
	return st
}
