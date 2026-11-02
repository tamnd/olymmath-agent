package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/review"
	"github.com/tamnd/olymmath-agent/store"
)

func init() {
	register(command{"anchor", "compare the problems with the official papers of a source and link each problem that agrees to its paper (om anchor imo-official, om anchor om-sem)", runAnchor})
}

func runAnchor(env Env, args []string) int {
	fs := newFlags("anchor", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 1959, "the first year")
	to := fs.Int("to", time.Now().Year(), "the last year")
	dry := fs.Bool("dry", false, "print the result and write no file")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 {
		fmt.Fprintln(env.Stderr, "usage: om anchor [-from YEAR] [-to YEAR] SOURCE\nSOURCE is imo-official, or a source whose papers om papers records, such as om-sem.")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om anchor: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	src := corpus.Source(pos[0])
	if src == nil {
		fmt.Fprintf(env.Stderr, "om anchor: %q is not in the source register\n", pos[0])
		return ExitUsage
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
		return ExitFailure
	}
	if src.ID != "imo-official" {
		return anchorEditions(env, *root, corpus, st, src, *from, *to, *dry)
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	ctx := context.Background()
	owners := official.Owners(corpus)
	var items []canon.Item
	done := map[string]bool{}
	var links, have, missing, traced int
	code := ExitOK
	for _, p := range append(official.IMO(*from, *to), official.Shortlist(*from, min(*to, time.Now().Year()-1))...) {
		segs, err := paper(ctx, fc, src.ID, p)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %s: %v\n", p.Edition, err)
			code = ExitFailure
			continue
		}
		done[p.Edition] = true
		r := official.Anchor(owners, src.ID, p, segs)
		fmt.Fprintf(env.Stdout, "%s: %d segments, %d links, %d review items, %d linked before, %d traced, %d with no problem\n", p.Edition, len(segs), len(r.Links), len(r.Items), r.Have, len(r.Traced), len(r.Missing))
		traced += len(r.Traced)
		links += len(r.Links)
		have += r.Have
		missing += len(r.Missing)
		items = append(items, r.Items...)
		if *dry {
			continue
		}
		// paperID holds the id of the statement that links each problem to the paper.
		paperID := map[*content.Problem]string{}
		for _, l := range r.Links {
			c := &canon.Candidate{Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: p.Lang, Name: p.Name, URL: p.URL}
			a := canon.AddLink(l.Problem, c, l.Segment.Page)
			if err := addLink(*root, l.Problem, a); err != nil {
				fmt.Fprintf(env.Stderr, "om anchor: %s: %v\n", l.Problem.Path, err)
				return ExitFailure
			}
			paperID[l.Problem] = a.ID(c)
		}
		_, rule := canon.Rank(&canon.Candidate{Origin: src.Origin, Lang: p.Lang}, original(corpus.Competition(path.Dir(p.Edition))))
		for _, t := range r.Traced {
			id := paperID[t.Problem]
			for _, s := range t.Problem.Statements {
				if s.Source == src.ID {
					id = s.ID
				}
			}
			if id == "" {
				continue
			}
			if err := trace(*root, t.Problem, t.Statement, id, rule); err != nil {
				fmt.Fprintf(env.Stderr, "om anchor: %s: %v\n", t.Problem.Path, err)
				return ExitFailure
			}
		}
	}
	if !*dry {
		if err := replaceItems(*root, src.ID, done, items); err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
			return ExitFailure
		}
	}
	fmt.Fprintf(env.Stdout, "%d papers. %d new links, %d linked before, %d traced, %d review items, %d segments with no problem.\n", len(done), links, have, traced, len(items), missing)
	return code
}

// replaceItems replaces the review items of kind official of the source for the editions of done with items.
func replaceItems(root, source string, done map[string]bool, items []canon.Item) error {
	old, err := review.Queue(root)
	if err != nil {
		return err
	}
	old = slices.DeleteFunc(old, func(it canon.Item) bool { return it.Kind == "official" && it.Source == source && done[edition(it.At)] })
	return writeQueue(root, append(old, items...))
}

