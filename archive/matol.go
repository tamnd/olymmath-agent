package archive

import (
	"html"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"
)

// MatolBase is the site of the Al-Farabi olympiad centre of Kazakhstan, which gives the papers of the Kazakhstan olympiads in Russian (source matol-kz).
const MatolBase = "https://matol.kz"

// MatolRoot is a node of the site whose papers are the papers of a competition of the register.
type MatolRoot struct {
	Node string
	Code string
}

// MatolRoots are the nodes of the site that om matol reads, with the competition of each. The other nodes, as the international olympiads that the site gives in translation, are not read.
var MatolRoots = []MatolRoot{
	{Node: "/nodes/11", Code: "kz-mo-district"},
	{Node: "/nodes/12", Code: "kz-mo-regional"},
	{Node: "/nodes/468", Code: "kz-junior"},
	{Node: "/nodes/157", Code: "kz-smagulov"},
}

// MatolCrumb is a link of the breadcrumb of a page.
type MatolCrumb struct {
	Path string
	Name string
}

// MatolProblem is a problem of a paper page.
type MatolProblem struct {
	// N is the number of the problem in the paper, as 3 for Задача №3.
	N int
	// Comment is the id of the page of comments and solutions of the problem.
	Comment string
	// Body is the HTML of the statement, with the authors taken out.
	Body string
	// Authors are the names of the authors that the page gives, as Сатылханов К.
	Authors []string
}

// MatolPage is a page of the site: a node, which lists nodes and papers, or a paper.
type MatolPage struct {
	// Crumbs are the links of the breadcrumb, from the top. The page itself is the leaf.
	Crumbs []MatolCrumb
	Leaf   string
	// Links are the paths of the nodes and papers that the table of a node lists.
	Links []string
	// Problems are the problems of a paper.
	Problems []MatolProblem
}

var (
	matolCrumbsRE  = regexp.MustCompile(`(?s)<ol class="breadcrumb">(.*?)</ol>`)
	matolCrumbRE   = regexp.MustCompile(`(?s)<a href="([^"]+)">(.*?)</a>`)
	matolLeafRE    = regexp.MustCompile(`(?s)<li class="active">(.*?)</li>`)
	matolTableRE   = regexp.MustCompile(`(?s)<table class="table table-striped">(.*?)</table>`)
	matolLinkRE    = regexp.MustCompile(`href="(/(?:nodes|olympiads)/[0-9]+)"`)
	matolProblemRE = regexp.MustCompile(`(?s)<b>\s*Задача\s*№\s*([0-9]+)\.?\s*</b>(.*?)<br/>\s*<a href="/comments/([0-9]+)/show"`)
	matolAuthorRE  = regexp.MustCompile(`(?s)<font size="2">\s*\((.*?)\)\s*</font>`)
	matolNameRE    = regexp.MustCompile(`(?s)<a [^>]*>(.*?)</a>`)
	matolTagRE     = regexp.MustCompile(`<[^>]+>`)
	matolSpaceRE   = regexp.MustCompile(`\s+`)
)

// ParseMatol reads a page of the site.
func ParseMatol(b []byte) MatolPage {
	s := string(b)
	var p MatolPage
	if m := matolCrumbsRE.FindStringSubmatch(s); m != nil {
		for _, c := range matolCrumbRE.FindAllStringSubmatch(m[1], -1) {
			p.Crumbs = append(p.Crumbs, MatolCrumb{Path: c[1], Name: matolPlain(c[2])})
		}
		if l := matolLeafRE.FindStringSubmatch(m[1]); l != nil {
			p.Leaf = matolPlain(l[1])
		}
	}
	if m := matolTableRE.FindStringSubmatch(s); m != nil {
		seen := map[string]bool{}
		for _, l := range matolLinkRE.FindAllStringSubmatch(m[1], -1) {
			if !seen[l[1]] {
				seen[l[1]] = true
				p.Links = append(p.Links, l[1])
			}
		}
	}
	for _, m := range matolProblemRE.FindAllStringSubmatch(s, -1) {
		n, _ := strconv.Atoi(m[1])
		pr := MatolProblem{N: n, Comment: m[3], Body: m[2]}
		if a := matolAuthorRE.FindStringSubmatchIndex(pr.Body); a != nil {
			for _, x := range matolNameRE.FindAllStringSubmatch(pr.Body[a[2]:a[3]], -1) {
				if name := matolPlain(x[1]); name != "" {
					pr.Authors = append(pr.Authors, name)
				}
			}
			pr.Body = pr.Body[:a[0]] + pr.Body[a[1]:]
		}
		p.Problems = append(p.Problems, pr)
	}
	return p
}

// matolPlain gives the text of a piece of HTML on one line.
func matolPlain(s string) string {
	return strings.TrimSpace(matolSpaceRE.ReplaceAllString(html.UnescapeString(matolTagRE.ReplaceAllString(s, "")), " "))
}

// MatolRootOf gives the root of the register that a page is under, by its breadcrumb, and the names of the crumbs below the root.
func MatolRootOf(p MatolPage) (MatolRoot, []string, bool) {
	for i, c := range p.Crumbs {
		for _, r := range MatolRoots {
			if c.Path == r.Node {
				var mid []string
				for _, m := range p.Crumbs[i+1:] {
					mid = append(mid, m.Name)
				}
				return r, mid, true
			}
		}
	}
	return MatolRoot{}, nil, false
}

