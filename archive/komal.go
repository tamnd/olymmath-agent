package archive

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

// KomalFirst is the first school year of the English month pages of KöMaL with open texts. The pages of the years before have the problems in English from 1984, but only for subscribers.
const KomalFirst = 2004

// komalMonths are the months of an issue: September to December of a school year, then January to May.
var komalMonths = []int{9, 10, 11, 12, 1, 2, 3, 4, 5}

// Komal gives the English month pages of KöMaL for the school years that start from the year from to the year to. The year of a page is the calendar year of its month.
func Komal(from, to int) []Doc {
	var out []Doc
	for y := max(from, KomalFirst); y <= to; y++ {
		for _, m := range komalMonths {
			year := y
			if m < 9 {
				year++
			}
			out = append(out, Doc{Year: year, Label: "KöMaL Problem Contests", URL: fmt.Sprintf("https://www.komal.hu/feladat?a=honap&l=en&h=%d%02d&t=%d%02d", year, m, year, m)})
		}
	}
	return out
}

var (
	// komalStart finds a mathematics problem of a month page, for example <p id="B3802" ...>. The series P, M and G are physics, and I and S are informatics.
	komalStart = regexp.MustCompile(`(?i)<p id="([ABCK])(\d+)"[^>]*>\s*<b>[^<]*</b>`)
	// komalEnd is the end of a statement: the line of the proposer, the points or the rule between problems.
	komalEnd = regexp.MustCompile(`(?i)<p align="?(?:right|left)"?>|<hr|<p[^>]*>\s*\(?\s*Proposed by`)
	// komalItalic is a variable in italics, for example <I>x</I> or <I>AB</I>. A longer text in italics is a word.
	komalItalic = regexp.MustCompile(`(?i)<i>\s*([a-z]{1,2}'*|[a-z][0-9]{1,2})\s*</i>`)
	komalImage  = regexp.MustCompile(`(?i)<img[^>]*>`)
	komalAlt    = regexp.MustCompile(`(?i)\balt="([^"]*)"`)
	komalTeX    = regexp.MustCompile(`(?s)\\\(\s*(?:\\displaystyle)?\s*(.*?)\s*\\\)`)
	komalShow   = regexp.MustCompile(`(?s)\\\[\s*(?:\\displaystyle)?\s*(.*?)\s*\\\]`)
)

var (
	// errKomalFigure tells that a problem has a picture with no TeX in it.
	errKomalFigure = errors.New("the problem has a figure")
	// errKomalLocked tells that the text of a problem is only for subscribers.
	errKomalLocked = errors.New("the text is only for subscribers")
)

// ParseKomal reads the mathematics problems of one English month page of KöMaL. The number of a problem is its sign, for example B3802.
func ParseKomal(page string) []Problem {
	var out []Problem
	for _, m := range komalStart.FindAllStringSubmatchIndex(page, -1) {
		body := page[m[1]:]
		if e := komalEnd.FindStringIndex(body); e != nil {
			body = body[:e[0]]
		}
		p := Problem{Number: strings.ToUpper(page[m[2]:m[3]]) + page[m[4]:m[5]]}
		figure := false
		// A formula of an old page is a picture with its TeX in the alt text.
		body = komalImage.ReplaceAllStringFunc(body, func(img string) string {
			a := komalAlt.FindStringSubmatch(img)
			if a == nil || strings.TrimSpace(a[1]) == "" {
				figure = true
				return ""
			}
			return "\\(" + a[1] + "\\)"
		})
		if figure {
			p.Err = errKomalFigure
			out = append(out, p)
			continue
		}
		body = komalItalic.ReplaceAllString(body, "\x02$1\x02")
		body = komalShow.ReplaceAllString(body, "$$$$$1$$$$")
		body = komalTeX.ReplaceAllString(body, "$$$1$$")
		if p.Text = htmlText(body); strings.HasPrefix(p.Text, "Subscribers can reach the text") {
			p.Text, p.Err = "", errKomalLocked
		}
		out = append(out, p)
	}
	return out
}
