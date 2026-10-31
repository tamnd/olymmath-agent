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
	h := &hf.Harvest{Hub: &hf.Client{Agent: fc.Agent, Token: os.Getenv("HF_TOKEN")}, Fetch: fc, LocalMax: *localMax << 20, MaxFiles: *maxFiles, Log: env.Stdout}
	code := ExitOK
	for _, s := range sources {
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
