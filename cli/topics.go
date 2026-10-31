package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/tamnd/olymmath-agent/content"
)

func init() {
	register(command{"topics", "set the topics of problems from the letters of shortlist positions (om topics -dry)", runTopics})
}

func runTopics(env Env, args []string) int {
	fs := newFlags("topics", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dry := fs.Bool("dry", false, "print the changes and write no file")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om topics: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	var set, failed int
	for _, p := range corpus.All() {
		if len(p.Topics) > 0 {
			continue
		}
		topic := ""
		for _, a := range p.Appearances {
			if t := content.PositionTopic(corpus.Competition(a.Locator.Comp), a.Locator.Position); t != "" {
				topic = t
				break
			}
		}
		if topic == "" {
			continue
		}
		set++
		fmt.Fprintf(env.Stdout, "%s %s\n", p.Path, topic)
		if *dry {
			continue
		}
		full := filepath.Join(*root, filepath.FromSlash(p.Path))
		b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
		if err == nil {
			b, err = content.SetTopics(b, []string{topic})
		}
		if err == nil {
			err = os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "%s: %v\n", p.Path, err)
			failed++
		}
	}
	fmt.Fprintf(env.Stderr, "om topics: %d problems get a topic, %d failed\n", set, failed)
	if failed > 0 {
		return ExitFailure
	}
	return ExitOK
}
