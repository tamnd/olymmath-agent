package archive

import (
	"fmt"
	"strings"
	"testing"
)

func positions(marks []Mark) string {
	var out []string
	for _, m := range marks {
		out = append(out, fmt.Sprintf("%s@%d", m.Position, m.Page))
	}
	return strings.Join(out, " ")
}

func TestCruxIssues(t *testing.T) {
	listing := `<li><a href='/publications/crux/issue?volume=52&issue=8' title='See the Oct 2026 (v52, n8) issue'>Oct</a></li>
<li><a href='/publications/crux/issue?volume=52&amp;issue=8' title='See the Oct 2026 (v52, n8) issue'>again</a></li>
<li><a href='/publications/crux/issue/?volume=1&#038;issue=10' title='See the Dec 1975 (v1, n10) issue'>Dec</a></li>
<li><a href='/publications/crux/other'>other</a></li>`
	is := CruxIssues(listing)
	if len(is) != 2 {
		t.Fatalf("issues %+v", is)
	}
	if is[0].Year != 2026 || is[0].Volume != 52 || is[0].Number != 8 || is[0].Month != "Oct" || is[0].Page != "https://cms.math.ca/publications/crux/issue/?volume=52&issue=8" {
		t.Fatalf("first issue %+v", is[0])
	}
	if is[1].Year != 1975 || is[1].Volume != 1 || is[1].Number != 10 {
		t.Fatalf("second issue %+v", is[1])
	}
	if !cruxScanned(is[1]) || cruxScanned(is[0]) {
		t.Fatal("only the old volumes are scans")
	}
}

func TestCruxPDF(t *testing.T) {
	page := `<a class='crux-download-link' href='https://cms.math.ca/wp-content/uploads/2026/10/Whole_52_8.pdf?a=1&amp;b=2.pdf'>Download</a>`
	if got := CruxPDF(page); got != "https://cms.math.ca/wp-content/uploads/2026/10/Whole_52_8.pdf?a=1&b=2.pdf" {
		t.Fatalf("pdf %q", got)
	}
	if got := CruxPDF("<p>no link</p>"); got != "" {
		t.Fatalf("pdf %q", got)
	}
}

func TestCruxProblemsContents(t *testing.T) {
	// The contents give the range, and the section gives the page of each number it finds.
	text := "Contents\n  Olympiad Corner: OC11–OC15\n  Problems: 4401–4404\n\f" +
		"   PROBLEMS\n4401. Proposed by A. Find x.\n4401. Proposé par A. Trouver x.\n\f" +
		"4402. Proposed by B. Show that\n1234. is not a number of the list.\n4404 *. Proposed by C.\n\f" +
		"   SOLUTIONS\n4301. A solution.\n"
	if got := positions(CruxProblems(text)); got != "p4401@2 p4402@3 p4403@0 p4404@3" {
		t.Fatalf("marks %s", got)
	}
}

func TestCruxProblemsSection(t *testing.T) {
	// An old issue has no range in its contents, so the numbers come from the longest problem section.
	text := "   PROBLEMS\n2001. Notes.\n\f" +
		"   PROBLEMS\n2101. Proposed by A.\n2102. Proposed by B.\n\f2103. Proposed by C.\n150. Not a number.\n" +
		"   SOLUTIONS\n2010. A solution.\n"
	if got := positions(CruxProblems(text)); got != "p2101@2 p2102@2 p2103@3" {
		t.Fatalf("marks %s", got)
	}
	if got := CruxProblems("No problem here."); len(got) != 0 {
		t.Fatalf("marks %+v", got)
	}
}

func TestMRIssues(t *testing.T) {
	listing := `<a href="/wp-pdf-files/math-reflections/mr-2026-05/mr_5_2026_problems_1.pdf">Problems</a>
<a href="/wp-pdf-files/math-reflections/mr-2026-05/mr_5_2026_problems_1.pdf">again</a>
<a href="/wp-pdf-files/math-reflections/mr-2026-05/mr_5_2026_solutions.pdf">Solutions</a>
<a href="/wp-pdf-files/math-reflections/mr-2022-01/mr_1_2022_problems.pdf">Problems</a>`
	is := MRIssues(listing)
	if len(is) != 2 {
		t.Fatalf("issues %+v", is)
	}
	if is[0].Year != 2026 || is[0].Number != 5 || is[0].PDF != "https://www.awesomemath.org/wp-pdf-files/math-reflections/mr-2026-05/mr_5_2026_problems_1.pdf" {
		t.Fatalf("first issue %+v", is[0])
	}
	if is[1].Year != 2022 || is[1].Number != 1 {
		t.Fatalf("second issue %+v", is[1])
	}
}

func TestMRProblems(t *testing.T) {
	text := "Junior problems\nJ745. Proposed by A.\nJ746. Proposed by B.\n\fSenior problems\nS745. Proposed by C.\n  O750. Proposed by D.\nJ745. A repeat.\nX1. Not a problem.\n"
	if got := positions(MRProblems(text)); got != "j745@1 j746@1 s745@2 o750@2" {
		t.Fatalf("marks %s", got)
	}
	m := MRProblems(text)[0]
	if got := mrName(Issue{Year: 2026, Number: 5}, m); got != "Mathematical Reflections 5 (2026), problem J745" {
		t.Fatalf("name %q", got)
	}
}

func TestJournals(t *testing.T) {
	for id, j := range Journals {
		if j.Comp == "" || j.Listing == "" || j.Issues == nil || j.Problems == nil || j.Name == nil || j.Scanned == nil {
			t.Fatalf("%s is not complete", id)
		}
	}
	if got := cruxName(Issue{Year: 2026, Volume: 52, Number: 8, Month: "Oct"}, Mark{Label: "5171"}); got != "Crux Mathematicorum Vol. 52, No. 8 (Oct 2026), problem 5171" {
		t.Fatalf("name %q", got)
	}
}
