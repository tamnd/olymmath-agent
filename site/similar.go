package site

import (
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/similar"
)

// The number of similar problems on a problem page, and the least score of a similar problem.
const (
	similarCount = 5
	similarMin   = 0.25
)

// findSimilar finds the similar problems of each problem from the words of the statements.
func (b *builder) findSimilar() error {
	all := b.corpus.All()
	texts := make([]string, len(all))
	for i, p := range all {
		texts[i] = Plain(p.Statement)
	}
	b.similar = map[*content.Problem][]*content.Problem{}
	for i, ms := range similar.Find(texts, similarCount, similarMin) {
		for _, m := range ms {
			b.similar[all[i]] = append(b.similar[all[i]], all[m.Doc])
		}
	}
	return nil
}
