package canon

import (
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
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

func TestChooseReworded(t *testing.T) {
	kalva := cand("apmo/1989/p1", "kalva", "H3", "$a_i$ are positive reals. s = $a_1$ + ... + $a_n$. Prove that (1 + $a_1$) ... (1 + $a_n$) < 1 + s + $s^2$/2! + ... + $s^n$/n!.", "")
	kalva.Reworded = true
	kalva.Key = match.WeakKey
	d := cand("apmo/1989/p1", "hf-d", "D", "Let $x_1, x_2, \\dots, x_n$ be positive real numbers and let $S = x_1 + \\dots + x_n$. Prove that $(1+x_1)\\cdots(1+x_n) \\le 1 + S + S^2/2! + \\dots + S^n/n!$.", "")
	d.Key = match.WeakKey
	ch, ok := Choose("apmo/1989/p1", []*Candidate{kalva, d}, "en")
	if !ok {
		t.Fatal("no choice")
	}
	if ch.Best.Source != "hf-d" || len(ch.Doubts) != 0 || len(ch.Kept) != 1 || ch.Kept[0].Source != "kalva" {
		t.Fatalf("best %s, %d doubts, kept %v", ch.Best.Source, len(ch.Doubts), ch.Kept)
	}
	only, _ := Choose("apmo/1989/p1", []*Candidate{kalva}, "en")
	if only.Best.Source != "kalva" {
		t.Fatalf("best %s, want kalva when it is the only text", only.Best.Source)
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

func TestPatternLonglist(t *testing.T) {
	for _, c := range []struct{ a, b, want string }{
		{"ill/1967/p29", "imo/1967/p4", "ill-imo"},
		{"isl/1992/p5", "ill/1992/p13", "ill-isl"},
		{"ill/1967/p29", "imo/1968/p4", ""},
		{"ill/1967/p29", "ill/1967/p30", ""},
	} {
		if got := Pattern(c.a, c.b); got != c.want {
			t.Errorf("Pattern(%s, %s) = %q, want %q", c.a, c.b, got, c.want)
		}
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

func TestTrace(t *testing.T) {
	file := "---\ntag: 0123456789\nkind: problem\nappearances:\n  - at: imo/2024/p1\n    primary: true\ncanonical:\n  statement: S1\n  rule: derived\nstatements:\n  - id: S1\n    lang: en\n    source: a\n    origin: D\n    zone: Z2\n    kind: transcription\n  - id: S2\n    lang: en\n    source: b\n    origin: D\n    zone: Z2\n    kind: transcription\n  - id: S3\n    lang: en\n    source: imo-official\n    origin: H1\n    zone: Z2\n    kind: official\nstatus:\n  statement: captured\n---\nText one.\n\n<!-- S2 -->\nText two.\n"
	b, err := Trace([]byte(file), "S2", "S3", "official-translation")
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"statement: S2\n  rule: official-translation", "    kind: transcription\n    traced: S3\n", "statement: sourced", "---\nText two.\n\n<!-- S1 -->\nText one.\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("no %q in\n%s", want, got)
		}
	}
	if _, err := Trace([]byte(file), "S3", "S1", "x"); err == nil {
		t.Error("a statement with no text was traced")
	}
}

func TestConfirm(t *testing.T) {
	p1 := "The incircle of triangle ABC touches BC, CA, AB at K, L, M. The line through A parallel to LK meets MK at P. Show that PQ bisects AB."
	p1b := "The in-circle of triangle ABC touches the sides BC, CA and AB in K, L and M respectively. The line through A and parallel to LK meets MK in P. Prove that the line PQ bisects the side AB."
	p2 := "Find all functions f from the reals to the reals with f(x+y) = f(x) f(y) for all reals x and y."
	best1 := &Candidate{Source: "a", Text: p1}
	best2 := &Candidate{Source: "a", Text: p2}
	para := &Candidate{Source: "b", Text: p1b}
	wrong := &Candidate{Source: "c", Text: p2 + " Prove it."}
	choices := Confirm([]Choice{
		{At: "x/2000/p1", Best: best1, Doubts: []Doubt{{C: para}, {C: wrong}}},
		{At: "x/2000/p2", Best: best2},
	})
	ch := choices[0]
	if len(ch.Kept) != 1 || ch.Kept[0] != para {
		t.Errorf("kept %v, want the paraphrase", ch.Kept)
	}
	if len(ch.Doubts) != 1 || ch.Doubts[0].C != wrong {
		t.Errorf("doubts %v, want the text of p2", ch.Doubts)
	}
}

func TestAddReworded(t *testing.T) {
	p := &content.Problem{Statements: []content.Statement{{ID: "S1", Lang: "en", Source: "hf-d", Origin: "D"}}}
	p.Canonical.Statement = "S1"
	kalva := cand("apmo/1989/p1", "kalva", "H3", geo, "")
	kalva.Reworded = true
	wiki := cand("apmo/1989/p1", "aops-wiki", "H3", geo, "")
	if a := Add(p, &Record{Primary: Choice{Best: kalva}}, "en"); a.Better != nil || len(a.Statements) != 1 {
		t.Fatalf("reworded: better %v, %d statements", a.Better, len(a.Statements))
	}
	if a := Add(p, &Record{Primary: Choice{Best: wiki}}, "en"); a.Better != wiki {
		t.Fatalf("not reworded: better %v", a.Better)
	}
}

func TestRenderLink(t *testing.T) {
	c := &Candidate{Source: "ukmt", Origin: "H1", Zone: "Z2", Lang: "en", Name: "BMO1 2015 official paper", URL: "https://bmos.ukmt.org.uk/home/bmo1-2015.pdf"}
	b, err := RenderLink("uk-bmo1/2015/p3", "a1b2", c, 2, "official-original")
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, want := range []string{"tag: a1b2", "at: uk-bmo1/2015/p3", "primary: true", "rule: official-original", "statement: sourced", "kind: official", "page: 2", "source: ukmt"} {
		if !strings.Contains(s, want) {
			t.Errorf("render has no %q:\n%s", want, s)
		}
	}
	if !strings.HasSuffix(s, "---\n") {
		t.Errorf("the body is not empty:\n%s", s)
	}
}

func TestRenderScan(t *testing.T) {
	c := &Candidate{Source: "om-sem", Origin: "H1", Zone: "Z2", Lang: "pl", Name: "Polish MO 1997 official paper", URL: "https://om.sem.edu.pl/static/app_main/problems/om48_3.pdf"}
	s := Scan{Page: 1, Text: "Udowodnić, że $a^2 \\ge 0$.\n", Extraction: "vision/claude@2.1.318", Confidence: 0.904}
	b, err := RenderScan("pl-om/1997/p2", "a1b2", c, s, "official-original")
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	for _, want := range []string{"statement: captured", "kind: official", "origin: H1", "page: 1", "    extraction: vision/claude@2.1.318\n    confidence: 0.9\n", "---\nUdowodnić, że $a^2 \\ge 0$.\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("render has no %q:\n%s", want, got)
		}
	}
	s.Text = ""
	if b, err = RenderScan("pl-om/1997/p2", "a1b2", c, s, "official-original"); err != nil || strings.Contains(string(b), "extraction") || !strings.Contains(string(b), "statement: sourced") {
		t.Errorf("a scan with no text is not a link:\n%s %v", b, err)
	}
	if _, err := RenderScan("pl-om/1997/p2", "a1b2", c, Scan{Page: 1, Text: "x"}, "official-original"); err == nil {
		t.Error("a text with no extraction gave no error")
	}
}
