package hf

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/store"
)

func sum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}

const small = `{"problem":"a"}` + "\n"

var big = strings.Repeat("x", 4096)

func hub(t *testing.T) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.EscapedPath() {
		case "/robots.txt":
			w.WriteHeader(http.StatusNotFound)
		case "/api/datasets/old/name":
			http.Redirect(w, r, "/api/datasets/org/set", http.StatusTemporaryRedirect)
		case "/api/datasets/org/set":
			io.WriteString(w, `{"id":"org/set","sha":"abc123","lastModified":"2026-01-02T03:04:05.000Z","cardData":{"license":"mit"}}`)
		case "/api/datasets/org/set/tree/abc123":
			if r.URL.Query().Get("cursor") == "" {
				w.Header().Set("Link", fmt.Sprintf(`<%s/api/datasets/org/set/tree/abc123?recursive=true&cursor=2>; rel="next"`, srv.URL))
				io.WriteString(w, `[{"type":"file","path":"README.md","size":10},{"type":"directory","path":"data"},{"type":"file","path":"data/a.jsonl","size":16}]`)
				return
			}
			fmt.Fprintf(w, `[{"type":"file","path":"data/b.parquet","size":4096,"lfs":{"oid":%q,"size":4096}},{"type":"file","path":"data/c.parquet","size":4096}]`, sum(big))
		case "/datasets/org/set/resolve/abc123/data/a.jsonl":
			io.WriteString(w, small)
		case "/datasets/org/set/resolve/abc123/data/b.parquet":
			io.WriteString(w, big)
		case "/api/datasets/org/set/revision/refs%2Fconvert%2Fparquet":
			io.WriteString(w, `{"id":"org/set","sha":"pq1"}`)
		case "/api/datasets/org/set/tree/pq1":
			io.WriteString(w, `[{"type":"file","path":"default/train/0000.parquet","size":16}]`)
		case "/datasets/org/set/resolve/pq1/default/train/0000.parquet":
			io.WriteString(w, small)
		case "/api/datasets/gated/set":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			t.Errorf("unexpected request %s", r.URL)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return srv
}

func TestHarvest(t *testing.T) {
	srv := hub(t)
	defer srv.Close()
	old := Base
	Base = srv.URL
	defer func() { Base = old }()
	st, _ := store.Open(t.TempDir(), nil)
	fc := fetch.New(st)
	fc.Delay = 0
	h := &Harvest{Hub: &Client{}, Fetch: fc, LocalMax: 1024}
	m, c, err := h.Run(context.Background(), "hf-set", "old/name")
	if err != nil {
		t.Fatal(err)
	}
	if m.Dataset != "org/set" || m.Revision != "abc123" || m.Licence != "mit" || m.LastModified.Year() != 2026 {
		t.Fatalf("manifest %+v", m)
	}
	if len(m.Files) != 3 {
		t.Fatalf("files %+v", m.Files)
	}
	if f := m.Files[0]; f.Where != "local" || f.SHA256 != sum(small) {
		t.Errorf("small file %+v", f)
	}
	// The large file has no remote, so it stays a reference with the hash from LFS. The other large file has no LFS hash, so it is a reference with no hash.
	if f := m.Files[1]; f.Where != "ref" || f.SHA256 != sum(big) {
		t.Errorf("large file %+v", f)
	}
	if f := m.Files[2]; f.Where != "ref" || f.SHA256 != "" {
		t.Errorf("large file with no hash %+v", f)
	}
	got, err := LatestManifest(context.Background(), st, "hf-set")
	if err != nil || got.Revision != "abc123" || c.Note != "manifest" {
		t.Fatalf("latest manifest %+v %v", got, err)
	}
	if _, err := (&Client{}).Info(context.Background(), "gated/set"); err == nil || !strings.Contains(err.Error(), "gated") {
		t.Errorf("gated: %v", err)
	}
}

func TestHarvestParquetForm(t *testing.T) {
	srv := hub(t)
	defer srv.Close()
	old := Base
	Base = srv.URL
	defer func() { Base = old }()
	st, _ := store.Open(t.TempDir(), nil)
	fc := fetch.New(st)
	fc.Delay = 0
	m, _, err := (&Harvest{Hub: &Client{}, Fetch: fc, MaxFiles: 2}).Run(context.Background(), "hf-set", "org/set")
	if err != nil {
		t.Fatal(err)
	}
	if !m.Converted || m.ParquetRevision != "pq1" || m.Revision != "abc123" || len(m.Files) != 1 || m.Files[0].Where != "local" {
		t.Fatalf("manifest %+v", m)
	}
}

func TestDatasetID(t *testing.T) {
	for u, want := range map[string]string{
		"https://huggingface.co/datasets/ShadenA/MathNet":          "ShadenA/MathNet",
		"https://huggingface.co/datasets/a/b/tree/main":            "a/b",
		"https://huggingface.co/MathArena":                         "",
		"https://huggingface.co/datasets/onlyowner":                "",
		"https://huggingface.co/datasets/Pandores/aime-1983-2026/": "Pandores/aime-1983-2026",
	} {
		got, _ := DatasetID(u)
		if got != want {
			t.Errorf("DatasetID(%q) = %q, want %q", u, got, want)
		}
	}
}

func TestLicence(t *testing.T) {
	i := &Info{}
	i.CardData.License = []byte(`["mit","cc-by-4.0"]`)
	if i.Licence() != "mit, cc-by-4.0" {
		t.Error(i.Licence())
	}
	i = &Info{Tags: []string{"task:x", "license:apache-2.0"}}
	if i.Licence() != "apache-2.0" {
		t.Error(i.Licence())
	}
}

func TestDataFiles(t *testing.T) {
	fs := DataFiles([]File{{Type: "file", Path: "README.md"}, {Type: "file", Path: "x.py"}, {Type: "file", Path: "d/train.parquet"}, {Type: "file", Path: "d/x.jsonl.gz"}, {Type: "file", Path: "dataset_infos.json"}, {Type: "directory", Path: "d"}})
	if len(fs) != 2 {
		t.Errorf("got %+v", fs)
	}
}

type putCounter struct{ puts int }

func (p *putCounter) Put(_ context.Context, _ string, r io.Reader, _ int64) error {
	p.puts++
	_, err := io.Copy(io.Discard, r)
	return err
}
func (p *putCounter) Get(context.Context, string) (io.ReadCloser, error) {
	return nil, store.ErrNotFound
}
func (p *putCounter) Delete(context.Context, string) error { return nil }

func TestHarvestRemoteMax(t *testing.T) {
	srv := hub(t)
	defer srv.Close()
	old := Base
	Base = srv.URL
	defer func() { Base = old }()
	rem := &putCounter{}
	st, _ := store.Open(t.TempDir(), rem)
	fc := fetch.New(st)
	fc.Delay = 0
	// The large file is bigger than the remote limit, so it stays a reference and nothing goes to the remote.
	h := &Harvest{Hub: &Client{}, Fetch: fc, LocalMax: 1024, RemoteMax: 1024}
	m, _, err := h.Run(context.Background(), "hf-set", "org/set")
	if err != nil {
		t.Fatal(err)
	}
	if f := m.Files[1]; f.Where != "ref" || f.SHA256 != sum(big) || rem.puts != 0 {
		t.Errorf("large file %+v, %d puts", f, rem.puts)
	}
}
