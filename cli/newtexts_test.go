package cli

import (
	"fmt"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/seed"
)

func TestNewTextsSample(t *testing.T) {
	// Each text has 10 bytes and the limit is 25, so the sample keeps the 2 texts with the smallest keys.
	n := newNewTexts(25)
	for _, k := range []uint64{7, 3, 9, 1, 3, 10, 2, 5, 8, 4, 6} {
		n.add(k, fmt.Sprintf("text %5d", k))
	}
	if len(n.seen) != 10 || !n.sampled() || len(n.kept) != 2 || n.kept[1] == "" || n.kept[2] == "" {
		t.Fatalf("seen %d, kept %v", len(n.seen), n.kept)
	}
	var a Against
	n.count(&a, 1)
	if a.NewTexts != 10 || a.Sample != 2 || a.SampleProblems != 2 || a.NewProblems != 10 {
		t.Errorf("against: %+v", a)
	}
	if !strings.Contains(a.line(100), "about 10 new problems (about 1000 for each 10,000 rows; 2 groups in a sample of 2 of the 10 new texts)") {
		t.Errorf("line: %s", a.line(100))
	}
	var b strings.Builder
	rateTables(func(f string, x ...any) { fmt.Fprintf(&b, f, x...) }, []MatchRate{{Source: "hf-big", Rows: 100, Placed: a, All: a}})
	for _, want := range []string{"| hf-big | 100 | about 1000 | about 1000 |", "- hf-big against all the records: 2 groups in a sample of 2 of the 10 new texts."} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("no %q in\n%s", want, b.String())
		}
	}
}

func TestNewTextsAll(t *testing.T) {
	n := newNewTexts(1 << 20)
	n.add(1, "Kangaroos jump along hexagonal lattices until purple tulips bloom.")
	n.add(2, "Kangaroos jump along hexagonal lattices until purple tulips bloom!")
	n.add(3, "Wombats stack copper teacups beside quiet volcanoes.")
	var a Against
	n.count(&a, 2)
	if n.sampled() || a.NewTexts != 3 || a.Sample != 0 || a.NewProblems != 2 {
		t.Errorf("against: %+v", a)
	}
}

func TestOpenMathReasoningFiles(t *testing.T) {
	f := seed.GenericFiles["hf-openmathreasoning"]
	if f == nil || f("data/genselect-00000-of-00010.parquet") || !f("data/cot-00000-of-00144.parquet") {
		t.Error("the genselect rows of OpenMathReasoning are read")
	}
}
