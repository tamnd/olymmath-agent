package official

import (
	"regexp"
	"strconv"
)

// BAMOListing is the page of www.bamo.org that om papers reads: the archive of the Bay Area Mathematical Olympiad, which links one file of the problems and the solutions of each year from 1999.
const BAMOListing = "https://www.bamo.org/archives/problems_and_solutions"

// bamoRE matches the link to the file of one year on the archive page: /archives/examfiles/bamo2020examsol.pdf, /archives/examfiles/BAMO2016ProblemsAndSolutions.pdf or /archives/examfiles/bamo99draft.pdf for 1999.
var bamoRE = regexp.MustCompile(`href="(/archives/examfiles/(?i:bamo)((?:19|20)[0-9]{2}|[0-9]{2})[^"/]*\.pdf)"`)

// BAMOPapers gives the file of each year on the archive page of the BAMO. A file holds the problems of BAMO-8 and of BAMO-12 of its year, so it holds all the positions of the edition. A year of two digits is of the 1900s when it is 99 and of the 2000s otherwise.
func BAMOPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range bamoRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[2])
		switch {
		case y == 99:
			y = 1999
		case y < 100:
			y += 2000
		}
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, National{Competition: "bamo", Year: y, URL: "https://www.bamo.org" + m[1]})
	}
	return out
}
