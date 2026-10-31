// Package cli is the command tree of om. Each command is a function that gets its arguments and returns an exit code.
//
// The exit codes are the same for each command: 0 when the command succeeds, 1 when it finds errors in the content, 2 for a usage error, 3 for a command that a later milestone adds, and 4 for any other failure.
package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// The exit codes of om.
const (
	ExitOK       = 0
	ExitFindings = 1
	ExitUsage    = 2
	ExitLater    = 3
	ExitFailure  = 4
)

// Env is what a command can use: the version of the binary and the two output streams.
type Env struct {
	Version string
	Stdout  io.Writer
	Stderr  io.Writer
}

type command struct {
	name    string
	summary string
	run     func(env Env, args []string) int
}

var commands = map[string]command{}

func register(c command) { commands[c.name] = c }

func init() {
	register(command{"help", "show the commands", runHelp})
	register(command{"version", "show the version of om", runVersion})
}

// Run runs the command that args names and returns its exit code.
func Run(args []string, env Env) int {
	if len(args) == 0 {
		usage(env.Stderr)
		return ExitUsage
	}
	name := args[0]
	if name == "-h" || name == "-help" || name == "--help" {
		name = "help"
	}
	c, ok := commands[name]
	if !ok {
		fmt.Fprintf(env.Stderr, "om: unknown command %q. Run om help to see the commands.\n", args[0])
		return ExitUsage
	}
	return c.run(env, args[1:])
}

func runHelp(env Env, _ []string) int {
	usage(env.Stdout)
	return ExitOK
}

func runVersion(env Env, _ []string) int {
	fmt.Fprintf(env.Stdout, "om %s\n", env.Version)
	return ExitOK
}

func usage(w io.Writer) {
	names := make([]string, 0, len(commands))
	width := 0
	for n := range commands {
		names = append(names, n)
		width = max(width, len(n))
	}
	sort.Strings(names)
	var b strings.Builder
	b.WriteString("om is the toolchain of olymmath.\n\nUsage: om <command> [flags]\n\nCommands:\n")
	for _, n := range names {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, n, commands[n].summary)
	}
	b.WriteString("\nRun om <command> -h to see the flags of a command.\n")
	fmt.Fprint(w, b.String())
}
