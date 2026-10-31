// Package coverage measures how much of the closed universe of the register olymmath holds. Spec 14 defines the measures. This package computes C0 to C2 from the content and, for each expected position, tells if a seed row is a candidate for it.
package coverage

import (
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/seed"
)

// Report is the coverage report.
type Report struct {
	ThisYear     int       `json:"this_year"`
	Totals       Totals    `json:"totals"`
	Competitions []*Comp   `json:"competitions"`
	Sources      []*Source `json:"sources"`
	Unknown      []*Label  `json:"unknown"`
	Aliases      int       `json:"aliases"`
}

// Totals sums the measures of all competitions.
type Totals struct {
	Expected    int `json:"expected"`
	C0          int `json:"c0"`
	C1          int `json:"c1"`
	C2          int `json:"c2"`
	Candidate   int `json:"candidate"`
	NoCandidate int `json:"no_candidate"`
	Rows        int `json:"rows"`
}

// Comp is the coverage of one competition.
type Comp struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Group string `json:"group"`
	Totals
	// Shape is false when the register gives no positions for the competition. Then Expected is 0 and the year list shows only the positions that the seed rows give.
	Shape bool    `json:"shape"`
	Years []*Year `json:"years,omitempty"`
}

// Year lists the positions of one edition by state.
type Year struct {
	Year int `json:"year"`
	// Record are the positions with a problem record. Candidate are the positions with no record and with a seed candidate. Missing are the expected positions with neither.
	Record    []string `json:"record,omitempty"`
	Candidate []string `json:"candidate,omitempty"`
	Missing   []string `json:"missing,omitempty"`
	// Rows counts the seed rows of this year, with or without a position.
	Rows int `json:"rows"`
}

// Source counts the seed rows of one source.
type Source struct {
	ID         string `json:"id"`
	Rows       int    `json:"rows"`
	Located    int    `json:"located"`
	Positioned int    `json:"positioned"`
	Excluded   int    `json:"excluded"`
}

// Label is a competition label that the alias table does not resolve.
type Label struct {
	Source string `json:"source"`
	Label  string `json:"label"`
	Rows   int    `json:"rows"`
}

