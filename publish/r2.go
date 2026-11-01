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
	"maps"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strconv"
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
	// Rate is the most requests each second. The Cloudflare API takes 1200 requests in 5 minutes for each user, so the rate is 3.5 when it is 0. A negative rate has no limit.
	Rate float64

	once sync.Once
	tick <-chan time.Time
}

// pace waits for the next request slot.
func (r *R2) pace(ctx context.Context) error {
	r.once.Do(func() {
		rate := r.Rate
		if rate == 0 {
			rate = 3.5
		}
		if rate > 0 {
			r.tick = time.NewTicker(time.Duration(float64(time.Second) / rate)).C
		}
	})
	if r.tick == nil {
		return nil
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.tick:
		return nil
	}
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

// do sends a request, and tries again after a pause when the API answers 429 or a 5xx code. The pause doubles each time up to a minute, and it is longer when the API asks for it with Retry-After.
func (r *R2) do(ctx context.Context, method, key string, body []byte, ctype string) (*http.Response, error) {
	hc := r.HTTP
	if hc == nil {
		hc = http.DefaultClient
	}
	wait := time.Second
	for try := 0; ; try++ {
		if err := r.pace(ctx); err != nil {
			return nil, err
		}
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
		if try == 8 {
			if err != nil {
				return nil, err
			}
			return resp, nil
		}
		pause := wait
		if err == nil {
			if n, e := strconv.Atoi(resp.Header.Get("Retry-After")); e == nil && time.Duration(n)*time.Second > pause {
				pause = time.Duration(n) * time.Second
			}
			_ = resp.Body.Close()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pause):
		}
		wait = min(2*wait, time.Minute)
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
	// Left is the number of changes that SyncUntil did not send because its time was over.
	Left int
}

// SavePoint is the number of changes after which Sync writes the manifest.
const SavePoint = 500

// Sync makes the bucket hold the files of dir. It compares the hash of each file with the manifest of the last sync, uploads only the new and changed files, deletes the objects of the files that are gone, and then writes the new manifest. workers requests run at the same time. Sync also writes the manifest after each SavePoint changes and when a request fails, so a new run does not send again the files that are already in the bucket.
func Sync(ctx context.Context, r *R2, dir string, workers int) (SyncResult, error) {
	return SyncUntil(ctx, r, dir, workers, time.Time{})
}

// SyncUntil is Sync, but it sends no new change after the time until, when until is not zero. It then writes the manifest of the objects that the bucket holds, and the result tells how many changes are left. A new run sends them.
func SyncUntil(ctx context.Context, r *R2, dir string, workers int, until time.Time) (SyncResult, error) {
	return SyncWith(ctx, r, dir, SyncOptions{Workers: workers, Until: until})
}

// SyncOptions are the options of SyncWith.
type SyncOptions struct {
	// Workers is the number of requests at the same time.
	Workers int
	// Until is the time after which no new change is sent. The zero time has no limit.
	Until time.Time
	// Keep keeps the objects of the files that are gone. A deploy uploads the new packs with Keep before the new Worker is live, because the old Worker still reads the old packs, and it deletes them after.
	Keep bool
}

// SyncWith is Sync with options.
func SyncWith(ctx context.Context, r *R2, dir string, opt SyncOptions) (SyncResult, error) {
	workers, until := opt.Workers, opt.Until
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
		if _, ok := files[k]; !ok && !opt.Keep {
			del = append(del, k)
		}
	}
	sort.Strings(put)
	sort.Strings(del)
	type job struct {
		key    string
		delete bool
	}
	// done is the manifest of the objects that the bucket holds now.
	done := maps.Clone(old)
	if done == nil {
		done = map[string]string{}
	}
	save := func() error {
		b, err := json.Marshal(done)
		if err != nil {
			return err
		}
		return r.Put(ctx, ManifestKey, b, "application/json")
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
					delete(done, j.key)
				case err == nil:
					res.Put++
					done[j.key] = files[j.key]
				}
				var snap []byte
				if err == nil && (res.Put+res.Deleted)%SavePoint == 0 {
					snap, _ = json.Marshal(done)
				}
				mu.Unlock()
				if snap != nil {
					_ = r.Put(ctx, ManifestKey, snap, "application/json")
				}
			}
		}()
	}
	all := make([]job, 0, len(put)+len(del))
	for _, k := range put {
		all = append(all, job{key: k})
	}
	for _, k := range del {
		all = append(all, job{key: k, delete: true})
	}
	for i, j := range all {
		if !until.IsZero() && time.Now().After(until) {
			res.Left = len(all) - i
			break
		}
		jobs <- j
	}
	close(jobs)
	wg.Wait()
	if first != nil {
		_ = save()
		return res, first
	}
	if res.Left > 0 || opt.Keep {
		return res, save()
	}
	done = files
	return res, save()
}
