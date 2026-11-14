package calibrate

import (
	"bytes"
	"math"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/grade"
	"github.com/tamnd/olymmath-agent/model"
)

func TestKappa(t *testing.T) {
	// The example of Cohen (1960) in the form of two raters and two categories: 20 yes-yes, 5 yes-no, 10 no-yes, 15 no-no.
	var a, b []int
	add := func(x, y, n int) {
		for range n {
			a, b = append(a, x), append(b, y)
		}
	}
	add(1, 1, 20)
	add(1, 0, 5)
	add(0, 1, 10)
	add(0, 0, 15)
	if k := Kappa(a, b, 2, false); math.Abs(k-0.4) > 1e-9 {
		t.Fatalf("Kappa = %v, want 0.4", k)
	}
	if k := Kappa([]int{0, 1, 2, 3}, []int{0, 1, 2, 3}, 4, true); k != 1 {
		t.Fatalf("the kappa of equal ratings is %v", k)
	}
	if k := Kappa([]int{1, 1}, []int{1, 1}, 2, false); !math.IsNaN(k) {
		t.Fatalf("the kappa of one category is %v, want NaN", k)
	}
	// Quadratic weights cost a near miss less than a far one.
	near := Kappa([]int{0, 7, 3, 5}, []int{0, 7, 4, 5}, 8, true)
	far := Kappa([]int{0, 7, 3, 5}, []int{0, 7, 0, 5}, 8, true)
	if near <= far {
		t.Fatalf("near %v, far %v", near, far)
	}
}

func TestSampleInterval(t *testing.T) {
	var items []Item
	for i := range 100 {
		items = append(items, Item{ID: strconv.Itoa(i)})
	}
	a, b := Sample(items, 10, 7), Sample(items, 10, 7)
	if len(a) != 10 || !slices.EqualFunc(a, b, func(x, y Item) bool { return x.ID == y.ID }) {
		t.Fatalf("the samples of one seed differ: %v and %v", a, b)
	}
	if c := Sample(items, 10, 8); slices.EqualFunc(a, c, func(x, y Item) bool { return x.ID == y.ID }) {
		t.Fatal("two seeds give the same sample")
	}
	if len(Sample(items, 0, 7)) != 100 || len(Sample(items, 200, 7)) != 100 {
		t.Fatal("a sample of 0 or of more than all is not all")
	}
	// Agreement on 40 of 50 items: the kappa is in its interval, and the interval is not empty.
	var x, y []int
	for i := range 50 {
		x = append(x, i%2)
		if i < 40 {
			y = append(y, i%2)
		} else {
			y = append(y, 1-i%2)
		}
	}
	k := Kappa(x, y, 2, false)
	lo, hi := Interval(x, y, 2, false)
	if !(lo < k && k < hi) || lo < -1 || hi > 1 {
		t.Fatalf("kappa %v, interval %v to %v", k, lo, hi)
	}
	if lo, _ := Interval([]int{1, 1}, []int{1, 1}, 2, false); !math.IsNaN(lo) {
		t.Fatal("the interval of a NaN kappa is not NaN")
	}
}

func TestRead(t *testing.T) {
	gb, err := SetByName("gradingbench").Read("testdata/gradingbench.csv")
	if err != nil || len(gb) != 4 || gb[2].Human != 6 || gb[2].Label != "almost" || gb[0].Guidelines == "" || gb[0].Reference != "Reflexivity." {
		t.Fatalf("gradingbench: %v, %+v", err, gb)
	}
	pb, err := SetByName("proofbench").Read("testdata/proofbench.jsonl")
	if err != nil || len(pb) != 3 || pb[1].Human != 7 || pb[1].ID == pb[2].ID || pb[1].Model != model.Sonnet {
		t.Fatalf("proofbench: %v, %+v", err, pb)
	}
	opc, err := SetByName("opc").Read("testdata/opc.jsonl")
	if err != nil || len(opc) != 2 || opc[0].Human != 1 || opc[1].Human != 0 {
		t.Fatalf("opc: %v, %+v", err, opc)
	}
	// A juror that wrote the proof is not on its jury.
	if j := pb[1].Task(grade.DefaultJurors).Jurors; len(j) != 2 || j[0] == model.Sonnet || j[1] == model.Sonnet {
		t.Fatalf("the jury of a Sonnet proof is %v", j)
	}
}

func TestCompareReport(t *testing.T) {
	set := SetByName("gradingbench")
	items, _ := set.Read("testdata/gradingbench.csv")
	jury := map[string]int{}
	for _, it := range items {
		jury[it.ID] = it.Human
	}
	a := Compare(set, items, jury)
	if a.N != 4 || a.Exact != 1 || a.Kappa != 1 || a.Quadratic != 1 {
		t.Fatalf("the agreement is %+v", a)
	}
	opc := SetByName("opc")
	oi, _ := opc.Read("testdata/opc.jsonl")
	b := Compare(opc, oi, map[string]int{oi[0].ID: 7, oi[1].ID: 6})
	if b.N != 2 || b.Exact != 1 || !math.IsNaN(b.Quadratic) {
		t.Fatalf("the binary agreement is %+v", b)
	}
	var w bytes.Buffer
	all := Overall([]*Set{set, opc}, map[string][]Item{"gradingbench": items, "opc": oi}, map[string]int{"gradingbench/" + items[0].ID: 7, "gradingbench/" + items[1].ID: 0, "opc/" + oi[0].ID: 7, "opc/" + oi[1].ID: 6})
	if all.N != 4 || all.Exact != 1 || all.Kappa != 1 {
		t.Fatalf("the overall agreement is %+v", all)
	}
	Report(&w, []Agreement{a, b}, &all, grade.DefaultJurors, "a test", "The sample has 4 items.")
	out := w.String()
	if !strings.Contains(out, "| IMO-GradingBench | CC-BY-4.0 | 4 | 100.0% | 1.000 | 1.000 to 1.000 | 1.000 | 100.0% | yes |") || !strings.Contains(out, "CC-BY-NC-SA-4.0") || !strings.Contains(out, "| All sets, pass or fail | | 4 | 100.0% | 1.000 |") || !strings.Contains(out, "The sample has 4 items.") {
		t.Fatalf("the report is\n%s", out)
	}
}
