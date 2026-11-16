package official

import (
	"encoding/json"
	"strings"
)

// OMAListing is the index of the archive of statements of the Olimpíada Matemática Argentina (OMA). The page https://www.oma.org.ar/enunciados/ builds its list from it with a script. It lists the papers of the OMA, of the Ñandú and of the international contests that the OMA runs or sends a team to.
const OMAListing = "https://www.oma.org.ar/contents/enunciados/index.json"

// omaIndex is the part of the archive index that om papers reads. Each item is one contest of one year with its files.
type omaIndex struct {
	Items []struct {
		Competition string `json:"competition"`
		Year        int    `json:"year"`
		Visible     *bool  `json:"visible"`
		Files       []struct {
			Label string `json:"label"`
			URL   string `json:"url"`
		} `json:"files"`
	} `json:"items"`
}

// omaLevels gives the paper of the layout of the May Olympiad for the label of a file that holds one level. A file with another label holds both levels.
var omaLevels = map[string]string{"primer nivel": "level1", "segundo nivel": "level2"}

// OMAPapers gives the papers of the May Olympiad (may-olympiad) in the archive index of the OMA. Most years have one file with both levels, the first level on the first page and the second on the next, so the paper is "" and om ocr reads each level with -paper and -pages. A year with a file for each level gives each file its paper. The OMA papers and the Ñandú have no layout in the register, so the index gives only the May Olympiad.
func OMAPapers(page string) []National {
	var idx omaIndex
	if err := json.Unmarshal([]byte(page), &idx); err != nil {
		return nil
	}
	var out []National
	seen := map[string]bool{}
	for _, it := range idx.Items {
		if it.Competition != "mayo" || (it.Visible != nil && !*it.Visible) {
			continue
		}
		for _, f := range it.Files {
			if !strings.HasSuffix(strings.ToLower(f.URL), ".pdf") {
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
			out = append(out, National{Competition: "may-olympiad", Year: it.Year, Paper: omaLevels[strings.ToLower(strings.TrimSpace(f.Label))], URL: u})
		}
	}
	return out
}
