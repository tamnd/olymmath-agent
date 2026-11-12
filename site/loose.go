package site

import (
	"fmt"
	"sort"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// LoosePage is the number of problems on one page of the list of a source.
var LoosePage = 500

// looseItem is one loose problem in a list.
type looseItem struct {
	Title string
	Link  string
	Text  string
	// Label is what the source says about where the problem was set, when the list is not by competition.
	Label string
}

// looseGroup is a heading and the problems under it.
type looseGroup struct {
	// ID is the anchor of the group, y-<edition>, or "" for none.
	ID       string
	Name     string
	Problems []looseItem
}

// looseCount is a line of the index of loose problems.
type looseCount struct {
	Name  string
	Link  string
	Count int
}

type looseIndex struct {
	Total        int
	Competitions []looseCount
	Sources      []looseCount
	// Nowhere is the number of loose problems with no unplaced entry.
	Nowhere int
}

type looseList struct {
	Title  string
	Lede   string
	Groups []looseGroup
	// Pages are the links of the pages of a long list, and Page the number of this page from 1.
	Pages []string
	Page  int
}

// looseSource gives the source of the canonical statement of a problem.
func looseSource(p *content.Problem) string {
	for _, s := range p.Statements {
		if s.ID == p.Canonical.Statement {
			return s.Source
		}
	}
	if len(p.Statements) > 0 {
		return p.Statements[0].Source
	}
	return ""
}

// unplacedTitle names an unplaced entry, for example "China MO 2015", or the label of the source when the code is not known.
func (b *builder) unplacedTitle(u content.Unplaced) string {
	code, edition, _ := strings.Cut(u.At, "/")
	if c := b.corpus.Competition(code); c != nil {
		return strings.TrimSpace(c.Short + " " + edition)
	}
	if u.Label != "" {
		return u.Label
	}
	return u.At
}

// loose writes the index of the problems with no position (spec 04.3): /loose/, a page for each competition that the sources name, grouped by year, and the pages of each source.
func (b *builder) loose() error {
	if len(b.corpus.Loose) == 0 {
		return nil
	}
	item := func(p *content.Problem) looseItem {
		return looseItem{Title: problemTitle(p), Link: b.link(problemPath(p)), Text: truncate(Plain(p.Statement), 160)}
	}
	byComp := map[string]map[string][]looseItem{}
	bySource := map[string][]*content.Problem{}
	idx := looseIndex{Total: len(b.corpus.Loose)}
	for _, p := range b.corpus.Loose {
		bySource[looseSource(p)] = append(bySource[looseSource(p)], p)
		if len(p.Unplaced) == 0 {
			idx.Nowhere++
		}
		seen := map[string]bool{}
		for _, u := range p.Unplaced {
			code, edition, _ := strings.Cut(u.At, "/")
			if b.corpus.Competition(code) == nil || seen[u.At] {
				continue
			}
			seen[u.At] = true
			if byComp[code] == nil {
				byComp[code] = map[string][]looseItem{}
			}
			byComp[code][edition] = append(byComp[code][edition], item(p))
		}
	}
	for _, c := range b.corpus.Competitions {
		eds := byComp[c.Code]
		if eds == nil {
			continue
		}
		var keys []string
		n := 0
		for k, ps := range eds {
			keys = append(keys, k)
			n += len(ps)
		}
		// The newest edition first, and the problems with no edition at the end.
		sort.Slice(keys, func(i, j int) bool {
			if (keys[i] == "") != (keys[j] == "") {
				return keys[j] == ""
			}
			return keys[i] > keys[j]
		})
		list := looseList{
			Title: c.Name + ": problems with no position",
			Lede:  fmt.Sprintf("Sources name %s for these %d problems but give no problem number, so they have no locator yet.", c.Name, n),
		}
		for _, k := range keys {
			g := looseGroup{ID: "y-" + k, Name: k, Problems: eds[k]}
			if k == "" {
				g.ID, g.Name = "y-none", "Year not given"
			}
			list.Groups = append(list.Groups, g)
		}
		link := "/loose/" + c.Code + "/"
		idx.Competitions = append(idx.Competitions, looseCount{Name: c.Name, Link: b.link(link), Count: n})
		if err := b.render("loose-list.html", page{
			Title:       list.Title,
			Description: list.Lede,
			Path:        link,
			Section:     "competitions",
			Data:        list,
		}); err != nil {
			return err
		}
	}
	var ids []string
	for id := range bySource {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		ps := bySource[id]
		name := id
		if r := b.corpus.Source(id); r != nil {
			name = r.Name
		}
		if id == "" {
			id, name = "none", "No source"
		}
		base := "/loose/source/" + id + "/"
		idx.Sources = append(idx.Sources, looseCount{Name: name, Link: b.link(base), Count: len(ps)})
		pages := (len(ps) + LoosePage - 1) / LoosePage
		var links []string
		for i := range pages {
			links = append(links, b.link(loosePagePath(base, i+1)))
		}
		for i := range pages {
			list := looseList{
				Title: name + ": problems with no position",
				Lede:  fmt.Sprintf("The %d problems from %s that have no locator yet.", len(ps), name),
				Page:  i + 1,
			}
			if pages > 1 {
				list.Pages = links
				list.Title += fmt.Sprintf(", page %d of %d", i+1, pages)
			}
			g := looseGroup{}
			for _, p := range ps[i*LoosePage : min(len(ps), (i+1)*LoosePage)] {
				it := item(p)
				var at []string
				for _, u := range p.Unplaced {
					at = append(at, b.unplacedTitle(u))
				}
				it.Label = strings.Join(at, ", ")
				g.Problems = append(g.Problems, it)
			}
			list.Groups = []looseGroup{g}
			if err := b.render("loose-list.html", page{
				Title:       list.Title,
				Description: list.Lede,
				Path:        loosePagePath(base, i+1),
				Section:     "sources",
				Data:        list,
			}); err != nil {
				return err
			}
		}
	}
	sort.SliceStable(idx.Sources, func(i, j int) bool { return idx.Sources[i].Count > idx.Sources[j].Count })
	return b.render("loose.html", page{
		Title:       "Problems with no position",
		Description: fmt.Sprintf("The %d problems on olymmath that have no known position in a competition, by competition and by source.", idx.Total),
		Path:        "/loose/",
		Section:     "competitions",
		Data:        idx,
	})
}

// loosePagePath gives the path of page n of a list: the first page is the index of the folder, the others are n.html.
func loosePagePath(base string, n int) string {
	if n == 1 {
		return base
	}
	return fmt.Sprintf("%s%d", base, n)
}
