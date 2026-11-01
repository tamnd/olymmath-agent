package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/canon"
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
