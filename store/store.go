// Package store is the raw store of olymmath. Each capture is kept under the SHA-256 hash of its bytes, so the same bytes are kept one time only. Each source has an index of its captures in JSON Lines. The bytes can be on the local disk, in the R2 bucket, or in both places.
package store

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Capture is one record of the index: where the bytes came from, when, and under which terms.
type Capture struct {
	Hash         string    `json:"hash"`
	Size         int64     `json:"size"`
	Source       string    `json:"source"`
	URL          string    `json:"url"`
	Fetched      time.Time `json:"fetched"`
	Type         string    `json:"type,omitempty"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"last_modified,omitempty"`
	Licence      string    `json:"licence,omitempty"`
	// Local is true when the bytes are on the local disk. Remote is true when the bytes are in the R2 bucket.
	Local  bool   `json:"local,omitempty"`
	Remote bool   `json:"remote,omitempty"`
	Note   string `json:"note,omitempty"`
}

// ErrNotFound is the error when the store does not have the bytes of a hash.
var ErrNotFound = errors.New("not in the store")

// Remote is a place that keeps the bytes away from the local disk, for example the R2 bucket.
type Remote interface {
	Put(ctx context.Context, key string, r io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

// Store is a raw store in the directory Dir. Remote can be nil.
type Store struct {
	Dir    string
	Remote Remote
	mu     sync.Mutex
}

// Open returns the store in dir and makes the directory if it does not exist.
func Open(dir string, remote Remote) (*Store, error) {
	for _, d := range []string{"blobs", "index"} {
		if err := os.MkdirAll(filepath.Join(dir, d), 0o750); err != nil {
			return nil, err
		}
	}
	return &Store{Dir: dir, Remote: remote}, nil
}

// DefaultDir is the directory of the store when the user does not give one.
func DefaultDir() string {
	if d := os.Getenv("OLYMMATH_RAW"); d != "" {
		return d
	}
	if d, err := os.UserCacheDir(); err == nil {
		return filepath.Join(d, "olymmath", "raw")
	}
	return filepath.Join(os.TempDir(), "olymmath", "raw")
}

// ValidHash tells if h is a SHA-256 hash in lower case hexadecimal.
func ValidHash(h string) bool {
	if len(h) != 64 {
		return false
	}
	_, err := hex.DecodeString(h)
	return err == nil && strings.ToLower(h) == h
}

// Key is the name of the bytes of a hash in the R2 bucket.
func Key(hash string) string { return "sha256/" + hash[:2] + "/" + hash }

func (s *Store) blob(hash string) string {
	return filepath.Join(s.Dir, "blobs", hash[:2], hash[2:4], hash)
}

// LocalPath returns the path of the blob of hash on the local disk, and false when the blob is not there.
func (s *Store) LocalPath(hash string) (string, bool) {
	p := s.blob(hash)
	_, err := os.Stat(p)
	return p, err == nil
}

// HasLocal tells if the bytes of hash are on the local disk.
func (s *Store) HasLocal(hash string) bool {
	_, err := os.Stat(s.blob(hash))
	return err == nil
}

// Put reads all of r into the local store and adds c to the index of its source. Put sets the hash, the size and the time of c when they are empty.
func (s *Store) Put(r io.Reader, c Capture) (Capture, error) {
	tmp, err := os.CreateTemp(filepath.Join(s.Dir, "blobs"), "put-*")
	if err != nil {
		return c, err
	}
	defer os.Remove(tmp.Name())
	h := sha256.New()
	n, err := io.Copy(io.MultiWriter(tmp, h), r)
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return c, err
	}
	hash := hex.EncodeToString(h.Sum(nil))
	if c.Hash != "" && c.Hash != hash {
		return c, fmt.Errorf("hash mismatch: want %s, got %s", c.Hash, hash)
	}
	c.Hash, c.Size, c.Local = hash, n, true
	if !s.HasLocal(hash) {
		dst := s.blob(hash)
		if err := os.MkdirAll(filepath.Dir(dst), 0o750); err != nil {
			return c, err
		}
		if err := os.Rename(tmp.Name(), dst); err != nil {
			return c, err
		}
	}
	return c, s.add(c)
}

// PutBytes is Put for bytes in memory.
func (s *Store) PutBytes(b []byte, c Capture) (Capture, error) {
	return s.Put(bytes.NewReader(b), c)
}

// PutRemote sends r to the remote store and does not keep the bytes on the local disk. The hash of c must be the expected hash of the bytes, because the key of the bytes comes from the hash. When the bytes have a different hash, PutRemote deletes them from the remote store and returns an error.
func (s *Store) PutRemote(ctx context.Context, r io.Reader, size int64, c Capture) (Capture, error) {
	if s.Remote == nil {
		return c, errors.New("the store has no remote")
	}
	if !ValidHash(c.Hash) {
		return c, fmt.Errorf("PutRemote needs the expected hash, got %q", c.Hash)
	}
	h := sha256.New()
	cr := &counter{r: io.TeeReader(r, h)}
	key := Key(c.Hash)
	if err := s.Remote.Put(ctx, key, cr, size); err != nil {
		return c, err
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != c.Hash {
		_ = s.Remote.Delete(ctx, key)
		return c, fmt.Errorf("hash mismatch: want %s, got %s", c.Hash, got)
	}
	c.Size, c.Remote = cr.n, true
	c.Local = s.HasLocal(c.Hash)
	return c, s.add(c)
}

type counter struct {
	r io.Reader
	n int64
}

func (c *counter) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}

// Get returns the bytes of hash, from the local disk when they are there, or else from the remote store.
func (s *Store) Get(ctx context.Context, hash string) (io.ReadCloser, error) {
	if !ValidHash(hash) {
		return nil, fmt.Errorf("not a hash: %q", hash)
	}
	f, err := os.Open(s.blob(hash))
	if err == nil {
		return f, nil
	}
	if s.Remote == nil {
		return nil, fmt.Errorf("%s: %w", hash, ErrNotFound)
	}
	return s.Remote.Get(ctx, Key(hash))
}

// Push sends the local bytes of hash to the remote store.
func (s *Store) Push(ctx context.Context, hash string) error {
	if s.Remote == nil {
		return errors.New("the store has no remote")
	}
	f, err := os.Open(s.blob(hash))
	if err != nil {
		return fmt.Errorf("%s: %w", hash, ErrNotFound)
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return err
	}
	return s.Remote.Put(ctx, Key(hash), f, st.Size())
}

// Evict removes the local bytes of hash. Use it only after Push.
func (s *Store) Evict(hash string) error {
	err := os.Remove(s.blob(hash))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func indexName(source string) string {
	r := strings.NewReplacer("/", "__", ":", "--", "\\", "_", "..", "_")
	return r.Replace(source) + ".jsonl"
}

func (s *Store) add(c Capture) error {
	if c.Source == "" {
		return errors.New("a capture needs a source")
	}
	if c.Fetched.IsZero() {
		c.Fetched = time.Now().UTC()
	}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(filepath.Join(s.Dir, "index", indexName(c.Source)), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(b, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Captures returns the index of a source, oldest first.
func (s *Store) Captures(source string) ([]Capture, error) {
	f, err := os.Open(filepath.Join(s.Dir, "index", indexName(source)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Capture
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var c Capture
		if err := json.Unmarshal(sc.Bytes(), &c); err != nil {
			return nil, fmt.Errorf("%s: %w", indexName(source), err)
		}
		out = append(out, c)
	}
	return out, sc.Err()
}

// Latest returns the newest capture of url in the index of source.
func (s *Store) Latest(source, url string) (Capture, bool, error) {
	cs, err := s.Captures(source)
	if err != nil {
		return Capture{}, false, err
	}
	for i := len(cs) - 1; i >= 0; i-- {
		if cs[i].URL == url {
			return cs[i], true, nil
		}
	}
	return Capture{}, false, nil
}

// Sources returns the source of each index of the store.
func (s *Store) Sources() ([]string, error) {
	es, err := os.ReadDir(filepath.Join(s.Dir, "index"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range es {
		if !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		f, err := os.Open(filepath.Join(s.Dir, "index", e.Name()))
		if err != nil {
			return nil, err
		}
		var c Capture
		err = json.NewDecoder(f).Decode(&c)
		f.Close()
		if err == nil && c.Source != "" {
			out = append(out, c.Source)
		}
	}
	return out, nil
}

// Cursor returns the saved position of a harvest of source, so that a harvest that stops can resume. It returns "" when there is no cursor.
func (s *Store) Cursor(source string) (string, error) {
	b, err := os.ReadFile(filepath.Join(s.Dir, "cursors", strings.TrimSuffix(indexName(source), ".jsonl")))
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	return strings.TrimSpace(string(b)), err
}

// SetCursor saves the position of a harvest of source.
func (s *Store) SetCursor(source, cursor string) error {
	dir := filepath.Join(s.Dir, "cursors")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	name := filepath.Join(dir, strings.TrimSuffix(indexName(source), ".jsonl"))
	tmp := name + ".tmp"
	if err := os.WriteFile(tmp, []byte(cursor+"\n"), 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, name)
}