// anchorEditions anchors the problems to the official papers of the editions files that come from the source, such as the papers that om papers records. The papers must be in the store: om fetch gets them. The papers are not in English, so a problem links to its paper when Locate places it at its own position (see official.AnchorEdition).
func anchorEditions(env Env, root string, corpus *content.Corpus, st *store.Store, src *content.SourceRecord, from, to int, dry bool) int {
	var comps []string
	for _, c := range corpus.Competitions {
		for _, e := range c.Editions {
			if e.Year >= from && e.Year <= to && slices.ContainsFunc(e.Official, func(o content.OfficialPaper) bool { return o.Source == src.ID }) {
				comps = append(comps, c.Code)
				break
			}
		}
	}
	if len(comps) == 0 {
		fmt.Fprintf(env.Stderr, "om anchor: no edition has an official paper of %s. Run om papers %s first.\n", src.ID, src.ID)
		return ExitUsage
	}
	papers, n, err := official.LoadEditions(context.Background(), st, corpus, comps)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
		return ExitFailure
	}
	owners := official.Owners(corpus)
	var items []canon.Item
	done := map[string]bool{}
	var links, have, missing int
	for _, code := range comps {
		c := corpus.Competition(code)
		for _, e := range c.Editions {
			at := e.Locator()
			if e.Year < from || e.Year > to || len(papers[at]) == 0 {
				continue
			}
			done[at] = true
			r := official.AnchorEdition(owners, src.ID, at, papers)
			fmt.Fprintf(env.Stdout, "%s: %d segments, %d links, %d review items, %d linked before, %d with no problem\n", at, len(papers[at]), len(r.Links), len(r.Items), r.Have, len(r.Missing))
			links += len(r.Links)
			have += r.Have
			missing += len(r.Missing)
			items = append(items, r.Items...)
			if dry {
				continue
			}
			for _, l := range r.Links {
				lang := l.Segment.Lang
				if lang == "" {
					lang = "en"
				}
				cd := &canon.Candidate{Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: lang, Name: fmt.Sprintf("%s %d official paper", c.Name, e.Year), URL: l.Segment.URL}
				if err := addLink(root, l.Problem, canon.AddLink(l.Problem, cd, l.Segment.Page)); err != nil {
					fmt.Fprintf(env.Stderr, "om anchor: %s: %v\n", l.Problem.Path, err)
					return ExitFailure
				}
			}
		}
	}
	if !dry {
		if err := replaceItems(root, src.ID, done, items); err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
			return ExitFailure
		}
	}
	fmt.Fprintf(env.Stdout, "%d papers of %d editions. %d new links, %d linked before, %d review items, %d segments with no problem.\n", n, len(done), links, have, len(items), missing)
	return ExitOK
}

// paper gets a paper into the store and returns its segments.
func paper(ctx context.Context, fc *fetch.Client, source string, p official.Paper) ([]official.Segment, error) {
	res, err := fc.Get(ctx, source, p.URL, fetch.Options{})
	if err != nil {
		return nil, err
	}
	rc, err := fc.Store.Get(ctx, res.Capture.Hash)
	if err != nil {
		return nil, err
	}
	b, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return nil, err
	}
	text, err := official.Text(ctx, b)
	if err != nil {
		return nil, err
	}
	return p.Segments(text), nil
}

func addLink(root string, p *content.Problem, a canon.Additions) error {
	full := filepath.Join(root, filepath.FromSlash(p.Path))
	b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	if b, err = canon.Apply(b, a); err != nil {
		return err
	}
	return os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
}

// trace marks the statement id of a problem as traced to the official statement official, and makes it canonical.
func trace(root string, p *content.Problem, id, official, rule string) error {
	full := filepath.Join(root, filepath.FromSlash(p.Path))
	b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	if b, err = canon.Trace(b, id, official, rule); err != nil {
		return err
	}
	return os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
}

// edition gives the edition part of an appearance: imo/2024/p3 gives imo/2024.
func edition(at string) string { return path.Dir(at) }
