// Package similar finds the problems with similar statements. It weighs the words of each text by TF-IDF and compares two texts by the cosine of their weights.
package similar

import (
	"math"
	"sort"
	"strings"
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

// Find gives, for each text, at most n other texts with a score of at least least, the best first. A word that is in more than a tenth of the texts does not count, because it is too common to tell texts apart.
func Find(texts []string, n int, least float64) [][]Match {
	type posting struct {
		doc int
		w   float64
	}
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
	for i := range texts {
		vecs[i] = map[string]float64{}
		var norm float64
		for w, c := range tf[i] {
			if df[w] < 2 || df[w] > limit {
				continue
			}
			x := (1 + math.Log(c)) * math.Log(float64(len(texts))/float64(df[w]))
			vecs[i][w] = x
			norm += x * x
		}
		norm = math.Sqrt(norm)
		for w := range vecs[i] {
			vecs[i][w] /= norm
		}
	}
	// The words of each text in order, so that the sums of the scores do not depend on the order of a map.
	sorted := make([][]string, len(texts))
	for i := range texts {
		for w := range vecs[i] {
			sorted[i] = append(sorted[i], w)
		}
		sort.Strings(sorted[i])
		for _, w := range sorted[i] {
			index[w] = append(index[w], posting{i, vecs[i][w]})
		}
	}
	out := make([][]Match, len(texts))
	for i := range texts {
		score := map[int]float64{}
		for _, w := range sorted[i] {
			x := vecs[i][w]
			for _, p := range index[w] {
				if p.doc != i {
					score[p.doc] += x * p.w
				}
			}
		}
		var ms []Match
		for d, s := range score {
			if s >= least {
				ms = append(ms, Match{d, s})
			}
		}
		sort.Slice(ms, func(a, b int) bool {
			if ms[a].Score != ms[b].Score {
				return ms[a].Score > ms[b].Score
			}
			return ms[a].Doc < ms[b].Doc
		})
		if len(ms) > n {
			ms = ms[:n]
		}
		out[i] = ms
	}
	return out
}
