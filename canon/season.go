package canon

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/seed"
)

var (
	// seasonWordRE finds the season of an edition in a title, such as "Fall" in "Tournament of Towns Fall 2002 Junior A Level P5". Autumn is the same season as fall.
	seasonWordRE = regexp.MustCompile(`(?i)\b(fall|autumn|spring)\b`)
	// tierRE finds the tier of a paper in a title, such as "Junior" or "Seniors".
	tierRE = regexp.MustCompile(`(?i)\b(junior|senior)s?\b`)
	// levelRE finds the level of a paper in a title, such as "A Level", "O-level", "Tour A" or "0 Level", where the digit 0 stands for the letter O.
	levelRE = regexp.MustCompile(`(?i)\b(?:([ao0])\s*-?\s*level|tour\s+([ao]))\b`)
	// otherLevelRE finds a word that names a paper that has no position of the form above, such as the oral round or the P level.
	otherLevelRE = regexp.MustCompile(`(?i)\b(?:oral|[b-np-z]\s*-?\s*level)\b`)
	// seasonLetterRE matches the letter that Topics gives an edition with a season, such as "-fall".
	seasonLetterRE = regexp.MustCompile(`^-(?:fall|spring)$`)
)

// SeasonPosition reads the title of a problem of a competition with two editions each year, such as "Tournament of Towns Fall 2002 Junior A Level P5". It gives the year of the title, the season ("fall" or "spring"), and the position: the letter of the tier (j for junior, s for senior), the letter of the level (a or o) and the number of the problem, as sa3 for problem 3 of the senior A level paper. The title must name one year, one season, one tier and one level, and no other paper, such as the oral round or the P level. The number comes from the one problem mark of the title or from the number column, and the two must agree when the row has both. The result ok is false when the title does not give all of these, so that a row is never placed by its number alone.
func SeasonPosition(number, title string) (year int, season, pos string, ok bool) {
	title = strings.TrimSpace(title)
	if !isTitle(title) || otherLevelRE.MatchString(title) {
		return 0, "", "", false
	}
	ys := titleYearRE.FindAllString(title, -1)
	if len(ys) == 0 {
		return 0, "", "", false
	}
	for _, y := range ys[1:] {
		if y != ys[0] {
			return 0, "", "", false
		}
	}
	year, _ = strconv.Atoi(ys[0])
	season, ok = oneWord(seasonWordRE.FindAllStringSubmatch(title, -1), func(m []string) string {
		if s := strings.ToLower(m[1]); s != "autumn" {
			return s
		}
		return "fall"
	})
	if !ok {
		return 0, "", "", false
	}
	tier, ok := oneWord(tierRE.FindAllStringSubmatch(title, -1), func(m []string) string { return strings.ToLower(m[1][:1]) })
	if !ok {
		return 0, "", "", false
	}
	level, ok := oneWord(levelRE.FindAllStringSubmatch(title, -1), func(m []string) string {
		if l := strings.ToLower(m[1] + m[2]); l != "0" {
			return l
		}
		return "o"
	})
	if !ok {
		return 0, "", "", false
	}
	n := 0
	if p := seed.Position(strings.TrimRight(strings.TrimSpace(number), ".")); p != "" {
		if !strings.HasPrefix(p, "p") {
			return 0, "", "", false
		}
		n, _ = strconv.Atoi(p[1:])
	}
	if len(titleMarkRE.FindAllString(title, -1)) > 1 {
		return 0, "", "", false
	}
	if ms := titleMarkNumRE.FindAllStringSubmatch(title, -1); len(ms) == 1 {
		t, _ := strconv.Atoi(ms[0][1])
		if n != 0 && t != n {
			return 0, "", "", false
		}
		n = t
	} else if len(ms) > 1 {
		return 0, "", "", false
	}
	if n < 1 || n > 9 {
		return 0, "", "", false
	}
	return year, season, tier + level + strconv.Itoa(n), true
}

// oneWord gives the value of the matches ms when they all give the same value, and ok false when there is no match or two values differ.
func oneWord(ms [][]string, value func([]string) string) (string, bool) {
	v := ""
	for _, m := range ms {
		w := value(m)
		if v != "" && w != v {
			return "", false
		}
		v = w
	}
	return v, v != ""
}
