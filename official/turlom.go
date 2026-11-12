package official

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

var (
	// turlomHeadRE is a heading of a page of the Lomonosov Tournament.
	turlomHeadRE = regexp.MustCompile(`(?is)<h([1-6])\b[^>]*>(.*?)</h[1-6]>`)
	// turlomMathRE is the heading of the problems of mathematics.
	turlomMathRE = regexp.MustCompile(`(?i)^(?:математика|конкурс по математике|задания конкурса по математике)$`)
	// turlomPartRE is a heading inside the problems of mathematics: the word "Задачи", a group of grades or the extra problems. Another heading ends the problems.
	turlomPartRE = regexp.MustCompile(`(?i)^(?:задачи|\d{1,2}(?:\s*[-–—]\s*\d{1,2})?\s*класс.*|старшие классы.*|дополнительные задачи.*)$`)
	turlomItemRE = regexp.MustCompile(`(?i)<li\b[^>]*>`)
	turlomOLEnd  = regexp.MustCompile(`(?i)</[ou]l>`)
	turlomPre    = regexp.MustCompile(`(?i)<pre\b`)
	// turlomGrades is the label of the grades at the start of a problem, as (7-9), (6, 7, 8), (6&ndash;7) or (6–7; ответ). It is kept out of the text.
	turlomGrades = regexp.MustCompile(`^((?:\s|<[^>]+>)*)\(\s*\d{1,2}(?:\s*(?:-|–|—|&ndash;|&mdash;|,|;)\s*\d{1,2})*\s*(?:класс\S*)?\s*\)\.?\s*`)
	// turlomFigure is a word that sends the reader to a picture, as рис. 1 or на рисунке.
	turlomFigure = regexp.MustCompile(`(?i)рис\.|рисун`)
)

// TurlomHTML reads the problems of mathematics of a year page of the Lomonosov Tournament. The problems are under the heading "Математика" or "Конкурс по математике", maybe in groups of grades with their own numbers. The positions are p1, p2 and so on in the order of the page, across the groups, as on problems.ru. edition is the edition part of the locator and url the address of the page.
func TurlomHTML(page, edition, url string) []RuProblem {
	page = ruComment.ReplaceAllString(page, "")
	heads := turlomHeadRE.FindAllStringSubmatchIndex(page, -1)
	var blocks []string
	from := -1
	for _, h := range heads {
		name := RuPlain(html.UnescapeString(ruTag.ReplaceAllString(page[h[4]:h[5]], " ")))
		if from < 0 {
			if turlomMathRE.MatchString(name) {
				from = h[1]
			}
			continue
		}
		blocks = append(blocks, page[from:h[0]])
		from = h[1]
		if !turlomPartRE.MatchString(name) {
			from = -2
			break
		}
	}
	if from >= 0 {
		blocks = append(blocks, page[from:])
	}
	var out []RuProblem
	for _, b := range blocks {
		for _, h := range turlomItems(b) {
			h = turlomGrades.ReplaceAllString(h, "$1")
			// A dollar of the page is money, as in 5$, and not math.
			text, fig := RuHTML(strings.ReplaceAll(h, "$", "\x03"))
			text = strings.ReplaceAll(text, "\x03", `\$`)
			p := RuProblem{Edition: edition, Position: "p" + strconv.Itoa(len(out)+1), Text: text, URL: url}
			p.Figure = fig || turlomPre.MatchString(h) || turlomFigure.MatchString(text)
			if RuLayout(h) {
				p.Flags = append(p.Flags, "layout")
			}
			out = append(out, p)
		}
	}
	return out
}

// turlomItems cuts a group of problems into its problems: the items of a list, or else the numbered paragraphs.
func turlomItems(s string) []string {
	if !turlomItemRE.MatchString(s) {
		return RuNumbered(s)
	}
	locs := turlomItemRE.FindAllStringIndex(s, -1)
	var out []string
	for i, l := range locs {
		end := len(s)
		if i+1 < len(locs) {
			end = locs[i+1][0]
		}
		item := s[l[1]:end]
		if e := turlomOLEnd.FindStringIndex(item); e != nil {
			item = item[:e[0]]
		}
		out = append(out, item)
	}
	return out
}

var (
	// turlomPDFGrades are the grades in brackets after the number of a problem, as (6–7) or (7–11; ответ).
	turlomPDFGrades = `\(\d{1,2}(?:\s*[-–—]\s*\d{1,2})?(?:;[^)\n]*)?\)\.?[ \t]*`
	// turlomPDFEnd is the text after the problems: the note to sign the work, or the address of the solutions.
	turlomPDFEnd = regexp.MustCompile(`(?m)^(?:Не забудьте|Задания, решения)`)
	// turlomPDFSkip is a line of the paper that is not a part of a problem: a header of a page, a note to turn the page, the place for a draft or the names of pictures.
	turlomPDFSkip = regexp.MustCompile(`(?m)^(?:.*Турнир имени.*|Конкурс по математике|\d{1,2} \p{L}+ \d{4} года|лист \d+ из \d+|\(См\. также.*\)|Место для черновика.*|(?:Рис\. \d+\s*)+)\n`)
)

