package official

import (
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// The marks of totLines: the end of a block of the page, as a rule or the end of a <pre>, and a picture.
const (
	totEnd = "\x04"
	totImg = "\x03"
	// totArtLine stands for a formula drawn in characters.
	totArtLine = "\x06"
)

// totNames is one or more names of authors, as А. В. Шаповалов or Д.Баранов, И.Ященко.
// totName is one author, as А. Я. Канель-Белов, with the part of the problem that is the author's, as "— п. а)".
const totName = `(?:\p{Lu}\.\s?){1,2}\s?\p{Lu}\p{Ll}+(?:-\p{Lu}?\p{Ll}+)?(?:\s*—\s*п\.\s*[а-ж]\))?`

// totNames is the authors of a problem, as "фольклор, предложил А. Бердников" or "Венгерский фольклор".
const totNames = `(?:(?:\p{Lu}\p{Ll}+\s)?фольклор(?:,\s*предложил\s+` + totName + `)?|` + totName + `(?:,\s*` + totName + `)*)`

var (
	// totInHyphen is a word cut in two in the line, as считает- ся, which the source kept from print.
	totInHyphen   = regexp.MustCompile(`\p{L}+- \p{L}+`)
	totPtsCell    = regexp.MustCompile(`(?i)<td align="center">(?:\s|&nbsp;)*\d{1,2}(?:\s*\+\s*\d{1,2})*(?:\s|&nbsp;)*</td>`)
	totAuthorCell = regexp.MustCompile(`(?i)<td[^>]*>\s*(?:<div[^>]*>\s*)?<i>[^<]*</i>\s*(?:</div>\s*)?</td>`)
	// totLetterWord is a number with a letter before a word, as 2n-угольник.
	totLetterWord = regexp.MustCompile(`(^|[\s(])(\d*[a-zA-Z])-`)
	// totDots is dots between two formulas, as $a_1+$ ... $+a_k$.
	totDots = regexp.MustCompile(`\$(\s*[<>=+-]?\s*)(?:\.\.\.|…)(\s*[<>=+-]?\s*)\$`)
	// totLoose is math that the text left out of a formula, or an empty pair of brackets where a formula was a picture.
	totLoose  = regexp.MustCompile(`[\^_<>=\\{}]|\b[a-zA-Z]\s*[+*/]\s*\w|\w\s*[+*/]\s*[a-zA-Z]\b|\(\s*\)`)
	totInWord = regexp.MustCompile(`\p{L}+-\p{L}+`)
	totPre    = regexp.MustCompile(`(?is)<pre\b[^>]*>(.*?)</pre>`)
	// totPrePapers is a <pre> that holds papers: it has numbered problems or a title.
	totPrePapers = regexp.MustCompile(`(?m)^\s*(?:Задача\s*)?\d{1,2}\s*\.|^\s*Задача\s*\d|ТУРНИР`)
	totHR        = regexp.MustCompile(`(?i)<hr\b[^>]*>`)
	// totBoth is a heading of one paper for both age groups, as "8 - 9 и 10 - 11 классы".
	totBoth = regexp.MustCompile(`\d\s*(?:--|[–—-])\s*9\s+и\s+10`)
	// totFall and totSpring are the months of the two tours.
	totFall   = regexp.MustCompile(`(?i)сентябр|октябр|ноябр|декабр|осенн`)
	totSpring = regexp.MustCompile(`(?i)январ|феврал|март|апрел|весенн`)
	// totNumLine is the start of a problem: "1. Text", "Задача 1.(2+2)", "Задача 2(5)", "7  а) Text" or "3." alone.
	totNumLine = regexp.MustCompile(`^(\s*)(Задача\s*)?(\d{1,2})(\s*\.\s*|\s*)`)
	// totPtsCol is the points of a problem in the column left of the numbers.
	totPtsCol = regexp.MustCompile(`^\s*\d{1,2}(?:\s*\+\s*\d{1,2})*(?:\s|$)`)
	// totPtsIn is the points of a problem or a part in the text, as (2+2), (3 балла). or [5 баллов].
	totPtsIn = regexp.MustCompile(`^\s*[(\[]\s*\d{1,2}(?:\s*\+\s*\d{1,2})*\s*(?:бал\p{L}*)?\s*[)\]]\.?\s*`)
	// totSub is the start of a part of a problem, as а) or 2).
	totSub       = regexp.MustCompile(`^(?:[а-жa-h]|\d)\)`)
	totLetterSub = regexp.MustCompile(`^[а-ж]\)`)
	// totRule is a line of dashes that parts the papers of a <pre>.
	totRule = regexp.MustCompile(`^-{5,}$`)
	// totAuthorLine is the name of an author on a line of its own, as (Н.И.Авилов) or В. А. Сендеров. totAuthorEnd is the names at the end of a text.
	totAuthorLine = regexp.MustCompile(`^\([^()]*\)$|^` + totNames + `$`)
	totAuthorEnd  = regexp.MustCompile(`\s+\(?` + totNames + `\)?\.?$`)
	// totArt is a part of a line that draws a formula or a picture in characters.
	totArt = regexp.MustCompile(`-{3,}|_{3,}|\|\s+\||\\\s+/`)
	// totVerbal is math in words or a sign that the text cannot hold, so the text goes to a person.
	totVerbal = regexp.MustCompile(`(?i)корень из|корн[яе]м? из|квадратн\p{L}* кор|sqrt|в степени|дробь`)
	// totLike is a formula in the text, as А_1, х_2=b or АК=АО, in which a Cyrillic letter may stand for a Latin one.
	totLike = regexp.MustCompile(`[\p{L}\d_^'\x01\x02]+(?:[=<>+][\p{L}\d_^'\x01\x02]+)*`)
	// totDash is a dash of an old page: " - " or "---".
	totDash = regexp.MustCompile(` -{1,3} |-{2,3}`)
)

// TotPage reads the problems of the year page of tournament n of the Tournament of Towns, of both tours, from the papers in its text: the HTML of the years to 2002 and the <pre> blocks of the years from 2003. The edition of a problem is its tour: the autumn tour of tournament n is in the year 1978+n, its spring tour in the year after.
func TotPage(page string, n int, url string) []RuProblem {
	lines := totLines(page)
	var out []RuProblem
	seen := map[string]bool{}
	for i := 0; i < len(lines); i++ {
		groups, tier, seasons, ok := totPageHeading(lines, i)
		if !ok {
			continue
		}
		// The tour is the first one whose papers of the groups are not yet read.
		season := ""
		for _, x := range seasons {
			if !seen[x+string([]byte{groups[0], tier})] {
				season = x
				break
			}
		}
		end := i + 1
		for end < len(lines) {
			if _, _, _, h := totPageHeading(lines, end); h || strings.Contains(lines[end], "ТУРНИР ГОРОДОВ") {
				break
			}
			end++
		}
		if season == "" {
			i = end - 1
			continue
		}
		year := 1978 + n
		if season == "spring" {
			year++
		}
		edition := strconv.Itoa(year) + "-" + season
		probs := totTextProblems(lines[i+1 : end])
		for _, g := range groups {
			prefix := string([]byte{g, tier})
			if seen[season+prefix] {
				continue
			}
			seen[season+prefix] = true
			for k, p := range probs {
				p.Edition, p.URL = edition, url
				p.Position = prefix + strconv.Itoa(k+1)
				if p.Text != "" {
					out = append(out, p)
				}
			}
		}
		i = end - 1
	}
	return out
}

// totPageHeading reads the line i as a paper heading of a page: the age groups, the level and the tour. The title of the tournament is on a line just above it.
func totPageHeading(lines []string, i int) (groups []byte, tier byte, seasons []string, ok bool) {
	t := strings.TrimSpace(lines[i])
	g, tier, ok := TotHeading(t)
	if !ok || tier == 0 || strings.Contains(t, "Итог") || strings.Contains(t, "формат") || utf8.RuneCountInString(t) > 120 {
		return nil, 0, nil, false
	}
	title := false
	tour := ""
	for k := i - 1; k >= 0 && k >= i-6; k-- {
		l := lines[k]
		if strings.Contains(l, "ТУРНИР ГОРОДОВ") {
			title = true
			break
		}
		if strings.Contains(l, "Осенний тур") {
			tour = "fall"
		} else if strings.Contains(l, "Весенний тур") {
			tour = "spring"
		}
	}
	if !title {
		return nil, 0, nil, false
	}
	// The date of the heading tells the tour, and the line of the tour when the heading has no date. A page can give a wrong month or a wrong tour, so when the two differ both are kept, the date first.
	date := ""
	switch {
	case totFall.MatchString(t) && !totSpring.MatchString(t):
		date = "fall"
	case totSpring.MatchString(t) && !totFall.MatchString(t):
		date = "spring"
	}
	for _, x := range []string{date, tour} {
		if x != "" && !slices.Contains(seasons, x) {
			seasons = append(seasons, x)
		}
	}
	if len(seasons) == 0 {
		return nil, 0, nil, false
	}
	groups = []byte{g}
	if totBoth.MatchString(t) {
		groups = []byte{'j', 's'}
	}
	return groups, tier, seasons, true
}

// totLines gives the lines of a page: the lines of each <pre> as they are, and the lines of the HTML around them, with a superscript or a subscript as ^ or _, a picture as totImg and the end of a block as totEnd.
func totLines(page string) []string {
	// The foot of the page is no part of the papers.
	if i := strings.Index(page, "Дизайн сайта"); i >= 0 {
		page = page[:i]
	}
	page = ruComment.ReplaceAllString(page, "")
	page = ruScript.ReplaceAllString(page, "")
	var out []string
	last := 0
	for _, m := range totPre.FindAllStringSubmatchIndex(page, -1) {
		out = append(out, totHTMLLines(page[last:m[0]])...)
		last = m[1]
		// A <pre> in the HTML of a problem draws a formula in characters, which the text cannot hold.
		if !totPrePapers.MatchString(page[m[2]:m[3]]) {
			out = append(out, totArtLine)
			continue
		}
		out = append(out, totEnd)
		body := ruImage.ReplaceAllString(page[m[2]:m[3]], totImg)
		body = html.UnescapeString(ruTag.ReplaceAllString(body, ""))
		for _, l := range strings.Split(strings.ReplaceAll(body, "\r", ""), "\n") {
			out = append(out, totTabs(l))
		}
		out = append(out, totEnd)
	}
	return append(out, totHTMLLines(page[last:])...)
}

// totHTMLLines gives the lines of a part of a page outside <pre>.
func totHTMLLines(s string) []string {
	s = strings.NewReplacer("\r", "", "\n", " ").Replace(s)
	// A brace of the HTML is math, as in {x_n}, and becomes \{ in TeX. It is marked here, before the braces of the scripts are made.
	s = strings.NewReplacer("{", "\x01", "}", "\x02").Replace(s)
	s = ruScripts.ReplaceAllStringFunc(s, func(m string) string {
		g := ruScripts.FindStringSubmatch(m)
		arg := strings.TrimSpace(html.UnescapeString(ruTag.ReplaceAllString(g[2], "")))
		op := "^"
		if strings.EqualFold(g[1], "sub") {
			op = "_"
		}
		if op == "^" && (arg == "o" || arg == "о" || arg == "0") {
			return "°"
		}
		if utf8.RuneCountInString(arg) == 1 {
			return op + arg
		}
		return op + "{" + arg + "}"
	})
	// The pages in tables put the points in a cell before the number and the author in a cell of its own.
	s = totPtsCell.ReplaceAllString(s, "</td>")
	s = totAuthorCell.ReplaceAllString(s, "<td></td>")
	s = ruImage.ReplaceAllString(s, " "+totImg+" ")
	s = totHR.ReplaceAllString(s, "\n"+totEnd+"\n")
	s = ruPara.ReplaceAllString(s, "\n")
	s = ruItem.ReplaceAllString(s, "\n")
	s = ruBreak.ReplaceAllString(s, "\n")
	s = html.UnescapeString(ruTag.ReplaceAllString(s, ""))
	return strings.Split(s, "\n")
}

// totTabs puts spaces in the place of the tabs of a line, with a tab stop each 8 columns.
func totTabs(l string) string {
	if !strings.Contains(l, "\t") {
		return l
	}
	var b strings.Builder
	col := 0
	for _, r := range l {
		if r == '\t' {
			for {
				b.WriteByte(' ')
				col++
				if col%8 == 0 {
					break
				}
			}
			continue
		}
		b.WriteRune(r)
		col++
	}
	return b.String()
}

// totIndent gives the column of the first letter of a line.
func totIndent(l string) int {
	return utf8.RuneCountInString(l) - utf8.RuneCountInString(strings.TrimLeft(l, " "))
}

// totTextProblems reads the problems of one paper from the lines after its heading. The problems are numbered from 1 in a column; in the <pre> blocks the points are in a column left of it.
func totTextProblems(lines []string) []RuProblem {
	numCol, textCol := -1, -1
	var starts []int
	for i, l := range lines {
		if l == totEnd || totRule.MatchString(strings.TrimSpace(l)) {
			if len(starts) > 0 {
				lines = lines[:i]
				break
			}
			continue
		}
		n, ind, rest, ok := totNum(l)
		if !ok {
			continue
		}
		// A page can give a number twice by mistake, as Задача 4. for the fifth problem.
		if (n != len(starts)+1 && (n != len(starts) || n == 0 || !strings.Contains(l, "Задача"))) || (numCol >= 0 && (ind < numCol-2 || ind > numCol+2)) {
			continue
		}
		if numCol < 0 {
			numCol = ind
			if strings.TrimSpace(l[rest:]) != "" && !totPtsIn.MatchString(l[rest:]) && !totSub.MatchString(l[rest:]) {
				textCol = utf8.RuneCountInString(l[:rest])
			}
		}
		starts = append(starts, i)
	}
	if len(starts) == 0 {
		return nil
	}
	if textCol < 0 {
		textCol = numCol + 3
	}
	// The points left of the numbers go.
	if numCol >= 3 {
		for i, l := range lines {
			if m := totPtsCol.FindStringIndex(l); m != nil && utf8.RuneCountInString(strings.TrimRight(l[:m[1]], " ")) < numCol {
				lines[i] = strings.Repeat(" ", m[1]) + l[m[1]:]
			}
		}
	}
	var out []RuProblem
	for k, s := range starts {
		end := len(lines)
		if k+1 < len(starts) {
			end = starts[k+1]
		}
		out = append(out, totTextProblem(lines[s:end], textCol))
	}
	return out
}

// totNum reads the number of a problem at the start of a line: the number, its column and the index of the text after it. A number with no dot must have the word Задача before it or a part а) after it.
func totNum(l string) (n, col, rest int, ok bool) {
	m := totNumLine.FindStringSubmatchIndex(l)
	if m == nil {
		return 0, 0, 0, false
	}
	dot := strings.Contains(l[m[8]:m[9]], ".")
	word := m[4] >= 0
	after := l[m[1]:]
	if !dot && !word && (m[9] == m[8] || !totLetterSub.MatchString(after)) {
		return 0, 0, 0, false
	}
	n, _ = strconv.Atoi(l[m[6]:m[7]])
	return n, utf8.RuneCountInString(l[:m[3]]), m[1], true
}

// totTextProblem reads one problem from its lines. The first line starts with the number.
func totTextProblem(lines []string, textCol int) RuProblem {
	var p RuProblem
	layout := false
	var b strings.Builder
	for i, l := range lines {
		if strings.Contains(l, totArtLine) {
			layout = true
			l = strings.ReplaceAll(l, totArtLine, " ")
		}
		if strings.Contains(l, totImg) {
			p.Figure = true
			l = strings.ReplaceAll(l, totImg, " ")
		}
		if i == 0 {
			_, _, rest, _ := totNum(l)
			l = strings.Repeat(" ", textCol) + totPtsIn.ReplaceAllString(l[rest:], "")
		}
		t := strings.TrimSpace(l)
		if t == "" || t == totEnd {
			continue
		}
		ind := totIndent(l)
		if i > 0 && totAuthorLine.MatchString(t) && (ind > textCol+15 || !strings.HasPrefix(t, "(")) {
			continue
		}
		if i > 0 && ind > textCol+8 {
			layout = true
		}
		if totArt.MatchString(t) {
			layout = true
		}
		sub := totSub.MatchString(t)
		if sub {
			// The points of a part go: а)(2) or а) (3 балла).
			t = t[:strings.Index(t, ")")+1] + " " + totPtsIn.ReplaceAllString(t[strings.Index(t, ")")+1:], "")
			t = strings.TrimSpace(t)
		}
		cur := b.String()
		switch {
		case cur == "":
			b.WriteString(t)
		case sub:
			// A part in the middle of a sentence, as "из а) 32 б) 22 монет", does not stand on a line of its own.
			if totDigits.MatchString(strings.TrimSpace(t[strings.Index(t, ")")+1:])) {
				layout = true
			}
			b.WriteString("\n" + t)
		case totHyphen(cur, t):
			b.Reset()
			b.WriteString(strings.TrimSuffix(cur, "-") + t)
		default:
			b.WriteString(" " + t)
		}
	}
	text := strings.Join(strings.Fields(strings.ReplaceAll(b.String(), "\n", " \x05 ")), " ")
	text = strings.ReplaceAll(text, " \x05 ", "\n")
	text = strings.TrimPrefix(text, "\x05 ")
	text = totInHyphen.ReplaceAllStringFunc(text, func(m string) string {
		i := strings.Index(m, "- ")
		if totHyphen(m[:i+1], m[i+2:]) {
			return m[:i] + m[i+2:]
		}
		return m
	})
	text = totLike.ReplaceAllStringFunc(text, totLatin)
	text = strings.NewReplacer("<<", "«", ">>", "»").Replace(text)
	text = RuPlain(text)
	text = ruAuthor.ReplaceAllString(text, "")
	text = totAuthorEnd.ReplaceAllString(text, "")
	text = strings.NewReplacer("\x01", "{", "\x02", "}").Replace(RuTeX(text))
	text = totEllipsis(text)
	// A dash of the text, out of the math, and a letter before a word, as in 2n-угольник.
	parts := strings.Split(text, "$")
	for k := 0; k < len(parts); k += 2 {
		parts[k] = totDash.ReplaceAllString(parts[k], " — ")
		parts[k] = totLetterWord.ReplaceAllString(parts[k], "$1$$$2$$-")
	}
	text = strings.Join(parts, "$")
	for k := 0; k < len(parts); k += 2 {
		if totLoose.MatchString(parts[k]) {
			p.Flags = append(p.Flags, "math")
			break
		}
	}
	if totFigure.MatchString(text) {
		p.Figure = true
	}
	// A hyphen in a word that is not кто-то or по-разному may be a cut the source kept from print.
	for _, m := range totInWord.FindAllString(text, -1) {
		if i := strings.Index(m, "-"); totHyphen(m[:i+1], m[i+1:]) {
			p.Flags = append(p.Flags, "hyphen")
			break
		}
	}
	if layout || totVerbal.MatchString(text) || strings.Contains(text, "\x00") {
		p.Flags = append(p.Flags, "layout")
	}
	p.Text = strings.TrimSpace(text)
	return p
}

// totEllipsis puts dots between two formulas in the math when an operator joins them, as $a_1+$ ... $+a_k$ or $u_1 > u_2$ > ... > $u_k$.
func totEllipsis(text string) string {
	var b strings.Builder
	last := 0
	for _, m := range totDots.FindAllStringSubmatchIndex(text, -1) {
		before := strings.TrimSpace(text[:m[0]])
		after := strings.TrimSpace(text[m[1]:])
		op := strings.TrimSpace(text[m[2]:m[3]]+text[m[4]:m[5]]) != "" ||
			strings.HasSuffix(before, "+") || strings.HasSuffix(before, "-") || strings.HasSuffix(before, "=") ||
			strings.HasPrefix(after, "+") || strings.HasPrefix(after, "-") || strings.HasPrefix(after, "=")
		// The match must close a formula: an even count of $ before it.
		if !op || strings.Count(text[:m[0]], "$")%2 == 0 {
			continue
		}
		b.WriteString(strings.TrimRight(text[last:m[0]], " "))
		b.WriteString(strings.TrimSpace(text[m[2]:m[3]]) + ` \ldots ` + strings.TrimSpace(text[m[4]:m[5]]))
		last = m[1]
	}
	b.WriteString(text[last:])
	return b.String()
}

// totLatinLike are the small Cyrillic letters that stand for Latin ones in a formula.
var totLatinLike = map[rune]rune{'а': 'a', 'е': 'e', 'о': 'o', 'р': 'p', 'с': 'c', 'х': 'x', 'у': 'y'}

// totLatin gives a formula with Latin letters when it has a script or a sign and all its letters are Latin or look like Latin ones, as х_1 or АК=АО.
func totLatin(m string) string {
	if !strings.ContainsAny(m, "_^=<>+") {
		return m
	}
	var b strings.Builder
	for _, r := range m {
		switch {
		case ruLike[r] != 0:
			r = ruLike[r]
		case totLatinLike[r] != 0:
			r = totLatinLike[r]
		case unicode.Is(unicode.Cyrillic, r):
			return m
		}
		b.WriteRune(r)
	}
	return b.String()
}
