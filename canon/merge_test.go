package canon

import (
	"strings"
	"testing"
)

const mergeInto = `---
tag: AAAAAAAAAA
kind: problem
appearances:
  - at: imo/2019/p1
topics: [A]
canonical:
  statement: S1
  rule: derived
statements:
  - id: S1
    lang: en
    source: hf-a
    origin: D
    zone: Z2
    kind: transcription
status:
  statement: captured
---
Determine all functions $f$ such that $f(2a)+2f(b)=f(f(a+b))$ for all integers $a$ and $b$.
`

const mergeFrom = `---
tag: BBBBBBBBBB
kind: problem
appearances:
  - at: isl/2019/a1
type: answer
answer: f(x)=0
topics: [A, N]
canonical:
  statement: S1
  rule: derived
statements:
  - id: S1
    lang: en
    source: hf-b
    origin: D
    zone: Z2
    kind: transcription
  - id: S2
    lang: en
    source: hf-c
    origin: D
    zone: Z2
    kind: transcription
status:
  statement: captured
---
Let Z be the set of integers. Find each function from the integers to the integers with this property.

<!-- S2 -->
Determine all functions $f$ such that $f(2a)+2f(b)=f(f(a+b))$ for all integers $a$ and $b$.
`

func TestMergeFiles(t *testing.T) {
	b, err := MergeFiles([]byte(mergeInto), []byte(mergeFrom))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"tag: AAAAAAAAAA", "  - at: imo/2019/p1\n    primary: true\n  - at: isl/2019/a1\n", "id: S2\n    lang: en\n    source: hf-b", "id: S3\n    lang: en\n    source: hf-c", "answer: f(x)=0", "type: answer", "<!-- S2 -->\nLet Z be the set"} {
		if !strings.Contains(s, want) {
			t.Errorf("the merged file has no %q:\n%s", want, s)
		}
	}
	// The text of S2 of from is the canonical text, so it stays a link with no text.
	if strings.Contains(s, "<!-- S3 -->") {
		t.Errorf("the same text came in twice:\n%s", s)
	}
	if !strings.Contains(s, "topics: [A, N]") {
		t.Errorf("the topics did not join:\n%s", s)
	}
	other := strings.Replace(mergeInto, "status:", "answer: f(x)=2x\nstatus:", 1)
	if _, err := MergeFiles([]byte(other), []byte(mergeFrom)); err == nil {
		t.Error("two answers that disagree merged")
	}
}

func TestFirst(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"imo/2019/p1", "isl/2019/a1", true},
		{"isl/2019/a1", "imo/2019/p1", false},
		{"jbmo/2016/p2", "jbmo-sl/2016/a2", true},
		{"jbmo-sl/2016/a2", "jbmo/2016/p2", false},
		{"ru-allrussian/1970/p2", "se-skolornas/1995/p4", true},
		{"usamo/2001/p1", "aime/1990/p1", true},
	}
	for _, c := range cases {
		if got := First(c.a, c.b); got != c.want {
			t.Errorf("First(%s, %s) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}
