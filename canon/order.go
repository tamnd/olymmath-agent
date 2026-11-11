package canon

import (
	"cmp"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/seed"
)

// Order places the rows of a dataset that keeps the order of the AoPS wiki pages, as the amc_aime rows of NuminaMath 1.5 do. The rows come by competition in the order of Sections, then by year, then by paper, and the problems of a paper come in the order of the page titles: 1, 10, 11, ..., 19, 2, 20, ... The rows have no year and no number. Order finds the year of each row from the reference rows of other sources that have the same text, and gives positions only to a run of rows of one year that holds exactly the problems of the register for that year. A run that misses a problem or has one too many gives no position, because a missing problem moves every row after it.
type Order struct {
	// Sections are the competition codes in the order of the dataset.
	Sections []string
	// Shared maps a competition to another one that shares problems with it, so that a reference row of the first stands for the second too, as an AMC 12 row does for the AMC 10 paper of the same year.
	Shared map[string]string
	// Papers gives the paper letters of a year of a competition and the number of problems in each paper. A part such as "fall" names a second set of papers in the year. It gives no letters when the year is not known well enough to place rows.
	Papers func(code string, year int, part string) ([]string, int)

	refs  []orderRef
	index map[string][]int
}

type orderKey struct {
	sec  int
	year int
	part string
}

func (a orderKey) compare(b orderKey) int {
	return cmp.Or(cmp.Compare(a.sec, b.sec), cmp.Compare(a.year, b.year), strings.Compare(a.part, b.part))
}

type orderRef struct {
	keys  []orderKey
	words map[string]bool
}

// Placed is the place that Order gives a row.
type Placed struct {
	Code   string
	Year   int
	Letter string
	Number int
}

// MinOrderMatch is the least share of common words for a reference row to give its year to a row.
const MinOrderMatch = 0.5

var (
	orderWordRE = regexp.MustCompile(`[a-z]+|[0-9]+`)
	orderTeXRE  = regexp.MustCompile(`\\(?:textbf|mathrm|text|qquad|quad|frac|dfrac)`)
)

func orderWords(text string) map[string]bool {
	text = orderTeXRE.ReplaceAllString(strings.ToLower(text), " ")
	out := map[string]bool{}
	for _, w := range orderWordRE.FindAllString(text, -1) {
		if len(w) > 1 || (w[0] >= '0' && w[0] <= '9') {
			out[w] = true
		}
	}
	return out
}

// Ref adds a reference row: a text of the competition code in the year. The part is "" or a name such as "fall" for a second set of papers in the year.
func (o *Order) Ref(code string, year int, part, text string) {
	sec := slices.Index(o.Sections, code)
	if sec < 0 || year == 0 {
		return
	}
	keys := []orderKey{{sec, year, part}}
	if s := slices.Index(o.Sections, o.Shared[code]); s >= 0 {
		keys = append(keys, orderKey{s, year, part})
	}
	w := orderWords(text)
	if len(w) == 0 {
		return
	}
	if o.index == nil {
		o.index = map[string][]int{}
	}
	for k := range w {
		o.index[k] = append(o.index[k], len(o.refs))
	}
	o.refs = append(o.refs, orderRef{keys: keys, words: w})
}

// keysOf gives the keys of the reference rows that match the text best, from the last to the first.
func (o *Order) keysOf(text string) []orderKey {
	w := orderWords(text)
	hits := map[int]int{}
	for k := range w {
		// A word that many rows have does not tell them apart.
		if ids := o.index[k]; len(ids) < 400 {
			for _, id := range ids {
				hits[id]++
			}
		}
	}
	ids := make([]int, 0, len(hits))
	for id := range hits {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return cmp.Or(hits[ids[j]]-hits[ids[i]], ids[i]-ids[j]) < 0 })
	if len(ids) > 20 {
		ids = ids[:20]
	}
	best := 0.0
	score := map[int]float64{}
	for _, id := range ids {
		r := o.refs[id]
		common := 0
		for k := range w {
			if r.words[k] {
				common++
			}
		}
		v := float64(common) / float64(len(w)+len(r.words)-common)
		score[id] = v
		best = max(best, v)
	}
	if best < MinOrderMatch {
		return nil
	}
	var keys []orderKey
	for _, id := range ids {
		if score[id] >= best-0.1 && score[id] >= MinOrderMatch {
			for _, k := range o.refs[id].keys {
				if !slices.Contains(keys, k) {
					keys = append(keys, k)
				}
			}
		}
	}
	slices.SortFunc(keys, func(a, b orderKey) int { return b.compare(a) })
	return keys
}

