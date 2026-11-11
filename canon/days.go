package canon

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// unitNameRE matches the name of a paper that is one day or one test of an edition, such as "Day 2" or "Test 1".
var unitNameRE = regexp.MustCompile(`^(Day|Test) ([1-9])$`)

// unitKind gives "day" when the layout of c in the year is a list of two papers or more named "Day 1", "Day 2" and so on, and "test" for "Test 1", "Test 2" and so on. It gives "" for any other layout. Only such a layout gives the number of problems of each day, so only then can a title that counts the problems in the day or across the days be read.
func unitKind(c *content.Competition, year int) string {
	// A year with no edition, such as 1980 for the IMO, has no layout. The AoPS wiki lists other contests under such a year.
	if c == nil || slices.Contains(c.Gaps, year) {
		return ""
	}
	layout := c.Shape(year).Layout
	if len(layout) < 2 {
		return ""
	}
	kind := ""
	for i, l := range layout {
		m := unitNameRE.FindStringSubmatch(l.Name)
		if m == nil || m[2] != strconv.Itoa(i+1) || len(l.Positions) == 0 || l.Days > 1 || (kind != "" && kind != m[1]) {
			return ""
		}
		kind = m[1]
	}
	return strings.ToLower(kind)
}

var (
	// unitRE finds a day in a title, such as "Day 2", "day2", "Day II", "2nd day", "second day", or a test, such as "Test 2", "TST 3", "TST3", "Exam 2", "2nd test" or "second exam". The name "TST" alone, as in "Vietnam TST 2019", names no test. Group 1 is the word of a number after the unit, group 2 the word before it, and group 3 the unit.
	unitRE = regexp.MustCompile(`(?i)\b(?:(day|test|tst|exam)\s*([1-9]|iv|i{1,3})\b|([1-4])(?:st|nd|rd|th)\s+(day|test|exam)\b|(first|second|third|fourth)\s+(day|test|exam)\b)`)
	// unitWords gives the number of a word in a unit, such as "II" in "Day II" or "second" in "second day".
	unitWords = map[string]int{"i": 1, "ii": 2, "iii": 3, "iv": 4, "first": 1, "second": 2, "third": 3, "fourth": 4}
)

// titleUnits gives the days and the tests that a title names, such as 2 days and no test for "Vietnam TST 2019 Day 2 P1" or test 2 and day 1 for "Iran TST 2015 exam 2 day 1 problem 2". A value is 0 when the title names none, and -1 when it names two different ones.
func titleUnits(title string) (day, test int) {
	add := func(old *int, v int) {
		if *old != 0 && *old != v {
			*old = -1
		} else if *old == 0 {
			*old = v
		}
	}
	for _, m := range unitRE.FindAllStringSubmatch(title, -1) {
		unit, word := m[1], m[2]
		switch {
		case m[3] != "":
			unit, word = m[4], m[3]
		case m[5] != "":
			unit, word = m[6], m[5]
		}
		v, err := strconv.Atoi(word)
		if err != nil {
			v = unitWords[strings.ToLower(word)]
		}
		if strings.EqualFold(unit, "day") {
			add(&day, v)
		} else {
			add(&test, v)
		}
	}
	return day, test
}

// unitPlace is what a title and the number column give a row of a competition whose layout is a list of days or tests, as read by unitPosition.
type unitPlace struct {
	// pos is the position when the title and the number give exactly one.
	pos string
	// open lists the positions that the row can have when the title does not tell its day, such as p1 and p4 for a row with the number 1 and no day on a year of two days of three problems. It is nil when pos is set.
	open []string
	// running is the position of the number counted across the days, "" when there is none.
	running string
	// perDay tells that pos counts the number of the row from 1 in its day, such as the number 1 of "Vietnam TST 2019 Day 2 P4". The wiki then numbers the year by day.
	perDay bool
	// ok is false when the title gives a number or a day that no position of the layout agrees with.
	ok bool
}

