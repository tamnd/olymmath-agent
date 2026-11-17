package official

import (
	"regexp"
	"strings"
)

// hrMainSize is the size of the most text of the runs whose font math does not match, the larger of two sizes with as much.
func hrMainSize(items []PDFItem, math *regexp.Regexp) int {
	count := map[int]int{}
	for _, it := range items {
		if !math.MatchString(it.Family) {
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

// HrAccentsIt are the letters with the accents of Italian that the fonts of Computer Modern set as the accent and the letter, as "`e" for "è" and "cos`ı" for "così", with the dotless i.
var HrAccentsIt = map[string]string{"`a": "à", "`e": "è", "`i": "ì", "`ı": "ì", "`o": "ò", "`u": "ù", "`A": "À", "`E": "È", "`I": "Ì", "`O": "Ò", "`U": "Ù", "´a": "á", "´e": "é", "´i": "í", "´ı": "í", "´o": "ó", "´u": "ú", "´E": "É"}

// hrAccentItRE is an accent of Computer Modern with the letter under it, perhaps after a space.
var hrAccentItRE = regexp.MustCompile("([`´]) ?([aeiouAEIOUı])")

// hrJoinLetters joins an item that ends with an accent to the next item of its line that starts with a letter under it, as "pi`" and "u", and puts the letters with their accents of accents in the text of each item.
func hrJoinLetters(items []PDFItem, accents map[string]string) []PDFItem {
	var out []PDFItem
	for _, it := range items {
		if n := len(out); n > 0 {
			prev := &out[n-1]
			if (strings.HasSuffix(prev.Text, "`") || strings.HasSuffix(prev.Text, "´")) && prev.Page == it.Page && pxAbs(float64(prev.Top-it.Top)) <= 3 && it.Left <= prev.Right()+2 && it.Left >= prev.Left && strings.IndexAny(it.Text, "aeiouAEIOUı") == 0 {
				prev.Text += it.Text
				prev.Width = max(prev.Width, it.Right()-prev.Left)
				continue
			}
		}
		out = append(out, it)
	}
	for i := range out {
		out[i].Text = hrAccentItRE.ReplaceAllStringFunc(out[i].Text, func(m string) string {
			if a, ok := accents[strings.Replace(m, " ", "", 1)]; ok {
				return a
			}
			return m
		})
	}
	return out
}
