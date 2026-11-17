package archive

import (
	"regexp"
	"strconv"
	"strings"
)

// ImosuisseIndex is the page of the problems of the archive of the Swiss Mathematical Olympiad (source imosuisse). It names each problem by its id, as 2015-3-07: the year, the round and the number.
const ImosuisseIndex = "https://imosuisse.github.io/problems/"

// imosuisseTeX is the TeX file of a problem in the GitHub repository of the archive. The archive writes its problems in English.
const imosuisseTeX = "https://raw.githubusercontent.com/imosuisse/imosuisse.github.io/main/_latex/"

// imosuisseIDRE matches the id of a problem of the second round (2), the final (3) or the IMO selection (4) on the index. The number is two digits, or from 2017 in the second round a subject letter and a digit, as 2017-2-G1 (G geometry, K combinatorics, N number theory). The MEMO, the IMO and the French olympiad problems of the archive have other ids, and they are not Swiss papers.
var imosuisseIDRE = regexp.MustCompile(`\bid: "((\d{4})-([234])-(\d{2}|[GKN][1-9]))"`)

// imosuisseSubjects are the labels that the papers print for the subject letters of the ids. The papers write number theory as Z (Zahlentheorie) in every language, where the ids write N.
var imosuisseSubjects = map[byte]string{'G': "G", 'K': "K", 'N': "Z"}

// imosuisseRounds are the competitions of the rounds of the ids.
var imosuisseRounds = map[string]string{"2": "ch-smo-r2", "3": "ch-smo", "4": "ch-tst"}

// ImosuisseDocs gives the TeX file of each problem of the index from the year from to the year to, with its competition as the label and its number.
func ImosuisseDocs(index string, from, to int) []Doc {
	var out []Doc
	seen := map[string]bool{}
	for _, m := range imosuisseIDRE.FindAllStringSubmatch(index, -1) {
		y, _ := strconv.Atoi(m[2])
		if seen[m[1]] || y < from || y > to {
			continue
		}
		seen[m[1]] = true
		num := m[4]
		if n, err := strconv.Atoi(num); err == nil {
			num = strconv.Itoa(n)
		} else {
			num = imosuisseSubjects[num[0]] + num[1:]
		}
		out = append(out, Doc{Year: y, URL: imosuisseTeX + m[1] + ".tex", Label: imosuisseRounds[m[3]], Number: num})
	}
	return out
}

// ParseImosuisse reads the TeX file of one problem. The file is the statement alone, with no preamble. Its number comes from its Doc.
func ParseImosuisse(tex string) []Problem {
	tex = strings.TrimSpace(Expand(tex))
	if tex == "" {
		return nil
	}
	p := Problem{}
	p.Text, p.Err = Markdown(tex)
	if p.Err != nil {
		p.Text = ""
	}
	return []Problem{p}
}
