package audit

import (
	"slices"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/review"
)

// reviews loads the reviews once for the rules D01 to D03. A review file that is not valid is a finding of D03.
func (c *Context) reviews() map[string]*review.Review {
	if c.revs == nil {
		var errs []error
		c.revs, errs = review.Load(c.Root)
		c.revErrs = errs
	}
	return c.revs
}

// pairReviewed tells if a review decided that the two appearances are two problems.
func pairReviewed(revs map[string]*review.Review, a, b string) bool {
	for _, id := range []string{canon.ItemID("pair", a, b), canon.ItemID("pair", b, a)} {
		if r := revs[id]; r != nil && (r.Decision == "different" || r.Decision == "variant") {
			return true
		}
	}
	return false
}

func d01(c *Context) {
	revs := c.reviews()
	x := match.NewIndex()
	byAt := map[string]int{}
	ps := c.Corpus.All()
	for i, p := range ps {
		if len(p.Appearances) == 0 || p.Statement == "" {
			continue
		}
		at := p.Appearances[0].At
		byAt[at] = i
		x.Add(at, match.Shingles(p.Statement))
	}
	x.Pairs(match.Duplicate, func(a, b string, t float64) {
		pa, pb := ps[byAt[a]], ps[byAt[b]]
		if linked(pa.Links, pb.Tag) || linked(pb.Links, pa.Tag) || pairReviewed(revs, a, b) {
			return
		}
		c.add(pa.Path, "the canonical text of %s and the canonical text of %s have T %.3f. Merge the problems, add a variant_of edge, or record a review", a, b, t)
	})
}

func linked(ls []content.Link, tag string) bool {
	return slices.ContainsFunc(ls, func(l content.Link) bool { return l.Type == "variant_of" && l.To == tag })
}

func d02(c *Context) {
	revs := c.reviews()
	items, err := review.Queue(c.Root)
	if err != nil {
		c.add(review.QueueFile, "%v", err)
		return
	}
	for _, it := range items {
		if it.Kind == "pair" && revs[it.ID] == nil {
			c.add(review.QueueFile, "the pair %s and %s (T %.3f) has no review %s", it.At, it.With, it.T, it.ID)
		}
	}
}

func d03(c *Context) {
	revs := c.reviews()
	for _, err := range c.revErrs {
		c.add(review.Dir, "%v", err)
	}
	ms, err := review.Merges(c.Root)
	if err != nil {
		c.add(review.MergesFile, "%v", err)
		return
	}
	for _, m := range ms {
		switch {
		case m.Review != "-":
			if r := revs[m.Review]; r == nil || r.Decision != "accept" {
				c.add(review.MergesFile, "line %d: the merge of %s into %s names the review %s, which does not accept it", m.Line, m.At, m.Into, m.Review)
			}
		case !m.Strict():
			c.add(review.MergesFile, "line %d: the merge of %s into %s has fewer than two strict signals and no review", m.Line, m.At, m.Into)
		}
	}
}
