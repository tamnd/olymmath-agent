package vision

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// HTTP posts to a server at a base URL, such as http://127.0.0.1:8079/v1.
type HTTP struct {
	BaseURL string
	// Client is the HTTP client, or http.DefaultClient when it is nil. The time limit of a call is on the context.
	Client *http.Client
}

// Name gives the host and port of the base URL.
func (h *HTTP) Name() string {
	if u, err := url.Parse(h.BaseURL); err == nil && u.Host != "" {
		return u.Host
	}
	return h.BaseURL
}

// Do implements Transport.
func (h *HTTP) Do(ctx context.Context, method, path string, body []byte) (Reply, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(h.BaseURL, "/")+"/"+path, rd)
	if err != nil {
		return Reply{}, fmt.Errorf("vision: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	c := h.Client
	if c == nil {
		c = http.DefaultClient
	}
	res, err := c.Do(req) // #nosec G704 -- the base URL is a server that the user names.
	if err != nil {
		if ctx.Err() != nil {
			return Reply{}, fmt.Errorf("%s: %w", h.Name(), ctx.Err())
		}
		return Reply{}, fmt.Errorf("%s: %w: %s", h.Name(), ErrUnreachable, oneLine(err.Error(), 200))
	}
	defer func() { _ = res.Body.Close() }()
	b, err := io.ReadAll(io.LimitReader(res.Body, 64<<20))
	if err != nil {
		return Reply{}, fmt.Errorf("%s: %w: %s", h.Name(), ErrUnreachable, oneLine(err.Error(), 200))
	}
	return Reply{Status: res.StatusCode, Body: b, RetryAfter: retryAfter(res.Header.Get("Retry-After"))}, nil
}

// Runner runs a command on a host through ssh, with stdin on its standard input. Tests replace it.
type Runner func(ctx context.Context, host string, argv []string, stdin []byte) (stdout, stderr []byte, err error)

// SSH posts to a server that listens on the loopback address of a host that ssh reaches. It runs curl on the host and sends the body on the standard input of curl. So the server can bind 127.0.0.1, and no port forward must be started and kept up. The ssh configuration of this machine gives the user, the address and the key of the host.
type SSH struct {
	// Host is an ssh destination, such as gpc.
	Host string
	// URL is the base URL as the host sees it, such as http://127.0.0.1:8801/v1.
	URL string
	// Run runs the command. It is RunSSH when it is nil.
	Run Runner
}

// Name gives the host.
func (s *SSH) Name() string { return s.Host }

// statusMark comes before the status that curl writes after the body, so the status is not a part of the body.
const statusMark = "\n--status "

// Do implements Transport. curl writes the body and then the status code, so the transport knows a 429 from a 400. It cannot see the Retry-After header.
func (s *SSH) Do(ctx context.Context, method, path string, body []byte) (Reply, error) {
	seconds := 600
	if dl, ok := ctx.Deadline(); ok {
		seconds = max(1, int(time.Until(dl)/time.Second))
	}
	argv := []string{"curl", "--silent", "--show-error", "--max-time", strconv.Itoa(seconds), "-X", method, "--write-out", statusMark + "%{http_code}"}
	if body != nil {
		argv = append(argv, "-H", "Content-Type: application/json", "--data-binary", "@-")
	}
	argv = append(argv, strings.TrimSuffix(s.URL, "/")+"/"+path)
	run := s.Run
	if run == nil {
		run = RunSSH
	}
	out, errb, err := run(ctx, s.Host, argv, body)
	if err != nil {
		if ctx.Err() != nil {
			return Reply{}, fmt.Errorf("%s: %w", s.Host, ctx.Err())
		}
		// ssh exits with 255 when it cannot reach the host, and curl exits with its own code when nothing listens on the port. Both are a server that does not answer.
		msg := oneLine(string(errb), 200)
		if msg == "" {
			msg = err.Error()
		}
		return Reply{}, fmt.Errorf("%s: %w: %s", s.Host, ErrUnreachable, msg)
	}
	i := bytes.LastIndex(out, []byte(statusMark))
	if i < 0 {
		return Reply{}, fmt.Errorf("%s: %w: curl gave no status", s.Host, ErrUnreachable)
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(out[i+len(statusMark):])))
	if err != nil || code == 0 {
		return Reply{}, fmt.Errorf("%s: %w: curl gave no status", s.Host, ErrUnreachable)
	}
	return Reply{Status: code, Body: out[:i]}, nil
}

// RunSSH runs the command with the ssh client of this machine. BatchMode is on, because a prompt for a passphrase in a batch waits until a person sees it. ssh gives the command to a shell on the host, so each argument is quoted.
func RunSSH(ctx context.Context, host string, argv []string, stdin []byte) ([]byte, []byte, error) {
	if strings.HasPrefix(host, "-") {
		return nil, nil, errors.New("vision: a host cannot start with -")
	}
	args := []string{"-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=30", host, Quote(argv)}
	cmd := exec.CommandContext(ctx, "ssh", args...) // #nosec G204 -- the host is an ssh destination that the user names, and the command is quoted.
	cmd.Stdin = bytes.NewReader(stdin)
	var out, errb bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errb
	err := cmd.Run()
	return out.Bytes(), errb.Bytes(), err
}

// Quote gives the command line for a remote shell, with each argument in single quotes.
func Quote(argv []string) string {
	parts := make([]string, len(argv))
	for i, a := range argv {
		parts[i] = "'" + strings.ReplaceAll(a, "'", `'\''`) + "'"
	}
	return strings.Join(parts, " ")
}
