package archive

import (
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
)

// ProblemsRuBase is the address of problems.ru, the problem base of MCCME.
const ProblemsRuBase = "https://problems.ru"

// PRNode is a node of the source catalogue of problems.ru: an olympiad, a year or a paper of a year.
type PRNode struct {
	ID    int
	Name  string
	Count int
}

// PRProblem is a problem of the listing of a paper node of problems.ru.
type PRProblem struct {
	// ID is the unique number of the problem on problems.ru.
	ID int
	// Number is the number of the problem in its paper, or 0 when the listing does not give it.
	Number int
	// Year and Grade are given by a listing that writes the number as year.grade.number, such as (#2010.8.1), or as grade.number, such as (#8.1). They are 0 when the listing does not give them.
	Year, Grade int
	Text        string
	// Figure tells that the statement has a picture. Text is then the statement without the picture, so it is good for a link and a match, not for a copy.
	Figure bool
	// Err tells why the problem has no Markdown form. Text is empty when Err is not nil.
	Err error
}

var (
	// prNodeRE is a child node of the header box, for example <a class="componentboxlink" href="/view_by_source_new.php?parent=216670">8 класс</a> (6 задач).
	prNodeRE = regexp.MustCompile(`<a class="componentboxlink" href="/view_by_source_new\.php\?parent=(\d+)">\s*([^<]*?)\s*</a>\s*\((\d+) задач`)
	// prCrumbRE is a link of the path at the top of a node page.
	prCrumbRE = regexp.MustCompile(`<a class="componentboxheaderlink" href="/view_by_source_new\.php\?parent=(\d+)">\s*([^<]*?)\s*</a>`)
	prTotalRE = regexp.MustCompile(`Всего задач: (\d+)`)
	// prHeadRE is the head of a problem of a listing, for example Задача <a ...id=67463">67463</a>&nbsp;(#1). Some listings write the number as year.grade.number, for example (#2010.8.1), or as grade.number, for example (#8.1).
	prHeadRE = regexp.MustCompile(`(?s)<td class="problemsmallnumber">.*?view_problem_details_new\.php\?id=(\d+)">\s*\d+</a>\s*(?:&nbsp;\(#(?:(\d{4})\.)?(?:(\d{1,2})\.)?(\d+)\))?`)
	prBodyRE = regexp.MustCompile(`(?s)<td class="problemsmalldifficulty">.*?</table>(.*?)<table class="problemfootertable">`)
	// prAuthorRE is the line of the author above a statement.
	prAuthorRE = regexp.MustCompile(`(?s)<div class="catalogueproblemauthor[^"]*">.*?</div>`)
	prImageRE  = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	prYearRE   = regexp.MustCompile(`^(\d{4}) год$|^[IVXLC]+ .*\((\d{4}) г\.\)$|^\d{2} \((\d{4})\), математика$`)
	prPaperRE  = regexp.MustCompile(`^(\d{1,2}) [кК]ласс(?:\. (Первый|Второй) день)?$`)
	// prSpanRE is a year node of an olympiad whose year runs over a school year: "46 турнир (2024/2025 год)" or "1 турнир (1980 год)" of the Tournament of Towns, or "1992-1993" of the All-Russian olympiad.
	prSpanRE = regexp.MustCompile(`^\d+ турнир \((\d{4})(?:/(\d{4}))? год\)$|^(\d{4})-(\d{4})$`)
	// prTourPaperRE is a paper of the Tournament of Towns: the tour, the level and the grades, as in "осенний тур, базовый вариант, 8-9 класс".
	prTourPaperRE = regexp.MustCompile(`^(осенний|весенний) тур, (базовый|тренировочный|подготовительный|сложный|основной) вариант, (7-8|8-9|9-10|10-11) класс$`)
)

// PRNodeURL gives the page of a node of the source catalogue, from the problem start of its listing.
func PRNodeURL(id, start int) string {
	if start == 0 {
		return fmt.Sprintf("%s/view_by_source_new.php?parent=%d", ProblemsRuBase, id)
	}
	return fmt.Sprintf("%s/view_by_source_new.php?parent=%d&start=%d", ProblemsRuBase, id, start)
}

// PRProblemURL gives the page of a problem, which carries its unique number. The terms of problems.ru ask a site that shows a problem to link to it and give that number.
func PRProblemURL(id int) string {
	return fmt.Sprintf("%s/view_problem_details_new.php?id=%d", ProblemsRuBase, id)
}

