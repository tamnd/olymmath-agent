package canon

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/match"
)

const geo = "Let $ABC$ be an acute triangle with circumcentre $O$. Prove that the line through $O$ parallel to $BC$ meets the circle $AOB$ again at a point on the bisector of angle $A$."

func cand(at, src, origin, text, answer string) *Candidate {
	return &Candidate{At: at, Source: src, Origin: origin, Lang: "en", Text: text, Answer: answer, Key: match.StrongKey, Ref: src + "#1"}
}

func TestChooseHumanFirst(t *testing.T) {
	cs := []*Candidate{
		cand("imo/2006/p1", "hf-d", "D", geo, ""),
		cand("imo/2006/p1", "aops-wiki", "H3", geo, ""),
		cand("imo/2006/p1", "imo-compendium", "H2", geo, ""),
		cand("imo/2006/p1", "model", "M", geo, ""),
	}
	ch, ok := Choose("imo/2006/p1", cs, "en")
	if !ok {
		t.Fatal("no choice")
	}
	if ch.Best.Source != "imo-compendium" || ch.Rule != "curated" || ch.Status() != "sourced" {
		t.Fatalf("best %s rule %s status %s", ch.Best.Source, ch.Rule, ch.Status())
	}
	if len(ch.Kept) != 2 {
		t.Fatalf("kept %d, want 2", len(ch.Kept))
	}
}

func TestChooseDispute(t *testing.T) {
	cs := []*Candidate{
		cand("aime/2021a/p3", "aops-wiki", "H3", geo, "12"),
		cand("aime/2021a/p3", "hf-d", "D", geo, "13"),
	}
	ch, _ := Choose("aime/2021a/p3", cs, "en")
	if !ch.Disputed || ch.Status() != "disputed" || ch.Answer != "" {
		t.Fatalf("disputed %v status %s answer %q", ch.Disputed, ch.Status(), ch.Answer)
	}
}

func TestPlanPatternMerge(t *testing.T) {
	a, _ := Choose("imo/2006/p1", []*Candidate{cand("imo/2006/p1", "aops-wiki", "H3", geo, "2")}, "en")
	b, _ := Choose("isl/2006/g1", []*Candidate{cand("isl/2006/g1", "aops-wiki", "H3", geo+" Show also that the point is unique and find the ratio.", "2")}, "en")
	if tt := match.Jaccard(a.Best.Shingles(), b.Best.Shingles()); tt >= match.Same || tt < match.Doubt {
		t.Fatalf("T = %.3f, the test needs a value between Doubt and Same", tt)
	}
	recs, _ := Plan([]Choice{b, a})
	if len(recs) != 1 || recs[0].Primary.At != "imo/2006/p1" || len(recs[0].Merged) != 1 || recs[0].Merged[0].Rule != "isl-imo" {
		t.Fatalf("records %+v", recs)
	}
}

func TestPlanSameEdition(t *testing.T) {
	var cs []Choice
	for _, at := range []string{"imo/1977/p2", "isl/1977/p3", "isl/1977/p10"} {
		ch, _ := Choose(at, []*Candidate{cand(at, "aops-wiki", "H3", geo, "")}, "en")
		cs = append(cs, ch)
	}
	recs, items := Plan(cs)
	if len(recs) != 2 {
		t.Fatalf("%d records, want 2", len(recs))
	}
	if len(items) == 0 || items[0].Kind != "pair" {
		t.Fatalf("items %+v", items)
	}
}

func TestGate(t *testing.T) {
	d := []Doubt{{}}
	cs := []Choice{{At: "hmmt/2020/p1", Doubts: d}, {At: "hmmt/2020/p2"}, {At: "imo/2020/p1"}}
	out, gated := Gate(cs, MaxDoubt)
	if len(out) != 1 || len(gated) != 1 || gated[0].Comp != "hmmt" {
		t.Fatalf("out %d gated %+v", len(out), gated)
	}
}

func TestRender(t *testing.T) {
	ch, _ := Choose("imo/2006/p1", []*Candidate{cand("imo/2006/p1", "imo-compendium", "H2", geo, ""), cand("imo/2006/p1", "aops-wiki", "H3", geo, "")}, "en")
	b, err := Render(&Record{Primary: ch}, "a1b2", "")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"tag: a1b2", "rule: curated", "statement: sourced", "source: aops-wiki", geo} {
		if !strings.Contains(s, want) {
			t.Errorf("render has no %q:\n%s", want, s)
		}
	}
	if strings.Count(s, geo) != 1 {
		t.Errorf("the same text is written twice")
	}
}

func TestLetter(t *testing.T) {
	cases := map[string]string{"AIME I": "a", "AIME II": "b", "AMC 10B": "b", "AMC 12A": "a", "USAMO": ""}
	for in, want := range cases {
		if got := Letter([]string{in}); got != want {
			t.Errorf("Letter(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestAnswer(t *testing.T) {
	cases := map[string]string{"proof": "", " Proof. ": "", "not found": "", "$12$": "$12$", "7": "7"}
	for in, want := range cases {
		if got := Answer(in); got != want {
			t.Errorf("Answer(%q) = %q, want %q", in, got, want)
		}
	}
}
