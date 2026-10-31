package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
)

func init() {
	register(command{"solutions", "link the human solutions of a curated archive to their problems (om solutions kedlaya-putnam)", runSolutions})
}

// linkedSolution is the front matter of a solution that is a link only (zone Z3).
type linkedSolution struct {
	Tag     string `yaml:"tag"`
	Kind    string `yaml:"kind"`
	Problem string `yaml:"problem"`
	Origin  string `yaml:"origin"`
	Author  struct {
		Display string `yaml:"display"`
	} `yaml:"author"`
	Source   string `yaml:"source"`
	URL      string `yaml:"url"`
	Capture  string `yaml:"capture,omitempty"`
	Zone     string `yaml:"zone"`
	Lang     string `yaml:"lang"`
	Complete bool   `yaml:"complete"`
}

func runSolutions(env Env, args []string) int {
	fs := newFlags("solutions", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", archive.PutnamSolutionsFirst, "the first year")
	to := fs.Int("to", time.Now().Year()-1, "the last year. The solutions of a year come out after the contest in December.")
	dry := fs.Bool("dry", false, "count the solutions and write nothing")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 || pos[0] != "kedlaya-putnam" {
		fmt.Fprintln(env.Stderr, "usage: om solutions [-from YEAR] [-to YEAR] [-dry] kedlaya-putnam")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om solutions: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	src := corpus.Source(pos[0])
	if src == nil || src.Zone.Solution == "" {
		fmt.Fprintf(env.Stderr, "om solutions: the source register has no %s with a solution zone\n", pos[0])
		return ExitUsage
	}
	byAt := map[string]*content.Problem{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			byAt[a.At] = p
		}
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om solutions: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	ctx := context.Background()
	code := ExitOK
	var linked, have, missing int
	for _, d := range archive.PutnamSolutions(*from, *to) {
		tex, hash, err := getCapture(ctx, fc, pos[0], d.URL)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om solutions: %d: %v\n", d.Year, err)
			code = ExitFailure
			continue
		}
		authors := archive.Authors(tex)
		for _, label := range archive.Labels(tex) {
			p := byAt[fmt.Sprintf("putnam/%d/%s", d.Year, strings.ToLower(label))]
			if p == nil {
				missing++
				continue
			}
			if hasSolution(p, pos[0]) {
				have++
				continue
			}
			linked++
			if *dry {
				continue
			}
			if err := writeLinked(corpus, *root, p, src, hash, archive.SolutionLink(d.Year), authors); err != nil {
				fmt.Fprintf(env.Stderr, "om solutions: %s: %v\n", p.Path, err)
				return ExitFailure
			}
		}
	}
	fmt.Fprintf(env.Stdout, "%d solutions linked, %d already linked, %d with no problem.\n", linked, have, missing)
	return code
}

func hasSolution(p *content.Problem, source string) bool {
	for _, s := range p.Solutions {
		if s.Source == source {
			return true
		}
	}
	return false
}

// writeLinked writes a solution that is a link to the solutions of a curated archive. A solution in zone Z3 has no text.
func writeLinked(corpus *content.Corpus, root string, p *content.Problem, src *content.SourceRecord, capture, url, authors string) error {
	tags, err := mintTags(corpus, 1)
	if err != nil {
		return err
	}
	s := linkedSolution{Tag: tags[0], Kind: "solution", Problem: p.Tag, Origin: "human", Source: src.ID, URL: url, Capture: capture, Zone: src.Zone.Solution, Lang: "en", Complete: true}
	s.Author.Display = authors
	if s.Author.Display == "" {
		s.Author.Display = src.Name
	}
	var front bytes.Buffer
	enc := yaml.NewEncoder(&front)
	enc.SetIndent(2)
	if err := enc.Encode(s); err != nil {
		return err
	}
	rel := path.Join("solutions", strings.TrimSuffix(strings.TrimPrefix(p.Path, "problems/"), ".md"), s.Tag+".md")
	file := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(file), 0o750); err != nil {
		return err
	}
	if err := os.WriteFile(file, []byte("---\n"+front.String()+"---\n"), 0o644); err != nil { // #nosec G306 -- a public content file.
		return err
	}
	corpus.Tags[s.Tag] = "solution"
	p.Solutions = append(p.Solutions, &content.Solution{Tag: s.Tag, Source: src.ID})
	return addTag(root, s.Tag, "solution")
}
