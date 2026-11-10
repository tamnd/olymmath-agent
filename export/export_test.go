package export

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/parquet-go/parquet-go"

	"github.com/tamnd/olymmath-agent/content"
)

func load(t *testing.T) *content.Corpus {
	t.Helper()
	corpus, fs := content.Load("../content/testdata/valid")
	if len(fs) != 0 {
		t.Fatal(fs)
	}
	return corpus
}

func readRows(t *testing.T, dir, config string) []Row {
	t.Helper()
	rows, err := parquet.ReadFile[Row](filepath.Join(dir, config, "train.parquet"))
	if err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestConfigs(t *testing.T) {
	dir := t.TempDir()
	st, err := Write(load(t), dir, Options{URL: "https://example.org", Commit: "abc"})
	if err != nil {
		t.Fatal(err)
	}
	if st.Rows["z1-text"] != 1 || st.Rows["z2-facts"] != 2 || st.Rows["z3-links"] != 0 {
		t.Errorf("rows: %v", st.Rows)
	}
	z1 := readRows(t, dir, "z1-text")
	if len(z1) != 1 || z1[0].Tag != "0J8TP3MFS9" || z1[0].Zone != "Z1" || z1[0].Source != "mathse" || z1[0].Licence != "CC-BY-SA-4.0" || z1[0].Statement == "" || z1[0].URL != "https://example.org/p/0J8TP3MFS9" {
		t.Errorf("z1-text: %+v", z1)
	}
	z2 := readRows(t, dir, "z2-facts")
	if len(z2) != 2 || z2[0].Locator != "imo/2024/p1" || z2[0].Zone != "Z2" || z2[0].Licence != Reference || z2[0].Answer == "" || z2[0].Statement == "" {
		t.Fatalf("z2-facts: %+v", z2)
	}
	if s := z2[0].Solutions; len(s) != 2 || s[0].Text == "" || s[0].Licence != "CC-BY-SA-4.0" || s[1].Origin != "ai" || s[1].Model != "claude-opus-5-5" {
		t.Errorf("solution: %+v", s)
	}
	if got := readRows(t, dir, "z3-links"); len(got) != 0 {
		t.Errorf("z3-links: %+v", got)
	}
	card, err := os.ReadFile(filepath.Join(dir, "README.md")) // #nosec G304 -- the path is in the test.
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"- config_name: z1-text", "path: z2-facts/train.parquet", "| `z2-facts` | 2 |", "ai-train=no", "commit abc"} {
		if !strings.Contains(string(card), want) {
			t.Errorf("the card has no %q", want)
		}
	}
}

// TestNoAITrainText wants no text of a source with ai_train: no in any config, and the facts and the link kept.
func TestNoAITrainText(t *testing.T) {
	corpus := load(t)
	corpus.Source("imo-official").AITrain = "no"
	corpus.Source("mathse").AITrain = "no"
	// The solution of IMO 2024 P1 now comes from Math SE too.
	sol := corpus.ByTag("0HVHSDNNMY").Solutions[0]
	sol.Source = "mathse"
	dir := t.TempDir()
	st, err := Write(corpus, dir, Options{URL: "https://example.org"})
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, c := range Configs {
		for _, r := range readRows(t, dir, c.Name) {
			n++
			if r.Statement != "" || r.LeftOut != LeftAITrain || r.URL == "" || r.Source == "" {
				t.Errorf("%s %s: the statement of an ai-train=no source: %+v", c.Name, r.Tag, r)
			}
			for _, s := range r.Solutions {
				// An AI solution has no source, so the flag of a source does not apply to it.
				if s.Origin == "ai" {
					continue
				}
				if s.Text != "" || s.LeftOut != LeftAITrain {
					t.Errorf("%s: the solution of an ai-train=no source: %+v", r.Tag, s)
				}
			}
		}
	}
	if n != 3 || st.LeftOut[LeftAITrain] != 4 || strings.Join(st.Sources, ",") != "imo-official,mathse" {
		t.Errorf("%d rows, stats %+v", n, st)
	}
}

func TestDoNotRepublishAndLinkOnly(t *testing.T) {
	corpus := load(t)
	corpus.Source("mathse").DoNotRepublish = true
	p := corpus.ByTag("03JXB69J3N")
	p.Source.Zone = "Z3"
	rows, _ := Rows(corpus, Options{})
	if r := rows["z1-text"][0]; r.Statement != "" || r.LeftOut != LeftNoRepub {
		t.Errorf("do-not-republish: %+v", r)
	}
	if r := rows["z3-links"]; len(r) != 1 || r[0].Statement != "" || r[0].LeftOut != LeftLinkOnly {
		t.Errorf("link only: %+v", r)
	}
}

func TestSameBytes(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	for _, d := range []string{a, b} {
		if _, err := Write(load(t), d, Options{URL: "https://example.org"}); err != nil {
			t.Fatal(err)
		}
	}
	for _, rel := range []string{"README.md", "z1-text/train.parquet", "z2-facts/train.parquet", "z3-links/train.parquet"} {
		x, _ := os.ReadFile(filepath.Join(a, rel)) // #nosec G304 -- the path is in the test.
		y, _ := os.ReadFile(filepath.Join(b, rel)) // #nosec G304 -- the path is in the test.
		if len(x) == 0 || !bytes.Equal(x, y) {
			t.Errorf("%s differs between two exports", rel)
		}
	}
}

func TestFigures(t *testing.T) {
	corpus := load(t)
	p := corpus.ByTag("03JXB69J3N")
	p.Statement += "\n\n![Figure 1](p2.fig1.svg)"
	rows, _ := Rows(corpus, Options{URL: "https://example.org"})
	for _, r := range rows["z2-facts"] {
		want := "[]"
		if r.Tag == p.Tag {
			want = "[https://example.org/imo/2024/p2.fig1.svg]"
		}
		if got := fmt.Sprint(r.Figures); got != want || r.Figures == nil {
			t.Errorf("%s: figures %s, want %s", r.Tag, got, want)
		}
	}
	// A figure is part of the statement, so a source with ai-train=no leaves both out.
	corpus.Source("imo-official").AITrain = "no"
	rows, _ = Rows(corpus, Options{URL: "https://example.org"})
	for _, r := range rows["z2-facts"] {
		if len(r.Figures) != 0 || r.Figures == nil {
			t.Errorf("%s: figures %v of an ai-train=no source", r.Tag, r.Figures)
		}
	}
}
