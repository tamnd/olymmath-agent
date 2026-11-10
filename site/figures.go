package site

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// figures copies the figure files that the texts of the problems show into the site (spec 4.3), so each page can show them. A figure next to its problem goes next to the page of the problem: problems/imo/2024/p1.fig1.svg is at /imo/2024/p1.fig1.svg. The step runs before the texts render, since the texts link to the copies.
func (b *builder) figures() error {
	b.figure = map[string]string{}
	add := func(p *content.Problem, text, file string) error {
		if !strings.Contains(text, "![") {
			return nil
		}
		for _, ref := range content.FigureRefs(text) {
			f := content.FigurePath(file, ref)
			if f == "" || !content.IsFigureName(path.Base(f)) {
				continue
			}
			if _, ok := b.figure[f]; ok {
				continue
			}
			data, err := os.ReadFile(filepath.Join(b.corpus.Root, filepath.FromSlash(f))) // #nosec G304 -- the path is a figure of the content.
			if err != nil {
				// om audit reports a figure without a file (S08). The text keeps the ref as it is.
				b.stats.Errors = append(b.stats.Errors, fmt.Errorf("%s: the figure %s has no file", file, ref))
				b.figure[f] = ""
				continue
			}
			url := figureURL(p, f)
			b.figure[f] = url
			if err := b.write(strings.TrimPrefix(url, "/"), data); err != nil {
				return err
			}
		}
		return nil
	}
	for _, ps := range [][]*content.Problem{b.corpus.All(), b.corpus.Loose} {
		for _, p := range ps {
			if err := add(p, p.Statement, p.Path); err != nil {
				return err
			}
			if err := add(p, p.Answer, p.Path); err != nil {
				return err
			}
			for _, s := range p.Solutions {
				if err := add(p, s.Body, s.Path); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// text gives the Markdown text of the file at file with the link of the site for each figure that the site has. A page of an edition shows the text too, so a link next to the problem would not work there.
func (b *builder) text(src, file string) string {
	if !strings.Contains(src, "![") {
		return src
	}
	return content.MapFigures(src, func(ref string) string {
		if url := b.figure[content.FigurePath(file, ref)]; url != "" {
			return b.link(url)
		}
		return ref
	})
}

// figureURL gives the URL path of the figure f of the problem p in the site. A figure next to the problem file is next to the page of the problem, and any other figure keeps its path under problems.
func figureURL(p *content.Problem, f string) string {
	if path.Dir(f) == path.Dir(p.Path) {
		return path.Join(path.Dir(problemPath(p)), path.Base(f))
	}
	return "/" + strings.TrimPrefix(f, "problems/")
}
