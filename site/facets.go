package site

import (
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// facetOption is one choice of a filter of the faceted list.
type facetOption struct {
	Value, Name string
	// Country is the country of a competition, for the country filter.
	Country string
}

// facetsView is the data of the faceted list: the choices of each filter. The list itself comes from search.json in the browser.
type facetsView struct {
	Competitions, Countries, Topics, Types []facetOption
	Index                                  string
}

// facets writes the faceted list of problems (spec 10.2).
func (b *builder) facets() error {
	v := facetsView{Index: b.link("search.json")}
	countries := map[string]bool{}
	types := map[string]bool{}
	for _, c := range b.corpus.Competitions {
		has := false
		for _, e := range c.Editions {
			if len(e.Problems) > 0 {
				has = true
				break
			}
		}
		if !has {
			continue
		}
		v.Competitions = append(v.Competitions, facetOption{Value: c.Code, Name: c.Name, Country: c.Country})
		if c.Country != "" {
			countries[c.Country] = true
		}
	}
	slices.SortFunc(v.Competitions, func(x, y facetOption) int { return strings.Compare(x.Name, y.Name) })
	for c := range countries {
		v.Countries = append(v.Countries, facetOption{Value: c, Name: countryName(c)})
	}
	slices.SortFunc(v.Countries, func(x, y facetOption) int { return strings.Compare(x.Name, y.Name) })
	for t, name := range content.Topics {
		v.Topics = append(v.Topics, facetOption{Value: t, Name: name})
	}
	slices.SortFunc(v.Topics, func(x, y facetOption) int { return strings.Compare(x.Value, y.Value) })
	for _, p := range b.corpus.All() {
		if p.Type != "" {
			types[p.Type] = true
		}
	}
	for t := range types {
		v.Types = append(v.Types, facetOption{Value: t, Name: strings.ReplaceAll(t, "+", " and ")})
	}
	slices.SortFunc(v.Types, func(x, y facetOption) int { return strings.Compare(x.Value, y.Value) })
	return b.render("problems.html", page{
		Title:       "Problems",
		Description: "All problems on olymmath, with filters for competition, country, year, topic, difficulty, type and solutions.",
		Path:        "/problems/",
		Section:     "problems",
		Data:        v,
	})
}
