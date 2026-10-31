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
)

func init() {
	register(command{"anchor", "compare the problems with the official papers of a source and link each problem that agrees to its paper (om anchor imo-official)", runAnchor})
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
	if len(pos) != 1 || pos[0] != "imo-official" {
		fmt.Fprintln(env.Stderr, "usage: om anchor [-from YEAR] [-to YEAR] imo-official")
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
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	ctx := context.Background()
	owners := official.Owners(corpus)
	var items []canon.Item
	done := map[string]bool{}
	var links, have, missing int
	code := ExitOK
	for _, p := range official.IMO(*from, *to) {
		segs, err := paper(ctx, fc, src.ID, p)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %s: %v\n", p.Edition, err)
			code = ExitFailure
			continue
		}
		done[p.Edition] = true
		r := official.Anchor(owners, src.ID, p, segs)
		fmt.Fprintf(env.Stdout, "%s: %d segments, %d links, %d review items, %d linked before, %d with no problem\n", p.Edition, len(segs), len(r.Links), len(r.Items), r.Have, len(r.Missing))
		links += len(r.Links)
		have += r.Have
		missing += len(r.Missing)
		items = append(items, r.Items...)
		if *dry {
			continue
		}
		for _, l := range r.Links {
			c := &canon.Candidate{Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: p.Lang, Name: p.Name, URL: p.URL}
			if err := addLink(*root, l.Problem, canon.AddLink(l.Problem, c, l.Segment.Page)); err != nil {
				fmt.Fprintf(env.Stderr, "om anchor: %s: %v\n", l.Problem.Path, err)
				return ExitFailure
			}
		}
	}
	if !*dry {
		old, err := review.Queue(*root)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
			return ExitFailure
		}
		old = slices.DeleteFunc(old, func(it canon.Item) bool { return it.Kind == "official" && it.Source == src.ID && done[edition(it.At)] })
		if err := writeQueue(*root, append(old, items...)); err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
			return ExitFailure
		}
	}
	fmt.Fprintf(env.Stdout, "%d papers. %d new links, %d linked before, %d review items, %d segments with no problem.\n", len(done), links, have, len(items), missing)
	return code
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
	return official.Split(text), nil
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

// edition gives the edition part of an appearance: imo/2024/p3 gives imo/2024.
func edition(at string) string { return path.Dir(at) }
