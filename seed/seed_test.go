package seed

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/parquet-go/parquet-go"
)

func TestNorm(t *testing.T) {
	cases := map[string]string{
		"51st Ukrainian National Mathematical Olympiad, 2015": "ukrainian national mathematical olympiad",
		"NewZealand_MO":                        "new zealand mathematical olympiad",
		"BalticWay":                            "baltic way",
		"2023 Germany TST":                     "germany team selection test",
		"The Balkan Math Olympiad":             "balkan mathematical olympiad",
		"IMO":                                  "imo",
		"South-Afrika 2011-2013":               "south afrika",
		"Belarus2022":                          "belarus",
		"IV Soros Olympiad 1997 - 98 (Russia)": "soros olympiad russia",
		"53. Bulgarian Mathematical Olympiad":  "bulgarian mathematical olympiad",
		"AMC 10 A":                             "amc 10 a",
	}
	for in, want := range cases {
		if got := Norm(in); got != want {
			t.Errorf("Norm(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPosition(t *testing.T) {
	cases := map[string]string{"3": "p3", "P3": "p3", "Problem 12": "p12", "A1": "a1", "A-1": "a1", "B_6": "b6", "c7": "c7", "": "", "3a": "", "0": "", "B3802": "b3802", "K914": "k914", "1995": "", "P1234": ""}
	for in, want := range cases {
		if got := Position(in); got != want {
			t.Errorf("Position(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestYear(t *testing.T) {
	cases := map[string]int{"IRL_ABooklet_2023": 2023, "2016 Taiwan TST Round 1": 2016, "IMO": 0, "12345": 0}
	for in, want := range cases {
		if got := Year(in); got != want {
			t.Errorf("Year(%q) = %d, want %d", in, got, want)
		}
	}
}

type pqRow struct {
	Exam      string   `parquet:"exam"`
	Year      string   `parquet:"year"`
	Problem   string   `parquet:"problem"`
	Solutions []string `parquet:"solutions,list"`
	Image     []byte   `parquet:"image"`
	Meta      struct {
		Path string `parquet:"path"`
	} `parquet:"metadata"`
}

func parquetBytes(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := parquet.NewGenericWriter[pqRow](&buf, parquet.MaxRowsPerRowGroup(2))
	rows := []pqRow{
		{Exam: "IMO", Year: "2024", Problem: "first", Solutions: []string{"a", "b"}, Image: []byte("xx")},
		{Exam: "APMO", Year: "2001", Problem: "second"},
		{Exam: "USAMO", Year: "1999", Problem: "third", Solutions: []string{"c"}},
	}
	rows[2].Meta.Path = "USAMO/1999.jsonl"
	if _, err := w.Write(rows); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParquet(t *testing.T) {
	b := parquetBytes(t)
	tb, err := Open("x.parquet", bytes.NewReader(b), int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	if cols := strings.Join(tb.Columns(), ","); !strings.Contains(cols, "solutions") || !strings.Contains(cols, "metadata.path") {
		t.Fatalf("columns %s", cols)
	}
	var got []Record
	err = tb.Each([]string{"exam", "problem", "solutions", "metadata.path"}, func(i int, r Record) error {
		if len(got) != i {
			t.Errorf("row %d out of order", i)
		}
		got = append(got, r)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("%d rows", len(got))
	}
	if got[0].Get("exam") != "IMO" || len(got[0]["solutions"]) != 2 || len(got[1]["solutions"]) != 0 || got[2].Get("metadata.path") != "USAMO/1999.jsonl" {
		t.Errorf("rows %v", got)
	}
	if got[0].Has("image") {
		t.Error("the image column was read")
	}
}

func TestRanged(t *testing.T) {
	b := parquetBytes(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(w, r, "x.parquet", testTime, bytes.NewReader(b))
	}))
	defer srv.Close()
	rg := NewRanged(t.Context(), srv.URL, int64(len(b)))
	rg.Block = 64
	tb, err := Open("x.parquet", rg, int64(len(b)))
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	if err := tb.Each([]string{"problem"}, func(int, Record) error { n++; return nil }); err != nil {
		t.Fatal(err)
	}
	if n != 3 || rg.Fetched == 0 {
		t.Errorf("rows %d, fetched %d", n, rg.Fetched)
	}
	p := make([]byte, 10)
	if _, err := rg.ReadAt(p, int64(len(b))-5); err != io.EOF {
		t.Errorf("read past the end: %v", err)
	}
}

func TestJSONL(t *testing.T) {
	data := `{"problem":"p1","source":"imo","answer":3,"meta":{"year":2001}}
{"problem":"p2","source":"usamo","solution":["s1","s2"]}
`
	tb, err := Open("x.jsonl", strings.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	p := Generic(tb.Columns())
	var rows []Row
	err = tb.Each(append(p.Columns, "meta.year"), func(i int, r Record) error {
		row, ok := p.Map(r)
		if ok {
			rows = append(rows, row)
		}
		if i == 0 && r.Get("meta.year") != "2001" {
			t.Errorf("meta.year %q", r.Get("meta.year"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0].Answer != "3" || rows[1].Solutions != 2 || rows[1].Labels[0] != "usamo" {
		t.Errorf("rows %+v", rows)
	}
}

func TestCSV(t *testing.T) {
	data := "question,year\n\"What is 1+1?\",2020\n"
	tb, err := Open("x.csv", strings.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	p := Generic(tb.Columns())
	var row Row
	if err := tb.Each(p.Columns, func(_ int, r Record) error { row, _ = p.Map(r); return nil }); err != nil {
		t.Fatal(err)
	}
	if row.Statement != "What is 1+1?" || row.Year != 2020 {
		t.Errorf("row %+v", row)
	}
}

func TestProfiles(t *testing.T) {
	r := Record{"metadata.path": {"Contest Collections/2023 Contests/2023 Germany Team Selection Test/3107329.json"}, "problem": {"x"}}
	row, ok := Profiles["hf-ai-mo-aops"].Map(r)
	if !ok || row.Year != 2023 || row.Labels[0] != "Germany Team Selection Test" || row.Ref != "aops-topic:3107329" {
		t.Errorf("aops row %+v", row)
	}
	if _, ok := Profiles["hf-numinamath-1-5"].Map(Record{"synthetic": {"true"}, "problem": {"x"}, "source": {"olympiads"}}); ok {
		t.Error("a synthetic row was kept")
	}
	if _, ok := Profiles["hf-numinamath-1-5"].Map(Record{"synthetic": {"false"}, "problem": {"x"}, "source": {"cn_k12"}}); ok {
		t.Error("a textbook row was kept")
	}
	row, _ = Profiles["hf-numina-olympiads"].Map(Record{"exam": {"IMO"}, "year": {"1990"}, "problem_label": {"1"}, "problem": {"x"}, "metadata.resource_path": {"IMO/segmented/en-compendium.jsonl"}})
	if row.Labels[0] != "IMO Compendium" {
		t.Errorf("compendium label %q", row.Labels[0])
	}
	row, _ = Profiles["hf-numina-olympiads-ref"].Map(Record{"metadata.exam": {"IMO"}, "problem": {"x"}, "metadata.resource_path": {"IMO/segmented/en-IMO-2010-notes.jsonl"}})
	if row.Labels[0] != "IMO" || row.Author != "evanchen" {
		t.Errorf("notes row %+v", row)
	}
	row, _ = Profiles["hf-ai-mo-aops"].Map(Record{"metadata.path": {"Contest Collections/2022 Contests/2022 3rd Memorial &quot;A&quot;/1.json"}, "problem": {"x"}})
	if row.Labels[0] != `3rd Memorial "A"` {
		t.Errorf("label %q", row.Labels[0])
	}
	if p := Generic([]string{"problem_statement", "reasoning", "label"}); len(p.Columns) != 2 {
		t.Errorf("generic columns %v", p.Columns)
	}
	if !IsData("data/x.parquet") || IsData("config.json") || IsData("sub/dataset_info.json") {
		t.Error("IsData")
	}
	g := Generic([]string{"question", "id", "part", "index"})
	row, _ = g.Map(Record{"question": {"q"}, "id": {"A-1"}})
	if row.Number != "A-1" {
		t.Errorf("number from id %+v", row)
	}
	row, _ = g.Map(Record{"question": {"q"}, "part": {"AIME I"}, "index": {"7"}})
	if row.Number != "7" || row.Labels[0] != "AIME I" {
		t.Errorf("aime row %+v", row)
	}
	row, ok = Profiles["hf-amc12-full"].Map(Record{"problem_id": {"2019A-P15"}, "question": {"x"}, "answer": {"D"}})
	if !ok || row.Year != 2019 || row.Number != "15" || row.Labels[0] != "AMC 12A" || row.Answer != "D" {
		t.Errorf("amc12 row %+v", row)
	}
	row, _ = Profiles["hf-amc12-full"].Map(Record{"problem_id": {"2001A-P3"}, "question": {"x"}})
	if row.Labels[0] != "AMC 12" || row.Number != "3" {
		t.Errorf("amc12 2001 row %+v", row)
	}
	row, ok = Profiles["hf-mathnet"].Map(Record{"problem_markdown": {"Problem:\nShow it."}, "competition": {"IRL_ABooklet_2023"}, "country": {"Ireland"}})
	if !ok || row.Statement != "Show it." || row.Year != 2023 {
		t.Errorf("mathnet row %+v", row)
	}
}

func TestReadWrite(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	enc := json.NewEncoder(gz)
	for _, r := range []Row{{Source: "s", Statement: "a"}, {Source: "s", Statement: "b"}} {
		if err := enc.Encode(r); err != nil {
			t.Fatal(err)
		}
	}
	_ = gz.Close()
	if err := os.WriteFile(filepath.Join(dir, "s.jsonl.gz"), buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	srcs, _ := Sources(dir)
	if len(srcs) != 1 || srcs[0] != "s" {
		t.Fatalf("sources %v", srcs)
	}
	n := 0
	if err := Read(dir, "s", func(Row) error { n++; return nil }); err != nil || n != 2 {
		t.Errorf("read %d rows, %v", n, err)
	}
}

var testTime = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
