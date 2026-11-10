// Package keymatch places a seed row with no position by its content: it compares the row with the texts of the link-only problems of a competition, such as the problems.ru texts of the raw store, which are a private key and never a text of the content. The two texts can be in two languages, so the comparison reads only what a translation keeps: the numbers, the letters of the formulas, the names of points, the functions and the symbols, and the order in which they come.
package keymatch

import (
	"regexp"
	"strings"
	"unicode"
)

// Features are the language-free features of a text.
type Features struct {
	// Tokens are the kinds and values of the formula parts: n:17 for a number, v:x for a letter, p:ABC for the name of a figure, f:sin for a function and s:sqrt for a symbol.
	Tokens map[string]bool
	// Grams are the runs of three formula parts in the order of the text, such as "x ^ 2". They give the shape of the expressions.
	Grams map[string]bool
}

// Empty tells if a text has no feature.
func (f Features) Empty() bool { return len(f.Tokens) == 0 }

var (
	// mathSegRE finds the formulas of a TeX text.
	mathSegRE = regexp.MustCompile(`(?s)\$\$(.*?)\$\$|\$(.*?)\$|\\\((.*?)\\\)|\\\[(.*?)\\\]`)
	// leadRE is the mark of the problem at the start of a seed row, such as "5.", "11.4.", "Problem 3:" or "Задача 2.", which the key text does not have.
	leadRE = regexp.MustCompile(`^\s*(?:(?:Problem|Задача|Task)\s*)?[0-9]{1,3}(?:\.[0-9]{1,3})*\s*[.):]\s*`)
	// pointsRE is the score of a problem, such as "(10 points)" or "[7 баллов]".
	pointsRE = regexp.MustCompile(`(?i)[(\[]\s*[0-9]+\s*(?:points?|pts?|marks?|балл\p{L}*)\s*[)\]]`)
	// commandRE is a TeX command.
	commandRE = regexp.MustCompile(`\\[A-Za-z]+`)
)

// commands maps the TeX commands that a translation keeps to a symbol of the stream. Other commands, such as \left or \mathrm, are left out.
var commands = map[string]string{
	`\le`: "<=", `\leq`: "<=", `\leqslant`: "<=", `\ge`: ">=", `\geq`: ">=", `\geqslant`: ">=",
	`\ne`: "!=", `\neq`: "!=", `\sqrt`: "sqrt", `\frac`: "/", `\dfrac`: "/", `\tfrac`: "/",
	`\cdot`: "*", `\times`: "*", `\circ`: "deg", `\angle`: "angle", `\pi`: "pi", `\infty`: "inf",
	`\sin`: "sin", `\cos`: "cos", `\tan`: "tan", `\tg`: "tan", `\cot`: "cot", `\ctg`: "cot", `\log`: "log", `\ln`: "ln", `\lg`: "lg",
	`\max`: "max", `\min`: "min", `\gcd`: "gcd", `\sum`: "sum", `\prod`: "prod", `\triangle`: "triangle", `\parallel`: "parallel", `\perp`: "perp",
	`\alpha`: "alpha", `\beta`: "beta", `\gamma`: "gamma", `\delta`: "delta", `\varphi`: "phi", `\phi`: "phi", `\lambda`: "lambda", `\omega`: "omega",
}

// runeSymbols maps the Unicode signs of a plain text to the symbols of the stream.
var runeSymbols = map[rune]string{
	'²': "^ 2", '³': "^ 3", '¹': "^ 1", '⁴': "^ 4", '⁰': "^ 0", '⁵': "^ 5", '⁶': "^ 6", '⁷': "^ 7", '⁸': "^ 8", '⁹': "^ 9", 'ⁿ': "^ n",
	'×': "*", '·': "*", '⋅': "*", '−': "-", '≤': "<=", '⩽': "<=", '≥': ">=", '⩾': ">=", '≠': "!=", '√': "sqrt", '°': "deg",
	'∠': "angle", 'π': "pi", '∞': "inf", '÷': "/", 'α': "alpha", 'β': "beta", 'γ': "gamma", 'δ': "delta", 'φ': "phi", 'λ': "lambda", 'ω': "omega",
}

