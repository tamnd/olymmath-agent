package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/store"
)

func init() {
	register(command{"apex", "write the seed rows of the Ukrainian olympiads of apex-minds.org from its full records, or with -figures put the figures of its texts next to their problems (om apex, then om import -comp ua-umo-district, then om apex -figures)", runApex})
}

// apexSource is the id of Apex Minds in the source register.
const apexSource = "apex-minds"

func runApex(env Env, args []string) int {
	fs := newFlags("apex", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	out := fs.String("out", seed.DefaultDir(), "the directory of the seed rows")
	maxAge := fs.Duration("max-age", 30*24*time.Hour, "use a capture of the raw store that is younger than this, with no request; 0 asks the site for each file")
	limit := fs.Int("limit", 0, "read at most this number of records; 0 means all")
	cached := fs.Bool("cached", false, "read only the records that the raw store has, and send no request")
	figures := fs.Bool("figures", false, "put each figure of the site that a text of the content links to next to its problem, as p93.fig1.png, and link the text to the file")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	fc.Delay = 2 * time.Second
	ctx := context.Background()
	if *figures {
		return apexFigures(ctx, env, fc, *root, *maxAge)
	}
	manifest, err := apexRead(ctx, fc, archive.ApexManifest, *maxAge)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
		return ExitFailure
	}
	slugs, err := archive.ApexSlugs(manifest)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
		return ExitFailure
	}
	if *limit > 0 && len(slugs) > *limit {
		slugs = slugs[:*limit]
	}
	placed := map[string]int{}
	why := map[string]int{}
	failed := 0
	var rows []seed.Row
	// The index of the store is read once: Store.Latest reads it for each URL.
	caps, err := st.Captures(apexSource)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
		return ExitFailure
	}
	have := map[string]store.Capture{}
	for _, c := range caps {
		if c.Local && time.Since(c.Fetched) < *maxAge {
			have[c.URL] = c
		}
	}
	for _, slug := range slugs {
		u := archive.ApexFull(slug)
		var b []byte
		if c, ok := have[u]; ok {
			b, err = apexBlob(ctx, st, c.Hash)
		} else if *cached {
			why["not in the raw store"]++
			continue
		} else {
			b, err = apexRead(ctx, fc, u, *maxAge)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "om apex: %s: %v\n", slug, err)
			failed++
			continue
		}
		r, err := archive.ParseApex(b)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om apex: %s: %v\n", slug, err)
			failed++
			continue
		}
		code, year, number, reason := archive.ApexPlace(r)
		if reason != "" {
			why[reason]++
			continue
		}
		rows = append(rows, apexRow(r, code, year, number))
	}
	// Two records at one position are two papers of one stage, as the papers of two regions, that the position cannot tell apart, so neither is placed.
	at := map[string]int{}
	key := func(r seed.Row) string { return r.Labels[0] + "/" + strconv.Itoa(r.Year) + "/" + r.Number }
	for _, r := range rows {
		at[key(r)]++
	}
	err = seed.WriteFile(*out, apexSource, func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		for _, r := range rows {
			if at[key(r)] > 1 {
				why["several records at one position"]++
				continue
			}
			if err := enc.Encode(r); err != nil {
				return err
			}
			placed[r.Labels[0]+"/"+strconv.Itoa(r.Year)]++
		}
		return nil
	})
	if err != nil {
		fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
		return ExitFailure
	}
	byCode := map[string]int{}
	for k, n := range placed {
		code, _, _ := strings.Cut(k, "/")
		byCode[code] += n
	}
	n := 0
	for _, code := range slices.Sorted(maps.Keys(byCode)) {
		fmt.Fprintf(env.Stdout, "%s: %d rows\n", code, byCode[code])
		n += byCode[code]
	}
	for _, r := range slices.Sorted(maps.Keys(why)) {
		fmt.Fprintf(env.Stdout, "left out: %d, %s\n", why[r], r)
	}
	fmt.Fprintf(env.Stdout, "%d records, %d seed rows, %d failed\n", len(slugs), n, failed)
	if failed > 0 {
		return ExitFailure
	}
	return ExitOK
}

