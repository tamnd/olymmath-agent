package katex

import (
	"bufio"
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

// streamFormat is the comment in the gzip header of a cache file that is a stream of entries, one gob value each. A file of an older om has no comment and one gob value with all the entries. The comment tells the two apart before the first read, because a gob decoder reads all of a value into memory before it checks the type.
const streamFormat = "om katex entries"

// LoadCache reads the formulas that an earlier run saved in dir with SaveCache. A missing or broken file is an empty cache, as the renders give the same result. A file in the stream format loads one entry at a time, so the load needs no buffer for the whole file. A file of an older om loads too.
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
	dec := gob.NewDecoder(bufio.NewReader(z))
	r.mu.Lock()
	defer r.mu.Unlock()
	if z.Comment != streamFormat {
		var entries []cacheEntry
		if dec.Decode(&entries) == nil {
			for _, e := range entries {
				if r.add(e) != nil {
					return
				}
			}
		}
		return
	}
	for {
		var e cacheEntry
		if dec.Decode(&e) != nil || r.add(e) != nil {
			return
		}
	}
}

// add puts an entry of a cache file in the cache, unless the formula is there. r.mu must be held.
func (r *Renderer) add(e cacheEntry) error {
	k := key{e.TeX, e.Display}
	if _, ok := r.cache[k]; ok {
		return nil
	}
	html, err := r.squeeze(e.HTML)
	if err != nil {
		return err
	}
	res := result{html: html, size: len(e.HTML)}
	if e.Failed {
		res.err = &Error{TeX: e.TeX, Message: e.Error}
	}
	r.cache[k] = res
	return nil
}

// SaveCache writes the formulas that the Renderer gave since New to a file in dir, and deletes the cache files of other releases or macros. The formulas that no render asked for go, so the file stays as large as the formulas of one site.
func (r *Renderer) SaveCache(dir string) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	// The entries keep the compressed HTML until the encoder needs it, so the save needs no memory for the HTML of all the formulas.
	type saved struct {
		k   key
		res result
	}
	r.mu.Lock()
	entries := make([]saved, 0, len(r.cache))
	for k, res := range r.cache {
		if res.used && !res.timeout {
			entries = append(entries, saved{k, res})
		}
	}
	r.mu.Unlock()
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].k.tex != entries[j].k.tex {
			return entries[i].k.tex < entries[j].k.tex
		}
		return !entries[i].k.display && entries[j].k.display
	})
	tmp, err := os.CreateTemp(dir, r.name+".*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	z := gzip.NewWriter(tmp)
	z.Comment = streamFormat
	// One gob value for each entry, so the encoder needs no buffer for the whole file.
	enc := gob.NewEncoder(z)
	for _, s := range entries {
		e := cacheEntry{TeX: s.k.tex, Display: s.k.display}
		var ke *Error
		if errors.As(s.res.err, &ke) {
			e.Error, e.Failed = ke.Message, true
		}
		html, err := r.expand(s.res)
		if err == nil {
			e.HTML = html
			err = enc.Encode(e)
		}
		if err != nil {
			_ = tmp.Close()
			return err
		}
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
