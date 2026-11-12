package official

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	// totGradeRE is the heading of a paper of the Tournament of Towns: "8 – 9 классы, базовый вариант" or "10-11 кл., тренировочный вариант".
	totGradeRE = regexp.MustCompile(`(\d{1,2})\s*(?:--|[–—-])\s*(\d{1,2})\s*(?:класс|кл\.)`)
	// totBase and totHard name the two levels of a tour: the base (O) level and the hard (A) level.
	totBase = regexp.MustCompile(`(?i)базов|тренировоч|подготовит`)
	totHard = regexp.MustCompile(`(?i)сложн|основн`)
	// totTitle is a line of the title of a tour: "СОРОК ШЕСТОЙ ТУРНИР ГОРОДОВ" or "Весенний тур,".
	totTitle = regexp.MustCompile(`ТУРНИР ГОРОДОВ|^(?:Осенний|Весенний) тур`)
	// totMathPunc is a space between a formula and the punctuation after it.
	totMathPunc = regexp.MustCompile(`\$ +([,.;:?!)])`)
	totList     = regexp.MustCompile(`\s*\n- +`)
	// totGlued is a word glued to the formula after it, as "точке$C$".
	totGlued  = regexp.MustCompile(`(\p{Cyrillic})\$`)
	totDigits = regexp.MustCompile(`^\d{1,2}$`)
	// totUmlaut is a letter ё that the PDF gives as a diaeresis and a letter е.
	totUmlaut  = regexp.MustCompile(`¨\s*([еЕ])`)
	totNumber  = regexp.MustCompile(`^(\d{1,2})\.$`)
	totPart    = regexp.MustCompile(`^[а-жa-h]\)$`)
	totPoints  = regexp.MustCompile(`^[\d+\s()]+$`)
	totFigure  = regexp.MustCompile(`(?i)рисун|рис\.`)
	totEndPunc = regexp.MustCompile(`[.?!:]$`)
	// totKeepAfter and totAdverb are the parts of a word with a hyphen, as какой-то or по-разному, so that a hyphen at the end of a line before them stays.
	totKeepAfter = regexp.MustCompile(`^(?:то|либо|нибудь|таки)(?:[^\p{L}]|$)`)
	totAdverb    = regexp.MustCompile(`^\p{L}+(?:ому|ему|ски|цки|ьи)(?:[^\p{L}]|$)`)
)

// TotTier gives the level of a paper heading: 'o' for the base level and 'a' for the hard level, or 0.
func TotTier(s string) byte {
	switch {
	case totBase.MatchString(s):
		return 'o'
	case totHard.MatchString(s):
		return 'a'
	}
	return 0
}

// TotHeading reads a paper heading of the Tournament of Towns: the age group, j for the younger grades and s for the older ones, and the level. ok is false when s is not a heading.
func TotHeading(s string) (group, tier byte, ok bool) {
	m := totGradeRE.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, false
	}
	g, _ := strconv.Atoi(m[1])
	group = 'j'
	if g >= 10 {
		group = 's'
	}
	return group, TotTier(s), true
}

// totSection is the lines of one paper of a PDF.
type totSection struct {
	group, tier byte
	lines       []*pxLine
}

// TotPDF reads the problems of a paper PDF of the Tournament of Towns, from the runs of text that pdftohtml -xml gives. The PDF has a heading for each age group, as "8 – 9 классы, базовый вариант", and problems numbered from 1. tier is the level of the PDF, from its name, for a heading that does not name it. The positions are ja1, jo1, sa1 or so1 (spec 04).
func TotPDF(items []PDFItem, edition, url string, tier byte) []RuProblem {
	main := pxMainSize(items)
	lines, orphans := pxLines(items, main)
	var secs []*totSection
	for _, l := range lines {
		if g, t, ok := TotHeading(l.text()); ok && l.items[0].Size >= main && !strings.Contains(l.text(), "Итог") {
			if t == 0 {
				t = tier
			}
			secs = append(secs, &totSection{group: g, tier: t})
			continue
		}
		if totTitle.MatchString(l.text()) {
			continue
		}
		if len(secs) > 0 {
			secs[len(secs)-1].lines = append(secs[len(secs)-1].lines, l)
		}
	}
	var out []RuProblem
	seen := map[string]bool{}
	for _, s := range secs {
		prefix := string([]byte{s.group, s.tier})
		if s.tier == 0 || seen[prefix] {
			continue
		}
		seen[prefix] = true
		out = append(out, s.problems(edition, url, prefix, orphans)...)
	}
	return out
}

// totStart is the line of the start of a problem and the index of the item after its number.
type totStart struct {
	line, from int
}