// apexRow makes the seed row of a record placed at the code, the year and the number. The text keeps its language, Ukrainian, and links to the figures of the site until om apex -figures puts them next to the problem.
func apexRow(r archive.ApexRecord, code string, year int, number string) seed.Row {
	return seed.Row{
		Source: apexSource, File: "data_v2/full/" + r.Slug + ".json.gz", ID: r.PublicID,
		Labels: []string{code}, Year: year, Number: number, Country: "UA", Lang: "uk",
		Statement: archive.ApexText(r), Answer: archive.ApexAnswer(r), Solutions: len(r.Solutions),
		Ref: archive.ApexPage(r.PublicID),
	}
}

// apexRead gives the body of a file of the site from the raw store, and gets it when the store has no capture younger than maxAge.
func apexRead(ctx context.Context, fc *fetch.Client, u string, maxAge time.Duration) ([]byte, error) {
	res, err := fc.Get(ctx, apexSource, u, fetch.Options{MaxAge: maxAge})
	if err != nil {
		return nil, err
	}
	rc, err := fc.Store.Get(ctx, res.Capture.Hash)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

// apexBlob gives the bytes of a capture of the store.
func apexBlob(ctx context.Context, st *store.Store, hash string) ([]byte, error) {
	rc, err := st.Get(ctx, hash)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	return io.ReadAll(rc)
}

// apexFigures puts the figures of the site that the problem files link to next to their problems. A figure that is not a PNG or an SVG keeps its link and is reported.
func apexFigures(ctx context.Context, env Env, fc *fetch.Client, root string, maxAge time.Duration) int {
	corpus, findings := content.Load(root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om apex: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	code := ExitOK
	files, figs := 0, 0
	for _, p := range corpus.All() {
		full := filepath.Join(root, filepath.FromSlash(p.Path))
		b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
		if err != nil {
			fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
			code = ExitFailure
			continue
		}
		text := string(b)
		var refs []string
		for _, ref := range content.FigureRefs(text) {
			if _, ok := archive.IsApexImage(ref); ok && !slices.Contains(refs, ref) {
				refs = append(refs, ref)
			}
		}
		if len(refs) == 0 {
			continue
		}
		base := strings.TrimSuffix(path.Base(p.Path), ".md")
		next := apexNextFigure(filepath.Dir(full), base)
		local := map[string]string{}
		for _, ref := range refs {
			name, _ := archive.IsApexImage(ref)
			ext := strings.ToLower(path.Ext(name))
			if ext != ".png" && ext != ".svg" {
				fmt.Fprintf(env.Stderr, "om apex: %s: the figure %s is not a PNG or an SVG\n", p.Path, ref)
				code = ExitFailure
				continue
			}
			img, err := apexRead(ctx, fc, ref, maxAge)
			if err != nil {
				fmt.Fprintf(env.Stderr, "om apex: %s: %s: %v\n", p.Path, ref, err)
				code = ExitFailure
				continue
			}
			file := fmt.Sprintf("%s.fig%d%s", base, next, ext)
			next++
			if err := os.WriteFile(filepath.Join(filepath.Dir(full), file), img, 0o644); err != nil { // #nosec G306 G703 -- figures are public, and the name is made here from the record path.
				fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
				return ExitFailure
			}
			local[ref] = file
			figs++
		}
		if len(local) == 0 {
			continue
		}
		text = content.MapFigures(text, func(ref string) string { return cmpOr(local[ref], ref) })
		if err := os.WriteFile(full, []byte(text), 0o644); err != nil { // #nosec G306 G703 -- content files are public.
			fmt.Fprintf(env.Stderr, "om apex: %v\n", err)
			return ExitFailure
		}
		files++
	}
	fmt.Fprintf(env.Stdout, "%d figures put next to %d problems\n", figs, files)
	return code
}

// apexNextFigure gives the number of the next figure of the problem base in dir: one more than the highest number of its figure files.
func apexNextFigure(dir, base string) int {
	n := 1
	ms, _ := filepath.Glob(filepath.Join(dir, base+".fig*"))
	for _, m := range ms {
		s := strings.TrimPrefix(filepath.Base(m), base+".fig")
		s = strings.TrimSuffix(s, filepath.Ext(s))
		if k, err := strconv.Atoi(s); err == nil && k >= n {
			n = k + 1
		}
	}
	return n
}
