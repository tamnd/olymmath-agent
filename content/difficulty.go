package content

import "strconv"

// Bands is the number of difficulty bands. Band 1 is the easiest.
const Bands = 10

// sixBands gives the band of the problems 1 to 6 of a competition with two papers of three problems, by the position method of spec 4.9.
var sixBands = map[string][7]int{
	"imo":   {0, 6, 7, 9, 6, 7, 9},
	"usamo": {0, 6, 7, 8, 6, 7, 8},
}

// shortlistBands gives the band of the problems of a shortlist by their number in the topic: A1 and A2 are band 5, A7 and later are band 9.
var shortlistBands = map[string]bool{"isl": true}

// PositionBand gives the difficulty band of a position by the position method of spec 4.9, or 0 when the method does not apply to the competition.
func PositionBand(comp, position string) int {
	if len(position) < 2 {
		return 0
	}
	n, err := strconv.Atoi(position[1:])
	if err != nil || n < 1 {
		return 0
	}
	if bands, ok := sixBands[comp]; ok && position[0] == 'p' && n <= 6 {
		return bands[n]
	}
	if shortlistBands[comp] {
		if _, ok := Topics[string(position[0]-'a'+'A')]; ok {
			return min(5+(n-1)/2, 9)
		}
	}
	return 0
}

// Band gives the difficulty band of the problem, from 1 to Bands, or 0 when it has no band. The band comes from the first appearance that has one, the primary appearance first.
func (p *Problem) Band() int {
	best := 0
	for _, a := range p.Appearances {
		b := PositionBand(a.Locator.Comp, a.Locator.Position)
		if b > 0 && (best == 0 || a.Primary) {
			best = b
		}
	}
	return best
}
