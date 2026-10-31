package normalize

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
)

// MaxShare is the default share of characters that a fix of an extraction error can change (spec 06.4 step 6).
const MaxShare = 0.05

// Check is the result of the no-rewrite guard.
type Check struct {
	// Distance is the edit distance in characters between the input and the output.
	Distance int
	// Share is Distance divided by the length of the longer text.
	Share float64
	// Flags lists each number, variable or operator that the output adds or removes. A person reviews each flag.
	Flags []string
}

// OK reports whether the output passes: the share is at most max and no flag is set.
func (c Check) OK(maxShare float64) bool { return c.Share <= maxShare && len(c.Flags) == 0 }

// Guard compares a raw extraction with a fixed text. A model may fix extraction errors, but it may not rewrite: the edit distance must stay small, and each change to a number, a variable or an operator is flagged.
func Guard(raw, fixed string) Check {
	a, b := []rune(raw), []rune(fixed)
	c := Check{Distance: distance(a, b)}
	if n := max(len(a), len(b)); n > 0 {
		c.Share = float64(c.Distance) / float64(n)
	}
	ma, mb := mathAtoms(raw), mathAtoms(fixed)
	for _, k := range sortedKeys(ma, mb) {
		switch d := mb[k] - ma[k]; {
		case d > 0:
			c.Flags = append(c.Flags, fmt.Sprintf("adds %q", k))
		case d < 0:
			c.Flags = append(c.Flags, fmt.Sprintf("removes %q", k))
		}
	}
	return c
}

// mathAtoms counts the numbers, the one-letter variables and the operators of s.
func mathAtoms(s string) map[string]int {
	m := map[string]int{}
	for _, t := range Tokens(s) {
		r := []rune(t)
		switch {
		case unicode.IsDigit(r[0]):
			m[t]++
		case len(r) == 1 && unicode.IsLetter(r[0]):
			m[t]++
		case len(r) == 1 && strings.ContainsRune("+-*/=<>^_|!", r[0]):
			m[t]++
		}
	}
	return m
}

func sortedKeys(a, b map[string]int) []string {
	var ks []string
	for k := range a {
		ks = append(ks, k)
	}
	for k := range b {
		if _, ok := a[k]; !ok {
			ks = append(ks, k)
		}
	}
	slices.Sort(ks)
	return ks
}

// distance is the Levenshtein distance with two rows of memory.
func distance(a, b []rune) int {
	prev := make([]int, len(b)+1)
	cur := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev, cur = cur, prev
	}
	return prev[len(b)]
}
