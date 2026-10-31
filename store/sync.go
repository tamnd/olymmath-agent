package store

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/parquet-go/parquet-go"
)

// The remote keeps a copy of each index under IndexKey and the list of the index files under ListKey. The index is small, and with the copy a new machine can read the raw store from R2 alone.
const (
	indexPrefix = "index/"
	ListKey     = indexPrefix + "LIST"
	// ParquetKey is the remote key of all captures of all sources in one Parquet file, for queries with DuckDB.
	ParquetKey = indexPrefix + "captures.parquet"
	syncedFile = "synced"
)

// IndexKey is the remote key of the index file name.
func IndexKey(name string) string { return indexPrefix + name }

// SyncStats counts the work of Sync.
type SyncStats struct {
	Blobs   int
	Bytes   int64
	Skipped int
	Indexes int
	// Captures is the number of rows of the Parquet index.
	Captures int
}

// Sync sends each local blob that the remote does not have yet, then each index file and the list of the index files. The file "synced" in the store directory keeps the hashes that Sync sent, so a second run sends only the new blobs.
func (s *Store) Sync(ctx context.Context, log io.Writer) (SyncStats, error) {
	var st SyncStats
	if s.Remote == nil {
		return st, errors.New("the store has no remote")
	}
	done, err := s.synced()
	if err != nil {
		return st, err
	}
	sf, err := os.OpenFile(filepath.Join(s.Dir, syncedFile), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return st, err
	}
	defer sf.Close()
	names, err := s.indexNames()
	if err != nil {
		return st, err
	}
	for _, name := range names {
		cs, err := s.readIndex(name)
		if err != nil {
			return st, err
		}
		for _, c := range cs {
			if done[c.Hash] || !s.HasLocal(c.Hash) {
				if done[c.Hash] {
					st.Skipped++
				}
				continue
			}
			if err := s.Push(ctx, c.Hash); err != nil {
				return st, err
			}
			done[c.Hash] = true
			st.Blobs++
			st.Bytes += c.Size
			if _, err := fmt.Fprintln(sf, c.Hash); err != nil {
				return st, err
			}
			if log != nil {
				fmt.Fprintf(log, "pushed %s %d\n", c.Hash, c.Size)
			}
		}
	}
	for _, name := range names {
		b, err := os.ReadFile(filepath.Join(s.Dir, "index", name))
		if err != nil {
			return st, err
		}
		if err := s.Remote.Put(ctx, IndexKey(name), bytes.NewReader(b), int64(len(b))); err != nil {
			return st, err
		}
		st.Indexes++
	}
	var all []Capture
	for _, name := range names {
		cs, err := s.readIndex(name)
		if err != nil {
			return st, err
		}
		all = append(all, cs...)
	}
	pq, err := ParquetIndex(all)
	if err != nil {
		return st, err
	}
	if err := s.Remote.Put(ctx, ParquetKey, bytes.NewReader(pq), int64(len(pq))); err != nil {
		return st, err
	}
	st.Captures = len(all)
	list := []byte(strings.Join(names, "\n") + "\n")
	return st, s.Remote.Put(ctx, ListKey, bytes.NewReader(list), int64(len(list)))
}

// ParquetIndex writes the captures as one Parquet file.
func ParquetIndex(cs []Capture) ([]byte, error) {
	var buf bytes.Buffer
	w := parquet.NewGenericWriter[Capture](&buf)
	if _, err := w.Write(cs); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Restore gets the index files from the remote. It does not get the blobs: Get reads a blob from the remote when the local disk does not have it. A local index file that exists is kept when it is longer than the remote copy.
func (s *Store) Restore(ctx context.Context) (int, error) {
	if s.Remote == nil {
		return 0, errors.New("the store has no remote")
	}
	rc, err := s.Remote.Get(ctx, ListKey)
	if err != nil {
		return 0, err
	}
	b, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return 0, err
	}
	n := 0
	for _, name := range strings.Fields(string(b)) {
		if strings.Contains(name, "/") || !strings.HasSuffix(name, ".jsonl") {
			return n, fmt.Errorf("the remote list has the bad name %q", name)
		}
		rc, err := s.Remote.Get(ctx, IndexKey(name))
		if err != nil {
			return n, err
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			return n, err
		}
		p := filepath.Join(s.Dir, "index", name)
		if fi, err := os.Stat(p); err == nil && fi.Size() >= int64(len(data)) {
			continue
		}
		if err := os.WriteFile(p, data, 0o640); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (s *Store) indexNames() ([]string, error) {
	es, err := os.ReadDir(filepath.Join(s.Dir, "index"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range es {
		if strings.HasSuffix(e.Name(), ".jsonl") {
			out = append(out, e.Name())
		}
	}
	return out, nil
}

// readIndex reads the index file name. The first capture gives the source.
func (s *Store) readIndex(name string) ([]Capture, error) {
	f, err := os.Open(filepath.Join(s.Dir, "index", name))
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var c Capture
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, nil
	}
	return s.Captures(c.Source)
}

func (s *Store) synced() (map[string]bool, error) {
	out := map[string]bool{}
	f, err := os.Open(filepath.Join(s.Dir, syncedFile))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		out[strings.TrimSpace(sc.Text())] = true
	}
	return out, sc.Err()
}
