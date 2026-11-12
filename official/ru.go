package official

import (
	"html"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/encoding/htmlindex"
)

// RuProblem is one problem of an official Russian paper: its position in the edition and its text in Markdown with TeX.
type RuProblem struct {
	// Edition is the edition part of the locator, as 2025 or 2024-fall. Position is the position, as p81 or d113.
	Edition, Position string
	Text              string
	// URL is the page or the PDF that gives the text, and Page is the page of a PDF, from 1, or 0 for a web page.
	URL  string
	Page int
	// Tool is the tool that read the text layer of a PDF, as pdftohtml. It is empty for pdftotext.
	Tool string
	// Figure is true when the problem has a picture that the text does not hold.
	Figure bool
	// Flags are the warnings of the native lane for the page of a PDF, as layout.
	Flags []string
	// Model is the model that read the text from the image of the page, when the text layer was not good enough, or "" for the text of this package.
	Model string
}

// At gives the locator of the problem in the competition comp.
func (p RuProblem) At(comp string) string { return comp + "/" + p.Edition + "/" + p.Position }

var (
	ruComment = regexp.MustCompile(`(?s)<!--.*?-->`)
	ruScript  = regexp.MustCompile(`(?is)<(script|style)\b.*?</(?:script|style)>`)
	ruScripts = regexp.MustCompile(`(?is)<(sup|sub)\b[^>]*>((?:[^<]|<(?:/?[ib]|/?em|/?font[^>]*)>)*?)</(?:sup|sub)>`)
	ruAngle   = regexp.MustCompile(`(?is)<u>\s*/\s*</u>`)
	ruPar     = regexp.MustCompile(`(?is)<u>(?:\s|&nbsp;?)*\|(?:\s|&nbsp;?)*</u>`)
	ruSymFont = regexp.MustCompile(`(?is)<font\s+face="?symbol"?\s*>(.*?)</font>`)
	ruLayout  = regexp.MustCompile(`(?i)<table\b`)
	ruBreak   = regexp.MustCompile(`(?i)<br\s*/?>|</?(?:tr|table|div|h[1-6])\b[^>]*>`)
	ruPara    = regexp.MustCompile(`(?i)</?p\b[^>]*>`)
	ruItem    = regexp.MustCompile(`(?i)<li\b[^>]*>`)
	ruImage   = regexp.MustCompile(`(?i)<img\b[^>]*>`)
	// ruTag is a tag of HTML. Another < is a sign of the text, as in c<0 or a<b, which some old pages do not escape.
	ruTag    = regexp.MustCompile(`(?i)</?(?:a|b|i|u|p|br|hr|font|sup|sub|span|div|table|tbody|thead|tr|td|th|li|ul|ol|dl|dt|dd|em|strong|center|img|h[1-6]|html|head|body|title|meta|link|blockquote|pre|tt|small|big|nobr|code|s|strike|label|caption|col|colgroup|section|article|nav|header|footer|main|abbr|cite|q|address|wbr|form|input|iframe|object|param|noscript|base|basefont|o:p|math|mi|mo|mn|mrow|svg|path|g|use|defs|mjx-[a-z-]+)(?:\s[^<>]*)?/?>|<![^<>]*>`)
	ruSpaces = regexp.MustCompile(`[ \t]+`)
	ruBlank  = regexp.MustCompile(`\n\s*\n\s*`)
	ruLine   = regexp.MustCompile(` *\n *`)
	// ruAuthor is the name of the author in parentheses at the end of a problem, as (А. Шаповалов), (И. Ященко, А. Шаповалов) or (Фольклор).
	ruAuthor     = regexp.MustCompile(`(?s)\s*\((?:(?:\p{Lu}\.\s?){1,2}\s?\p{Lu}[\p{L}-]+(?:,\s*(?:\p{Lu}\.\s?){1,2}\s?\p{Lu}[\p{L}-]+)*|[Фф]ольклор)\)\s*$`)
	ruAuthorSpan = regexp.MustCompile(`(?is)<span class="author">.*?</span>`)
	// ruLink is a link to the solution, the answer or a hint of a problem.
	ruLink = regexp.MustCompile(`(?is)<a\b[^>]*>\s*(?:Решение|Решения|Ответ|Указание|Подсказка|Комментарий)\s*</a>`)
)

