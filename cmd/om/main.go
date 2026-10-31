// Command om is the toolchain of olymmath. It builds the site, checks the content and, in later milestones, harvests the sources and grades the solutions.
package main

import (
	"os"

	"github.com/tamnd/olymmath-agent/cli"
)

// Version is the release of the binary. The release workflow sets it with -ldflags.
var Version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], cli.Env{Version: Version, Stdout: os.Stdout, Stderr: os.Stderr}))
}