// PRNodes gives the child nodes of a node page, with the number of problems of each.
func PRNodes(page string) []PRNode {
	var out []PRNode
	for _, m := range prNodeRE.FindAllStringSubmatch(page, -1) {
		id, _ := strconv.Atoi(m[1])
		n, _ := strconv.Atoi(m[3])
		out = append(out, PRNode{ID: id, Name: html.UnescapeString(m[2]), Count: n})
	}
	return out
}

// PRName gives the name of the node of a node page: the last link of the path at the top.
func PRName(page string) string {
	ms := prCrumbRE.FindAllStringSubmatch(page, -1)
	if len(ms) == 0 {
		return ""
	}
	return html.UnescapeString(ms[len(ms)-1][2])
}

// PRTotal gives the number of problems of the listing of a node page.
func PRTotal(page string) int {
	if m := prTotalRE.FindStringSubmatch(page); m != nil {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 0
}

// PRYear reads the year of a year node, for example "2025 год", "XV Олимпиада по геометрии имени И.Ф. Шарыгина (2019 г.)" for an olympiad that numbers its years, or "44 (2021), математика" for the mathematics of the Lomonosov tournament. A year of the Lomonosov tournament that does not name its subject, or names another one such as "математические игры", gives false.
func PRYear(name string) (int, bool) {
	m := prYearRE.FindStringSubmatch(strings.TrimSpace(name))
	if m == nil {
		return 0, false
	}
	y, _ := strconv.Atoi(m[1] + m[2] + m[3])
	return y, true
}

// PRPaper reads the grade and the day of a paper node, for example "8 класс", "8 Класс" or "11 класс. Второй день". The day is 0 for a paper of one day.
func PRPaper(name string) (grade, day int, ok bool) {
	m := prPaperRE.FindStringSubmatch(strings.TrimSpace(name))
	if m == nil {
		return 0, 0, false
	}
	grade, _ = strconv.Atoi(m[1])
	switch m[2] {
	case "Первый":
		day = 1
	case "Второй":
		day = 2
	}
	return grade, day, true
}

// PRSpan reads the years of a year node that runs over a school year, for example 2024 and 2025 for "46 турнир (2024/2025 год)" or "2024-2025". A tournament of one year gives that year two times.
func PRSpan(name string) (first, last int, ok bool) {
	m := prSpanRE.FindStringSubmatch(strings.TrimSpace(name))
	if m == nil {
		return 0, 0, false
	}
	first, _ = strconv.Atoi(m[1] + m[3])
	last = first
	if m[2]+m[4] != "" {
		last, _ = strconv.Atoi(m[2] + m[4])
	}
	return first, last, last == first || last == first+1
}

// PRFinal tells that a paper node is the final round of the All-Russian olympiad, "Заключительный этап", whose child nodes are the papers of the grades.
func PRFinal(name string) bool {
	return strings.TrimSpace(name) == "Заключительный этап"
}

// PRTour is a paper of the Tournament of Towns.
type PRTour struct {
	// Spring is true for the spring tour (весенний тур) and false for the autumn tour (осенний тур).
	Spring bool
	// Senior is true for the older grades, 10-11 (9-10 before 1990), and false for the younger grades, 8-9 (7-8 before 1990).
	Senior bool
	// Advanced is true for the hard level: сложный вариант, and основной вариант in the early years. It is false for the base level: базовый, тренировочный or подготовительный вариант.
	Advanced bool
}

// Prefix gives the start of the positions of the paper: j or s for the tier, then o or a for the level, as in the English names of the papers (Junior O-Level, Senior A-Level). Problem 3 of the senior A-level paper is sa3.
func (t PRTour) Prefix() string {
	p := "j"
	if t.Senior {
		p = "s"
	}
	if t.Advanced {
		return p + "a"
	}
	return p + "o"
}

// PRTourPaper reads a paper node of the Tournament of Towns, for example "весенний тур, сложный вариант, 10-11 класс". A paper that does not name the tour, one of the two levels and the grades of one tier, such as "устный тур" or "весенний тур, вариант для москвичей, 7-8 класс", gives false.
func PRTourPaper(name string) (PRTour, bool) {
	m := prTourPaperRE.FindStringSubmatch(strings.TrimSpace(name))
	if m == nil {
		return PRTour{}, false
	}
	return PRTour{Spring: m[1] == "весенний", Senior: m[3] == "9-10" || m[3] == "10-11", Advanced: m[2] == "сложный" || m[2] == "основной"}, true
}

// ParsePRListing reads the problems of one page of the listing of a node. A statement keeps its TeX. The author line, the topics and the difficulty are not part of it: the topics and the difficulty are the work of MCCME and stay on problems.ru. A picture is left out of the text and is never fetched, because its path (/show_document.php?) is disallowed by robots.txt; the problem gets Figure.
func ParsePRListing(page string) []PRProblem {
	var out []PRProblem
	heads := prHeadRE.FindAllStringSubmatchIndex(page, -1)
	for i, h := range heads {
		end := len(page)
		if i+1 < len(heads) {
			end = heads[i+1][0]
		}
		p := PRProblem{}
		p.ID, _ = strconv.Atoi(page[h[2]:h[3]])
		if h[8] >= 0 {
			p.Number, _ = strconv.Atoi(page[h[8]:h[9]])
		}
		if h[4] >= 0 {
			p.Year, _ = strconv.Atoi(page[h[4]:h[5]])
		}
		if h[6] >= 0 {
			p.Grade, _ = strconv.Atoi(page[h[6]:h[7]])
		}
		b := prBodyRE.FindStringSubmatch(page[h[1]:end])
		if b == nil {
			p.Err = errors.New("the listing has no statement")
			out = append(out, p)
			continue
		}
		body := prAuthorRE.ReplaceAllString(b[1], "")
		if prImageRE.MatchString(body) {
			p.Figure = true
			body = prImageRE.ReplaceAllString(body, " ")
		}
		body = komalItalic.ReplaceAllString(body, "\x02$1\x02")
		if p.Text = htmlText(body); p.Text == "" {
			p.Err = errors.New("the statement is empty")
		}
		out = append(out, p)
	}
	return out
}

// PRSource is one use of a problem in an olympiad, read from the table of sources of its own page.
type PRSource struct {
	Year, Grade, Number int
	// Other is true when the source names more than the year, the grade and the number, for example a day or a variant. Such a source does not tell the paper well enough.
	Other bool
}

var (
	// prSourceRowRE is a row of the table of sources: a head with a node link and its name, or a key and its value.
	prSourceRowRE = regexp.MustCompile(`(?s)<td class="problemdetailssourcetablecellhead" colspan=2>\s*<a class="componentboxlink" href="/view_by_source_new\.php\?parent=(\d+)">([^<]*)</a>|<td class="problemdetailssourcetablecell">\s*([^<]*?)\s*</td>\s*<td class="problemdetailssourcetablecell">\s*([^<]*?)\s*</td>|<td class="problemdetailssourcetablecellhead" colspan=2>\s*([^<]*?)\s*</td>`)
	// prGradeNumberRE is a number of a problem written as grade.number, for example 10.4.
	prGradeNumberRE = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})$`)
)

// ParsePRSources reads the table of sources of the page of a problem and gives each use of the problem in the olympiad of node. A use is a head that links to node, then the heads год, вариант (or класс) and задача, which give Год, Класс and Номер. A Номер written as grade.number, such as 10.4, must name the grade of the use. Anything else before Номер makes the use Other, and anything after it ends the use.
func ParsePRSources(page string, node int) []PRSource {
	var out []PRSource
	var cur *PRSource
	section, done := "", false
	for _, m := range prSourceRowRE.FindAllStringSubmatch(page, -1) {
		if m[1] != "" || m[3] == "" {
			id, _ := strconv.Atoi(m[1])
			name := strings.TrimSpace(m[2] + m[5])
			switch {
			case id == node:
				out = append(out, PRSource{})
				cur, section, done = &out[len(out)-1], "", false
			case cur == nil || done:
				cur = nil
			case name == "год" || name == "вариант" || name == "класс" || name == "задача":
				section = name
			default:
				cur.Other = true
			}
			continue
		}
		if cur == nil || done {
			continue
		}
		v, err := strconv.Atoi(m[4])
		switch {
		case section == "" && m[3] == "Название":
		case section == "год" && m[3] == "Номер":
		case section == "год" && m[3] == "Год" && err == nil:
			cur.Year = v
		case (section == "вариант" || section == "класс") && m[3] == "Класс" && err == nil:
			cur.Grade = v
		case section == "задача" && m[3] == "Номер" && err == nil:
			cur.Number, done = v, true
		case section == "задача" && m[3] == "Номер" && prGradeNumberRE.MatchString(m[4]):
			gn := prGradeNumberRE.FindStringSubmatch(m[4])
			g, _ := strconv.Atoi(gn[1])
			cur.Number, done = 0, true
			if g == cur.Grade {
				cur.Number, _ = strconv.Atoi(gn[2])
			} else {
				cur.Other = true
			}
		default:
			cur.Other = true
		}
	}
	return out
}