func (s *totSection) problems(edition, url, prefix string, orphans []PDFItem) []RuProblem {
	// The numbers of the problems are in a column left of the text: 1., 2., and so on.
	// A number can come in two runs, as "1" in the math font and "." in the text font.
	for _, l := range s.lines {
		for k := 0; k+1 < len(l.items); k++ {
			it, dot := l.items[k], l.items[k+1]
			if totDigits.MatchString(it.Text) && dot.Text == "." && dot.Left-it.Right() < 3 && len(l.scripts[k]) == 0 {
				it.Text += "."
				it.Width = dot.Right() - it.Left
				l.items = slices.Delete(slices.Clone(l.items), k+1, k+2)
				scripts := map[int][]pxScript{}
				for i, sc := range l.scripts {
					switch {
					case i <= k:
						scripts[i] = sc
					case i == k+1:
						scripts[k] = append(scripts[k], sc...)
					default:
						scripts[i-1] = sc
					}
				}
				l.items[k], l.scripts = it, scripts
			}
		}
	}
	var starts []totStart
	numLeft, textLeft := -1, -1
	for i, l := range s.lines {
		for k, it := range l.items {
			m := totNumber.FindStringSubmatch(strings.TrimSpace(it.Text))
			if m == nil {
				continue
			}
			n, _ := strconv.Atoi(m[1])
			if n != len(starts)+1 || (numLeft >= 0 && (it.Left < numLeft-6 || it.Left > numLeft+6)) {
				continue
			}
			// A number can stand alone on its line, when the problem starts with a part "а)" on the next line.
			if numLeft < 0 {
				numLeft = it.Left
			}
			if textLeft < 0 && k+1 < len(l.items) && !totPart.MatchString(strings.TrimSpace(l.items[k+1].Text)) {
				textLeft = l.items[k+1].Left
			}
			starts = append(starts, totStart{i, k + 1})
			break
		}
	}
	if len(starts) == 0 {
		return nil
	}
	if textLeft < 0 {
		textLeft = numLeft + 35
	}
	// The points of a problem are in a column left of the numbers.
	for _, l := range s.lines {
		var keep []PDFItem
		scripts := map[int][]pxScript{}
		for k, it := range l.items {
			if it.Right() < numLeft-2 && totPoints.MatchString(it.Text) {
				continue
			}
			scripts[len(keep)] = l.scripts[k]
			keep = append(keep, it)
		}
		if len(keep) == 0 {
			keep = append(keep, PDFItem{Page: l.page(), Top: l.top(), Left: textLeft, Text: " "})
		}
		l.items, l.scripts = keep, scripts
	}
	// The points of a problem are gone, so the start of each problem is again the item after its number.
	for i, st := range starts {
		for k, it := range s.lines[st.line].items {
			if totNumber.MatchString(strings.TrimSpace(it.Text)) && it.Left >= numLeft-6 && it.Left <= numLeft+6 {
				starts[i].from = k + 1
				break
			}
		}
	}
	colRight := 0
	for _, l := range s.lines {
		colRight = max(colRight, l.right())
	}
	var out []RuProblem
	for i, st := range starts {
		end := len(s.lines)
		if i+1 < len(starts) {
			end = starts[i+1].line
		}
		p := s.problem(s.lines[st.line:end], st.from, textLeft, colRight, orphans)
		p.Edition, p.URL = edition, url
		p.Position = prefix + strconv.Itoa(i+1)
		p.Page = s.lines[st.line].page()
		if p.Text == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

// problem reads one problem from its lines. from is the first item of the first line after the number.
func (s *totSection) problem(lines []*pxLine, from, textLeft, colRight int, orphans []PDFItem) RuProblem {
	var p RuProblem
	layout := false
	first, last := lines[0], lines[len(lines)-1]
	for _, o := range orphans {
		if o.Page < first.page() || o.Page > last.page() {
			continue
		}
		if (o.Page > first.page() || o.Top >= first.top()-4) && (o.Page < last.page() || o.Top <= last.top()+last.items[0].Height+4) {
			p.Figure = true
		}
	}
	type part struct {
		text      string
		start     int
		right     int
		firstMath bool
		mathOnly  bool
		top, page int
	}
	var parts []part
	for li, l := range lines {
		start := 0
		if li == 0 {
			start = from
		}
		items := l.items[start:]
		if len(items) == 0 || strings.TrimSpace(l.text()) == "" {
			continue
		}
		// An author: a line in italics at the right.
		italic := true
		for _, it := range items {
			italic = italic && it.Italic()
		}
		if italic && items[0].Left > textLeft+60 {
			continue
		}
		// A label of a figure: a short line far right of the text.
		if items[0].Left > textLeft+150 && utf8.RuneCountInString(strings.ReplaceAll(l.text(), " ", "")) <= 3 {
			p.Figure = true
			continue
		}
		// Items after a wide gap are the labels of a figure at the side of the text.
		cut := len(l.items)
		for k := start + 1; k < len(l.items); k++ {
			prev := l.items[k-1].Right()
			if ss := l.scripts[k-1]; len(ss) > 0 {
				prev = max(prev, ss[len(ss)-1].Right())
			}
			if l.items[k].Left-prev > 30 {
				cut = k
				p.Figure = true
				break
			}
		}
		sub := &pxLine{items: l.items[:cut], scripts: l.scripts}
		ps, ok := sub.pieces(start)
		if !ok {
			layout = true
		}
		var b strings.Builder
		mathOnly := true
		for _, pc := range ps {
			if pc.math {
				m := strings.TrimSpace(pc.s)
				tail := ""
				for m != "" && strings.ContainsAny(m[len(m)-1:], ".,;:?!") && !strings.HasSuffix(m, `\`+m[len(m)-1:]) {
					tail = m[len(m)-1:] + tail
					m = strings.TrimSpace(m[:len(m)-1])
				}
				switch m = PDFTidy(m); {
				case m == `\bullet`:
					// A bullet starts an item of a list.
					b.WriteString("\n- ")
				case m != "":
					b.WriteString("$" + m + "$")
				}
				b.WriteString(tail)
			} else {
				b.WriteString(strings.ReplaceAll(pc.s, ". . .", "..."))
				if strings.TrimSpace(pc.s) != "" {
					mathOnly = false
				}
			}
		}
		text := strings.TrimSpace(totMathPunc.ReplaceAllString(b.String(), "$$$1"))
		text = totList.ReplaceAllString(text, "\n- ")
		text = totGlued.ReplaceAllString(text, "$1 $$")
		if strings.HasPrefix(strings.TrimLeft(b.String(), " "), "\n- ") {
			text = "\n" + text
		}
		if text == "" {
			continue
		}
		left := l.items[start].Left
		if totPart.MatchString(strings.TrimSpace(l.items[start].Text)) && left < textLeft-5 {
			left = textLeft
			text = "\n" + text
		}
		parts = append(parts, part{text: text, start: left, right: l.items[cut-1].Right(), firstMath: len(ps) > 0 && ps[0].math, mathOnly: mathOnly, top: l.top(), page: l.page()})
	}
	// Two lines of math alone close above each other are a fraction or a table of a formula.
	gaps := []int{}
	for i := 1; i < len(parts); i++ {
		if parts[i].page == parts[i-1].page {
			gaps = append(gaps, parts[i].top-parts[i-1].top)
		}
	}
	if len(gaps) > 0 {
		sorted := slices.Clone(gaps)
		slices.Sort(sorted)
		med := sorted[len(sorted)/2]
		for i := 1; i < len(parts); i++ {
			if parts[i].page == parts[i-1].page && parts[i].mathOnly && parts[i-1].mathOnly && float64(parts[i].top-parts[i-1].top) < 0.75*float64(med) {
				layout = true
			}
		}
	}
	var b strings.Builder
	for i, pt := range parts {
		if i == 0 {
			b.WriteString(pt.text)
			continue
		}
		prev := parts[i-1]
		cur := b.String()
		switch {
		case strings.HasPrefix(pt.text, "\n"):
			b.WriteString(pt.text)
		case totHyphen(cur, pt.text):
			b.Reset()
			b.WriteString(strings.TrimSuffix(cur, "-") + pt.text)
		case strings.HasSuffix(cur, "-") && !strings.HasSuffix(cur, " -"):
			b.WriteString(pt.text)
		case prev.right < colRight-40 && totEndPunc.MatchString(cur) && pt.start <= textLeft+5:
			b.WriteString("\n" + pt.text)
		default:
			b.WriteString(" " + pt.text)
		}
	}
	text := RuPlain(totUmlaut.ReplaceAllStringFunc(b.String(), func(m string) string {
		if strings.HasSuffix(m, "Е") {
			return "Ё"
		}
		return "ё"
	}))
	text = ruAuthor.ReplaceAllString(text, "")
	if totFigure.MatchString(text) {
		p.Figure = true
	}
	if layout {
		p.Flags = append(p.Flags, "layout")
	}
	p.Text = text
	return p
}

// totHyphen tells if a line that ends in cur with a hyphen goes on in next with the rest of a word, so that the hyphen goes.
func totHyphen(cur, next string) bool {
	if !strings.HasSuffix(cur, "-") {
		return false
	}
	word := cur[:len(cur)-1]
	if i := strings.LastIndexAny(word, " \n($«"); i >= 0 {
		word = word[i+1:]
	}
	r, _ := utf8.DecodeLastRuneInString(word)
	n, _ := utf8.DecodeRuneInString(next)
	if !unicode.Is(unicode.Cyrillic, r) || !unicode.Is(unicode.Cyrillic, n) || !unicode.IsLower(n) {
		return false
	}
	if totKeepAfter.MatchString(next) {
		return false
	}
	// по-разному and во-первых keep the hyphen, but по-ставили is поставили.
	switch strings.ToLower(word) {
	case "кое", "кой":
		return false
	case "по":
		return !totAdverb.MatchString(next)
	case "во":
		return !strings.HasPrefix(next, "первых") && !strings.HasPrefix(next, "вторых") && !strings.HasPrefix(next, "третьих")
	case "из":
		return !strings.HasPrefix(next, "за") && !strings.HasPrefix(next, "под")
	}
	return true
}
