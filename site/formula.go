package site

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// formulaSkip are the commands that change only the look of a formula. The tokens of a formula leave them out.
var formulaSkip = map[string]bool{
	"left": true, "right": true, "big": true, "Big": true, "bigg": true, "Bigg": true, "bigl": true, "bigr": true, "Bigl": true, "Bigr": true,
	"displaystyle": true, "textstyle": true, "mathrm": true, "mathbf": true, "mathit": true, "text": true, "operatorname": true, "quad": true, "qquad": true,
}

// formulaSame maps a command to the command with the same meaning, so that \leq and \le give the same token.
var formulaSame = map[string]string{"leq": "le", "geq": "ge", "neq": "ne", "dfrac": "frac", "tfrac": "frac", "lvert": "|", "rvert": "|", "vert": "|", "mid": "|", "ldots": "dots", "cdots": "dots", "leqslant": "le", "geqslant": "ge"}

// FormulaTokens gives the tokens of a TeX formula, separated by one space. A command gives its name. A letter, a digit and an operator each give themselves, so \frac12 and \frac{1}{2} give the same tokens. Braces, scripts, dollars and spaces give nothing. The search script has the same rules, so that a formula in a query matches the same formula in a statement: \frac{a}{b} gives "frac a b".
func FormulaTokens(tex string) string {
	var out []string
	for i := 0; i < len(tex); {
		c := tex[i]
		switch {
		case c == '\\':
			j := i + 1
			for j < len(tex) && isLetter(tex[j]) {
				j++
			}
			name := tex[i+1 : j]
			if name == "" {
				if j < len(tex) && (tex[j] == '{' || tex[j] == '}' || tex[j] == '|') {
					out = append(out, tex[j:j+1])
				}
				j++
			} else if !formulaSkip[name] {
				if s, ok := formulaSame[name]; ok {
					name = s
				}
				out = append(out, name)
			}
			i = j
		case isLetter(c) || c >= '0' && c <= '9' || strings.IndexByte("+-=<>()[]|,.!/*:;", c) >= 0:
			out = append(out, tex[i:i+1])
			i++
		case c >= utf8.RuneSelf:
			r, n := utf8.DecodeRuneInString(tex[i:])
			if !unicode.IsSpace(r) {
				out = append(out, string(r))
			}
			i += n
		default:
			i++
		}
	}
	return strings.Join(out, " ")
}

func isLetter(c byte) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// formulas gives the tokens of each formula of a Markdown text, separated by " ; ".
func formulas(src string) string {
	_, spans := splitMath(src)
	parts := make([]string, 0, len(spans))
	for _, s := range spans {
		if t := FormulaTokens(s.tex); t != "" {
			parts = append(parts, t)
		}
	}
	return strings.Join(parts, " ; ")
}