// functions are the words of formulas that are the same in each language.
var functions = map[string]string{"sin": "sin", "cos": "cos", "tg": "tan", "tan": "tan", "ctg": "cot", "cot": "cot", "log": "log", "ln": "ln", "lg": "lg", "max": "max", "min": "min", "gcd": "gcd", "lcm": "lcm", "mod": "mod", "arctg": "arctan", "arctan": "arctan", "arcsin": "arcsin", "arccos": "arccos"}

// numberWords are the English words of small numbers. A translation often writes a number in words on one side and in digits on the other. One is left out, because "one" and "один" are also pronouns.
var numberWords = map[string]string{
	"two": "2", "three": "3", "four": "4", "five": "5", "six": "6", "seven": "7", "eight": "8", "nine": "9", "ten": "10", "eleven": "11", "twelve": "12", "twenty": "20", "hundred": "100", "thousand": "1000",
	"twice": "2", "thrice": "3", "triple": "3", "double": "2",
}

// ruNumberForms are the forms of the Russian words of small numbers, with the number.
var ruNumberForms = map[string]string{
	"два": "2", "две": "2", "двух": "2", "двум": "2", "двумя": "2", "дважды": "2",
	"три": "3", "трёх": "3", "трех": "3", "трём": "3", "трем": "3", "тремя": "3", "трижды": "3",
	"четыре": "4", "четырёх": "4", "четырех": "4", "четырём": "4", "четырем": "4", "четырьмя": "4",
	"пять": "5", "пяти": "5", "пятью": "5", "шесть": "6", "шести": "6", "шестью": "6", "семь": "7", "семи": "7", "семью": "7",
	"восемь": "8", "восьми": "8", "восемью": "8", "девять": "9", "девяти": "9", "девятью": "9", "десять": "10", "десяти": "10", "десятью": "10",
	"одиннадцать": "11", "одиннадцати": "11", "двенадцать": "12", "двенадцати": "12", "двадцать": "20", "двадцати": "20",
	"сто": "100", "ста": "100", "тысяча": "1000", "тысячи": "1000", "тысячу": "1000", "тысяч": "1000",
}

// operators are the signs that go into the stream.
const operators = "=<>+-*/^!"

// Extract gives the features of a text. A text in Russian, where each Latin letter is part of a formula, is read whole. In another text only the formulas are read whole; the plain text gives its numbers, its capital letters except A and I, its runs of capital letters, such as ABC, and the signs = < >.
func Extract(text string) Features {
	text = leadRE.ReplaceAllString(text, "")
	text = pointsRE.ReplaceAllString(text, " ")
	var stream []string
	if cyrillic(text) {
		stream = read(text, true)
	} else {
		last := 0
		for _, m := range mathSegRE.FindAllStringSubmatchIndex(text, -1) {
			stream = append(stream, read(text[last:m[0]], false)...)
			for k := 2; k < len(m); k += 2 {
				if m[k] >= 0 {
					stream = append(stream, read(text[m[k]:m[k+1]], true)...)
				}
			}
			last = m[1]
		}
		stream = append(stream, read(text[last:], false)...)
	}
	f := Features{Tokens: map[string]bool{}, Grams: map[string]bool{}}
	for _, t := range stream {
		if k := kind(t); k != "" {
			f.Tokens[k+":"+t] = true
		}
	}
	for i := 0; i+3 <= len(stream); i++ {
		f.Grams[strings.Join(stream[i:i+3], " ")] = true
	}
	return f
}

// cyrillic tells if most letters of a text outside its formulas are Cyrillic.
func cyrillic(text string) bool {
	text = mathSegRE.ReplaceAllString(text, " ")
	cy, lat := 0, 0
	for _, r := range text {
		switch {
		case unicode.Is(unicode.Cyrillic, r):
			cy++
		case r < 128 && unicode.IsLetter(r):
			lat++
		}
	}
	return cy > lat
}

