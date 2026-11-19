package official

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// MathHourListings are the pages of the archive of the Math Hour Olympiad of the University of Washington that om papers reads: the page of each year from 2012 to 2025. The olympiad was cancelled in 2020 and 2021, the page of 2010 has no paper file and the page of 2011 has none either, so those years are not read.
var MathHourListings = mathHourListings()

// mathHourArchive is the address of the archive of the Math Hour Olympiad.
const mathHourArchive = "https://sites.math.washington.edu/~mathcircle/olympiad/archive/"

func mathHourListings() []string {
	var out []string
	for y := 2012; y <= 2025; y++ {
		if y == 2020 || y == 2021 {
			continue
		}
		out = append(out, fmt.Sprintf("%solympiad_%d/", mathHourArchive, y))
	}
	return out
}

// mathHourRE matches the link to one file on a page of the archive. The links are relative to the page of the year, and the name starts with the year in two or four digits: ./MHO2012_5-7_1-5.pdf, 19main67.pdf, ./2025-810-extra.pdf, 13adv57.pdf or 22problems.pdf.
var mathHourRE = regexp.MustCompile(`href="(?:\./)?((?:MHO)?(?:20)?([0-9]{2})[^"/]*\.pdf)"`)

// MathHourPapers gives the files of the Math Hour Olympiad on the pages of the archive. Each level of a year has a file of the five main problems and a file of the two extra problems, both of the paper of the level: grades8-10 when the name has 810 or 8-10, and grades6-7 when it has 67, 57 or 5-7. A file with no level, as the one file of 2022, holds all the positions of the edition.
func MathHourPapers(page string) []National {
	var out []National
	seen := map[string]bool{}
	for _, m := range mathHourRE.FindAllStringSubmatch(page, -1) {
		y, _ := strconv.Atoi(m[2])
		y += 2000
		name := strings.TrimPrefix(m[1], "MHO")
		name = strings.TrimPrefix(name, strconv.Itoa(y))
		name = strings.TrimPrefix(name, m[2])
		paper := ""
		switch {
		case strings.Contains(name, "810") || strings.Contains(name, "8-10"):
			paper = "grades8-10"
		case strings.Contains(name, "67") || strings.Contains(name, "57") || strings.Contains(name, "5-7"):
			paper = "grades6-7"
		}
		u := fmt.Sprintf("%solympiad_%d/%s", mathHourArchive, y, m[1])
		if seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, National{Competition: "math-hour", Year: y, Paper: paper, URL: u})
	}
	return out
}
