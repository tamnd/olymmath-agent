package canon

import (
	"slices"
	"strings"
)

// MaxDoubt is the largest share of doubts per appearance that an import accepts for one competition. A higher share shows that the seed rows of the competition do not locate well, for example because two rounds share one position, so the import skips the competition until an adapter gives better rows.
const MaxDoubt = 0.15

// Gated is a competition that Gate removed.
type Gated struct {
	Comp        string
	Appearances int
	Doubts      int
}

// Gate removes the choices of each competition whose doubts per appearance are above maxShare. It returns the choices that remain and the removed competitions, sorted by code.
func Gate(choices []Choice, maxShare float64) ([]Choice, []Gated) {
	apps := map[string]int{}
	doubts := map[string]int{}
	for _, ch := range choices {
		comp, _, _ := strings.Cut(ch.At, "/")
		apps[comp]++
		doubts[comp] += len(ch.Doubts)
	}
	var gated []Gated
	drop := map[string]bool{}
	for comp, n := range apps {
		if float64(doubts[comp])/float64(n) > maxShare {
			drop[comp] = true
			gated = append(gated, Gated{Comp: comp, Appearances: n, Doubts: doubts[comp]})
		}
	}
	slices.SortFunc(gated, func(a, b Gated) int { return strings.Compare(a.Comp, b.Comp) })
	out := slices.DeleteFunc(choices, func(ch Choice) bool {
		comp, _, _ := strings.Cut(ch.At, "/")
		return drop[comp]
	})
	return out, gated
}
