package loose

import (
	"regexp"
	"strconv"
	"strings"
)

// polishRE is the mark of a problem of the Polish olympiad at the start of a text, as in "XXXVIII OM - I - Problem 11", "XXV - I - Problem 3", "LIX OM - III -Zadanie 2" or "I OM - B - Task 4": the olympiad in Roman numerals, the round and the number of the problem in the round.
var polishRE = regexp.MustCompile(`^\s*([IVXLC]{1,8})\s+(?:OM\s+)?-\s*(III|II|I|B)\s*-\s*(?:Problem|Task|Zadanie)\s+([0-9]{1,2})\b\s*[.:]?`)

// Mark is the mark of a problem of the Polish olympiad (spec 06.8).
type Mark struct {
	// Olympiad is the number of the olympiad, 1 for the olympiad of 1949/50.
	Olympiad int
	// Round is I, II or III, or B for a round of the first olympiad. Round III is the final.
	Round string
	// Number is the number of the problem in its round.
	Number int
	// Label is the mark as the text gives it, with its spaces made single.
	Label string
}

// Year is the year of the final of the olympiad: the first olympiad was held in 1949/50.
func (m Mark) Year() int { return 1949 + m.Olympiad }

// PolishMark finds the mark of a problem of the Polish olympiad at the start of a text. It gives the mark and the text after it.
func PolishMark(text string) (Mark, string, bool) {
	m := polishRE.FindStringSubmatchIndex(text)
	if m == nil {
		return Mark{}, text, false
	}
	n := roman(text[m[2]:m[3]])
	if n == 0 {
		return Mark{}, text, false
	}
	num, _ := strconv.Atoi(text[m[6]:m[7]])
	label := strings.Join(strings.Fields(strings.TrimRight(text[m[0]:m[1]], ".: ")), " ")
	return Mark{Olympiad: n, Round: text[m[4]:m[5]], Number: num, Label: label}, strings.TrimSpace(text[m[1]:]), true
}

// roman gives the value of a Roman numeral, or 0 when it is not one in the usual form.
func roman(s string) int {
	vals := map[byte]int{'I': 1, 'V': 5, 'X': 10, 'L': 50, 'C': 100}
	n := 0
	for i := 0; i < len(s); i++ {
		v := vals[s[i]]
		if i+1 < len(s) && v < vals[s[i+1]] {
			n -= v
		} else {
			n += v
		}
	}
	if toRoman(n) != s {
		return 0
	}
	return n
}

// toRoman writes a number from 1 to 399 in Roman numerals.
func toRoman(n int) string {
	var b strings.Builder
	for _, x := range []struct {
		v int
		s string
	}{{100, "C"}, {90, "XC"}, {50, "L"}, {40, "XL"}, {10, "X"}, {9, "IX"}, {5, "V"}, {4, "IV"}, {1, "I"}} {
		for n >= x.v {
			b.WriteString(x.s)
			n -= x.v
		}
	}
	return b.String()
}
