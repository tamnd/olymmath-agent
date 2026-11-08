package archive

import (
	"fmt"
	"html"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// Issue is one issue of a journal with a problem section.
type Issue struct {
	Year   int
	Volume int
	Number int
	Month  string
	// Page is the web page of the issue that links its PDF. It is "" when the listing links the PDF itself.
	Page string
	PDF  string
}

// Mark is the number of a problem in the PDF of an issue. The journals of this file are in zone Z3, so a lane keeps only the number and the page, and never the text.
type Mark struct {
	// Position is the position of the problem in the locator, for example p5171 or j745.
	Position string
	// Label is the number of the problem as the journal prints it, for example 5171 or J745.
	Label string
	// Page is the page of the PDF where the problem starts, from 1. It is 0 when the page is not known.
	Page int
}

// Journal reads the problem numbers of the issues of a journal.
type Journal struct {
	// Comp is the competition code of the journal in the register.
	Comp string
	// Listing is the page that lists the issues.
	Listing string
	Issues  func(listing string) []Issue
	// PDF finds the link to the PDF on the page of an issue. It is nil when the listing links the PDFs.
	PDF      func(page string) string
	Problems func(text string) []Mark
	// Name names the problem in an issue, for the statement of a record.
	Name func(i Issue, m Mark) string
	// Scanned tells that the PDF of an issue is a scan, so its numbers need the vision lane.
	Scanned func(i Issue) bool
}

// Journals maps a source id of the source register to its journal lane.
var Journals = map[string]Journal{
	"crux":            {Comp: "journal-crux", Listing: CruxListing, Issues: CruxIssues, PDF: CruxPDF, Problems: CruxProblems, Name: cruxName, Scanned: cruxScanned},
	"mathreflections": {Comp: "journal-mr", Listing: MRListing, Issues: MRIssues, Problems: MRProblems, Name: mrName, Scanned: func(Issue) bool { return false }},
}

// CruxListing is the page of the Crux digital archive, with a link to each issue.
const CruxListing = "https://cms.math.ca/publications/crux/"

// CruxLastScan is the last volume of Crux whose PDFs are scans of the paper issues. From volume 22 (1996) the PDFs are born digital.
const CruxLastScan = 21

var (
	// cruxIssueRE finds the link to an issue, for example <a href='/publications/crux/issue?volume=52&issue=8' title='See the Oct 2026 (v52, n8) issue'>.
	cruxIssueRE = regexp.MustCompile(`href='/publications/crux/issue/?\?volume=(\d+)&(?:amp;|#038;)?issue=(\d+)' title='See the (\w+) (\d{4}) \(v\d+, n\d+\) issue'`)
	cruxPDFRE   = regexp.MustCompile(`class='crux-download-link' href='([^']+\.pdf)'`)
	// cruxTOCRE finds the range of the main problems in the contents of an issue, for example "Problems: 5171–5180". The ranges of the other sections have letters, for example OC796–OC800.
	cruxTOCRE = regexp.MustCompile(`Problems:\s*(\d{3,4})\s*[–—-]+\s*(\d{3,4})\b`)
	// cruxHeadRE is the heading of the problem section, and cruxSolRE is the heading of the solution section.
	cruxHeadRE = regexp.MustCompile(`(?m)^\s*PROBLEMS\s*$`)
	cruxSolRE  = regexp.MustCompile(`(?m)^\s*SOLUTIONS\s*$`)
	// cruxNumRE is a problem number at the start of a line, for example "5171. Proposed by".
	cruxNumRE = regexp.MustCompile(`(?m)^\s*(\d{3,4})\s*\*?\.\s`)
)

// CruxIssues gives the issues of the Crux digital archive page, in the order of the page.
func CruxIssues(listing string) []Issue {
	var out []Issue
	seen := map[string]bool{}
	for _, m := range cruxIssueRE.FindAllStringSubmatch(listing, -1) {
		v, _ := strconv.Atoi(m[1])
		n, _ := strconv.Atoi(m[2])
		y, _ := strconv.Atoi(m[4])
		key := m[1] + "/" + m[2]
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Issue{Year: y, Volume: v, Number: n, Month: m[3], Page: fmt.Sprintf("https://cms.math.ca/publications/crux/issue/?volume=%d&issue=%d", v, n)})
	}
	return out
}

// CruxPDF gives the link to the PDF of a Crux issue page, or "".
func CruxPDF(page string) string {
	if m := cruxPDFRE.FindStringSubmatch(page); m != nil {
		return html.UnescapeString(m[1])
	}
	return ""
}

