package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

func init() {
	register(command{"check", "check the content files and report the errors", runCheck})
	register(command{"site", "build the site, split it into its outlets and upload the R2 part (om site build, om site split, om site upload)", runSite})
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
	if len(findings) > 0 {
		return ExitFindings
	}
	return ExitOK
}

func runSite(env Env, args []string) int {
	if len(args) > 0 && args[0] == "split" {
		return runSiteSplit(env, args[1:])
	}
	if len(args) > 0 && args[0] == "upload" {
		return runSiteUpload(env, args[1:])
	}
	if len(args) == 0 || args[0] != "build" {
		fmt.Fprintln(env.Stderr, "Usage: om site build|split|upload [flags]. Run om site build -h to see the flags.")
		return ExitUsage
	}
	fs := newFlags("site build", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	out := fs.String("out", "build/site", "the output directory. The build replaces it")
	url := fs.String("url", "https://olymmath.tamnd.com", "the public URL of the site")
	base := fs.String("base", "/", "the path prefix of the links, for example /olymmath/ for the GitHub Pages mirror")
	commit := fs.String("commit", "", "the commit of the content repository")
	date := fs.String("date", "now", "the build date, now or YYYY-MM-DD. A fixed date gives the same output for the same input")
	if err := fs.Parse(args[1:]); err != nil {
		return ExitUsage
	}
	when := time.Now().UTC()
	if *date != "now" {
		t, err := time.Parse("2006-01-02", *date)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om site build: the date %q is not now or YYYY-MM-DD\n", *date)
			return ExitUsage
		}
		when = t
	}
	corpus, findings := content.Load(*root)
	if len(findings) > 0 {
		fmt.Fprintf(env.Stderr, "om site build: the content has %d findings. Run om check to see them.\n", len(findings))
		return ExitFindings
	}
	stats, err := site.Build(corpus, site.Options{Out: *out, URL: *url, Base: *base, Commit: *commit, Version: env.Version, Date: when})
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
