package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/match"
)

func TestWriteExplain(t *testing.T) {
	traces := []rowTrace{
		{code: "imo", year: 2020, number: "1", source: "a", ref: "f#1", at: "imo/2020/p1"},
		{code: "imo", year: 2020, number: "1", source: "b", ref: "f#2", at: "imo/2020/p1"},
		{code: "imo", year: 2020, number: "2", source: "a", ref: "f#3", at: "imo/2020/p2"},
		{code: "hmmt", year: 2020, number: "1", source: "a", ref: "f#4", at: "hmmt/2020/p1"},
		{code: "imo", year: 0, number: "3", source: "a", ref: "f#5", reason: "no valid year"},
	}
	choices := []canon.Choice{{At: "imo/2020/p1", Doubts: []canon.Doubt{{C: &canon.Candidate{Ref: "f#2"}}}}}
	path := filepath.Join(t.TempDir(), "explain.tsv")
	if err := writeExplain(path, traces, choices, []canon.Gated{{Comp: "hmmt"}}); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path) // #nosec G304 -- a test file.
	if err != nil {
		t.Fatal(err)
	}
	var reasons []string
	for _, l := range strings.Split(strings.TrimSpace(string(b)), "\n")[1:] {
		f := strings.Split(l, "\t")
		reasons = append(reasons, f[len(f)-1])
	}
	if got := strings.Join(reasons, ","); got != "ok,doubt,no choice,gated,no valid year" {
		t.Errorf("reasons %s", got)
	}
}

func TestDuplicateOf(t *testing.T) {
	texts := match.NewIndex()
	texts.Add("cn-mo/2010/p1", match.Shingles("Let $m,n\\ge 1$ and $a_1 < a_2 < \\ldots < a_n$ be integers. Prove that there exists a subset $T$ of the natural numbers with a small size."))
	texts.Add("cn-mo/2010/p2", match.Shingles("Find all positive integers $n$ such that $2^n + 1$ is divisible by $n^2$, and prove that there are no others."))
	h, ok := duplicateOf(texts, match.Shingles("Let $m,n\\ge 1$ and $a_1 < a_2 < \\ldots < a_n$ be integers. Prove that there exists a subset $T$ of the natural numbers with a small size."))
	if !ok || h.ID != "cn-mo/2010/p1" || h.T < match.Duplicate {
		t.Fatalf("duplicateOf gives %+v, %v", h, ok)
	}
	if h, ok := duplicateOf(texts, match.Shingles("Prove that every triangle with two equal angles has two equal sides, and find the third angle.")); ok {
		t.Fatalf("a new text gives %+v", h)
	}
}

func TestWriteQueueOnce(t *testing.T) {
	root := t.TempDir()
	items := []canon.Item{
		{ID: "a1", Kind: "pair", At: "hu-komal/2022/a834", With: "nemotron-imo-bench/2026/g17", Text: "new"},
		{ID: "b2", Kind: "pair", At: "cono-sur/1997/p1", With: "cono-sur/1998/p1"},
		{ID: "a1", Kind: "pair", At: "hu-komal/2022/a834", With: "nemotron-imo-bench/2026/g17", Text: "old"},
	}
	if err := writeQueue(root, items); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(filepath.Join(root, "reports", "review", "queue.jsonl")) // #nosec G304 -- a test file.
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 || !strings.Contains(lines[0], `"text":"new"`) {
		t.Errorf("the queue:\n%s", b)
	}
}

func TestWikiAnswer(t *testing.T) {
	for in, want := range map[string]string{
		`\textbf{(B)}\40`:        "B",
		`\textbf{(E)}\`:          "E",
		"D":                      "D",
		"(C)":                    "C",
		"12":                     "12",
		`12\sqrt{3}`:             `12\sqrt{3}`,
		`\frac{85\pi}{8}`:        `\frac{85\pi}{8}`,
		"=3xor=-2x":              "",
		`\frac{md}{+r}`:          "",
		`\frac{}{b}`:             "",
		`\frac{b^2-2ac}{^2}`:     "",
		"notfound":               "",
		"y+3x-4=0":               "",
		`=\frac{15}{2}\implies=`: "",
	} {
		if got := wikiAnswer(in); got != want {
			t.Errorf("wikiAnswer(%q) = %q, want %q", in, got, want)
		}
	}
}
