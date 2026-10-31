package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"solutions", "link the human solutions of a curated archive to their problems (om solutions kedlaya-putnam, om solutions imo-official)", runSolutions})
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
	if len(pos) != 1 || pos[0] != "kedlaya-putnam" && pos[0] != "imo-official" {
		fmt.Fprintln(env.Stderr, "usage: om solutions [-from YEAR] [-to YEAR] [-dry] kedlaya-putnam|imo-official")
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
	links, err := solutionLinks(ctx, fc, pos[0], *from, *to, env.Stderr)
	code := ExitOK
	if err != nil {
		code = ExitFailure
	}
	var linked, have, missing int
	for _, l := range links {
		p := byAt[l.at]
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
		l.sol.Source, l.sol.Zone = src.ID, src.Zone.Solution
		if l.sol.Author.Display == "" {
			l.sol.Author.Display = src.Name
		}
		if err := writeLinked(corpus, *root, p, l.sol); err != nil {
			fmt.Fprintf(env.Stderr, "om solutions: %s: %v\n", p.Path, err)
			return ExitFailure
		}
	}
	fmt.Fprintf(env.Stdout, "%d solutions linked, %d already linked, %d with no problem.\n", linked, have, missing)
	return code
}

// solutionLink is a solution that a source gives for the problem at a locator.
type solutionLink struct {
	at  string
	sol linkedSolution
}

// solutionLinks reads the solution files of a source and gives a link for each solution. It returns an error when a file could not be read, with the links of the other files.
func solutionLinks(ctx context.Context, fc *fetch.Client, source string, from, to int, stderr io.Writer) ([]solutionLink, error) {
	var out []solutionLink
	var failed error
	switch source {
	case "kedlaya-putnam":
		for _, d := range archive.PutnamSolutions(from, to) {
			tex, hash, err := getCapture(ctx, fc, source, d.URL)
			if err != nil {
				fmt.Fprintf(stderr, "om solutions: %d: %v\n", d.Year, err)
				failed = err
				continue
			}
			authors := archive.Authors(tex)
			for _, label := range archive.Labels(tex) {
				l := solutionLink{at: fmt.Sprintf("putnam/%d/%s", d.Year, strings.ToLower(label))}
				l.sol = linkedSolution{Origin: "human", URL: archive.SolutionLink(d.Year), Capture: hash, Lang: "en", Complete: true}
				l.sol.Author.Display = authors
				out = append(out, l)
			}
		}
	case "imo-official":
		for _, p := range official.Shortlist(max(from, official.ShortlistFirst), to) {
			res, err := fc.Get(ctx, source, p.URL, fetch.Options{})
			if err != nil {
				fmt.Fprintf(stderr, "om solutions: %s: %v\n", p.Edition, err)
				failed = err
				continue
			}
			text, err := captureText(ctx, fc, res.Capture.Hash)
			if err != nil {
				fmt.Fprintf(stderr, "om solutions: %s: %v\n", p.Edition, err)
				continue
			}
			pages := official.SolutionPages(text)
			for _, seg := range p.Segments(text) {
				page, ok := pages[seg.Position]
				if !ok {
					continue
				}
				l := solutionLink{at: p.Edition + "/" + seg.Position}
				l.sol = linkedSolution{Origin: "official", URL: fmt.Sprintf("%s#page=%d", p.URL, page), Capture: res.Capture.Hash, Lang: "en", Complete: true}
				l.sol.Author.Display = "the IMO Problem Selection Committee"
				out = append(out, l)
			}
		}
	}
	return out, failed
}

// captureText gives the text of the PDF of a capture.
func captureText(ctx context.Context, fc *fetch.Client, hash string) (string, error) {
	rc, err := fc.Store.Get(ctx, hash)
	if err != nil {
		return "", err
	}
	b, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return "", err
	}
	return official.Text(ctx, b)
}

func hasSolution(p *content.Problem, source string) bool {
	for _, s := range p.Solutions {
		if s.Source == source {
			return true
		}
	}
	return false
}

// writeLinked writes a solution that is a link to the solutions of a source. A solution in zone Z3 has no text, and olymmath does not copy an excerpt of a solution in zone Z2 either.
func writeLinked(corpus *content.Corpus, root string, p *content.Problem, s linkedSolution) error {
	tags, err := mintTags(corpus, 1)
	if err != nil {
		return err
	}
	s.Tag, s.Kind, s.Problem = tags[0], "solution", p.Tag
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
	p.Solutions = append(p.Solutions, &content.Solution{Tag: s.Tag, Source: s.Source})
	return addTag(root, s.Tag, "solution")
}
