package official

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

// HrVersion names the rules of HrPDF in the extraction field of a statement.
const HrVersion = "om-hr/1"

// HrRules are the lines that shape a kind of paper of the Croatian Mathematical Society on natjecanja.math.hr.
type HrRules struct {
	// Start is the heading of a problem at the start of a line, as "3." or "Zadatak K-1.3. [10 bodova]". Its group 1 is the label of the problem, and its last group with a number counts the problems of a part of the paper.
	Start *regexp.Regexp
	// Section starts a part of the paper, as "1. razred – srednja škola – A varijanta" or "Prvi dan". Its first group that matched, or the whole line, names the part.
	Section *regexp.Regexp
	// Stop is a line that ends a problem and is no part of it, as "Svaki zadatak vrijedi 10 bodova.", the header of a paper or "Rješenje.".
	Stop *regexp.Regexp
	// Skip is a line that a problem goes past and that is no part of it, as the footer of a page or "Okreni list!".
	Skip *regexp.Regexp
	// Tail is a note that a paper sets right after the last problem of a page, as "Svaki se zadatak boduje s 10 bodova.": the text of a problem ends where it starts.
	Tail *regexp.Regexp
	// Drop is a note at the end of a line, set apart from its text or on a line of its own, that is no part of the problem, as the names of the authors "(P. Černek)" that the Slovak papers set at the right margin.
	Drop *regexp.Regexp
	// Not is a line that Start matches but that is no heading, as "4 360. Aké dve čísla…", a number with a space between its thousands at the start of a line.
	Not *regexp.Regexp
	// Lossy are the fonts of the papers whose text loses letters: a paper with a run in one of them gives each of its problems the flag glyph, so that it keeps its link only.
	Lossy *regexp.Regexp
	// Letters are the letters and the signs beyond ASCII that the text outside the formulas may hold; with none they are those of Croatian, hrLetters.
	Letters string
	// Titled tells that the rest of the line of the heading is the title of the problem, as "1. Aljmez", which becomes a paragraph of its own in bold. The heading of a titled problem may stand right of a figure, so it need not line up with the first heading of its part.
	Titled bool
	// First tells that the first problem of a part is 1, so that a date as "26. siječnja 2026." is no heading.
	First bool
}

// HrProblem is one problem of a Croatian paper: Code is its label, as 3 or K-1.3, N its number in the part of the paper, and Section the name of the part of the paper it is in.
type HrProblem struct {
	DemoProblem
	N       int
	Section string
}

// hrFonts are the fonts of the papers: Latin Modern and Computer Modern with the AMS symbols, the Latin Modern of the MEDO papers, whose names start with MLM, the EC fonts of cm-super of the older books of the HMO, as SFRM1200, and the fonts of the papers made with Word, with Cambria Math and Symbol. A bullet of a list is the text "-" of the font hrBullet.
var hrFonts = paperFonts{
	math: regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:M?(?:LMMathItalic|LMMathSymbols|LMMathExtension|CMMI|CMSY|CMEX|MSAM|MSBM)\d|CambriaMath$|Cambria Math$|Symbol$|SymbolMT$)`),
	text: regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:M?(?:LMRoman|LMSans|LMRomanSlant|CMR|CMBX|CMTI|CMSL|CMSS|CMBXSL|CMSSBX|CMSSI)\d|SF(?:RM|SX|TI|SL|SS|BX|CC)\d{4}$|cs(?:r|bx|sl|ti|ss|sx|tt|csc|bxti)\d+$|Times(?:-Roman|-Italic|-Bold|-BoldItalic)?$|Helvetica(?:-Bold|-Oblique|-BoldOblique)?$|(?:Calibri|Cambria|TimesNewRoman|Times New Roman|Arial)(?:PS|MT|PSMT)?(?:[-,](?:Bold|Italic|BoldItalic|BoldMT|ItalicMT|BoldItalicMT|BoldPSMT|ItalicPSMT))?$|` + hrBullet + `$)`),
	part: regexp.MustCompile(`^(?:\(?[a-z]\)|\(\d+\)|- )\s`),
}

