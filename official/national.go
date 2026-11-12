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
	// Lang is the language of the paper when it is not the language of the source, or the zero Lang.
	Lang Lang
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

// PAMOListing is the page of the Pan-African Mathematics Olympiad that shows the problems of each year.
const PAMOListing = "https://pamoofficial.org/problems/"

var (
	// pamoSectionRE matches the start of the section of one year on the problems page of PAMO: <section class="paper" id="pamo-2025" ...>.
	pamoSectionRE = regexp.MustCompile(`<section class="paper" id="pamo-((?:19|20)[0-9]{2})"`)
	// pamoProblemRE matches the start of one problem of a section: <li class="problem" data-n="4">.
	pamoProblemRE = regexp.MustCompile(`<li class="problem" data-n="([0-9]{1,2})">`)
	// tagRE matches an HTML tag.
	tagRE = regexp.MustCompile(`<[^>]*>`)
)

// PAMOPapers gives the paper of each year of the Pan-African Mathematics Olympiad (pamo) on its problems page. The page holds all the years, and the section of a year holds the six problems of its two days, numbered 1 to 6. So the paper of a year is the section with the id pamo-YEAR, and its URL is the URL of the page with that id as the fragment. A section with no problem gives no paper.
func PAMOPapers(page string) []National {
	var out []National
	seen := map[int]bool{}
	for _, m := range pamoSectionRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[1])
		if seen[y] || len(pamoProblemRE.FindAllString(PAMOSection(page, "pamo-"+m[1]), -1)) == 0 {
			continue
		}
		seen[y] = true
		out = append(out, National{Competition: "pamo", Year: y, URL: PAMOListing + "#pamo-" + m[1]})
	}
	slices.SortStableFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}

// PAMOSection gives the HTML of the section with the id id of the problems page of PAMO, up to the next section. It gives "" when the page has no such section.
func PAMOSection(page, id string) string {
	i := strings.Index(page, `<section class="paper" id="`+id+`"`)
	if i < 0 {
		return ""
	}
	s := page[i+1:]
	if j := strings.Index(s, `<section class="paper"`); j >= 0 {
		s = s[:j]
	}
	return s
}

// PAMOText gives the text of the section with the id id of the problems page of PAMO, with each problem as "Problem N." and its statement. The statements keep their TeX. It gives "" when the section has no problem.
func PAMOText(page, id string) string {
	s := PAMOSection(page, id)
	ms := pamoProblemRE.FindAllStringSubmatchIndex(s, -1)
	var b strings.Builder
	for i, m := range ms {
		end := len(s)
		if i+1 < len(ms) {
			end = ms[i+1][0]
		}
		body := strings.ReplaceAll(s[m[1]:end], "<br>", "\n")
		body = strings.ReplaceAll(body, "</p>", "\n")
		body = html.UnescapeString(tagRE.ReplaceAllString(body, " "))
		var lines []string
		for _, l := range strings.Split(body, "\n") {
			if l = strings.Join(strings.Fields(l), " "); l != "" {
				lines = append(lines, l)
			}
		}
		b.WriteString("Problem " + s[m[2]:m[3]] + ". " + strings.Join(lines, "\n") + "\n\n")
	}
	return b.String()
}

// TuymaadaListing is the archive page of the Tuymaada olympiad, which lists the booklets of each year.
const TuymaadaListing = "https://tuymaada.lensky-kray.ru/en/arhiv/"

// tuymaadaRE matches the link to the English booklet of one day of the mathematics olympiad, with the day and the year as the text of the link: <a href="http://tuymaada.lensky-kray.ru/wp-content/uploads/2026/07/T26-booklet-day1-eng.pdf" ...>Mathematics (first day) 2026</a>.
var tuymaadaRE = regexp.MustCompile(`href="https?://tuymaada\.lensky-kray\.ru(/wp-content/uploads/[^"]+\.pdf)"[^>]*>\s*Mathematics \((first|second) day\) ((?:19|20)[0-9]{2})\s*<`)

