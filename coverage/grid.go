package coverage

import (
	"slices"
	"strconv"

	"github.com/tamnd/olymmath-agent/content"
)

// The measures of spec 14.2, as indexes of Count.Reached.
const (
	C0       = iota // a problem record
	C1              // a canonical statement of class H1 or H2
	C2              // a canonical statement of class H1
	C3              // a statement with status verified
	C4              // a human or official solution, shown or linked
	C5              // a solution graded 7/7
	C6              // an answer checked against an official answer
	C7              // a formal statement checked by two persons
	C8              // a checked proof of such a formal statement
	Measures        // the number of measures
)

// Names are the short names of the measures.
var Names = [Measures]string{"positions", "human text", "official", "verified", "human solution", "graded", "answer", "formal", "proved"}

// Recorded tells which measures the content has records for. The content has no grades, no checked answers and no formal statements yet, so C5 to C8 are not measured.
var Recorded = [Measures]bool{C0: true, C1: true, C2: true, C3: true, C4: true}

// Cell is one position of an edition in the year grid.
type Cell struct {
	Position string
	// Problem is nil when the content has no problem at the position.
	Problem *content.Problem
	// Class is the class of the canonical statement, or "" when there is no problem.
	Class string
	// Human is true when the problem has a human or official solution. AI is true when it has a model solution.
	Human, AI bool
	// Counted is false for a position that the shape of the edition does not have. Such a cell shows, but it is not in the measures.
	Counted bool
}

// Row is one year of a competition in the year grid. Edition is nil when the content has no edition for the year.
type Row struct {
	Year    int
	Edition *content.Edition
	Cells   []Cell
	Count   Count
}

// Count gives, for each measure, the number of counted positions that reach it.
type Count struct {
	Expected int
	Reached  [Measures]int
	// Shape is false when the register gives no positions. Then the problems of the content are the expected positions, and C0 is not measured.
	Shape bool
}

// Add adds another count to the count.
func (c *Count) Add(o Count) {
	c.Expected += o.Expected
	for i := range c.Reached {
		c.Reached[i] += o.Reached[i]
	}
	c.Shape = c.Shape || o.Shape
}

// Percent gives the share of measure m as a whole percent, for example "98%", or "-" when the measure has no value.
func (c Count) Percent(m int) string {
	if !Recorded[m] || c.Expected == 0 || (m == C0 && !c.Shape) {
		return "-"
	}
	return strconv.Itoa(100*c.Reached[m]/c.Expected) + "%"
}

// Grid gives the year grid of a competition, the newest year first, and the total count. The years go from the first year of the register to thisYear, or to the last year, with no gap year, and include each edition of the content.
func Grid(c *content.Competition, thisYear int) ([]Row, Count) {
	usual := shape(c)
	editions := map[int]*content.Edition{}
	years := map[int]bool{}
	for _, e := range c.Editions {
		if _, ok := editions[e.Year]; !ok {
			editions[e.Year] = e
		}
		years[e.Year] = true
	}
	if c.FirstYear > 0 && len(usual) > 0 {
		last := thisYear
		if c.LastYear > 0 {
			last = c.LastYear
		}
		for y := c.FirstYear; y <= last; y++ {
			years[y] = !slices.Contains(c.Gaps, y)
		}
	}
	for _, e := range c.Editions {
		years[e.Year] = true
	}
	ys := make([]int, 0, len(years))
	for y, ok := range years {
		if ok {
			ys = append(ys, y)
		}
	}
	slices.Sort(ys)
	slices.Reverse(ys)
	var rows []Row
	var total Count
	for _, y := range ys {
		r := row(y, editions[y], usual)
		rows = append(rows, r)
		total.Add(r.Count)
	}
	return rows, total
}

func row(year int, e *content.Edition, usual []string) Row {
	r := Row{Year: year, Edition: e}
	expected := usual
	at := map[string]*content.Problem{}
	if e != nil {
		if ps := e.Positions(); len(ps) > 0 {
			expected = ps
		}
		for _, p := range e.Problems {
			at[p.Position] = p
		}
		for _, p := range e.Also {
			if pos := p.PositionIn(e); pos != "" && at[pos] == nil {
				at[pos] = p
			}
		}
	}
	r.Count.Shape = len(expected) > 0
	set := map[string]bool{}
	for _, p := range expected {
		set[p] = true
	}
	var extra []string
	for p := range at {
		if !set[p] {
			extra = append(extra, p)
		}
	}
	slices.SortFunc(extra, comparePos)
	for _, pos := range append(slices.Clone(expected), extra...) {
		c := cell(pos, at[pos])
		c.Counted = set[pos] || !r.Count.Shape
		r.Cells = append(r.Cells, c)
		if !c.Counted {
			continue
		}
		r.Count.Expected++
		for m, ok := range c.reached() {
			if ok {
				r.Count.Reached[m]++
			}
		}
	}
	return r
}

func cell(pos string, p *content.Problem) Cell {
	c := Cell{Position: pos, Problem: p}
	if p == nil {
		return c
	}
	if s := p.CanonicalStatement(); s != nil {
		c.Class = p.Class(s)
	}
	for _, s := range p.Solutions {
		switch s.Origin {
		case "human", "official", "human+ai":
			c.Human = true
		case "ai":
			c.AI = true
		}
	}
	return c
}

// reached tells which measures the cell reaches.
func (c Cell) reached() [Measures]bool {
	var r [Measures]bool
	if c.Problem == nil {
		return r
	}
	r[C0] = true
	r[C1] = c.Class == "H1" || c.Class == "H2"
	r[C2] = c.Class == "H1"
	r[C3] = c.Problem.Status.Statement == "verified"
	r[C4] = c.Human
	return r
}
