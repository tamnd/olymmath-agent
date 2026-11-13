package similar

import (
	"math"
	"slices"
	"sort"
)

// Index holds the TF-IDF weights of a set of docs, for many queries. It keeps only the postings of the words, and not the texts or their term counts, so a large set of docs or queries needs little memory. An Index is safe for many goroutines, but each goroutine needs its own Searcher.
type Index struct {
	n     int
	least int
	limit int
	df    map[string]int
	index map[string][]posting
}

// NewIndex indexes the docs. A word counts when it is in at least least docs and in at most a tenth of them. Nearest uses 1. Find uses 2, so that a word of one text alone does not lower the scores of that text.
func NewIndex(docs []string, least int) *Index {
	x := &Index{n: len(docs), least: least, limit: max(len(docs)/10, 2), df: map[string]int{}, index: map[string][]posting{}}
	for _, t := range docs {
		for w := range termFreq(t) {
			x.df[w]++
		}
	}
	// The term counts are made again in the second pass, so the index never holds the counts of all the docs at once.
	for i, t := range docs {
		ws, v := x.vector(t)
		for _, w := range ws {
			x.index[w] = append(x.index[w], posting{i, v[w]})
		}
	}
	return x
}

// Len gives the number of docs.
func (x *Index) Len() int { return x.n }

// vector gives the words of a text that count, sorted, and their weights, with a norm of 1.
func (x *Index) vector(text string) ([]string, map[string]float64) {
	tf := termFreq(text)
	v := map[string]float64{}
	var ws []string
	var norm float64
	for w := range tf {
		if d := x.df[w]; d < x.least || d > x.limit {
			continue
		}
		ws = append(ws, w)
	}
	sort.Strings(ws)
	for _, w := range ws {
		y := (1 + math.Log(tf[w])) * math.Log(float64(x.n)/float64(x.df[w]))
		v[w] = y
		norm += y * y
	}
	norm = math.Sqrt(norm)
	for _, w := range ws {
		v[w] /= norm
	}
	return ws, v
}

// Searcher scores queries against an Index. It keeps one buffer of scores for all its queries. It is not safe for two goroutines.
type Searcher struct {
	x *Index
	s *scores
}

// Searcher gives a new Searcher of the index.
func (x *Index) Searcher() *Searcher { return &Searcher{x: x, s: newScores(x.n)} }

// sum adds the scores of the query to the buffer and calls fn with each doc that shares a word with the query, and its score. It leaves the buffer empty.
func (s *Searcher) sum(q string, fn func(doc int, score float64)) {
	ws, v := s.x.vector(q)
	for _, w := range ws {
		y := v[w]
		for _, p := range s.x.index[w] {
			if !s.s.seen[p.doc] {
				s.s.seen[p.doc] = true
				s.s.hit = append(s.s.hit, p.doc)
			}
			s.s.score[p.doc] += y * p.w
		}
	}
	for _, d := range s.s.hit {
		fn(d, s.s.score[d])
		s.s.score[d], s.s.seen[d] = 0, false
	}
	s.s.hit = s.s.hit[:0]
}

// Nearest gives the doc with the best score for the query, and the doc with the best score among the docs before index below. A Match has Doc -1 when no doc shares a word with the query. Of two docs with the same score, the first one wins.
func (s *Searcher) Nearest(q string, below int) (best, under Match) {
	best, under = Match{Doc: -1}, Match{Doc: -1}
	s.sum(q, func(d int, sc float64) {
		if sc > best.Score || (sc == best.Score && d < best.Doc) {
			best = Match{d, sc}
		}
		if d < below && (sc > under.Score || (sc == under.Score && d < under.Doc)) {
			under = Match{d, sc}
		}
	})
	return best, under
}

// Above gives at most n docs other than doc skip with a score of at least least for the query, the best first.
func (s *Searcher) Above(q string, skip, n int, least float64) []Match {
	var ms []Match
	s.sum(q, func(d int, sc float64) {
		if d != skip && sc >= least {
			ms = append(ms, Match{d, sc})
		}
	})
	sort.Slice(ms, func(a, b int) bool {
		if ms[a].Score != ms[b].Score {
			return ms[a].Score > ms[b].Score
		}
		return ms[a].Doc < ms[b].Doc
	})
	return slices.Clip(ms[:min(n, len(ms))])
}