// TuymaadaPapers gives the papers of the Tuymaada olympiad (tuymaada) on its archive page. The booklet of a day holds the four problems of the senior league, then the four problems of the junior league, then the solutions. The first day has the problems 1 to 4 and the second day the problems 5 to 8. So each booklet gives two papers with the same URL: the paper senior and the paper junior of the layout. TuymaadaPart reads the part of each league.
func TuymaadaPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range tuymaadaRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[3])
		key := m[3] + m[2]
		if seen[key] {
			continue
		}
		seen[key] = true
		u := "https://tuymaada.lensky-kray.ru" + m[1]
		out = append(out, National{Competition: "tuymaada", Year: y, Paper: "senior", URL: u}, National{Competition: "tuymaada", Year: y, Paper: "junior", URL: u})
	}
	slices.SortStableFunc(out, func(a, b National) int {
		if a.Year != b.Year {
			return a.Year - b.Year
		}
		if a.URL != b.URL {
			return strings.Compare(a.URL, b.URL)
		}
		return strings.Compare(b.Paper, a.Paper)
	})
	return out
}

var (
	// tuymaadaLeagueRE matches the heading of the problems of a league in a booklet of Tuymaada: "Senior league" or "Junior League".
	tuymaadaLeagueRE = regexp.MustCompile(`(?im)^\s*(senior|junior) league\s*$`)
	// tuymaadaEndRE matches the heading of the solutions of a booklet of Tuymaada.
	tuymaadaEndRE = regexp.MustCompile(`(?m)^\s*SOLUTIONS\s*$`)
)

// TuymaadaPart gives the problems of the league paper ("senior" or "junior") from the text of a booklet of Tuymaada: the text from the heading of the league to the heading of the other league or of the solutions. It gives "" when the booklet has no such heading.
func TuymaadaPart(text, paper string) string {
	if loc := tuymaadaEndRE.FindStringIndex(text); loc != nil {
		text = text[:loc[0]]
	}
	ms := tuymaadaLeagueRE.FindAllStringSubmatchIndex(text, -1)
	for i, m := range ms {
		if !strings.EqualFold(text[m[2]:m[3]], paper) {
			continue
		}
		end := len(text)
		if i+1 < len(ms) {
			end = ms[i+1][0]
		}
		return tuymaadaNumber(text[m[1]:end])
	}
	return ""
}

// tuymaadaBareRE matches the number at the start of a problem of a booklet of Tuymaada: "5." at the start of a line.
var tuymaadaBareRE = regexp.MustCompile(`(?m)^[ \t]*([1-8])\.\s`)

// tuymaadaNumber puts the word Problem before the numbers of the problems of a league, so that Split keeps them. The problems of the second day start at 5, and Split only takes bare numbers from 1.
func tuymaadaNumber(text string) string { return numberRun(text, tuymaadaBareRE) }

// numberRun puts the word Problem before each number that re matches, from the first one, as long as the numbers follow each other. So a paper whose problems start at a number other than 1 keeps its numbers in Split. A number out of the sequence, as in a list in a problem, stays as it is.
func numberRun(text string, re *regexp.Regexp) string {
	var b strings.Builder
	last, next := 0, 0
	for _, m := range re.FindAllStringSubmatchIndex(text, -1) {
		n, _ := strconv.Atoi(text[m[2]:m[3]])
		if next != 0 && n != next {
			continue
		}
		b.WriteString(text[last:m[2]])
		b.WriteString("Problem ")
		last, next = m[2], n+1
	}
	b.WriteString(text[last:])
	return b.String()
}

// GeorgMohrListing is the page of the Georg Mohr Contest that lists the papers of the second round.
const GeorgMohrListing = "https://georgmohr.dk/gmopg/"

var (
	// georgMohrTableRE matches the table of papers of the page, which a script fills: the pattern of the name of each column, with $$ for the last two digits of the year, and the first and the last year of each column.
	georgMohrTableRE = regexp.MustCompile(`paths="([^"]+)"\s+from="([0-9 ]+)"\s+to="([0-9 ]+)"`)
	// Danish is the language of the papers in Danish.
	Danish = Lang{Lang: "Danish", Code: "da"}
)

// Lang is the language of a paper when it is not the language of its source.
type Lang struct{ Lang, Code string }

