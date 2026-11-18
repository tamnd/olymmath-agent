package archive

import (
	"regexp"
	"strconv"
	"strings"
)

// apexNoNumber are the competitions of the site whose records give no number in the paper, so that they have no position: the math battles, carousels and team contests, and the open olympiad of the Richelieu Lyceum.
var apexNoNumber = map[string]string{
	"KyivMathBattle":       "ua-kyiv-math-battle",
	"LyashkoMathBattle":    "ua-lyashko-math-battle",
	"KyivMathCarousel":     "ua-kyiv-carousel",
	"KyivJuniorMO":         "ua-kyiv-junior",
	"KyivTeamChampionship": "ua-kyiv-team-championship",
	"RichelieuMO":          "ua-richelieu",
}

var (
	// apexRoundRE is the round of a sub-event or a ref, as I тур, IV (додатковий) тур or V тур (додатковий).
	apexRoundRE = regexp.MustCompile(`^\s*(I|II|III|IV|V)\s*(?:\(додатковий\)\s*)?тур`)
	// apexGradeRefRE is the grade and the number of a ref, after the round or the field when the ref names it, as 8.3, I тур · 8.3 or алгебра · 9.3.
	apexGradeRefRE = regexp.MustCompile(`^\s*(?:([^·]+?)\s*·\s*)?([1-9]|1[01])\s*\.\s*([1-9][0-9]?)\s*$`)
	// apexNumberRefRE is a ref with the number only, as 7, or with the group after it, as 11.2.
	apexNumberRefRE = regexp.MustCompile(`^\s*([1-9][0-9]?)(?:\.([12]))?\s*$`)
	// apexLabelYearRE is the year that the name of a selection gives, as Відбір на міжнародні змагання (2021).
	apexLabelYearRE = regexp.MustCompile(`\(((?:19|20)[0-9]{2})\)`)
)

// apexRoundLetters are the letters of the rounds I to V, as the register numbers the rounds of a paper with a, b, c and on.
var apexRoundLetters = map[string]string{"I": "a", "II": "b", "III": "c", "IV": "d", "V": "e"}

// apexFields are the letters of the four olympiads of ua-thematic: algebra, number theory, geometry and combinatorics.
var apexFields = []struct{ word, letter string }{{"алгебр", "a"}, {"теорі", "n"}, {"геометр", "g"}, {"комбінатор", "c"}}

// apexMore places the records of the Ukrainian selections, the thematic olympiads and the other competitions of the site that are not stages of the Ukrainian olympiad. It tells with ok whether the competition is one of them. Their rows of the register (ua-kyiv-selection, ua-tst and the others) give the positions.
func apexMore(r ApexRecord) (code string, year int, number, why string, ok bool) {
	if c, found := apexNoNumber[r.Competition]; found {
		return c, 0, "", "the site gives no number in the paper", true
	}
	switch r.Competition {
	case "KyivSelection", "ZakarpattiaSelection", "UkrTST", "IntlTeamSelection", "ThematicMO", "SETUniversityMO", "NewYearZanzibar":
	default:
		return "", 0, "", "", false
	}
	m := apexSchoolRE.FindStringSubmatch(r.School)
	if m == nil {
		return "", 0, "", "no school year", true
	}
	first, _ := strconv.Atoi(m[1])
	year, _ = strconv.Atoi(m[2])
	if year != first+1 {
		return "", 0, "", "no school year", true
	}
	ref, sub := apexLatin(r.Ref), apexLatin(r.SubEvent)
	if apexVariantRE.MatchString(ref) {
		return "", 0, "", "a variant of a problem, as 8.2.1", true
	}
	switch r.Competition {
	case "KyivSelection", "ZakarpattiaSelection":
		code = "ua-kyiv-selection"
		if r.Competition == "ZakarpattiaSelection" {
			code = "ua-zakarpattia-selection"
		}
		if strings.Contains(r.Round, "гуртки") {
			return "", 0, "", "the selection for the Kyiv study groups", true
		}
		g := apexGradeRefRE.FindStringSubmatch(ref)
		if g == nil {
			return "", 0, "", "no grade and number", true
		}
		if code == "ua-zakarpattia-selection" {
			return code, year, "p" + g[2] + g[3], "", true
		}
		round := ""
		if t := apexRoundRE.FindStringSubmatch(g[1]); t != nil {
			round = t[1]
		}
		if t := apexRoundRE.FindStringSubmatch(sub); t != nil {
			if round != "" && round != t[1] {
				return "", 0, "", "two rounds in the ref and the sub-event", true
			}
			round = t[1]
		}
		if round == "" {
			return "", 0, "", "no round", true
		}
		return code, year, apexRoundLetters[round] + g[2] + g[3], "", true
	case "UkrTST":
		n := apexNumberRefRE.FindStringSubmatch(ref)
		if n == nil || n[2] != "" {
			return "", 0, "", "no number", true
		}
		// The training camps of 2008 and 2009 number each of their four rounds from 1; the other years number on through the rounds.
		if t := apexRoundRE.FindStringSubmatch(sub); t != nil && strings.HasPrefix(r.Round, "Відбірково-тренувальні збори") {
			return "ua-tst", year, apexRoundLetters[t[1]] + n[1], "", true
		}
		return "ua-tst", year, "p" + n[1], "", true
	case "IntlTeamSelection":
		// The name of the selection gives the year in which it was written, as (2021) for the selection of the autumn of 2021; with no year in the name, the selection is that of the spring.
		if y := apexLabelYearRE.FindStringSubmatch(r.Round); y != nil {
			year, _ = strconv.Atoi(y[1])
			if year != first && year != first+1 {
				return "", 0, "", "a year in the name that is not of the school year", true
			}
			if year == 2020 && first == 2020 {
				// Both the selection of 2019/20 and that of 2020/21 are named (2020), and their positions would be the same.
				return "", 0, "", "a second selection named 2020", true
			}
		}
		n := apexNumberRefRE.FindStringSubmatch(ref)
		if n == nil {
			return "", 0, "", "no number", true
		}
		letter := "p"
		switch {
		case strings.Contains(sub, "відбору на EGMO"):
			letter = "e"
		case n[2] == "1" || strings.Contains(sub, "молодша група"):
			letter = "j"
		case n[2] == "2" || strings.Contains(sub, "група IMO"):
			letter = "i"
		}
		return "ua-intl-selection", year, letter + n[1], "", true
	case "ThematicMO":
		g := apexGradeRefRE.FindStringSubmatch(ref)
		if g == nil {
			return "", 0, "", "no grade and number", true
		}
		field := strings.ToLower(g[1] + " " + r.SubEvent + " " + r.Round)
		for _, f := range apexFields {
			if strings.Contains(field, f.word) {
				return "ua-thematic", year, f.letter + g[2] + g[3], "", true
			}
		}
		return "", 0, "", "no field of the thematic olympiad", true
	case "SETUniversityMO":
		g := apexGradeRefRE.FindStringSubmatch(ref)
		if g == nil {
			return "", 0, "", "no grade and number", true
		}
		t := apexRoundRE.FindStringSubmatch(g[1])
		if t == nil || (t[1] != "I" && t[1] != "II") {
			return "", 0, "", "no round", true
		}
		return "ua-set-university", year, apexRoundLetters[t[1]] + g[2] + g[3], "", true
	}
	// NewYearZanzibar
	g := apexGradeRefRE.FindStringSubmatch(ref)
	if g == nil || g[1] != "" {
		return "", 0, "", "no grade and number", true
	}
	return "ua-newyear-zanzibar", year, "p" + g[2] + g[3], "", true
}
