package official

import (
	"html"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// PDFItem is one run of text of a page of a PDF, as pdftohtml -xml gives it: its box in pixels from the top left corner of the page, its font and its text.
type PDFItem struct {
	Page                     int
	Top, Left, Width, Height int
	Size                     int
	Family                   string
	Text                     string
}

// Right gives the right edge of the item.
func (it PDFItem) Right() int { return it.Left + it.Width }

func (it PDFItem) center() float64 { return float64(it.Top) + float64(it.Height)/2 }

// Math tells if the font of the item is a font of TeX math: math italic, math symbols, the roman of math, the large symbols and the AMS symbols. The text of a paper made with TeX has other fonts.
func (it PDFItem) Math() bool { return pxMathFont.MatchString(it.Family) }

// Italic tells if the font of the item is an italic text font, as the names of the authors of a paper.
func (it PDFItem) Italic() bool { return pxItalicFont.MatchString(it.Family) }

var (
	pxFontRE     = regexp.MustCompile(`<fontspec id="(\d+)" size="(-?\d+)" family="([^"]*)"`)
	pxTextRE     = regexp.MustCompile(`<text top="(-?\d+)" left="(-?\d+)" width="(-?\d+)" height="(-?\d+)" font="(\d+)">(.*?)</text>`)
	pxPageRE     = regexp.MustCompile(`<page number="(\d+)"`)
	pxInnerTag   = regexp.MustCompile(`</?[a-z]+[^>]*>`)
	pxMathFont   = regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:CMMI|CMSY|CMR|CMEX|MSAM|MSBM|CMBSY|CMMIB|EUFM|EUSM|RSFS)\d`)
	pxItalicFont = regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:SFTI|SFSI|CMTI|LCMSSI)\d`)
)

// ParsePDFXML reads the output of pdftohtml -xml: the runs of text of all pages, in the order of the file.
func ParsePDFXML(b []byte) []PDFItem {
	type font struct {
		size   int
		family string
	}
	fonts := map[string]font{}
	page := 0
	var out []PDFItem
	for _, line := range strings.Split(string(b), "\n") {
		if m := pxPageRE.FindStringSubmatch(line); m != nil {
			page, _ = strconv.Atoi(m[1])
			continue
		}
		if m := pxFontRE.FindStringSubmatch(line); m != nil {
			size, _ := strconv.Atoi(m[2])
			fonts[m[1]] = font{size, m[3]}
			continue
		}
		m := pxTextRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		n := make([]int, 4)
		for i := range n {
			n[i], _ = strconv.Atoi(m[i+1])
		}
		f := fonts[m[5]]
		text := html.UnescapeString(pxInnerTag.ReplaceAllString(m[6], ""))
		if strings.TrimSpace(text) == "" {
			continue
		}
		out = append(out, PDFItem{Page: page, Top: n[0], Left: n[1], Width: n[2], Height: n[3], Size: f.size, Family: f.family, Text: text})
	}
	return out
}

// pxLine is one line of a page: its items of the main size from left to right, and the small items that are the scripts of them.
type pxLine struct {
	items   []PDFItem
	scripts map[int][]pxScript
}

// pxScript is a superscript or a subscript of an item of a line.
type pxScript struct {
	PDFItem
	sup bool
}

func (l *pxLine) top() int   { return l.items[0].Top }
func (l *pxLine) page() int  { return l.items[0].Page }
func (l *pxLine) right() int { return l.items[len(l.items)-1].Right() }
func (l *pxLine) center() float64 {
	var c float64
	for _, it := range l.items {
		c += it.center()
	}
	return c / float64(len(l.items))
}

// text gives the plain text of the line, with no scripts.
func (l *pxLine) text() string {
	var b strings.Builder
	for i, it := range l.items {
		if i > 0 && it.Left-l.items[i-1].Right() > 2 {
			b.WriteString(" ")
		}
		b.WriteString(it.Text)
	}
	return RuPlain(b.String())
}

// pxMainSize gives the most common size of the items of the text fonts.
func pxMainSize(items []PDFItem) int {
	count := map[int]int{}
	for _, it := range items {
		if !it.Math() {
			count[it.Size] += len(it.Text)
		}
	}
	best, n := 0, -1
	for s, c := range count {
		if c > n || (c == n && s > best) {
			best, n = s, c
		}
	}
	return best
}

