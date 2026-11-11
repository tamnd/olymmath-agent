package ocr

import (
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// TeXVersion names the rules of TeX. A text that TeX changed records it in its extraction, as in native/pdftotext@26.10.0 (om-tex/1).
const TeXVersion = "om-tex/1"

// greek are the TeX commands of the Greek letters. A capital letter that looks like a Latin letter becomes that letter.
var greek = map[rune]string{
	'α': `\alpha`, 'β': `\beta`, 'γ': `\gamma`, 'δ': `\delta`, 'ε': `\varepsilon`, 'ϵ': `\epsilon`, 'ζ': `\zeta`, 'η': `\eta`,
	'θ': `\theta`, 'ϑ': `\vartheta`, 'ι': `\iota`, 'κ': `\kappa`, 'ϰ': `\varkappa`, 'λ': `\lambda`, 'μ': `\mu`, 'ν': `\nu`, 'ξ': `\xi`,
	'ο': `o`, 'π': `\pi`, 'ϖ': `\varpi`, 'ρ': `\rho`, 'ϱ': `\varrho`, 'σ': `\sigma`, 'ς': `\varsigma`, 'τ': `\tau`, 'υ': `\upsilon`,
	'φ': `\varphi`, 'ϕ': `\phi`, 'χ': `\chi`, 'ψ': `\psi`, 'ω': `\omega`,
	'Γ': `\Gamma`, 'Δ': `\Delta`, 'Θ': `\Theta`, 'ϴ': `\Theta`, 'Λ': `\Lambda`, 'Ξ': `\Xi`, 'Π': `\Pi`, 'Σ': `\Sigma`, 'Υ': `\Upsilon`,
	'Φ': `\Phi`, 'Ψ': `\Psi`, 'Ω': `\Omega`,
	'Α': `A`, 'Β': `B`, 'Ε': `E`, 'Ζ': `Z`, 'Η': `H`, 'Ι': `I`, 'Κ': `K`, 'Μ': `M`, 'Ν': `N`, 'Ο': `O`, 'Ρ': `P`, 'Τ': `T`, 'Χ': `X`,
}

// symbols are the TeX forms of the math symbols of a text layer.
var symbols = map[rune]string{
	'≤': `\le`, '⩽': `\le`, '≥': `\ge`, '⩾': `\ge`, '≠': `\ne`, '≈': `\approx`, '≡': `\equiv`, '≢': `\not\equiv`, '∼': `\sim`,
	'≅': `\cong`, '∝': `\propto`, '×': `\times`, '·': `\cdot`, '⋅': `\cdot`, '∙': `\cdot`, '÷': `\div`, '±': `\pm`, '∓': `\mp`,
	'−': `-`, '∗': `*`, '→': `\to`, '←': `\leftarrow`, '↔': `\leftrightarrow`, '⇒': `\Rightarrow`, '⇐': `\Leftarrow`,
	'⇔': `\Leftrightarrow`, '↦': `\mapsto`, '∞': `\infty`, '∈': `\in`, '∉': `\notin`, '∋': `\ni`, '⊂': `\subset`, '⊃': `\supset`,
	'⊆': `\subseteq`, '⊇': `\supseteq`, '∪': `\cup`, '∩': `\cap`, '∅': `\emptyset`, '∖': `\setminus`, '∠': `\angle`,
	'∡': `\measuredangle`, '△': `\triangle`, '⊥': `\perp`, '∥': `\parallel`, '‖': `\|`, '∣': `\mid`, '≺': `\prec`, '≻': `\succ`,
	'≪': `\ll`, '≫': `\gg`, '∀': `\forall`, '∃': `\exists`, '¬': `\neg`, '∧': `\wedge`, '∨': `\vee`, '∘': `\circ`, '∑': `\sum`,
	'∏': `\prod`, '∫': `\int`, '∂': `\partial`, '∇': `\nabla`, 'ℝ': `\mathbb{R}`, 'ℕ': `\mathbb{N}`, 'ℤ': `\mathbb{Z}`,
	'ℚ': `\mathbb{Q}`, 'ℂ': `\mathbb{C}`, '⌊': `\lfloor`, '⌋': `\rfloor`, '⌈': `\lceil`, '⌉': `\rceil`, '⟨': `\langle`, '⟩': `\rangle`,
	'°': `^\circ`, '′': `'`, '″': `''`, 'ℎ': `h`,
}

// weakSymbols become TeX inside a run of math, but they do not make a run of math alone, because prose has them too.
var weakSymbols = map[rune]string{'…': `\ldots`, '⋯': `\cdots`}

// supers and subs are the Unicode superscripts and subscripts.
var (
	supers = map[rune]rune{'⁰': '0', '¹': '1', '²': '2', '³': '3', '⁴': '4', '⁵': '5', '⁶': '6', '⁷': '7', '⁸': '8', '⁹': '9', '⁺': '+', '⁻': '-', '⁼': '=', '⁽': '(', '⁾': ')', 'ⁿ': 'n', 'ⁱ': 'i', 'ˣ': 'x', 'ʸ': 'y', 'ᵏ': 'k', 'ᵐ': 'm'}
	subs   = map[rune]rune{'₀': '0', '₁': '1', '₂': '2', '₃': '3', '₄': '4', '₅': '5', '₆': '6', '₇': '7', '₈': '8', '₉': '9', '₊': '+', '₋': '-', '₌': '=', '₍': '(', '₎': ')', 'ₐ': 'a', 'ₑ': 'e', 'ₒ': 'o', 'ₓ': 'x', 'ₕ': 'h', 'ₖ': 'k', 'ₗ': 'l', 'ₘ': 'm', 'ₙ': 'n', 'ₚ': 'p', 'ₛ': 's', 'ₜ': 't', 'ᵢ': 'i', 'ⱼ': 'j'}
)

// vulgar are the Unicode fractions.
var vulgar = map[rune]string{'½': `\frac{1}{2}`, '⅓': `\frac{1}{3}`, '⅔': `\frac{2}{3}`, '¼': `\frac{1}{4}`, '¾': `\frac{3}{4}`, '⅕': `\frac{1}{5}`, '⅖': `\frac{2}{5}`, '⅗': `\frac{3}{5}`, '⅘': `\frac{4}{5}`, '⅙': `\frac{1}{6}`, '⅚': `\frac{5}{6}`, '⅛': `\frac{1}{8}`, '⅜': `\frac{3}{8}`, '⅝': `\frac{5}{8}`, '⅞': `\frac{7}{8}`}

// mathLetter gives the plain letter or digit of a letter of the Unicode block of mathematical letters, such as 𝑥 for x, which word processors write for variables.
func mathLetter(r rune) (rune, bool) {
	switch {
	case r >= 0x1D400 && r <= 0x1D6A3:
		o := (r - 0x1D400) % 52
		if o < 26 {
			return 'A' + o, true
		}
		return 'a' + o - 26, true
	case r >= 0x1D6A8 && r <= 0x1D7C9:
		o := (r - 0x1D6A8) % 58
		switch {
		case o == 17:
			return 'Θ', true
		case o < 25:
			return 'Α' + o, true
		case o == 25:
			return '∇', true
		case o < 51:
			return 'α' + o - 26, true
		case o == 51:
			return '∂', true
		}
		if tail := []rune("ϵϑϰϕϱϖ"); int(o-52) < len(tail) {
			return tail[o-52], true
		}
		return 0, false
	case r >= 0x1D7CE && r <= 0x1D7FF:
		return '0' + (r-0x1D7CE)%10, true
	}
	return 0, false
}

// plainChars are the forms of a text layer that are not math: ligatures, soft hyphens and odd spaces.
var plainChars = strings.NewReplacer("ﬁ", "fi", "ﬂ", "fl", "ﬀ", "ff", "ﬃ", "ffi", "ﬄ", "ffl", "\u00ad", "", "\u00a0", " ", "\u2009", " ", "\u202f", " ", "\u200b", "")

var spaceRun = regexp.MustCompile(`[ \t]{2,}`)

// Layout makes the text layer of a page plain: it trims each line, joins runs of spaces that pdftotext -layout writes for columns, and writes ligatures as letters.
func Layout(s string) string {
	s = plainChars.Replace(strings.ReplaceAll(s, "\r\n", "\n"))
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = spaceRun.ReplaceAllString(strings.TrimSpace(l), " ")
	}
	return strings.TrimSpace(blankRun.ReplaceAllString(strings.Join(lines, "\n"), "\n\n"))
}

