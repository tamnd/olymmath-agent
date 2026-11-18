package archive

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// ApexBase is the site of Apex Minds, an open archive of Ukrainian olympiad problems (source apex-minds).
const ApexBase = "https://apex-minds.org/"

// ApexManifest is the list of all problems of the site: one row for each problem, with its slug and the code of its competition.
const ApexManifest = ApexBase + "data_v2/manifest.json"

// ApexFull gives the URL of the full record of the problem with the slug, as ukr_2002_03697c. The site serves it gzipped only.
func ApexFull(slug string) string { return ApexBase + "data_v2/full/" + slug + ".json.gz" }

// ApexPage gives the URL of the page of the problem with the public id, as VUOM-01-02-018.
func ApexPage(id string) string { return ApexBase + "problem/" + id }

// ApexUkrainian are the codes of the Ukrainian competitions of the site. The other competitions of the site, as the IMO or the EGMO, are translations into Ukrainian, and the source register says that they are not reference texts.
var ApexUkrainian = []string{
	"UkrMO", "KyivCityMO", "KyivDistrictMO", "KyivSelection", "ZakarpattiaSelection", "UkrTST", "IntlTeamSelection",
	"KyivJuniorMO", "KyivMathBattle", "LyashkoMathBattle", "KyivMathCarousel", "KyivTeamChampionship",
	"NewYearZanzibar", "RichelieuMO", "SETUniversityMO", "ThematicMO",
}

// ApexSlugs gives the slugs of the problems of the Ukrainian competitions in the manifest, in the order of the manifest.
func ApexSlugs(manifest []byte) ([]string, error) {
	var m struct {
		Schema []string            `json:"schema"`
		Rows   [][]json.RawMessage `json:"rows"`
	}
	if err := json.Unmarshal(manifest, &m); err != nil {
		return nil, err
	}
	gi, ci := slices.Index(m.Schema, "g"), slices.Index(m.Schema, "c")
	if gi < 0 || ci < 0 {
		return nil, fmt.Errorf("the manifest has no slug or competition column: %v", m.Schema)
	}
	var out []string
	for _, r := range m.Rows {
		if len(r) <= max(gi, ci) {
			continue
		}
		var slug, comp string
		if json.Unmarshal(r[gi], &slug) != nil || json.Unmarshal(r[ci], &comp) != nil {
			continue
		}
		if slices.Contains(ApexUkrainian, comp) {
			out = append(out, slug)
		}
	}
	return out, nil
}

// ApexRecord is the full record of a problem of the site, with the fields that the import reads.
type ApexRecord struct {
	Slug        string   `json:"slug"`
	PublicID    string   `json:"public_id"`
	Problem     string   `json:"problem_markdown"`
	Answers     []string `json:"answers_markdown"`
	Solutions   []string `json:"solutions_markdown"`
	Competition string   `json:"competition_code"`
	Level       string   `json:"level"`
	Round       string   `json:"round"`
	SubEvent    string   `json:"sub_event"`
	School      string   `json:"academic_year"`
	Year        int      `json:"year"`
	Ref         string   `json:"competition_ref"`
	HasImages   bool     `json:"has_images"`
	Language    string   `json:"booklet_language"`
}

// ParseApex reads a full record: the gzipped JSON that the site serves, or the JSON itself.
func ParseApex(b []byte) (ApexRecord, error) {
	if len(b) > 2 && b[0] == 0x1f && b[1] == 0x8b {
		zr, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return ApexRecord{}, err
		}
		if b, err = io.ReadAll(zr); err != nil {
			return ApexRecord{}, err
		}
	}
	var r ApexRecord
	err := json.Unmarshal(b, &r)
	return r, err
}

// apexStage is the competition of the register that a competition and level of the site belong to, whether its paper is written in the first calendar year of the school year, and whether its two rounds are numbered each from 1.
type apexStage struct {
	code   string
	first  bool
	rounds bool
}

// apexStages maps the competition and the level of the site to the register. The order of the stages of the Ukrainian olympiad is fixed by the ministry each year: the district stage (II) in November and December, the regional stage (III), which in Kyiv is the Kyiv city olympiad, in January and February, and the final stage (IV) in March and April. So the district stage is the edition of the first year of the school year, and the others are the edition of its last year (spec 01f: an edition is the calendar year in which the paper is written).
var apexStages = map[string]apexStage{
	"UkrMO/district":          {code: "ua-umo-district", first: true},
	"UkrMO/regional":          {code: "ua-umo-regional", rounds: true},
	"UkrMO/national":          {code: "ua-umo"},
	"KyivCityMO/regional":     {code: "ua-kyiv", rounds: true},
	"KyivDistrictMO/district": {code: "ua-umo-district", first: true},
}

