package official

import (
	"regexp"
	"strconv"
	"strings"
)

// chiuchangPages is the address of the pages of past problems of the International Mathematics Competition (IMC) on chiuchang.org.
const chiuchangPages = "https://chiuchang.org/imc/en/%e6%ad%b7%e5%b1%86%e8%a9%a6%e9%a1%8c/"

// ChiuchangListings are the pages of the IMC archive of chiuchang.org that om papers reads: the page of each year from 1999 to 2019 and from 2021 to 2026, as the home page https://chiuchang.org/imc/en/home-en/ links them. 2020 has no page.
var ChiuchangListings = []string{
	"https://chiuchang.org/imc/en/2026/07/29/2026-problems/",
	"https://chiuchang.org/imc/en/2025/08/27/2025-problems/",
	"https://chiuchang.org/imc/en/2024/08/13/2024-problems/",
	"https://chiuchang.org/imc/en/2023/07/26/2023-problems/",
	"https://chiuchang.org/imc/en/2022/07/19/2022-problems/",
	"https://chiuchang.org/imc/en/2021/08/02/2021/",
	"https://chiuchang.org/imc/en/2019-2/",
	chiuchangPages + "2018-2/",
	chiuchangPages + "2017-en/",
	chiuchangPages + "2016-en/",
	chiuchangPages + "2015-en/",
	chiuchangPages + "2014-en/",
	chiuchangPages + "2013-en/",
	chiuchangPages + "2012-en/",
	chiuchangPages + "2011-2/",
	chiuchangPages + "2010-2/",
	chiuchangPages + "2009-2/",
	chiuchangPages + "2008-2/",
	chiuchangPages + "2007-2/",
	chiuchangPages + "2006-2/",
	chiuchangPages + "2005-2/",
	chiuchangPages + "2004-2/",
	chiuchangPages + "2003-2/",
	chiuchangPages + "2002-2/",
	chiuchangPages + "2001-2/",
	chiuchangPages + "2000-2/",
	chiuchangPages + "1999-2/",
}

// chiuchangRE matches the link to one file on a page of the archive, whole or from the root of the site: https://chiuchang.org/imc/wp-content/uploads/sites/2/2024/08/InIMC-2024_Keystage-3_Individual.x17381.pdf or /wp-content/uploads/sites/2/2018/02/2005-PEMIC-Team.x17381.pdf. Both forms give the same file.
var chiuchangRE = regexp.MustCompile(`href="(?:https://chiuchang\.org/imc)?(/wp-content/uploads/sites/2/[0-9]{4}/[0-9]{2}/([^"/]+\.pdf))"`)

// chiuchangYearRE is the year of the IMC in the name of a file: 2005-PEMIC-Team, 2017InIMC_Keystage_II_Team_ENG or VIMC-2025_Keystage-3_Team_Final.
var chiuchangYearRE = regexp.MustCompile(`(?:^|[^0-9])((?:19|20)[0-9]{2})(?:[^0-9]|$)`)

// ChiuchangPapers gives the individual and team papers of the IMC on the pages of the archive. A paper of Keystage III, or of the IWYMIC before 2015, is of the iwymic, and a paper of Keystage II, or of the primary contest before 2015 (EMIC, PEMIC, TEMIC, KEMIC, HEMIC or INEMIC), is of the emic. The individual file of the iwymic holds the papers section-a and section-b, so it gives two papers. The files of the answers and the solutions are not papers. The iwymic of 1999 to 2004 has no shape in the register, so its files are left out.
func ChiuchangPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range chiuchangRE.FindAllStringSubmatch(page, -1) {
		name := strings.ToLower(m[2])
		ym := chiuchangYearRE.FindStringSubmatch(name)
		if ym == nil || strings.Contains(name, "answer") || strings.Contains(name, "sol") {
			continue
		}
		y, _ := strconv.Atoi(ym[1])
		comp := ""
		switch {
		case strings.Contains(name, "keystage-3") || strings.Contains(name, "keystage_iii") || strings.Contains(name, "iwymic"):
			comp = "iwymic"
		case strings.Contains(name, "keystage-2") || strings.Contains(name, "keystage_ii") || strings.Contains(name, "emic"):
			comp = "emic"
		}
		var papers []string
		switch {
		case strings.Contains(name, "team"):
			papers = []string{"team"}
		case strings.Contains(name, "individual") && comp == "iwymic":
			papers = []string{"section-a", "section-b"}
		case strings.Contains(name, "individual"):
			papers = []string{"individual"}
		}
		u := "https://chiuchang.org/imc" + m[1]
		if comp == "" || papers == nil || (comp == "iwymic" && y <= 2004) || seen[u] {
			continue
		}
		seen[u] = true
		for _, p := range papers {
			out = append(out, National{Competition: comp, Year: y, Paper: p, URL: u})
		}
	}
	return out
}