// TeX puts the math of a plain text into TeX (spec 05.7, the TeX lane). This is the form that the text layer of a PDF gives: Unicode superscripts and subscripts, Greek letters, fractions, letters of the math block and symbols become TeX, and each run of math goes between dollars. A run of math is a run of words with at least one such symbol or an operator, with the numbers and the short names of points and variables next to an operator, as in "x ≤ 2y". The words of the text do not change, and a text that is already between dollars stays as it is.
func TeX(s string) string {
	var out, plain strings.Builder
	flush := func() {
		lines := strings.Split(plain.String(), "\n")
		for i, l := range lines {
			lines[i] = texLine(l)
		}
		out.WriteString(strings.Join(lines, "\n"))
		plain.Reset()
	}
	display, inline := false, false
	for i := 0; i < len(s); {
		in := display || inline
		switch {
		case strings.HasPrefix(s[i:], `\$`):
			if in {
				out.WriteString(`\$`)
			} else {
				plain.WriteString(`\$`)
			}
			i += 2
		case strings.HasPrefix(s[i:], "$$") && !inline:
			flush()
			display = !display
			out.WriteString("$$")
			i += 2
		case s[i] == '$' && !display:
			flush()
			inline = !inline
			out.WriteString("$")
			i++
		default:
			if in {
				out.WriteByte(s[i])
			} else {
				plain.WriteByte(s[i])
			}
			i++
		}
	}
	if display || inline {
		// A $ with no pair: the text is not clear, so it stays as it was.
		return s
	}
	flush()
	return out.String()
}