// GeorgMohrPapers gives the paper of each year of the second round of the Georg Mohr Contest (dk-georgmohr), which is the national olympiad of Denmark. The page has no links: a script makes them from the attributes of its table. The column gm$$opg.pdf has the problems in Danish, and the column gm$$pb.pdf has them in English, with brief solutions after them. A year gets the English paper when the page has it, and the Danish paper when it does not. The year of a paper is the year of the round, which is held in January.
func GeorgMohrPapers(page string) []National {
	m := georgMohrTableRE.FindStringSubmatch(page)
	if m == nil {
		return nil
	}
	paths, from, to := strings.Fields(m[1]), years(m[2]), years(m[3])
	if len(from) == 0 || len(to) == 0 {
		return nil
	}
	span := func(k int) (int, int) {
		f, t := from[0], to[0]
		if k < len(from) {
			f = from[k]
		}
		if k < len(to) {
			t = to[k]
		}
		return f, t
	}
	col := func(suffix string) int {
		return slices.IndexFunc(paths, func(p string) bool { return strings.HasSuffix(p, suffix) })
	}
	da, en := col("opg.pdf"), col("pb.pdf")
	byYear := map[int]National{}
	for _, k := range []int{da, en} {
		if k < 0 {
			continue
		}
		f, t := span(k)
		for y := f; y <= t; y++ {
			n := National{Competition: "dk-georgmohr", Year: y, URL: GeorgMohrListing + strings.Replace(paths[k], "$$", strconv.Itoa(y)[2:], 1)}
			if k == da {
				n.Lang = Danish
			}
			byYear[y] = n
		}
	}
	out := make([]National, 0, len(byYear))
	for _, n := range byYear {
		out = append(out, n)
	}
	slices.SortStableFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}

// years reads a list of years separated by spaces.
func years(s string) []int {
	var out []int
	for _, f := range strings.Fields(s) {
		if y, err := strconv.Atoi(f); err == nil && y >= 1900 {
			out = append(out, y)
		}
	}
	return out
}

// OPMListing is the page of the Sociedade Portuguesa de Matemática that lists the papers of each Portuguese Mathematical Olympiad.
const OPMListing = "https://olimpiadas.spm.pt/index.php?id=75&tipo=1"

var (
	// opmLinkRE matches the link to a file of one olympiad. The folder names the school year, such as "XXXI OPM(2012-13)".
	opmLinkRE = regexp.MustCompile(`href="(/Downloads/Ficheiros/Provas/[^"]*\(((?:19|20)[0-9]{2})-[0-9]{2}\)/([^"/]+\.pdf))"`)
	// opmFinalRE matches the name of a file with the problems of the final of category B: XXXI_f1b.pdf, XXVIII_f2bp.pdf, XXIV_fdia1b.pdf, XX_fb2.pdf or catBfinaldia1.pdf for one day, and XXXIX_fb.pdf or XXVI_f12bps.pdf for both days. A file of solutions ends in s, and category A and the juniors have the letters a and j.
	opmFinalRE = regexp.MustCompile(`(?i)^(?:[XLVI]+_(?:f([12])?bp?|f12bps|fdia([12])b|fb([12]))|catBfinaldia([12]))\.pdf$`)
	// opmBareRE matches the number at the start of a problem of a paper of the OPM: "4." at the start of a line.
	opmBareRE = regexp.MustCompile(`(?m)^[ \t]*([1-9])\.\s`)
)

