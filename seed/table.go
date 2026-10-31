// Package seed reads the public datasets of wave 0 from the raw store and writes one seed row for each problem row. A seed row keeps the labels of the dataset row, its statement and a pointer back to the row. The rows are evidence only: they show what exists and where. The canonical text comes from milestone M4.
package seed

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/parquet-go/parquet-go"
)

// Record is one row of a table. It maps a column name to the values of the column in the row. A plain column has one value, a list column has one value for each item, and a null has no value.
type Record map[string][]string

// Get returns the first value of the column, or "".
func (r Record) Get(col string) string {
	if v := r[col]; len(v) > 0 {
		return v[0]
	}
	return ""
}

// First returns the first value of the first column that has a value that is not empty.
func (r Record) First(cols ...string) string {
	for _, c := range cols {
		if v := strings.TrimSpace(r.Get(c)); v != "" {
			return v
		}
	}
	return ""
}

// Has tells if the table of the record has the column.
func (r Record) Has(col string) bool {
	_, ok := r[col]
	return ok
}

// Table is a file of rows. Each walks the rows in order and gives only the columns that are in cols. A column name is a top-level field, or a dotted path to a field in a struct.
type Table interface {
	Columns() []string
	Each(cols []string, fn func(i int, r Record) error) error
}

// Open returns the table of a data file. The format comes from the extension of the name.
func Open(name string, r io.ReaderAt, size int64) (Table, error) {
	switch ext(name) {
	case "parquet":
		f, err := parquet.OpenFile(r, size, parquet.SkipPageIndex(true), parquet.SkipBloomFilters(true), parquet.ReadBufferSize(1<<20))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
		return &parquetTable{f: f}, nil
	case "jsonl", "ndjson":
		return &jsonTable{r: io.NewSectionReader(r, 0, size), lines: true}, nil
	case "json":
		return &jsonTable{r: io.NewSectionReader(r, 0, size)}, nil
	case "csv":
		return &csvTable{r: io.NewSectionReader(r, 0, size)}, nil
	}
	return nil, fmt.Errorf("%s: the format is not known", name)
}

// IsData tells if Open can read the file.
func IsData(name string) bool {
	switch ext(name) {
	case "parquet", "jsonl", "ndjson", "json", "csv":
		return true
	}
	return false
}

func ext(name string) string {
	name = strings.TrimSuffix(strings.ToLower(name), "/")
	if i := strings.LastIndexByte(name, '.'); i >= 0 {
		return name[i+1:]
	}
	return ""
}

type parquetTable struct{ f *parquet.File }

