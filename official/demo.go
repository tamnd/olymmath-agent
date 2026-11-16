package official

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

// DemoVersion names the rules of DemoPDF in the extraction field of a statement.
const DemoVersion = "om-demo/1"

// DemoPaper is a paper PDF of the archive of the German olympiad on olympiade-mathematik.de: the olympiad, the grade and the round. The file 34104_a.pdf is the paper of the 34th olympiad, grade 10, round 4.
type DemoPaper struct {
	Olympiad, Grade, Round int
}

var demoFileRE = regexp.MustCompile(`^(\d{2})(\d{2})([1-4])_a\.pdf$`)

// DemoFile reads the name of a paper PDF, as 34104_a.pdf.
func DemoFile(name string) (DemoPaper, bool) {
	m := demoFileRE.FindStringSubmatch(name)
	if m == nil {
		return DemoPaper{}, false
	}
	o, _ := strconv.Atoi(m[1])
	g, _ := strconv.Atoi(m[2])
	r, _ := strconv.Atoi(m[3])
	if o == 0 || g < 1 || g > 13 {
		return DemoPaper{}, false
	}
	return DemoPaper{o, g, r}, true
}

// DemoCode reads the number of a problem, as 341042: the olympiad, the grade, the round and the problem. n is the number of the problem in the paper, with a small letter for one of two problems to choose from, as 6a for 101246A.
func DemoCode(code string) (p DemoPaper, n string, ok bool) {
	if len(code) != 6 && (len(code) != 7 || (code[6] != 'A' && code[6] != 'B')) {
		return DemoPaper{}, "", false
	}
	p, ok = DemoFile(code[:5] + "_a.pdf")
	n = strings.ToLower(code[5:])
	return p, n, ok && code[5] >= '1' && code[5] <= '9'
}

// DemoProblem is one problem of a paper.
type DemoProblem struct {
	// Code is the number of the problem in the paper, as 341042.
	Code string
	// Same are the numbers of the same problem in the papers of other grades, from the heading "Aufgabe 341042 = 340943:".
	Same []string
	// Page is the page of the PDF where the problem starts, from 1.
	Page int
	// Text is the Markdown text with TeX math. It is "" when the problem has no line of text.
	Text string
	// Figure is true when the pages of the problem have ink outside the runs of text: a drawing, the rule of a fraction or of a table, or a picture.
	Figure bool
	// Flags are the warnings of the reading, as layout: the text may not hold all of the problem.
	Flags []string
}

// DemoInk tells if a page has ink outside the runs of text in the rows from top to bottom, in the pixels of pdftohtml. A nil DemoInk finds no ink.
type DemoInk func(page, top, bottom int) bool

var (
	// demoHeadRE is the heading of a problem: "Aufgabe 011241:", "Aufgabe 341042 = 340943:" or "Aufgabe 101246A:", one of two problems to choose from.
	demoHeadRE = regexp.MustCompile(`^Aufgabe (\d{6}[AB]?)((?:\s*=\s*\d{6}[AB]?)*)\s*:$`)
	demoCodeRE = regexp.MustCompile(`\d{6}[AB]?`)
	// demoSiteRE and demoFootRE are the header and the footer of a page: the address of the site, and "34. Olympiade - Bundesrunde - Klasse 10 - Saison 1994/1995" with the page number.
	demoSiteRE = regexp.MustCompile(`^https?://www\.olympiade-mathematik\.de/?$`)
	demoFootRE = regexp.MustCompile(`^\d+\. Olympiade - .* Saison \d{4}/\d{4}(?: \d+)?$`)
	// demoMathFont and demoTextFont are the fonts of math and of text of the papers, which are made with LaTeX and the Latin Modern fonts.
	demoMathFont = regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:LMMathItalic|LMMathSymbols|CMMI|CMSY)\d`)
	// demoSymbolFont is the math symbols font, some of whose glyphs pdftohtml gives as the letters and the digits at their places in the font.
	demoSymbolFont = regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:LMMathSymbols|CMSY)\d`)
	// demoOMS gives the symbols of those glyphs: the prime, the parallel sign, the triangle and the angle brackets. The 6, the slash of ≠, stays for demoRender.
	demoOMS      = strings.NewReplacer("0", "′", "k", "∥", "4", "△", "h", "⟨", "i", "⟩")
	demoTextFont = regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:LMRoman|LMSans|CMR|CMBX|CMTI)\d`)
	// demoNumber is a word of a text font that can be a part of a formula, as "(1", "= 7" or "2)": digits, brackets and the signs of the relations and the operations.
	demoNumber = regexp.MustCompile(`^[0-9()\[\]=+<>/|−\-*'·.,;:!?]+$`)
	demoPunct  = regexp.MustCompile(`^[.,;:!?]+$`)
	// demoFunc is the name of a function in a formula, which the papers set upright.
	demoFunc = regexp.MustCompile(`^(sin|cos|tan|cot|lg|log|ln|exp)([.,;:!?]*)$`)
	// demoPlain is a formula that is only a number, as 0,6 with the comma of the math font: the text keeps it as text.
	demoPlain = regexp.MustCompile(`^\d+(?:[,.]\d+)*$`)
	// demoPart starts a part of a problem: "a)", "(1)" or "1.".
	demoPart    = regexp.MustCompile(`^(?:\(?[a-z]\)|\(\d+\)|\d+\.)\s`)
	demoEndPunc = regexp.MustCompile(`[.?!:]$`)
)

