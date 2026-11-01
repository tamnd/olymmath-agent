package coverage

import (
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

func TestGrid(t *testing.T) {
	c := &content.Competition{Code: "imo", FirstYear: 2022, Gaps: []int{2022}, Layout: []content.Layout{{Positions: []string{"p1", "p2"}}}}
	e := &content.Edition{Year: 2024, ID: "2024", Competition: c}
	h1 := &content.Problem{Position: "p1", Canonical: content.Canonical{Statement: "S1"}, Statements: []content.Statement{{ID: "S1", Origin: "H1"}}}
	h1.Solutions = []*content.Solution{{Origin: "official"}}
	d := &content.Problem{Position: "p3", Canonical: content.Canonical{Statement: "S1"}, Statements: []content.Statement{{ID: "S1", Origin: "D"}}}
	e.Problems = []*content.Problem{h1, d}
	c.Editions = []*content.Edition{e}
	rows, total := Grid(c, 2025)
	if len(rows) != 3 || rows[0].Year != 2025 || rows[1].Edition != e || rows[2].Edition != nil {
		t.Fatalf("rows %+v", rows)
	}
	r := rows[1]
	if len(r.Cells) != 3 || r.Cells[1].Problem != nil || r.Cells[2].Position != "p3" || r.Cells[2].Counted {
		t.Errorf("cells %+v", r.Cells)
	}
	if !r.Cells[0].Human || r.Cells[0].Class != "H1" {
		t.Errorf("p1 %+v", r.Cells[0])
	}
	if total.Expected != 6 || total.Reached[C0] != 1 || total.Reached[C4] != 1 {
		t.Errorf("total %+v", total)
	}
	if got := total.Percent(C0); got != "16%" {
		t.Errorf("C0 %s", got)
	}
	if got := total.Percent(C7); got != "-" {
		t.Errorf("C7 %s", got)
	}
	free := &content.Competition{Code: "isl", Editions: []*content.Edition{{Year: 2024, Problems: []*content.Problem{d}}}}
	_, ft := Grid(free, 2025)
	if ft.Expected != 1 || ft.Percent(C0) != "-" || ft.Percent(C1) != "0%" {
		t.Errorf("no shape %+v", ft)
	}
	eras := &content.Competition{Code: "ahsme", FirstYear: 2000, LastYear: 2003, PerYear: 3, Eras: []content.Era{{From: 2000, To: 2001, PerYear: 2}}, Overrides: map[int][]string{2003: {"p1"}}}
	_, et := Grid(eras, 2025)
	if et.Expected != 2+2+3+1 {
		t.Errorf("eras %+v", et)
	}
}
