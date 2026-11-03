package official

import (
	"html"
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

// HBCSEListing is the page of the Homi Bhabha Centre for Science Education that lists the past papers of the Indian olympiads.
const HBCSEListing = "https://olympiads.hbcse.tifr.res.in/how-to-prepare/past-papers/"

var (
	// hbcseLinkRE matches a link to a PDF file and its text.
	hbcseLinkRE = regexp.MustCompile(`(?s)<a\s[^>]*?href="([^"]+\.pdf)"[^>]*>(.*?)</a>`)
	// hbcseOldRE matches the old links of the page, to the address of the server or to the folder /olympiads/. They give no file.
	hbcseOldRE = regexp.MustCompile(`^http://[0-9.]+/|/olympiads/wp-content/`)
	// hbcseTagRE matches an HTML tag.
	hbcseTagRE = regexp.MustCompile(`<[^>]+>`)
	// hbcseOtherRE matches the files of the other olympiads and of the first round, and the files in Hindi only.
	hbcseOtherRE = regexp.MustCompile(`prmo|prermo|ioqm|inao|inbo|incho|inpho|injso|ijso|answerkey|-key|_key`)
	// hbcseRMORE matches the name of the regional olympiad, but not PRMO.
	hbcseRMORE = regexp.MustCompile(`(?:^|[^p])c?rmo`)
	// hbcseYearRE matches a year with four digits.
	hbcseYearRE = regexp.MustCompile(`(?:19|20)[0-9]{2}`)
	// hbcseShortRE matches a year with two digits after the name of the olympiad, as in crmo-14-2.pdf, inmo23_qp.pdf or RMO23-24.
	hbcseShortRE = regexp.MustCompile(`(?:inmo|rmo)(?:sol)?[-_]?([0-9]{2})(?:[^0-9]|$)`)
	// hbcsePaperRE matches the number of the paper of a year with several papers: crmo-14-2.pdf, crmo-2013-paper-2.pdf, RMO-2025-Paper-2.pdf, crmo-2013-solutions-2.pdf or the text "CRMO 2016 Paper 2".
	hbcsePaperRE = regexp.MustCompile(`rmo(?:sol)?[-_][0-9]{2}[-_]([1-5])\.pdf|(?:paper|solutions?)[-_ ]?([1-5])(?:[^0-9]|$)`)
)

// HBCSEPapers gives the papers of the Regional Mathematical Olympiad (in-rmo) and of the Indian National Mathematical Olympiad (in-inmo) on the past papers page of HBCSE. The year of a paper is the year in which it was held: the INMO of January 2026 is INMO 2026.
//
// A year of the RMO can have several papers. From 2012 to 2016 the regions wrote CRMO papers 1 to 4, and the Mumbai region wrote its own paper (paper 5): their paper ids are paper1 to paper4 and mumbai. In 2018 the Kerala and Tamil Nadu regions wrote a later paper, and from 2023 the students of KV and JNV schools write their own paper: these papers have the id paper2, and the main paper has the id paper1. A file of a year with one paper has no paper id.
//
// A file of solutions holds the problems too, so it is used when the page has no question paper for the same paper. A file in Hindi only is not used, but a bilingual file is. The old links of the page give no file, so they are not used: the papers of RMO 2000, RMO 2003 and INMO 2008.
func HBCSEPapers(page string) []National {
	type key struct {
		comp  string
		year  int
		paper string
	}
	type found struct {
		National
		solution bool
	}
	var keys []key
	best := map[key]found{}
	for _, m := range hbcseLinkRE.FindAllStringSubmatch(page, -1) {
		url := m[1]
		if hbcseOldRE.MatchString(url) {
			continue
		}
		name := strings.ToLower(url[strings.LastIndex(url, "/")+1:])
		text := strings.ToLower(strings.Join(strings.Fields(html.UnescapeString(hbcseTagRE.ReplaceAllString(m[2], " "))), " "))
		both := name + " " + text
		if hbcseOtherRE.MatchString(both) || (strings.Contains(both, "hindi") && !strings.Contains(both, "english") && !strings.Contains(both, "bilingual")) {
			continue
		}
		comp := ""
		switch {
		case strings.Contains(both, "inmo"):
			comp = "in-inmo"
		case hbcseRMORE.MatchString(both):
			comp = "in-rmo"
		default:
			continue
		}
		year := hbcseYear(name, text)
		if year == 0 {
			continue
		}
		paper := ""
		if comp == "in-rmo" {
			paper = hbcsePaper(name, text, both)
		}
		k := key{comp, year, paper}
		f := found{National{Competition: comp, Year: year, Paper: paper, URL: url}, strings.Contains(name, "sol") || strings.Contains(text, "solution") || strings.Contains(text, "answer key")}
		old, ok := best[k]
		if !ok {
			keys = append(keys, k)
		}
		if !ok || (old.solution && !f.solution) {
			best[k] = f
		}
	}
	out := make([]National, 0, len(keys))
	for _, k := range keys {
		n := best[k].National
		if k.paper == "" {
			// The file of a year with several papers that names no paper is the file of the main paper, or the solutions of the main paper when the page has its question paper.
			_, first := best[key{k.comp, k.year, "paper1"}]
			_, second := best[key{k.comp, k.year, "paper2"}]
			if first {
				continue
			}
			if second {
				n.Paper = "paper1"
			}
		}
		out = append(out, n)
	}
	slices.SortStableFunc(out, func(a, b National) int {
		if a.Competition != b.Competition {
			return strings.Compare(a.Competition, b.Competition)
		}
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		return strings.Compare(a.Paper, b.Paper)
	})
	return out
}

// hbcseYear gives the year of a file of HBCSE from its name, or else from the text of its link.
func hbcseYear(name, text string) int {
	for _, s := range []string{name, text} {
		if y := hbcseYearRE.FindString(s); y != "" {
			n, _ := strconv.Atoi(y)
			return n
		}
		if m := hbcseShortRE.FindStringSubmatch(s); m != nil {
			n, _ := strconv.Atoi(m[1])
			return 2000 + n
		}
	}
	return 0
}

// hbcsePaper gives the paper id of a file of the RMO, see HBCSEPapers.
func hbcsePaper(name, text, both string) string {
	switch {
	case strings.Contains(both, "mumbai"):
		return "mumbai"
	case strings.Contains(both, "kl-tn"):
		return "paper2"
	case strings.Contains(both, "non kv") || strings.Contains(both, "non-kv") || strings.Contains(both, "nonkv"):
		return "paper1"
	case strings.Contains(both, "kv"):
		return "paper2"
	}
	for _, s := range []string{name, text} {
		if m := hbcsePaperRE.FindStringSubmatch(s); m != nil {
			n := m[1] + m[2]
			if n == "5" {
				return "mumbai"
			}
			return "paper" + n
		}
	}
	return ""
}

// IMOJPListings are the pages of the Mathematical Olympiad Foundation of Japan that list the past papers of the JMO and of the JJMO.
var IMOJPListings = []string{
	"https://www.imojp.org/domestic/jmo_overview.html",
	"https://www.imojp.org/domestic/jjmo_overview.html",
}

// imojpRE matches the link to a paper of the JMO or the JJMO: jmo26mq.html is the final (honsen) of the 26th JMO, and jmo36yqa.pdf is the preliminary round (yosen) of the 36th JMO. The final is mq up to 2019 and hq after. The file jjmo1q.html is the one paper of the 1st JJMO.
var imojpRE = regexp.MustCompile(`href="(?:\.\./|/|https://www\.imojp\.org/)((?:archive/)[^"]*/(j?jmo)([0-9]{1,2})(yqa|yq|hq|mq|q)\.(?:html|pdf))"`)

// IMOJPPapers gives the papers of the Japan Mathematical Olympiad (jp-jmo) and of the Japan Junior Mathematical Olympiad (jp-jjmo) on the listing pages of imojp.org. The nth JMO is held in the year 1990 + n, and the nth JJMO in the year 2002 + n, in January and February. The final has the five problems of the edition, so its paper holds all the positions. The preliminary round has its own short-answer problems, which have no position in the content: its paper has the paper id prelim. The first six JJMO, from 2003 to 2008, had one paper of twelve short-answer problems and no final, so their papers are not used. The page of a paper is an HTML page before 2024 and a PDF file after.
func IMOJPPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range imojpRE.FindAllStringSubmatch(page, -1) {
		n, _ := strconv.Atoi(m[3])
		if n == 0 || seen[m[1]] {
			continue
		}
		seen[m[1]] = true
		comp, year := "jp-jmo", 1990+n
		if m[2] == "jjmo" {
			comp, year = "jp-jjmo", 2002+n
		}
		paper := ""
		switch m[4] {
		case "yq", "yqa":
			paper = "prelim"
		case "q":
			continue
		}
		out = append(out, National{Competition: comp, Year: year, Paper: paper, URL: "https://www.imojp.org/" + m[1]})
	}
	slices.SortStableFunc(out, func(a, b National) int {
		if a.Competition != b.Competition {
			return strings.Compare(a.Competition, b.Competition)
		}
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		return strings.Compare(a.Paper, b.Paper)
	})
	return out
}