// DemoPDF reads the problems of a paper PDF of the archive of the German olympiad from the runs of text that pdftohtml -xml gives. Each problem starts at its heading, as "Aufgabe 341042:", and ends at the next heading or at the end of the paper; the cover page and the note before the first problem are left out. ink finds the drawings of a page.
func DemoPDF(items []PDFItem, ink DemoInk) []DemoProblem {
	items = slices.Clone(items)
	for i, it := range items {
		if demoSymbolFont.MatchString(it.Family) {
			items[i].Text = demoOMS.Replace(it.Text)
		}
	}
	main := pxMainSize(items)
	lines, orphans := pxLines(items, main)
	// The header and the footer of each page bound the text of the page.
	top, bottom := map[int]int{}, map[int]int{}
	var body []*pxLine
	for _, l := range lines {
		t := l.text()
		switch {
		case demoSiteRE.MatchString(t):
			top[l.page()] = max(top[l.page()], l.top()+l.items[0].Height)
		case demoFootRE.MatchString(t):
			if b, ok := bottom[l.page()]; !ok || l.top() < b {
				bottom[l.page()] = l.top()
			}
		default:
			body = append(body, l)
		}
	}
	type start struct {
		line, from int
		code       string
		same       []string
	}
	var starts []start
	for i, l := range body {
		var b strings.Builder
		for k, it := range l.items {
			if k > 0 && it.Left-l.items[k-1].Right() > 2 {
				b.WriteString(" ")
			}
			b.WriteString(it.Text)
			m := demoHeadRE.FindStringSubmatch(RuPlain(b.String()))
			if m == nil {
				continue
			}
			starts = append(starts, start{line: i, from: k + 1, code: m[1], same: demoCodeRE.FindAllString(m[2], -1)})
			break
		}
	}
	var out []DemoProblem
	for i, st := range starts {
		end := len(body)
		if i+1 < len(starts) {
			end = starts[i+1].line
		}
		ls := body[st.line:end]
		p := demoProblem(ls, st.from)
		p.Code, p.Same, p.Page = st.code, st.same, ls[0].page()
		// The rows of the problem on each of its pages: from its heading or the header to the next heading or the footer.
		for pg := ls[0].page(); pg <= ls[len(ls)-1].page(); pg++ {
			from, to := top[pg], bottom[pg]
			if to == 0 {
				to = 1 << 20
			}
			if pg == ls[0].page() {
				from = ls[0].top() + ls[0].items[0].Height + 2
			}
			if end < len(body) && body[end].page() == pg {
				to = body[end].top() - 2
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

// demoAtom is a word of a line of text, or the TeX of a run of a math font, with the TeX of its scripts.
type demoAtom struct {
	s      string
	kind   int
	space  bool
	script string
}

const (
	demoText = iota
	demoMath
	demoNum
	demoPunc
)

// demoLine is one line of a problem: its Markdown and the facts that join it to the next line.
type demoLine struct {
	text          string
	left, right   int
	display, part bool
}

// demoProblem reads the text of a problem from its lines; the first line starts at the item from.
func demoProblem(lines []*pxLine, from int) DemoProblem {
	var p DemoProblem
	flag := func(f string) {
		if !slices.Contains(p.Flags, f) {
			p.Flags = append(p.Flags, f)
		}
	}
	var parts []demoLine
	textLeft := 1 << 20
	colRight := 0
	for li, l := range lines {
		start := 0
		if li == 0 {
			start = from
		}
		if start >= len(l.items) {
			continue
		}
		textLeft = min(textLeft, l.items[start].Left)
		colRight = max(colRight, l.right())
	}
	for li, l := range lines {
		start := 0
		if li == 0 {
			start = from
		}
		if start >= len(l.items) {
			continue
		}
		var atoms []demoAtom
		for i := start; i < len(l.items); i++ {
			it := l.items[i]
			space := false
			if i > start {
				prev := l.items[i-1].Right()
				for _, sc := range l.scripts[i-1] {
					prev = max(prev, sc.Right())
				}
				gap := it.Left - prev
				space = gap > 2
				// The comma of the math font in 4,5 has a thin space after it, which is no space of the text.
				if pi := l.items[i-1]; gap <= 3 && strings.TrimSpace(pi.Text) == "," && demoMathFont.MatchString(pi.Family) {
					space = false
				}
				// A wide gap is a column of a table or the side of a figure.
				if gap > 40 {
					flag("layout")
				}
			}
			switch {
			case demoMathFont.MatchString(it.Family):
				tex, ok := demoMathTeX(it.Text)
				if !ok {
					flag("glyph")
				}
				atoms = append(atoms, demoAtom{s: tex, kind: demoMath, space: space})
			case demoTextFont.MatchString(it.Family):
				lead := strings.HasPrefix(it.Text, " ")
				for k, w := range strings.Fields(it.Text) {
					a := demoAtom{s: w, space: (k == 0 && (space || lead)) || k > 0}
					switch {
					case demoPunct.MatchString(w):
						a.kind = demoPunc
					case demoNumber.MatchString(w):
						a.kind = demoNum
					case demoFunc.MatchString(w):
						a.kind = demoNum
						m := demoFunc.FindStringSubmatch(w)
						a.s = `\` + m[1] + m[2]
					}
					atoms = append(atoms, a)
				}
			default:
				// Another font, as the symbols of wasy or the large symbols, is a formula that a line cannot hold.
				flag("font")
				atoms = append(atoms, demoAtom{s: it.Text, kind: demoText, space: space})
			}
			if ss := l.scripts[i]; len(ss) > 0 && len(atoms) > 0 {
				last := &atoms[len(atoms)-1]
				if last.kind == demoText || last.kind == demoPunc {
					// A script of a word, as the 2 of P2O5, is chemistry or a table.
					flag("layout")
				}
				if r, _ := utf8.DecodeLastRuneInString(last.s); unicode.IsDigit(r) && demoSupSub(ss) {
					// Small digits over and under each other after a number are a fraction, as the 1/2 of 1½.
					flag("layout")
				}
				for _, g := range pxScriptGroups(ss) {
					st, ok := demoMathTeX(g.text)
					if !ok {
						flag("glyph")
					}
					last.script += pxScriptTeX(g.sup, st)
				}
			}
		}
		text, display := demoRender(atoms)
		if strings.TrimSpace(text) == "" {
			continue
		}
		left := l.items[start].Left
		if !display && left > textLeft+60 {
			// A line of text far right of the others is beside a figure.
			flag("layout")
		}
		parts = append(parts, demoLine{text: text, left: left, right: l.right(), display: display && left > textLeft+20, part: demoPart.MatchString(text + " ")})
	}
	var b strings.Builder
	for i, pt := range parts {
		text := pt.text
		if pt.display {
			end := strings.LastIndex(text, "$")
			text = "$$" + text[1:end] + text[end+1:] + "$$"
			if i > 0 {
				b.WriteString("\n\n")
			}
			b.WriteString(text)
			continue
		}
		if i == 0 {
			b.WriteString(text)
			continue
		}
		prev := parts[i-1]
		cur := b.String()
		switch {
		case prev.display || pt.part:
			b.WriteString("\n\n" + text)
		case demoHyphen(cur, text):
			b.Reset()
			b.WriteString(strings.TrimSuffix(cur, "-") + text)
		case strings.HasSuffix(cur, "-") && !strings.HasSuffix(cur, " -"):
			b.WriteString(text)
		case prev.right < colRight-60 && demoEndPunc.MatchString(cur) && pt.left <= textLeft+5:
			b.WriteString("\n\n" + text)
		default:
			b.WriteString(" " + text)
		}
	}
	p.Text = RuPlain(b.String())
	return p
}

// demoSupSub tells if the scripts of an item hold both a superscript and a subscript.
func demoSupSub(ss []pxScript) bool {
	sup, sub := false, false
	for _, s := range ss {
		sup = sup || s.sup
		sub = sub || !s.sup
	}
	return sup && sub
}

// demoHyphen tells if a line that ends in cur with a hyphen goes on in next with the rest of a word, so that the hyphen goes: the rest starts with a small letter, as in "Gesamt-" and "länge". A capital letter, as in "DDR-" and "Olympiade", keeps the hyphen.
func demoHyphen(cur, next string) bool {
	if !strings.HasSuffix(cur, "-") || strings.HasSuffix(cur, " -") {
		return false
	}
	r, _ := utf8.DecodeLastRuneInString(strings.TrimSuffix(cur, "-"))
	n, _ := utf8.DecodeRuneInString(next)
	return unicode.IsLetter(r) && unicode.IsLower(n)
}

// demoMathTeX gives the TeX of a run of a math font. The papers set ≠ as a slash, which pdftohtml gives as a 6 of the math symbols, over an =.
func demoMathTeX(s string) (string, bool) {
	s = strings.NewReplacer("...", "…", ". . .", "…").Replace(s)
	var b strings.Builder
	ok := true
	for k, part := range strings.Split(s, "…") {
		if k > 0 {
			pxAppend(&b, `\ldots`)
		}
		tex, good := pxMath(part, false)
		ok = ok && good
		pxAppend(&b, tex)
	}
	return b.String(), ok
}

// demoRender gives the Markdown of the atoms of a line: each run of math atoms with the words of numbers and signs next to them becomes one formula. display tells that the line is one formula only.
func demoRender(atoms []demoAtom) (text string, display bool) {
	// The 6 of the math symbols before an = or an ∈ is the slash of ≠ or ∉.
	for i := 0; i+1 < len(atoms); i++ {
		if atoms[i].kind == demoMath && atoms[i].s == "6" {
			atoms[i].s = `\not`
			atoms[i+1].space = false
		}
	}
	var b strings.Builder
	formulas, words, mathStart := 0, 0, false
	i := 0
	for i < len(atoms) {
		a := atoms[i]
		if a.kind == demoText {
			if a.space && b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString(a.s)
			words++
			i++
			continue
		}
		j := i
		seed := false
		for j < len(atoms) && atoms[j].kind != demoText {
			seed = seed || atoms[j].kind == demoMath || atoms[j].script != ""
			j++
		}
		run := atoms[i:j]
		if !seed {
			for _, r := range run {
				if r.space && b.Len() > 0 {
					b.WriteString(" ")
				}
				b.WriteString(strings.ReplaceAll(r.s, "−", "-"))
				words++
			}
			i = j
			continue
		}
		// Punctuation before the formula stays text.
		for len(run) > 0 && run[0].kind == demoPunc && run[0].script == "" {
			if run[0].space && b.Len() > 0 {
				b.WriteString(" ")
			}
			b.WriteString(run[0].s)
			run = run[1:]
		}
		var m strings.Builder
		for k, r := range run {
			s := r.s
			if r.kind != demoMath && !strings.HasPrefix(s, `\`) {
				tex, _ := pxMath(s, false)
				s = tex
			}
			if k > 0 && r.space {
				m.WriteString(" ")
				m.WriteString(s)
			} else {
				pxAppend(&m, s)
			}
			pxAppend(&m, r.script)
		}
		tex := strings.TrimSpace(m.String())
		tail := ""
		for tex != "" && strings.ContainsAny(tex[len(tex)-1:], ".,;:!?") && !strings.HasSuffix(tex, `\`+tex[len(tex)-1:]) {
			tail = tex[len(tex)-1:] + tail
			tex = strings.TrimSpace(tex[:len(tex)-1])
		}
		if len(run) > 0 && run[0].space && b.Len() > 0 {
			b.WriteString(" ")
		}
		tex = PDFTidy(tex)
		switch {
		case tex == "":
		case demoPlain.MatchString(strings.ReplaceAll(tex, " ", "")):
			b.WriteString(tex)
			words++
		default:
			if b.Len() == 0 {
				mathStart = true
			}
			b.WriteString("$" + tex + "$")
			formulas++
		}
		b.WriteString(tail)
		i = j
	}
	text = strings.TrimSpace(b.String())
	return text, formulas == 1 && words == 0 && mathStart
}
