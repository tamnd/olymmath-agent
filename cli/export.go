package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/export"
)

func init() {
	register(command{"export", "write the Parquet export for Hugging Face, one config for each licence zone (om export hf)", runExport})
	register(command{"publish", "push the Parquet export to Hugging Face (om publish hf). It runs only with -confirm and HF_TOKEN", runPublish})
}

func runExport(env Env, args []string) int {
	if len(args) == 0 || args[0] != "hf" {
		fmt.Fprintln(env.Stderr, "Usage: om export hf [flags]. Run om export hf -h to see the flags.")
		return ExitUsage
	}
	fs := newFlags("export hf", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	out := fs.String("out", "build/hf", "the output directory. The export replaces it")
	url := fs.String("url", "https://olymmath.tamnd.com", "the public URL of the site, which the rows link to")
	commit := fs.String("commit", "", "the commit of the content repository, for the card")
	if err := fs.Parse(args[1:]); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om export hf: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	st, err := export.Write(corpus, *out, export.Options{URL: *url, Commit: *commit})
	if err != nil {
		fmt.Fprintf(env.Stderr, "om export hf: %v\n", err)
		return ExitFailure
	}
	var parts []string
	for _, c := range export.Configs {
		parts = append(parts, fmt.Sprintf("%s %d", c.Name, st.Rows[c.Name]))
	}
	fmt.Fprintf(env.Stdout, "%s rows in %s; %d texts left out for ai-train=no\n", strings.Join(parts, ", "), *out, st.LeftOut[export.LeftAITrain])
	return ExitOK
}

// hfCommand is the program that uploads to Hugging Face. A test changes it.
var hfCommand = "hf"

func runPublish(env Env, args []string) int {
	if len(args) == 0 || args[0] != "hf" {
		fmt.Fprintln(env.Stderr, "Usage: om publish hf [flags]. Run om publish hf -h to see the flags.")
		return ExitUsage
	}
	fs := newFlags("publish hf", env.Stderr)
	dir := fs.String("dir", "build/hf", "the export from om export hf")
	repo := fs.String("repo", "tamnd/olymmath", "the dataset repository on Hugging Face")
	confirm := fs.Bool("confirm", false, "push the export. Without it, om publish hf only says what it would push")
	if err := fs.Parse(args[1:]); err != nil {
		return ExitUsage
	}
	if _, err := os.Stat(*dir + "/README.md"); err != nil {
		fmt.Fprintf(env.Stderr, "om publish hf: %s has no export. Run om export hf first.\n", *dir)
		return ExitUsage
	}
	if !*confirm {
		fmt.Fprintf(env.Stderr, "om publish hf: this pushes %s to the public dataset %s. Run it again with -confirm to push.\n", *dir, *repo)
		return ExitUsage
	}
	if os.Getenv("HF_TOKEN") == "" {
		fmt.Fprintln(env.Stderr, "om publish hf: set HF_TOKEN to a token that can write to the dataset")
		return ExitUsage
	}
	// The hf program of huggingface_hub reads HF_TOKEN from the environment, so the token is not on the command line.
	cmd := exec.Command(hfCommand, "upload", *repo, *dir, ".", "--repo-type", "dataset", "--commit-message", "Weekly export of olymmath") // #nosec G204 -- the flags come from the user.
	cmd.Stdout, cmd.Stderr = env.Stdout, env.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(env.Stderr, "om publish hf: %v\n", err)
		return ExitFailure
	}
	return ExitOK
}
