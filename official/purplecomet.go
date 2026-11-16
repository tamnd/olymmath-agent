package official

import (
	"html"
	"regexp"
	"strconv"
	"strings"
)

// PurpleCometFirst and PurpleCometLast are the first and the last year of the papers that the answers page of Purple Comet links, as read on 2026-11-16. The pages of 2003 and 2004 have no papers.
const (
	PurpleCometFirst = 2005
	PurpleCometLast  = 2026
)

// PurpleCometListings are the contest pages of the Purple Comet! Math Meet, one for the middle school paper and one for the high school paper of each year, such as https://purplecomet.org/contest/2019/HS. Each page shows the PDF of its paper in a frame.
var PurpleCometListings = purpleCometListings(PurpleCometFirst, PurpleCometLast)

func purpleCometListings(from, to int) []string {
	var out []string
	for y := from; y <= to; y++ {
		for _, l := range []string{"MS", "HS"} {
			out = append(out, "https://purplecomet.org/contest/"+strconv.Itoa(y)+"/"+l)
		}
	}
	return out
}

// purpleCometRE matches the heading of a contest page and the frame of its PDF.
var purpleCometRE = regexp.MustCompile(`(?s)<h3[^>]*>\s*((?:19|20)[0-9]{2}) (HS|MS) Contest Questions\s*</h3>\s*<iframe src="(https://purplecomet\.org/files/[^"]+\.pdf)"`)

// PurpleCometPapers gives the papers of Purple Comet (purple-comet) on its contest pages, which om papers joins into one text. The middle school paper is ms and the high school paper hs, as in the layout of the competition.
func PurpleCometPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range purpleCometRE.FindAllStringSubmatch(page, -1) {
		u := html.UnescapeString(m[3])
		if seen[u] {
			continue
		}
		seen[u] = true
		y, _ := strconv.Atoi(m[1])
		out = append(out, National{Competition: "purple-comet", Year: y, Paper: strings.ToLower(m[2]), URL: u})
	}
	return out
}
