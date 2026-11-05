package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// server is a fake vision server. answer writes the reply to each request, and the server keeps the body of the last request.
type server struct {
	*httptest.Server
	got map[string]any
}

func newServer(t *testing.T, answer func(w http.ResponseWriter, body map[string]any)) *server {
	t.Helper()
	s := &server{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/chat/completions" || r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}
		b, _ := io.ReadAll(r.Body)
		s.got = map[string]any{}
		if err := json.Unmarshal(b, &s.got); err != nil {
			t.Errorf("the request is not JSON: %v", err)
		}
		answer(w, s.got)
	}))
	t.Cleanup(s.Close)
	return s
}

// reply writes a chat completion with the text and the finish reason.
func reply(w http.ResponseWriter, text, finish string) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"model":"reader","choices":[{"finish_reason":%q,"message":{"role":"assistant","content":%q}}],"usage":{"prompt_tokens":1200,"completion_tokens":40}}`, finish, text)
}

// fakeSSH is a Runner that does what ssh and curl do: it posts stdin to the URL of the curl command, which is the last argument, on the test server, and it writes the body and the status mark. down makes it fail as ssh fails when the host does not answer.
func fakeSSH(t *testing.T, down bool, hosts *[]string) Runner {
	return func(ctx context.Context, host string, argv []string, stdin []byte) ([]byte, []byte, error) {
		*hosts = append(*hosts, host)
		if down {
			return nil, []byte("ssh: connect to host gpu port 22: Operation timed out\n"), errors.New("exit status 255")
		}
		if argv[0] != "curl" || !strings.Contains(Quote(argv), "'--data-binary' '@-'") {
			t.Errorf("command %q", Quote(argv))
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, argv[len(argv)-1], bytes.NewReader(stdin))
		if err != nil {
			return nil, nil, err
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, []byte("curl: (7) Failed to connect\n"), errors.New("exit status 7")
		}
		defer func() { _ = res.Body.Close() }()
		b, _ := io.ReadAll(res.Body)
		return append(b, []byte(fmt.Sprintf("%s%d", statusMark, res.StatusCode))...), nil, nil
	}
}

// transports gives the two transports to the server: HTTP, and SSH through the fake runner.
func transports(t *testing.T, s *server) map[string]Transport {
	var hosts []string
	return map[string]Transport{
		"http": &HTTP{BaseURL: s.URL + "/v1"},
		"ssh":  &SSH{Host: "gpu", URL: s.URL + "/v1", Run: fakeSSH(t, false, &hosts)},
	}
}

var page = Request{Model: "reader", Prompt: "Write the text of the page.", Image: []byte{0x89, 'P', 'N', 'G'}, MediaType: "image/png"}

func TestReadGoodPage(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, _ map[string]any) { reply(w, "**1.** Prove that $a^2 \\ge 0$.", "stop") })
	for name, tr := range transports(t, s) {
		t.Run(name, func(t *testing.T) {
			res, err := Read(t.Context(), tr, page)
			if err != nil {
				t.Fatal(err)
			}
			if res.Text != "**1.** Prove that $a^2 \\ge 0$." || res.Model != "reader" || res.Finish != "stop" || res.Input != 1200 || res.Output != 40 {
				t.Errorf("%+v", res)
			}
			if s.got["temperature"] != 0.0 {
				t.Errorf("temperature %v, want 0", s.got["temperature"])
			}
			if _, ok := s.got["max_tokens"]; ok {
				t.Error("the request has max_tokens. With no limit the server gives the answer the rest of the context")
			}
			msgs := s.got["messages"].([]any)
			parts := msgs[0].(map[string]any)["content"].([]any)
			img := parts[0].(map[string]any)
			text := parts[1].(map[string]any)
			if img["type"] != "image_url" || text["text"] != page.Prompt {
				t.Errorf("parts %v", parts)
			}
			if url := img["image_url"].(map[string]any)["url"].(string); url != "data:image/png;base64,iVBORw==" {
				t.Errorf("data URL %q", url)
			}
		})
	}
}

func TestReadRefusals(t *testing.T) {
	for _, tc := range []struct{ name, text, finish, why string }{
		{"refusal phrase", "I'm sorry, I can't transcribe this image.", "stop", "I'm sorry"},
		{"token limit", "**1.** 2.5.1.1. 2.5.1.1. 2.5.1.1.", "length", "token limit"},
		{"empty message", "  ", "stop", "empty message"},
	} {
		s := newServer(t, func(w http.ResponseWriter, _ map[string]any) { reply(w, tc.text, tc.finish) })
		for name, tr := range transports(t, s) {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				_, err := Read(t.Context(), tr, page)
				if !errors.Is(err, ErrRefused) || !strings.Contains(err.Error(), tc.why) {
					t.Errorf("error %v, want ErrRefused with %q", err, tc.why)
				}
			})
		}
	}
}

func TestReadServerErrors(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		want   string
		busy   bool
		down   bool
	}{
		{"400 with an OpenAI error", 400, `{"error":{"message":"input length 18303 exceeds the maximum context length 16384","type":"BadRequestError"}}`, "HTTP 400: input length 18303 exceeds the maximum context length 16384", false, false},
		{"400 with a FastAPI detail", 400, `{"detail":"this lane reads exactly one image per call"}`, "HTTP 400: this lane reads exactly one image per call", false, false},
		{"429", 429, `{"error":{"message":"too many requests"}}`, "HTTP 429: too many requests", true, false},
		{"502 with a page", 502, `<html>Bad Gateway</html>`, "HTTP 502: <html>Bad Gateway</html>", false, true},
	} {
		s := newServer(t, func(w http.ResponseWriter, _ map[string]any) {
			if tc.status == 429 {
				w.Header().Set("Retry-After", "7")
			}
			w.WriteHeader(tc.status)
			_, _ = w.Write([]byte(tc.body))
		})
		for name, tr := range transports(t, s) {
			t.Run(tc.name+"/"+name, func(t *testing.T) {
				_, err := Read(t.Context(), tr, page)
				var ve *Error
				if !errors.As(err, &ve) {
					t.Fatalf("error %v, want a vision.Error", err)
				}
				if ve.Error() != tc.want || ve.Busy() != tc.busy || ve.Down() != tc.down {
					t.Errorf("error %q busy %v down %v", ve.Error(), ve.Busy(), ve.Down())
				}
				// curl does not give the headers to the ssh transport, so only HTTP sees Retry-After.
				if wait := map[string]time.Duration{"http": 7 * time.Second}[name]; tc.busy && ve.RetryAfter != wait {
					t.Errorf("retry after %v, want %v", ve.RetryAfter, wait)
				}
			})
		}
	}
}

func TestReadDownTarget(t *testing.T) {
	s := newServer(t, func(w http.ResponseWriter, _ map[string]any) { reply(w, "text", "stop") })
	url := s.URL + "/v1"
	s.Close()
	var hosts []string
	for name, tr := range map[string]Transport{
		"http":           &HTTP{BaseURL: url},
		"ssh host down":  &SSH{Host: "gpu", URL: url, Run: fakeSSH(t, true, &hosts)},
		"ssh no service": &SSH{Host: "gpu", URL: url, Run: fakeSSH(t, false, &hosts)},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Read(t.Context(), tr, page)
			if !errors.Is(err, ErrUnreachable) {
				t.Errorf("error %v, want ErrUnreachable", err)
			}
		})
	}
	if len(hosts) != 2 || hosts[0] != "gpu" {
		t.Errorf("hosts %v", hosts)
	}
}

func TestHealthAsksForOneToken(t *testing.T) {
	stuck := false
	s := newServer(t, func(w http.ResponseWriter, body map[string]any) {
		if body["max_tokens"] != 1.0 || body["model"] != "reader" {
			t.Errorf("health request %v", body)
		}
		if stuck {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		// One token reaches the limit. The health check does not take that as a refusal.
		reply(w, "Hello", "length")
	})
	for name, tr := range transports(t, s) {
		t.Run(name, func(t *testing.T) {
			stuck = false
			if err := Health(t.Context(), tr, "reader"); err != nil {
				t.Errorf("health: %v", err)
			}
			stuck = true
			var ve *Error
			if err := Health(t.Context(), tr, "reader"); !errors.As(err, &ve) || !ve.Down() {
				t.Errorf("health of a stuck engine: %v", err)
			}
		})
	}
}

func TestQuote(t *testing.T) {
	got := Quote([]string{"curl", "-H", "Content-Type: application/json", "it's"})
	want := `'curl' '-H' 'Content-Type: application/json' 'it'\''s'`
	if got != want {
		t.Errorf("%s", got)
	}
}