// CruxProblems gives the numbers of the main problems of the text of a Crux issue: the problem section of each issue, and not the solutions, the Olympiad Corner, MathemAttic, Mayhem or Skoliad. The contents of an issue from 2014 give the range of the numbers. An older issue has no range, so the numbers come from the section under the heading PROBLEMS.
func CruxProblems(text string) []Mark {
	pages := strings.Split(text, "\f")
	starts := make([]int, len(pages))
	at := 0
	for i, p := range pages {
		starts[i] = at
		at += len(p) + 1
	}
	pageOf := func(off int) int {
		i, _ := slices.BinarySearch(starts, off+1)
		return i
	}
	// The block from each PROBLEMS heading to the next SOLUTIONS heading. The block with the most numbers is the problem section.
	var best []Mark
	for _, h := range cruxHeadRE.FindAllStringIndex(text, -1) {
		block := text[h[1]:]
		if e := cruxSolRE.FindStringIndex(block); e != nil {
			block = block[:e[0]]
		}
		if e := cruxHeadRE.FindStringIndex(block); e != nil {
			block = block[:e[0]]
		}
		var marks []Mark
		for _, m := range cruxNumRE.FindAllStringSubmatchIndex(block, -1) {
			label := block[m[2]:m[3]]
			marks = append(marks, Mark{Position: "p" + label, Label: label, Page: pageOf(h[1] + m[2])})
		}
		if len(marks) > len(best) {
			best = marks
		}
	}
	best = run(best)
	if m := cruxTOCRE.FindStringSubmatch(text); m != nil {
		lo, _ := strconv.Atoi(m[1])
		hi, _ := strconv.Atoi(m[2])
		if hi >= lo && hi-lo < 40 {
			page := map[string]int{}
			for _, k := range best {
				page[k.Label] = k.Page
			}
			var out []Mark
			for n := lo; n <= hi; n++ {
				label := strconv.Itoa(n)
				out = append(out, Mark{Position: "p" + label, Label: label, Page: page[label]})
			}
			return out
		}
	}
	return best
}

// run keeps the numbers of a problem section that follow the first one: each number once, in order, and within 40 of the first. The French text of a problem repeats its number, and a line of a problem can start with a number of three or four digits.
func run(marks []Mark) []Mark {
	var out []Mark
	seen := map[string]bool{}
	first := -1
	for _, m := range marks {
		n, _ := strconv.Atoi(m.Label)
		if first < 0 {
			first = n
		}
		if n < first || n >= first+40 || seen[m.Label] {
			continue
		}
		seen[m.Label] = true
		out = append(out, m)
	}
	return out
}

func cruxName(i Issue, m Mark) string {
	return fmt.Sprintf("Crux Mathematicorum Vol. %d, No. %d (%s %d), problem %s", i.Volume, i.Number, i.Month, i.Year, m.Label)
}

func cruxScanned(i Issue) bool { return i.Volume <= CruxLastScan }

// MRListing is the archive page of Mathematical Reflections. It links the issues from 2022. The issues of 2006 to 2021 are no longer on the site.
const MRListing = "https://www.awesomemath.org/mathematical-reflections/archives/"

var (
	// mrPDFRE finds the problem column of an issue, for example /wp-pdf-files/math-reflections/mr-2026-05/mr_5_2026_problems_1.pdf.
	mrPDFRE = regexp.MustCompile(`href="(/wp-pdf-files/math-reflections/[^"/]+/mr_(\d)_(\d{4})_problems[^"/]*\.pdf)"`)
	// mrNumRE is a problem number at the start of a line, for example J745. for the junior problems. S, U and O are the senior, undergraduate and olympiad problems.
	mrNumRE = regexp.MustCompile(`(?m)^\s*([JSUO])(\d{1,4})\.\s`)
)

// MRIssues gives the problem column of each issue of the archive page of Mathematical Reflections.
func MRIssues(listing string) []Issue {
	var out []Issue
	seen := map[string]bool{}
	for _, m := range mrPDFRE.FindAllStringSubmatch(listing, -1) {
		n, _ := strconv.Atoi(m[2])
		y, _ := strconv.Atoi(m[3])
		key := m[3] + "/" + m[2]
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, Issue{Year: y, Number: n, PDF: "https://www.awesomemath.org" + m[1]})
	}
	return out
}

// MRProblems gives the numbers of the problems of the problem column of an issue of Mathematical Reflections.
func MRProblems(text string) []Mark {
	var out []Mark
	seen := map[string]bool{}
	for i, page := range strings.Split(text, "\f") {
		for _, m := range mrNumRE.FindAllStringSubmatch(page, -1) {
			label := m[1] + m[2]
			if seen[label] {
				continue
			}
			seen[label] = true
			out = append(out, Mark{Position: strings.ToLower(label), Label: label, Page: i + 1})
		}
	}
	return out
}

func mrName(i Issue, m Mark) string {
	return fmt.Sprintf("Mathematical Reflections %d (%d), problem %s", i.Number, i.Year, m.Label)
}
