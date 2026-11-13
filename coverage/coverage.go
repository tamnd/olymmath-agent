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

	"github.com/tamnd/olymmath-agent/canon"
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
	// Tier1 sums the national competitions of tier 1 (registers/site-tiers.yaml). It is nil when the caller gives no tier.
	Tier1 *Totals `json:"tier1_national,omitempty"`
	// Gaps are the entries of the gap register, each with the number of positions it gives the reason of.
	Gaps []*Gap `json:"gaps,omitempty"`
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
	// LinkOnly, Robots and NoSource count the expected positions with no record that the gap register gives that reason. Open counts the other expected positions with no record.
	LinkOnly int `json:"link_only"`
	Robots   int `json:"robots"`
	NoSource int `json:"no_source"`
	Open     int `json:"open"`
}

// Allowed is the number of expected positions that can have a record under the rules: the expected positions less the positions that are link-only by rule, blocked by robots.txt or with no allowed source.
func (t Totals) Allowed() int { return t.Expected - t.LinkOnly - t.Robots - t.NoSource }

func (t *Totals) add(o Totals) {
	t.Expected += o.Expected
	t.C0 += o.C0
	t.C1 += o.C1
	t.C2 += o.C2
	t.Candidate += o.Candidate
	t.NoCandidate += o.NoCandidate
	t.Rows += o.Rows
	t.LinkOnly += o.LinkOnly
	t.Robots += o.Robots
	t.NoSource += o.NoSource
	t.Open += o.Open
}

// Sum sums the competitions for which in is true.
func (r *Report) Sum(in func(c *Comp) bool) Totals {
	var t Totals
	for _, c := range r.Competitions {
		if in(c) {
			t.add(c.Totals)
		}
	}
	return t
}

// Comp is the coverage of one competition.
type Comp struct {
	Code  string `json:"code"`
	Name  string `json:"name"`
	Group string `json:"group"`
	// Tier1 is true for a national competition of tier 1.
	Tier1 bool `json:"tier1_national,omitempty"`
	Totals
	// Shape is false when the register gives no positions for the competition. Then Expected is 0 and the year list shows only the positions that the seed rows give.
	Shape bool    `json:"shape"`
	Years []*Year `json:"years,omitempty"`
}

