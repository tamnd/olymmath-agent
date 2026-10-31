package archive

import (
	"fmt"
	"regexp"
	"strings"
)

// Doc is one file of an archive: the problems of one edition.
type Doc struct {
	Year int
	URL  string
}

// Problem is one problem of a file.
type Problem struct {
	// Number is the label of the problem in the file, for example A1.
	Number string
	Text   string
	// Err tells why the TeX of the problem has no Markdown form. Text is empty when Err is not nil.
	Err error
}

// Lane reads one archive.
type Lane struct {
	// Label is the competition label of the rows.
	Label string
	Docs  func(from, to int) []Doc
	Parse func(tex string) []Problem
}

// Lanes maps a source id of the source register to its lane.
var Lanes = map[string]Lane{
	"kedlaya-putnam": {Label: "Putnam", Docs: Putnam, Parse: ParsePutnam},
}

// PutnamFirst is the first year of the Putnam archive.
const PutnamFirst = 1985

// Putnam gives the problem files of the Putnam archive of Kiran Kedlaya from the year from to the year to.
func Putnam(from, to int) []Doc {
	var out []Doc
	for y := max(from, PutnamFirst); y <= to; y++ {
		out = append(out, Doc{Year: y, URL: fmt.Sprintf("https://kskedlaya.org/putnam-archive/%d.tex", y)})
	}
	return out
}

var (
	newcommandRE = regexp.MustCompile(`\\newcommand\{\\([a-zA-Z]+)\}\{(.*)\}\s*$`)
	// textboxRE matches the boxes of text that KaTeX knows as \text.
	textboxRE = regexp.MustCompile(`\\mbox\{|\\parbox\{[^}]*\}\{`)
)

// Expand replaces each macro that the file defines with \\newcommand and no arguments, for example \\RR for \\mathbb{R}, and each \\mbox or \\parbox with \\text. The lines of the definitions are removed.
func Expand(tex string) string {
	var keep []string
	type macro struct {
		re   *regexp.Regexp
		name string
		body string
	}
	var macros []macro
	for _, line := range strings.Split(tex, "\n") {
		if m := newcommandRE.FindStringSubmatch(strings.TrimSpace(line)); m != nil {
			macros = append(macros, macro{regexp.MustCompile(`\\` + m[1] + `([^a-zA-Z]|$)`), m[1], m[2]})
			continue
		}
		keep = append(keep, line)
	}
	tex = strings.Join(keep, "\n")
	for _, m := range macros {
		tex = m.re.ReplaceAllStringFunc(tex, func(s string) string { return m.body + s[len(m.name)+1:] })
	}
	return textboxRE.ReplaceAllString(tex, `\text{`)
}

// putnamRE matches the label of a problem: \item[A1] or, in the early files, \item[A--1].
var putnamRE = regexp.MustCompile(`\\item\[([AB])(?:--|-)?([1-6])\]`)

// ParsePutnam gives the problems of one file of the Putnam archive. The problems are the items of the top list, and the text of the last problem ends at the end of that list.
func ParsePutnam(tex string) []Problem {
	tex = Expand(tex)
	if i := strings.LastIndex(tex, `\end{itemize}`); i >= 0 {
		tex = tex[:i]
	}
	ms := putnamRE.FindAllStringSubmatchIndex(tex, -1)
	var out []Problem
	for k, m := range ms {
		end := len(tex)
		if k+1 < len(ms) {
			end = ms[k+1][0]
		}
		p := Problem{Number: tex[m[2]:m[3]] + tex[m[4]:m[5]]}
		p.Text, p.Err = Markdown(tex[m[1]:end])
		if p.Err != nil {
			p.Text = ""
		}
		out = append(out, p)
	}
	return out
}