// hrBullet is the font of an item that was a bullet of a list.
const hrBullet = "hr-bullet"

// hrBullets are the glyphs of the bullets of a list: of the math symbols, of the symbol fonts and of Wingdings.
var hrBullets = []string{"\u2022", "\u2219", "\u25aa", "\u25e6", "\u00b7", "\uf0b7", "\uf0a7", "\uf076", "\uf0d8", "\uf0fc"}

// hrSpaces are the odd spaces and the ligatures of the text of an item, and the í and the non-breaking hyphen of XeLaTeX, which sets the í as a dotless i with an accent.
var hrSpaces = strings.NewReplacer("\u00a0", " ", "\u2009", " ", "\u202f", " ", "\ufb01", "fi", "\ufb02", "fl", "\ufb00", "ff", "\ufb03", "ffi", "\ufb04", "ffl", "ı\u0301", "í", "\u2011", "-")

// hrAccentRE is a caron or an acute accent that the fonts of Computer Modern set as a glyph of its own before the letter, as "rjeˇsenja".
var hrAccentRE = regexp.MustCompile(`([ˇ´])\s?([sczSCZ])`)

// hrAccented gives the letters with the accents of Croatian.
var hrAccented = map[string]string{"ˇs": "š", "ˇc": "č", "ˇz": "ž", "ˇS": "Š", "ˇC": "Č", "ˇZ": "Ž", "´c": "ć", "´C": "Ć"}

// hrJoinAccents joins an item that ends with a caron or an acute accent to the next item of its line that starts with the letter under it: the fonts of Computer Modern set the accent as a glyph of its own, and pdftohtml may end a run of text with it.
func hrJoinAccents(items []PDFItem) []PDFItem {
	var out []PDFItem
	for _, it := range items {
		if n := len(out); n > 0 {
			prev := &out[n-1]
			if (strings.HasSuffix(prev.Text, "ˇ") || strings.HasSuffix(prev.Text, "´")) && prev.Page == it.Page && pxAbs(float64(prev.Top-it.Top)) <= 3 && it.Left <= prev.Right()+2 && it.Left >= prev.Left && strings.IndexAny(it.Text, "sczSCZ") == 0 {
				prev.Text += it.Text
				prev.Width = max(prev.Width, it.Right()-prev.Left)
				continue
			}
		}
		out = append(out, it)
	}
	return out
}

// hrDjWords are the words whose đ the fonts of Computer Modern lose, as "izmedu" for "između", with the đ put back: none of them is a word without the đ.
var hrDjWords = map[string]string{"izmedu": "između", "medu": "među", "medusobno": "međusobno", "medusobni": "međusobni", "medusobna": "međusobna", "odreden": "određen", "odredeni": "određeni", "odredena": "određena", "odredeno": "određeno", "rodendan": "rođendan", "rodendana": "rođendana", "dode": "dođe", "dodu": "dođu"}

// hrWordRE is a word of a text.
var hrWordRE = regexp.MustCompile(`[\p{L}]+`)

// hrDj puts back the đ of the words of hrDjWords, small or with a capital.
func hrDj(s string) string {
	return hrWordRE.ReplaceAllStringFunc(s, func(w string) string {
		if d, ok := hrDjWords[w]; ok {
			return d
		}
		if lw := strings.ToLower(w[:1]) + w[1:]; lw != w {
			if d, ok := hrDjWords[lw]; ok {
				return strings.ToUpper(d[:1]) + d[1:]
			}
		}
		return w
	})
}

// hrLetters are the letters and the signs beyond ASCII that the text of a Croatian paper has. Another one outside a formula, as the ö or the £ of a font with a broken encoding, flags the problem.
const hrLetters = "čćžšđČĆŽŠĐ–—„“”‘’…°€§«»·×−±≤≥é"

// hrOdd tells that the text outside the formulas has a letter or a sign that is not in hrLetters.
func hrOdd(s string) bool {
	return hrOddIn(s, hrLetters)
}

// hrOddIn tells that the text outside the formulas has a letter or a sign that is not in letters.
func hrOddIn(s, letters string) bool {
	for k, part := range strings.Split(s, "$") {
		if k%2 == 1 {
			continue
		}
		for _, r := range part {
			if r > 127 && !strings.ContainsRune(letters, r) {
				return true
			}
		}
	}
	return false
}

