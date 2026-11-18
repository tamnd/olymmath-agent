package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/store"
)

func init() {
	register(command{"matol", "write the seed rows of the Kazakhstan olympiads of matol.kz from its paper pages: the district, regional and junior stages and the Smagulov olympiad (om matol, then om import -comp kz-mo-district)", runMatol})
}

// matolSource is the id of matol.kz in the source register.
const matolSource = "matol-kz"

func runMatol(env Env, args []string) int {
	fs := newFlags("matol", env.Stderr)
	dir, remote := storeFlags(fs)
	out := fs.String("out", seed.DefaultDir(), "the directory of the seed rows")
	maxAge := fs.Duration("max-age", 30*24*time.Hour, "use a capture of the raw store that is younger than this, with no request; 0 asks the site for each page")
	cached := fs.Bool("cached", false, "read only the pages that the raw store has, and send no request")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om matol: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	fc.Delay = 2 * time.Second
	ctx := context.Background()
	caps, err := st.Captures(matolSource)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om matol: %v\n", err)
		return ExitFailure
	}
	have := map[string]store.Capture{}
	for _, c := range caps {
		if c.Local && time.Since(c.Fetched) < *maxAge && (have[c.URL].Fetched.IsZero() || c.Fetched.After(have[c.URL].Fetched)) {
			have[c.URL] = c
		}
	}
	read := func(p string) ([]byte, error) {
		u := archive.MatolBase + p
		if c, ok := have[u]; ok {
			return apexBlob(ctx, st, c.Hash)
		}
		if *cached {
			return nil, errMatolNotStored
		}
		res, err := fc.Get(ctx, matolSource, u, fetch.Options{MaxAge: *maxAge})
		if err != nil {
			return nil, err
		}
		return apexBlob(ctx, st, res.Capture.Hash)
	}
	why := map[string]int{}
	failed, papers := 0, 0
	var rows []seed.Row
	// The nodes are read from each root down, and a page is kept only when its breadcrumb is under the root, so the links of a page to other olympiads are not followed.
	seen := map[string]bool{}
	for _, root := range archive.MatolRoots {
		queue := []string{root.Node}
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			if seen[p] {
				continue
			}
			seen[p] = true
			b, err := read(p)
			if errors.Is(err, errMatolNotStored) {
				why["a page not in the raw store"]++
				continue
			}
			if err != nil {
				fmt.Fprintf(env.Stderr, "om matol: %s: %v\n", p, err)
				failed++
				continue
			}
			page := archive.ParseMatol(b)
			if r, _, ok := archive.MatolRootOf(page); p != root.Node && (!ok || r.Node != root.Node) {
				why["a page outside its root"]++
				continue
			}
			if strings.HasPrefix(p, "/nodes/") {
				queue = append(queue, page.Links...)
				continue
			}
			papers++
			for _, pr := range page.Problems {
				code, year, number, reason := archive.MatolPlace(page, pr.N)
				if reason != "" {
					why[reason]++
					continue
				}
				text, reason := archive.MatolText(pr)
				if reason != "" {
					why[reason]++
					continue
				}
				rows = append(rows, seed.Row{
					Source: matolSource, File: strings.TrimPrefix(p, "/"), Row: pr.N, ID: pr.Comment,
					Labels: []string{code}, Year: year, Number: number, Country: "KZ", Lang: "ru",
					Statement: text, Ref: archive.MatolBase + p,
				})
			}
		}
	}
	// Two problems at one position are two papers that the position cannot tell apart, so neither is placed.
	at := map[string]int{}
	key := func(r seed.Row) string { return r.Labels[0] + "/" + strconv.Itoa(r.Year) + "/" + r.Number }
	for _, r := range rows {
		at[key(r)]++
	}
	byCode := map[string]int{}
	err = seed.WriteFile(*out, matolSource, func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		for _, r := range rows {
			if at[key(r)] > 1 {
				why["several problems at one position"]++
				continue
			}
			if err := enc.Encode(r); err != nil {
				return err
			}
			byCode[r.Labels[0]]++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(env.Stderr, "om matol: %v\n", err)
		return ExitFailure
	}
	n := 0
	for _, code := range slices.Sorted(maps.Keys(byCode)) {
		fmt.Fprintf(env.Stdout, "%s: %d rows\n", code, byCode[code])
		n += byCode[code]
	}
	for _, r := range slices.Sorted(maps.Keys(why)) {
		fmt.Fprintf(env.Stdout, "left out: %d, %s\n", why[r], r)
	}
	fmt.Fprintf(env.Stdout, "%d papers, %d seed rows, %d failed\n", papers, n, failed)
	if failed > 0 {
		return ExitFailure
	}
	return ExitOK
}

// errMatolNotStored is a page that om matol -cached does not have.
var errMatolNotStored = errors.New("not in the raw store")
