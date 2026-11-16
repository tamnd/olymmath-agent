package official

import (
	"html"
	"regexp"
	"strconv"
)

// NZMOListing is the resources page of the New Zealand Mathematical Olympiad Committee (NZMOC). It links the problems and the solutions of both rounds of the NZMO of each year from 2019. Its robots.txt asks for a crawl delay of 20 seconds.
const NZMOListing = "https://www.mathsolympiad.org.nz/resources/"

// nzmoRE matches a link to the problems of one round of one year. A link with &type=solutions is the file of the solutions, and the expression does not match it.
var nzmoRE = regexp.MustCompile(`href="(?:https:)?(//www\.mathsolympiad\.org\.nz/competitions/nzmo/files/problems\.php\?round=([12])&(?:amp;)?year=((?:19|20)[0-9]{2}))"`)

// NZMOPapers gives the papers of the NZMO (nz-nzmo) on the resources page of the NZMOC. Round One is the paper round1 with eight problems (nine from 2026), and Round Two the paper round2 with five problems, which the layout numbers after those of Round One.
func NZMOPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range nzmoRE.FindAllStringSubmatch(page, -1) {
		u := "https:" + html.UnescapeString(m[1])
		if seen[u] {
			continue
		}
		seen[u] = true
		y, _ := strconv.Atoi(m[3])
		out = append(out, National{Competition: "nz-nzmo", Year: y, Paper: "round" + m[2], URL: u})
	}
	return out
}

func init() {
	// Each paper starts with a page of instructions, whose numbered rules are not problems. The page of the problems has the heading "NZMO Round One 2025 — Problems". The instructions of Round Two 2019 have no such heading, and they end with the rule "Do not turn over until told to do so."
	Readings["nzmoc"] = Reading{Cover: regexp.MustCompile(`(?s)\A(?:.*NZMO Round (?:One|Two) [0-9]{4}\s*(?:—|-)\s*Instructions[^\n]*|.*?Do not turn over until told to do so\.[^\n]*)`)}
}
