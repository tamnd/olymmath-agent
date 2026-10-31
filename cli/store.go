package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/tamnd/olymmath-agent/store"
)

func init() {
	register(command{"store", "use the raw store (om store ls, put, cat, push, sync, restore)", runStore})
}

// storeFlags adds the flags that each command of the raw store has.
func storeFlags(fs *flag.FlagSet) (dir *string, remote *bool) {
	dir = fs.String("raw", store.DefaultDir(), "the directory of the raw store")
	remote = fs.Bool("r2", false, "use the R2 bucket, with CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN")
	return
}

func openStore(dir string, remote bool) (*store.Store, error) {
	var r store.Remote
	if remote {
		r2 := store.R2FromEnv()
		if r2 == nil {
			return nil, fmt.Errorf("-r2 needs CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN")
		}
		r = r2
	}
	return store.Open(dir, r)
}

const storeUsage = `usage:
  om store ls [SOURCE]                    list the sources, or the captures of a source
  om store put -source S [-url U] FILE    add a file to the store
  om store cat HASH                       write the bytes of a hash to stdout
  om store push [-evict] HASH...          send local bytes to the R2 bucket
  om store sync                           send each new local blob and each index file to the R2 bucket
  om store restore                        get the index files from the R2 bucket
`

func runStore(env Env, args []string) int {
	if len(args) == 0 {
		fmt.Fprint(env.Stderr, storeUsage)
		return ExitUsage
	}
	sub := args[0]
	fs := newFlags("store "+sub, env.Stderr)
	dir, remote := storeFlags(fs)
	source := fs.String("source", "", "the source code of the capture")
	url := fs.String("url", "", "the URL of the capture")
	licence := fs.String("licence", "", "the licence of the capture")
	evict := fs.Bool("evict", false, "remove the local bytes after the push")
	pos, err := parseMixed(fs, args[1:])
	if err != nil {
		return ExitUsage
	}
	if sub == "push" || sub == "sync" || sub == "restore" {
		*remote = true
	}
	s, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om store: %v\n", err)
		return ExitFailure
	}
	ctx := context.Background()
	fail := func(err error) int {
		fmt.Fprintf(env.Stderr, "om store %s: %v\n", sub, err)
		return ExitFailure
	}
	switch sub {
	case "ls":
		if len(pos) == 0 {
			srcs, err := s.Sources()
			if err != nil {
				return fail(err)
			}
			for _, src := range srcs {
				cs, _ := s.Captures(src)
				fmt.Fprintf(env.Stdout, "%s\t%d\n", src, len(cs))
			}
			return ExitOK
		}
		cs, err := s.Captures(pos[0])
		if err != nil {
			return fail(err)
		}
		for _, c := range cs {
			fmt.Fprintf(env.Stdout, "%s\t%d\t%s\t%s\t%s\n", c.Hash, c.Size, where(c), c.Fetched.Format("2006-01-02"), c.URL)
		}
		return ExitOK
	case "put":
		if len(pos) != 1 || *source == "" {
			fmt.Fprint(env.Stderr, storeUsage)
			return ExitUsage
		}
		f, err := os.Open(pos[0])
		if err != nil {
			return fail(err)
		}
		defer func() { _ = f.Close() }()
		c, err := s.Put(f, store.Capture{Source: *source, URL: *url, Licence: *licence})
		if err != nil {
			return fail(err)
		}
		fmt.Fprintln(env.Stdout, c.Hash)
		return ExitOK
	case "cat":
		if len(pos) != 1 {
			fmt.Fprint(env.Stderr, storeUsage)
			return ExitUsage
		}
		rc, err := s.Get(ctx, pos[0])
		if err != nil {
			return fail(err)
		}
		defer func() { _ = rc.Close() }()
		if _, err := io.Copy(env.Stdout, rc); err != nil {
			return fail(err)
		}
		return ExitOK
	case "sync":
		st, err := s.Sync(ctx, env.Stderr)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(env.Stdout, "%d blobs sent (%d MB), %d already sent, %d index files\n", st.Blobs, st.Bytes>>20, st.Skipped, st.Indexes)
		return ExitOK
	case "restore":
		n, err := s.Restore(ctx)
		if err != nil {
			return fail(err)
		}
		fmt.Fprintf(env.Stdout, "%d index files restored\n", n)
		return ExitOK
	case "push":
		if len(pos) == 0 {
			fmt.Fprint(env.Stderr, storeUsage)
			return ExitUsage
		}
		for _, h := range pos {
			if err := s.Push(ctx, h); err != nil {
				return fail(err)
			}
			if *evict {
				if err := s.Evict(h); err != nil {
					return fail(err)
				}
			}
			fmt.Fprintf(env.Stdout, "pushed %s\n", h)
		}
		return ExitOK
	}
	fmt.Fprint(env.Stderr, storeUsage)
	return ExitUsage
}

func where(c store.Capture) string {
	var w []string
	if c.Local {
		w = append(w, "local")
	}
	if c.Remote {
		w = append(w, "r2")
	}
	if len(w) == 0 {
		return "ref"
	}
	return strings.Join(w, "+")
}
