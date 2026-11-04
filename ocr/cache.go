package ocr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// Entry is one answer in the cache.
type Entry struct {
	Text    string    `json:"text"`
	Lane    string    `json:"lane"`
	Model   string    `json:"model,omitempty"`
	Version string    `json:"version,omitempty"`
	Prompt  string    `json:"prompt"`
	At      time.Time `json:"at"`
}

// Cache keeps the answers of the model on disk, one file for each key.
type Cache struct {
	Dir string
}

// DefaultCache gives the cache in the cache directory of the user: $OLYMMATH_OCR, or olymmath/ocr in the user cache directory.
func DefaultCache() (*Cache, error) {
	if d := os.Getenv("OLYMMATH_OCR"); d != "" {
		return &Cache{Dir: d}, nil
	}
	d, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	return &Cache{Dir: filepath.Join(d, "olymmath", "ocr")}, nil
}

// Key gives the key of a page image: the hash of the bytes of the image and of the prompt version.
func Key(image []byte) string {
	h := sha256.New()
	h.Write([]byte(PromptVersion + "\n"))
	h.Write(image)
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Cache) path(key string) string {
	return filepath.Join(c.Dir, key[:2], key+".json")
}

// Get gives the entry of the key, and false when the cache has none.
func (c *Cache) Get(key string) (Entry, bool, error) {
	b, err := os.ReadFile(c.path(key))
	if errors.Is(err, fs.ErrNotExist) {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, err
	}
	var e Entry
	if err := json.Unmarshal(b, &e); err != nil {
		// A broken file is an answer that is not there. The next answer writes over it.
		return Entry{}, false, nil //nolint:nilerr // see above
	}
	return e, true, nil
}

// Put writes the entry of the key. It writes to a new file and renames it, so a reader never sees half a file.
func (c *Cache) Put(key string, e Entry) error {
	p := c.path(key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(p), ".put-")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), p)
}
