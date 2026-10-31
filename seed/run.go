package seed

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/tamnd/olymmath-agent/hf"
	"github.com/tamnd/olymmath-agent/store"
)

// Runner reads the data files of a source from the raw store and writes its seed rows.
type Runner struct {
	Store *store.Store
	HTTP  *http.Client
	Agent string
	Token string
	// Limit stops each file after this number of rows. Zero means no limit.
	Limit int
	Log   io.Writer
}

// Stats counts the rows of one source.
type Stats struct {
	Files   int   `json:"files"`
	Read    int   `json:"read"`
	Kept    int   `json:"kept"`
	Fetched int64 `json:"fetched"`
}

var errLimit = errors.New("limit")

// Run reads the newest manifest of source and writes one JSON line for each kept row to w.
func (r *Runner) Run(ctx context.Context, source string, w io.Writer) (Stats, error) {
	var st Stats
	m, err := hf.LatestManifest(ctx, r.Store, source)
	if err != nil {
		return st, err
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	prof, known := Profiles[source]
	for _, f := range m.Files {
		if !IsData(f.Path) || (known && prof.Files != nil && !prof.Files(f.Path)) || (GenericFiles[source] != nil && !GenericFiles[source](f.Path)) {
			continue
		}
		ra, closer, ranged, err := r.open(ctx, f)
		if err != nil {
			return st, fmt.Errorf("%s: %w", f.Path, err)
		}
		t, err := Open(f.Path, ra, f.Size)
		if err != nil {
			closer()
			return st, err
		}
		p := prof
		if !known {
			p = Generic(t.Columns())
		}
		if r.Log != nil {
			fmt.Fprintf(r.Log, "%s: %s (%s, %d bytes)\n", source, f.Path, f.Where, f.Size)
		}
		err = t.Each(p.Columns, func(i int, rec Record) error {
			if r.Limit > 0 && i >= r.Limit {
				return errLimit
			}
			st.Read++
			row, ok := p.Map(rec)
			if !ok {
				return nil
			}
			row.Source, row.File, row.Row = source, f.Path, i
			if len(row.Labels) == 0 {
				// A dataset of one competition often has no label column. The alias table can map the source label to that competition.
				row.Labels = []string{SourceLabel(source)}
			}
			row.Statement = strings.TrimSpace(row.Statement)
			st.Kept++
			return enc.Encode(row)
		})
		if ranged != nil {
			st.Fetched += ranged.Fetched
		}
		closer()
		if err != nil && !errors.Is(err, errLimit) {
			return st, fmt.Errorf("%s: %w", f.Path, err)
		}
		st.Files++
	}
	return st, nil
}

// open gives a reader of a data file: the blob on the local disk, or else range requests to the URL of the file at its pinned revision.
func (r *Runner) open(ctx context.Context, f hf.ManifestFile) (io.ReaderAt, func(), *Ranged, error) {
	if f.SHA256 != "" {
		if p, ok := r.Store.LocalPath(f.SHA256); ok {
			fh, err := os.Open(p) // #nosec G304 -- the path is a blob of the store.
			if err != nil {
				return nil, nil, nil, err
			}
			return fh, func() { _ = fh.Close() }, nil, nil
		}
	}
	if f.URL == "" {
		return nil, nil, nil, fmt.Errorf("the file is not on the local disk and has no URL")
	}
	rg := NewRanged(ctx, f.URL, f.Size)
	if r.HTTP != nil {
		rg.Client = r.HTTP
	}
	rg.Agent, rg.Token = r.Agent, r.Token
	return rg, func() {}, rg, nil
}

// SourceLabel is the label of a seed row that has no label of its own: "source:" and the source id.
func SourceLabel(source string) string { return "source:" + source }

// DefaultDir is the work directory of the seed rows: OLYMMATH_WORK/seed, or ~/.cache/olymmath/work/seed.
func DefaultDir() string {
	if d := os.Getenv("OLYMMATH_WORK"); d != "" {
		return filepath.Join(d, "seed")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".cache", "olymmath", "work", "seed")
}

// File is the path of the seed rows of source in dir.
func File(dir, source string) string { return filepath.Join(dir, source+".jsonl.gz") }

// Write runs source and writes its rows to File(dir, source). It writes to a temporary file first, so a failed run keeps the old file.
func (r *Runner) Write(ctx context.Context, dir, source string) (Stats, error) {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return Stats{}, err
	}
	tmp, err := os.CreateTemp(dir, source+"-*.tmp")
	if err != nil {
		return Stats{}, err
	}
	defer func() { _ = os.Remove(tmp.Name()) }()
	bw := bufio.NewWriterSize(tmp, 1<<20)
	gz := gzip.NewWriter(bw)
	st, err := r.Run(ctx, source, gz)
	if err == nil {
		err = gz.Close()
	}
	if err == nil {
		err = bw.Flush()
	}
	if cerr := tmp.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return st, err
	}
	return st, os.Rename(tmp.Name(), File(dir, source))
}

// Read calls fn for each seed row in the file of source in dir.
func Read(dir, source string, fn func(Row) error) error {
	f, err := os.Open(File(dir, source))
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gz, err := gzip.NewReader(bufio.NewReaderSize(f, 1<<20))
	if err != nil {
		return err
	}
	dec := json.NewDecoder(gz)
	for {
		var row Row
		if err := dec.Decode(&row); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		if err := fn(row); err != nil {
			return err
		}
	}
}

// Sources lists the sources that have seed rows in dir.
func Sources(dir string) ([]string, error) {
	m, err := filepath.Glob(filepath.Join(dir, "*.jsonl.gz"))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, p := range m {
		out = append(out, strings.TrimSuffix(filepath.Base(p), ".jsonl.gz"))
	}
	return out, nil
}