var (
	// apexRefRE is the grade and the number of a problem, as 9.3 or 10.12, after the round when the ref names it, as I тур · 8.3.
	apexRefRE = regexp.MustCompile(`^\s*(?:(I|II)\s*тур\s*·\s*)?([5-9]|1[01])\s*\.\s*([1-9][0-9]?)\s*$`)
	// apexVariantRE is a variant of a problem, as 8.2.1: a second problem in the place of problem 2 of grade 8, which a position cannot tell from the first.
	apexVariantRE = regexp.MustCompile(`^\s*(?:(I|II)\s*тур\s*·\s*)?([5-9]|1[01])\.[1-9][0-9]?\.[1-9]\s*$`)
	// apexTourRE is the round of a sub-event, as I тур or II тур.
	apexTourRE = regexp.MustCompile(`^\s*(I|II)\s*тур\s*$`)
	// apexSchoolRE is a school year, as 2001-2002.
	apexSchoolRE = regexp.MustCompile(`^\s*((?:19|20)[0-9]{2})\s*[-–/]\s*((?:19|20)[0-9]{2})\s*$`)
	// apexKyivRE marks a round of the regional stage that is the Kyiv city olympiad.
	apexKyivRE = regexp.MustCompile(`Київськ\S* міськ\S* олімпіад`)
	// apexImageRE is a figure of a record, as ![](images/0ffe159c.png). The site serves it from the root.
	apexImageRE = regexp.MustCompile(`(!\[[^\]]*\]\()images/([0-9a-f-]+\.(?:png|svg|jpe?g|gif))\)`)
)

// ApexPlace gives the competition code, the edition year and the problem number of a record, as ua-umo-district, 2001 and p93, or the reason why the record has no place. The number is p, the grade and the number of the problem in the paper of the grade. The Kyiv city olympiad and some regional olympiads have two rounds, each numbered from 1, so problem N of grade G of the first round is aG followed by N and of the second round bG followed by N, as the register numbers two rounds with a and b. The final stage numbers its two days on, as 9.1 to 9.8, so its rounds need no letter.
func ApexPlace(r ApexRecord) (code string, year int, number, why string) {
	if code, year, number, why, ok := apexMore(r); ok {
		return code, year, number, why
	}
	st, ok := apexStages[r.Competition+"/"+r.Level]
	if !ok {
		return "", 0, "", "no competition of the register for " + r.Competition + "/" + r.Level
	}
	if st.code == "ua-umo-regional" && apexKyivRE.MatchString(r.Round) {
		st = apexStage{code: "ua-kyiv", rounds: true}
	}
	if strings.Contains(r.SubEvent, "відбор") {
		// The rounds of the selection of the Kyiv team for the final stage, which the booklets give with the city olympiad.
		return "", 0, "", "a round of the selection of the Kyiv team"
	}
	if r.Competition == "KyivCityMO" && strings.Contains(r.SubEvent, "етап") {
		// The online olympiad of 2021 had three stages in the place of the two rounds.
		return "", 0, "", "a stage of the Kyiv online olympiad"
	}
	m := apexSchoolRE.FindStringSubmatch(r.School)
	if m == nil {
		return "", 0, "", "no school year"
	}
	first, _ := strconv.Atoi(m[1])
	last, _ := strconv.Atoi(m[2])
	if last != first+1 {
		return "", 0, "", "no school year"
	}
	year = last
	if st.first {
		year = first
	}
	ref := apexLatin(r.Ref)
	if apexVariantRE.MatchString(ref) {
		return "", 0, "", "a variant of a problem, as 8.2.1"
	}
	g := apexRefRE.FindStringSubmatch(ref)
	if g == nil {
		return "", 0, "", "no grade and number"
	}
	tour := g[1]
	if t := apexTourRE.FindStringSubmatch(apexLatin(r.SubEvent)); t != nil {
		if tour != "" && tour != t[1] {
			return "", 0, "", "two rounds in the ref and the sub-event"
		}
		tour = t[1]
	}
	letter := "p"
	if st.rounds && tour != "" {
		letter = map[string]string{"I": "a", "II": "b"}[tour]
	}
	return st.code, year, letter + g[2] + g[3], ""
}

// apexLatin writes the Cyrillic І of a Roman numeral, as in І тур, as the Latin I.
func apexLatin(s string) string { return strings.ReplaceAll(s, "І", "I") }

// ApexText gives the statement of a record with each figure linked to its file on the site, as ![](https://apex-minds.org/images/0ffe159c.png). om apex -figures then puts the figure next to the problem.
func ApexText(r ApexRecord) string {
	return strings.TrimSpace(apexImageRE.ReplaceAllString(r.Problem, "${1}"+ApexBase+"images/${2})"))
}

// ApexAnswer gives the answer of a record when it has exactly one, or "".
func ApexAnswer(r ApexRecord) string {
	if len(r.Answers) != 1 {
		return ""
	}
	return strings.TrimSpace(r.Answers[0])
}

// IsApexImage tells if a figure ref is a figure of the site, and gives the name of its file.
func IsApexImage(ref string) (string, bool) {
	name, ok := strings.CutPrefix(ref, ApexBase+"images/")
	return name, ok && name != "" && !strings.Contains(name, "/")
}