var (
	matolTextRules = []struct {
		re   *regexp.Regexp
		with string
	}{
		{regexp.MustCompile(`(?i)</?br\s*/?>`), "\n"},
		{regexp.MustCompile(`(?i)</?(?:p|div)[^>]*>`), "\n"},
		{regexp.MustCompile(`(?i)</?(?:b|strong)>`), "**"},
		{regexp.MustCompile(`(?i)</?(?:i|em)>`), "*"},
		{regexp.MustCompile(`(?is)<sup>(.*?)</sup>`), "$$^{${1}}$$"},
		{regexp.MustCompile(`(?is)<sub>(.*?)</sub>`), "$$_{${1}}$$"},
	}
	matolMarkupRE = regexp.MustCompile(`<[a-zA-Z/]`)
	matolEmptyRE  = regexp.MustCompile(`\*\*\s*\*\*`)
	matolBlankRE  = regexp.MustCompile(`\n{3,}`)
)

// MatolText gives the Markdown of the statement of a problem, or the reason why it has none: a figure, which the page gives as an image, markup that does not convert, or a text too short to be a statement. The TeX of the page is kept as it is.
func MatolText(pr MatolProblem) (string, string) {
	s := pr.Body
	if strings.Contains(s, "<img") {
		return "", "a figure"
	}
	for _, r := range matolTextRules {
		s = r.re.ReplaceAllString(s, r.with)
	}
	if matolMarkupRE.MatchString(s) {
		return "", "markup that does not convert"
	}
	s = strings.ReplaceAll(html.UnescapeString(s), "\u00a0", " ")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(l)
	}
	s = strings.Join(lines, "\n")
	s = matolEmptyRE.ReplaceAllString(s, "")
	s = strings.TrimSpace(matolBlankRE.ReplaceAllString(s, "\n\n"))
	if utf8.RuneCountInString(s) < 15 {
		return "", "a text too short to be a statement"
	}
	return s, ""
}

var (
	matolSchoolRE = regexp.MustCompile(`([0-9]{4})\s*-\s*([0-9]{4})`)
	matolYearRE   = regexp.MustCompile(`[0-9]{4}`)
	matolGradeRE  = regexp.MustCompile(`([0-9]+)\s*класс`)
	matolTourRE   = regexp.MustCompile(`([0-9]+)\s*(?:тур|день)`)
)

// MatolPlace gives the competition code, the edition year and the position of problem n of a paper page, or the reason why it has no place. The edition is the calendar year of the paper (spec 01f): the district stage is written in the autumn, so it is the first year of the school year, and the other stages and olympiads are written in the spring, so they are its last year.
//
// The position is the grade and the number, after a letter for the round:
//   - the district and regional stages: p, as p93 for problem 3 of grade 9;
//   - the junior olympiad: d, r and f for its district, regional and final stage, as r72; the test round of the final stage of 2021 has no position;
//   - the Smagulov olympiad: a, b and c for its three rounds, as b64, and t for the team round of the grades 6 and 7 together, as t5.
func MatolPlace(p MatolPage, n int) (code string, year int, number, why string) {
	root, mid, ok := MatolRootOf(p)
	if !ok {
		return "", 0, "", "not under a node of the register"
	}
	label := strings.Join(mid, " ")
	if label == "" {
		label = p.Leaf
	}
	if m := matolSchoolRE.FindStringSubmatch(label); m != nil {
		first, _ := strconv.Atoi(m[1])
		last, _ := strconv.Atoi(m[2])
		year = last
		if root.Code == "kz-mo-district" {
			year = first
		}
	} else if m := matolYearRE.FindString(label); m != "" {
		year, _ = strconv.Atoi(m)
	} else {
		return "", 0, "", "no year"
	}
	leaf := strings.ToLower(p.Leaf)
	grade, tour := "", ""
	if m := matolGradeRE.FindStringSubmatch(leaf); m != nil {
		grade = m[1]
	}
	if m := matolTourRE.FindStringSubmatch(leaf); m != nil {
		tour = m[1]
	}
	num := strconv.Itoa(n)
	switch root.Code {
	case "kz-mo-district", "kz-mo-regional":
		if tour != "" {
			return "", 0, "", "a round that the position cannot tell"
		}
		if grade == "" {
			return "", 0, "", "no grade"
		}
		return root.Code, year, "p" + grade + num, ""
	case "kz-junior":
		if grade == "" {
			return "", 0, "", "no grade"
		}
		switch {
		case strings.Contains(leaf, "тестовый тур"):
			return "", 0, "", "the test round of the final stage of the junior olympiad"
		case strings.Contains(leaf, "районный"):
			return root.Code, year, "d" + grade + num, ""
		case strings.Contains(leaf, "областной"):
			return root.Code, year, "r" + grade + num, ""
		case strings.Contains(leaf, "заключительн"):
			return root.Code, year, "f" + grade + num, ""
		}
		return "", 0, "", "no stage of the junior olympiad"
	case "kz-smagulov":
		if strings.Contains(leaf, "командн") {
			return root.Code, year, "t" + num, ""
		}
		letter := map[string]string{"1": "a", "2": "b", "3": "c"}[tour]
		if letter == "" || grade == "" {
			return "", 0, "", "no grade and round"
		}
		return root.Code, year, letter + grade + num, ""
	}
	return "", 0, "", "no rule for " + root.Code
}
