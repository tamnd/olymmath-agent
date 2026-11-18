package ocr

import (
	"regexp"
	"slices"
	"strings"
)

// Labeled is one problem of a paper in the order of the paper, with the label that the paper prints before it.
type Labeled struct {
	// Label is the number of the problem as printed, with a letter after it when the paper has one, as 011246A in the papers of the German olympiad or 13 in a paper whose problems are Problem 1 to Problem 30.
	Label string
	// Page is the page where the problem starts, from 1.
	Page int
	Text string
	// Flags are the flags of the checks of the text, with FlagShort for a text that is too short to be a problem and FlagFigure for a text that names a figure.
	Flags []string
}

// Digit gives the last digit of the number of the label, as 6 for 011246A, or -1 when the label has no digit.
func (l Labeled) Digit() int { return lastDigit(l.Label) }

// labelBoldRE is a label in bold at the start of a line, as the prompt asks: **1.**, **Problem 13.**, **Aufgabe 011231:** or **Aufgabe 011135 = 011234:**. The number can have up to seven digits, as the six-digit numbers of the German olympiad, and a letter after it.
var labelBoldRE = regexp.MustCompile(`^\*\*\s*(?:\pL[\pL ]{0,23}\s)?([0-9]{1,7}[A-Za-z]?)(?:\s*=\s*[0-9]{1,7}[A-Za-z]?)?\s*[.:)]?-?\s*\*\*\s*[.:]?\s*`)

// labelWordRE is a label with a word and no bold, as Problem 13 or Aufgabe 011231:.
var labelWordRE = regexp.MustCompile(`^(?:` + words + `)\s+([0-9]{1,7}[A-Za-z]?)(?:\s*[.:)]\s*|\s+|\s*$)`)

// labelBareRE is a bare number at the start of a line: 1. or 1.- as in the Spanish papers.
var labelBareRE = regexp.MustCompile(`^([0-9]{1,2})[.)]-?\s+`)

// figureWords are words that name a figure, in the languages of the papers. A text that names a figure needs it, even when the model did not mark the figure with [figure].
var figureWords = regexp.MustCompile(`(?i)\b(?:figure|figures|diagram|picture|abbildung|abb\.|skizze|zeichnung|figur|slika|slici|slike|slikom|crtež|crtežu|figura|figurii|desen|desenul|obrázok|obrázku|obrázek|rysunek|rysunku|ábra|ábrán|рисун\pL*|рис\.)`)

// Labels cuts the text of the pages into problems in the order of the paper. A problem starts at its label and ends at the next label, at a heading, or at a solution, as in Split. Unlike Split it keeps each problem, also a number that comes again, as in a paper with one section for each grade that numbers each section from 1, and it reads the long labels of some papers. When the text has labels in bold, only those count.
func Labels(pages []Page) []Labeled {
	re := pickLabel(pages)
	if re == nil {
		return nil
	}
	var out []Labeled
	var cur *Labeled
	var body []string
	end := func() {
		if cur != nil {
			cur.Text = strings.TrimSpace(strings.Join(trimRules(body), "\n"))
			cur.Flags = Check(cur.Text)
			if len([]rune(cur.Text)) < 20 {
				cur.Flags = append(cur.Flags, FlagShort)
			}
			if figureWords.MatchString(cur.Text) && !slices.Contains(cur.Flags, FlagFigure) {
				cur.Flags = append(cur.Flags, FlagFigure)
			}
			out = append(out, *cur)
		}
		cur, body = nil, nil
	}
	for _, p := range pages {
		for _, line := range strings.Split(p.Text, "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "#") || endRE.MatchString(t) {
				end()
				continue
			}
			if m := re.FindStringSubmatchIndex(t); m != nil {
				end()
				cur = &Labeled{Label: t[m[2]:m[3]], Page: p.N}
				body = []string{t[m[1]:]}
				continue
			}
			if cur != nil {
				body = append(body, line)
			}
		}
	}
	end()
	return out
}

// pickLabel gives the marker of the labels of the text: labels in bold when the text has them, or else labels with a word, or else bare numbers.
func pickLabel(pages []Page) *regexp.Regexp {
	for _, re := range []*regexp.Regexp{labelBoldRE, labelWordRE, labelBareRE} {
		for _, p := range pages {
			for _, line := range strings.Split(p.Text, "\n") {
				if re.MatchString(strings.TrimSpace(line)) {
					return re
				}
			}
		}
	}
	return nil
}

// lastDigit gives the last digit of the first run of digits of s, or -1 when s has no digit.
func lastDigit(s string) int {
	d := -1
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			d = int(r - '0')
		case d >= 0:
			return d
		}
	}
	return d
}

// PositionDigit gives the last digit of the number of a position, as 6 for p126a or 2 for a12, or -1 when the position has no digit.
func PositionDigit(pos string) int {
	return lastDigit(strings.TrimRight(pos, "abcdefghijklmnopqrstuvwxyz"))
}