// RuHTML turns the HTML of one Russian problem into Markdown with TeX math. A superscript or a subscript joins the word before it, as in a<sup>2</sup>, which gives $a^2$. The math of a page that has it already, between dollars, stays as it is. Each run of Latin letters, digits and operators becomes inline TeX (see RuTeX). It tells if the HTML has a picture. The name of the author at the end is not part of the text.
func RuHTML(s string) (text string, figure bool) {
	s = ruComment.ReplaceAllString(s, "")
	s = ruScript.ReplaceAllString(s, "")
	s = ruAuthorSpan.ReplaceAllString(s, "")
	s = ruLink.ReplaceAllString(s, "")
	// A brace of the text is math, as in {x_n}, and becomes \{ in TeX. It is marked here, before the braces of the scripts are made.
	s = strings.NewReplacer("\r", "", "\n", " ", "{", "\x01", "}", "\x02").Replace(s)
	figure = ruImage.MatchString(s)
	s = ruImage.ReplaceAllString(s, " ")
	s = ruAngle.ReplaceAllString(s, "∠")
	s = ruPar.ReplaceAllString(s, " ∥ ")
	s = ruSymFont.ReplaceAllStringFunc(s, func(m string) string {
		g := ruSymFont.FindStringSubmatch(m)
		var b strings.Builder
		for _, r := range html.UnescapeString(g[1]) {
			t, ok := symbolFont[r]
			if !ok {
				// A run of dashes is the bar of a fraction drawn in lines, which the text cannot hold.
				return "\x00"
			}
			b.WriteRune(t)
		}
		return b.String()
	})
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
	s = ruItem.ReplaceAllString(s, "\n- ")
	s = ruPara.ReplaceAllString(s, "\n\n")
	s = ruBreak.ReplaceAllString(s, "\n")
	s = ruTag.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = RuPlain(s)
	s = ruNote.ReplaceAllString(s, "")
	s = ruAuthor.ReplaceAllString(s, "")
	return strings.NewReplacer("\x01", "{", "\x02", "}").Replace(RuTeX(s)), figure
}

// RuLayout tells if the HTML of a problem has math that the text cannot hold: a table that draws a formula, or a fraction bar of the Symbol font.
func RuLayout(s string) bool {
	if ruLayout.MatchString(s) {
		return true
	}
	t, _ := RuHTML(s)
	return strings.Contains(t, "\x00")
}

// symbolFont gives the letter of a character in the Symbol font, which old pages use for Greek letters and math signs. Δ before the name of a triangle is the sign of a triangle.
var symbolFont = map[rune]rune{
	'a': 'α', 'b': 'β', 'g': 'γ', 'd': 'δ', 'e': 'ε', 'j': 'φ', 'f': 'φ', 'l': 'λ', 'm': 'μ', 'p': 'π', 'w': 'ω', 'D': '△',
	'®': '→', '^': '⊥', '¹': '≠', '£': '≤', '³': '≥', '°': '°', 'Ð': '∠', '×': '×', 'Î': '∈', '¥': '∞', '·': '·', 'Ö': '√', '|': '∥', ' ': ' ',
}

