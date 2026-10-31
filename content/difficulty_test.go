package content

import (
	"testing"

	"github.com/tamnd/olymmath-agent/locator"
)

func TestPositionBand(t *testing.T) {
	for _, c := range []struct {
		comp, pos string
		want      int
	}{{"imo", "p1", 6}, {"imo", "p6", 9}, {"usamo", "p3", 8}, {"imo", "p7", 0}, {"isl", "a1", 5}, {"isl", "g4", 6}, {"isl", "c9", 9}, {"isl", "x1", 0}, {"putnam", "a1", 0}, {"imo", "", 0}} {
		if got := PositionBand(c.comp, c.pos); got != c.want {
			t.Errorf("%s %s: %d", c.comp, c.pos, got)
		}
	}
	p := &Problem{Appearances: []Appearance{
		{Locator: locator.Locator{Comp: "isl", Edition: "2024", Position: "n1"}},
		{Primary: true, Locator: locator.Locator{Comp: "imo", Edition: "2024", Position: "p3"}},
	}}
	if p.Band() != 9 {
		t.Errorf("band %d", p.Band())
	}
}