// Place gives the place of each row of rows that it can place, by the index of the row. The rows must be in the order of the dataset.
func (o *Order) Place(rows []seed.Row) map[int]Placed {
	key := o.longest(rows)
	out := map[int]Placed{}
	for _, r := range o.runs(key) {
		letters, per := o.Papers(o.Sections[r.key.sec], r.key.year, r.key.part)
		if len(letters) == 0 || per <= 0 || r.hi-r.lo+1 != len(letters)*per {
			continue
		}
		order := make([]int, per)
		for i := range order {
			order[i] = i + 1
		}
		slices.SortFunc(order, func(a, b int) int { return strings.Compare(strconv.Itoa(a), strconv.Itoa(b)) })
		for p, l := range letters {
			for i, n := range order {
				out[r.lo+p*per+i] = Placed{Code: o.Sections[r.key.sec], Year: r.key.year, Letter: l, Number: n}
			}
		}
	}
	return out
}

// longest gives each row the key of the longest chain of rows whose keys do not go down, so that a row whose text matches a row of another year by chance gets no key.
func (o *Order) longest(rows []seed.Row) []*orderKey {
	type item struct {
		row, parent int
		key         orderKey
	}
	var items []item
	var tails []orderKey
	var tailItem []int
	for i, r := range rows {
		// The keys of a row come from the last to the first, so that the chain takes at most one key of a row.
		for _, k := range o.keysOf(r.Statement) {
			j := sort.Search(len(tails), func(n int) bool { return tails[n].compare(k) > 0 })
			parent := -1
			if j > 0 {
				parent = tailItem[j-1]
			}
			items = append(items, item{row: i, parent: parent, key: k})
			if j == len(tails) {
				tails = append(tails, k)
				tailItem = append(tailItem, len(items)-1)
			} else {
				tails[j], tailItem[j] = k, len(items)-1
			}
		}
	}
	out := make([]*orderKey, len(rows))
	if len(tailItem) == 0 {
		return out
	}
	for p := tailItem[len(tailItem)-1]; p >= 0; p = items[p].parent {
		k := items[p].key
		out[items[p].row] = &k
	}
	return out
}

type orderRun struct {
	key            orderKey
	lo, hi         int
	openLo, openHi bool
}

// runs gives the closed runs of rows of one key. A row with no key inside a run belongs to the run. Rows with no key between two runs belong to the next run when only the run before is full, and to the run before when only the next run is full. Else both runs stay open at that side, and an open run gives no position.
func (o *Order) runs(key []*orderKey) []orderRun {
	var runs []orderRun
	for i := 0; i < len(key); {
		if key[i] == nil {
			i++
			continue
		}
		r := orderRun{key: *key[i], lo: i, hi: i}
		for j := i + 1; j < len(key) && (key[j] == nil || *key[j] == r.key); j++ {
			if key[j] != nil {
				r.hi = j
			}
		}
		r.openLo = i > 0 && key[i-1] == nil
		r.openHi = r.hi < len(key)-1 && key[r.hi+1] == nil
		runs = append(runs, r)
		i = r.hi + 1
	}
	full := func(r orderRun) bool {
		letters, per := o.Papers(o.Sections[r.key.sec], r.key.year, r.key.part)
		return len(letters) > 0 && r.hi-r.lo+1 == len(letters)*per
	}
	isFull := make([]bool, len(runs))
	for i, r := range runs {
		isFull[i] = full(r)
	}
	for i := 0; i+1 < len(runs); i++ {
		a, b := &runs[i], &runs[i+1]
		if !a.openHi {
			continue
		}
		switch {
		case isFull[i] && !isFull[i+1]:
			b.lo, b.openLo, a.openHi = a.hi+1, false, false
		case isFull[i+1] && !isFull[i]:
			a.hi, a.openHi, b.openLo = b.lo-1, false, false
		}
	}
	out := runs[:0]
	for _, r := range runs {
		if !r.openLo && !r.openHi {
			out = append(out, r)
		}
	}
	return out
}
