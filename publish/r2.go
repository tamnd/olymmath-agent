package publish

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// ManifestKey is the key of the object that lists the hash of each object that Sync uploaded.
const ManifestKey = "_manifest.json"

// R2 is a client of the R2 objects of one bucket over the Cloudflare REST API. It needs an API token with R2 write access, and no S3 keys.
type R2 struct {
	Account, Bucket, Token string
	// Base is the URL of the API. It is the Cloudflare API when it is empty.
	Base string
	HTTP *http.Client
}

func (r *R2) url(key string) string {
	base := r.Base
	if base == "" {
		base = "https://api.cloudflare.com/client/v4"
	}
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return base + "/accounts/" + url.PathEscape(r.Account) + "/r2/buckets/" + url.PathEscape(r.Bucket) + "/objects/" + strings.Join(parts, "/")
}

// do sends a request, and tries again after a pause when the API answers 429 or a 5xx code.
func (r *R2) do(ctx context.Context, method, key string, body []byte, ctype string) (*http.Response, error) {
	hc := r.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	wait := time.Second
	for try := 0; ; try++ {
		req, err := http.NewRequestWithContext(ctx, method, r.url(key), bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+r.Token)
		if ctype != "" {
			req.Header.Set("Content-Type", ctype)
		}
		resp, err := hc.Do(req)
		if err == nil && resp.StatusCode != http.StatusTooManyRequests && resp.StatusCode < 500 {
			return resp, nil
		}
		if try == 5 {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}
		if err == nil {
			_ = resp.Body.Close()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(wait):
		}
		wait *= 2
	}
}

// Put writes an object.
func (r *R2) Put(ctx context.Context, key string, body []byte, ctype string) error {
	resp, err := r.do(ctx, http.MethodPut, key, body, ctype)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("r2: put %s: %s", key, resp.Status)
	}
	return nil
}

// Get reads an object. It gives nil and no error when the object does not exist.
func (r *R2) Get(ctx context.Context, key string) ([]byte, error) {
	resp, err := r.do(ctx, http.MethodGet, key, nil, "")
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("r2: get %s: %s", key, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

// Delete removes an object.
func (r *R2) Delete(ctx context.Context, key string) error {
	resp, err := r.do(ctx, http.MethodDelete, key, nil, "")
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("r2: delete %s: %s", key, resp.Status)
	}
	return nil
}

// ContentType gives the media type of a file by its extension.
func ContentType(name string) string {
	switch path.Ext(name) {
	case ".html":
		return "text/html; charset=utf-8"
	case ".md":
		return "text/markdown; charset=utf-8"
	case ".json":
		return "application/json"
	}
	if t := mime.TypeByExtension(path.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// SyncResult tells what Sync did.
type SyncResult struct {
	Files, Put, Deleted int
}

// Sync makes the bucket hold the files of dir. It compares the hash of each file with the manifest of the last sync, uploads only the new and changed files, deletes the objects of the files that are gone, and then writes the new manifest. workers requests run at the same time.
func Sync(ctx context.Context, r *R2, dir string, workers int) (SyncResult, error) {
	var res SyncResult
	files := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p) // #nosec G304 G122 -- the path is inside the directory.
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		files[Key(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		return res, err
	}
	res.Files = len(files)
	old := map[string]string{}
	if b, err := r.Get(ctx, ManifestKey); err != nil {
		return res, err
	} else if b != nil {
		if err := json.Unmarshal(b, &old); err != nil {
			return res, fmt.Errorf("r2: %s: %w", ManifestKey, err)
		}
	}
	var put, del []string
	for k, h := range files {
		if old[k] != h {
			put = append(put, k)
		}
	}
	for k := range old {
		if _, ok := files[k]; !ok {
			del = append(del, k)
		}
	}
	sort.Strings(put)
	sort.Strings(del)
	type job struct {
		key    string
		delete bool
	}
	jobs := make(chan job)
	var mu sync.Mutex
	var first error
	var wg sync.WaitGroup
	for range max(workers, 1) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				var err error
				if j.delete {
					err = r.Delete(ctx, j.key)
				} else {
					var b []byte
					b, err = os.ReadFile(filepath.Join(dir, filepath.FromSlash(j.key))) // #nosec G304 -- the key comes from the walk of the directory.
					if err == nil {
						err = r.Put(ctx, j.key, b, ContentType(j.key))
					}
				}
				mu.Lock()
				switch {
				case err != nil && first == nil:
					first = err
				case err == nil && j.delete:
					res.Deleted++
				case err == nil:
					res.Put++
				}
				mu.Unlock()
			}
		}()
	}
	for _, k := range put {
		jobs <- job{key: k}
	}
	for _, k := range del {
		jobs <- job{key: k, delete: true}
	}
	close(jobs)
	wg.Wait()
	if first != nil {
		return res, first
	}
	b, err := json.Marshal(files)
	if err != nil {
		return res, err
	}
	return res, r.Put(ctx, ManifestKey, b, "application/json")
}
