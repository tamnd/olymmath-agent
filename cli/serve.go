package cli

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/serve"
)

func init() {
	register(command{"serve", "answer the read methods of the API over HTTP, or run the MCP server on stdin and stdout with -mcp", runServe})
}

func runServe(env Env, args []string) int {
	fs := newFlags("serve", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	addr := fs.String("addr", "127.0.0.1:8080", "the address to listen on")
	url := fs.String("url", "https://olymmath.tamnd.com", "the public URL of the site, which the records link to")
	mcp := fs.Bool("mcp", false, "run the MCP server on stdin and stdout, and do not listen")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om serve: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	s := serve.New(corpus, *url, env.Version)
	if *mcp {
		if err := s.ServeMCP(os.Stdin, env.Stdout); err != nil {
			fmt.Fprintf(env.Stderr, "om serve: %v\n", err)
			return ExitFailure
		}
		return ExitOK
	}
	fmt.Fprintf(env.Stderr, "om serve: %d problems at http://%s/api/ and the MCP server at http://%s/mcp\n", len(corpus.All()), *addr, *addr)
	srv := &http.Server{Addr: *addr, Handler: s.Handler(), ReadHeaderTimeout: 10 * time.Second}
	if err := srv.ListenAndServe(); err != nil {
		fmt.Fprintf(env.Stderr, "om serve: %v\n", err)
		return ExitFailure
	}
	return ExitOK
}
