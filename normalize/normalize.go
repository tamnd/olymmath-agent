// Package normalize makes statement texts comparable and renderable (spec 06.4). It does not change mathematics: numbers, variables and operators stay as written. The functions are deterministic, and Version names the rules so that a record can say which normalizer made its text.
package normalize

import (
	"regexp"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Version names the normalization rules. Change it when a rule changes.
const Version = "norm/3"

var punct = strings.NewReplacer(
	"‘", "'", "’", "'", "‚", "'", "‛", "'",
	"“", `"`, "”", `"`, "„", `"`, "«", `"`, "»", `"`,
	"–", "-", "—", "-", "−", "-",
	"\u00a0", " ", "\u2009", " ", "\u202f", " ", "\u200b", "",
	"\r\n", "\n", "\r", "\n", "\t", " ",
)

// aops replaces the old AoPS macros \plus{}, \minus{} and \equal{} with their signs. KaTeX does not know these macros.
var aops = strings.NewReplacer(`\plus{}`, "+", `\minus{}`, "-", `\equal{}`, "=")

var (
	displayRE = regexp.MustCompile(`(?s)\\\[(.*?)\\\]`)
	inlineRE  = regexp.MustCompile(`(?s)\\\((.*?)\\\)`)
	noOpRE    = regexp.MustCompile(`\\(displaystyle|textstyle|;|,|!|quad|qquad)(?:\s|\b|$)`)
	hideRE    = regexp.MustCompile(`(?is)\[hide(?:=[^\]]*)?\].*?\[/hide\]`)
	spaceRE   = regexp.MustCompile(`[ ]+`)
	blankRE   = regexp.MustCompile(`\n\s*\n+`)
	// markerRE matches a problem marker at the start of a text: "Problem 3.", "P3)", "3." or "#3:".
	markerRE = regexp.MustCompile(`(?i)^\s*(?:\*\*)?(?:problem|prob\.?|p|#)?\s*([a-z]?\s*[0-9]{1,2})\s*(?:[.):]|\s-)(?:\*\*)?\s+`)
)

// Text normalizes a statement: Unicode NFC, a fixed set of quotes and dashes, the signs for the old AoPS macros \plus{}, \minus{} and \equal{}, `\[...\]` to `$$...$$`, `\(...\)` to `$...$`, no spacing commands that have no meaning, no `[hide]` blocks, one space between words and one blank line between paragraphs.
func Text(s string) string {
	s = norm.NFC.String(s)
	s = punct.Replace(s)
	s = aops.Replace(s)
	s = displayRE.ReplaceAllString(s, "$$$$$1$$$$")
	s = inlineRE.ReplaceAllString(s, "$$$1$$")
	s = noOpRE.ReplaceAllString(s, " ")
	s = hideRE.ReplaceAllString(s, "")
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = strings.TrimSpace(spaceRE.ReplaceAllString(l, " "))
	}
	s = strings.Join(lines, "\n")
	s = blankRE.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

// StripMarker removes a problem marker at the start of s when the marker repeats the position, for example "Problem 3." for p3 or "A1." for a1.
func StripMarker(s, pos string) string {
	m := markerRE.FindStringSubmatchIndex(s)
	if m == nil {
		return s
	}
	got := strings.ToLower(strings.ReplaceAll(s[m[2]:m[3]], " ", ""))
	if got[0] >= '0' && got[0] <= '9' {
		got = "p" + got
	}
	if got != pos {
		return s
	}
	return s[m[1]:]
}

// Statement is Text, then StripMarker, then StripCountry, then CutTail.
func Statement(s, pos string) string { return CutTail(StripCountry(StripMarker(Text(s), pos))) }

// countryRE matches the code of the proposing country at the start of a shortlist text, for example "BEL Let $ABC$ be a triangle".
var countryRE = regexp.MustCompile(`^[A-Z]{3} (?:Variant [0-9]+\. )?([A-Z][a-z]|[A-Z] )`)

// StripCountry removes the code of the proposing country at the start of a text.
func StripCountry(s string) string {
	m := countryRE.FindStringSubmatchIndex(s)
	if m == nil {
		return s
	}
	return s[m[2]:]
}

// tailRE finds the start of text that follows a statement in some datasets: a heading of the next problem, such as "## C2", or the start of an answer or a solution.
var tailRE = regexp.MustCompile(`\s(?:#{1,6}\s|(?:Answer|Solution|Proof|Hint)\s*[.:]\s)`)

// minHead is the shortest text, in bytes, that CutTail keeps before a cut.
const minHead = 40

// CutTail removes the text after the end of a statement: the next problem or the answer that a dataset joined to the statement. It does not cut in the first minHead bytes.
func CutTail(s string) string {
	for _, m := range tailRE.FindAllStringIndex(s, -1) {
		if m[0] >= minHead {
			return strings.TrimSpace(s[:m[0]])
		}
	}
	return s
}

// Tokens splits a normalized text into lower case words for matching. A word is a run of letters or digits. A TeX command, such as \frac, is one token, and each other symbol is one token. Thus "$x^2$" and "$x ^ 2$" give the same tokens.
func Tokens(s string) []string {
	var out []string
	rs := []rune(strings.ToLower(s))
	for i := 0; i < len(rs); {
		r := rs[i]
		switch {
		case unicode.IsSpace(r) || r == '$' || r == '{' || r == '}':
			i++
		case r == '\\':
			j := i + 1
			for j < len(rs) && unicode.IsLetter(rs[j]) {
				j++
			}
			if j == i+1 && j < len(rs) {
				j++
			}
			out = append(out, string(rs[i:j]))
			i = j
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			j := i
			for j < len(rs) && (unicode.IsLetter(rs[j]) || unicode.IsDigit(rs[j])) {
				j++
			}
			out = append(out, string(rs[i:j]))
			i = j
		default:
			out = append(out, string(r))
			i++
		}
	}
	return out
}
