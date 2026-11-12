package official

import (
	"html"
	"regexp"
	"strconv"
)

var (
	// ruHeadRE is a heading of a Russian page.
	ruHeadRE = regexp.MustCompile(`(?is)<h([1-4])\b[^>]*>(.*?)</h[1-4]>`)
	// ruGradeRE is a heading that names a grade paper: "8 класс", "11 класс (2 день)" or "11 класс, 1 день".
	ruGradeRE = regexp.MustCompile(`^(\d{1,2}) класс(?:\s*[,(]\s*([12])\s*день\)?)?$`)
)

// RuSection is the HTML of the problems of one grade paper of a page.
type RuSection struct {
	Grade, Day int
	HTML       string
}

// Prefix gives the start of the positions of the paper: pG, or dG for the second day of a grade (spec 04, as in ru-moscow/2025/d113).
func (s RuSection) Prefix() string {
	if s.Day == 2 {
		return "d" + strconv.Itoa(s.Grade)
	}
	return "p" + strconv.Itoa(s.Grade)
}

// RuGrades cuts a page into the papers of its grades. A paper starts at a heading that names only a grade and maybe a day, and ends at the next heading. The first paper of a grade and a day counts, so that the solutions after the problems are left out. A heading such as "11 класс, вариант А" names a variant, which has no position, so its paper is left out.
func RuGrades(page string) []RuSection {
	page = ruComment.ReplaceAllString(page, "")
	heads := ruHeadRE.FindAllStringSubmatchIndex(page, -1)
	seen := map[string]bool{}
	var out []RuSection
	for i, h := range heads {
		name := RuPlain(html.UnescapeString(ruTag.ReplaceAllString(page[h[4]:h[5]], " ")))
		m := ruGradeRE.FindStringSubmatch(name)
		if m == nil {
			continue
		}
		end := len(page)
		if i+1 < len(heads) {
			end = heads[i+1][0]
		}
		s := RuSection{HTML: page[h[1]:end]}
		s.Grade, _ = strconv.Atoi(m[1])
		s.Day, _ = strconv.Atoi(m[2])
		if seen[s.Prefix()] {
			continue
		}
		seen[s.Prefix()] = true
		out = append(out, s)
	}
	return out
}

// ruMarkRE gives the start of problem N of a paper: a new paragraph, line or item, then N and a dot, maybe in bold and after "Задача" or "Задача №".
func ruMarkRE(n int) *regexp.Regexp {
	return regexp.MustCompile(`(?is)(?:<p\b[^>]*>|<br\s*/?>|<li\b[^>]*>|<dt\b[^>]*>|^)\s*(?:<a\b[^>]*>\s*</a>\s*)?(?:<(?:b|strong)>\s*)?(?:Задача\s*(?:№|&#8470;|&#x2116;)?\s*)?` + strconv.Itoa(n) + `\s*\.\s*(?:</(?:b|strong)>)?`)
}

// RuNumbered cuts the HTML of one paper into its problems, numbered from 1. Problem N+1 must come after problem N, so that a number in the text of a problem does not start a new one. The texts are HTML.
func RuNumbered(s string) []string {
	var marks [][]int
	from := 0
	for n := 1; n <= 20; n++ {
		loc := ruMarkRE(n).FindStringIndex(s[from:])
		if loc == nil {
			break
		}
		marks = append(marks, []int{from + loc[0], from + loc[1]})
		from += loc[1]
	}
	var out []string
	for i, m := range marks {
		end := len(s)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}
		out = append(out, s[m[1]:end])
	}
	return out
}

// RuPage reads the problems of a page that has a heading for each grade paper, as the year pages of the Moscow olympiad. edition is the edition part of the locator and url the address of the page. grades are the grades that have positions; a paper of another grade is left out.
func RuPage(page, edition, url string, grades func(int) bool) []RuProblem {
	var out []RuProblem
	for _, s := range RuGrades(page) {
		if !grades(s.Grade) {
			continue
		}
		for i, h := range RuNumbered(s.HTML) {
			text, fig := RuHTML(h)
			p := RuProblem{Edition: edition, Position: s.Prefix() + strconv.Itoa(i+1), Text: text, URL: url, Figure: fig}
			if RuLayout(h) {
				p.Flags = append(p.Flags, "layout")
			}
			out = append(out, p)
		}
	}
	return out
}