// AMTListing is the page of the Australian Maths Trust that lists its past papers.
const AMTListing = "https://amt.edu.au/department/past-papers"

// amtRE matches the link to a paper of the Australian Mathematical Olympiad: 2016-AMO-Paper-and-Solutions.pdf or AMO-2019-paper-and-solutions.pdf. The papers of the AIMO and the AJMO have other names.
var amtRE = regexp.MustCompile(`(?i)href="(?:https?://(?:www\.)?amt\.edu\.au)?(/wp-content/uploads/[0-9]{4}/[0-9]{2}/(?:([0-9]{4})-AMO|AMO-([0-9]{4}))-[^"/]*\.pdf)"`)

// AMTPapers gives the papers of the Australian Mathematical Olympiad (au-amo) on the past papers page of the AMT. Each file holds the eight problems of the two days and then the solutions, so its paper holds all the positions. The year is the year in the name of the file, which is the year of the olympiad.
func AMTPapers(page string) []National {
	var out []National
	seen := map[int]bool{}
	for _, m := range amtRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[2] + m[3])
		if y == 0 || seen[y] {
			continue
		}
		seen[y] = true
		out = append(out, National{Competition: "au-amo", Year: y, URL: "https://amt.edu.au" + m[1]})
	}
	slices.SortStableFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}

