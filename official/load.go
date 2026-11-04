package official

import (
	"context"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/store"
)

// Load reads the papers of a source from the captures of the store. It reads only the papers of the list, and it skips a paper that is not in the store or that has no readable text.
func Load(ctx context.Context, st *store.Store, source string, papers []Paper) (Papers, error) {
	out := Papers{}
	for _, p := range papers {
		c, ok, err := st.Latest(source, p.URL)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		rc, err := st.Get(ctx, c.Hash)
		if err != nil {
			continue
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		text, err := Text(ctx, b)
		if err != nil {
			continue
		}
		out[p.Edition] = p.Segments(text)
	}
	return out, nil
}

// LoadEditions reads the official papers of the editions of the competitions from the captures of the store. It reads the papers of each competition of comps, or of each competition when comps is empty. It skips a paper that is not in the store or that has no readable text. A paper with no source comes from the first source of its competition that has a capture of its URL. LoadEditions gives the papers and the number of papers that it read.
func LoadEditions(ctx context.Context, st *store.Store, corpus *content.Corpus, comps []string) (Papers, int, error) {
	out := Papers{}
	captures := map[string]map[string]store.Capture{}
	latest := func(source, url string) (store.Capture, bool, error) {
		if captures[source] == nil {
			cs, err := st.Captures(source)
			if err != nil {
				return store.Capture{}, false, err
			}
			m := map[string]store.Capture{}
			for _, c := range cs {
				m[c.URL] = c
			}
			captures[source] = m
		}
		c, ok := captures[source][url]
		return c, ok, nil
	}
	n := 0
	for _, comp := range corpus.Competitions {
		if len(comps) > 0 && !slices.Contains(comps, comp.Code) {
			continue
		}
		for _, e := range comp.Editions {
			for _, o := range e.Official {
				sources := comp.Sources
				if o.Source != "" {
					sources = []string{o.Source}
				}
				for _, src := range sources {
					c, ok, err := latest(src, o.URL)
					if err != nil {
						return nil, 0, err
					}
					if base, _, cut := strings.Cut(o.URL, "#"); !ok && cut {
						// A paper that is one section of a page has the URL of the page with the id of the section as the fragment. The store has the page under its URL with no fragment.
						if c, ok, err = latest(src, base); err != nil {
							return nil, 0, err
						}
					}
					if !ok {
						continue
					}
					segs, err := segments(ctx, st, src, c.Hash, comp.Shape(e.Year), o)
					if err != nil {
						break
					}
					out[e.Locator()] = append(out[e.Locator()], segs...)
					n++
					break
				}
			}
		}
	}
	return out, n, nil
}

// segments reads one official paper from the store and gives its segments. The positions come from the layout of the competition when the paper names a paper of the layout: the segment p2 of the second paper of a layout gets the second position of that paper. A paper that names a paper that is not in the layout gives segments with no position.
func segments(ctx context.Context, st *store.Store, source, hash string, shape content.Shape, o content.OfficialPaper) ([]Segment, error) {
	rc, err := st.Get(ctx, hash)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return nil, err
	}
	var text string
	if section, ok := Sections[source]; ok {
		_, id, _ := strings.Cut(o.URL, "#")
		if text = section(string(b), id); text == "" {
			return nil, ErrNoText
		}
	} else if text, err = Read(ctx, source, b); err != nil {
		return nil, err
	}
	if part, ok := Parts[source]; ok {
		if text = part(text, o.Paper); text == "" {
			return nil, ErrNoText
		}
	}
	segs := Split(text)
	lang := o.Code
	if English(lang) {
		lang = ""
	}
	for i := range segs {
		segs[i].Lang, segs[i].URL = lang, o.URL
	}
	Place(shape, o.Paper, segs)
	return segs, nil
}

// Place gives the segments of the paper id of the layout of shape their positions in the edition. A paper that numbers its problems through the edition keeps its numbers. A paper that numbers from 1 gets the positions of its paper in the layout: the segment p2 of the second paper gets the second position of that paper. A segment with a number that the paper does not have gets no position. When id is "", the paper holds the whole edition and the segments keep their numbers.
func Place(shape content.Shape, id string, segs []Segment) {
	if id == "" || numberedThrough(shape, id, segs) {
		return
	}
	for i := range segs {
		segs[i].Position = inPaper(shape, id, segs[i].Position)
	}
}

// numberedThrough reports whether a paper of the layout numbers its problems through the edition, as the second day of the IMC does with the problems 6 to 10. That is so when the first position of the paper is not p1 and the number of each segment is a position of the paper. Then each segment keeps its number as its position.
func numberedThrough(shape content.Shape, id string, segs []Segment) bool {
	for _, l := range shape.Layout {
		if l.Paper != id {
			continue
		}
		if len(l.Positions) == 0 || l.Positions[0] == "p1" || len(segs) == 0 {
			return false
		}
		for _, sg := range segs {
			if !slices.Contains(l.Positions, sg.Position) {
				return false
			}
		}
		return true
	}
	return false
}

// inPaper gives the position of the edition for the position pos, such as p2, of the paper id of the layout. It gives "" when the layout has no such paper or position.
func inPaper(shape content.Shape, id, pos string) string {
	k, err := strconv.Atoi(strings.TrimPrefix(pos, "p"))
	if err != nil || k < 1 {
		return ""
	}
	for _, l := range shape.Layout {
		if l.Paper == id && k <= len(l.Positions) {
			return l.Positions[k-1]
		}
	}
	return ""
}
