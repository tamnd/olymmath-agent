package canon

import (
	"strconv"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/seed"
)

// orderText gives a text of letters only, so that two problems share no word but the words of the year.
func orderText(year, n int) string {
	w := func(v int) string {
		var b strings.Builder
		for _, c := range strconv.Itoa(v) {
			b.WriteByte(byte('g' + c - '0'))
			b.WriteByte('q')
		}
		return b.String()
	}
	return "Find " + w(year) + " plus " + w(n*7+year*11) + " and " + w(n*13+year*17) + "."
}

func TestOrder(t *testing.T) {
	o := &Order{
		Sections: []string{"x", "y"},
		Shared:   map[string]string{"y": "x"},
		Papers: func(code string, year int, part string) ([]string, int) {
			if part != "" {
				return nil, 0
			}
			if code == "x" && year == 2003 {
				return []string{"a", "b"}, 12
			}
			return []string{""}, 12
		},
	}
	lex := []int{1, 10, 11, 12, 2, 3, 4, 5, 6, 7, 8, 9}
	var rows []seed.Row
	add := func(code string, year, n, plus int, ref bool) {
		text := orderText(year, n+plus)
		if ref {
			o.Ref(code, year, "", text)
		}
		rows = append(rows, seed.Row{Statement: text})
	}
	// x 2000 is full. One row of it has no reference row and stays inside the run.
	for _, n := range lex {
		add("x", 2000, n, 0, n != 11)
	}
	// x 2001 misses problem 4, so no row of it gets a place.
	for _, n := range lex {
		if n != 4 {
			add("x", 2001, n, 0, true)
		}
	}
	// x 2003 has two papers. Half of the problems of paper b match rows of y, which stand for x too.
	for _, n := range lex {
		add("x", 2003, n, 0, true)
	}
	for _, n := range lex {
		add([]string{"x", "y"}[n%2], 2003, n, 100, true)
	}
	// The section of y starts again from an earlier year. A row with no reference row lies between y 1998, which misses it, and y 1999, which is full.
	for _, n := range lex {
		add("y", 1998, n, 0, n != 9)
	}
	for _, n := range lex {
		add("y", 1999, n, 0, true)
	}
	got := o.Place(rows)
	want := map[string]int{"x/2000/": 12, "x/2003/a": 12, "x/2003/b": 12, "y/1998/": 12, "y/1999/": 12}
	count := map[string]int{}
	for i, p := range got {
		k := p.Code + "/" + strconv.Itoa(p.Year) + "/" + p.Letter
		count[k]++
		if rows[i].Statement != orderText(p.Year, p.Number+map[bool]int{true: 100}[p.Letter == "b"]) {
			t.Errorf("row %d placed at %+v", i, p)
		}
	}
	for k, n := range want {
		if count[k] != n {
			t.Errorf("%s: %d rows, want %d", k, count[k], n)
		}
	}
	if len(count) != len(want) {
		t.Errorf("placed %v", count)
	}
}

func TestOrderAnchors(t *testing.T) {
	o := &Order{
		Sections: []string{"x"},
		Papers: func(_ string, _ int, part string) ([]string, int) {
			if part == "fall" {
				return []string{"a", "b"}, 12
			}
			return []string{""}, 12
		},
	}
	lex := []int{1, 10, 11, 12, 2, 3, 4, 5, 6, 7, 8, 9}
	var rows []seed.Row
	add := func(year int, part string, n int, anchor bool, letter string) {
		text := orderText(year, n)
		o.Ref("x", year, part, text)
		if anchor {
			o.Anchor("x", year, part, letter, n, text)
		}
		rows = append(rows, seed.Row{Statement: text})
	}
	// x 2005 misses problem 5. The anchors 1, 12 and 3 close two runs of the right length. Between 3 and 9 one row is missing, so those rows get no place, and the rows after 9 have no anchor after them.
	for _, n := range lex {
		if n != 5 {
			add(2005, "", n, n == 1 || n == 12 || n == 3 || n == 9, "")
		}
	}
	// x 2006 has one anchor only, so a number alone gives no place.
	for _, n := range lex[:8] {
		add(2006, "", n, n == 2, "")
	}
	// The fall of 2007 has the papers a and b, and it misses problem 4 of paper a. The anchors 7 of paper a and 10 of paper b close a run of the right length across the two papers.
	for _, l := range []string{"a", "b"} {
		for _, n := range lex {
			if l == "a" && n == 4 {
				continue
			}
			plus := map[string]int{"a": 0, "b": 100}[l]
			text := orderText(2007, n+plus)
			o.Ref("x", 2007, "fall", text)
			if (l == "a" && n == 7) || (l == "b" && n == 10) {
				o.Anchor("x", 2007, "fall", l, n, text)
			}
			rows = append(rows, seed.Row{Statement: text})
		}
	}
	got := o.Place(rows)
	var placed []string
	for i := range rows {
		p, ok := got[i]
		if !ok {
			continue
		}
		if !p.Anchored {
			t.Errorf("row %d placed without anchors: %+v", i, p)
		}
		placed = append(placed, strconv.Itoa(p.Year)+p.Part+p.Letter+"/"+strconv.Itoa(p.Number))
	}
	want := "2005/1 2005/10 2005/11 2005/12 2005/2 2005/3 2007falla/7 2007falla/8 2007falla/9 2007fallb/1 2007fallb/10"
	if s := strings.Join(placed, " "); s != want {
		t.Errorf("placed %s\nwant   %s", s, want)
	}
	for i, p := range got {
		plus := map[string]int{"b": 100}[p.Letter]
		if rows[i].Statement != orderText(p.Year, p.Number+plus) {
			t.Errorf("row %d placed at %+v", i, p)
		}
	}
}
