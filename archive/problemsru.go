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
	Text   string
	// Err tells why the problem has no Markdown form. Text is empty when Err is not nil.
	Err error
}

var (
	// prNodeRE is a child node of the header box, for example <a class="componentboxlink" href="/view_by_source_new.php?parent=216670">8 класс</a> (6 задач).
	prNodeRE = regexp.MustCompile(`<a class="componentboxlink" href="/view_by_source_new\.php\?parent=(\d+)">\s*([^<]*?)\s*</a>\s*\((\d+) задач`)
	// prCrumbRE is a link of the path at the top of a node page.
	prCrumbRE = regexp.MustCompile(`<a class="componentboxheaderlink" href="/view_by_source_new\.php\?parent=(\d+)">\s*([^<]*?)\s*</a>`)
	prTotalRE = regexp.MustCompile(`Всего задач: (\d+)`)
	// prHeadRE is the head of a problem of a listing, for example Задача <a ...id=67463">67463</a>&nbsp;(#1).
	prHeadRE = regexp.MustCompile(`(?s)<td class="problemsmallnumber">.*?view_problem_details_new\.php\?id=(\d+)">\s*\d+</a>\s*(?:&nbsp;\(#(\d+)\))?`)
	prBodyRE = regexp.MustCompile(`(?s)<td class="problemsmalldifficulty">.*?</table>(.*?)<table class="problemfootertable">`)
	// prAuthorRE is the line of the author above a statement.
	prAuthorRE = regexp.MustCompile(`(?s)<div class="catalogueproblemauthor[^"]*">.*?</div>`)
	prImageRE  = regexp.MustCompile(`(?i)<img\b`)
	prYearRE   = regexp.MustCompile(`^(\d{4}) год$`)
	prPaperRE  = regexp.MustCompile(`^(\d{1,2}) класс(?:\. (Первый|Второй) день)?$`)
)

// errPRFigure tells that a statement of problems.ru has a picture. The pictures are under a path that robots.txt of the site disallows.
var errPRFigure = errors.New("the problem has a picture")

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

// PRYear reads the year of a year node, for example "2025 год".
func PRYear(name string) (int, bool) {
	m := prYearRE.FindStringSubmatch(strings.TrimSpace(name))
	if m == nil {
		return 0, false
	}
	y, _ := strconv.Atoi(m[1])
	return y, true
}

// PRPaper reads the grade and the day of a paper node, for example "8 класс" or "11 класс. Второй день". The day is 0 for a paper of one day.
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

// ParsePRListing reads the problems of one page of the listing of a node. A statement keeps its TeX. The author line, the topics and the difficulty are not part of it: the topics and the difficulty are the work of MCCME and stay on problems.ru.
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
		if h[4] >= 0 {
			p.Number, _ = strconv.Atoi(page[h[4]:h[5]])
		}
		b := prBodyRE.FindStringSubmatch(page[h[1]:end])
		if b == nil {
			p.Err = errors.New("the listing has no statement")
			out = append(out, p)
			continue
		}
		body := prAuthorRE.ReplaceAllString(b[1], "")
		if prImageRE.MatchString(body) {
			p.Err = errPRFigure
			out = append(out, p)
			continue
		}
		body = komalItalic.ReplaceAllString(body, "\x02$1\x02")
		if p.Text = htmlText(body); p.Text == "" {
			p.Err = errors.New("the statement is empty")
		}
		out = append(out, p)
	}
	return out
}