// Columns gives the top-level fields and the dotted paths of the struct fields. The "list" and "element" levels of a list are not in the names.
func (t *parquetTable) Columns() []string {
	seen := map[string]bool{}
	var out []string
	for _, path := range t.f.Schema().Columns() {
		name := leafName(path)
		for i := 1; i <= len(name); i++ {
			n := strings.Join(name[:i], ".")
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out
}

// leafName removes the levels that the Parquet list form adds.
func leafName(path []string) []string {
	out := make([]string, 0, len(path))
	for i := 0; i < len(path); i++ {
		if i > 0 && path[i] == "list" && i+1 < len(path) && (path[i+1] == "element" || path[i+1] == "item") {
			i++
			continue
		}
		out = append(out, path[i])
	}
	return out
}

// Each reads only the column chunks of the selected columns, so a large column such as an image column is not read. A selected name that is a struct gives the values of its first leaf.
func (t *parquetTable) Each(cols []string, fn func(int, Record) error) error {
	leaves := t.f.Schema().Columns()
	pick := map[int]string{}
	for _, c := range cols {
		for i, path := range leaves {
			n := strings.Join(leafName(path), ".")
			if n == c || strings.HasPrefix(n, c+".") {
				pick[i] = c
				break
			}
		}
	}
	base := 0
	for _, rg := range t.f.RowGroups() {
		n := int(rg.NumRows())
		rows := make([]Record, n)
		for i := range rows {
			rows[i] = Record{}
		}
		chunks := rg.ColumnChunks()
		for i, name := range pick {
			vals, err := readChunk(chunks[i], n)
			if err != nil {
				return fmt.Errorf("column %s: %w", name, err)
			}
			for j := range rows {
				rows[j][name] = vals[j]
			}
		}
		for j, r := range rows {
			if err := fn(base+j, r); err != nil {
				return err
			}
		}
		base += n
	}
	return nil
}

// readChunk gives the values of a column chunk, grouped by row. A value with repetition level 0 starts a new row.
func readChunk(cc parquet.ColumnChunk, n int) ([][]string, error) {
	out := make([][]string, 0, n)
	pages := cc.Pages()
	defer pages.Close()
	buf := make([]parquet.Value, 512)
	for {
		p, err := pages.ReadPage()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		vr := p.Values()
		for {
			k, err := vr.ReadValues(buf)
			for _, v := range buf[:k] {
				if v.RepetitionLevel() == 0 {
					out = append(out, nil)
				}
				if v.IsNull() || len(out) == 0 {
					continue
				}
				out[len(out)-1] = append(out[len(out)-1], valueString(v))
			}
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				parquet.Release(p)
				return nil, err
			}
		}
		parquet.Release(p)
	}
	for len(out) < n {
		out = append(out, nil)
	}
	return out[:n], nil
}

func valueString(v parquet.Value) string {
	switch v.Kind() {
	case parquet.ByteArray, parquet.FixedLenByteArray:
		return string(v.ByteArray())
	case parquet.Boolean:
		return strconv.FormatBool(v.Boolean())
	case parquet.Int32:
		return strconv.FormatInt(int64(v.Int32()), 10)
	case parquet.Int64:
		return strconv.FormatInt(v.Int64(), 10)
	case parquet.Float:
		return strconv.FormatFloat(float64(v.Float()), 'g', -1, 32)
	case parquet.Double:
		return strconv.FormatFloat(v.Double(), 'g', -1, 64)
	}
	return v.String()
}

// jsonTable reads a file of JSON lines, or a file with one JSON array of objects.
type jsonTable struct {
	r     io.ReadSeeker
	lines bool
	cols  []string
}

func (t *jsonTable) Columns() []string {
	if t.cols == nil {
		seen := map[string]bool{}
		t.cols = []string{}
		_ = t.Each(nil, func(i int, r Record) error {
			keys := make([]string, 0, len(r))
			for k := range r {
				if !seen[k] {
					seen[k] = true
					keys = append(keys, k)
				}
			}
			sort.Strings(keys)
			t.cols = append(t.cols, keys...)
			if i >= 199 {
				return io.EOF
			}
			return nil
		})
	}
	return t.cols
}

func (t *jsonTable) Each(cols []string, fn func(int, Record) error) error {
	if _, err := t.r.Seek(0, io.SeekStart); err != nil {
		return err
	}
	dec := json.NewDecoder(bufio.NewReaderSize(t.r, 1<<20))
	dec.UseNumber()
	if !t.lines {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		if d, ok := tok.(json.Delim); !ok || d != '[' {
			return fmt.Errorf("the JSON file is not an array")
		}
	}
	for i := 0; dec.More(); i++ {
		var obj map[string]any
		if err := dec.Decode(&obj); err != nil {
			return fmt.Errorf("row %d: %w", i, err)
		}
		r := Record{}
		if cols == nil {
			for k := range obj {
				r[k] = flat(obj[k])
			}
		}
		for _, c := range cols {
			if v, ok := lookup(obj, c); ok {
				r[c] = flat(v)
			}
		}
		if err := fn(i, r); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
	return nil
}

func lookup(obj map[string]any, path string) (any, bool) {
	if v, ok := obj[path]; ok {
		return v, true
	}
	head, rest, ok := strings.Cut(path, ".")
	if !ok {
		return nil, false
	}
	sub, ok := obj[head].(map[string]any)
	if !ok {
		return nil, false
	}
	return lookup(sub, rest)
}

func flat(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case string:
		return []string{x}
	case json.Number:
		return []string{x.String()}
	case bool:
		return []string{strconv.FormatBool(x)}
	case []any:
		var out []string
		for _, e := range x {
			out = append(out, flat(e)...)
		}
		return out
	}
	b, _ := json.Marshal(v)
	return []string{string(b)}
}

// csvTable reads a CSV file with a header row.
type csvTable struct {
	r    io.ReadSeeker
	cols []string
}

func (t *csvTable) Columns() []string {
	if t.cols == nil {
		_ = t.Each(nil, func(int, Record) error { return io.EOF })
	}
	return t.cols
}

func (t *csvTable) Each(cols []string, fn func(int, Record) error) error {
	if _, err := t.r.Seek(0, io.SeekStart); err != nil {
		return err
	}
	cr := csv.NewReader(bufio.NewReaderSize(t.r, 1<<20))
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	head, err := cr.Read()
	if err != nil {
		return err
	}
	t.cols = head
	for i := 0; ; i++ {
		rec, err := cr.Read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("row %d: %w", i, err)
		}
		r := Record{}
		for j, h := range head {
			if j < len(rec) && (cols == nil || contains(cols, h)) {
				r[h] = []string{rec[j]}
			}
		}
		if err := fn(i, r); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
