package cli

import (
	"context"
	"io"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/keymatch"
	"github.com/tamnd/olymmath-agent/lane"
	"github.com/tamnd/olymmath-agent/review"
)

func TestYesNo(t *testing.T) {
	for in, want := range map[string][2]bool{"yes": {true, true}, " Yes.\n": {true, true}, "**No**": {false, true}, "no": {false, true}, "maybe": {false, false}, "yes, the same": {false, false}} {
		same, ok := yesNo(in)
		if same != want[0] || ok != want[1] {
			t.Errorf("yesNo(%q) = %v %v, want %v", in, same, ok, want)
		}
	}
}

func TestMatchDecide(t *testing.T) {
	root := t.TempDir()
	pairs := []keymatch.Pair{
		{Ref: "a#1", At: "c/2010/p1", Class: keymatch.ClassStrong, Text: "row 1"},
		{Ref: "a#2", At: "c/2010/p2", Class: keymatch.ClassDoubtful, Text: "row 2"},
		{Ref: "a#3", At: "c/2010/p3", Class: keymatch.ClassDoubtful, Text: "row 3"},
		{Ref: "a#4", At: "c/2010/p4", Class: keymatch.ClassDoubtful, Text: "row 4"},
		{Ref: "a#5", At: "c/2010/p5", Class: keymatch.ClassStrong, Text: "row 5"},
	}
	revs := map[string]*review.Review{
		// A person accepted the doubtful pair 2 and rejected the strong pair 5.
		matchItemID("c/2010/p2", "a#2"): {Decision: "accept"},
		matchItemID("c/2010/p5", "a#5"): {Decision: "reject"},
	}
	var asked []string
	ask := lane.AskerFunc(func(_ context.Context, q lane.Question) (lane.Answer, error) {
		asked = append(asked, q.Prompt)
		if strings.Contains(q.Prompt, "row 3") {
			return lane.Answer{Text: "Yes", Lane: "fake"}, nil
		}
		return lane.Answer{Text: "no", Lane: "fake"}, nil
	})
	keyText := func(at string) string { return "key of " + at }
	placed, n, err := matchDecide(context.Background(), root, pairs, revs, ask, 5, "2026-11-10", keyText, io.Discard)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, p := range placed {
		got = append(got, p.At)
	}
	if strings.Join(got, " ") != "c/2010/p1 c/2010/p2 c/2010/p3" || n != 2 || len(asked) != 2 {
		t.Fatalf("placed %v, asked %d", got, n)
	}
	if !strings.Contains(asked[0], "row 3") || !strings.Contains(asked[0], "key of c/2010/p3") {
		t.Errorf("question %q", asked[0])
	}
	// The answers are reviews of the agent, so a second run asks nothing.
	saved, errs := review.Load(root)
	if len(errs) > 0 || len(saved) != 2 {
		t.Fatalf("reviews %v %v", saved, errs)
	}
	r := saved[matchItemID("c/2010/p4", "a#4")]
	if r == nil || r.Decision != "reject" || r.By != "agent" || r.Kind != "match" || !strings.HasPrefix(r.Note, "Agent decision, not a person's review.") {
		t.Errorf("review %+v", r)
	}
	placed, n, _ = matchDecide(context.Background(), root, pairs, revs, ask, 5, "2026-11-10", keyText, io.Discard)
	if n != 0 || len(placed) != 3 {
		t.Errorf("second run asked %d and placed %d", n, len(placed))
	}
	// With no lane, a doubtful pair with no review is not placed.
	placed, _, _ = matchDecide(context.Background(), root, pairs, map[string]*review.Review{}, nil, 5, "2026-11-10", keyText, io.Discard)
	if len(placed) != 2 {
		t.Errorf("with no lane placed %d", len(placed))
	}
}