// token is one word of a line, with the punctuation around it apart.
type token struct {
	lead, core, trail string
	tex               string
	strong, weak, op  bool
}

// atom is a number, a letter with at most one digit, a function of a short argument, or a short group, as a part of an expression such as a+b=c.
const atom = `(?:[0-9]+|\pL[0-9]?|\pL\([\pL0-9,]{1,8}\)|\([\pL0-9+,-]{1,9}\))`

var (
	weakRE   = regexp.MustCompile(`^(?:[0-9]+(?:[.,][0-9]+)?|\pL[0-9]*|[0-9]+\pL{1,3}|\p{Lu}{2,4}|\((?:[\pL0-9]{1,3}\s*,\s*)+[\pL0-9]{1,3}\))$`)
	funcRE   = regexp.MustCompile(`^\pL\(.+\)$`)
	opRE     = regexp.MustCompile(`^(?:[=<>+*/^|-]|<=|>=|:=)$`)
	exprRE   = regexp.MustCompile(`^` + atom + `(?:[=<>+*/^_|]` + atom + `)+$`)
	greekRun = regexp.MustCompile(`\p{Greek}{3,}`)
)

// texLine puts the math of one line, with no dollars, into TeX.
func texLine(line string) string {
	if strings.TrimSpace(line) == "" {
		return line
	}
	fields := strings.Split(line, " ")
	toks := make([]token, len(fields))
	// A line with a Greek word is Greek text, so its Greek letters are letters and not math.
	words := greekRun.MatchString(line)
	for i, f := range fields {
		toks[i] = newToken(f, words)
	}
	var out []string
	for i := 0; i < len(toks); {
		if !toks[i].strong && !toks[i].weak {
			out = append(out, fields[i])
			i++
			continue
		}
		// A run of math and weak words. A word with punctuation after it ends the run.
		j := i
		for j < len(toks) && (toks[j].strong || toks[j].weak) {
			j++
			if toks[j-1].trail != "" || (j < len(toks) && toks[j].lead != "") {
				break
			}
		}
		lo, hi := trimWeak(toks[i:j])
		for k := i; k < i+lo; k++ {
			out = append(out, fields[k])
		}
		if lo < hi {
			run := toks[i+lo : i+hi]
			var b strings.Builder
			for k, t := range run {
				if k > 0 {
					b.WriteString(" ")
				}
				b.WriteString(t.tex)
			}
			out = append(out, run[0].lead+"$"+strings.TrimSpace(b.String())+"$"+run[len(run)-1].trail)
		}
		for k := i + hi; k < j; k++ {
			out = append(out, fields[k])
		}
		i = j
	}
	return strings.Join(out, " ")
}

// trimWeak gives the part of a run that is math: from the first strong word to the last, with the weak words at each end that touch an operator.
func trimWeak(run []token) (int, int) {
	first, last := -1, -1
	for k, t := range run {
		if t.strong {
			if first < 0 {
				first = k
			}
			last = k
		}
	}
	if first < 0 {
		return 0, 0
	}
	lo, hi := first, last+1
	for lo > 0 && (run[lo].op || run[lo-1].op) {
		lo--
	}
	for hi < len(run) && (run[hi-1].op || run[hi].op) {
		hi++
	}
	return lo, hi
}

