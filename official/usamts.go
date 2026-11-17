package official

import (
	"regexp"
	"strconv"
)

// USAMTSListings are the pages of www.usamts.org that om papers reads: the archive of past rounds, which links the problems, the solutions and the rubric of each round from Year 10 (1998-1999).
var USAMTSListings = []string{"https://www.usamts.org/contest/past-problems"}

// usamtsRE matches the link to the problems of one round on the archive page: https://files.usamts.org/Problems_37_1.pdf is round 1 of Year 37.
var usamtsRE = regexp.MustCompile(`href="(https://files\.usamts\.org/Problems_([0-9]{1,2})_([1-9])\.pdf)"`)

// USAMTSPapers gives the round papers on the archive page of the USAMTS. Year N starts in the fall of 1988 + N, and its edition is that year, so round 1 of Year 37 is the paper round1 of 2025. Year 22 (2010) is left out: its third round is missing from the archive, so the register gives 2010 no shape.
func USAMTSPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range usamtsRE.FindAllStringSubmatch(page, -1) {
		n, _ := strconv.Atoi(m[2])
		if n == 22 || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		out = append(out, National{Competition: "usamts", Year: 1988 + n, Paper: "round" + m[3], URL: m[1]})
	}
	return out
}
