package katex

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// cacheEntry is one formula in a cache file.
type cacheEntry struct {
	TeX     string
	Display bool
	HTML    string
	Error   string // the message of KaTeX when it cannot parse the formula
	Failed  bool
}

// cacheName gives the name of the cache file for the macros. The name has a hash of the script, the options and the macros, so a new release of KaTeX or a new macro starts a new cache.
func cacheName(macros map[string]string) string {
	h := sha256.New()
	for _, s := range []string{Version, prelude, script, glue, lexerOnce} {
		h.Write([]byte(s))
		h.Write([]byte{0})
	}
	keys := make([]string, 0, len(macros))
	for k := range macros {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		h.Write([]byte(k + "\x00" + macros[k] + "\x00"))
	}
	return "katex-" + hex.EncodeToString(h.Sum(nil))[:16] + ".gob.gz"
}

// LoadCache reads the formulas that an earlier run saved in dir with SaveCache. A missing or broken file is an empty cache, as the renders give the same result.
func (r *Renderer) LoadCache(dir string) {
	f, err := os.Open(filepath.Join(dir, r.name)) // #nosec G304 -- the cache directory is from the user.
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	z, err := gzip.NewReader(f)
	if err != nil {
		return
	}
	var entries []cacheEntry
	if err := gob.NewDecoder(z).Decode(&entries); err != nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, e := range entries {
		res := result{html: e.HTML}
		if e.Failed {
			res.err = &Error{TeX: e.TeX, Message: e.Error}
		}
		k := key{e.TeX, e.Display}
		if _, ok := r.cache[k]; !ok {
			r.cache[k] = res
		}
	}
}

// SaveCache writes the formulas that the Renderer gave since New to a file in dir, and deletes the cache files of other releases or macros. The formulas that no render asked for go, so the file stays as large as the formulas of one site.
func (r *Renderer) SaveCache(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	r.mu.Lock()
	entries := make([]cacheEntry, 0, len(r.cache))
	for k, res := range r.cache {
		if !res.used || res.timeout {
			continue
		}
		e := cacheEntry{TeX: k.tex, Display: k.display, HTML: res.html}
		var ke *Error
		if errors.As(res.err, &ke) {
			e.Error, e.Failed = ke.Message, true
		}
		entries = append(entries, e)
	}
	r.mu.Unlock()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].TeX != entries[j].TeX {
			return entries[i].TeX < entries[j].TeX
		}
		return !entries[i].Display && entries[j].Display
	})
	tmp, err := os.CreateTemp(dir, r.name+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	z := gzip.NewWriter(tmp)
	if err := gob.NewEncoder(z).Encode(entries); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := z.Close(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, r.name)); err != nil {
		return err
	}
	old, err := fs.Glob(os.DirFS(dir), "katex-*.gob.gz")
	if err != nil {
		return err
	}
	for _, name := range old {
		if name != r.name && strings.HasPrefix(name, "katex-") {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
	return nil
}