// kind gives the kind of a part of the stream, or "" for an operator, which only goes into the grams.
func kind(t string) string {
	r := []rune(t)
	switch {
	case unicode.IsDigit(r[0]):
		return "n"
	case len(r) == 1 && unicode.IsLetter(r[0]):
		return "v"
	case strings.ContainsRune(operators, r[0]) && len(r) <= 2:
		return ""
	case capitalRun(t):
		return "p"
	}
	return "s"
}

func capitalRun(t string) bool {
	if len(t) < 2 || len(t) > 5 {
		return false
	}
	for _, r := range t {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// read gives the stream of one piece of a text. In a formula, or in a Russian text, a run of Latin letters is a function name, a run of capitals, which names a figure, or else a product of letters, each of which is a part. In plain text in another language, only numbers, capitals other than A and I, runs of capitals and the signs = < > count.
func read(s string, formula bool) []string {
	var out []string
	s = commandRE.ReplaceAllStringFunc(s, func(c string) string {
		if v, ok := commands[c]; ok {
			return " \x01" + v + "\x01 "
		}
		if formula {
			return " "
		}
		return c
	})
	rs := []rune(s)
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case r == '\x01':
			// A symbol that a command gave.
			j := i + 1
			for j < len(rs) && rs[j] != '\x01' {
				j++
			}
			out = append(out, string(rs[i+1:j]))
			i = j + 1
		case unicode.IsDigit(r):
			j := i
			for j < len(rs) && unicode.IsDigit(rs[j]) {
				j++
			}
			// A decimal fraction, such as 0.5 or 2,5, is one number.
			if j+1 < len(rs) && (rs[j] == '.' || rs[j] == ',') && unicode.IsDigit(rs[j+1]) && formula {
				j++
				for j < len(rs) && unicode.IsDigit(rs[j]) {
					j++
				}
			}
			out = append(out, number(string(rs[i:j])))
			i = j
		case r < 128 && unicode.IsLetter(r):
			j := i
			for j < len(rs) && rs[j] < 128 && unicode.IsLetter(rs[j]) {
				j++
			}
			w := string(rs[i:j])
			i = j
			out = append(out, word(w, formula)...)
		case unicode.IsLetter(r):
			// A word of another script, such as a Russian word, is not a formula, but a Russian number word gives its number. A Greek letter is a formula.
			j := i
			for j < len(rs) && unicode.IsLetter(rs[j]) && rs[j] >= 128 {
				j++
			}
			w := strings.ToLower(string(rs[i:j]))
			if v, ok := runeSymbols[r]; ok && j == i+1 {
				out = append(out, strings.Fields(v)...)
			} else if v, ok := ruNumberForms[w]; ok {
				out = append(out, v)
			}
			i = j
		default:
			if v, ok := runeSymbols[r]; ok && (formula || v != "-") {
				out = append(out, strings.Fields(v)...)
			} else if strings.ContainsRune(operators, r) && (formula || strings.ContainsRune("=<>", r)) {
				op := string(r)
				if i+1 < len(rs) && rs[i+1] == '=' && strings.ContainsRune("<>!", r) {
					op += "="
					i++
				}
				out = append(out, op)
			}
			i++
		}
	}
	return out
}

// word gives the parts of a run of Latin letters.
func word(w string, formula bool) []string {
	if v, ok := functions[strings.ToLower(w)]; ok && (formula || w == strings.ToLower(w)) {
		return []string{v}
	}
	if capitalRun(w) {
		return []string{w}
	}
	if v, ok := numberWords[strings.ToLower(w)]; ok && len(w) > 2 {
		return []string{v}
	}
	if !formula {
		if len(w) == 1 && w != "A" && w != "I" && w[0] >= 'A' && w[0] <= 'Z' {
			return []string{w}
		}
		return nil
	}
	if len(w) > 6 {
		// A long word in a formula is a word, as in \text{if}.
		return nil
	}
	out := make([]string, 0, len(w))
	for _, r := range w {
		out = append(out, string(r))
	}
	return out
}

// number removes the zeros at the start of a whole number.
func number(s string) string {
	if strings.ContainsAny(s, ".,") {
		return strings.ReplaceAll(s, ",", ".")
	}
	if t := strings.TrimLeft(s, "0"); t != "" {
		return t
	}
	return "0"
}
