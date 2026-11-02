// Command om is the toolchain of olymmath. It builds the site, checks the content and, in later milestones, harvests the sources and grades the solutions.
package main

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/tamnd/olymmath-agent/cli"
)

// Version is the release of the binary. The release workflow sets it with -ldflags.
var Version = "dev"

func main() {
	// OM_CPUPROFILE names a file for a CPU profile of the run, for go tool pprof.
	if p := os.Getenv("OM_CPUPROFILE"); p != "" {
		stop, err := startProfile(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "om: %v\n", err)
			os.Exit(1)
		}
		code := cli.Run(os.Args[1:], cli.Env{Version: Version, Stdout: os.Stdout, Stderr: os.Stderr})
		stop()
		os.Exit(code)
	}
	os.Exit(cli.Run(os.Args[1:], cli.Env{Version: Version, Stdout: os.Stdout, Stderr: os.Stderr}))
}

func startProfile(p string) (func(), error) {
	f, err := os.Create(p) // #nosec G304 G703 -- the user names the profile file.
	if err != nil {
		return nil, err
	}
	if err := pprof.StartCPUProfile(f); err != nil {
		_ = f.Close()
		return nil, err
	}
	return func() {
		pprof.StopCPUProfile()
		_ = f.Close()
	}, nil
}
