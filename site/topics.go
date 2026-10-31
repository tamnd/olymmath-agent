package site

import (
	"slices"
	"sort"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// topicView is one topic on the topics page, or the data of the page of one topic.
type topicView struct {
	Code, Name string
	Link       string
	Count      int
	Groups     []topicGroup
}

// topicGroup is the problems of one competition on the page of a topic, the newest first.
type topicGroup struct {
	Competition *content.Competition
	Problems    []*problemView
}

// topicPath gives the path of the page of a topic: A.inequality gives /t/a/.
func topicPath(t string) string {
	return "/t/" + strings.ToLower(content.TopicRoot(t)) + "/"
}

// topics writes the topics page and the page of each topic. A problem is on the page of each root of its topics.
func (b *builder) topics() error {
	codes := make([]string, 0, len(content.Topics))
	for t := range content.Topics {
		codes = append(codes, t)
	}
	slices.Sort(codes)
	byTopic := map[string]map[string][]*content.Problem{}
	for _, p := range b.corpus.All() {
		seen := map[string]bool{}
		for _, t := range p.Topics {
			root := content.TopicRoot(t)
			if _, ok := content.Topics[root]; !ok || seen[root] {
				continue
			}
			seen[root] = true
			if byTopic[root] == nil {
				byTopic[root] = map[string][]*content.Problem{}
			}
			byTopic[root][p.Competition] = append(byTopic[root][p.Competition], p)
		}
	}
	var all []topicView
	for _, t := range codes {
		v := topicView{Code: t, Name: content.Topics[t], Link: b.link(topicPath(t))}
		for _, c := range b.corpus.Competitions {
			ps := byTopic[t][c.Code]
			if len(ps) == 0 {
				continue
			}
			sort.SliceStable(ps, func(i, j int) bool {
				if ps[i].Year != ps[j].Year {
					return ps[i].Year > ps[j].Year
				}
				return comparePositions(ps[i].Position, ps[j].Position) < 0
			})
			g := topicGroup{Competition: c}
			for _, p := range ps {
				g.Problems = append(g.Problems, &problemView{Problem: p, Title: problemTitle(p), Link: b.link(problemPath(p))})
			}
			v.Groups = append(v.Groups, g)
			v.Count += len(ps)
		}
		if ps := byTopic[t][""]; len(ps) > 0 {
			g := topicGroup{}
			for _, p := range ps {
				g.Problems = append(g.Problems, &problemView{Problem: p, Title: problemTitle(p), Link: b.link(problemPath(p))})
			}
			v.Groups = append(v.Groups, g)
			v.Count += len(ps)
		}
		if err := b.render("topic.html", page{
			Title:       v.Name,
			Description: "The " + strings.ToLower(v.Name) + " problems on olymmath, by competition.",
			Path:        topicPath(t),
			Section:     "topics",
			Data:        v,
		}); err != nil {
			return err
		}
		v.Groups = nil
		all = append(all, v)
	}
	return b.render("topics.html", page{
		Title:       "Topics",
		Description: "The olympiad problems on olymmath by topic: algebra, combinatorics, geometry and number theory.",
		Path:        "/t/",
		Section:     "topics",
		Data:        all,
	})
}