// RuPlain makes the spaces of a text plain: odd spaces become spaces, each line is trimmed, and a run of blank lines becomes one.
func RuPlain(s string) string {
	s = strings.NewReplacer("\u00a0", " ", "\u2009", " ", "\u202f", " ", "\u200b", "", "\u00ad", "", "\t", " ").Replace(s)
	s = ruSpaces.ReplaceAllString(s, " ")
	s = ruLine.ReplaceAllString(s, "\n")
	s = ruBlank.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// ruSymbols are the TeX forms of the math symbols of a Russian page.
var ruSymbols = map[rune]string{
	'≤': `\le`, '⩽': `\le`, '≥': `\ge`, '⩾': `\ge`, '≠': `\ne`, '≈': `\approx`, '≡': `\equiv`, '×': `\times`, '·': `\cdot`, '⋅': `\cdot`,
	'±': `\pm`, '−': `-`, '→': `\to`, '∞': `\infty`, '∈': `\in`, '∠': `\angle`, '⊥': `\perp`, '∥': `\parallel`, '°': `^\circ`,
	'√': `\sqrt`, 'π': `\pi`, 'α': `\alpha`, 'β': `\beta`, 'γ': `\gamma`, 'δ': `\delta`, 'ε': `\varepsilon`, 'φ': `\varphi`,
	'λ': `\lambda`, 'μ': `\mu`, 'ω': `\omega`, 'Δ': `\Delta`, '△': `\triangle`, '…': `\ldots`, '⌊': `\lfloor`, '⌋': `\rfloor`,
	'⌈': `\lceil`, '⌉': `\rceil`, '\x01': `\{`, '\x02': `\}`, '∪': `\cup`, '∩': `\cap`, '⊂': `\subset`, '∑': `\sum`,
}

var (
	ruOps = map[string]string{"=": "=", "<": "<", ">": ">", "+": "+", "-": "-", "−": "-", "*": `\cdot`, "/": "/", "<=": `\le`, ">=": `\ge`, "≤": `\le`, "≥": `\ge`, "≠": `\ne`, "·": `\cdot`, "×": `\times`, ":": ":", "||": `\parallel`}
	// ruNumeric is a number or a sum of numbers, as 2001, 0,5 or 2+3=5. A hyphen between numbers is a range, as 5-11, and not a minus.
	ruNumber  = regexp.MustCompile(`^[0-9]+(?:[.,][0-9]+)?$`)
	ruNumeric = regexp.MustCompile(`^\(?[0-9]+\)?(?:[=<>+*/^:×·][-]?\(?[0-9]+\)?)+$`)
	ruTimes   = regexp.MustCompile(`([0-9])\s*[*xх×]\s*([0-9])`)
	// ruFunc is a run of Latin letters, maybe after a backslash. A run that is the name of a function, as cos in "cos x" or "2cos x", gets a backslash.
	ruFunc = regexp.MustCompile(`\\?[A-Za-z]+`)
	// ruNote is a last paragraph that is a rule of the paper and not a part of the problem, as "Из трех последних задач 11 класса в зачет идут две."
	ruDollar = regexp.MustCompile(`\$\$?`)
	ruNote   = regexp.MustCompile(`(?s)\n\n[^\n]*в\s+зач[её]т[^\n]*$`)
)

// ruToken is one word of a line, with the punctuation around it apart.
type ruToken struct {
	lead, core, trail string
	strong, weak, op  bool
}

func newRuToken(f string) ruToken {
	t := ruToken{core: f}
	for t.core != "" {
		r, n := utf8.DecodeLastRuneInString(t.core)
		if strings.ContainsRune(".,;:!?»\"", r) || (r == ')' && strings.Count(t.core, "(") < strings.Count(t.core, ")")) || (r == ']' && strings.Count(t.core, "[") < strings.Count(t.core, "]")) {
			t.trail = t.core[len(t.core)-n:] + t.trail
			t.core = t.core[:len(t.core)-n]
			continue
		}
		break
	}
	for t.core != "" {
		r, n := utf8.DecodeRuneInString(t.core)
		if strings.ContainsRune("«\"", r) || (r == '(' && strings.Count(t.core, "(") > strings.Count(t.core, ")")) || (r == '[' && strings.Count(t.core, "[") > strings.Count(t.core, "]")) {
			t.lead += t.core[:n]
			t.core = t.core[n:]
			continue
		}
		break
	}
	if t.core == "" {
		return t
	}
	if _, ok := ruOps[t.core]; ok && t.core != ":" {
		t.op = true
		return t
	}
	if lat, ok := ruPoints(t.core); ok {
		t.core, t.strong = lat, true
		return t
	}
	cyr, latin, sym := false, false, false
	for _, r := range t.core {
		switch {
		case unicode.Is(unicode.Cyrillic, r):
			cyr = true
		case r < 128 && unicode.IsLetter(r):
			latin = true
		case r == '^' || r == '_' || r == '\\' || ruSymbols[r] != "" || unicode.Is(unicode.Greek, r):
			sym = true
		}
	}
	switch {
	case cyr:
	case latin || sym || ruNumeric.MatchString(t.core):
		t.strong = true
	case ruNumber.MatchString(t.core):
		t.weak = true
	}
	return t
}

// ruLike are the Cyrillic capitals that look like Latin capitals. Some pages write the names of points with them, as АВС for ABC.
var ruLike = map[rune]rune{'А': 'A', 'В': 'B', 'С': 'C', 'Е': 'E', 'Н': 'H', 'К': 'K', 'М': 'M', 'О': 'O', 'Р': 'P', 'Т': 'T', 'Х': 'X'}

// ruPoints reads a word of two to five capitals, Latin or Cyrillic capitals that look like Latin ones, with maybe digits, primes and an angle sign, as АВС or ∠ЕМK. Such a word is the name of points, which is math, so it gives the word with Latin letters. A word of Latin capitals alone is not changed here.
func ruPoints(s string) (string, bool) {
	if s == "СССР" {
		return "", false
	}
	var b strings.Builder
	letters, cyr := 0, false
	for _, r := range s {
		switch l, ok := ruLike[r]; {
		case ok:
			b.WriteRune(l)
			letters++
			cyr = true
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
			letters++
		case unicode.IsDigit(r) || r == '\'' || r == '∠' || r == '△':
			b.WriteRune(r)
		default:
			return "", false
		}
	}
	return b.String(), cyr && letters >= 2 && letters <= 5
}

// RuTeX puts the math of a Russian text into TeX. In a Russian text a Latin letter is always math, so each run of words with a Latin letter, a symbol, a superscript or an operator between numbers becomes inline TeX, with the numbers and the operators next to it, as in "AM^2+CN^2=MN^2" or "x + y = 5". A word with a Cyrillic letter, punctuation at the end of a word and a number alone stay text. The text between dollars stays as it is.
func RuTeX(s string) string {
	// The text is cut at each $ or $$. The parts of odd index are math, each one closed by the sign that opened it.
	var parts, signs []string
	from, open := 0, ""
	for _, m := range ruDollar.FindAllStringIndex(s, -1) {
		sign := s[m[0]:m[1]]
		if open != "" && sign != open {
			continue
		}
		parts = append(parts, s[from:m[0]])
		signs = append(signs, sign)
		from = m[1]
		if open == "" {
			open = sign
		} else {
			open = ""
		}
	}
	if open != "" {
		// A $ with no pair: the text is not clear, so its math is not touched.
		return s
	}
	parts = append(parts, s[from:])
	for i := 1; i < len(parts); i += 2 {
		parts[i] = ruMathLatin(parts[i])
	}
	for i := 0; i < len(parts); i += 2 {
		lines := strings.Split(parts[i], "\n")
		for j, l := range lines {
			lines[j] = ruLineTeX(l)
		}
		parts[i] = strings.Join(lines, "\n")
	}
	var b strings.Builder
	for i, p := range parts {
		b.WriteString(p)
		if i < len(signs) {
			b.WriteString(signs[i])
		}
	}
	return b.String()
}

// ruMathLatin makes the Cyrillic capitals in the math of a page into the Latin ones they look like, as Р in "\sqrt{Р + Т}". Math with another Cyrillic letter, as a word in \text, is not changed.
func ruMathLatin(s string) string {
	for _, r := range s {
		if _, ok := ruLike[r]; !ok && unicode.Is(unicode.Cyrillic, r) {
			return s
		}
	}
	return strings.Map(func(r rune) rune {
		if l, ok := ruLike[r]; ok {
			return l
		}
		return r
	}, s)
}

func ruLineTeX(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}
	fields := strings.Split(line, " ")
	toks := make([]ruToken, len(fields))
	for i, f := range fields {
		toks[i] = newRuToken(f)
		// One Cyrillic capital that looks like a Latin one, inside a sentence, is the name of a point, as in "в точке Е".
		if l, ok := ruLike[[]rune(toks[i].core + " ")[0]]; ok && utf8.RuneCountInString(toks[i].core) == 1 && i > 0 && !strings.ContainsAny(fields[i-1][max(len(fields[i-1])-1, 0):], ".!?:") && fields[i-1] != "" && fields[i-1] != "-" && fields[i-1] != "—" {
			toks[i].core, toks[i].strong = string(l), true
		}
	}
	math := func(t ruToken) bool { return t.strong || t.weak || t.op }
	var out []string
	for i := 0; i < len(toks); {
		if !math(toks[i]) {
			out = append(out, fields[i])
			i++
			continue
		}
		j := i
		for j < len(toks) && math(toks[j]) {
			j++
			if toks[j-1].trail != "" || (j < len(toks) && toks[j].lead != "") {
				break
			}
		}
		run := toks[i:j]
		first, last := -1, -1
		for k, t := range run {
			if t.strong {
				if first < 0 {
					first = k
				}
				last = k
			}
		}
		lo, hi := 0, 0
		if first >= 0 {
			lo, hi = first, last+1
			for lo > 0 && (run[lo].op || run[lo-1].op || ruOpEdge(run[lo].core, true)) && math(run[lo-1]) {
				lo--
			}
			for hi < len(run) && (run[hi-1].op || run[hi].op || ruOpEdge(run[hi-1].core, false)) {
				hi++
			}
			// An operator at an end of the run is a dash or a sign of the prose.
			for lo < hi && run[lo].op {
				lo++
			}
			for hi > lo && run[hi-1].op {
				hi--
			}
		}
		for k := i; k < i+lo; k++ {
			out = append(out, fields[k])
		}
		if lo < hi {
			var b strings.Builder
			for k, t := range run[lo:hi] {
				if k > 0 {
					b.WriteString(" ")
				}
				if k > 0 && t.lead != "" {
					b.WriteString(t.lead)
				}
				b.WriteString(ruCore(t.core))
				if k < hi-lo-1 {
					b.WriteString(t.trail)
				}
			}
			out = append(out, run[lo].lead+"$"+b.String()+"$"+run[hi-1].trail)
		}
		for k := i + hi; k < j; k++ {
			out = append(out, fields[k])
		}
		i = j
	}
	return strings.Join(out, " ")
}

