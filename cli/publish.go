package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/publish"
	"github.com/tamnd/olymmath-agent/site"
)

func runSiteSplit(env Env, args []string) int {
	fs := newFlags("site split", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	in := fs.String("in", "build/site", "the built site. The static assets stay here")
	out := fs.String("out", "build/r2", "the directory for the files that go to R2")
	maxAssets := fs.Int("max-assets", 19000, "the most static asset files. The free plan of Workers allows 20000 for each version, and om site pack adds one index for each folder")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	tiers, err := publish.LoadTiers(filepath.Join(*root, "registers", "site-tiers.yaml"))
	if err != nil {
		fmt.Fprintf(env.Stderr, "om site split: %v\n", err)
		return ExitFailure
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om site split: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	deleted, moved, err := publish.Split(*in, *out, corpus, tiers)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om site split: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d redirect pages deleted, %d files moved to %s\n", deleted, moved, *out)
	fit, codes, err := publish.Fit(*in, *out, corpus, *maxAssets)
	if len(codes) > 0 {
		fmt.Fprintf(env.Stdout, "%d more files moved to %s to keep the static assets at most %d: %s\n", fit, *out, *maxAssets, strings.Join(codes, ", "))
	}
	if err != nil {
		fmt.Fprintf(env.Stderr, "om site split: %v\n", err)
		return ExitFailure
	}
	return ExitOK
}

// runSiteRelease builds the site in memory and writes the result of om site build, om site split and om site pack: the static assets with the index of each pack in -out, and the packs in -pack. The files that go to R2 and the redirect pages never touch the disk.
func runSiteRelease(env Env, args []string) int {
	fs := newFlags("site release", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	out := fs.String("out", "build/site", "the directory for the static assets. The release replaces it")
	pack := fs.String("pack", "build/pack", "the directory for the packs. The release replaces it, and om site upload sends it")
	maxAssets := fs.Int("max-assets", 19000, "the most static asset files, as in om site split")
	opts := siteFlags(fs)
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	fail := func(err error) int {
		fmt.Fprintf(env.Stderr, "om site release: %v\n", err)
		return ExitFailure
	}
	o, code := opts(env, "om site release")
	if code != ExitOK {
		return code
	}
	tiers, err := publish.LoadTiers(filepath.Join(*root, "registers", "site-tiers.yaml"))
	if err != nil {
		return fail(err)
	}
	corpus, code := loadForSite(env, "om site release", *root)
	if code != ExitOK {
		return code
	}
	files, stats, err := site.Render(corpus, o)
	if err != nil {
		return fail(err)
	}
	for _, e := range stats.Errors {
		fmt.Fprintf(env.Stderr, "warning: %v\n", e)
	}
	fmt.Fprintf(env.Stdout, "%d pages, %d files, %d problems\n", stats.Pages, stats.Files, stats.Problems)
	plan, err := publish.PlanSplit(files, corpus, tiers, *maxAssets)
	fmt.Fprintf(env.Stdout, "%d redirect pages dropped, %d files to R2\n", plan.Deleted, plan.Moved)
	if len(plan.Fit) > 0 {
		fmt.Fprintf(env.Stdout, "%d more files to R2 to keep the static assets at most %d: %s\n", plan.Fitted, *maxAssets, strings.Join(plan.Fit, ", "))
	}
	if err != nil {
		return fail(err)
	}
	res, err := publish.Release(files, plan, *out, *pack)
	if err != nil {
		return fail(err)
	}
	fmt.Fprintf(env.Stdout, "%d static files in %s, %d files in %d packs in %s\n", len(files)-plan.Deleted-res.Files, strings.TrimSuffix(*out, "/"), res.Files, res.Packs, strings.TrimSuffix(*pack, "/"))
	return ExitOK
}

func runSitePack(env Env, args []string) int {
	fs := newFlags("site pack", env.Stderr)
	in := fs.String("in", "build/r2", "the files that go to R2, from om site split")
	out := fs.String("out", "build/pack", "the directory for the packs. om site upload sends it")
	index := fs.String("index", "build/site/_pack", "the directory for the index of each pack. It is in the static assets, so that the Worker reads it with no R2 request")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	res, err := publish.Pack(*in, *out, *index)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om site pack: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d files in %d packs in %s\n", res.Files, res.Packs, *out)
	return ExitOK
}

func runSiteUpload(env Env, args []string) int {
	fs := newFlags("site upload", env.Stderr)
	dir := fs.String("dir", "build/pack", "the directory to upload")
	bucket := fs.String("bucket", "olymmath-site", "the R2 bucket")
	workers := fs.Int("workers", 8, "the number of requests at the same time")
	keep := fs.Bool("keep", false, "keep the objects of the files that are gone; a deploy uploads with -keep before the new Worker is live and without it after")
	budget := fs.Duration("budget", 0, "send no new file after this time, for example 40m, and leave the rest to the next run; 0 has no limit")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	r := &publish.R2{Account: os.Getenv("CLOUDFLARE_ACCOUNT_ID"), Bucket: *bucket, Token: os.Getenv("CLOUDFLARE_API_TOKEN")}
	if r.Account == "" || r.Token == "" {
		fmt.Fprintln(env.Stderr, "om site upload: set CLOUDFLARE_ACCOUNT_ID and CLOUDFLARE_API_TOKEN")
		return ExitUsage
	}
	if err := os.MkdirAll(*dir, 0o755); err != nil { // #nosec G301 -- the site is public.
		fmt.Fprintf(env.Stderr, "om site upload: %v\n", err)
		return ExitFailure
	}
	var until time.Time
	if *budget > 0 {
		until = time.Now().Add(*budget)
	}
	res, err := publish.SyncWith(context.Background(), r, *dir, publish.SyncOptions{Workers: *workers, Until: until, Keep: *keep})
	if err != nil {
		fmt.Fprintf(env.Stderr, "om site upload: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d files, %d uploaded, %d deleted in R2 bucket %s\n", res.Files, res.Put, res.Deleted, *bucket)
	if res.Left > 0 {
		fmt.Fprintf(env.Stdout, "The time budget is over. %d changes are left for the next run.\n", res.Left)
	}
	return ExitOK
}
