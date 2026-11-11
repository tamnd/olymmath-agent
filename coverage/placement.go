package coverage

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/locator"
)

// Expected gives the positions that an edition expects: the positions of the papers of its edition file when the file lists them, or else the positions of the shape of the year in the register. An edition file lists its papers when the edition has more or fewer problems than its era, as a year with an extra problem or with a number that was not used. Expected gives nil when neither gives positions.
func Expected(c *content.Competition, year int, e *content.Edition) []string {
	if e != nil {
		if ps := e.Positions(); len(ps) > 0 {
			return ps
		}
	}
	return c.Shape(year).Positions()
}

// lastYear gives the last year of the register range of a competition: its last year, or thisYear when it still runs.
func lastYear(c *content.Competition, thisYear int) int {
	if c.LastYear > 0 {
		return c.LastYear
	}
	return thisYear
}

// Unexpected tells why the register says that the appearance l of the competition c cannot exist, or gives "" when the register expects it or cannot tell. e is the edition of l in the content, or nil.
//
// The appearance cannot exist when its year is a gap of the register, before the first year, after the last year or after thisYear, when its edition is not a paper that the register gives for the year, or when its position is not one of the positions that Expected gives. A competition with no shape for the year accepts any position.
func Unexpected(c *content.Competition, e *content.Edition, l locator.Locator, thisYear int) string {
	y := editionYear(l.Edition)
	if y == 0 {
		return ""
	}
	switch {
	case slices.Contains(c.Gaps, y):
		return fmt.Sprintf("the register marks %d as a year with no %s", y, c.Code)
	case c.FirstYear > 0 && y < c.FirstYear:
		return fmt.Sprintf("%d is before the first year %d of %s", y, c.FirstYear, c.Code)
	case y > lastYear(c, thisYear):
		return fmt.Sprintf("%d is after the last year %d of %s", y, lastYear(c, thisYear), c.Code)
	}
	switch ids := c.EditionIDs(y); {
	case len(c.Papers) > 0:
		if !slices.Contains(ids, l.Edition) {
			return fmt.Sprintf("the register gives the papers %s for %d, and %s is not one of them", strings.Join(ids, ", "), y, l.Edition)
		}
	case c.EditionForm == "year-season":
		// The register does not list the rounds of a year.
	case l.Edition != strconv.Itoa(y):
		return fmt.Sprintf("the register gives one edition a year, %d, and no edition %s", y, l.Edition)
	}
	ps := Expected(c, y, e)
	if len(ps) == 0 || slices.Contains(ps, l.Position) {
		return ""
	}
	return fmt.Sprintf("%s/%s has the positions %s, and %s is not one of them", c.Code, l.Edition, span(ps), l.Position)
}

// span writes a list of positions in short form: p1 to p6 when the list is p1, p2 and so on with no gap, or else the list.
func span(ps []string) string {
	run := true
	for i, p := range ps {
		if p != "p"+strconv.Itoa(i+1) {
			run = false
			break
		}
	}
	if run && len(ps) > 2 {
		return ps[0] + " to " + ps[len(ps)-1]
	}
	return strings.Join(ps, ", ")
}

// editionYear gives the year at the start of an edition id, for example 2024 for 2024, 2021a or 2021-fall-a, or 0.
func editionYear(id string) int {
	if len(id) < 4 {
		return 0
	}
	y, err := strconv.Atoi(id[:4])
	if err != nil {
		return 0
	}
	return y
}
