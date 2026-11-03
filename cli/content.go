package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

func init() {
	register(command{"check", "check the content files and report the errors", runCheck})
	register(command{"site", "build the site, split it into its outlets, pack the R2 part and upload it (om site build, om site split, om site pack, om site release, om site upload)", runSite})
}

func newFlags(name string, w io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet("om "+name, flag.ContinueOnError)
	fs.SetOutput(w)
	return fs
}

func runCheck(env Env, args []string) int {
	fs := newFlags("check", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	quiet := fs.Bool("q", false, "print only the findings")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	m, err := site.NewMarkup(nil)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om check: %v\n", err)
		return ExitFailure
	}
	// Each statement and each answer must render. KaTeX finds the formulas that do not parse.
	for _, p := range corpus.Problems() {
		for _, text := range []string{p.Statement, p.Answer} {
			_, errs := m.HTML(text)
			for _, e := range errs {
				if errors.Is(e, site.ErrFigure) {
					findings = append(findings, content.Finding{Path: p.Path, Level: content.Warning, Message: e.Error()})
					continue
				}
				findings = append(findings, content.Finding{Path: p.Path, Message: "a formula does not render: " + e.Error()})
			}
		}
	}
	for _, f := range findings {
		fmt.Fprintln(env.Stdout, f)
	}
	if !*quiet {
		fmt.Fprintf(env.Stdout, "%d competitions, %d editions, %d problems, %d findings\n", len(corpus.Competitions), len(corpus.Editions()), len(corpus.Problems()), len(findings))
	}
	if findings.Errors() > 0 {
		return ExitFindings
	}
	return ExitOK
}

func runSite(env Env, args []string) int {
	if len(args) > 0 && args[0] == "split" {
		return runSiteSplit(env, args[1:])
	}
	if len(args) > 0 && args[0] == "pack" {
		return runSitePack(env, args[1:])
	}
	if len(args) > 0 && args[0] == "release" {
		return runSiteRelease(env, args[1:])
	}
	if len(args) > 0 && args[0] == "upload" {
		return runSiteUpload(env, args[1:])
	}
	if len(args) == 0 || args[0] != "build" {
		fmt.Fprintln(env.Stderr, "Usage: om site build|split|pack|release|upload [flags]. Run om site build -h to see the flags.")
		return ExitUsage
	}
	fs := newFlags("site build", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	out := fs.String("out", "build/site", "the output directory. The build replaces it")
	opts := siteFlags(fs)
	if err := fs.Parse(args[1:]); err != nil {
		return ExitUsage
	}
	o, code := opts(env, "om site build")
	if code != ExitOK {
		return code
	}
	o.Out = *out
	corpus, code := loadForSite(env, "om site build", *root)
	if code != ExitOK {
		return code
	}
	stats, err := site.Build(corpus, o)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om site build: %v\n", err)
		return ExitFailure
	}
	for _, e := range stats.Errors {
		fmt.Fprintf(env.Stderr, "warning: %v\n", e)
	}
	fmt.Fprintf(env.Stdout, "%d pages, %d files, %d problems in %s\n", stats.Pages, stats.Files, stats.Problems, strings.TrimSuffix(*out, "/"))
	return ExitOK
}

// siteFlags adds the flags of a build of the site to fs, but -content and -out. The function it gives reads them after the parse.
func siteFlags(fs *flag.FlagSet) func(env Env, name string) (site.Options, int) {
	url := fs.String("url", "https://olymmath.tamnd.com", "the public URL of the site")
	base := fs.String("base", "/", "the path prefix of the links, for example /olymmath/ for the GitHub Pages mirror")
	commit := fs.String("commit", "", "the commit of the content repository")
	date := fs.String("date", "now", "the build date, now or YYYY-MM-DD. A fixed date gives the same output for the same input")
	cache := fs.String("cache", defaultCache(), "the directory that keeps the rendered formulas from one build to the next, or off. The output is the same with and without it")
	return func(env Env, name string) (site.Options, int) {
		when := time.Now().UTC()
		if *date != "now" {
			t, err := time.Parse("2006-01-02", *date)
			if err != nil {
				fmt.Fprintf(env.Stderr, "%s: the date %q is not now or YYYY-MM-DD\n", name, *date)
				return site.Options{}, ExitUsage
			}
			when = t
		}
		c := *cache
		if c == "off" {
			c = ""
		}
		return site.Options{URL: *url, Base: *base, Commit: *commit, Version: env.Version, Date: when, Cache: c}, ExitOK
	}
}

// loadForSite loads the content for a build of the site. A build needs content with no finding.
func loadForSite(env Env, name, root string) (*content.Corpus, int) {
	corpus, findings := content.Load(root)
	if len(findings) > 0 {
		fmt.Fprintf(env.Stderr, "%s: the content has %d findings. Run om check to see them.\n", name, len(findings))
		return nil, ExitFindings
	}
	return corpus, ExitOK
}

// defaultCache is the cache directory of om site build: olymmath/site in the cache directory of the user, or off when the system has none.
func defaultCache() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "off"
	}
	return filepath.Join(dir, "olymmath", "site")
}
