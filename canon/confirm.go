package canon

import (
	"math"
	"path"

	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/similar"
)

// The thresholds of signal E for Confirm.
const (
	// EFloor is the least E of a doubt with the canonical candidate when the edition has other appearances to compare with.
	EFloor = 0.2
	// EAlone is the least E when the edition has no other appearance, so that E cannot be compared.
	EAlone = 0.7
)

// Confirm settles the doubts that signal E explains (spec 06.5). Signal E is the cosine of the TF-IDF weights of the words of two texts. Two texts of one problem in other words have a low T but a high E. A doubt becomes a kept candidate when its answer does not disagree, its E with the canonical candidate is at least EFloor, and its E is higher than its E with the canonical candidate of each other appearance of the edition. A text that is closer to another problem of the edition stays a doubt, because its key can be wrong. Confirm sets E on each doubt that stays.
func Confirm(choices []Choice) []Choice {
	vec := vectors(choices)
	editions := map[string][]int{}
	for i, ch := range choices {
		ed := path.Dir(ch.At)
		editions[ed] = append(editions[ed], i)
	}
	for i := range choices {
		ch := &choices[i]
		if len(ch.Doubts) == 0 {
			continue
		}
		seen := map[string]bool{ch.Best.Source: true}
		for _, c := range ch.Kept {
			seen[c.Source] = true
		}
		var keep []Doubt
		for _, d := range ch.Doubts {
			v := vec[d.C]
			d.E = dot(v, vec[ch.Best])
			other := -1.0
			for _, j := range editions[path.Dir(ch.At)] {
				if j != i {
					other = max(other, dot(v, vec[choices[j].Best]))
				}
			}
			ok := d.A != match.AnswerMismatch && d.E > other && (d.E >= EFloor && other >= 0 || d.E >= EAlone)
			if !ok {
				keep = append(keep, d)
				continue
			}
			if !seen[d.C.Source] {
				seen[d.C.Source] = true
				ch.Kept = append(ch.Kept, d.C)
			}
		}
		ch.Doubts = keep
	}
	return choices
}

// vectors gives the unit TF-IDF vector of the text of each candidate of the choices. The document frequency comes from the canonical candidates.
func vectors(choices []Choice) map[*Candidate]map[string]float64 {
	df := map[string]int{}
	for _, ch := range choices {
		seen := map[string]bool{}
		for _, w := range similar.Words(ch.Best.Text) {
			if !seen[w] {
				seen[w] = true
				df[w]++
			}
		}
	}
	n := float64(len(choices))
	out := map[*Candidate]map[string]float64{}
	add := func(c *Candidate) {
		if _, ok := out[c]; ok {
			return
		}
		tf := map[string]float64{}
		for _, w := range similar.Words(c.Text) {
			tf[w]++
		}
		v := map[string]float64{}
		var norm float64
		for w, k := range tf {
			x := (1 + math.Log(k)) * math.Log((n+1)/float64(df[w]+1))
			v[w] = x
			norm += x * x
		}
		norm = math.Sqrt(norm)
		for w := range v {
			v[w] /= norm
		}
		out[c] = v
	}
	for _, ch := range choices {
		add(ch.Best)
		for _, d := range ch.Doubts {
			add(d.C)
		}
	}
	return out
}

func dot(a, b map[string]float64) float64 {
	if len(a) > len(b) {
		a, b = b, a
	}
	var s float64
	for w, x := range a {
		s += x * b[w]
	}
	return s
}
