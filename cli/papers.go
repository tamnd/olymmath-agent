package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"papers", "record the official papers of each IMO edition in all languages (om papers imo-official)", runPapers})
}

func runPapers(env Env, args []string) int {
	fs := newFlags("papers", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 1959, "the first year")
	to := fs.Int("to", time.Now().Year(), "the last year")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 || pos[0] != "imo-official" {
		fmt.Fprintln(env.Stderr, "usage: om papers [-from YEAR] [-to YEAR] imo-official")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om papers: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	comp := corpus.Competition("imo")
	if comp == nil || corpus.Source(pos[0]) == nil {
		fmt.Fprintln(env.Stderr, "om papers: the content has no competition imo or no source imo-official")
		return ExitUsage
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om papers: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	ctx := context.Background()
	code := ExitOK
	var editions, papers int
	for _, e := range comp.Editions {
		if e.Year < *from || e.Year > *to {
			continue
		}
		page, err := get(ctx, fc, pos[0], official.YearPage(e.Year))
		if err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", e.Locator(), err)
			code = ExitFailure
			continue
		}
		langs := official.Languages(page)
		if len(langs) == 0 {
			continue
		}
		if err := setOfficial(filepath.Join(*root, filepath.FromSlash(e.Path)), langs); err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", e.Path, err)
			return ExitFailure
		}
		editions++
		papers += len(langs)
	}
	fmt.Fprintf(env.Stdout, "%d editions, %d official papers.\n", editions, papers)
	return code
}

// get gets a page into the store and returns its text.
func get(ctx context.Context, fc *fetch.Client, source, url string) (string, error) {
	text, _, err := getCapture(ctx, fc, source, url)
	return text, err
}

// getCapture gets a page into the store and returns its text and the hash of its capture.
func getCapture(ctx context.Context, fc *fetch.Client, source, url string) (text, hash string, err error) {
	res, err := fc.Get(ctx, source, url, fetch.Options{})
	if err != nil {
		return "", "", err
	}
	rc, err := fc.Store.Get(ctx, res.Capture.Hash)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	return string(b), res.Capture.Hash, err
}

// setOfficial sets the list of official papers of an edition file. The other fields keep their order.
func setOfficial(file string, langs []content.OfficialPaper) error {
	b, err := os.ReadFile(file) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return err
	}
	var list yaml.Node
	if err := list.Encode(langs); err != nil {
		return err
	}
	m := doc.Content[0]
	found := false
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == "official" {
			m.Content[i+1], found = &list, true
		}
	}
	if !found {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "official"}, &list)
	}
	f, err := os.Create(file) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		_ = f.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
