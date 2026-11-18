package official

import (
	"encoding/xml"
	"errors"
	"io"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MathMLTeX gives the TeX of a MathML element, as make4ht writes it for the HTML of a LaTeX paper: the tokens and the layout elements of presentation MathML. ok is false when the element holds a token or an element that has no TeX here, so that the TeX may not be all of the formula.
func MathMLTeX(src string) (tex string, ok bool) {
	root, err := mmlParse(src)
	if err != nil {
		return "", false
	}
	c := &mmlConv{ok: true}
	tex = c.kids(root)
	tex = mmlSpaceRE.ReplaceAllString(strings.TrimSpace(tex), " ")
	return mmlTidyRE.ReplaceAllString(tex, "$1"), c.ok
}

var (
	mmlSpaceRE = regexp.MustCompile(`\s+`)
	// mmlTidyRE is a space before a closing brace or a script sign, or after an opening brace, which the commands leave.
	mmlTidyRE = regexp.MustCompile(` ([}_^])`)
)

// mmlNode is an element of a MathML tree, or a text node when name is "".
type mmlNode struct {
	name  string
	attrs map[string]string
	kids  []*mmlNode
	text  string
}

func mmlParse(src string) (*mmlNode, error) {
	d := xml.NewDecoder(strings.NewReader(mmlAttrRE.ReplaceAllString(src, `$1="$2"`)))
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	top := &mmlNode{name: "#root"}
	stack := []*mmlNode{top}
	for {
		tok, err := d.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		cur := stack[len(stack)-1]
		switch t := tok.(type) {
		case xml.StartElement:
			n := &mmlNode{name: strings.ToLower(t.Name.Local), attrs: map[string]string{}}
			for _, a := range t.Attr {
				n.attrs[strings.ToLower(a.Name.Local)] = a.Value
			}
			cur.kids = append(cur.kids, n)
			stack = append(stack, n)
		case xml.EndElement:
			if len(stack) > 1 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			cur.kids = append(cur.kids, &mmlNode{text: string(t)})
		}
	}
	if len(stack) != 1 {
		return nil, errors.New("mathml: an element is not closed")
	}
	return top, nil
}

// elems gives the element children of n, without the text between them.
func (n *mmlNode) elems() []*mmlNode {
	var out []*mmlNode
	for _, k := range n.kids {
		if k.name != "" {
			out = append(out, k)
		}
	}
	return out
}

// textOf gives the text of n and its children.
func (n *mmlNode) textOf() string {
	if n.name == "" {
		return n.text
	}
	var b strings.Builder
	for _, k := range n.kids {
		b.WriteString(k.textOf())
	}
	return strings.TrimSpace(b.String())
}

type mmlConv struct{ ok bool }

func (c *mmlConv) kids(n *mmlNode) string {
	var b strings.Builder
	for _, k := range n.elems() {
		b.WriteString(c.node(k))
	}
	return b.String()
}

// group gives the TeX of n as one argument: in braces unless it is one letter, digit or command.
func (c *mmlConv) group(n *mmlNode) string {
	s := strings.TrimSpace(c.node(n))
	if utf8.RuneCountInString(s) == 1 || mmlCommandRE.MatchString(s) {
		return s
	}
	return "{" + s + "}"
}

// arg gives the TeX of n in braces.
func (c *mmlConv) arg(n *mmlNode) string {
	return "{" + strings.TrimSpace(c.node(n)) + "}"
}

var (
	mmlCommandRE = regexp.MustCompile(`^\\[A-Za-z]+$`)
	// mmlEndCmdRE is TeX that ends with the name of a command, which a letter may not follow.
	mmlEndCmdRE = regexp.MustCompile(`\\[A-Za-z]+$`)
	// mmlAttrRE is an attribute of HTML whose value has no quotes.
	mmlAttrRE = regexp.MustCompile(`(\s[A-Za-z:-]+)=([^\s"'>]+)`)
)

func (c *mmlConv) node(n *mmlNode) string {
	es := n.elems()
	switch n.name {
	case "math", "mrow", "mpadded", "semantics":
		return c.kids(n)
	case "mstyle":
		in := c.kids(n)
		switch n.attrs["mathvariant"] {
		case "bold":
			return `\mathbf{` + strings.TrimSpace(in) + "} "
		}
		return in
	case "mphantom", "annotation", "annotation-xml":
		return ""
	case "mi":
		return mmlIdent(n.textOf(), n.attrs["mathvariant"], c)
	case "mn":
		return mmlChars(n.textOf(), c)
	case "mo":
		return mmlChars(n.textOf(), c)
	case "mtext":
		// The spaces at the ends of a text are its spacing in the formula, as in "6623 digits".
		var b strings.Builder
		for _, k := range n.kids {
			b.WriteString(k.textOf())
		}
		t := strings.Join(strings.Fields(b.String()), " ")
		if t == "" {
			return " "
		}
		if r := b.String(); strings.TrimLeftFunc(r, unicode.IsSpace) != r {
			t = " " + t
		}
		if r := b.String(); strings.TrimRightFunc(r, unicode.IsSpace) != r {
			t += " "
		}
		return `\text{` + mmlTextEscape(t) + "} "
	case "ms":
		return `\text{"` + mmlTextEscape(n.textOf()) + `"} `
	case "mspace":
		if strings.Contains(n.attrs["class"], "thinspace") || strings.HasPrefix(n.attrs["width"], "0.1") {
			return `\, `
		}
		return " "
	case "msub", "msup", "msubsup":
		if len(es) < 2 || n.name == "msubsup" && len(es) < 3 {
			c.ok = false
			return c.kids(n)
		}
		base := strings.TrimSpace(c.node(es[0]))
		if base == "" || strings.HasSuffix(base, "}") && !strings.HasPrefix(base, `\`) || strings.Contains(base, " ") && !mmlCommandRE.MatchString(base) {
			base = "{" + base + "}"
		}
		switch n.name {
		case "msub":
			return base + "_" + c.group(es[1]) + " "
		case "msup":
			return base + "^" + mmlSup(c, es[1]) + " "
		}
		return base + "_" + c.group(es[1]) + "^" + mmlSup(c, es[2]) + " "
	case "munder", "mover", "munderover":
		return c.under(n, es)
	case "mfrac":
		if len(es) != 2 {
			c.ok = false
			return c.kids(n)
		}
		if t := n.attrs["linethickness"]; t == "0" || t == "0pt" || t == "0em" {
			return `\genfrac{}{}{0pt}{}` + c.arg(es[0]) + c.arg(es[1]) + " "
		}
		return `\frac` + c.arg(es[0]) + c.arg(es[1]) + " "
	case "msqrt":
		return `\sqrt{` + strings.TrimSpace(c.kids(n)) + "} "
	case "mroot":
		if len(es) != 2 {
			c.ok = false
			return c.kids(n)
		}
		return `\sqrt[` + strings.TrimSpace(c.node(es[1])) + "]" + c.arg(es[0]) + " "
	case "mfenced":
		open, ok := n.attrs["open"]
		if !ok {
			open = "("
		}
		closing, ok := n.attrs["close"]
		if !ok {
			closing = ")"
		}
		var parts []string
		for _, e := range es {
			parts = append(parts, strings.TrimSpace(c.node(e)))
		}
		return `\left` + mmlFence(open) + strings.Join(parts, ",") + `\right` + mmlFence(closing) + " "
	case "mtable":
		var rows []string
		cols := 0
		for _, r := range es {
			var cells []string
			for _, d := range r.elems() {
				cells = append(cells, strings.TrimSpace(c.node(d)))
			}
			cols = max(cols, len(cells))
			rows = append(rows, strings.Join(cells, " & "))
		}
		return `\begin{array}{` + strings.Repeat("l", max(cols, 1)) + "}" + strings.Join(rows, ` \\ `) + `\end{array} `
	case "mtr", "mlabeledtr", "mtd":
		return c.kids(n)
	case "menclose":
		if strings.Contains(n.attrs["notation"], "box") {
			return `\boxed{` + strings.TrimSpace(c.kids(n)) + "} "
		}
		switch n.attrs["notation"] {
		case "updiagonalstrike":
			return `\cancel{` + strings.TrimSpace(c.kids(n)) + "} "
		case "downdiagonalstrike":
			return `\bcancel{` + strings.TrimSpace(c.kids(n)) + "} "
		case "updiagonalstrike downdiagonalstrike", "downdiagonalstrike updiagonalstrike":
			return `\xcancel{` + strings.TrimSpace(c.kids(n)) + "} "
		case "horizontalstrike":
			return `\sout{` + strings.TrimSpace(c.kids(n)) + "} "
		}
		c.ok = false
		return c.kids(n)
	case "":
		return ""
	}
	c.ok = false
	return c.kids(n)
}

// mmlSup gives a superscript: a prime as a prime, other scripts as a group.
func mmlSup(c *mmlConv, n *mmlNode) string {
	switch t := n.textOf(); t {
	case "′", "'":
		return `\prime`
	case "″":
		return `{\prime\prime}`
	case "∘", "°":
		return `\circ`
	}
	return c.group(n)
}

// under gives the TeX of munder, mover and munderover: limits of a big operator, an accent over its base, or one thing set over another.
func (c *mmlConv) under(n *mmlNode, es []*mmlNode) string {
	if len(es) < 2 || n.name == "munderover" && len(es) < 3 {
		c.ok = false
		return c.kids(n)
	}
	base := strings.TrimSpace(c.node(es[0]))
	if mmlBigOp[base] || base == `\lim` || base == `\max` || base == `\min` {
		switch n.name {
		case "munder":
			return base + "_" + c.group(es[1]) + " "
		case "mover":
			return base + "^" + c.group(es[1]) + " "
		}
		return base + "_" + c.group(es[1]) + "^" + c.group(es[2]) + " "
	}
	if n.name == "mover" {
		if es[1].textOf() == "⏞" {
			return `\overbrace{` + base + "} "
		}
		if acc, ok := mmlAccents[es[1].textOf()]; ok {
			return acc + "{" + base + "} "
		}
		return `\overset` + c.arg(es[1]) + "{" + base + "} "
	}
	if n.name == "munder" {
		switch es[1].textOf() {
		case "_", "‾", "¯", "̲":
			return `\underline{` + base + "} "
		case "⏟":
			return `\underbrace{` + base + "} "
		}
		return `\underset` + c.arg(es[1]) + "{" + base + "} "
	}
	return `\overset` + c.arg(es[2]) + `{\underset` + c.arg(es[1]) + "{" + base + "}} "
}

var mmlBigOp = map[string]bool{`\sum`: true, `\prod`: true, `\bigcup`: true, `\bigcap`: true, `\int`: true, `\coprod`: true, `\bigoplus`: true}

// mmlAccents are the TeX accents of the marks that mover sets over its base.
var mmlAccents = map[string]string{"‾": `\overline`, "¯": `\overline`, "_": `\overline`, "̅": `\overline`, "→": `\overrightarrow`, "⟶": `\overrightarrow`, "⃗": `\vec`, "^": `\widehat`, "ˆ": `\widehat`, "̂": `\widehat`, "~": `\widetilde`, "˜": `\widetilde`, "⌢": `\overset{\frown}`, "˙": `\dot`, "⋅": `\dot`, "¨": `\ddot`, "←": `\overleftarrow`, "↔": `\overleftrightarrow`, "⏜": `\overset{\frown}`}

func mmlFence(s string) string {
	switch s {
	case "":
		return "."
	case "{":
		return `\{`
	case "}":
		return `\}`
	case "⌊":
		return `\lfloor `
	case "⌋":
		return `\rfloor `
	case "⌈":
		return `\lceil `
	case "⌉":
		return `\rceil `
	case "⟨", "〈":
		return `\langle `
	case "⟩", "〉":
		return `\rangle `
	case "‖", "∥":
		return `\| `
	}
	return s
}

// mmlFunctions are the names that TeX sets upright as operators.
var mmlFunctions = map[string]bool{"sin": true, "cos": true, "tan": true, "cot": true, "sec": true, "csc": true, "arcsin": true, "arccos": true, "arctan": true, "sinh": true, "cosh": true, "tanh": true, "log": true, "ln": true, "lg": true, "exp": true, "lim": true, "max": true, "min": true, "sup": true, "inf": true, "gcd": true, "deg": true, "det": true, "dim": true, "ker": true, "arg": true, "mod": true, "Pr": true}

// mmlIdent gives the TeX of an identifier: one letter in italics, unless its variant is normal, a name of an operator, or a word upright.
func mmlIdent(t, variant string, c *mmlConv) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return ""
	}
	if mmlFunctions[t] {
		if t == "mod" {
			return `\bmod `
		}
		return `\` + t + " "
	}
	if utf8.RuneCountInString(t) == 1 {
		r, _ := utf8.DecodeRuneInString(t)
		s := mmlChars(t, c)
		switch {
		case variant == "double-struck" && unicode.IsUpper(r):
			return `\mathbb{` + t + "} "
		case variant == "script" || variant == "fraktur":
			return `\math` + map[string]string{"script": "cal", "fraktur": "frak"}[variant] + "{" + t + "} "
		case variant == "bold":
			return `\mathbf{` + s + "} "
		case variant == "normal" && unicode.IsLetter(r) && r < 0x80:
			return `\mathrm{` + t + "} "
		}
		return s
	}
	if !mmlWordRE.MatchString(t) {
		return mmlChars(t, c)
	}
	return `\mathrm{` + t + "} "
}

var mmlWordRE = regexp.MustCompile(`^[A-Za-z][A-Za-z ]*$`)

// mmlChars gives the TeX of the characters of a token. A character that has no TeX here leaves the token not ok.
func mmlChars(t string, c *mmlConv) string {
	var b strings.Builder
	for _, r := range t {
		switch {
		case r < 0x80:
			switch r {
			case '{', '}', '%', '#', '&', '$':
				b.WriteString(`\` + string(r))
			case '\\':
				b.WriteString(`\backslash `)
			case '~':
				b.WriteString(`\sim `)
			case '^':
				b.WriteString(`\hat{} `)
			case '_':
				b.WriteString(`\_`)
			default:
				b.WriteRune(r)
			}
		default:
			if s, ok := mmlSymbols[r]; ok {
				b.WriteString(s)
				if mmlEndCmdRE.MatchString(s) {
					b.WriteString(" ")
				}
				continue
			}
			if unicode.Is(unicode.Latin, r) && unicode.IsLetter(r) {
				// A letter with an accent, as the á of a name.
				b.WriteString(`\text{` + string(r) + "}")
				continue
			}
			if unicode.Is(unicode.Mn, r) || r == 0x2061 || r == 0x2062 || r == 0x2063 || r == 0x200b || r == 0xa0 {
				// The invisible function application, times and separator, and spaces.
				if r == 0xa0 {
					b.WriteString(" ")
				}
				continue
			}
			c.ok = false
			b.WriteRune(r)
		}
	}
	return b.String()
}

// mmlSymbols are the TeX of the characters beyond ASCII of the formulas.
var mmlSymbols = map[rune]string{
	'−': "-", '–': "-", '∗': "*", '×': `\times`, '·': `\cdot`, '⋅': `\cdot`, '∙': `\cdot`, '÷': `\div`, '±': `\pm`, '∓': `\mp`,
	'≤': `\le`, '≥': `\ge`, '⩽': `\leqslant`, '⩾': `\geqslant`, '≠': `\ne`, '≈': `\approx`, '≡': `\equiv`, '≢': `\not\equiv`, '∼': `\sim`, '≃': `\simeq`, '≅': `\cong`, '∝': `\propto`, '≪': `\ll`, '≫': `\gg`,
	'→': `\to`, '←': `\leftarrow`, '↔': `\leftrightarrow`, '⇒': `\Rightarrow`, '⇐': `\Leftarrow`, '⇔': `\Leftrightarrow`, '⟹': `\implies`, '⟺': `\iff`, '↦': `\mapsto`, '⟶': `\longrightarrow`, '↑': `\uparrow`, '↓': `\downarrow`,
	'∞': `\infty`, '∠': `\angle`, '∡': `\measuredangle`, '°': `^\circ`, '∘': `\circ`, '△': `\triangle`, '▵': `\triangle`, '□': `\square`, '◻': `\square`, '⊥': `\perp`, '∥': `\parallel`, '∦': `\nparallel`, '∣': `\mid`, '∤': `\nmid`, '|': "|", '‖': `\|`,
	'∈': `\in`, '∉': `\notin`, '∋': `\ni`, '⊂': `\subset`, '⊃': `\supset`, '⊆': `\subseteq`, '⊇': `\supseteq`, '⊊': `\subsetneq`, '∪': `\cup`, '∩': `\cap`, '∅': `\emptyset`, '∖': `\setminus`, '∀': `\forall`, '∃': `\exists`, '¬': `\neg`, '∧': `\wedge`, '∨': `\vee`, '⊕': `\oplus`, '⊗': `\otimes`,
	'∑': `\sum`, '∏': `\prod`, '∫': `\int`, '∂': `\partial`, '∇': `\nabla`, '√': `\surd`, '…': `\ldots`, '⋯': `\cdots`, '⋮': `\vdots`, '⋱': `\ddots`, '′': `'`, '″': `''`,
	'⌊': `\lfloor`, '⌋': `\rfloor`, '⌈': `\lceil`, '⌉': `\rceil`, '⟨': `\langle`, '⟩': `\rangle`, '〈': `\langle`, '〉': `\rangle`,
	'α': `\alpha`, 'β': `\beta`, 'γ': `\gamma`, 'δ': `\delta`, 'ε': `\varepsilon`, 'ϵ': `\epsilon`, 'ζ': `\zeta`, 'η': `\eta`, 'θ': `\theta`, 'ϑ': `\vartheta`, 'ι': `\iota`, 'κ': `\kappa`, 'λ': `\lambda`, 'μ': `\mu`, 'ν': `\nu`, 'ξ': `\xi`, 'π': `\pi`, 'ρ': `\rho`, 'ϱ': `\varrho`, 'σ': `\sigma`, 'ς': `\varsigma`, 'τ': `\tau`, 'υ': `\upsilon`, 'φ': `\varphi`, 'ϕ': `\phi`, 'χ': `\chi`, 'ψ': `\psi`, 'ω': `\omega`,
	'Γ': `\Gamma`, 'Δ': `\Delta`, 'Θ': `\Theta`, 'Λ': `\Lambda`, 'Ξ': `\Xi`, 'Π': `\Pi`, 'Σ': `\Sigma`, 'Υ': `\Upsilon`, 'Φ': `\Phi`, 'Ψ': `\Psi`, 'Ω': `\Omega`,
	'∕': "/", '⊙': `\odot`, '♡': `\heartsuit`, '♥': `\heartsuit`, '♣': `\clubsuit`, '♠': `\spadesuit`, '♢': `\diamondsuit`, '♦': `\diamondsuit`,
	'𝜀': `\varepsilon`, '𝜗': `\vartheta`, '𝜑': `\varphi`, '𝜋': `\pi`, '𝛼': `\alpha`, '𝛽': `\beta`, '𝛾': `\gamma`, '𝛿': `\delta`, '𝜆': `\lambda`, '𝜇': `\mu`, '𝜎': `\sigma`, '𝜔': `\omega`, '𝜃': `\theta`,
	'ℕ': `\mathbb{N}`, 'ℤ': `\mathbb{Z}`, 'ℚ': `\mathbb{Q}`, 'ℝ': `\mathbb{R}`, 'ℂ': `\mathbb{C}`, 'ℓ': `\ell`, '✓': `\checkmark`, '⋆': `\star`, '★': `\bigstar`,
}

// mmlTextEscape escapes the signs of TeX in the text of \text.
func mmlTextEscape(t string) string {
	r := strings.NewReplacer(`\`, `\textbackslash{}`, "{", `\{`, "}", `\}`, "$", `\$`, "%", `\%`, "#", `\#`, "&", `\&`, "_", `\_`, "^", `\textasciicircum{}`, "~", `\textasciitilde{}`)
	return r.Replace(t)
}
