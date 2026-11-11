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
//
// A run that is not full can still get positions between anchors. An anchor is a row of the run whose text matches one numbered reference row of the same year, as a row of the AMC 12 dataset gives. Between two anchors, the rows get positions only when their number is exactly the number of problems between the two anchors in the order of the page titles. So a missing or extra row elsewhere in the run does not move them, and a number alone never gives a position.
type Order struct {
	// Sections are the competition codes in the order of the dataset.
	Sections []string
	// Shared maps a competition to another one that shares problems with it, so that a reference row of the first stands for the second too, as an AMC 12 row does for the AMC 10 paper of the same year.
	Shared map[string]string
	// Papers gives the paper letters of a year of a competition and the number of problems in each paper. A part such as "fall" names a second set of papers in the year. It gives no letters when the year is not known well enough to place rows.
	Papers func(code string, year int, part string) ([]string, int)

	refs    []orderRef
	index   map[string][]int
	anchors map[orderKey][]orderAnchor
}

// orderAnchor is a numbered reference row: problem number of the paper letter.
type orderAnchor struct {
	letter string
	number int
	words  map[string]bool
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

// Placed is the place that Order gives a row. Part is "" or a name such as "fall" for a second set of papers in the year. Anchored is true when the place comes from the anchors in a run that is not full.
type Placed struct {
	Code     string
	Year     int
	Part     string
	Letter   string
	Number   int
	Anchored bool
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

// Anchor adds a numbered reference row: the text of problem number of the paper letter of competition code in the year. The letter is "" for a year with one paper. The text should also be a reference row, given to Ref, so that the row of the dataset gets the year.
func (o *Order) Anchor(code string, year int, part, letter string, number int, text string) {
	sec := slices.Index(o.Sections, code)
	w := orderWords(text)
	if sec < 0 || year == 0 || number <= 0 || len(w) == 0 {
		return
	}
	if o.anchors == nil {
		o.anchors = map[orderKey][]orderAnchor{}
	}
	k := orderKey{sec, year, part}
	o.anchors[k] = append(o.anchors[k], orderAnchor{letter: letter, number: number, words: w})
}

// jaccard gives the share of common words of two sets of words.
func jaccard(a, b map[string]bool) float64 {
	common := 0
	for k := range a {
		if b[k] {
			common++
		}
	}
	if n := len(a) + len(b) - common; n > 0 {
		return float64(common) / float64(n)
	}
	return 0
}

// anchorAt gives the index of a row text in the order of the papers of the key, or -1. The text must match one problem of the anchors of the key by MinOrderMatch, and no other problem nearly as well.
func (o *Order) anchorAt(k orderKey, letters []string, rank []int, text string) int {
	w := orderWords(text)
	best, second, at := 0.0, 0.0, -1
	seen := map[int]float64{}
	for _, a := range o.anchors[k] {
		p := slices.Index(letters, a.letter)
		if p < 0 || a.number > len(rank) {
			continue
		}
		i := p*len(rank) + rank[a.number-1]
		seen[i] = max(seen[i], jaccard(w, a.words))
	}
	for i, v := range seen {
		switch {
		case v > best:
			best, second, at = v, best, i
		case v > second:
			second = v
		}
	}
	if best < MinOrderMatch || best-second < 0.1 {
		return -1
	}
	return at
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

// titleOrder gives the numbers 1 to per in the order of the page titles: 1, 10, 11, ..., 19, 2, 20, ...
func titleOrder(per int) []int {
	order := make([]int, per)
	for i := range order {
		order[i] = i + 1
	}
	slices.SortFunc(order, func(a, b int) int { return strings.Compare(strconv.Itoa(a), strconv.Itoa(b)) })
	return order
}

// Place gives the place of each row of rows that it can place, by the index of the row. The rows must be in the order of the dataset.
func (o *Order) Place(rows []seed.Row) map[int]Placed {
	key := o.longest(rows)
	out := map[int]Placed{}
	full := map[orderKey]bool{}
	for _, r := range o.runs(key) {
		letters, per := o.Papers(o.Sections[r.key.sec], r.key.year, r.key.part)
		if len(letters) == 0 || per <= 0 || r.hi-r.lo+1 != len(letters)*per {
			continue
		}
		full[r.key] = true
		order := titleOrder(per)
		for p, l := range letters {
			for i, n := range order {
				out[r.lo+p*per+i] = Placed{Code: o.Sections[r.key.sec], Year: r.key.year, Part: r.key.part, Letter: l, Number: n}
			}
		}
	}
	for i, p := range o.anchored(rows, key, full) {
		out[i] = p
	}
	return out
}

// anchored gives the places between anchors in the runs of keys that are not full.
func (o *Order) anchored(rows []seed.Row, key []*orderKey, full map[orderKey]bool) map[int]Placed {
	out := map[int]Placed{}
	type anchor struct{ row, at int }
	byKey := map[orderKey][]anchor{}
	for i, k := range key {
		if k == nil || full[*k] || len(o.anchors[*k]) == 0 {
			continue
		}
		letters, per := o.Papers(o.Sections[k.sec], k.year, k.part)
		if len(letters) == 0 || per <= 0 {
			continue
		}
		rank := make([]int, per)
		for r, n := range titleOrder(per) {
			rank[n-1] = r
		}
		if at := o.anchorAt(*k, letters, rank, rows[i].Statement); at >= 0 {
			byKey[*k] = append(byKey[*k], anchor{i, at})
		}
	}
	for k, as := range byKey {
		letters, per := o.Papers(o.Sections[k.sec], k.year, k.part)
		order := titleOrder(per)
		// Two rows that match one problem make both of them no anchor.
		count := map[int]int{}
		for _, a := range as {
			count[a.at]++
		}
		as = slices.DeleteFunc(as, func(a anchor) bool { return count[a.at] > 1 })
		for n := 0; n+1 < len(as); n++ {
			a, b := as[n], as[n+1]
			if b.at <= a.at || b.row-a.row != b.at-a.at || !sameKey(key[a.row:b.row+1], k) {
				continue
			}
			for i := a.row; i <= b.row; i++ {
				at := a.at + i - a.row
				out[i] = Placed{Code: o.Sections[k.sec], Year: k.year, Part: k.part, Letter: letters[at/per], Number: order[at%per], Anchored: true}
			}
		}
	}
	return out
}

// sameKey tells if each row of the slice has the key k or no key.
func sameKey(keys []*orderKey, k orderKey) bool {
	for _, x := range keys {
		if x != nil && *x != k {
			return false
		}
	}
	return true
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