var (
	hrMathRE = regexp.MustCompile(`\$\$(.*?)\$\$|\$(.*?)\$`)
	// hrLoose is a command, a script or an operator without its operand in the text outside the formulas, as the + of a fraction whose bars and parts went astray.
	hrLoose = regexp.MustCompile(`\\[A-Za-z]|[\^_{}]|[+=]\s*[.,?;:)]|[+=]\s*$`)
	// hrStacked is a display that holds only short words, as "x y", the parts of fractions set one over the other.
	hrStacked = regexp.MustCompile(`^\s*[A-Za-z0-9]{1,3}(?:\s+[A-Za-z0-9]{1,3})+\s*[.,]?\s*$`)
	// hrHole is a formula with a hole, as $A ,-1$ for A(1/2, -1), $30^{()}$ or $,1$, or with a script that lost its place, as the index of $(a_n)_{n=0}^\infty$ in $(a_n)_n^\infty_{=0}$.
	hrHole    = regexp.MustCompile(`\s,\s*[-\d]|^,|\{\s*\}|\{\(\)\}|\(\s*\)|_\{?=`)
	hrCommand = regexp.MustCompile(`\\[A-Za-z]+`)
	// hrBare is a word and a stop with a hole between them, as "napísaný zlomok . Určte", where a fraction set over two lines went astray.
	hrBare = regexp.MustCompile(`\p{L} [.,;:](?:\s|$)`)
	// hrGlued is a word glued to a formula that starts with an operator, as mn$+2$m, or a sign of a formula twice, at its end and after it, as $P(t)=$ = 0.
	hrGlued = regexp.MustCompile(`[a-zčćšžđ]\$[+=-]|=\$\s*=`)
)

