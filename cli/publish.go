package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/publish"
)

func runSiteSplit(env Env, args []string) int {
	fs := newFlags("site split", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	in := fs.String("in", "build/site", "the built site. The static assets stay here")
	out := fs.String("out", "build/r2", "the directory for the files that go to R2")
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
	return ExitOK
}

func runSiteUpload(env Env, args []string) int {
	fs := newFlags("site upload", env.Stderr)
	dir := fs.String("dir", "build/r2", "the directory to upload")
	bucket := fs.String("bucket", "olymmath-site", "the R2 bucket")
	workers := fs.Int("workers", 8, "the number of requests at the same time")
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
	res, err := publish.Sync(context.Background(), r, *dir, *workers)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om site upload: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d files, %d uploaded, %d deleted in R2 bucket %s\n", res.Files, res.Put, res.Deleted, *bucket)
	return ExitOK
}
