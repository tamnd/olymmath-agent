package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/coverage"
	"github.com/tamnd/olymmath-agent/seed"
)

func init() {
	register(command{"seed", "read the datasets of a source from the raw store and write its seed rows (om seed hf-omni-math)", runSeed})
	register(command{"coverage", "write the coverage report from the register, the content and the seed rows", runCoverage})
}

func runSeed(env Env, args []string) int {
	fs := newFlags("seed", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	out := fs.String("out", seed.DefaultDir(), "the directory of the seed rows")
	wave := fs.Int("wave", -1, "read each source of this wave that has a manifest in the raw store")
	limit := fs.Int("limit", 0, "read at most this number of rows of each file; 0 means all rows")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) == 0 && *wave < 0 {
		fmt.Fprintln(env.Stderr, "usage: om seed SOURCE... | om seed -wave N")
		return ExitUsage
	}
	corpus, _ := content.Load(*root)
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om seed: %v\n", err)
		return ExitFailure
	}
	ids := pos
	if *wave >= 0 {
		have, _ := st.Sources()
		ok := map[string]bool{}
		for _, h := range have {
			ok[h] = true
		}
		for _, s := range corpus.Sources {
			if s.Wave == *wave && ok[s.ID] {
				ids = append(ids, s.ID)
			}
		}
	}
	r := &seed.Runner{Store: st, Agent: fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version), Token: os.Getenv("HF_TOKEN"), Limit: *limit, Log: env.Stderr}
	code := ExitOK
	for _, id := range ids {
		start := time.Now()
		s, err := r.Write(context.Background(), *out, id)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om seed: %s: %v\n", id, err)
			code = ExitFailure
			continue
		}
		fmt.Fprintf(env.Stdout, "%s: %d files, %d rows read, %d rows kept, %d MB over the network, %s\n", id, s.Files, s.Read, s.Kept, s.Fetched>>20, time.Since(start).Round(time.Second))
	}
	return code
}

func runCoverage(env Env, args []string) int {
	fs := newFlags("coverage", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	out := fs.String("out", "", "the directory of the report; the default is reports/coverage in the content repository")
	year := fs.Int("year", time.Now().Year(), "the last year of a competition that is still held")
	top := fs.Int("unknown", 100, "the number of unknown labels in the Markdown report")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	corpus, fsx := content.Load(*root)
	if len(corpus.Competitions) == 0 {
		fmt.Fprintf(env.Stderr, "om coverage: %s has no competition register\n", *root)
		for _, f := range fsx {
			fmt.Fprintln(env.Stderr, f)
		}
		return ExitFailure
	}
	al, err := seed.LoadAliases(*root, corpus)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om coverage: %v\n", err)
		return ExitFailure
	}
	sources, err := seed.Sources(*dir)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om coverage: %v\n", err)
		return ExitFailure
	}
	rep, err := coverage.Build(corpus, al, *year, func(fn func(seed.Row) error) error {
		for _, s := range sources {
			if err := seed.Read(*dir, s, fn); err != nil {
				return fmt.Errorf("%s: %w", s, err)
			}
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(env.Stderr, "om coverage: %v\n", err)
		return ExitFailure
	}
	if *out == "" {
		*out = filepath.Join(*root, "reports", "coverage")
	}
	if err := os.MkdirAll(*out, 0o750); err != nil {
		fmt.Fprintf(env.Stderr, "om coverage: %v\n", err)
		return ExitFailure
	}
	files := []struct {
		name  string
		write func(f *os.File) error
	}{
		{"coverage.md", func(f *os.File) error { return rep.Markdown(f, *top) }},
		{"coverage.json", func(f *os.File) error { return rep.JSON(f) }},
		{"unknown-competitions.tsv", func(f *os.File) error { return rep.UnknownTSV(f) }},
	}
	for _, x := range files {
		f, err := os.Create(filepath.Join(*out, x.name))
		if err == nil {
			err = x.write(f)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "om coverage: %s: %v\n", x.name, err)
			return ExitFailure
		}
	}
	t := rep.Totals
	fmt.Fprintf(env.Stdout, "%d expected positions: %d with a record, %d with a candidate, %d with no candidate. %d unknown labels. The report is in %s.\n", t.Expected, t.C0, t.Candidate, t.NoCandidate, len(rep.Unknown), *out)
	return ExitOK
}
