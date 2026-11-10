package keymatch

import (
	"cmp"
	"math"
	"slices"

	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/normalize"
)

// Key is a link-only problem with its private key text.
type Key struct {
	// At is the appearance of the problem, such as ru-allrussian/2004/p91.
	At string
	// Comp is the code of its competition and Year the year of its edition.
	Comp string
	Year int
	// Season tells that an edition runs from fall to spring, as the Tournament of Towns, so a row of the year before can hold it.
	Season bool
	F      Features
}

// Row is a seed row with no position.
type Row struct {
	// Ref names the row, as source|file#row.
	Ref string
	// Comp is the competition that the labels of the row name, or "" when they name none. Year is 0 when the row has no year.
	Comp string
	Year int
	F    Features
	// Text is the text of the row. It is kept only for a row that comes near a key, for a second check.
	Text string
	// Shingles are the word 5-grams of its text. Two rows with nearly the same words are one text, so they do not compete for a key. They are made only for a row that comes near a key.
	Shingles []uint64
}

// Thresholds of the content match. They were set on the problems of the content that have both a problems.ru link and a text of a licensed source: see the tests and the notes of the pull request.
const (
	// Strong is the lowest score of a pair that is placed with no other check.
	Strong = 0.55
	// Doubtful is the lowest score of a pair that a second check (a model, or a person) can place.
	Doubtful = 0.35
	// Margin is the smallest gap between the score of a pair and the next score on either side, the next key of the row and the next row of the key.
	Margin = 0.15
	// MinShared is the smallest weight of the shared tokens of a pair. A short text with only 1, 2 and n in common is not evidence.
	MinShared = 6.0
	// MinTokens is the smallest number of the shared tokens of a pair, and MinTokensOpen the number for a row that names no competition and no year. Two short texts of many rows can have the same few numbers by chance, as "two", 50 and 100.
	MinTokens     = 3
	MinTokensOpen = 5
	// Common is the share of the keys above which a token does not find keys (see NewMatcher).
	Common = 0.1
	// SameText is the Jaccard similarity of word 5-grams at which two rows are one text.
	SameText = 0.6
)

// Class of a pair.
const (
	ClassStrong   = "strong"
	ClassDoubtful = "doubtful"
)

// Pair is a row and a key that agree.
type Pair struct {
	Ref, At string
	Score   float64
	// Shared is the weight of the shared tokens, and Tokens their number.
	Shared float64
	Tokens int
	// RowNext is the best score of the row with another key, and KeyNext the best score of the key with a row of another text.
	RowNext, KeyNext float64
	Class            string
	// Text is the text of the row.
	Text string
}

// Matcher compares rows with a set of keys. Add reads the rows one by one and keeps only the rows that come near a key, so a large dataset does not stay in memory.
type Matcher struct {
	keys []Key
	// weight gives the weight of a token: the log of the number of keys over the number of keys that have it.
	weight map[string]float64
	none   float64
	// index maps a token to the keys that have it.
	index map[string][]int

	// rows are the rows that come near a key. byKey holds the scores of each key with them. rowBest[r] is the best key of row r, and rowNext[r] the score of its second key.
	rows    []Row
	byKey   map[int][]scored
	rowBest []scored
	rowNext []float64
	seen    map[int]bool
}

// NewMatcher makes a matcher for keys.
func NewMatcher(keys []Key) *Matcher {
	m := &Matcher{keys: keys, weight: map[string]float64{}, index: map[string][]int{}, byKey: map[int][]scored{}, seen: map[int]bool{}}
	df := map[string]int{}
	for i, k := range keys {
		for t := range k.F.Tokens {
			df[t]++
			m.index[t] = append(m.index[t], i)
		}
	}
	n := float64(len(keys))
	for t, d := range df {
		m.weight[t] = math.Log(1 + n/float64(d))
		if float64(d) > n*Common && d > 1 {
			// A token that many keys have, such as 2 or n, does not find the keys of a row: a row is compared only with the keys that share a rarer token with it.
			delete(m.index, t)
		}
	}
	m.none = math.Log(1 + n)
	return m
}

func (m *Matcher) w(t string) float64 {
	if w, ok := m.weight[t]; ok {
		return w
	}
	return m.none
}

