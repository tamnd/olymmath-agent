// Package similar finds the problems with similar statements. It weighs the words of each text by TF-IDF and compares two texts by the cosine of their weights.
package similar

import (
	"math"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

// Match is a text that is similar to another text, with its index and its score from 0 to 1.
type Match struct {
	Doc   int
	Score float64
}

// stop are the common words that tell nothing about a problem.
var stop = map[string]bool{}

func init() {
	for _, w := range strings.Fields(`the and that for are with all such let show prove find each this there which from than then where any its their have has not one two be is of in to a an as at by it or on if no so we can
		real number numbers positive integer integers given determine exists exist only some every holds following other what when how many least most same also into these those does do whose them they`) {
		stop[w] = true
	}
}

// Words gives the words of a text that count for the similarity: lower case, with at least two letters, and not in the stop list.
func Words(text string) []string {
	var out []string
	for _, w := range strings.FieldsFunc(strings.ToLower(text), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }) {
		if len([]rune(w)) < 2 || stop[w] {
			continue
		}
		out = append(out, w)
	}
	return out
}

// posting is one text of a word in the index, with the weight of the word in the text.
type posting struct {
	doc int
	w   float64
}

// Find gives, for each text, at most n other texts with a score of at least least, the best first. A word that is in more than a tenth of the texts does not count, because it is too common to tell texts apart.
func Find(texts []string, n int, least float64) [][]Match {
	sorted, vecs, index := weigh(texts)
	out := make([][]Match, len(texts))
	// The scores of one text do not depend on the other scores, so the workers share the texts.
	var next atomic.Int64
	var wg sync.WaitGroup
	for range min(max(runtime.NumCPU(), 1), max(len(texts), 1)) {
		wg.Go(func() {
			// Each worker keeps one score buffer for all its texts, so a text does not make a new map.
			s := newScores(len(texts))
			for {
				i := int(next.Add(1)) - 1
				if i >= len(texts) {
					return
				}
				out[i] = best(i, sorted[i], vecs[i], index, n, least, s)
			}
		})
	}
	wg.Wait()
	return out
}

// weigh gives the words of each text that count, in order, the weight of each word in each text, and the index of the texts of each word.
func weigh(texts []string) ([][]string, []map[string]float64, map[string][]posting) {
	tf := make([]map[string]float64, len(texts))
	df := map[string]int{}
	for i, t := range texts {
		tf[i] = map[string]float64{}
		for _, w := range Words(t) {
			tf[i][w]++
		}
		for w := range tf[i] {
			df[w]++
		}
	}
	limit := max(len(texts)/10, 2)
	index := map[string][]posting{}
	vecs := make([]map[string]float64, len(texts))
	// The words of each text in order, so that the sums do not depend on the order of a map.
	sorted := make([][]string, len(texts))
	for i := range texts {
		vecs[i] = map[string]float64{}
		for w := range tf[i] {
			if df[w] >= 2 && df[w] <= limit {
				sorted[i] = append(sorted[i], w)
			}
		}
		sort.Strings(sorted[i])
		var norm float64
		for _, w := range sorted[i] {
			x := (1 + math.Log(tf[i][w])) * math.Log(float64(len(texts))/float64(df[w]))
			vecs[i][w] = x
			norm += x * x
		}
		norm = math.Sqrt(norm)
		for _, w := range sorted[i] {
			vecs[i][w] /= norm
			index[w] = append(index[w], posting{i, vecs[i][w]})
		}
	}
	return sorted, vecs, index
}

// scores are the sums of the scores of one text with each other text. score has one entry for each text, and hit lists the texts with a sum, in the order of their first sum. A text with a sum of 0 is in hit too, as in a map.
type scores struct {
	score []float64
	seen  []bool
	hit   []int
}

func newScores(n int) *scores {
	return &scores{score: make([]float64, n), seen: make([]bool, n)}
}

// best gives the n best matches of text i. It uses s for the sums and leaves it empty for the next text.
func best(i int, words []string, vec map[string]float64, index map[string][]posting, n int, least float64, s *scores) []Match {
	for _, w := range words {
		x := vec[w]
		for _, p := range index[w] {
			if p.doc != i {
				if !s.seen[p.doc] {
					s.seen[p.doc] = true
					s.hit = append(s.hit, p.doc)
				}
				s.score[p.doc] += x * p.w
			}
		}
	}
	var ms []Match
	for _, d := range s.hit {
		if s.score[d] >= least {
			ms = append(ms, Match{d, s.score[d]})
		}
		s.score[d], s.seen[d] = 0, false
	}
	s.hit = s.hit[:0]
	sort.Slice(ms, func(a, b int) bool {
		if ms[a].Score != ms[b].Score {
			return ms[a].Score > ms[b].Score
		}
		return ms[a].Doc < ms[b].Doc
	})
	if len(ms) > n {
		ms = ms[:n]
	}
	return ms
}

// Nearest gives, for each query, the doc with the best score, or a Match with Doc -1 when no doc shares a word with the query. The weights of the words come from the docs only, so that a large set of queries does not change them, and only the docs are indexed.
func Nearest(docs, queries []string) []Match {
	df := map[string]int{}
	tfs := make([]map[string]float64, len(docs))
	for i, t := range docs {
		tfs[i] = termFreq(t)
		for w := range tfs[i] {
			df[w]++
		}
	}
	limit := max(len(docs)/10, 2)
	weigh := func(tf map[string]float64, least int) map[string]float64 {
		v := map[string]float64{}
		var norm float64
		for w, c := range tf {
			if df[w] < least || df[w] > limit {
				continue
			}
			x := (1 + math.Log(c)) * math.Log(float64(len(docs))/float64(df[w]))
			v[w] = x
			norm += x * x
		}
		norm = math.Sqrt(norm)
		for w := range v {
			v[w] /= norm
		}
		return v
	}
	index := map[string][]posting{}
	for i, tf := range tfs {
		v := weigh(tf, 1)
		ws := make([]string, 0, len(v))
		for w := range v {
			ws = append(ws, w)
		}
		sort.Strings(ws)
		for _, w := range ws {
			index[w] = append(index[w], posting{i, v[w]})
		}
	}
	out := make([]Match, len(queries))
	for i, q := range queries {
		score := map[int]float64{}
		for w, x := range weigh(termFreq(q), 1) {
			for _, p := range index[w] {
				score[p.doc] += x * p.w
			}
		}
		best := Match{Doc: -1}
		for d, s := range score {
			if s > best.Score || (s == best.Score && d < best.Doc) {
				best = Match{d, s}
			}
		}
		out[i] = best
	}
	return out
}

func termFreq(text string) map[string]float64 {
	tf := map[string]float64{}
	for _, w := range Words(text) {
		tf[w]++
	}
	return tf
}