// VTRMCListing is the page of the Virginia Tech Regional Mathematics Contest that lists its past exams and their solutions.
const VTRMCListing = "https://math.vt.edu/undergrad-math/vtrmc.html"

// vtrmcRE matches the link to the exam of one year, with the year as the text of the link: <a href="http://intranet.math.vt.edu/vtrmc/exams/42nd%20VTRMC.pdf">2022</a>.
var vtrmcRE = regexp.MustCompile(`href="https?://intranet\.math\.vt\.edu/vtrmc/exams/[^"]+\.pdf"[^>]*>\s*((?:19|20)[0-9]{2})`)

// VTRMCPapers gives the exam of each year of the Virginia Tech Regional Mathematics Contest (vtrmc) on its page. The year is the text of the link. The file of an exam is named by the number of the contest: the nth contest was held in the year 1978 + n up to 2019, and the 42nd in 2022, because the contest was not held in 2020 and 2021. Two links of the page are wrong: the link of 2009 goes to the exam of 2010, and the link of 2011 names the file 33th, which does not exist. So the URL is made from the number of the contest of the year, and the file of each year exists.
func VTRMCPapers(page string) []National {
	var out []National
	seen := map[int]bool{}
	for _, m := range vtrmcRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[1])
		n := y - 1978
		if y >= 2022 {
			n = y - 1980
		}
		if n < 1 || seen[y] {
			continue
		}
		seen[y] = true
		out = append(out, National{Competition: "vtrmc", Year: y, URL: "https://intranet.math.vt.edu/vtrmc/exams/" + ordinal(n) + "%20VTRMC.pdf"})
	}
	slices.SortStableFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}

// ordinal gives the English ordinal of n: 1st, 2nd, 3rd, 4th, 11th, 22nd.
func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return strconv.Itoa(n) + suffix
}

// IMCFirst is the first year of the International Mathematics Competition for University Students.
const IMCFirst = 1994

// IMCListings gives the problems page of each year of the IMC from 1994 to the year last.
func IMCListings(last int) []string {
	var out []string
	for y := IMCFirst; y <= last; y++ {
		out = append(out, "https://www.imc-math.org.uk/?year="+strconv.Itoa(y)+"&item=problems")
	}
	return out
}

// imcRE matches the link to a file of the problems of one year or one day of the IMC: IMC_1994_Problems_and_Solutions.pdf, IMC_2012_Day_1_Problems.pdf, IMC_2016_Day_2_Questions.pdf or IMC_2025_Day_1_Solutions.pdf.
var imcRE = regexp.MustCompile(`href="(?:https://www\.imc-math\.org\.uk)?(/archive/problems/((?:19|20)[0-9]{2})/IMC_[0-9]{4}_(?:Day_([12])_)?(Problems_and_Solutions|Problems|Questions|Solutions)\.pdf)"`)

// imcRank gives the order of preference of the kinds of file of the IMC. A Questions file is the paper of the day, made with TeX. A Problems file of 2015 to 2017 is a print of the web page, and its text is cut at the right edge, so it comes second. The solutions come last, and they state the problems too.
var imcRank = map[string]int{"Questions": 0, "Problems": 1, "Problems_and_Solutions": 2, "Solutions": 3}

// IMCPapers gives the papers of the International Mathematics Competition for University Students (imc) on its problems pages. The IMC has two days, and the paper of day N has the id dayN. A year with one file for both days has a paper with no id, which holds all the positions.
func IMCPapers(page string) []National {
	type key struct {
		year  int
		paper string
	}
	best := map[key]string{}
	rank := map[key]int{}
	for _, m := range imcRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[2])
		k := key{y, ""}
		if m[3] != "" {
			k.paper = "day" + m[3]
		}
		if r, ok := rank[k]; ok && r <= imcRank[m[4]] {
			continue
		}
		rank[k] = imcRank[m[4]]
		best[k] = "https://www.imc-math.org.uk" + m[1]
	}
	out := make([]National, 0, len(best))
	for k, u := range best {
		out = append(out, National{Competition: "imc", Year: k.year, Paper: k.paper, URL: u})
	}
	slices.SortStableFunc(out, func(a, b National) int {
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		return strings.Compare(a.Paper, b.Paper)
	})
	return out
}