// OPMPapers gives the papers of the final of category B of the Portuguese Mathematical Olympiad (pt-opm) on the archive page of olimpiadas.spm.pt. The final has two days of three problems: the paper day1 has the problems 1 to 3 and the paper day2 the problems 4 to 6. A file for both days gives a paper with no id. The year of an olympiad is the year in which its school year ends, which is the year of the final.
func OPMPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range opmLinkRE.FindAllStringSubmatch(page, -1) {
		f := opmFinalRE.FindStringSubmatch(m[3])
		if f == nil {
			continue
		}
		y, _ := strconv.Atoi(m[2])
		paper := ""
		if day := f[1] + f[2] + f[3] + f[4]; day != "" {
			paper = "day" + day
		}
		u := "https://olimpiadas.spm.pt" + strings.ReplaceAll(m[1], " ", "%20")
		if key := strconv.Itoa(y+1) + paper; !seen[key] {
			seen[key] = true
			out = append(out, National{Competition: "pt-opm", Year: y + 1, Paper: paper, URL: u})
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

// OPMPart gives the problems of a paper of the OPM, with the word Problem before their numbers. The papers number the problems with no word, and the second day starts at 4.
func OPMPart(text, _ string) string { return numberRun(text, opmBareRE) }

// MOFirst is the first olympiad of the Czech Mathematical Olympiad that has a page of its papers on matematickaolympiada.cz.
const MOFirst = 43

// MOListings gives the page of each olympiad of the Czech Mathematical Olympiad from the 43rd to the olympiad that starts in the year last. The site has no page for the 45th and the 46th olympiad.
func MOListings(last int) []string {
	var out []string
	for n := MOFirst; n <= last-1950; n++ {
		if n == 45 || n == 46 {
			continue
		}
		out = append(out, "https://www.matematickaolympiada.cz/mo-pro-ss/rocnik/"+strconv.Itoa(n)+"-rocnik")
	}
	return out
}

var (
	// moFinalRE matches the link to the file of the final round of category A of one olympiad, with the problems and their solutions in Czech: a57iii.pdf is the third round of the 57th olympiad, which was the final. From the 73rd olympiad the final is the fourth round, a73iv.pdf, and a73iii.pdf is the regional round. The finals of the 68th and 69th olympiads are named a68iii-web.pdf and a69iii-web.pdf.
	moFinalRE = regexp.MustCompile(`href="(/media/[0-9]+/a([0-9]{2})(iii|iv)(?:-web)?\.pdf)"`)
	// moNumberRE matches the number at the start of a problem: "3." at the start of a line, with at most two spaces before it. A numbered list in a solution has more.
	moNumberRE = regexp.MustCompile(`^\f*[ \t]{0,2}([1-9])\.\s`)
	// moSkipRE matches a line that starts a part of the file with no problem: a solution, or the header of a page.
	moSkipRE = regexp.MustCompile(`^[ \t\f]*(?:Řešení|[0-9]{2}\. ročník)`)
)

// MOPapers gives the paper of the final round of category A of each olympiad on the pages of matematickaolympiada.cz. That round is the national olympiad of the Czech Republic (cz-sk-mo). The nth olympiad ends in the year 1951 + n, which is the year of the final. A page with the fourth round gives that file, and a page with no fourth round gives the third round.
func MOPapers(page string) []National {
	best := map[int]string{}
	for _, m := range moFinalRE.FindAllStringSubmatch(page, -1) {
		n, _ := strconv.Atoi(m[2])
		if m[3] == "iv" || best[n] == "" {
			best[n] = "https://www.matematickaolympiada.cz" + m[1]
		}
	}
	var out []National
	for n, u := range best {
		out = append(out, National{Competition: "cz-sk-mo", Year: 1951 + n, URL: u})
	}
	slices.SortFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}

// MOPart gives the problems of a file of the final of the Czech Mathematical Olympiad, with the word Problem before their numbers. The file numbers the problems with no word, and it has the solutions, which can have numbered lists, and a header on each page. A file of a late olympiad states the six problems first and then states each one again with its solution. So a line with the next number and little indent starts a problem, and a solution or a page header hides the text up to the line with the next number.
func MOPart(text, _ string) string {
	var b strings.Builder
	next, skip := 1, false
	for _, line := range strings.SplitAfter(text, "\n") {
		if m := moNumberRE.FindStringSubmatchIndex(line); m != nil {
			if n, _ := strconv.Atoi(line[m[2]:m[3]]); n == next {
				b.WriteString(line[:m[2]] + "Problem " + line[m[2]:])
				next, skip = n+1, false
				continue
			}
		}
		if moSkipRE.MatchString(line) {
			skip = true
		}
		if !skip {
			b.WriteString(line)
		}
	}
	return b.String()
}

// ELMOListing is the page of the ELMO site that lists the papers of each year.
const ELMOListing = "https://elmo.evanchen.cc/problems.html"

// elmoRE matches the link to the paper of one year of the ELMO: ELMO-2019.pdf. The files of the shortlist, the solutions, the ELSMO and the awards have other names.
var elmoRE = regexp.MustCompile(`href="(https://web\.evanchen\.cc/exams/ELMO-((?:19|20)[0-9]{2})\.pdf)"`)

// ELMOPapers gives the paper of each year of the ELMO (elmo) on the problems page of its site. The paper holds the six problems of the two days, numbered 1 to 6. A year whose paper is only an attachment on AoPS gives no paper.
func ELMOPapers(page string) []National {
	var out []National
	seen := map[int]bool{}
	for _, m := range elmoRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[2])
		if seen[y] {
			continue
		}
		seen[y] = true
		out = append(out, National{Competition: "elmo", Year: y, URL: m[1]})
	}
	slices.SortFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}

