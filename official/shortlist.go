package official

import (
	"fmt"
	"regexp"
	"strings"
)

// ShortlistFirst is the first year with a shortlist booklet on imo-official.org.
const ShortlistFirst = 2006

// Shortlist gives the shortlist booklets of the IMO from one year to another. A booklet has the problems and the official solutions, and it is published after the IMO of the next year.
func Shortlist(from, to int) []Paper {
	var out []Paper
	for y := max(from, ShortlistFirst); y <= to; y++ {
		out = append(out, Paper{
			Edition: fmt.Sprintf("isl/%d", y),
			URL:     fmt.Sprintf("https://www.imo-official.org/assets/documents/problems/%d/IMO%dSL.pdf", y, y),
			Lang:    "en",
			Name:    fmt.Sprintf("IMO %d Shortlist booklet", y),
			Booklet: true,
		})
	}
	return out
}

var (
	// labelRE finds the start of a problem of a booklet: "A1." or "N 7." at the start of a line, a line such as "A1 CZE (Czech Republic)" in the booklet of 2009, or a line with only the label in the booklet of 2011.
	labelRE = regexp.MustCompile(`(?m)^[ \t]*([ACGN])[ \t]?([0-9]{1,2})(?:\.|[ \t]+[A-Z]{3}[ \t]+\([^)\n]*\)[ \t]*$|[ \t]*$)`)
	// solutionRE finds the end of a statement in a booklet: the line where the solution or the answer starts.
	solutionRE = regexp.MustCompile(`(?m)^[ \t]*(Solution|Answer|Comment|Remark)s?\b`)
	// countryRE finds the country that proposed the problem, at the end of a statement.
	countryRE = regexp.MustCompile(`\(\s*[A-Z][A-Za-z.' -]*\)\s*$`)
	digitsRE  = regexp.MustCompile(`[0-9\s]+`)
	// sectionRE matches the headings of the parts of a booklet.
	sectionRE = regexp.MustCompile(`(?m)^[ \t]*(Problems|Solutions|Algebra|Combinatorics|Geometry|Number Theory)[ \t]*$`)
)

// SplitBooklet cuts the text of a shortlist booklet into problems. The text of a problem ends where its solution starts. A booklet that gives all the problems first and then the solutions has each label twice, and the first text is kept. The lines that each page repeats, such as the running head with its page number, are removed.
func SplitBooklet(text string) []Segment {
	// A form feed starts a line too, so that a label at the top of a page is found.
	text = strings.ReplaceAll(furniture(text), "\f", "\f\n")
	ms := labelRE.FindAllStringSubmatchIndex(text, -1)
	var out []Segment
	seen := map[string]bool{}
	for i, m := range ms {
		n := strings.TrimLeft(text[m[4]:m[5]], "0")
		pos := strings.ToLower(text[m[2]:m[3]]) + n
		if n == "" || seen[pos] {
			continue
		}
		end := len(text)
		if i+1 < len(ms) {
			end = ms[i+1][0]
		}
		body := text[m[1]:end]
		if l := solutionRE.FindStringIndex(body); l != nil {
			body = body[:l[0]]
		}
		body = countryRE.ReplaceAllString(clean(body), "")
		if strings.TrimSpace(body) == "" {
			continue
		}
		seen[pos] = true
		out = append(out, Segment{Position: pos, Page: strings.Count(text[:m[0]], "\f") + 1, Text: strings.TrimSpace(body)})
	}
	return out
}

// Repeats is the number of pages on which a line must occur to be a running head or a running foot.
const Repeats = 5

// furniture removes the running heads and feet of a text: the lines that occur on Repeats pages or more when their digits are removed, a page number on the first or the last line of a page, and the headings of the parts. The form feeds stay.
func furniture(text string) string {
	pages := strings.Split(text, "\f")
	key := func(l string) string { return digitsRE.ReplaceAllString(l, "") }
	count := map[string]int{}
	for _, pg := range pages {
		seen := map[string]bool{}
		for _, l := range strings.Split(pg, "\n") {
			if k := key(l); len(k) >= 8 && !seen[k] {
				seen[k] = true
				count[k]++
			}
		}
	}
	for i, pg := range pages {
		var keep []string
		for _, l := range strings.Split(pg, "\n") {
			if count[key(l)] < Repeats || len(key(l)) < 8 {
				keep = append(keep, l)
			}
		}
		// A line that is only a number is a page number when it is the first or the last line of the page.
		for _, end := range []int{0, 1} {
			for k := range keep {
				j := k
				if end == 1 {
					j = len(keep) - 1 - k
				}
				if t := strings.TrimSpace(keep[j]); t != "" {
					if key(t) == "" {
						keep = append(keep[:j], keep[j+1:]...)
					}
					break
				}
			}
		}
		pages[i] = strings.Join(keep, "\n")
	}
	return sectionRE.ReplaceAllString(strings.Join(pages, "\f"), "")
}

// SolutionPages gives, for each problem of a booklet, the page where its solution starts, from 1. The solution part of a booklet repeats the label and the text of each problem and then gives the solution, so the page is the page of the first label of the problem that a solution follows. A later label is often a point of a figure, for example A1.
func SolutionPages(text string) map[string]int {
	text = strings.ReplaceAll(furniture(text), "\f", "\f\n")
	ms := labelRE.FindAllStringSubmatchIndex(text, -1)
	out := map[string]int{}
	for i, m := range ms {
		n := strings.TrimLeft(text[m[4]:m[5]], "0")
		if n == "" {
			continue
		}
		end := len(text)
		if i+1 < len(ms) {
			end = ms[i+1][0]
		}
		pos := strings.ToLower(text[m[2]:m[3]]) + n
		if _, ok := out[pos]; !ok && solutionRE.MatchString(text[m[1]:end]) {
			out[pos] = strings.Count(text[:m[0]], "\f") + 1
		}
	}
	return out
}