// turlomPDFMarkRE gives the start of problem N of a paper: N at the start of a line, maybe a dot, and the grades, as "1. (6–7)" or "4 (7–11; ответ).".
func turlomPDFMarkRE(n int) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^` + strconv.Itoa(n) + `\.?\s*` + turlomPDFGrades)
}

// TurlomPDF reads the problems of mathematics of a paper of the Lomonosov Tournament in PDF. pages are the texts of the pages, as pdftotext -raw gives them. A problem starts with its number and the grades in brackets, as "1. (6–7)". The numbers go up from 1; a number that comes again starts a copy of the paper, which is left out.
func TurlomPDF(pages []string, edition, url string) []RuProblem {
	all, starts := pdfJoin(pages)
	if e := turlomPDFEnd.FindStringIndex(all); e != nil {
		all = all[:e[0]]
	}
	var marks [][]int
	from := 0
	for n := 1; n <= 20; n++ {
		loc := turlomPDFMarkRE(n).FindStringIndex(all[from:])
		if loc == nil {
			break
		}
		marks = append(marks, []int{from + loc[0], from + loc[1]})
		from += loc[1]
	}
	var out []RuProblem
	for i, m := range marks {
		end := len(all)
		if i+1 < len(marks) {
			end = marks[i+1][0]
		}
		raw := turlomPDFSkip.ReplaceAllString(all[m[1]:end]+"\n", "")
		text, fig := PDFText(raw)
		p := RuProblem{Edition: edition, Position: "p" + strconv.Itoa(i+1), URL: url, Page: pdfPage(starts, m[0])}
		p.Text = text
		p.Figure = fig || turlomFigure.MatchString(text)
		out = append(out, p)
	}
	return out
}

// pdfJoin joins the texts of the pages, each one ending with a new line, and gives the start of each page in the joined text.
func pdfJoin(pages []string) (string, []int) {
	var b strings.Builder
	var starts []int
	for _, p := range pages {
		starts = append(starts, b.Len())
		b.WriteString(strings.TrimRight(p, "\f\n "))
		b.WriteString("\n")
	}
	return b.String(), starts
}

// pdfPage gives the page, from 1, of the place at in a text that pdfJoin made.
func pdfPage(starts []int, at int) int {
	page := 0
	for i, s := range starts {
		if at >= s {
			page = i + 1
		}
	}
	return page
}

var (
	// pdfHyphen is a word cut at the end of a line, as "пере-\nсекаются".
	pdfHyphen = regexp.MustCompile(`(\p{Ll})-\n(\p{Ll})`)
	// pdfLabel is a line that is only a label of a picture: a few letters, digits, primes or dots.
	pdfLabel = regexp.MustCompile(`^(?:[A-Za-z0-9′'.]{1,3}\s?){1,4}$|^\.\.\.$|^или$`)
	// pdfSpacePunct is a space before a comma or a full stop, which a script that pdftotext puts on its own line leaves.
	pdfSpacePunct = regexp.MustCompile(` ([,.;:])`)
	// pdfDash is a dash between words, which pdftotext often gives with no space on one side.
	pdfDash = regexp.MustCompile(`[ \t]*—[ \t]*`)
	// pdfControl is a control character, which pdftotext gives for a sign of a font that it cannot read.
	pdfControl = regexp.MustCompile(`[\x00-\x08\x0b-\x1f]`)
	// pdfDollar is a dollar of the text, which is money and not math.
	pdfDollar = strings.NewReplacer("$", "\x03")
)

// PDFText makes the text of one problem of a PDF into Markdown with TeX. The lines of a paragraph are joined, and a word cut at the end of a line is made whole. The math letters of Unicode become plain letters, and a dash gets a space on each side. A sign that pdftotext cannot read becomes [?], so that the text goes to review. A run of three or more lines that are only labels, as the names of points of a picture, tells that the problem has a picture, and the run is left out.
func PDFText(raw string) (text string, figure bool) {
	raw = pdfHyphen.ReplaceAllString(raw, "$1$2")
	raw = pdfControl.ReplaceAllString(raw, "[?]")
	// A sign at the end of a line comes again at the start of the next line in Russian print, as "AB =\n= BC".
	for _, op := range []string{"=", "+", "−", "-", "·", "×", "<", ">"} {
		raw = strings.ReplaceAll(raw, op+"\n"+op+" ", op+"\n")
	}
	lines := strings.Split(raw, "\n")
	var keep []string
	for i := 0; i < len(lines); {
		j := i
		for j < len(lines) && pdfLabel.MatchString(strings.TrimSpace(lines[j])) {
			j++
		}
		if j-i >= 3 {
			figure = true
			i = j
			continue
		}
		if j == i {
			j = i + 1
		}
		keep = append(keep, lines[i:j]...)
		i = j
	}
	s := strings.Join(keep, " ")
	s = strings.Map(mathPlain, s)
	s = strings.NewReplacer("◦", "°", "̸=", "≠").Replace(s)
	s = RuPlain(pdfDollar.Replace(s))
	s = pdfSpacePunct.ReplaceAllString(s, "$1")
	s = pdfDash.ReplaceAllString(s, " — ")
	s = strings.ReplaceAll(RuTeX(s), "\x03", `\$`)
	return s, figure
}

// mathPlain makes a letter of the Unicode block of math letters, as 𝑛 or 𝐴, into the plain Latin letter.
func mathPlain(r rune) rune {
	switch {
	case r >= 0x1D400 && r <= 0x1D6A3:
		i := (r - 0x1D400) % 52
		if i < 26 {
			return 'A' + i
		}
		return 'a' + i - 26
	case r == 0x210E:
		return 'h'
	}
	return r
}