// unitPosition reads the title and the number n of a row of a competition whose layout in the year is a list of days or tests of the kind unitKind gives. Agent decision: the source numbers each day from 1 in some years and the whole year in others, and the number of a title, such as "P4" in "Vietnam TST 1985 P4" or "P1" in "China Western MO 2019 Day 2 P1", also counts in the day or across the days. A position fits a number when the number is its index in its day or its index across the days. The row gets the one position of the day that the title names, or of any day when it names none, that fits both n and the number of the title. A title with no number gives no condition. When more than one position fits, the day is not known, and the row gets the list of these positions, as described at resolve. A title that names a unit of another kind, such as a test in a layout of days, gives no position.
func unitPosition(layout []content.Layout, kind, title string, year, n int) unitPlace {
	title = strings.TrimSpace(title)
	day, test := 0, 0
	t := 0
	if isTitle(title) {
		if shortlistRE.MatchString(title) || longlistRE.MatchString(title) || prelimRE.MatchString(title) {
			return unitPlace{}
		}
		for _, y := range titleYearRE.FindAllString(title, -1) {
			if y != strconv.Itoa(year) {
				return unitPlace{}
			}
		}
		day, test = titleUnits(title)
		if kind == "test" {
			day, test = test, day
		}
		if day < 0 || test != 0 {
			return unitPlace{}
		}
		// A title can give the number twice, as "Indonesia National MO 2021 P4 (INAMO 2021/4)" does. Two different numbers, as in "INAMO 2020 Day 1 P3 or APMO 2019 P1", give no position.
		marks := titleMarkNumRE.FindAllStringSubmatch(title, -1)
		if len(marks) != len(titleMarkRE.FindAllString(title, -1)) {
			return unitPlace{}
		}
		for _, m := range marks {
			v, _ := strconv.Atoi(m[1])
			if t != 0 && v != t {
				return unitPlace{}
			}
			t = v
		}
		if m := titleBareRE.FindStringSubmatch(title); t == 0 && m != nil {
			t, _ = strconv.Atoi(m[1])
		}
	}
	var fit []string
	running := ""
	i := 0
	for d, l := range layout {
		for j, p := range l.Positions {
			i++
			nums := []int{i, j + 1}
			if i == n {
				running = p
			}
			if (day == 0 || day == d+1) && slices.Contains(nums, n) && (t == 0 || slices.Contains(nums, t)) && !slices.Contains(fit, p) {
				fit = append(fit, p)
			}
		}
	}
	u := unitPlace{running: running, ok: len(fit) > 0}
	switch {
	case len(fit) == 1:
		u.pos = fit[0]
		u.perDay = fit[0] != running
	case len(fit) > 1:
		u.open = fit
	}
	return u
}

// unitYear holds the rows of one year of a competition with days or tests, for resolve.
type unitYear struct {
	// prefix is the start of the appearances of the year, such as "vn-tst/2019/".
	prefix string
	// perDay tells that the title of a row shows that the source numbers the year by day.
	perDay bool
	// numbers gives the topics of the rows with each number.
	numbers map[int]map[string]bool
	// fixed gives the topics that the title or the first line of a row gives each position, as opposed to the number alone.
	fixed map[string]map[string]bool
	// open holds the rows whose day is not known.
	open []unitRow
	// at gives the position that resolve gives each topic of open.
	at map[string]string
}

// unitRow is a row whose day is not known.
type unitRow struct {
	topic   string
	open    []string
	running string
}

// resolve gives a position to the rows of the years with days whose day is not known, once all rows are added. Agent decision: a year where no title shows that the source numbers each day from 1, and where no two topics have the same number, is numbered across the days, so such a row takes the position of its number counted across the days, as a row of any other competition does. In a year numbered by day, a title with no day does not tell the day, so the row gets a position only by elimination: when all its possible positions but one are taken by other topics, by their titles or by an earlier step of this elimination, it takes the one left. A topic that has a position from its number alone, such as a row whose title names the shortlist, takes no position away, because the number does not tell the day. It still claims the position of its number, so a row that takes the same position by elimination gives it to neither. Rows that are left one position in the same step all claim it, so two such rows of one position give it to neither. A row that is left more than one position gets none, so that in a year where no title names the day, such as "Vietnam TST 2000" for each problem, no row is placed.
func (t *Topics) resolve() {
	if t.resolved {
		return
	}
	t.resolved = true
	for _, y := range t.units {
		y.at = map[string]string{}
		restart := y.perDay
		for _, topics := range y.numbers {
			restart = restart || len(topics) > 1
		}
		if !restart {
			for _, r := range y.open {
				if r.running != "" {
					t.claim(y.prefix+r.running, r.topic, false)
					y.at[r.topic] = r.running
				}
			}
			continue
		}
		// picked gives the topics that took each position in an earlier step.
		picked := map[string]map[string]bool{}
		taken := func(p, topic string) bool {
			for _, m := range []map[string]bool{y.fixed[p], picked[p]} {
				for o := range m {
					if o != topic {
						return true
					}
				}
			}
			return false
		}
		done := make([]bool, len(y.open))
		for {
			type pick struct {
				i int
				p string
			}
			var picks []pick
			for i, r := range y.open {
				if done[i] {
					continue
				}
				var free []string
				for _, p := range r.open {
					if !taken(p, r.topic) {
						free = append(free, p)
					}
				}
				if len(free) == 1 {
					picks = append(picks, pick{i, free[0]})
				}
			}
			if len(picks) == 0 {
				break
			}
			for _, k := range picks {
				done[k.i] = true
				t.claim(y.prefix+k.p, y.open[k.i].topic, false)
				y.at[y.open[k.i].topic] = k.p
				if picked[k.p] == nil {
					picked[k.p] = map[string]bool{}
				}
				picked[k.p][y.open[k.i].topic] = true
			}
		}
	}
}