// DMSListing is the page of the Mathematical Society of Serbia (DMS) that lists the papers of its secondary school competitions.
const DMSListing = "https://dms.rs/matematika-srednje-skole/"

// dmsSMORE matches the link to the paper of one year of the Serbian Mathematical Olympiad: zadaci_smo_2012.pdf, zadaci-smo-2014.pdf, zadaci_i_resenja_smo_2016.pdf, 2017_smo.pdf, SMO2019.pdf, SMO_2022_ZADACI.pdf or SMO_2023_SVE.pdf. The files of the solutions (resenja) and the results (rezultati) have other names.
var dmsSMORE = regexp.MustCompile(`href="(https://dms\.rs/wp-content/uploads/[0-9]{4}/[0-9]{2}/(?:zadaci(?:_i_resenja)?[_-]smo[_-]((?:19|20)[0-9]{2})(?:-1)?|((?:19|20)[0-9]{2})_smo|SMO((?:19|20)[0-9]{2})|SMO_((?:19|20)[0-9]{2})_(?:ZADACI|SVE))\.pdf)"`)

// DMSPapers gives the paper of each year of the Serbian Mathematical Olympiad (rs-smo) on the secondary school page of the DMS. The paper holds the six problems of the two days, numbered 1 to 6, in Serbian. The paper of 2016 also holds the solutions after the problems.
func DMSPapers(page string) []National {
	var out []National
	seen := map[int]bool{}
	for _, m := range dmsSMORE.FindAllStringSubmatch(page, -1) {
		y := 0
		for _, g := range m[2:] {
			if g != "" {
				y, _ = strconv.Atoi(g)
			}
		}
		if y == 0 || seen[y] {
			continue
		}
		seen[y] = true
		out = append(out, National{Competition: "rs-smo", Year: y, URL: m[1]})
	}
	slices.SortFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}

var (
	// awbLinkRE matches a link to a file of the German IMO selection on the listing page of the Bundeswettbewerb Mathematik.
	awbLinkRE = regexp.MustCompile(`href="(/fileadmin/Mathe-Wettbewerbe/AIMO/Aufgaben_und_Loesungen_AIMO/[^"?#]+\.pdf)"`)
	// awbNameRE matches the name of a file with the problems of the selection of one year: aufgaben_awb_05.pdf or 2022_IMO_Auswahlklausuren_HP.pdf. The files of the solutions (loes_awkl, Lsg) have other names.
	awbNameRE = regexp.MustCompile(`^(?:aufgaben_awb_([0-9]{2})|((?:19|20)[0-9]{2})_IMO_Auswahlklausuren(?:_HP)?)\.pdf$`)
)

// AWBPapers gives the paper of each year of the German IMO selection (de-tst) on the listing page of the Bundeswettbewerb Mathematik, which is also the page of the selection. The year of a paper is the year of the IMO for which the selection was held: aufgaben_awb_05.pdf is the selection for the IMO 2005. One file holds all the tests (Auswahlklausuren) of its year, and each test numbers its problems from 1.
func AWBPapers(page string) []National {
	var out []National
	seen := map[int]bool{}
	for _, m := range awbLinkRE.FindAllStringSubmatch(page, -1) {
		n := awbNameRE.FindStringSubmatch(m[1][strings.LastIndex(m[1], "/")+1:])
		if n == nil {
			continue
		}
		y, _ := strconv.Atoi(n[1] + n[2])
		if n[1] != "" {
			y += 2000
		}
		if seen[y] {
			continue
		}
		seen[y] = true
		out = append(out, National{Competition: "de-tst", Year: y, URL: "https://www.mathe-wettbewerbe.de" + m[1]})
	}
	slices.SortFunc(out, func(a, b National) int { return a.Year - b.Year })
	return out
}
