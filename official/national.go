package official

import (
	"regexp"
	"slices"
	"strconv"
	"strings"
)

// National is one official paper of a national olympiad, as the listing page of its organizer gives it.
type National struct {
	Competition string
	Year        int
	// Paper is the id of the paper of the layout of the competition, or "" when the file holds all the positions of the edition.
	Paper string
	URL   string
}

// OMSemListing is the page of the Polish Mathematical Olympiad that lists the papers of each olympiad.
const OMSemListing = "https://om.sem.edu.pl/problems/"

// omSemRE matches the link to the paper of one stage of one olympiad: om66_3.pdf is the paper of the third stage of the 66th olympiad. A file with the letter r after the stage holds the solutions too.
var omSemRE = regexp.MustCompile(`href="(/static/app_main/problems/om([0-9]{1,3})_([123])\.pdf)(?:\?[^"]*)?"`)

// OMSemPapers gives the papers of the stages of each olympiad on the listing page of om.sem.edu.pl. The nth olympiad ends in the year 1949 + n. The third stage is the final, and it has the six problems of the edition, so its paper holds all the positions. The first and the second stages have their own problems, which have no position in the content. Their papers have the paper id stage1 and stage2, and they only help to find the rows of those stages, which have no position.
func OMSemPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range omSemRE.FindAllStringSubmatch(page, -1) {
		n, _ := strconv.Atoi(m[2])
		if n == 0 || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		paper := ""
		if m[3] != "3" {
			paper = "stage" + m[3]
		}
		out = append(out, National{Competition: "pl-om", Year: 1949 + n, Paper: paper, URL: "https://om.sem.edu.pl" + m[1]})
	}
	slices.SortStableFunc(out, func(a, b National) int {
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		return strings.Compare(a.Paper, b.Paper)
	})
	return out
}

// BWMListing is the page of Bildung & Begabung that lists the papers of the Bundeswettbewerb Mathematik.
const BWMListing = "https://www.mathe-wettbewerbe.de/aufgaben"

var (
	// bwmLinkRE matches a link to a file of the Bundeswettbewerb Mathematik.
	bwmLinkRE = regexp.MustCompile(`href="(/fileadmin/Mathe-Wettbewerbe/Bundeswettbewerb_Mathematik/Dokumente/[^"?#]+\.pdf)"`)
	// bwmNameRE matches the name of a file with the problems of one round: aufgaben_10_2.pdf, BWM_Aufgabenblatt_2026_1.pdf, 24_2_Aufgabenblatt_Web_240823.pdf or BWM_2023.1_Aufgabenblatt.pdf. A file with the solutions has another name.
	bwmNameRE = regexp.MustCompile(`(?i)^(?:(?:bwm_)?aufgaben(?:blatt)?_((?:20)?[0-9]{2})_([12])[_.]|(?:bwm_)?((?:20)?[0-9]{2})[._]([12])_aufgabenblatt[_.])`)
)

// BWMPapers gives the papers of each round of the Bundeswettbewerb Mathematik on the listing page. A year has two rounds of four problems. A file that does not name its year and its round, such as BWM_2025_Aufgabenblatt_SCREEN.pdf, is not used. When two files name the same round, the first one on the page is used.
func BWMPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range bwmLinkRE.FindAllStringSubmatch(page, -1) {
		name := m[1][strings.LastIndex(m[1], "/")+1:]
		n := bwmNameRE.FindStringSubmatch(name)
		if n == nil {
			continue
		}
		year, round := n[1]+n[3], n[2]+n[4]
		y, _ := strconv.Atoi(year)
		if y < 100 {
			y += 2000
		}
		paper := "round" + round
		if key := strconv.Itoa(y) + paper; !seen[key] {
			seen[key] = true
			out = append(out, National{Competition: "de-bundeswettbewerb", Year: y, Paper: paper, URL: "https://www.mathe-wettbewerbe.de" + m[1]})
		}
	}
	slices.SortStableFunc(out, func(a, b National) int {
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		return strings.Compare(a.Paper, b.Paper)
	})
	return out
}

// UKMTListing is the page of the British Mathematical Olympiad Subtrust that lists the papers of BMO1 and BMO2.
const UKMTListing = "https://bmos.ukmt.org.uk/home/bmo.shtml"

// ukmtRE matches the link to the paper of one year: bmo-1985.pdf is the paper of the single British olympiad before 1992, bmo1-2015.pdf and bmo2-2015.pdf are the papers of the two rounds, and fist-1980.pdf is the paper of the Further International Selection Test, the round that became BMO2. A file with a report, solutions or a histogram has a longer name.
var ukmtRE = regexp.MustCompile(`href="(?:https://bmos\.ukmt\.org\.uk)?(/home/(bmo|bmo1|bmo2|fist)-([0-9]{4})\.pdf)"`)

// UKMTPapers gives the papers of BMO1 and BMO2 on the listing page of bmos.ukmt.org.uk. The year of a paper is the year in the name of its file, which is the year of the end of the school year: the BMO1 of November 2014 is BMO1 2015. The single olympiad before 1992 is the first round, and FIST is the second round. Each paper holds all the problems of its edition.
func UKMTPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range ukmtRE.FindAllStringSubmatch(page, -1) {
		if seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		y, _ := strconv.Atoi(m[3])
		comp := "uk-bmo1"
		if m[2] == "bmo2" || m[2] == "fist" {
			comp = "uk-bmo2"
		}
		out = append(out, National{Competition: comp, Year: y, URL: "https://bmos.ukmt.org.uk" + m[1]})
	}
	slices.SortStableFunc(out, func(a, b National) int {
		if a.Competition != b.Competition {
			return strings.Compare(a.Competition, b.Competition)
		}
		return a.Year - b.Year
	})
	return out
}