// hrBroken tells that the text lost a part of its formulas, as a fraction set as a stack of lines, a root or a pair of coordinates.
func hrBroken(s string) bool {
	if hrGlued.MatchString(s) || hrLoose.MatchString(hrMathRE.ReplaceAllString(s, " ")) || hrBare.MatchString(hrMathRE.ReplaceAllString(s, "0")) {
		return true
	}
	for _, m := range hrMathRE.FindAllStringSubmatch(s, -1) {
		if d := m[1]; d != "" && (hrStacked.MatchString(d) || !strings.ContainsFunc(hrCommand.ReplaceAllString(d, ""), func(r rune) bool {
			return r < 128 && (r >= '0' && r <= '9' || r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z')
		})) {
			return true
		}
		if hrHole.MatchString(m[1] + m[2]) {
			return true
		}
	}
	return false
}

// hrDrop takes the notes that drop matches off the ends of the lines: the last runs of a line whose text drop matches, set apart from the runs before them by a gap wider than a space. A line that holds only a note goes.
func hrDrop(lines []*pxLine, drop *regexp.Regexp) []*pxLine {
	out := lines[:0]
	for _, l := range lines {
		for k := len(l.items) - 1; k >= 0; k-- {
			var b strings.Builder
			for _, it := range l.items[k:] {
				b.WriteString(it.Text)
			}
			if !drop.MatchString(strings.TrimSpace(b.String())) {
				continue
			}
			if k > 0 {
				prev := l.items[k-1].Right()
				for _, sc := range l.scripts[k-1] {
					prev = max(prev, sc.Right())
				}
				if l.items[k].Left-prev <= 20 {
					break
				}
			}
			l.items = l.items[:k]
			for i := range l.scripts {
				if i >= k {
					delete(l.scripts, i)
				}
			}
			break
		}
		if len(l.items) > 0 {
			out = append(out, l)
		}
	}
	return out
}

// HrText gives the text of a run with the accents of Croatian put on their letters, and the letters that XeLaTeX gives as a letter and a combining accent, as c and a caron, composed.
func HrText(s string) string {
	return norm.NFC.String(hrAccentRE.ReplaceAllStringFunc(hrSpaces.Replace(s), func(m string) string {
		k := strings.Replace(m, " ", "", 1)
		if a, ok := hrAccented[k]; ok {
			return a
		}
		return m
	}))
}

// hrBarRE is the run of Cambria Math that Word sets over the letters of a segment for its bar: four apostrophes or four number signs.
var hrBarRE = regexp.MustCompile(`^(?:'{2,}|#{2,})$`)

// hrMSAM and hrMSBM are the fonts of the AMS symbols, whose glyphs the older books of the HMO and the older Slovak papers, made without the Unicode names of the glyphs, give as the ASCII letters at their places in the font, as CMSY does.
var (
	hrMSAM = regexp.MustCompile(`^(?:[A-Z]{6}\+)?MSAM\d`)
	hrMSBM = regexp.MustCompile(`^(?:[A-Z]{6}\+)?MSBM\d`)
	// hrAngles are the angle signs of MSAM: \angle, \measuredangle and \sphericalangle, which the HMO sets for an angle, and its relations \leqq, \geqq, \leqslant and \geqslant, which the Slovak papers set.
	hrAngles = strings.NewReplacer("\\", "∠", "]", "∡", "^", "∢", "5", "≦", "=", "≧", "6", "⩽", ">", "⩾")
)

// hrLegacy gives the glyphs of a run of the font family that come out as the ASCII letters at their places in the font: the prime and the parallel sign of CMSY, the angles of MSAM and the double-struck capitals of MSBM, as N for ℕ. A run that has the Unicode names of its glyphs has none of these letters.
func hrLegacy(family, s string) string {
	switch {
	case demoSymbolFont.MatchString(family):
		return demoOMS.Replace(s)
	case hrMSAM.MatchString(family):
		return hrAngles.Replace(s)
	case hrMSBM.MatchString(family):
		return strings.Map(func(r rune) rune {
			switch {
			case r == 'C':
				return 'ℂ'
			case r == 'N':
				return 'ℕ'
			case r == 'Q':
				return 'ℚ'
			case r == 'R':
				return 'ℝ'
			case r == 'Z':
				return 'ℤ'
			case r >= 'A' && r <= 'Z':
				return 0x1D538 + r - 'A'
			}
			return r
		}, s)
	}
	return s
}

// hrGreekVariants are the six variants at the end of a Greek alphabet of Unicode math: ϵ, ϑ, ϰ, ϕ, ϱ and ϖ.
var hrGreekVariants = [6]rune{'ϵ', 'ϑ', 'ϰ', 'ϕ', 'ϱ', 'ϖ'}

// hrMath gives the glyphs of a run of a math font as the plain letters and the Greek letters that pxMath knows: the italic letters of Unicode math, as 𝑎 and 𝛼, become a and α. The letters of the script and the double-struck alphabets stay, for pxSymbols.
func hrMath(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 0x1D49C && r <= 0x1D503, r >= 0x1D538 && r <= 0x1D59F:
			// script, fraktur and double-struck letters
		case r >= 0x1D6A8 && r <= 0x1D7C9:
			// the Greek letters of the five math alphabets, each a run of 58: 25 capitals, nabla, 25 small letters, the partial sign and six variants.
			i := (r - 0x1D6A8) % 58
			switch {
			case i < 25:
				r = 0x391 + i
			case i >= 26 && i < 51:
				r = 0x3B1 + i - 26
			case i > 51 && i < 58:
				r = hrGreekVariants[i-52]
			}
		default:
			r = mathPlain(r)
		}
		b.WriteRune(r)
	}
	return b.String()
}