// Year lists the positions of one edition by state. A competition with two papers in a year, such as the AMC 10 with the papers A and B, has one Year for each paper.
type Year struct {
	Year int `json:"year"`
	// Edition is the edition of the paper, such as 2021a or 2021-fall-a, when the register lists the papers of the year. It is empty when the year has one edition with the year as its name.
	Edition string `json:"edition,omitempty"`
	// Record are the positions with a problem record. Candidate are the positions with no record and with a seed candidate. Missing are the expected positions with neither.
	Record    []string `json:"record,omitempty"`
	Candidate []string `json:"candidate,omitempty"`
	Missing   []string `json:"missing,omitempty"`
	// Reasons gives the reason of each candidate or missing position that the gap register names. The other candidate and missing positions are open.
	Reasons map[string]string `json:"reasons,omitempty"`
	// Rows counts the seed rows of this edition, with or without a position. A row that does not name its paper counts in the first paper of the year.
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

// Build makes the report. gaps is the gap register, which can be nil. rows walks every seed row.
func Build(corpus *content.Corpus, al *seed.Aliases, gaps *Gaps, thisYear int, rows func(fn func(seed.Row) error) error) (*Report, error) {
	rep := &Report{ThisYear: thisYear, Aliases: al.Len()}
	if gaps != nil {
		for _, g := range gaps.List {
			g.Count = 0
		}
		rep.Gaps = gaps.List
	}
	// cand[code][edition][pos] is true for a position with a seed candidate. seen[code][edition] counts the rows of the edition. The edition is the year, or the edition of the paper when the register lists the papers of the year. paperless[code][year] counts the rows of a year with papers that do not name their paper.
	cand := map[string]map[string]map[string]bool{}
	seen := map[string]map[string]int{}
	paperless := map[string]map[int]int{}
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
		ed, known := paperOf(corpus.Competition(code), r.Year, r.Labels)
		if !known {
			if paperless[code] == nil {
				paperless[code] = map[int]int{}
			}
			paperless[code][r.Year]++
		} else {
			if seen[code] == nil {
				seen[code] = map[string]int{}
			}
			seen[code][ed]++
		}
		pos := seed.Position(r.Number)
		if pos == "" {
			return nil
		}
		s.Positioned++
		// A row that does not name its paper in a year with two papers is not a candidate of either paper.
		if !known {
			return nil
		}
		if cand[code] == nil {
			cand[code] = map[string]map[string]bool{}
		}
		if cand[code][ed] == nil {
			cand[code][ed] = map[string]bool{}
		}
		cand[code][ed][pos] = true
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, c := range corpus.Competitions {
		rep.Competitions = append(rep.Competitions, measure(c, cand[c.Code], seen[c.Code], paperless[c.Code], compRows[c.Code], thisYear, gaps))
	}
	rep.Totals = rep.Sum(func(*Comp) bool { return true })
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

// paperOf gives the edition of a row of the year: the year, or the edition of the paper that the labels of the row name when the register lists the papers of the year. The result known is false when the year has more than one paper and the labels do not name one, or name a season that has no papers in the year.
func paperOf(c *content.Competition, year int, labels []string) (edition string, known bool) {
	if c == nil || len(c.Papers) == 0 {
		return strconv.Itoa(year), true
	}
	part, letter, ok := canon.Paper(c, year, labels)
	if !ok {
		return "", false
	}
	letters := c.PartLetters(year, part)
	switch {
	case len(letters) == 1 && letters[0] == "":
		return content.EditionID(year, part, ""), true
	case slices.Contains(letters, letter):
		return content.EditionID(year, part, letter), true
	case len(letters) == 0 && part == "":
		// A year that the register does not list, such as a year before the first year, has one paper.
		return strconv.Itoa(year), true
	}
	return "", false
}

// paper is one edition of a year: the year, or one paper of the year when the register lists its papers.
type paper struct {
	id      string
	edition *content.Edition
}

// papers gives the editions of a year. A competition whose register lists papers has one edition for each paper of the year and each edition of the content in the year. Another competition has one edition for the year, which is the first edition of the content in the year when there is one.
func papers(c *content.Competition, year int, editions []*content.Edition) []paper {
	if len(c.Papers) == 0 {
		p := paper{id: strconv.Itoa(year)}
		if len(editions) > 0 {
			p.edition = editions[0]
		}
		return []paper{p}
	}
	var out []paper
	for _, id := range c.EditionIDs(year) {
		out = append(out, paper{id: id})
	}
	if len(out) == 0 && len(editions) == 0 {
		out = append(out, paper{id: strconv.Itoa(year)})
	}
	for _, e := range editions {
		i := slices.IndexFunc(out, func(p paper) bool { return p.id == e.ID })
		if i < 0 {
			out = append(out, paper{id: e.ID})
			i = len(out) - 1
		}
		out[i].edition = e
	}
	return out
}

func measure(c *content.Competition, cand map[string]map[string]bool, seen map[string]int, paperless map[int]int, rows, thisYear int, gaps *Gaps) *Comp {
	out := &Comp{Code: c.Code, Name: c.Name, Group: c.Group}
	out.Rows = rows
	out.Shape = c.HasShape()
	editions := map[int][]*content.Edition{}
	for _, e := range c.Editions {
		editions[e.Year] = append(editions[e.Year], e)
	}
	// The years: from the first year to the last year, or to this year. With no first year, the years that the content or the seed rows give.
	years := map[int]bool{}
	if c.FirstYear > 0 {
		for y := c.FirstYear; y <= lastYear(c, thisYear); y++ {
			if !slices.Contains(c.Gaps, y) {
				years[y] = true
			}
		}
	}
	for y := range editions {
		years[y] = true
	}
	for ed := range seen {
		if y, err := strconv.Atoi(ed[:4]); err == nil {
			years[y] = true
		}
	}
	for y := range paperless {
		years[y] = true
	}
	ys := make([]int, 0, len(years))
	for y := range years {
		ys = append(ys, y)
	}
	slices.Sort(ys)
	for _, y := range ys {
		for i, pp := range papers(c, y, editions[y]) {
			yr := &Year{Year: y, Rows: seen[pp.id]}
			if pp.id != strconv.Itoa(y) {
				yr.Edition = pp.id
			}
			if i == 0 {
				yr.Rows += paperless[y]
			}
			out.add(yr, Expected(c, y, pp.edition), pp.edition, cand[pp.id], gaps)
			out.Years = append(out.Years, yr)
		}
	}
	return out
}

// add measures one edition: the positions that the edition expects, the problems of the edition e, which is nil when the content has none, and the positions with a seed candidate. gaps gives the reason of each expected position with no record.
func (out *Comp) add(yr *Year, expected []string, e *content.Edition, cand map[string]bool, gaps *Gaps) {
	have := map[string]*content.Problem{}
	if e != nil {
		for _, p := range e.Problems {
			have[p.Position] = p
		}
		// A problem whose primary appearance is in another edition also fills its position here, as a shortlist problem that was chosen for the contest does.
		for _, p := range e.Also {
			for _, a := range p.Appearances {
				if a.Locator.Comp == e.Competition.Code && a.Locator.Edition == e.ID && have[a.Locator.Position] == nil {
					have[a.Locator.Position] = p
				}
			}
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
	for p := range cand {
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
		if cand[p] {
			yr.Candidate = append(yr.Candidate, p)
			if counted {
				out.Candidate++
			}
		} else if counted {
			yr.Missing = append(yr.Missing, p)
			out.NoCandidate++
		}
		if counted {
			out.reason(yr, p, gaps.entry(out.Code, yr.Year, p), cand[p])
		}
	}
}

// reason counts the reason of an expected position with no record. A seed candidate is a source to check, so a position with a candidate is open when the entry says that it has no allowed source.
func (out *Comp) reason(yr *Year, p string, g *Gap, cand bool) {
	r := Open
	if g != nil && (g.Reason != NoSource || !cand) {
		r = g.Reason
		g.Count++
		if yr.Reasons == nil {
			yr.Reasons = map[string]string{}
		}
		yr.Reasons[p] = r
	}
	switch r {
	case LinkOnly:
		out.LinkOnly++
	case Robots:
		out.Robots++
	case NoSource:
		out.NoSource++
	default:
		out.Open++
	}
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

// Pct gives n of d as a percent with one decimal, or "-" when d is 0.
func Pct(n, d int) string {
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
	p("The gap register (`%s`) gives the reason of an expected position with no record: link-only by rule, blocked by robots.txt, or no allowed source. A position that the register does not name is open. A position with a seed candidate is open, also when the register says it has no allowed source. The allowed positions are the expected positions less the positions that are link-only, blocked or with no allowed source. \"C0 allowed\" is the share of the allowed positions with a problem record.\n\n", GapFile)
	p("## Totals\n\n")
	p("| Expected | C0 | Allowed | C0 allowed | C1 | C2 | Candidate | No candidate | Located seed rows |\n|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	p("| %d | %s | %d | %s | %s | %s | %d (%s) | %d (%s) | %d |\n\n", t.Expected, Pct(t.C0, t.Expected), t.Allowed(), Pct(t.C0, t.Allowed()), Pct(t.C1, t.Expected), Pct(t.C2, t.Expected), t.Candidate, Pct(t.Candidate, t.Expected), t.NoCandidate, Pct(t.NoCandidate, t.Expected), t.Rows)
	p("### Reasons of the positions with no record\n\n")
	reasons(p, "All competitions", t)
	if r.Tier1 != nil {
		reasons(p, "Tier 1 nationals", *r.Tier1)
	}
	p("\n")
	if r.Tier1 != nil {
		n := *r.Tier1
		p("## Tier 1 nationals\n\nThe national competitions of the countries and codes of tier 1 in `registers/site-tiers.yaml`.\n\n")
		p("| Expected | With a record | C0 | Allowed | C0 allowed |\n|---:|---:|---:|---:|---:|\n")
		p("| %d | %d | %s | %d | %s |\n\n", n.Expected, n.C0, Pct(n.C0, n.Expected), n.Allowed(), Pct(n.C0, n.Allowed()))
		p("The tier 1 nationals with expected positions that have no record, the most open positions first:\n\n")
		p("| Code | Expected | C0 | Allowed | C0 allowed | Candidate | Link-only | Robots | No source | Open |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
		var list []*Comp
		for _, c := range r.Competitions {
			if c.Tier1 && c.Expected > c.C0 {
				list = append(list, c)
			}
		}
		slices.SortStableFunc(list, func(a, b *Comp) int { return cmp.Or(b.Open-a.Open, (b.Expected-b.C0)-(a.Expected-a.C0)) })
		for _, c := range list {
			p("| %s | %d | %s | %d | %s | %d | %d | %d | %d | %d |\n", c.Code, c.Expected, Pct(c.C0, c.Expected), c.Allowed(), Pct(c.C0, c.Allowed()), c.Candidate, c.LinkOnly, c.Robots, c.NoSource, c.Open)
		}
		p("\n")
	}
	p("## Sources\n\n| Source | Rows | Located | With a position | Not a competition |\n|---|---:|---:|---:|---:|\n")
	for _, s := range r.Sources {
		p("| %s | %d | %d (%s) | %d | %d |\n", s.ID, s.Rows, s.Located, Pct(s.Located, s.Rows), s.Positioned, s.Excluded)
	}
	p("\n## Competitions\n\nThe order is the order of the register. A competition with no shape in the register has no expected positions yet.\n\n")
	p("| Code | Competition | Expected | C0 | Allowed | C0 allowed | C1 | C2 | Candidate | No candidate | Link-only | Robots | No source | Open | Seed rows |\n|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	for _, c := range r.Competitions {
		exp := strconv.Itoa(c.Expected)
		if !c.Shape {
			exp = "no shape"
		}
		p("| %s | %s | %s | %s | %d | %s | %s | %s | %d | %d | %d | %d | %d | %d | %d |\n", c.Code, c.Name, exp, Pct(c.C0, c.Expected), c.Allowed(), Pct(c.C0, c.Allowed()), Pct(c.C1, c.Expected), Pct(c.C2, c.Expected), c.Candidate, c.NoCandidate, c.LinkOnly, c.Robots, c.NoSource, c.Open, c.Rows)
	}
	if len(r.Gaps) > 0 {
		p("\n## Gap register\n\nEach entry of `%s` with the number of expected positions with no record that it gives the reason of. An entry with 0 positions names no open position now, so it can go.\n\n", GapFile)
		p("| Competition | Years | Positions | Reason | Count | Note |\n|---|---|---|---|---:|---|\n")
		for _, g := range r.Gaps {
			years, pos := strings.Join(g.Years, ", "), strings.Join(g.Positions, ", ")
			if years == "" {
				years = "all"
			}
			if pos == "" {
				pos = "all"
			}
			p("| %s | %s | %s | %s | %d | %s |\n", g.Competition, years, pos, g.Reason, g.Count, mdCell(g.Note))
		}
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

// reasons writes one line of the reasons of the positions with no record.
func reasons(p func(string, ...any), name string, t Totals) {
	p("- %s: %d link-only by rule, %d blocked by robots.txt, %d with no allowed source, %d open.\n", name, t.LinkOnly, t.Robots, t.NoSource, t.Open)
}

// mdCell makes a text fit in one cell of a Markdown table.
func mdCell(s string) string {
	return strings.NewReplacer("|", "\\|", "\n", " ").Replace(strings.TrimSpace(s))
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
