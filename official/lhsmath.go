package official

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// LHSMathListings are the pages of the Lexington Math Tournament (LMT) archive of www.lhsmath.org that om papers reads: the page of each spring tournament from 2010 to 2017, and the page of the Girls LMT of 2025. Each page links the problems and the solutions of each round of its tournament. The tournaments from 2018, when the fall tournament started, change their rounds between the spring and the fall, and the register gives one shape a year, so their pages are not read.
var LHSMathListings = lhsmathListings()

func lhsmathListings() []string {
	var out []string
	for y := 2010; y <= 2017; y++ {
		out = append(out, fmt.Sprintf("https://www.lhsmath.org/LMT/%d_Problems_and_Solutions", y))
	}
	return append(out, "https://www.lhsmath.org/LMT/2025_GLMT_Problems_and_Solutions")
}

// lhsmathRE matches the link to the file of one round on a page of the archive: Archive/2017/problems/Team_Round_2017_prob.pdf, Archive/2016/problems/2016 Guts Problems.pdf or Archive/2025_GLMT/GLMT 2025 Team Round.pdf.
var lhsmathRE = regexp.MustCompile(`href="(Archive/((?:19|20)[0-9]{2})(_GLMT)?/[^"]+\.pdf)"`)

// lhsmathRounds gives the paper of the layout for a word of the name of a round file.
var lhsmathRounds = []struct{ word, paper string }{
	{"indiv", "individual"},
	{"theme", "theme"},
	{"team", "team"},
	{"guts", "guts"},
	{"scramble", "scramble"},
}

// LHSMathPapers gives the round papers of the LMT (lmt) of 2010 to 2017 and of the Girls LMT (glmt) on the archive pages of www.lhsmath.org. The files of the solutions and the answers are not papers.
func LHSMathPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range lhsmathRE.FindAllStringSubmatch(page, -1) {
		name := strings.ToLower(m[1][strings.LastIndex(m[1], "/")+1:])
		if strings.Contains(name, "sol") || strings.Contains(name, "answer") {
			continue
		}
		paper := ""
		for _, r := range lhsmathRounds {
			if strings.Contains(name, r.word) {
				paper = r.paper
				break
			}
		}
		y, _ := strconv.Atoi(m[2])
		if paper == "" || (m[3] == "" && y > 2017) {
			continue
		}
		u := "https://www.lhsmath.org/LMT/" + (&url.URL{Path: m[1]}).EscapedPath()
		if seen[u] {
			continue
		}
		seen[u] = true
		comp := "lmt"
		if m[3] != "" {
			comp = "glmt"
		}
		out = append(out, National{Competition: comp, Year: y, Paper: paper, URL: u})
	}
	return out
}

// LHSMathEdition gives the edition of a paper of LHSMathPapers: the spring of its year for the LMT, as 2017-spring, because the LMT has the edition form year-season, and the year for the Girls LMT.
func LHSMathEdition(n National) string {
	if n.Competition == "lmt" {
		return fmt.Sprintf("%d-spring", n.Year)
	}
	return strconv.Itoa(n.Year)
}
