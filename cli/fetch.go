package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/hf"
)

func init() {
	register(command{"fetch", "harvest a source of the source register into the raw store (om fetch hf-omni-math)", runFetch})
}

func runFetch(env Env, args []string) int {
	fs := newFlags("fetch", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	localMax := fs.Int64("local-max", 64, "the largest file in MB that goes to the local disk; a larger file goes to R2 with -r2, or else stays a reference")
	remoteMax := fs.Int64("remote-max", 500, "the largest file in MB that goes to R2; a larger file stays a reference to its pinned URL")
	maxFiles := fs.Int("max-files", 100, "the largest number of data files to get one by one; a dataset with more files uses the Parquet form of the hub")
	wave := fs.Int("wave", -1, "harvest each source of this wave that the tool can harvest")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) == 0 && *wave < 0 {
		fmt.Fprintln(env.Stderr, "usage: om fetch [-r2] SOURCE... | om fetch -wave N")
		return ExitUsage
	}
	corpus, _ := content.Load(*root)
	if len(corpus.Sources) == 0 {
		fmt.Fprintf(env.Stderr, "om fetch: %s has no source register\n", *root)
		return ExitFailure
	}
	var sources []*content.SourceRecord
	for _, id := range pos {
		s := corpus.Source(id)
		if s == nil {
			fmt.Fprintf(env.Stderr, "om fetch: %q is not in the source register\n", id)
			return ExitUsage
		}
		sources = append(sources, s)
	}
	if *wave >= 0 {
		for _, s := range corpus.Sources {
			if _, ok := hf.DatasetID(s.URL); s.Wave == *wave && s.Access == "hf" && ok {
				sources = append(sources, s)
			}
		}
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om fetch: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	fc.Delay = 200 * time.Millisecond
	h := &hf.Harvest{Hub: &hf.Client{Agent: fc.Agent, Token: os.Getenv("HF_TOKEN")}, Fetch: fc, LocalMax: *localMax << 20, RemoteMax: *remoteMax << 20, MaxFiles: *maxFiles, Log: env.Stdout}
	code := ExitOK
	for _, s := range sources {
		if ps := officialPapers(corpus, s.ID); len(ps) > 0 {
			fc.Delay = 2 * time.Second
			n, err := fetchPapers(context.Background(), fc, s.ID, ps)
			fmt.Fprintf(env.Stdout, "# %s: %d of %d official papers in the store\n", s.ID, n, len(ps))
			if err != nil {
				fmt.Fprintf(env.Stderr, "om fetch: %s: %v\n", s.ID, err)
				code = ExitFailure
			}
			continue
		}
		id, ok := hf.DatasetID(s.URL)
		if s.Access != "hf" || !ok {
			fmt.Fprintf(env.Stderr, "om fetch: %s: the adapter for the access %q comes in a later milestone\n", s.ID, s.Access)
			code = max(code, ExitLater)
			continue
		}
		fmt.Fprintf(env.Stdout, "# %s (%s)\n", s.ID, id)
		m, c, err := h.Run(context.Background(), s.ID, id)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om fetch: %s: %v\n", s.ID, err)
			code = ExitFailure
			continue
		}
		fmt.Fprintf(env.Stdout, "manifest %s: %s@%s, %d files, licence %q\n", c.Hash, m.Dataset, m.Revision, len(m.Files), m.Licence)
	}
	return code
}

// officialPapers gives the URLs of the official papers of the editions that come from a source, in the order of the editions.
func officialPapers(corpus *content.Corpus, source string) []string {
	var out []string
	for _, c := range corpus.Competitions {
		for _, e := range c.Editions {
			for _, o := range e.Official {
				if o.Source == source {
					out = append(out, o.URL)
				}
			}
		}
	}
	return out
}

// fetchPapers gets each paper into the store, one request at a time with the delay of the client. A paper that fails does not stop the others. It returns the number of papers in the store and the last error.
func fetchPapers(ctx context.Context, fc *fetch.Client, source string, urls []string) (int, error) {
	var n int
	var last error
	for _, u := range urls {
		if _, err := fc.Get(ctx, source, u, fetch.Options{}); err != nil {
			last = err
			continue
		}
		n++
	}
	return n, last
}
