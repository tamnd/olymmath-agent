package official

import (
	"encoding/json"
	"strings"
)

// omaRoundComps gives the competition of the register for a contest of the archive index of the OMA that has rounds by level: the OMA itself (ar-oma) and the Ñandú (ar-nandu).
var omaRoundComps = map[string]string{"oma": "ar-oma", "nandu": "ar-nandu"}

// omaRounds gives the letter of the papers of a round for the label of its file in the archive index. The OMA calls its school round Intercolegial and the Ñandú calls it Interescolar.
var omaRounds = map[string]string{"intercolegial": "i", "interescolar": "i", "zonal": "z", "regional": "r", "nacional": "n"}

// OMARoundPapers gives the round papers of the OMA (ar-oma) and of the Ñandú (ar-nandu) in the archive index of the OMA. A file holds one round of the three levels, one level after the other, so each file gives a paper for each level: i1, i2 and i3 for the school round, and z, r and n for the zonal, regional and national rounds. om ocr reads each level with -paper and -pages. A file with a label of no round is left out.
func OMARoundPapers(page string) []National {
	var idx omaIndex
	if err := json.Unmarshal([]byte(page), &idx); err != nil {
		return nil
	}
	var out []National
	seen := map[string]bool{}
	for _, it := range idx.Items {
		comp := omaRoundComps[it.Competition]
		if comp == "" || (it.Visible != nil && !*it.Visible) {
			continue
		}
		for _, f := range it.Files {
			r := omaRounds[strings.ToLower(strings.TrimSpace(f.Label))]
			if r == "" || !strings.HasSuffix(strings.ToLower(f.URL), ".pdf") {
				continue
			}
			u := f.URL
			if strings.HasPrefix(u, "/") {
				u = "https://www.oma.org.ar" + u
			}
			if seen[u] {
				continue
			}
			seen[u] = true
			for _, level := range []string{"1", "2", "3"} {
				out = append(out, National{Competition: comp, Year: it.Year, Paper: r + level, URL: u})
			}
		}
	}
	return out
}

// OMAAllPapers gives the papers of the May Olympiad, of the OMA and of the Ñandú in the archive index of the OMA.
func OMAAllPapers(page string) []National {
	return append(OMAPapers(page), OMARoundPapers(page)...)
}