// HrPDF reads the problems of a Croatian paper from the runs of text that pdftohtml -xml gives. A problem starts at its heading and ends at the next heading, at a section, at a stop line or at the end of the paper. The headings of a section count up from any number: a line that looks like the heading of a problem whose number does not come next is a part of the text. ink finds the drawings of a page.
func HrPDF(items []PDFItem, ink DemoInk, rules HrRules) []HrProblem {
	letters := rules.Letters
	if letters == "" {
		letters = hrLetters
	}
	items = hrJoinAccents(items)
	for i, it := range items {
		items[i].Text = HrText(it.Text)
		// Word ends a run with the space before the next run, as "Broj je " before the italic "palindrom", and the width of the run takes it in: the run ends before its spaces, so that the gap shows the space.
		if t := strings.TrimRight(items[i].Text, " "); t != "" && t != items[i].Text {
			items[i].Width = max(items[i].Width-max(it.Size*(len(items[i].Text)-len(t))/4, 3), 1)
			items[i].Text = t
		}
		if hrFonts.math.MatchString(it.Family) {
			items[i].Text = hrLegacy(it.Family, hrMath(items[i].Text))
			if hrBarRE.MatchString(items[i].Text) {
				// The bar of a segment, as the bar over AD, which Word sets as a run of its own over the letters: the problem keeps its link.
				items[i].Family = "hr-bar"
			}
		}
	}
	main := pxMainSize(items)
	lines, orphans := pxLines(items, main)
	if rules.Drop != nil {
		lines = hrDrop(lines, rules.Drop)
		// A note in a smaller size, as the names of the authors, is no line but no drawing either.
		orphans = slices.DeleteFunc(orphans, func(o PDFItem) bool { return rules.Drop.MatchString(strings.TrimSpace(o.Text)) })
	}
	lossy := false
	if rules.Lossy != nil {
		lossy = slices.ContainsFunc(items, func(it PDFItem) bool { return rules.Lossy.MatchString(it.Family) })
	}
	type start struct {
		line, from, n int
		label         string
		section       string
	}
	var starts []start
	// bounds are the lines that end a problem: the starts, the sections and the stop lines.
	bounds := map[int]bool{}
	skip := map[int]bool{}
	section := ""
	last := -1
	labelLeft := 0
	for i, l := range lines {
		t := l.text()
		switch {
		case rules.Skip != nil && rules.Skip.MatchString(t):
			skip[i] = true
			continue
		case rules.Section != nil && rules.Section.MatchString(t):
			m := rules.Section.FindStringSubmatch(t)
			section = m[0]
			for _, g := range m[1:] {
				if g != "" {
					section = g
					break
				}
			}
			bounds[i] = true
			last = -1
			continue
		case rules.Stop != nil && rules.Stop.MatchString(t):
			bounds[i] = true
			continue
		}
		// hrStart cuts the heading out of its run, so a line that is no heading gets its runs back.
		left, kept := l.items[0].Left, slices.Clone(l.items)
		label, n, from, ok := hrStart(l, rules.Start)
		if ok && rules.Not != nil && rules.Not.MatchString(strings.TrimSpace(t)) {
			ok = false
		}
		if !ok || (last >= 0 && (n != last+1 || (!rules.Titled && left > labelLeft+12))) || (last < 0 && rules.First && n != 1) {
			l.items = kept
			continue
		}
		if last < 0 {
			labelLeft = left
		}
		last = n
		starts = append(starts, start{line: i, from: from, n: n, label: label, section: section})
		bounds[i] = true
	}
	var out []HrProblem
	for _, st := range starts {
		end := st.line + 1
		for end < len(lines) && !bounds[end] {
			end++
		}
		var ls []*pxLine
		for k := st.line; k < end; k++ {
			if !skip[k] {
				ls = append(ls, hrBulletLine(lines[k], k == st.line, st.from))
			}
		}
		title := ""
		from := st.from
		if rules.Titled && from < len(ls[0].items) {
			var tb strings.Builder
			for k, it := range ls[0].items[from:] {
				if k > 0 {
					tb.WriteString(" ")
				}
				tb.WriteString(strings.TrimSpace(it.Text))
			}
			title = RuPlain(tb.String())
			from = len(ls[0].items)
		}
		p := HrProblem{DemoProblem: paperProblem(hrFonts, ls, from), N: st.n, Section: st.section}
		p.Text = hrDj(p.Text)
		if rules.Tail != nil {
			if loc := rules.Tail.FindStringIndex(p.Text); loc != nil {
				p.Text = strings.TrimSpace(p.Text[:loc[0]])
			}
		}
		if hrOddIn(p.Text, letters) && !slices.Contains(p.Flags, "glyph") {
			p.Flags = append(p.Flags, "glyph")
		}
		if hrBroken(p.Text) && !slices.Contains(p.Flags, "layout") {
			p.Flags = append(p.Flags, "layout")
		}
		if lossy && !slices.Contains(p.Flags, "glyph") {
			p.Flags = append(p.Flags, "glyph")
		}
		if title != "" && p.Text != "" {
			p.Text = "**" + title + "**\n\n" + p.Text
		}
		p.Code, p.Page = st.label, ls[0].page()
		// The rows of the problem on each of its pages: on its first page from below its heading, on the next pages from below the last line before it, and down to the line that ends it, the first skipped line below it or the end of the page.
		for pg := ls[0].page(); pg <= ls[len(ls)-1].page(); pg++ {
			from, to := 0, 1<<20
			if pg == ls[0].page() {
				from = ls[0].top() + ls[0].items[0].Height + 2
			} else {
				for k := st.line; k >= 0; k-- {
					if lines[k].page() < pg {
						break
					}
					if lines[k].page() == pg && !slices.Contains(ls, lines[k]) {
						from = lines[k].top() + lines[k].items[0].Height + 2
						break
					}
				}
			}
			for k := st.line + 1; k < len(lines); k++ {
				if lines[k].page() > pg {
					break
				}
				if lines[k].page() == pg && (k >= end || skip[k]) && lines[k].top() > from {
					to = lines[k].top() - 2
					break
				}
			}
			if ink != nil && to > from && ink(pg, from, to) {
				p.Figure = true
			}
			for _, o := range orphans {
				if o.Page == pg && o.Top >= from && o.Top < to {
					p.Figure = true
				}
			}
		}
		out = append(out, p)
	}
	return out
}

