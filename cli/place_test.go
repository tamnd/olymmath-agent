package cli

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

func TestPlaceFront(t *testing.T) {
	in := `---
tag: DF41DMESEF
kind: problem
canonical:
  statement: S1
  rule: derived
statements:
  - id: S1
    lang: en
    source: hf-ai-mo-aops
    origin: D
    zone: Z2
    kind: transcription
status:
  statement: captured
unplaced:
  - at: nz-nzmo/2019
    source: hf-ai-mo-aops
    label: 2019 New Zealand MO
  - at: nz-nzmo
    source: hf-mathnet
    label: New Zealand NZMO Round Two
---
Show that for all positive integers $k$, there exists a positive integer $n$ such that $n2^k - 7$ is a perfect square.
`
	got, err := placeFront([]byte(in), "nz-nzmo/2019/p12")
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if !strings.Contains(s, "kind: problem\nappearances:\n  - at: nz-nzmo/2019/p12\n    primary: true\n") {
		t.Errorf("no appearance after kind:\n%s", s)
	}
	if strings.Contains(s, "unplaced") {
		t.Errorf("unplaced entries left:\n%s", s)
	}
	if !strings.HasSuffix(s, "---\nShow that for all positive integers $k$, there exists a positive integer $n$ such that $n2^k - 7$ is a perfect square.\n") {
		t.Errorf("body changed:\n%s", s)
	}
	if _, err := placeFront(got, "nz-nzmo/2019/p13"); err == nil {
		t.Errorf("a placed problem placed again")
	}
}

func TestRelocateFront(t *testing.T) {
	in := `---
tag: 5NVGC04WHV
kind: problem
appearances:
  - at: nz-nzmo/2020/p1
    primary: true
status:
  statement: captured
---
Let $P(x) = x^{3} - 2x + 1$.
`
	got, err := relocateFront([]byte(in), "nz-nzmo/2020/p1", "nz-nzmo/2020/p9")
	if err != nil {
		t.Fatal(err)
	}
	if want := strings.Replace(in, "2020/p1\n", "2020/p9\n", 1); string(got) != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if _, err := relocateFront([]byte(in), "nz-nzmo/2020/p2", "nz-nzmo/2020/p9"); err == nil {
		t.Errorf("no error for a missing appearance")
	}
}

func TestRelocatedOwners(t *testing.T) {
	a, b, c := &content.Problem{Tag: "A"}, &content.Problem{Tag: "B"}, &content.Problem{Tag: "C"}
	owners := map[string]*content.Problem{"x/2020/p1": a, "x/2020/p9": b, "x/2020/p2": c}
	// A and B swap their positions, and C stays.
	got := relocatedOwners(owners, []placeRelocation{{p: a, from: "x/2020/p1", at: "x/2020/p9"}, {p: b, from: "x/2020/p9", at: "x/2020/p1"}})
	if got["x/2020/p1"] != b || got["x/2020/p9"] != a || got["x/2020/p2"] != c || len(got) != 3 {
		t.Errorf("got %v", got)
	}
	if owners["x/2020/p1"] != a {
		t.Errorf("the owners changed")
	}
	got = relocatedOwners(owners, []placeRelocation{{p: c, from: "x/2020/p2", at: "x/2020/p13"}})
	if _, ok := got["x/2020/p2"]; ok || got["x/2020/p13"] != c {
		t.Errorf("move: got %v", got)
	}
}
