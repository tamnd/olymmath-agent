package official

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// sharyginHeadRE is the heading of a paper of the final round of the Sharygin olympiad: "Финал. Первый день. 8 класс", "Финал. Второй день. 10–11 классы" or "9 класс. Первый день". The first grade names the paper.
	sharyginHeadRE = regexp.MustCompile(`(?m)^(?:Финал\.\s*(Первый|Второй)\s+день\.\s*(\d{1,2})(?:\s*[-–]\s*\d{1,2})?\s+класс\S*|(\d{1,2})(?:\s*[-–]\s*\d{1,2})?\s+класс\S*\.\s*(Первый|Второй)\s+день)`)
	// sharyginSolution is the start of the solution after a problem, in a paper with solutions.
	sharyginSolution = regexp.MustCompile(`(?:^|\s)(?:Ответ|Решение|Первое решение)\.`)
	// sharyginSkip is a line of a header of a page.
	sharyginSkip = regexp.MustCompile(`(?m)^(?:.*[Оо]лимпиада по геометрии.*|им\. И\. ?Ф\. ?Шарыгина|Ратмино,.*|Финальный тур,.*|Условия задач)\n`)
	// sharyginAuthor is the author in brackets at the start of a problem, as (Б.Френкин) or (M.Saghafian).
	sharyginAuthor = regexp.MustCompile(`^\(\s*[\p{L}.,\s-]{2,80}\)\s*`)
	// sharyginIndex is a letter with a digit after it, as A1 or ω2, which is a letter with an index in a paper of geometry.
	sharyginIndex = regexp.MustCompile(`([A-Za-zα-ωΩ])(\d{1,2})`)
)

// sharyginMarkRE gives the start of problem N of a paper of grade G: N or G.N, and a dot, at the start of a line.
func sharyginMarkRE(g string, n int) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^(?:` + g + `\.)?` + strconv.Itoa(n) + `\.\s+`)
}

// sharyginMark gives the first start of a problem in s that re finds, after a line that ends a sentence or a word. After a line that ends with a sign, as "√3 :", a number such as "2." is a part of the formula and not the start of a problem.
func sharyginMark(s string, re *regexp.Regexp) []int {
	for _, loc := range re.FindAllStringIndex(s, -1) {
		prev := strings.TrimRight(s[:loc[0]], " \n")
		if prev == "" || !strings.ContainsAny(prev[len(prev)-1:], ":=+-,(") && !strings.HasSuffix(prev, "−") && !strings.HasSuffix(prev, "·") && !strings.HasSuffix(prev, "×") {
			return loc
		}
	}
	return nil
}

// SharyginFinal reads the problems of the final round of the Sharygin olympiad from its paper in Russian. pages are the texts of the pages, as pdftotext -raw gives them. Each grade has a paper for each of the two days, and the problems of a grade have the numbers 1 to 8 across the two days, so problem N of grade G has the position pGN. The first paper of a grade and a day counts. In a paper with solutions, the text of a problem ends where its solution starts.
func SharyginFinal(pages []string, edition, url string) []RuProblem {
	all, starts := pdfJoin(pages)
	heads := sharyginHeadRE.FindAllStringSubmatchIndex(all, -1)
	seen := map[string]bool{}
	var out []RuProblem
	for i, h := range heads {
		m := sharyginHeadRE.FindStringSubmatch(all[h[0]:h[1]])
		day, grade := m[1], m[2]
		if grade == "" {
			day, grade = m[4], m[3]
		}
		first := 1
		if day == "Второй" {
			first = 5
		}
		key := grade + "/" + day
		if seen[key] {
			continue
		}
		seen[key] = true
		end := len(all)
		if i+1 < len(heads) {
			end = heads[i+1][0]
		}
		paper := all[h[1]:end]
		var marks [][]int
		from := 0
		for n := first; n < first+4; n++ {
			loc := sharyginMark(paper[from:], sharyginMarkRE(grade, n))
			if loc == nil {
				break
			}
			marks = append(marks, []int{from + loc[0], from + loc[1]})
			from += loc[1]
		}
		for j, mk := range marks {
			e := len(paper)
			if j+1 < len(marks) {
				e = marks[j+1][0]
			}
			raw := paper[mk[1]:e]
			if s := sharyginSolution.FindStringIndex(raw); s != nil {
				raw = raw[:s[0]]
			}
			raw = sharyginSkip.ReplaceAllString(raw+"\n", "")
			raw = sharyginAuthor.ReplaceAllString(raw, "")
			raw = sharyginIndex.ReplaceAllStringFunc(raw, func(s string) string {
				g := sharyginIndex.FindStringSubmatch(s)
				if len(g[2]) == 1 {
					return g[1] + "_" + g[2]
				}
				return g[1] + "_{" + g[2] + "}"
			})
			text, fig := PDFText(raw)
			out = append(out, RuProblem{Edition: edition, Position: "p" + grade + strconv.Itoa(first+j), Text: text, URL: url, Page: pdfPage(starts, h[1]+mk[0]), Figure: fig || turlomFigure.MatchString(text)})
		}
	}
	return out
}