func newToken(f string, greekWords bool) token {
	t := token{core: f}
	if f == "" {
		return t
	}
	// Punctuation at the end stays out of the math. A ) stays in when the word opens it.
	for t.core != "" {
		r, n := utf8.DecodeLastRuneInString(t.core)
		if strings.ContainsRune(".,;:!?", r) || (r == ')' && strings.Count(t.core, "(") < strings.Count(t.core, ")")) {
			t.trail = t.core[len(t.core)-n:] + t.trail
			t.core = t.core[:len(t.core)-n]
			continue
		}
		break
	}
	if strings.HasPrefix(t.core, "(") && strings.Count(t.core, "(") > strings.Count(t.core, ")") {
		t.lead, t.core = "(", t.core[1:]
	}
	if t.core == "" {
		return token{core: f}
	}
	t.tex, t.strong = convert(t.core, greekWords)
	switch {
	case t.strong:
	case strings.Trim(t.core, "…⋯") == "":
		// Dots alone join a run of math, as in x₁ + … + xₙ.
		t.weak, t.op = true, true
	case opRE.MatchString(t.core):
		t.strong, t.op = true, true
	case funcRE.MatchString(t.core) && !strings.ContainsAny(t.core, " "), exprRE.MatchString(t.core):
		t.strong = true
	case weakRE.MatchString(t.core):
		t.weak = true
	}
	if t.strong && !t.op {
		t.op = opRE.MatchString(t.tex) || strings.HasPrefix(t.tex, `\`) && symbolOp(t.tex)
	}
	return t
}

// symbolOp tells if the TeX of a word is one relation or operator alone, as \le.
func symbolOp(tex string) bool {
	for r, s := range symbols {
		if s == tex && r != '°' && r != '′' && r != '″' && r != 'ℎ' && !strings.HasPrefix(s, `\mathbb`) && s != `\infty` && s != `\emptyset` {
			return true
		}
	}
	return false
}

// convert gives the TeX of one word and tells if the word had a math character.
func convert(word string, greekWords bool) (string, bool) {
	if greekRun.MatchString(word) {
		// Three Greek letters in a row are a Greek word, not a product.
		return word, false
	}
	rs := []rune(word)
	var b strings.Builder
	strong := false
	command := false
	write := func(s string) {
		if command && s != "" {
			r, _ := utf8.DecodeRuneInString(s)
			if unicode.IsLetter(r) {
				b.WriteString(" ")
			}
		}
		b.WriteString(s)
		command = strings.HasPrefix(s, `\`) && unicode.IsLetter(rune(s[len(s)-1]))
	}
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		if m, ok := mathLetter(r); ok {
			strong = true
			r = m
		}
		switch {
		case supers[r] != 0 || subs[r] != 0:
			table, mark := supers, "^"
			if subs[r] != 0 {
				table, mark = subs, "_"
			}
			var run []rune
			for i < len(rs) && table[rs[i]] != 0 {
				run = append(run, table[rs[i]])
				i++
			}
			i--
			if len(run) == 1 {
				write(mark + string(run))
			} else {
				write(mark + "{" + string(run) + "}")
			}
			strong = true
		case r == '√':
			arg, n := sqrtArg(rs[i+1:])
			sub, _ := convert(arg, false)
			write(`\sqrt{` + sub + `}`)
			i += n
			strong = true
		case r == '⁄' && i > 0 && i+1 < len(rs):
			// The fraction slash between two numbers, as in 1⁄2.
			num := strings.TrimRightFunc(b.String(), unicode.IsDigit)
			top := b.String()[len(num):]
			j := i + 1
			for j < len(rs) && unicode.IsDigit(rs[j]) {
				j++
			}
			b.Reset()
			b.WriteString(num)
			write(`\frac{` + top + `}{` + string(rs[i+1:j]) + `}`)
			i = j - 1
			strong = true
		case vulgar[r] != "":
			write(vulgar[r])
			strong = true
		case greek[r] != "" && !greekWords:
			write(greek[r])
			strong = true
		case symbols[r] != "":
			write(symbols[r])
			strong = true
		case weakSymbols[r] != "":
			write(weakSymbols[r])
		case r == '%':
			write(`\%`)
		default:
			write(string(r))
		}
	}
	return b.String(), strong
}

// sqrtArg gives the argument of a √: a group in parentheses, or the letters and digits after it, and the number of runes it takes.
func sqrtArg(rs []rune) (string, int) {
	if len(rs) > 0 && rs[0] == '(' {
		depth := 0
		for i, r := range rs {
			switch r {
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					return string(rs[1:i]), i + 1
				}
			}
		}
	}
	n := 0
	for n < len(rs) && (unicode.IsLetter(rs[n]) || unicode.IsDigit(rs[n])) {
		n++
	}
	return string(rs[:n]), n
}

// encodingFlag tells if a text layer has characters that a font with no Unicode map gives: the private use area, the replacement character and control characters.
func encodingFlag(s string) bool {
	for _, r := range s {
		switch {
		case r == '\n' || r == '\t' || r == '\f':
		case r == utf8.RuneError, r >= 0xE000 && r <= 0xF8FF, unicode.IsControl(r):
			return true
		}
	}
	return false
}
