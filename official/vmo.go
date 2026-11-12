package official

import (
	"regexp"
	"strconv"
	"strings"
)

// vmoMarkRE is the bold mark of a problem on a page of the All-Russian olympiad: the grade and the number, as 9.1.
var vmoMarkRE = regexp.MustCompile(`(?i)<b>\s*(\d{1,2})\.(\d{1,2})\.\s*</b>`)

// vmoFixes are the signs that an old page draws with HTML tags, and their HTML entities: an underlined > is ≥, a struck C is ∈, a bold R is the set of real numbers, and -> is an arrow. A table 2*n is 2×n.
var vmoFixes = []struct {
	re *regexp.Regexp
	to string
}{
	{regexp.MustCompile(`(?i)<u>\s*&gt;\s*</u>`), "&ge;"},
	{regexp.MustCompile(`(?i)<u>\s*&lt;\s*</u>`), "&le;"},
	{regexp.MustCompile(`(?i)<s>\s*C\s*</s>`), "&isin;"},
	{regexp.MustCompile(`(?i)<b>\s*R\s*</b>`), "<i>ℝ</i>"},
	{regexp.MustCompile(`-&gt;`), "&rarr;"},
	{regexp.MustCompile(`(табл\S*\s+\d+)\*`), "$1&times;"},
	// The name of the author before the text of a problem.
	{regexp.MustCompile(`(?i)(<b>\s*\d{1,2}\.\d{1,2}\.\s*</b>)\s*<br\s*/?>\s*\(<i>[^<]*</i>\)`), "$1"},
}

// vmoPowerZero is a list of powers that starts with the power 0, as 2^0, 2^1, which RuHTML would take for a sign of degrees.
var vmoPowerZero = regexp.MustCompile(`(?i)\d<sup>0</sup>\s*,\s*\d<sup>1</sup>`)

// VmoGrades are the grades of the final round of the All-Russian olympiad that have positions.
func VmoGrades(g int) bool { return g >= 8 && g <= 11 }

// VmoPage reads the problems of a web page of the final round of the All-Russian olympiad, as 31/zadachi.htm of 2005 on olympiads.mccme.ru/vmo. The page has a heading for each grade, and a problem starts at its grade and number in bold, as 9.5. The numbers of the two days go on from 1 to 8 in one paper, so problem G.N has the position pG N. A paper with a mark of another grade is left out, because its numbers are not sure.
func VmoPage(page, edition, url string) []RuProblem {
	var out []RuProblem
	for _, s := range RuGrades(page) {
		if !VmoGrades(s.Grade) {
			continue
		}
		good := true
		for _, m := range vmoMarkRE.FindAllStringSubmatch(s.HTML, -1) {
			if g, _ := strconv.Atoi(m[1]); g != s.Grade {
				good = false
			}
		}
		if !good {
			continue
		}
		html := s.HTML
		for _, f := range vmoFixes {
			html = f.re.ReplaceAllString(html, f.to)
		}
		html = vmoMarkRE.ReplaceAllString(html, "<b>$2.</b>")
		for i, h := range RuNumbered(html) {
			text, fig := RuHTML(h)
			text = vmoReals(text)
			if vmoPowerZero.MatchString(h) {
				text = strings.Replace(text, `^\circ`, "^0", 1)
			}
			p := RuProblem{Edition: edition, Position: s.Prefix() + strconv.Itoa(i+1), Text: text, URL: url, Figure: fig}
			if RuLayout(h) {
				p.Flags = append(p.Flags, "layout")
			}
			out = append(out, p)
		}
	}
	return out
}

// vmoReals writes the set of real numbers ℝ as \mathbb{R}, in math. A ℝ next to math, with a space or none between them, joins it.
func vmoReals(s string) string {
	if !strings.Contains(s, "ℝ") {
		return s
	}
	var b strings.Builder
	in := false
	for _, r := range s {
		switch {
		case r == '$':
			in = !in
			b.WriteRune(r)
		case r == 'ℝ' && in:
			b.WriteString(`\mathbb{R}`)
		case r == 'ℝ':
			b.WriteString(`$\mathbb{R}$`)
		default:
			b.WriteRune(r)
		}
	}
	return strings.NewReplacer(`$ $\mathbb{R}$`, ` \mathbb{R}$`, `\mathbb{R}$ $`, `\mathbb{R} `).Replace(b.String())
}
