package ocr

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Problem is the text of one problem of a paper.
type Problem struct {
	// Number is the number of the problem as printed.
	Number int
	// Page is the page where the problem starts, from 1.
	Page int
	Text string
	// Flags are the flags of the checks of the text.
	Flags []string
}

// words are the words that can come before the number of a problem, in the languages of the papers.
const words = `Problem|Problema|Problème|Probleme|Aufgabe|Zadanie|Zadatak|Задатак|Opgave|Uppgift|Oppgave|Úloha|Uloha|Задача|Feladat|Exercice|Question|Bài|Soru`

// boldRE is a problem number in bold at the start of a line, as the prompt asks: **1.** or **Problem 1.** The label before the number can be any words, as in **IMO TST 4.**, because the number is in bold and the bold ends after it.
var boldRE = regexp.MustCompile(`^\*\*\s*(?:\pL[\pL ]{0,23}\s)?([0-9]{1,2})\s*[.:)]?\s*\*\*\s*[.:]?\s*`)

// wordRE is a problem number with a word and no bold: Problem 1. or Zadanie 1
var wordRE = regexp.MustCompile(`^(?:` + words + `)\s+([0-9]{1,2})(?:\s*[.:)]\s*|\s*$)`)

// bareRE is a bare number at the start of a line: 1.
var bareRE = regexp.MustCompile(`^([0-9]{1,2})[.)]\s+`)

// endRE is a line that ends the problem before it: a solution, an answer, a hint or a note to the marker. The prompt asks for a heading there, and this catches the answers that do not obey.
var endRE = regexp.MustCompile(`(?i)^(?:\*\*|\*|_)?\s*(?:solution|answer|hint|rozwiązanie|odpowiedź|lösung|antwort|řešení|riešenie|решение|ответ|rješenje|решење|megoldás|soluzione|solución|solução|lời giải|çözüm)\b`)

var ruleRE = regexp.MustCompile(`^(?:-{3,}|\*{3,}|_{3,})$`)

// Split cuts the text of the pages into problems. A problem starts at its number and ends at the next number, at a heading, or at a solution. When the text has numbers in bold, only those count, so a list in the text of a problem does not start a problem. When a number comes twice, as in a paper whose solutions follow the problems, the first one wins.
func Split(pages []Page) []Problem { return split(pages, false) }

func split(pages []Page, sequence bool) []Problem {
	re := pick(pages)
	if re == nil {
		return nil
	}
	var out []Problem
	var cur *Problem
	var body []string
	seen := map[int]bool{}
	end := func() {
		if cur != nil {
			cur.Text = strings.TrimSpace(strings.Join(trimRules(body), "\n"))
			cur.Flags = Check(cur.Text)
			if len([]rune(cur.Text)) < 20 {
				cur.Flags = append(cur.Flags, FlagShort)
			}
			out = append(out, *cur)
		}
		cur, body = nil, nil
	}
	for _, p := range pages {
		for _, line := range strings.Split(p.Text, "\n") {
			t := strings.TrimSpace(line)
			switch {
			case strings.HasPrefix(t, "#"), endRE.MatchString(t):
				end()
				continue
			}
			if m := re.FindStringSubmatchIndex(t); m != nil {
				n, _ := strconv.Atoi(t[m[2]:m[3]])
				end()
				if sequence {
					n = len(out) + 1
				}
				if n > 0 && !seen[n] {
					seen[n] = true
					cur = &Problem{Number: n, Page: p.N}
					body = []string{t[m[1]:]}
				}
				continue
			}
			if cur != nil {
				body = append(body, line)
			}
		}
	}
	end()
	slices.SortStableFunc(out, func(a, b Problem) int { return a.Number - b.Number })
	return out
}

// pick gives the marker of the text: bold numbers when the text has them, or else numbers with a word, or else bare numbers.
func pick(pages []Page) *regexp.Regexp {
	for _, re := range []*regexp.Regexp{boldRE, wordRE, bareRE} {
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

// trimRules drops the blank lines and the horizontal rules at the end of a problem.
func trimRules(lines []string) []string {
	for len(lines) > 0 {
		t := strings.TrimSpace(lines[len(lines)-1])
		if t != "" && !ruleRE.MatchString(t) {
			break
		}
		lines = lines[:len(lines)-1]
	}
	return lines
}

// Sequence numbers the problems 1, 2, 3 in the order of the paper. It is for a paper whose days each start at problem 1, where Split keeps only the first day. It keeps each problem, so it is not for a paper whose solutions number the problems again.
func Sequence(pages []Page) []Problem { return split(pages, true) }
