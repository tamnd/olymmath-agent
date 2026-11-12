package official

import (
	"html"
	"math"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// RuPDFTool is the tool that gives the text layer of a PDF to RuPDFXML, with the font and the box of each piece of text.
const RuPDFTool = "pdftohtml"

var (
	rxPage = regexp.MustCompile(`<page number="(\d+)"[^>]*height="([\d.]+)" width="([\d.]+)"`)
	rxFont = regexp.MustCompile(`<fontspec id="(\d+)" size="(-?[\d.]+)" family="([^"]*)"`)
	rxText = regexp.MustCompile(`<text top="(-?[\d.]+)" left="(-?[\d.]+)" width="([\d.]+)" height="([\d.]+)" font="(\d+)">(.*)</text>`)
	rxTag  = regexp.MustCompile(`<[^>]*>`)
	// rxMathFont names the fonts of TeX that set only math: letters, symbols, big signs and the fonts of the AMS.
	rxMathFont = regexp.MustCompile(`^(?:CMMI|CMSY|CMEX|MSAM|MSBM|EUFM|EUSM|EURM|RSFS|CMBSY|CMMIB|LASY|STMARY|WASY)`)
	// rxRomanFont names the roman fonts of TeX, which set the digits and signs of math and, in some papers, the digits of the text.
	rxRomanFont = regexp.MustCompile(`^(?:CMR|CMBX)\d`)
	// rxMarker is the start of a problem of a grade paper: the grade, the number and dots, as 9.1. or 10.5.
	rxMarker = regexp.MustCompile(`^(\d{1,2})\.(\d{1,2})\.\s*`)
	// rxHeader is a line of the head of a sheet: the name of the olympiad, the grade or the day.
	rxHeader = regexp.MustCompile(`(?i)^(?:[XVIL]+\s+)?Всероссийская.*олимпиада|^\d{1,2}\s+класс$|^(?:Первый|Второй)\s+день$|^\d+$`)
	// rxItem is a line that starts an item of a list, as а) or б).
	rxItem = regexp.MustCompile(`^[а-жa-f]\)`)
	rxDots = regexp.MustCompile(`\.\s*\.\s*\.`)
	// rxDotsOnly is a piece of text of three dots with spaces between them and marks of punctuation round them.
	rxDotsOnly = regexp.MustCompile(`^[,;\s]*\.\s+\.\s+\.[,;\s]*$`)
	rxFig      = regexp.MustCompile(`(?i)\bрис(?:\.|унк)`)
	// rxCmdEnd is the end of a command of TeX, as \angle.
	rxCmdEnd = regexp.MustCompile(`\\[A-Za-z]+$`)
	rxNotEq  = regexp.MustCompile(`\\not\s*=|=\s*\\not\b`)
	rxGreek  = regexp.MustCompile(`(?:^|[\s(])([ΓΔΘΛΞΠΣΦΨΩ])(?:$|[\s.,;:)])`)
	rxCapGap = regexp.MustCompile(`([A-Z]) ([A-Z])`)
	rxDotOut = regexp.MustCompile(`\$ ([.,;:!?])`)
	rxNotIn  = regexp.MustCompile(`(\\not|/)\s*\\in\b`)
)

// pdfFont is a font of the output of pdftohtml: its name and its size.
type pdfFont struct {
	family string
	size   float64
}

// pdfFrag is one piece of text of a page: a run of letters in one font.
type pdfFrag struct {
	x0, x1, top, h, mid float64
	text                string
	// kind is 's' for a font of math, 'w' for a roman font of TeX, and 't' for a font of text.
	kind byte
	// script is '^' or '_' for a small piece above or below the line, and 0 for a piece on the line.
	script byte
	// size is the size of the font, and small is true for a piece of math in a font well smaller than most of the page.
	size  float64
	small bool
}

// pdfLine is one line of a page, with its pieces from left to right.
type pdfLine struct {
	mid   float64
	frags []pdfFrag
	page  int
	// close is true when the line is nearer to the line before than the lines of the text are, as the parts of a fraction.
	close bool
	gap   float64
}

// RuPDFXML reads the problems of the sheets of a Russian olympiad from the output of pdftohtml -xml of the PDF pdf. A problem starts at a line that starts with the grade and its number, as 9.5., and its position is pG N, as p95. The fonts of TeX tell the math from the text: the math becomes TeX, a small piece above or below the line becomes a superscript or a subscript, and a problem with a fraction, a root or a big sign gets the layout flag. A page in landscape holds two copies of a sheet side by side, so only its left half is read. The first problem at a position counts. The letters and signs that the text layer loses are put back from the content of the PDF (see pdfLost); a problem that has one of them with no place gets the layout flag.
func RuPDFXML(xml, pdf []byte, edition, url string) []RuProblem {
	out, _ := ruPDF(xml, pdf, edition, url)
	return out
}

// ruPDF is RuPDF that also gives the lost letters and signs that found no place.
func ruPDF(xml, pdf []byte, edition, url string) ([]RuProblem, []pdfLostKey) {
	fonts := map[string]pdfFont{}
	var all []pdfLine
	// read tells which lines are read: the lines of the left half of a page in landscape, and all lines of another page.
	var read []bool
	s := string(xml)
	pages := rxPage.FindAllStringSubmatchIndex(s, -1)
	for i, pm := range pages {
		end := len(s)
		if i+1 < len(pages) {
			end = pages[i+1][0]
		}
		body := s[pm[0]:end]
		for _, f := range rxFont.FindAllStringSubmatch(body, -1) {
			fam := f[3]
			if j := strings.IndexByte(fam, '+'); j >= 0 {
				fam = fam[j+1:]
			}
			size, _ := strconv.ParseFloat(f[2], 64)
			fonts[f[1]] = pdfFont{strings.ToUpper(fam), size}
		}
		num, _ := strconv.Atoi(s[pm[2]:pm[3]])
		height, _ := strconv.ParseFloat(s[pm[4]:pm[5]], 64)
		width, _ := strconv.ParseFloat(s[pm[6]:pm[7]], 64)
		frags, big := pdfFrags(body, fonts)
		if width > height*1.1 {
			var left, right []pdfFrag
			for _, f := range frags {
				if f.x0 < width/2 {
					left = append(left, f)
				} else {
					right = append(right, f)
				}
			}
			l := pdfLines(left, big, num)
			all = append(all, l...)
			read = append(read, make([]bool, len(l))...)
			for k := len(read) - len(l); k < len(read); k++ {
				read[k] = true
			}
			r := pdfLines(right, big, num)
			all = append(all, r...)
			read = append(read, make([]bool, len(r))...)
			continue
		}
		l := pdfLines(frags, big, num)
		all = append(all, l...)
		for range l {
			read = append(read, true)
		}
	}
	var missed []pdfLostKey
	if len(pdf) > 0 {
		missed = pdfRepair(all, pdfLost(pdf))
	}
	var keep []pdfLine
	for i, l := range all {
		if read[i] {
			keep = append(keep, l)
		}
	}
	out := pdfProblems(keep, edition, url)
	// A problem with a lost letter or sign that found no place goes to review.
	for _, k := range missed {
		for i := range out {
			if strings.Contains(pdfKeys(out[i].Text), k.before+k.after) {
				out[i].Flags = pdfFlag(out[i].Flags, "layout")
			}
		}
	}
	return out, missed
}

// pdfFrags reads the pieces of text of one page, and the height of most of its letters.
func pdfFrags(body string, fonts map[string]pdfFont) ([]pdfFrag, float64) {
	var frags []pdfFrag
	var heights []float64
	for _, m := range rxText.FindAllStringSubmatch(body, -1) {
		top, _ := strconv.ParseFloat(m[1], 64)
		left, _ := strconv.ParseFloat(m[2], 64)
		w, _ := strconv.ParseFloat(m[3], 64)
		h, _ := strconv.ParseFloat(m[4], 64)
		text := html.UnescapeString(rxTag.ReplaceAllString(m[6], ""))
		fam := fonts[m[5]].family
		f := pdfFrag{x0: left, x1: left + w, top: top, h: h, kind: 't', size: fonts[m[5]].size}
		switch {
		case rxMathFont.MatchString(fam):
			f.kind = 's'
			if strings.HasPrefix(fam, "CMSY") {
				// Some signs of CMSY have no Unicode name and come out as the Latin letter of their code: the slash that crosses the next sign, as in ≠, is 6, and ∥ is k.
				switch strings.TrimSpace(text) {
				case "6":
					text = "̸"
				case "k":
					text = "∥"
				case "0":
					text = "′"
				}
			}
			// A space between two capital letters, as in P Q, is the slant of the letter.
			for rxCapGap.MatchString(text) {
				text = rxCapGap.ReplaceAllString(text, "$1$2")
			}
		case rxRomanFont.MatchString(fam):
			f.kind = 'w'
		default:
			text = pdfFixText(text)
		}
		f.text = text
		if strings.TrimSpace(text) == "" {
			continue
		}
		frags = append(frags, f)
		if h > 0 {
			heights = append(heights, h)
		}
	}
	big := pdfMode(heights)
	// A piece of math in a font well smaller than most of the pieces is small: a superscript, a subscript or a part of a fraction. The size of the font tells this better than the height of the piece, which depends on the letters.
	var sizes []float64
	for _, f := range frags {
		if f.size > 0 {
			sizes = append(sizes, f.size)
		}
	}
	common := pdfMode(sizes)
	for i := range frags {
		frags[i].small = frags[i].kind != 't' && frags[i].size > 0 && frags[i].size < 0.9*common
	}
	for i := range frags {
		f := &frags[i]
		if f.h > 0 {
			f.mid = f.top + f.h/2
		} else {
			// A font with no size puts the top of the piece on its base line.
			f.mid = f.top - 0.28*big
		}
	}
	return frags, big
}

// pdfLines puts the pieces of one page, or of one half of a page, into lines.
func pdfLines(frags []pdfFrag, big float64, page int) []pdfLine {
	if len(frags) == 0 {
		return nil
	}
	var main, scripts []pdfFrag
	for _, f := range frags {
		if f.small {
			scripts = append(scripts, f)
		} else {
			main = append(main, f)
		}
	}
	sort.SliceStable(main, func(i, j int) bool { return main[i].mid < main[j].mid })
	var lines []pdfLine
	for i, f := range main {
		// A piece near the piece before it in height is on the same line.
		if n := len(lines); n > 0 && f.mid-main[i-1].mid <= 0.3*big {
			lines[n-1].frags = append(lines[n-1].frags, f)
			continue
		}
		lines = append(lines, pdfLine{mid: f.mid, frags: []pdfFrag{f}, page: page})
	}
	// The middle of a line is the median of the middles of its pieces.
	for i := range lines {
		var mids []float64
		for _, f := range lines[i].frags {
			mids = append(mids, f.mid)
		}
		lines[i].mid = pdfMedian(mids)
	}
	for _, f := range scripts {
		best, bd := -1, math.Inf(1)
		for i, l := range lines {
			if d := math.Abs(f.mid - l.mid); d < bd {
				best, bd = i, d
			}
		}
		if best < 0 || bd > 0.65*big {
			// A small piece far from each line is a line of its own.
			lines = append(lines, pdfLine{mid: f.mid, frags: []pdfFrag{f}, page: page})
			continue
		}
		f.script = pdfScript(f, lines[best])
		lines[best].frags = append(lines[best].frags, f)
	}
	sort.SliceStable(lines, func(i, j int) bool { return lines[i].mid < lines[j].mid })
	var gaps []float64
	for i := 1; i < len(lines); i++ {
		lines[i].gap = lines[i].mid - lines[i-1].mid
		gaps = append(gaps, lines[i].gap)
	}
	step := pdfMedian(gaps)
	for i := 1; i < len(lines); i++ {
		lines[i].close = lines[i].gap < 0.75*step
	}
	for i := range lines {
		fs := lines[i].frags
		sort.SliceStable(fs, func(a, b int) bool { return fs[a].x0 < fs[b].x0 })
		// A small sign between two pieces of one superscript or subscript, as + in A_{i+1}, is a part of it.
		for j := 1; j+1 < len(fs); j++ {
			f := &fs[j]
			if f.small && f.script == 0 && pdfSign(f.text) && fs[j-1].script != 0 && fs[j-1].script == fs[j+1].script && f.x0-fs[j-1].x1 < 1.5 && fs[j+1].x0-f.x1 < 1.5 {
				f.script = fs[j-1].script
			}
		}
		lines[i].gap /= max(step, 1)
	}
	return lines
}

// pdfScript tells if a small piece of a line is a superscript ('^'), a subscript ('_') or on the line (0). The piece of math just before it on the line tells: the bottom of a superscript is well above its bottom, and the top of a subscript is below its top. A sign of an operation, as −, can be smaller than the letters while it is on the line, so it is a superscript only when it is well above the line and it is never a subscript. With no piece of math before it, the middle of the line tells.
func pdfScript(f pdfFrag, l pdfLine) byte {
	sign := pdfSign(f.text)
	var ref *pdfFrag
	for i := range l.frags {
		g := &l.frags[i]
		if g.h > 0 && !g.small && g.kind != 't' && g.x1 <= f.x0+1.5 && f.x0-g.x1 < 12 && (ref == nil || g.x1 > ref.x1) {
			ref = g
		}
	}
	if ref != nil {
		bottom := f.top + f.h - (ref.top + ref.h)
		switch {
		case sign && bottom <= -6, !sign && bottom <= -4:
			return '^'
		case !sign && f.top-ref.top >= 2:
			return '_'
		}
		return 0
	}
	lim := 1.5
	if sign {
		lim = 5
	}
	switch d := f.mid - l.mid; {
	case d < -lim:
		return '^'
	case d > lim && !sign:
		return '_'
	}
	return 0
}

// pdfFixText repairs a piece of a font of text whose letters came out as Latin-1 while they are Windows-1251, as Âñåðîññèéñêàÿ for Всероссийская.
func pdfFixText(s string) string {
	odd := false
	for _, r := range s {
		if r > 0xff {
			return s
		}
		if r >= 0xc0 {
			odd = true
		}
	}
	if !odd {
		return s
	}
	b := make([]byte, 0, len(s))
	for _, r := range s {
		b = append(b, byte(r)) // #nosec G115 -- each rune is below 0x100, as the loop before tells.
	}
	out, err := charmap.Windows1251.NewDecoder().Bytes(b)
	if err != nil {
		return s
	}
	// The T2A encoding of TeX has ё and Ё where Windows-1251 has ј and њ, which Russian does not use.
	return strings.NewReplacer("ј", "ё", "њ", "Ё").Replace(string(out))
}

// pdfSign tells if a piece holds only signs, as − or ≤, which a font of math can set smaller than the letters. A prime, a degree and a star are not such signs, since they are superscripts.
func pdfSign(t string) bool {
	t = strings.TrimSpace(t)
	if t == "" || strings.ContainsAny(t, "′◦∗*'") {
		return false
	}
	for _, r := range t {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func pdfMode(xs []float64) float64 {
	n := map[float64]int{}
	best, bn := 0.0, 0
	for _, x := range xs {
		n[x]++
		if n[x] > bn || (n[x] == bn && x > best) {
			best, bn = x, n[x]
		}
	}
	if best == 0 {
		return 18
	}
	return best
}

func pdfMedian(xs []float64) float64 {
	if len(xs) == 0 {
		return 20
	}
	ys := slices.Clone(xs)
	slices.Sort(ys)
	return ys[len(ys)/2]
}

// pdfPiece is a word of a problem: text, or math in TeX. strong math is math by its font, and weak math is a number or a sign of a roman font that joins the math next to it.
type pdfPiece struct {
	tex          string
	math, strong bool
	// sep is what comes before the piece: "", " ", or "\n" at the start of a line.
	sep string
}

// pdfDraft is a problem while its lines are read.
type pdfDraft struct {
	p      RuProblem
	pieces []pdfPiece
	lines  int
}

func pdfProblems(lines []pdfLine, edition, url string) []RuProblem {
	var out []RuProblem
	seen := map[string]bool{}
	var cur *pdfDraft
	flush := func() {
		if cur == nil {
			return
		}
		cur.p.Text = pdfText(cur.pieces)
		if !seen[cur.p.Position] && cur.p.Text != "" {
			seen[cur.p.Position] = true
			if rxFig.MatchString(cur.p.Text) {
				cur.p.Figure = true
			}
			out = append(out, cur.p)
		}
		cur = nil
	}
	for _, l := range lines {
		pieces, flags := pdfLinePieces(l)
		plain := strings.TrimSpace(pdfText(pieces))
		if rxHeader.MatchString(plain) {
			continue
		}
		if m := rxMarker.FindStringSubmatch(plain); m != nil && len(pieces) > 0 && !pieces[0].strong {
			flush()
			cur = &pdfDraft{p: RuProblem{Edition: edition, Position: "p" + m[1] + m[2], URL: url, Page: l.page, Tool: RuPDFTool}}
			// The marker is the start of the first piece of text.
			first := strings.TrimLeft(pieces[0].tex, " ")
			loc := rxMarker.FindStringIndex(first)
			if loc != nil {
				pieces[0].tex = first[loc[1]:]
			}
		} else if cur == nil {
			continue
		}
		if cur.lines > 0 {
			if l.close {
				cur.p.Flags = pdfFlag(cur.p.Flags, "layout")
			}
			if l.gap > 2.5 {
				cur.p.Figure = true
			}
		}
		if pdfLabels(l) {
			cur.p.Figure = true
		}
		for _, f := range flags {
			cur.p.Flags = pdfFlag(cur.p.Flags, f)
		}
		if len(pieces) > 0 {
			pieces[0].sep = "\n"
		}
		cur.pieces = append(cur.pieces, pieces...)
		cur.lines++
	}
	flush()
	for i := range out {
		out[i].Text = strings.TrimSpace(ruAuthor.ReplaceAllString(out[i].Text, ""))
	}
	return out
}

func pdfFlag(flags []string, f string) []string {
	if slices.Contains(flags, f) {
		return flags
	}
	return append(flags, f)
}

// pdfLabels tells if a line holds only a few letters of math, as the labels of a drawing.
func pdfLabels(l pdfLine) bool {
	n := 0
	for _, f := range l.frags {
		if f.kind != 's' {
			return false
		}
		n += utf8.RuneCountInString(strings.TrimSpace(f.text))
	}
	return n <= 2
}

// pdfLinePieces makes the pieces of one line. A superscript or a subscript goes on the piece of math before it. flags are the warnings of the line.
func pdfLinePieces(l pdfLine) ([]pdfPiece, []string) {
	var out []pdfPiece
	var flags []string
	last := -1.0
	prevEnd := math.Inf(-1)
	for i := 0; i < len(l.frags); i++ {
		f := l.frags[i]
		if f.script != 0 {
			// A superscript and a subscript at the same place go together, as in x_i^2.
			sub, sup := "", ""
			add := func(g pdfFrag) {
				t := pdfMathTeX(g.text, &flags)
				if g.script == '^' {
					sup += t
				} else {
					sub += t
				}
			}
			add(f)
			end := f.x1
			for i+1 < len(l.frags) && l.frags[i+1].script != 0 && l.frags[i+1].x0 < end+1.5 {
				i++
				add(l.frags[i])
				end = max(end, l.frags[i].x1)
			}
			if sub != "" && sup != "" && (f.x0-prevEnd > 1.5 || len(out) == 0 || !pdfBase(out[len(out)-1].tex)) {
				// Two small pieces one above the other with no letter before them are a fraction.
				flags = append(flags, "layout")
			}
			if strings.ContainsAny(sup, "→⃗¯˜") || strings.Contains(sup, `\to`) {
				flags = append(flags, "layout")
			}
			t := ""
			if sub != "" {
				t += "_" + pdfGroup(sub)
			}
			if sup != "" {
				if strings.Trim(sup, "'") == "" {
					t += sup
				} else {
					t += "^" + pdfGroup(sup)
				}
			}
			if n := len(out); n > 0 && f.x0-prevEnd < 3 {
				out[n-1].tex += t
				out[n-1].math, out[n-1].strong = true, true
			} else {
				// A small piece with nothing before it is a part of a fraction or of a sign that the text lost.
				flags = append(flags, "layout")
				out = append(out, pdfPiece{tex: "{}" + t, math: true, strong: true, sep: " "})
			}
			prevEnd = end
			last = end
			continue
		}
		sep := ""
		if last >= 0 && f.x0-last > 2.5 {
			sep = " "
		}
		switch f.kind {
		case 's':
			out = append(out, pdfPiece{tex: pdfMathTeX(f.text, &flags), math: true, strong: true, sep: sep})
		case 'w':
			// The name of a function, as cos, is in the roman font.
			// The name of a function is math, as a letter of the math font is.
			t := ruFuncTeX(pdfMathTeX(f.text, &flags))
			out = append(out, pdfPiece{tex: t, math: true, strong: strings.HasPrefix(t, `\`), sep: sep})
		default:
			t := f.text
			if strings.HasPrefix(t, " ") && sep == "" && last >= 0 {
				sep = " "
			}
			out = append(out, pdfGreek(strings.TrimSpace(ruSpaces.ReplaceAllString(t, " ")), sep)...)
			if strings.HasSuffix(t, " ") {
				f.x1 += 3
			}
		}
		last = f.x1
		prevEnd = f.x1
	}
	return out, flags
}

func pdfGroup(s string) string {
	if utf8.RuneCountInString(s) == 1 {
		return s
	}
	return "{" + s + "}"
}

// pdfMathTeX puts a piece of math into TeX. A root or a big sign has a part that the text layer does not give, so it adds the layout flag.
func pdfMathTeX(s string, flags *[]string) string {
	s = strings.TrimSpace(s)
	s = rxDots.ReplaceAllString(s, "…")
	var b strings.Builder
	rs := []rune(s)
	for i, r := range rs {
		t, ok := ruSymbols[r]
		switch {
		case r == '√' || r == '∑' || r == '∏' || r == '∫':
			*flags = append(*flags, "layout")
		case r == '′':
			t, ok = "'", true
		case r == '∗':
			t, ok = "*", true
		case r == '·':
			t, ok = `\cdot`, true
		case r == '◦':
			t, ok = `\circ`, true
		case r == '{' || r == '}':
			t, ok = `\`+string(r), true
		case r == '̸':
			t, ok = `\not`, true
		case r == 'ℓ':
			t, ok = `\ell`, true
		case pdfGreekTeX[string(r)] != "":
			t, ok = pdfGreekTeX[string(r)], true
		}
		if !ok {
			t = string(r)
		}
		b.WriteString(t)
		if ok && strings.HasPrefix(t, `\`) && i+1 < len(rs) && (unicode.IsLetter(rs[i+1]) || unicode.IsDigit(rs[i+1])) {
			b.WriteString(" ")
		}
	}
	return b.String()
}

// pdfOp tells if the TeX ends (or, with first, starts) with a sign that TeX repeats at a break of a line, as = or \le.
func pdfOp(t string, first bool) string {
	t = strings.TrimSpace(t)
	for _, op := range []string{`\le`, `\ge`, `\ne`, `\cdot`, `\times`, "=", "<", ">", "+", "-"} {
		if first && strings.HasPrefix(t, op) || !first && strings.HasSuffix(t, op) {
			return op
		}
	}
	return ""
}

// pdfText joins the pieces of a problem. Weak math joins strong math next to it, the pieces of one run of math go between dollars, a word cut at the end of a line is joined again, and a sign that TeX repeats at a break of a line inside math is kept once.
func pdfText(ps []pdfPiece) string {
	ps = pdfDots(ps)
	n := len(ps)
	isMath := make([]bool, n)
	for i, p := range ps {
		isMath[i] = p.strong
	}
	// A weak piece is math when strong math is next to it, through other weak pieces.
	for changed := true; changed; {
		changed = false
		for i, p := range ps {
			if isMath[i] || !p.math || !pdfWeakJoins(p.tex) {
				continue
			}
			if (i > 0 && isMath[i-1]) || (i+1 < n && isMath[i+1]) {
				isMath[i] = true
				changed = true
			}
		}
	}
	var b strings.Builder
	inMath := false
	lastMath := ""
	for i, p := range ps {
		sep := p.sep
		if sep == "\n" {
			if rxItem.MatchString(p.tex) {
				sep = "\n"
			} else {
				sep = " "
				cur := b.String()
				if !isMath[i] && !inMath && strings.HasSuffix(cur, "-") && pdfLetterBefore(cur) && pdfLower(p.tex) {
					// A word cut at the end of the line.
					s := strings.TrimSuffix(cur, "-")
					b.Reset()
					b.WriteString(s)
					sep = ""
				}
			}
		}
		if isMath[i] {
			t := p.tex
			if inMath && p.sep == "\n" {
				if op := pdfOp(lastMath, false); op != "" && pdfOp(t, true) == op {
					t = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(t), op))
				}
			}
			if !inMath {
				b.WriteString(sep)
				b.WriteString("$")
				inMath = true
			} else if sep != "" && !pdfCapitals(lastMath, t) || sep == "" && rxCmdEnd.MatchString(lastMath) && pdfLetterStart(t) {
				// The gap between two capital letters, as in P Q, is the slant of the letter, and TeX does not show it.
				// A command of TeX, as \angle, needs a space before a letter.
				b.WriteString(" ")
			}
			b.WriteString(t)
			lastMath = t
			continue
		}
		if inMath {
			pdfCloseMath(&b)
			inMath = false
		}
		b.WriteString(sep)
		b.WriteString(p.tex)
	}
	if inMath {
		pdfCloseMath(&b)
	}
	out := b.String()
	out = strings.ReplaceAll(out, "$$", "")
	out = rxNotEq.ReplaceAllString(out, `\ne`)
	out = rxNotIn.ReplaceAllString(out, `\notin`)
	out = rxDotOut.ReplaceAllString(out, "$$$1")
	out = ruSpaces.ReplaceAllString(out, " ")
	out = strings.ReplaceAll(out, " \n", "\n")
	out = strings.ReplaceAll(out, "\n ", "\n")
	return strings.TrimSpace(out)
}

// pdfGreekTeX are the TeX names of the capital Greek letters, which the text font of some papers has.
var pdfGreekTeX = map[string]string{
	"Γ": `\Gamma`, "Δ": `\Delta`, "Θ": `\Theta`, "Λ": `\Lambda`, "Ξ": `\Xi`, "Π": `\Pi`, "Σ": `\Sigma`, "Φ": `\Phi`, "Ψ": `\Psi`, "Ω": `\Omega`,
}

// pdfGreek makes the pieces of a piece of text, where a capital Greek letter alone is math.
func pdfGreek(t, sep string) []pdfPiece {
	var out []pdfPiece
	for {
		loc := rxGreek.FindStringSubmatchIndex(t)
		if loc == nil {
			break
		}
		if before := strings.TrimSpace(t[:loc[2]]); before != "" {
			out = append(out, pdfPiece{tex: before, sep: sep})
			sep = " "
		}
		if loc[2] > 0 && t[loc[2]-1] != ' ' {
			sep = ""
		}
		out = append(out, pdfPiece{tex: pdfGreekTeX[t[loc[2]:loc[3]]], math: true, strong: true, sep: sep})
		sep = ""
		t = t[loc[3]:]
		if strings.HasPrefix(t, " ") {
			sep = " "
		}
		t = strings.TrimSpace(t)
	}
	if t != "" || len(out) == 0 {
		out = append(out, pdfPiece{tex: t, sep: sep})
	}
	return out
}

// pdfBase tells if math ends with a sign that can take a subscript and a superscript together: a letter, a digit or a bracket that closes.
func pdfBase(t string) bool {
	r, _ := utf8.DecodeLastRuneInString(t)
	return r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r)) || strings.ContainsRune(")}]'", r)
}

// pdfCapitals tells if math a ends with a Latin letter that is not a part of a command, and math b starts with a capital Latin letter or a bracket, as in P Q or P (x). The gap is the slant of the letter.
func pdfCapitals(a, b string) bool {
	capital := func(r byte) bool { return r >= 'A' && r <= 'Z' }
	if a == "" || b == "" || !capital(b[0]) && b[0] != '(' {
		return false
	}
	if r := a[len(a)-1]; !capital(r) && (b[0] != '(' || r < 'a' || r > 'z') {
		return false
	}
	return !rxCmdEnd.MatchString(a)
}

// pdfWeakJoins tells if a weak piece may join math: a number or a sign, and not a lone mark of punctuation.
func pdfWeakJoins(t string) bool {
	return strings.ContainsAny(t, "0123456789=+-<>()[]/|!") || strings.Contains(t, `\`)
}

// pdfCloseMath ends a run of math. A mark of punctuation at the end of the math goes after the dollar.
func pdfCloseMath(b *strings.Builder) {
	s := b.String()
	end := strings.TrimRight(s, " ")
	if n := len(end); n > 0 && strings.ContainsRune(".,;:", rune(end[n-1])) && !strings.HasSuffix(end, `\ldots`) && !strings.HasSuffix(end, "$.") {
		mark := end[n-1:]
		body := strings.TrimRight(end[:n-1], " ")
		if !strings.HasSuffix(body, "$") {
			b.Reset()
			b.WriteString(body + "$" + mark)
			return
		}
	}
	b.WriteString("$")
}

// pdfDots makes a run of pieces of three dots, as ". .. ", into the math sign \ldots.
func pdfDots(ps []pdfPiece) []pdfPiece {
	var out []pdfPiece
	for i := 0; i < len(ps); i++ {
		dots, j := 0, i
		for j < len(ps) && strings.Trim(ps[j].tex, ". ") == "" && ps[j].tex != "" && dots < 3 {
			dots += strings.Count(ps[j].tex, ".")
			j++
		}
		if dots == 3 && j-i > 1 {
			out = append(out, pdfPiece{tex: `\ldots`, math: true, strong: true, sep: ps[i].sep})
			i = j - 1
			continue
		}
		if p := ps[i]; !p.math && rxDotsOnly.MatchString(p.tex) {
			// Dots with spaces between them in the text font, as in 1, . . . , n, are the dots of math.
			loc := rxDots.FindStringIndex(p.tex)
			if before := strings.TrimSpace(p.tex[:loc[0]]); before != "" {
				out = append(out, pdfPiece{tex: before, sep: p.sep})
				p.sep = " "
			}
			out = append(out, pdfPiece{tex: `\ldots`, math: true, strong: true, sep: p.sep})
			if after := strings.TrimSpace(p.tex[loc[1]:]); after != "" {
				out = append(out, pdfPiece{tex: after})
			}
			continue
		}
		out = append(out, ps[i])
	}
	return out
}

func pdfLetterStart(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsLetter(r)
}

func pdfLetterBefore(s string) bool {
	s = strings.TrimSuffix(s, "-")
	r, _ := utf8.DecodeLastRuneInString(s)
	return unicode.IsLetter(r)
}

func pdfLower(s string) bool {
	r, _ := utf8.DecodeRuneInString(s)
	return unicode.IsLower(r)
}
