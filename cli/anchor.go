package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
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
	create := fs.Bool("missing", false, "make a new problem for each position of a paper that has no problem. The problem links to the page of the paper and holds no text of it")
	pages := fs.String("pages", "", "a file of the pages of the problems of papers with no text, such as scans: one edition on each line, a tab, then the page of each problem from p1. With -missing")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 {
		fmt.Fprintln(env.Stderr, "usage: om anchor [-from YEAR] [-to YEAR] [-missing [-pages FILE]] SOURCE\nSOURCE is imo-official, or a source whose papers om papers records, such as om-sem.")
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
		return anchorEditions(env, *root, corpus, st, src, editionOptions{*from, *to, *dry, *create, *pages})
	} else if *create || *pages != "" {
		fmt.Fprintln(env.Stderr, "om anchor: -missing and -pages work for a source whose papers om papers records")
		return ExitUsage
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
func anchorEditions(env Env, root string, corpus *content.Corpus, st *store.Store, src *content.SourceRecord, o editionOptions) int {
	from, to, dry := o.from, o.to, o.dry
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
	scans := map[string][]int{}
	if o.pages != "" {
		if scans, err = readPages(o.pages); err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
			return ExitUsage
		}
	}
	owners := official.Owners(corpus)
	var items []canon.Item
	var news []newLink
	done := map[string]bool{}
	var links, have, missing int
	for _, code := range comps {
		c := corpus.Competition(code)
		for _, e := range c.Editions {
			at := e.Locator()
			if e.Year < from || e.Year > to {
				continue
			}
			name := fmt.Sprintf("%s %d official paper", c.Name, e.Year)
			if !slices.ContainsFunc(papers[at], func(sg official.Segment) bool { return sg.Position != "" }) {
				// A paper with no text gives the pages of its problems in the pages file. The text of a paper of an earlier round, such as a preliminary round, gives no position.
				url, lang := paperURL(e, src.ID)
				if !o.missing || url == "" || len(scans[at]) == 0 {
					continue
				}
				n := 0
				for i, pg := range scans[at] {
					pos := fmt.Sprintf("%s/p%d", at, i+1)
					if owners[pos] == nil {
						news = append(news, newLink{pos, name, lang, url, pg})
						n++
					}
				}
				fmt.Fprintf(env.Stdout, "%s: a paper with no text, %d problems in the pages file, %d with no problem\n", at, len(scans[at]), n)
				missing += n
				continue
			}
			done[at] = true
			r := official.AnchorEdition(owners, src.ID, at, papers)
			fmt.Fprintf(env.Stdout, "%s: %d segments, %d links, %d review items, %d linked before, %d with no problem\n", at, len(papers[at]), len(r.Links), len(r.Items), r.Have, len(r.Missing))
			links += len(r.Links)
			have += r.Have
			missing += len(r.Missing)
			items = append(items, r.Items...)
			if o.missing {
				for _, pos := range r.Missing {
					for _, sg := range papers[at] {
						if at+"/"+sg.Position == pos {
							news = append(news, newLink{pos, name, cmpOr(sg.Lang, "en"), sg.URL, sg.Page})
							break
						}
					}
				}
			}
			if dry {
				continue
			}
			for _, l := range r.Links {
				lang := l.Segment.Lang
				if lang == "" {
					lang = "en"
				}
				cd := &canon.Candidate{Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: lang, Name: name, URL: l.Segment.URL}
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
		if err := writeLinks(root, corpus, src, news); err != nil {
			fmt.Fprintf(env.Stderr, "om anchor: %v\n", err)
			return ExitFailure
		}
	}
	fmt.Fprintf(env.Stdout, "%d papers of %d editions. %d new links, %d linked before, %d review items, %d segments with no problem.\n", n, len(done), links, have, len(items), missing)
	if o.missing {
		fmt.Fprintf(env.Stdout, "%d new problems that link to their paper.\n", len(news))
	}
	return ExitOK
}

// editionOptions are the flags of om anchor for anchorEditions.
type editionOptions struct {
	from, to     int
	dry, missing bool
	pages        string
}

// newLink is a new problem at a position of an official paper: its only statement links to the page of the paper.
type newLink struct {
	at, name, lang, url string
	page                int
}

// paperURL gives the address and the language code of the official paper of the edition from the source, or "". A paper with no paper id, which holds all the positions of the edition, wins over a paper of one round, such as a preliminary round. The code is "en" when the paper has none.
func paperURL(e *content.Edition, source string) (url, lang string) {
	for _, all := range []bool{true, false} {
		for _, p := range e.Official {
			if p.Source == source && (!all || p.Paper == "") {
				return p.URL, cmpOr(p.Code, "en")
			}
		}
	}
	return "", ""
}

// readPages reads a pages file: an edition on each line, a tab, and the page of each problem from p1, such as "uk-bmo1/1975\t1 1 1 2 2". A line that starts with # is a comment.
func readPages(name string) (map[string][]int, error) {
	b, err := os.ReadFile(name) // #nosec G304 -- a file that the user names.
	if err != nil {
		return nil, err
	}
	out := map[string][]int{}
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		at, list, ok := strings.Cut(line, "\t")
		if !ok {
			return nil, fmt.Errorf("%s:%d: no tab", name, i+1)
		}
		for _, f := range strings.Fields(list) {
			pg, err := strconv.Atoi(f)
			if err != nil || pg < 1 {
				return nil, fmt.Errorf("%s:%d: %q is not a page", name, i+1, f)
			}
			out[strings.TrimSpace(at)] = append(out[strings.TrimSpace(at)], pg)
		}
	}
	return out, nil
}

// writeLinks writes a new problem file for each new link, with a new tag.
func writeLinks(root string, corpus *content.Corpus, src *content.SourceRecord, news []newLink) error {
	if len(news) == 0 {
		return nil
	}
	tags, err := mintTags(corpus, len(news))
	if err != nil {
		return err
	}
	for i, l := range news {
		c := &canon.Candidate{Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: l.lang, Name: l.name, URL: l.url}
		_, rule := canon.Rank(&canon.Candidate{Origin: src.Origin, Lang: l.lang}, original(corpus.Competition(strings.SplitN(l.at, "/", 2)[0])))
		b, err := canon.RenderLink(l.at, tags[i], c, l.page, rule)
		if err != nil {
			return fmt.Errorf("%s: %w", l.at, err)
		}
		r := &canon.Record{Primary: canon.Choice{At: l.at}}
		if err := writeNew(root, corpus, r, tags[i], b); err != nil {
			return fmt.Errorf("%s: %w", l.at, err)
		}
	}
	return nil
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
