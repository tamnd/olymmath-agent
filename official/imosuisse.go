package official

import (
	"regexp"
	"strconv"
)

// ImosuisseListing is the page of the exams of the archive of the Swiss Mathematical Olympiad (source imosuisse), with a link to the paper of each exam in each of its languages.
const ImosuisseListing = "https://imosuisse.github.io/exams/"

// imosuisseRE matches the link to the paper of one exam in one language: final-round-2015-german.pdf is the paper of the final of 2015 in German.
var imosuisseRE = regexp.MustCompile(`href\s*=\s*"(/files/exams/(second-round|final-round|selection)-((?:19|20)\d\d)-(german|french|italian|english)\.pdf)"`)

// imosuisseRounds are the competitions of the exams of the archive: the second round, the final and the IMO selection.
var imosuisseRounds = map[string]string{"second-round": "ch-smo-r2", "final-round": "ch-smo", "selection": "ch-tst"}

// imosuisseLangs are the languages of the papers.
var imosuisseLangs = map[string]Lang{"german": {"German", "de"}, "french": {"French", "fr"}, "italian": {"Italian", "it"}, "english": {"English", "en"}}

// ImosuissePapers gives the paper of each exam of the listing in each of its languages. A paper holds all the problems of its exam, numbered through the edition, so it holds all the positions.
func ImosuissePapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range imosuisseRE.FindAllStringSubmatch(page, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		y, _ := strconv.Atoi(m[3])
		out = append(out, National{Competition: imosuisseRounds[m[2]], Year: y, URL: "https://imosuisse.github.io" + m[1], Lang: imosuisseLangs[m[4]]})
	}
	return out
}
