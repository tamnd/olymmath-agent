package publish

import (
	"bufio"
	"cmp"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Spool keeps the files of a build in one temporary file on the disk, and in memory only the place of each file. A release puts each file of site.RenderTo in the spool, plans the split from the paths, and then reads each file once to write it or to pack it. Thus the memory of a release does not grow with the bytes of the site.
type Spool struct {
	mu    sync.Mutex
	f     *os.File
	w     *bufio.Writer
	size  int64
	spans map[string]span
}

// span is the place of a file in the spool.
type span struct {
	off int64
	n   int
}

// NewSpool makes a spool in a new temporary file in dir. Remove deletes the file.
func NewSpool(dir string) (*Spool, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, err
	}
	f, err := os.CreateTemp(dir, ".spool-*")
	if err != nil {
		return nil, err
	}
	return &Spool{f: f, w: bufio.NewWriterSize(f, 1<<20), spans: map[string]span{}}, nil
}

// Put adds the file at the slash path rel. A path that comes twice keeps its last data, as on disk. It is safe for use by several goroutines.
func (s *Spool) Put(rel string, data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := s.w.Write(data); err != nil {
		return err
	}
	s.spans[rel] = span{off: s.size, n: len(data)}
	s.size += int64(len(data))
	return nil
}

// Flush writes the buffered data to the file. Call it after the last Put and before the first Read.
func (s *Spool) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.w.Flush()
}

// Has tells if the spool has the file at the slash path rel.
func (s *Spool) Has(rel string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok := s.spans[rel]
	return ok
}

// Read gives the data of the file at the slash path rel, or os.ErrNotExist. It is safe for use by several goroutines after Flush.
func (s *Spool) Read(rel string) ([]byte, error) {
	s.mu.Lock()
	sp, ok := s.spans[rel]
	s.mu.Unlock()
	if !ok {
		return nil, &os.PathError{Op: "read", Path: rel, Err: os.ErrNotExist}
	}
	data := make([]byte, sp.n)
	if _, err := s.f.ReadAt(data, sp.off); err != nil {
		return nil, err
	}
	return data, nil
}

// Paths gives the slash paths of the files in the order of their data in the spool, so a reader in this order reads the file from start to end.
func (s *Spool) Paths() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	type at struct {
		path string
		off  int64
	}
	all := make([]at, 0, len(s.spans))
	for p, sp := range s.spans {
		all = append(all, at{p, sp.off})
	}
	// Files with no bytes share an offset, so the path breaks the tie and the order is the same in each run.
	slices.SortFunc(all, func(a, b at) int { return cmp.Or(cmp.Compare(a.off, b.off), strings.Compare(a.path, b.path)) })
	paths := make([]string, len(all))
	for i, a := range all {
		paths[i] = a.path
	}
	return paths
}

// Len is the number of files in the spool.
func (s *Spool) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.spans)
}

// Remove closes and deletes the file of the spool.
func (s *Spool) Remove() error {
	name := filepath.Clean(s.f.Name())
	err := s.f.Close()
	if rerr := os.Remove(name); err == nil {
		err = rerr
	}
	return err
}
