package cli

import (
	"testing"

	"github.com/tamnd/olymmath-agent/review"
)

func TestSampleResult(t *testing.T) {
	for _, c := range []struct {
		verdict, decision, want string
	}{
		{"exact", "", "match"},
		{"agrees", "", "match"},
		{"check", "", "open"},
		{"check", "accept", "match"},
		{"check", "reject", "differ"},
		{"exact", "reject", "differ"},
	} {
		it := sampleItem{verdict: c.verdict}
		if c.decision != "" {
			it.review = &review.Review{Decision: c.decision}
		}
		if got := sampleResult(it); got != c.want {
			t.Errorf("sampleResult(%s, %q) = %s, want %s", c.verdict, c.decision, got, c.want)
		}
	}
}

func TestSampleRankAndID(t *testing.T) {
	if sampleRank(1, "imo/1985/p4") != sampleRank(1, "imo/1985/p4") {
		t.Error("the rank of a position changes between calls")
	}
	if sampleRank(1, "imo/1985/p4") == sampleRank(2, "imo/1985/p4") {
		t.Error("the rank does not depend on the seed")
	}
	a := SampleID("imo-official", "imo/1985/p4", "text\n")
	if a != SampleID("imo-official", "imo/1985/p4", "text") || len(a) != 12 {
		t.Errorf("SampleID gives %q", a)
	}
	if a == SampleID("imo-official", "imo/1985/p4", "other text") {
		t.Error("SampleID does not depend on the text")
	}
}