// ruOpEdge tells if a word of math starts (first) or ends with an operator, as c< in "c< 0", so that the number next to it belongs to the formula.
func ruOpEdge(s string, first bool) bool {
	var r rune
	if first {
		r, _ = utf8.DecodeRuneInString(s)
	} else {
		r, _ = utf8.DecodeLastRuneInString(s)
	}
	return strings.ContainsRune("=<>+−≤≥≠·×*/", r)
}

// ruFuncTeX puts a backslash before the name of a function, as \cos. The Russian names tg and ctg have no command in TeX, so they become operators.
func ruFuncTeX(s string) string {
	return ruFunc.ReplaceAllStringFunc(s, func(w string) string {
		switch w {
		case "tg", "ctg", "arctg":
			return `\operatorname{` + w + `}`
		case "arcsin", "arccos", "arctan", "sin", "cos", "tan", "cot", "log", "lg", "ln", "exp", "max", "min":
			return `\` + w
		}
		return w
	})
}

// ruCore gives the TeX of the core of one word of math.
func ruCore(s string) string {
	if op, ok := ruOps[s]; ok {
		return op
	}
	s = ruTimes.ReplaceAllString(s, `$1 \times $2`)
	s = ruFuncTeX(s)
	var b strings.Builder
	command := false
	for _, r := range s {
		t, ok := ruSymbols[r]
		switch {
		case r == '*':
			t, ok = `\cdot `, true
		case !ok:
			t = string(r)
		}
		if command && unicode.IsLetter(r) {
			b.WriteString(" ")
		}
		b.WriteString(t)
		command = ok && strings.HasPrefix(t, `\`) && unicode.IsLetter(rune(t[len(t)-1]))
	}
	return strings.TrimSpace(b.String())
}

// ruYo makes ё into е, so that two texts that differ only there have the same words.
var ruYo = strings.NewReplacer("ё", "е", "Ё", "Е")

// RuSignal gives the word signal of an official text a and a key b, and the part of the words of the key that the official text has. A cut official text keeps the second number low, and an official text that adds a part, as a second question that the key leaves out, keeps it high. The words are runs of three or more letters, in lower case, with ё as е.
func RuSignal(a, b string) (jaccard, containment float64) {
	wa, wb := Words(ruYo.Replace(a)), Words(ruYo.Replace(b))
	if len(wa) == 0 || len(wb) == 0 {
		return 0, 0
	}
	n := 0
	for w := range wa {
		if wb[w] {
			n++
		}
	}
	return float64(n) / float64(len(wa)+len(wb)-n), float64(n) / float64(len(wb))
}

// RuAgree is the lowest word signal of an official text and the key of its position at which the two are one problem. RuContain and RuAgreeLow allow an official text that adds a part, as a second question that the key leaves out: the official text must have RuContain of the words of the key, and the signal must be at least RuAgreeLow.
const (
	RuAgree    = 0.5
	RuContain  = 0.8
	RuAgreeLow = 0.35
	// RuMargin is the smallest difference between the signal of the official text at the position and the best signal of another official text of the edition with the key.
	RuMargin = 0.1
)

// RuAgrees tells if the signal of two texts is enough for one problem.
func RuAgrees(j, c float64) bool { return j >= RuAgree || (c >= RuContain && j >= RuAgreeLow) }

// RuNear is the word signal at or above which two official texts of an edition are one problem, such as a problem that two grades share.
const RuNear = 0.9

// RuCheck compares the key of a position with the official texts of its edition.
type RuCheck struct {
	// J and C are the signal of the official text at the position with the key.
	J, C float64
	// Rival is the position of the other official text of the edition with the best signal with the key, and RivalJ is that signal. A text that is one problem with the text at the position does not count.
	Rival  string
	RivalJ float64
}

// OK tells if the official text at the position is the problem of the key: the two agree, and no other text of the edition comes within RuMargin.
func (k RuCheck) OK() bool { return RuAgrees(k.J, k.C) && k.J-k.RivalJ >= RuMargin }

// RuConfirm compares key with the official text at of an edition and with the other texts all of the edition.
func RuConfirm(at RuProblem, all []RuProblem, key string) RuCheck {
	var k RuCheck
	k.J, k.C = RuSignal(at.Text, key)
	for _, q := range all {
		if q.Position == at.Position {
			continue
		}
		if j, _ := RuSignal(q.Text, at.Text); j >= RuNear {
			continue
		}
		if j, _ := RuSignal(q.Text, key); j > k.RivalJ {
			k.Rival, k.RivalJ = q.Position, j
		}
	}
	return k
}

// ruCharsetRE is the charset of the meta tag of a page.
var ruCharsetRE = regexp.MustCompile(`(?i)<meta\b[^>]*charset\s*=\s*["']?([\w-]+)`)

// RuDecode gives a page in UTF-8. The charset comes from the meta tag of the page, as koi8-r or windows-1251 on the old pages of MCCME. A page with no charset stays as it is.
func RuDecode(b []byte) (string, error) {
	m := ruCharsetRE.FindSubmatch(b)
	if m == nil {
		return string(b), nil
	}
	enc, err := htmlindex.Get(string(m[1]))
	if err != nil {
		return "", err
	}
	if name, _ := htmlindex.Name(enc); name == "utf-8" {
		return string(b), nil
	}
	out, err := enc.NewDecoder().Bytes(b)
	return string(out), err
}

// RuVersion names the rules of RuHTML and RuTeX. It goes into the extraction field of a text that they read.
const RuVersion = "om-rutex/1"

// RuExtraction gives the extraction field of an official text: native/html for a web page and native/pdftotext, or the tool in Tool, for a page of a PDF, with the rules of this package in brackets, or the model that read the text from the image of the page.
func RuExtraction(p RuProblem) string {
	rules := RuVersion
	if p.Model != "" {
		rules = p.Model
	}
	if p.Page > 0 {
		tool := p.Tool
		if tool == "" {
			tool = "pdftotext"
		}
		return "native/" + tool + " (" + rules + ")"
	}
	return "native/html (" + rules + ")"
}
