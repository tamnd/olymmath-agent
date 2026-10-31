// Package hf reads datasets from the Hugging Face hub into the raw store. A harvest pins the commit of the dataset, records the hash of each data file, and writes a manifest of the dataset into the store.
package hf

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/fetch"
)

// Base is the address of the hub.
var Base = "https://huggingface.co"

// ParquetRevision is the branch where the hub keeps the Parquet form of each dataset.
const ParquetRevision = "refs/convert/parquet"

// Info is the part of the dataset metadata that a harvest uses.
type Info struct {
	ID           string    `json:"id"`
	SHA          string    `json:"sha"`
	LastModified time.Time `json:"lastModified"`
	Gated        any       `json:"gated"`
	Private      bool      `json:"private"`
	CardData     struct {
		License json.RawMessage `json:"license"`
	} `json:"cardData"`
	Tags []string `json:"tags"`
}

// Licence returns the licence of the dataset card, or the licence tag, or "".
func (i *Info) Licence() string {
	var one string
	if json.Unmarshal(i.CardData.License, &one) == nil && one != "" {
		return one
	}
	var many []string
	if json.Unmarshal(i.CardData.License, &many) == nil && len(many) > 0 {
		return strings.Join(many, ", ")
	}
	for _, t := range i.Tags {
		if l, ok := strings.CutPrefix(t, "license:"); ok {
			return l
		}
	}
	return ""
}

// File is one file of a dataset at a revision.
type File struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Size int64  `json:"size"`
	OID  string `json:"oid"`
	LFS  *struct {
		OID  string `json:"oid"`
		Size int64  `json:"size"`
	} `json:"lfs"`
}

// SHA256 returns the SHA-256 hash of the file when the hub knows it. The hub knows it for each file in LFS.
func (f File) SHA256() string {
	if f.LFS != nil {
		return f.LFS.OID
	}
	return ""
}

// Client talks to the hub.
type Client struct {
	HTTP  *http.Client
	Agent string
	// Token is a Hugging Face token for gated datasets. It can be empty.
	Token string
}

func (c *Client) get(ctx context.Context, u string, out any) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return "", err
	}
	agent := c.Agent
	if agent == "" {
		agent = fetch.Agent
	}
	req.Header.Set("User-Agent", agent)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	hc := c.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	resp, err := hc.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return "", fmt.Errorf("%s: %s: %w", u, resp.Status, ErrGated)
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%s: %s", u, resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return "", fmt.Errorf("%s: %w", u, err)
	}
	return resp.Header.Get("Link"), nil
}

// ErrGated is the error for a dataset that needs a login or an agreement.
var ErrGated = errors.New("the dataset is gated or private")

// DatasetID returns the dataset of a hub URL, for example ShadenA/MathNet for https://huggingface.co/datasets/ShadenA/MathNet.
func DatasetID(u string) (string, bool) {
	rest, ok := strings.CutPrefix(u, "https://huggingface.co/datasets/")
	if !ok {
		return "", false
	}
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

// Info returns the metadata of a dataset. The hub sends a redirect for a dataset with a new name, and the ID of the result is the new name.
func (c *Client) Info(ctx context.Context, id string) (*Info, error) {
	var i Info
	if _, err := c.get(ctx, Base+"/api/datasets/"+id, &i); err != nil {
		return nil, err
	}
	if i.ID == "" {
		i.ID = id
	}
	return &i, nil
}

// Revision returns the commit of a branch of a dataset.
func (c *Client) Revision(ctx context.Context, id, branch string) (string, error) {
	var i Info
	if _, err := c.get(ctx, Base+"/api/datasets/"+id+"/revision/"+url.PathEscape(branch), &i); err != nil {
		return "", err
	}
	if i.SHA == "" {
		return "", fmt.Errorf("%s: the branch %s has no commit", id, branch)
	}
	return i.SHA, nil
}

var nextLink = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

// Files returns each file of the dataset at the revision.
func (c *Client) Files(ctx context.Context, id, rev string) ([]File, error) {
	u := fmt.Sprintf("%s/api/datasets/%s/tree/%s?recursive=true", Base, id, url.PathEscape(rev))
	var out []File
	for u != "" {
		var page []File
		link, err := c.get(ctx, u, &page)
		if err != nil {
			return nil, err
		}
		out = append(out, page...)
		u = ""
		if m := nextLink.FindStringSubmatch(link); m != nil {
			u = m[1]
		}
	}
	return out, nil
}

// dataExt are the file types that have rows.
var dataExt = []string{".parquet", ".jsonl", ".json", ".csv", ".tsv", ".arrow"}

// DataFiles returns the files that have rows. It does not return the card, the scripts or the images.
func DataFiles(files []File) []File {
	var out []File
	for _, f := range files {
		if f.Type != "file" {
			continue
		}
		name := strings.ToLower(f.Path)
		ext := path.Ext(strings.TrimSuffix(strings.TrimSuffix(name, ".gz"), ".zst"))
		if !slices.Contains(dataExt, ext) || strings.HasPrefix(path.Base(name), ".") || path.Base(name) == "dataset_infos.json" || path.Base(name) == "dataset_info.json" {
			continue
		}
		out = append(out, f)
	}
	return out
}

// ResolveURL is the address of the bytes of a file at a revision.
func ResolveURL(id, rev, file string) string {
	parts := strings.Split(file, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return fmt.Sprintf("%s/datasets/%s/resolve/%s/%s", Base, id, url.PathEscape(rev), strings.Join(parts, "/"))
}

// Manifest is the record of one harvest of a dataset. The store keeps it as JSON, and its hash identifies the harvest.
type Manifest struct {
	Source   string `json:"source"`
	Dataset  string `json:"dataset"`
	Revision string `json:"revision"`
	// Converted is true when the files come from the Parquet form of the hub, at the commit ParquetRevision.
	Converted       bool           `json:"converted,omitempty"`
	ParquetRevision string         `json:"parquet_revision,omitempty"`
	LastModified    time.Time      `json:"last_modified"`
	Licence         string         `json:"licence,omitempty"`
	Fetched         time.Time      `json:"fetched"`
	Files           []ManifestFile `json:"files"`
}

// ManifestFile is one data file of a manifest.
type ManifestFile struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	URL    string `json:"url"`
	// Where is local, r2 or ref. A file in ref is not in the store; the hash and the URL at the pinned revision identify its bytes.
	Where string `json:"where"`
}
