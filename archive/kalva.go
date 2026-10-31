package archive

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// KalvaBase is the root of the mirror of the Kalva site of John Scholes.
const KalvaBase = "https://prase.cz/kalva/"

// kalvaSet is one competition of Kalva: one page for each year, for example vietnam/viet99.html.
type kalvaSet struct {
	Dir, Prefix, Label string
	From, To           int
	Skip               []int
}

// kalvaSets are the competitions of the mirror. Each label is a name of the competition register. The All-Russian pages are not here, because their numbers do not follow the grades.
var kalvaSets = []kalvaSet{
	{Dir: "imo", Prefix: "imo", Label: "International Mathematical Olympiad", From: 1959, To: 2003, Skip: []int{1980}},
	{Dir: "usa", Prefix: "usa", Label: "USAMO", From: 1972, To: 2003},
	{Dir: "apmo", Prefix: "apmo", Label: "Asian Pacific Mathematical Olympiad", From: 1989, To: 2004},
	{Dir: "vietnam", Prefix: "viet", Label: "Vietnam National Olympiad", From: 1962, To: 2003, Skip: []int{1973}},
}

// Kalva gives the year pages of the Kalva mirror from the year from to the year to.
func Kalva(from, to int) []Doc {
	var out []Doc
	for _, s := range kalvaSets {
		for y := max(from, s.From); y <= min(to, s.To); y++ {
			if slices.Contains(s.Skip, y) {
				continue
			}
			out = append(out, Doc{Year: y, Label: s.Label, URL: fmt.Sprintf("%s%s/%s%02d.html", KalvaBase, s.Dir, s.Prefix, y%100)})
		}
	}
	return out
}

var (
	// kalvaMark finds the label of a problem, for example <b>A1.</b> or <b>3.</b>.
	kalvaMark = regexp.MustCompile(`(?i)<b>\s*([AB]?\d{1,2})\.\s*</b>`)
	kalvaEnd  = regexp.MustCompile(`(?i)</td>`)
	// kalvaScript is a run of text before a superscript or a subscript, for example the 4 of 4<sup>2x</sup>.
	kalvaScript = regexp.MustCompile(`(?i)([^\s(){}\[\],;:<>]*)<(sup|sub)>(.*?)</(?:sup|sub)>`)
	kalvaTag    = regexp.MustCompile(`<[^>]*>`)
	htmlItem    = regexp.MustCompile(`(?i)<li[^>]*>`)
	kalvaSpace  = regexp.MustCompile(`\s+`)
)

// errKalvaImage tells that a problem has a picture, which the text alone does not give.
var errKalvaImage = errors.New("the problem has a picture")

// ParseKalva reads the problems of one year page of Kalva. The number of a problem is its place on the page, so that A1, A2, A3, B1 are 1, 2, 3, 4.
func ParseKalva(page string) []Problem {
	marks := kalvaMark.FindAllStringSubmatchIndex(page, -1)
	var out []Problem
	for i, m := range marks {
		body := page[m[1]:]
		if i+1 < len(marks) {
			body = page[m[1]:marks[i+1][0]]
		}
		if e := kalvaEnd.FindStringIndex(body); e != nil {
			body = body[:e[0]]
		}
		p := Problem{Number: strconv.Itoa(i + 1)}
		if strings.Contains(strings.ToLower(body), "<img") {
			p.Err = errKalvaImage
		} else {
			p.Text = htmlText(body)
		}
		out = append(out, p)
	}
	return out
}

// htmlText turns the HTML of a problem into Markdown. A superscript or a subscript and the text just before it become inline TeX: 4<sup>2x</sup> is $4^{2x}$ and a<sub>1</sub> is $a_1$. An item of a list starts a new line.
func htmlText(s string) string {
	// The TeX made here uses the mark \x02 for its dollars, so that two pieces next to each other join and the TeX of the page stays as it is.
	s = strings.NewReplacer("\r", "", "&nbsp;", " ").Replace(s)
	for {
		t := kalvaScript.ReplaceAllStringFunc(s, func(m string) string {
			g := kalvaScript.FindStringSubmatch(m)
			op := "^"
			if strings.EqualFold(g[2], "sub") {
				op = "_"
			}
			base := strings.Trim(g[1], "\x02")
			arg := strings.TrimSpace(kalvaTag.ReplaceAllString(g[3], ""))
			if len([]rune(arg)) != 1 {
				arg = "{" + arg + "}"
			}
			return "\x02" + base + op + arg + "\x02"
		})
		if t == s {
			break
		}
		s = t
	}
	s = htmlItem.ReplaceAllString(s, " \x01 ")
	s = kalvaTag.ReplaceAllString(s, " ")
	s = html.UnescapeString(s)
	s = strings.NewReplacer("\x02\x02", "", "\x02", "$").Replace(s)
	s = strings.TrimSpace(kalvaSpace.ReplaceAllString(s, " "))
	return strings.ReplaceAll(s, " \x01 ", "\n- ")
}
