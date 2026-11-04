package lane

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
)

// Kinds are the command-line clients that a CLI lane can run.
var Kinds = []string{"codex", "gemini", "claude"}

// DefaultTimeout is the longest time that one call can take.
const DefaultTimeout = 5 * time.Minute

// CLI is a lane that runs a local command-line client of a model, with the login of that client. It sends no API key.
type CLI struct {
	// Kind is one of Kinds. It gives the flags of the command line and the form of the output.
	Kind string
	// Path is the path of the program.
	Path string
	// Model is the model to ask, or "" for the default of the client.
	Model   string
	Timeout time.Duration
	// run runs a command line in a directory and gives what the program wrote to its standard output and its standard error. Tests replace it.
	run func(ctx context.Context, dir string, argv []string) (stdout, stderr []byte, err error)

	versionOnce sync.Once
	version     string
}

// versionRE finds a version number in the output of --version, as in "2.1.318 (Claude Code)" or "codex-cli 0.166.0".
var versionRE = regexp.MustCompile(`[0-9]+\.[0-9]+(?:\.[0-9]+)?`)

// Version gives the version of the client, or "" when the client does not tell it. It runs the client once.
func (l *CLI) Version(ctx context.Context) string {
	l.versionOnce.Do(func() {
		if l.Path == "" {
			return
		}
		ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
		out, errb, err := l.run(ctx, "", []string{l.Path, "--version"})
		if err == nil {
			l.version = versionRE.FindString(string(out) + " " + string(errb))
		}
	})
	return l.version
}

// NewCLI gives the lane of a client.
func NewCLI(kind, path string) (*CLI, error) {
	if !slices.Contains(Kinds, kind) {
		return nil, fmt.Errorf("lane: unknown client %q, not one of %s", kind, strings.Join(Kinds, ", "))
	}
	return &CLI{Kind: kind, Path: path, Timeout: DefaultTimeout, run: runCommand}, nil
}

// Name gives the kind of the client, such as codex.
func (l *CLI) Name() string { return l.Kind }

// runCommand runs the program with its standard input closed. A client that waits on its standard input waits until the timeout and gives no text, which looks like a network failure.
func runCommand(ctx context.Context, dir string, argv []string) ([]byte, []byte, error) {
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...) // #nosec G204 -- the program is a model client that the user names, and the prompt is one argument.
	cmd.Dir = dir
	cmd.Stdin = nil
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	return out.Bytes(), errb.Bytes(), err
}

// Argv gives the command line for a question, and the file where codex writes its last message. dir is the directory of the call. The prompt is the last argument.
func (l *CLI) Argv(q Question, dir string) (argv []string, outFile string) {
	argv = []string{l.Path}
	prompt := q.Prompt
	switch l.Kind {
	case "codex":
		outFile = filepath.Join(dir, "answer.txt")
		argv = append(argv, "exec", "--skip-git-repo-check", "--ephemeral", "-s", "read-only", "-C", dir, "-o", outFile)
		if l.Model != "" {
			argv = append(argv, "-m", l.Model)
		}
		for _, img := range q.Images {
			argv = append(argv, "-i", img)
		}
		argv = append(argv, "--")
	case "gemini":
		argv = append(argv, "--skip-trust", "--approval-mode", "plan", "-o", "json")
		if l.Model != "" {
			argv = append(argv, "-m", l.Model)
		}
		for _, img := range q.Images {
			argv = append(argv, "--include-directories", filepath.Dir(img))
			prompt += "\n\nThe image: @" + img
		}
		argv = append(argv, "-p")
	case "claude":
		argv = append(argv, "-p", "--output-format", "json", "--strict-mcp-config", "--allowedTools", "Read")
		if l.Model != "" {
			argv = append(argv, "--model", l.Model)
		}
		var names []string
		for _, img := range q.Images {
			argv = append(argv, "--add-dir", filepath.Dir(img))
			names = append(names, img)
		}
		if len(names) > 0 {
			prompt = "Read the image file " + strings.Join(names, " and the image file ") + " with the Read tool. Then do this task.\n\n" + prompt
		}
		// --add-dir and --allowedTools take more than one value, so without -- they take the prompt too.
		argv = append(argv, "--")
	}
	return append(argv, prompt), outFile
}

// unavailable are the phrases with which a client says that it has no login on this machine.
var unavailable = []string{
	"please set an auth method",
	"not logged in",
	"please run /login",
	"invalid api key",
	"run codex login",
}

// Ask sends one question to the client.
func (l *CLI) Ask(ctx context.Context, q Question) (Answer, error) {
	if l.Path == "" {
		return Answer{Lane: l.Kind}, fmt.Errorf("%s: %w: no program", l.Kind, ErrUnavailable)
	}
	dir, err := os.MkdirTemp("", "om-lane-")
	if err != nil {
		return Answer{Lane: l.Kind}, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	timeout := l.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	argv, outFile := l.Argv(q, dir)
	start := time.Now()
	stdout, stderr, runErr := l.run(ctx, dir, argv)
	a := Answer{Lane: l.Kind, Model: l.Model, Seconds: time.Since(start).Seconds()}
	said := string(stdout) + "\n" + string(stderr)
	if e := Quota(l.Kind, said); e != nil {
		return a, e
	}
	low := strings.ToLower(said)
	for _, p := range unavailable {
		if strings.Contains(low, p) {
			return a, fmt.Errorf("%s: %w: %s", l.Kind, ErrUnavailable, p)
		}
	}
	if runErr != nil {
		if ctx.Err() != nil {
			return a, fmt.Errorf("%s: %w", l.Kind, ctx.Err())
		}
		return a, fmt.Errorf("%s: %w: %s", l.Kind, runErr, oneLine(string(stderr)))
	}
	text, model, err := l.read(stdout, outFile)
	if err != nil {
		return a, fmt.Errorf("%s: %w", l.Kind, err)
	}
	if model != "" {
		a.Model = model
	}
	if strings.TrimSpace(text) == "" {
		return a, fmt.Errorf("%s: %w after %.0f s", l.Kind, ErrEmptyAnswer, a.Seconds)
	}
	a.Text = strings.TrimSpace(text)
	a.Version = l.Version(ctx)
	return a, nil
}

// read gives the text and the model of the answer from the output of the client.
func (l *CLI) read(stdout []byte, outFile string) (text, model string, err error) {
	switch l.Kind {
	case "codex":
		b, err := os.ReadFile(outFile) // #nosec G304 -- a file in the temporary directory of the call.
		if err != nil {
			return "", "", fmt.Errorf("no last message: %w", err)
		}
		return string(b), "", nil
	case "gemini":
		var r struct {
			Response string `json:"response"`
			Stats    struct {
				Models map[string]json.RawMessage `json:"models"`
			} `json:"stats"`
		}
		if err := json.Unmarshal(stdout, &r); err != nil {
			return string(stdout), "", nil
		}
		return r.Response, firstKey(r.Stats.Models), nil
	case "claude":
		var r struct {
			Result     string                     `json:"result"`
			IsError    bool                       `json:"is_error"`
			ModelUsage map[string]json.RawMessage `json:"modelUsage"`
		}
		if err := json.Unmarshal(stdout, &r); err != nil {
			return "", "", fmt.Errorf("the output is not JSON: %w", err)
		}
		if r.IsError {
			return "", "", fmt.Errorf("the client gave an error: %s", oneLine(r.Result))
		}
		return r.Result, firstKey(r.ModelUsage), nil
	}
	return string(stdout), "", nil
}

// firstKey gives the first key of a map in sorted order, or "".
func firstKey(m map[string]json.RawMessage) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}
