package archive

import (
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
)

// TSTContents is the API URL that lists the files of the USA TST Archive.
const TSTContents = "https://api.github.com/repos/usa-tst-public/USA-TST-Archive/contents/"

// TSTPaper is one problems file of the USA TST Archive.
type TSTPaper struct {
	// Competition is the code of the competition in the register, for example tstst.
	Competition string
	Year        int
	URL         string
}

// tstRE finds the test, the year and the kind of a file name, for example TSTST2025Problems.pdf.
var tstRE = regexp.MustCompile(`^(IMOTST|EGMOTST|TSTST)(\d{4})(Problems|Solutions|Statistics)\.pdf$`)

// tstCodes gives the competition code of each test of the archive.
var tstCodes = map[string]string{"IMOTST": "us-tst", "EGMOTST": "us-egmo-tst", "TSTST": "tstst"}

// TSTPapers gives the problems files in a listing of the GitHub contents API, sorted by competition and year.
func TSTPapers(listing []byte) ([]TSTPaper, error) {
	var files []struct {
		Name string `json:"name"`
		URL  string `json:"html_url"`
	}
	if err := json.Unmarshal(listing, &files); err != nil {
		return nil, err
	}
	var out []TSTPaper
	for _, f := range files {
		m := tstRE.FindStringSubmatch(f.Name)
		if m == nil || m[3] != "Problems" {
			continue
		}
		year, _ := strconv.Atoi(m[2])
		out = append(out, TSTPaper{Competition: tstCodes[m[1]], Year: year, URL: f.URL})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Competition != out[j].Competition {
			return out[i].Competition < out[j].Competition
		}
		return out[i].Year < out[j].Year
	})
	return out, nil
}