// Score gives the score of a row and a key: the mean of the weighted Jaccard index of their tokens and the Jaccard index of their grams, and the weight and the number of the shared tokens.
func (m *Matcher) Score(a, b Features) (score, shared float64, n int) {
	all := 0.0
	for t := range a.Tokens {
		w := m.w(t)
		all += w
		if b.Tokens[t] {
			shared += w
			n++
		}
	}
	for t := range b.Tokens {
		if !a.Tokens[t] {
			all += m.w(t)
		}
	}
	if all == 0 {
		return 0, 0, 0
	}
	g := 0
	for t := range a.Grams {
		if b.Grams[t] {
			g++
		}
	}
	gu := len(a.Grams) + len(b.Grams) - g
	gj := 0.0
	if gu > 0 {
		gj = float64(g) / float64(gu)
	}
	// The sums run over maps in no fixed order, so the score is rounded: two rows of one text get the same score, and the order of the rows decides between them.
	return math.Round((shared/all+gj)/2*1e9) / 1e9, shared, n
}

// fits tells if a row can be a problem of a key: the same competition when the row names one, and the same year when the row has one. The year of a row of an edition from fall to spring can be the year of the fall.
func fits(r Row, k Key) bool {
	if r.Comp != "" && r.Comp != k.Comp {
		return false
	}
	return r.Year == 0 || r.Year == k.Year || (k.Season && r.Year == k.Year-1)
}

type scored struct {
	row, key      int
	score, shared float64
	n             int
}

// Add compares one row with each key that fits it and shares a token with it that few keys have.
func (m *Matcher) Add(r Row) {
	clear(m.seen)
	best, next := scored{key: -1}, 0.0
	var near []scored
	for t := range r.F.Tokens {
		for _, ki := range m.index[t] {
			if m.seen[ki] || !fits(r, m.keys[ki]) {
				continue
			}
			m.seen[ki] = true
			s, sh, n := m.Score(r.F, m.keys[ki].F)
			c := scored{key: ki, score: s, shared: sh, n: n}
			if s >= Doubtful-Margin {
				near = append(near, c)
			}
			switch {
			case best.key < 0 || s > best.score || (s == best.score && ki < best.key):
				if best.key >= 0 {
					next = max(next, best.score)
				}
				best = c
			default:
				next = max(next, s)
			}
		}
	}
	if len(near) == 0 {
		return
	}
	ri := len(m.rows)
	r.F = Features{}
	if r.Shingles == nil {
		r.Shingles = match.Shingles(normalize.Statement(r.Text, ""))
	}
	m.rows = append(m.rows, r)
	best.row = ri
	m.rowBest = append(m.rowBest, best)
	m.rowNext = append(m.rowNext, next)
	for _, c := range near {
		c.row = ri
		m.byKey[c.key] = append(m.byKey[c.key], c)
	}
}

// Pairs places the rows on the keys. A pair is placed when the row and the key are the best of each other, with a score of at least Doubtful, a shared weight of at least MinShared, and a gap of at least Margin to the next key of the row and to the next row of the key with another text. Such a pair is strong when its score is at least Strong, and doubtful below it. The result is sorted by appearance.
func (m *Matcher) Pairs() []Pair {
	rows := m.rows
	var out []Pair
	for ki, cs := range m.byKey {
		slices.SortFunc(cs, func(a, b scored) int {
			if c := cmp.Compare(b.score, a.score); c != 0 {
				return c
			}
			return cmp.Compare(rows[a.row].Ref, rows[b.row].Ref)
		})
		top := cs[0]
		next := 0.0
		for _, c := range cs[1:] {
			if match.Jaccard(rows[c.row].Shingles, rows[top.row].Shingles) < SameText {
				next = c.score
				break
			}
		}
		least := MinTokens
		if rows[top.row].Comp == "" && rows[top.row].Year == 0 {
			least = MinTokensOpen
		}
		if m.rowBest[top.row].key != ki || top.score < Doubtful || top.shared < MinShared || top.n < least {
			continue
		}
		rn := m.rowNext[top.row]
		if top.score-rn < Margin || top.score-next < Margin {
			continue
		}
		class := ClassDoubtful
		if top.score >= Strong {
			class = ClassStrong
		}
		out = append(out, Pair{Ref: rows[top.row].Ref, At: m.keys[ki].At, Score: top.score, Shared: top.shared, Tokens: top.n, RowNext: rn, KeyNext: next, Class: class, Text: rows[top.row].Text})
	}
	slices.SortFunc(out, func(a, b Pair) int { return cmp.Compare(a.At, b.At) })
	return out
}

// Best gives, for each key, the best row that comes near it, with no test of the thresholds. A calibration reads it.
func (m *Matcher) Best() map[string]Pair {
	out := map[string]Pair{}
	for ki, cs := range m.byKey {
		top := slices.MaxFunc(cs, func(a, b scored) int { return cmp.Compare(a.score, b.score) })
		out[m.keys[ki].At] = Pair{Ref: m.rows[top.row].Ref, At: m.keys[ki].At, Score: top.score, Shared: top.shared, Tokens: top.n, RowNext: m.rowNext[top.row]}
	}
	return out
}