// Build makes the report. rows walks every seed row.
func Build(corpus *content.Corpus, al *seed.Aliases, thisYear int, rows func(fn func(seed.Row) error) error) (*Report, error) {
	rep := &Report{ThisYear: thisYear, Aliases: al.Len()}
	// cand[code][year][pos] is true for a position with a seed candidate. seen[code][year] counts the rows of the year.
	cand := map[string]map[int]map[string]bool{}
	seen := map[string]map[int]int{}
	compRows := map[string]int{}
	srcs := map[string]*Source{}
	unknown := map[[2]string]int{}
	err := rows(func(r seed.Row) error {
		s := srcs[r.Source]
		if s == nil {
			s = &Source{ID: r.Source}
			srcs[r.Source] = s
		}
		s.Rows++
		code := al.Code(r.Labels)
		switch code {
		case "":
			l := ""
			if len(r.Labels) > 0 {
				l = r.Labels[0]
			}
			unknown[[2]string{r.Source, l}]++
			return nil
		case "-":
			s.Excluded++
			return nil
		}
		s.Located++
		compRows[code]++
		if r.Year == 0 {
			return nil
		}
		if seen[code] == nil {
			seen[code] = map[int]int{}
		}
		seen[code][r.Year]++
		pos := seed.Position(r.Number)
		if pos == "" {
			return nil
		}
		s.Positioned++
		if cand[code] == nil {
			cand[code] = map[int]map[string]bool{}
		}
		if cand[code][r.Year] == nil {
			cand[code][r.Year] = map[string]bool{}
		}
		cand[code][r.Year][pos] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, c := range corpus.Competitions {
		rep.Competitions = append(rep.Competitions, measure(c, cand[c.Code], seen[c.Code], compRows[c.Code], thisYear))
	}
	for _, c := range rep.Competitions {
		t := &rep.Totals
		t.Expected += c.Expected
		t.C0 += c.C0
		t.C1 += c.C1
		t.C2 += c.C2
		t.Candidate += c.Candidate
		t.NoCandidate += c.NoCandidate
		t.Rows += c.Rows
	}
	for _, s := range srcs {
		rep.Sources = append(rep.Sources, s)
	}
	slices.SortFunc(rep.Sources, func(a, b *Source) int { return strings.Compare(a.ID, b.ID) })
	for k, n := range unknown {
		rep.Unknown = append(rep.Unknown, &Label{Source: k[0], Label: k[1], Rows: n})
	}
	slices.SortFunc(rep.Unknown, func(a, b *Label) int {
		return cmp.Or(b.Rows-a.Rows, strings.Compare(a.Source, b.Source), strings.Compare(a.Label, b.Label))
	})
	return rep, nil
}

// shape gives the usual positions of an edition, from the layout or from the number of problems for each year.
func shape(c *content.Competition) []string {
	var ps []string
	for _, l := range c.Layout {
		ps = append(ps, l.Positions...)
	}
	if len(ps) == 0 && c.PerYear > 0 {
		for i := 1; i <= c.PerYear; i++ {
			ps = append(ps, "p"+strconv.Itoa(i))
		}
	}
	return ps
}

func measure(c *content.Competition, cand map[int]map[string]bool, seen map[int]int, rows, thisYear int) *Comp {
	out := &Comp{Code: c.Code, Name: c.Name, Group: c.Group}
	out.Rows = rows
	usual := shape(c)
	out.Shape = len(usual) > 0
	editions := map[int]*content.Edition{}
	for _, e := range c.Editions {
		if _, ok := editions[e.Year]; !ok {
			editions[e.Year] = e
		}
	}
	// The years: from the first year to the last year, or to this year. With no first year, the years that the content or the seed rows give.
	years := map[int]bool{}
	if c.FirstYear > 0 {
		last := thisYear
		if c.LastYear > 0 {
			last = c.LastYear
		}
		for y := c.FirstYear; y <= last; y++ {
			if !slices.Contains(c.Gaps, y) {
				years[y] = true
			}
		}
	}
	for y := range editions {
		years[y] = true
	}
	for y := range seen {
		years[y] = true
	}
	ys := make([]int, 0, len(years))
	for y := range years {
		ys = append(ys, y)
	}
	slices.Sort(ys)
	for _, y := range ys {
		yr := &Year{Year: y, Rows: seen[y]}
		expected := usual
		have := map[string]*content.Problem{}
		if e := editions[y]; e != nil {
			if ps := e.Positions(); len(ps) > 0 {
				expected = ps
			}
			for _, p := range e.Problems {
				have[p.Position] = p
			}
		}
		set := map[string]bool{}
		for _, p := range expected {
			set[p] = true
		}
		// A position with a record or a candidate that the shape does not have still shows, so nothing found is hidden.
		var extra []string
		for p := range have {
			if !set[p] {
				extra = append(extra, p)
			}
		}
		for p := range cand[y] {
			if !set[p] && have[p] == nil {
				extra = append(extra, p)
			}
		}
		slices.SortFunc(extra, comparePos)
		for _, p := range append(slices.Clone(expected), extra...) {
			counted := set[p]
			if counted {
				out.Expected++
			}
			if pr := have[p]; pr != nil {
				yr.Record = append(yr.Record, p)
				if counted {
					out.C0++
					if s := pr.CanonicalStatement(); s != nil {
						if c := pr.Class(s); c == "H1" || c == "H2" {
							out.C1++
						}
						if pr.Class(s) == "H1" {
							out.C2++
						}
					}
				}
				continue
			}
			if cand[y][p] {
				yr.Candidate = append(yr.Candidate, p)
				if counted {
					out.Candidate++
				}
				continue
			}
			if counted {
				yr.Missing = append(yr.Missing, p)
				out.NoCandidate++
			}
		}
		out.Years = append(out.Years, yr)
	}
	return out
}

// comparePos orders positions by letter and then by number: p2 before p10.
func comparePos(a, b string) int {
	la, na := splitPos(a)
	lb, nb := splitPos(b)
	return cmp.Or(strings.Compare(la, lb), na-nb, strings.Compare(a, b))
}

func splitPos(p string) (string, int) {
	i := strings.IndexFunc(p, func(r rune) bool { return r >= '0' && r <= '9' })
	if i < 0 {
		return p, 0
	}
	n, _ := strconv.Atoi(p[i:])
	return p[:i], n
}

// JSON writes the report as indented JSON.
func (r *Report) JSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(r)
}

func pct(n, d int) string {
	if d == 0 {
		return "-"
	}
	return strconv.FormatFloat(100*float64(n)/float64(d), 'f', 1, 64) + "%"
}

// Markdown writes the summary of the report. The full lists of positions are in the JSON form.
func (r *Report) Markdown(w io.Writer, maxUnknown int) error {
	p := func(format string, a ...any) { fmt.Fprintf(w, format, a...) }
	t := r.Totals
	p("# Coverage\n\n")
	p("This report is made by `om coverage`. The register gives the expected positions of each competition. C0 is the share of expected positions with a problem record. C1 is the share with a canonical statement of class H1 or H2. C2 is the share with an H1 statement. A candidate is an expected position with no record and with at least one seed row from a wave 0 dataset. Spec 14 gives the definitions.\n\n")
	p("## Totals\n\n")
	p("| Expected | C0 | C1 | C2 | Candidate | No candidate | Located seed rows |\n|---:|---:|---:|---:|---:|---:|---:|\n")
	p("| %d | %s | %s | %s | %d (%s) | %d (%s) | %d |\n\n", t.Expected, pct(t.C0, t.Expected), pct(t.C1, t.Expected), pct(t.C2, t.Expected), t.Candidate, pct(t.Candidate, t.Expected), t.NoCandidate, pct(t.NoCandidate, t.Expected), t.Rows)
	p("## Sources\n\n| Source | Rows | Located | With a position | Not a competition |\n|---|---:|---:|---:|---:|\n")
	for _, s := range r.Sources {
		p("| %s | %d | %d (%s) | %d | %d |\n", s.ID, s.Rows, s.Located, pct(s.Located, s.Rows), s.Positioned, s.Excluded)
	}
	p("\n## Competitions\n\nThe order is the order of the register. A competition with no shape in the register has no expected positions yet.\n\n")
	p("| Code | Competition | Expected | C0 | C1 | C2 | Candidate | No candidate | Seed rows |\n|---|---|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, c := range r.Competitions {
		exp := strconv.Itoa(c.Expected)
		if !c.Shape {
			exp = "no shape"
		}
		p("| %s | %s | %s | %s | %s | %s | %d | %d | %d |\n", c.Code, c.Name, exp, pct(c.C0, c.Expected), pct(c.C1, c.Expected), pct(c.C2, c.Expected), c.Candidate, c.NoCandidate, c.Rows)
	}
	total := 0
	for _, u := range r.Unknown {
		total += u.Rows
	}
	p("\n## Unknown competitions\n\nThese labels do not resolve through the alias table (%d keys). There are %d labels with %d rows. The full list is in `unknown-competitions.tsv`. To resolve a label, add a line to `registers/competition-aliases.tsv`, or add the competition to the register.\n\n", r.Aliases, len(r.Unknown), total)
	p("| Source | Label | Rows |\n|---|---|---:|\n")
	for i, u := range r.Unknown {
		if i >= maxUnknown {
			break
		}
		p("| %s | %s | %d |\n", u.Source, strings.ReplaceAll(u.Label, "|", "\\|"), u.Rows)
	}
	return nil
}

// UnknownTSV writes the unknown labels, one for each line: source, label and rows.
func (r *Report) UnknownTSV(w io.Writer) error {
	if _, err := fmt.Fprintln(w, "source\tlabel\trows"); err != nil {
		return err
	}
	for _, u := range r.Unknown {
		l := strings.NewReplacer("\t", " ", "\n", " ").Replace(u.Label)
		if _, err := fmt.Fprintf(w, "%s\t%s\t%d\n", u.Source, l, u.Rows); err != nil {
			return err
		}
	}
	return nil
}