// hrBulletLine makes the bullet that starts a line, as •, into the text "-" that starts an item of a Markdown list. The first line of a problem starts at the item from.
func hrBulletLine(l *pxLine, first bool, from int) *pxLine {
	k := 0
	if first {
		k = from
	}
	if k >= len(l.items) || !slices.Contains(hrBullets, strings.TrimSpace(l.items[k].Text)) || len(l.scripts[k]) > 0 {
		return l
	}
	l.items[k].Text = "-"
	l.items[k].Family = hrBullet
	return l
}

// hrStart reads the heading of a problem at the start of a line: its label, its number, and the item where its text starts. An item that holds the end of the heading and the start of the text is cut at the end of the heading.
func hrStart(l *pxLine, re *regexp.Regexp) (label string, n, from int, ok bool) {
	var b strings.Builder
	offs := make([]int, len(l.items))
	for k, it := range l.items {
		if k > 0 && it.Left-l.items[k-1].Right() > 2 {
			b.WriteString(" ")
		}
		offs[k] = b.Len()
		b.WriteString(it.Text)
	}
	t := b.String()
	lead := len(t) - len(strings.TrimLeft(t, " "))
	m := re.FindStringSubmatchIndex(t[lead:])
	if m == nil || m[0] != 0 || m[2] < 0 {
		return "", 0, 0, false
	}
	label = t[lead+m[2] : lead+m[3]]
	num := label
	for g := len(m)/2 - 1; g >= 1; g-- {
		if m[2*g] >= 0 {
			if v, err := strconv.Atoi(t[lead+m[2*g] : lead+m[2*g+1]]); err == nil {
				num = strconv.Itoa(v)
				break
			}
		}
	}
	n, err := strconv.Atoi(num)
	if err != nil {
		return "", 0, 0, false
	}
	e := lead + m[1]
	for k, it := range l.items {
		if offs[k]+len(it.Text) <= e {
			continue
		}
		if offs[k] >= e {
			return label, n, k, true
		}
		cut := e - offs[k]
		rest := strings.TrimLeft(it.Text[cut:], " ")
		if rest == "" {
			return label, n, k + 1, true
		}
		all := utf8.RuneCountInString(it.Text)
		gone := all - utf8.RuneCountInString(rest)
		shift := it.Width * gone / max(all, 1)
		l.items[k].Text = rest
		l.items[k].Left += shift
		l.items[k].Width -= shift
		return label, n, k, true
	}
	return label, n, len(l.items), true
}