// pxLines groups the items into lines. An item of the main size joins the line whose middle is near its middle. A smaller item is a script of the item of a line that ends where it starts; a smaller item with no such item is an orphan, as a label of a figure.
func pxLines(items []PDFItem, main int) (lines []*pxLine, orphans []PDFItem) {
	var big, small []PDFItem
	for _, it := range items {
		if float64(it.Size) >= 0.85*float64(main) {
			big = append(big, it)
		} else {
			small = append(small, it)
		}
	}
	sort.SliceStable(big, func(i, j int) bool {
		if big[i].Page != big[j].Page {
			return big[i].Page < big[j].Page
		}
		return big[i].Top < big[j].Top
	})
	for _, it := range big {
		var at *pxLine
		for k := len(lines) - 1; k >= 0 && k >= len(lines)-3; k-- {
			l := lines[k]
			if l.page() == it.Page && pxAbs(l.center()-it.center()) <= 4 {
				at = l
				break
			}
		}
		if at == nil {
			at = &pxLine{scripts: map[int][]pxScript{}}
			lines = append(lines, at)
		}
		at.items = append(at.items, it)
	}
	for _, l := range lines {
		sort.SliceStable(l.items, func(i, j int) bool { return l.items[i].Left < l.items[j].Left })
	}
	slices.SortStableFunc(lines, func(a, b *pxLine) int {
		if a.page() != b.page() {
			return a.page() - b.page()
		}
		return a.top() - b.top()
	})
	for _, s := range small {
		found := false
		for _, l := range lines {
			if l.page() != s.Page || s.center() < float64(l.items[0].Top)-float64(s.Height) || s.center() > float64(l.items[0].Top+l.items[0].Height)+float64(s.Height) {
				continue
			}
			for i, it := range l.items {
				base := it.Right()
				if ss := l.scripts[i]; len(ss) > 0 {
					base = max(base, ss[len(ss)-1].Right())
				}
				if s.Left >= it.Right()-3 && s.Left <= base+3 {
					l.scripts[i] = append(l.scripts[i], pxScript{s, s.center() < l.center()})
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			orphans = append(orphans, s)
		}
	}
	return lines, orphans
}

func pxAbs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

// pxSymbols are the TeX forms of the glyphs of the math fonts, beyond the symbols of a Russian page.
var pxSymbols = map[rune]string{
	'{': `\{`, '}': `\}`, '◦': `\circ`, '∗': `*`, '′': `'`, '∼': `\sim`, '÷': `\div`, '⊆': `\subseteq`, '∅': `\varnothing`, '∤': `\nmid`,
	'θ': `\theta`, 'ϕ': `\phi`, 'σ': `\sigma`, 'τ': `\tau`, 'ρ': `\rho`, 'ν': `\nu`, 'ξ': `\xi`, 'η': `\eta`, 'ζ': `\zeta`, 'κ': `\kappa`,
	'ψ': `\psi`, 'χ': `\chi`, 'ϵ': `\epsilon`, 'Γ': `\Gamma`, 'Λ': `\Lambda`, 'Σ': `\Sigma`, 'Φ': `\Phi`, 'Π': `\Pi`, 'Θ': `\Theta`,
	'⩽': `\leqslant`, '⩾': `\geqslant`, 'ℓ': `\ell`, '•': `\bullet`, '∉': `\notin`, '̸': `\not`, 'Ω': `\Omega`,
}

// pxMath gives the TeX of the glyphs of one run of a math font, and false when a glyph has no known TeX form or is a part of a formula that a line cannot hold, as a root sign or a large symbol.
func pxMath(s string, roman bool) (string, bool) {
	s = strings.ReplaceAll(s, ". . .", "…")
	s = strings.ReplaceAll(s, "· · ·", "…")
	// A slash over the next glyph negates it: ̸= is ≠.
	s = pxNot.Replace(s)
	if roman {
		s = ruFuncTeX(s)
	}
	var b strings.Builder
	ok := true
	for _, r := range s {
		t, known := pxSymbols[r]
		if !known {
			t, known = ruSymbols[r]
		}
		switch {
		case r == ' ':
			continue
		case r == '√' || r == '∑' || r == '∫' || r == '∏':
			ok = false
			t = string(r)
		case known && pxBinary[t]:
			t = " " + t + " "
		case known:
		case r < 128:
			t = string(r)
		default:
			ok = false
			t = string(r)
		}
		pxAppend(&b, t)
	}
	return b.String(), ok
}

// pxBinary are the commands of the relations and the operations, which get a space on each side.
var pxBinary = map[string]bool{`\times`: true, `\cdot`: true, `\le`: true, `\ge`: true, `\ne`: true, `\leqslant`: true, `\geqslant`: true, `\pm`: true, `\approx`: true, `\equiv`: true, `\in`: true, `\to`: true, `\perp`: true, `\parallel`: true, `\sim`: true, `\subset`: true, `\subseteq`: true, `\cup`: true, `\cap`: true, `\div`: true}

var pxNot = strings.NewReplacer("̸=", "≠", "̸ =", "≠", "̸∈", "∉", "̸ ∈", "∉")

// pxOp and pxSpaces tidy the spaces of a formula: one space on each side of a relation, and of a plus or a minus between two terms.
var (
	pxOp     = regexp.MustCompile(`\s*([=<>+-])\s*`)
	pxSpaces = regexp.MustCompile(`\s+`)
)

// PDFTidy gives a formula with tidy spaces, as a + b = c.
func PDFTidy(m string) string {
	m = pxOp.ReplaceAllString(m, "$1")
	m = strings.ReplaceAll(m, `\not=`, ` \ne `)
	m = strings.ReplaceAll(m, `\not\in`, `\notin`)
	var b strings.Builder
	rs := []rune(m)
	for i, r := range rs {
		if !strings.ContainsRune("=<>+-", r) || i == 0 || i == len(rs)-1 {
			b.WriteRune(r)
			continue
		}
		prev := rs[i-1]
		binary := strings.ContainsRune("=<>", r) || unicode.IsLetter(prev) || unicode.IsDigit(prev) || strings.ContainsRune(")]}|'!", prev)
		if !binary || prev == '\\' || prev == '^' || prev == '_' || prev == '{' || strings.ContainsRune("=<>+-", prev) || strings.ContainsRune("=<>+-", rs[i+1]) {
			b.WriteRune(r)
			continue
		}
		b.WriteString(" " + string(r) + " ")
	}
	return strings.TrimSpace(pxSpaces.ReplaceAllString(b.String(), " "))
}

var pxCommandEnd = regexp.MustCompile(`\\[A-Za-z]+$`)

// pxAppend adds t to the TeX in b, with a space after a command when t starts with a letter, as \alpha x.
func pxAppend(b *strings.Builder, t string) {
	if t == "" {
		return
	}
	r, _ := utf8.DecodeRuneInString(t)
	if unicode.IsLetter(r) && pxCommandEnd.MatchString(b.String()) {
		b.WriteString(" ")
	}
	b.WriteString(t)
}

// pxScriptTeX gives ^ or _ with the TeX of a group of scripts.
func pxScriptTeX(sup bool, tex string) string {
	if sup && strings.Trim(tex, "'") == "" {
		return tex
	}
	op := "_"
	if sup {
		op = "^"
	}
	if utf8.RuneCountInString(tex) == 1 || (pxCommandEnd.MatchString(tex) && strings.LastIndex(tex, `\`) == 0) {
		return op + tex
	}
	return op + "{" + tex + "}"
}

// pxPiece is a part of a line: text, or the TeX of math.
type pxPiece struct {
	math bool
	s    string
}

// pieces gives the pieces of the line from the item from on. ok is false when the line has math that its text cannot hold.
func (l *pxLine) pieces(from int) (ps []pxPiece, ok bool) {
	ok = true
	for i := from; i < len(l.items); i++ {
		it := l.items[i]
		space := false
		if i > from {
			prev := l.items[i-1].Right()
			for _, sc := range l.scripts[i-1] {
				prev = max(prev, sc.Right())
			}
			space = it.Left-prev > max(2, it.Size/6)
		}
		var p pxPiece
		if it.Math() {
			tex, good := pxMath(it.Text, strings.Contains(it.Family, "CMR"))
			ok = ok && good
			var b strings.Builder
			b.WriteString(tex)
			for _, g := range pxScriptGroups(l.scripts[i]) {
				st, good := pxMath(g.text, false)
				ok = ok && good
				pxAppend(&b, pxScriptTeX(g.sup, st))
			}
			p = pxPiece{true, b.String()}
		} else {
			if len(l.scripts[i]) > 0 {
				ok = false
			}
			p = pxPiece{false, it.Text}
		}
		n := len(ps)
		switch {
		case n > 0 && ps[n-1].math && p.math:
			if space {
				ps[n-1].s += " " + p.s
			} else {
				var b strings.Builder
				b.WriteString(ps[n-1].s)
				pxAppend(&b, p.s)
				ps[n-1].s = b.String()
			}
			continue
		case space && n > 0 && !ps[n-1].math:
			ps[n-1].s += " "
		case space:
			ps = append(ps, pxPiece{false, " "})
		}
		if n := len(ps); n > 0 && !ps[n-1].math && !p.math {
			ps[n-1].s += p.s
			continue
		}
		ps = append(ps, p)
	}
	return ps, ok
}

// pxScriptGroup is a run of scripts of the same kind next to each other.
type pxScriptGroup struct {
	sup  bool
	text string
}

func pxScriptGroups(ss []pxScript) []pxScriptGroup {
	ss = slices.Clone(ss)
	sort.SliceStable(ss, func(i, j int) bool { return ss[i].Left < ss[j].Left })
	var out []pxScriptGroup
	// A subscript and a superscript at the same place, as x_1^2, give the subscript first.
	sort.SliceStable(ss, func(i, j int) bool {
		if pxAbs(float64(ss[i].Left-ss[j].Left)) <= 2 {
			return !ss[i].sup && ss[j].sup
		}
		return false
	})
	for _, s := range ss {
		if n := len(out); n > 0 && out[n-1].sup == s.sup {
			out[n-1].text += s.Text
			continue
		}
		out = append(out, pxScriptGroup{s.sup, s.Text})
	}
	return out
}
