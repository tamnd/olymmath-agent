package official

import (
	"slices"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
)

func TestOMSemPapers(t *testing.T) {
	page := `<a href="/static/app_main/problems/om69_1.pdf">I</a>
<a href="/static/app_main/problems/om69_1r.pdf">I r</a>
<a href="/static/app_main/problems/om69_3.pdf?v=2">III</a>
<a href="/static/app_main/problems/om69_2.pdf">II</a>
<a href="/static/app_main/problems/om69_3.pdf">III again</a>
<a href="/static/app_main/problems/om47_3.pdf">III</a>`
	got := OMSemPapers(page)
	want := []National{
		{"pl-om", 1996, "", "https://om.sem.edu.pl/static/app_main/problems/om47_3.pdf", Lang{}},
		{"pl-om", 2018, "", "https://om.sem.edu.pl/static/app_main/problems/om69_3.pdf", Lang{}},
		{"pl-om", 2018, "stage1", "https://om.sem.edu.pl/static/app_main/problems/om69_1.pdf", Lang{}},
		{"pl-om", 2018, "stage2", "https://om.sem.edu.pl/static/app_main/problems/om69_2.pdf", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestBWMPapers(t *testing.T) {
	const dir = "/fileadmin/Mathe-Wettbewerbe/Bundeswettbewerb_Mathematik/Dokumente/"
	page := `<a href="` + dir + `Aufgaben_und_Loesungen/aufgaben_10_2.pdf">a</a>
<a href="` + dir + `BWM_Aufgabenblatt_2026_1.pdf">b</a>
<a href="` + dir + `24_2_Aufgabenblatt_Web_240823.pdf">c</a>
<a href="` + dir + `BWM_2023.1_Aufgabenblatt.pdf">d</a>
<a href="` + dir + `BWM_2025_Aufgabenblatt_SCREEN.pdf">e</a>
<a href="` + dir + `loesungen_10_2.pdf">f</a>
<a href="` + dir + `aufgaben_10_2_neu.pdf">g</a>`
	got := BWMPapers(page)
	want := []struct {
		year  int
		paper string
		file  string
	}{
		{2010, "round2", "Aufgaben_und_Loesungen/aufgaben_10_2.pdf"},
		{2023, "round1", "BWM_2023.1_Aufgabenblatt.pdf"},
		{2024, "round2", "24_2_Aufgabenblatt_Web_240823.pdf"},
		{2026, "round1", "BWM_Aufgabenblatt_2026_1.pdf"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v", got)
	}
	for i, w := range want {
		if got[i].Year != w.year || got[i].Paper != w.paper || got[i].URL != "https://www.mathe-wettbewerbe.de"+dir+w.file || got[i].Competition != "de-bundeswettbewerb" {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], w)
		}
	}
}

func TestSplitNational(t *testing.T) {
	segs := Split("Aufgabe 1\nMan bestimme alle n.\nAufgabe 2: Es sei ABC ein Dreieck.\n")
	if len(segs) != 2 || segs[0].Position != "p1" || segs[0].Text != "Man bestimme alle n." || segs[1].Position != "p2" {
		t.Errorf("Aufgabe: %+v", segs)
	}
	segs = Split("Zadanie 3. Wyznaczyć wszystkie liczby x.\n")
	if len(segs) != 1 || segs[0].Position != "p3" {
		t.Errorf("Zadanie: %+v", segs)
	}
}

func TestFormulas(t *testing.T) {
	got := PaperFormulas("W trójkącie ABC punkt D leży na boku AB i 007 a x", "pl")
	for _, k := range []string{"ABC", "D", "AB", "7", "x"} {
		if !got[k] {
			t.Errorf("paper tokens: no %q in %v", k, got)
		}
	}
	for _, k := range []string{"i", "a", "trójkącie"} {
		if got[k] {
			t.Errorf("paper tokens: %q in %v", k, got)
		}
	}
	tex := TeXFormulas(`In a triangle $ABC$ the point $D$ is on $AB$, and $\frac{x}{7}$ is 2 times a.`)
	for _, k := range []string{"ABC", "D", "AB", "x", "7", "2"} {
		if !tex[k] {
			t.Errorf("TeX tokens: no %q in %v", k, tex)
		}
	}
	for _, k := range []string{"a", "frac", "In"} {
		if tex[k] {
			t.Errorf("TeX tokens: %q in %v", k, tex)
		}
	}
	if s := FormulaSignal("$ABC$ and $D$", "ABC i D", "pl"); s != 1 {
		t.Errorf("signal %v, want 1", s)
	}
	if s := FormulaSignal("no formula", "ABC", "pl"); s != 0 {
		t.Errorf("signal %v, want 0", s)
	}
}

func TestLocateNational(t *testing.T) {
	ps := Papers{"pl-om/2018": {
		{Position: "p1", Lang: "pl", Text: "Liczby a, b, c spełniają a + b = 17 oraz c = 2018."},
		{Position: "p2", Lang: "pl", Text: "W trójkącie ABC punkt M jest środkiem boku BC."},
		{Position: "", Lang: "pl", Text: "Wyznaczyć wszystkie n, dla których n + 5 dzieli 99."},
	}}
	if got := ps.Locate("pl-om/2018", "Let $ABC$ be a triangle and $M$ the midpoint of $BC$."); got != "p2" {
		t.Errorf("final problem: got %q, want p2", got)
	}
	if got := ps.Locate("pl-om/2018", "Find all $n$ such that $n+5$ divides $99$."); got != "" {
		t.Errorf("earlier stage: got %q, want no position", got)
	}
}

func TestInPaper(t *testing.T) {
	shape := content.Shape{Layout: []content.Layout{{Paper: "round1", Positions: []string{"p1", "p2"}}, {Paper: "round2", Positions: []string{"p3", "p4"}}}}
	for _, c := range []struct{ id, pos, want string }{{"round2", "p2", "p4"}, {"round1", "p1", "p1"}, {"round2", "p3", ""}, {"stage1", "p1", ""}, {"round1", "x", ""}} {
		if got := inPaper(shape, c.id, c.pos); got != c.want {
			t.Errorf("inPaper(%s, %s) = %q, want %q", c.id, c.pos, got, c.want)
		}
	}
}

func TestNumberedThrough(t *testing.T) {
	shape := content.Shape{Layout: []content.Layout{{Paper: "day1", Positions: []string{"p1", "p2"}}, {Paper: "day2", Positions: []string{"p3", "p4"}}}}
	segs := func(pos ...string) []Segment {
		var out []Segment
		for _, p := range pos {
			out = append(out, Segment{Position: p})
		}
		return out
	}
	for _, c := range []struct {
		id   string
		segs []Segment
		want bool
	}{
		{"day2", segs("p3", "p4"), true},
		{"day2", segs("p1", "p2"), false},
		{"day2", segs("p3", "p5"), false},
		{"day1", segs("p1", "p2"), false},
		{"day3", segs("p3"), false},
		{"day2", nil, false},
	} {
		if got := numberedThrough(shape, c.id, c.segs); got != c.want {
			t.Errorf("numberedThrough(%s, %v) = %v, want %v", c.id, c.segs, got, c.want)
		}
	}
}

func TestAnchorEdition(t *testing.T) {
	prob := func(at, text string) *content.Problem {
		return &content.Problem{Appearances: []content.Appearance{{At: at}}, Canonical: content.Canonical{Statement: "S1"},
			Statements: []content.Statement{{ID: "S1", Source: "aops", Text: text}}}
	}
	owners := map[string]*content.Problem{
		"pl-om/2018/p1": prob("pl-om/2018/p1", "Find all $n$ such that $n+5$ divides $99$."),
		"pl-om/2018/p2": prob("pl-om/2018/p2", "Let $ABC$ be a triangle and $M$ the midpoint of $BC$."),
	}
	ps := Papers{"pl-om/2018": {
		{Position: "p1", Lang: "pl", URL: "u", Text: "Liczby a, b, c spełniają a + b = 17 oraz c = 2018."},
		{Position: "p2", Lang: "pl", URL: "u", Page: 1, Text: "W trójkącie ABC punkt M jest środkiem boku BC."},
		{Position: "p2", Lang: "pl", URL: "u", Text: "W trójkącie ABC punkt M jest środkiem boku BC."},
		{Position: "p3", Lang: "pl", URL: "u", Text: "Udowodnić, że 3 > 2."},
		{Position: "", Lang: "pl", URL: "s", Text: "Wyznaczyć wszystkie n, dla których n + 5 dzieli 99."},
	}}
	r := AnchorEdition(owners, "om-sem", "pl-om/2018", ps)
	if len(r.Links) != 1 || r.Links[0].Problem != owners["pl-om/2018/p2"] || r.Links[0].Segment.Page != 1 {
		t.Errorf("links: %+v", r.Links)
	}
	if len(r.Items) != 1 || r.Items[0].At != "pl-om/2018/p1" || r.Items[0].Ref != "u" {
		t.Errorf("items: %+v", r.Items)
	}
	if len(r.Missing) != 1 || r.Missing[0] != "pl-om/2018/p3" {
		t.Errorf("missing: %v", r.Missing)
	}
}

func TestUKMTPapers(t *testing.T) {
	page := `<a href="/home/bmo1-2015.pdf">BMO1</a>
<a href="https://bmos.ukmt.org.uk/home/bmo2-2015.pdf">BMO2</a>
<a href="/home/bmo-1975.pdf">BMO</a>
<a href="/home/fist-1975.pdf">FIST</a>
<a href="/home/bmo1-2015.pdf">again</a>
<a href="/home/bmo1-2015-solutions.pdf">solutions</a>`
	got := UKMTPapers(page)
	want := []National{
		{"uk-bmo1", 1975, "", "https://bmos.ukmt.org.uk/home/bmo-1975.pdf", Lang{}},
		{"uk-bmo1", 2015, "", "https://bmos.ukmt.org.uk/home/bmo1-2015.pdf", Lang{}},
		{"uk-bmo2", 1975, "", "https://bmos.ukmt.org.uk/home/fist-1975.pdf", Lang{}},
		{"uk-bmo2", 2015, "", "https://bmos.ukmt.org.uk/home/bmo2-2015.pdf", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestCover(t *testing.T) {
	r := Readings["ukmt"]
	cover := "Instructions\n1. Time allowed: 3½ hours.\n2. Full written solutions.\nDo not turn over until told to do so.\n"
	got := r.cover([]string{cover, "1. Find all primes.\n", ""})
	if len(got) != 3 || got[0] != "" || got[1] != "1. Find all primes.\n" {
		t.Errorf("two pages: got %q", got)
	}
	got = r.cover([]string{cover + "1. Find all primes.\n"})
	if len(got) != 1 || strings.TrimSpace(got[0]) != "1. Find all primes." {
		t.Errorf("one page: got %q", got)
	}
	got = r.cover([]string{"1. Find all primes.\n", "2. Prove it.\n"})
	if got[0] != "1. Find all primes.\n" {
		t.Errorf("no cover: got %q", got)
	}
}

func TestInfoSize(t *testing.T) {
	out := []byte("Pages:          1\nPage size:      595.276 x 841.89 pts (A4)\nPage rot:       270\n")
	if w, h, ok := infoSize(out); !ok || w != 841 || h != 595 {
		t.Errorf("rot 270: got %d x %d %v", w, h, ok)
	}
	out = []byte("Page size:      595.276 x 841.89 pts (A4)\nPage rot:       0\n")
	if w, h, ok := infoSize(out); !ok || w != 595 || h != 841 {
		t.Errorf("rot 0: got %d x %d %v", w, h, ok)
	}
	if _, _, ok := infoSize([]byte("Pages: 1\n")); ok {
		t.Errorf("no size: ok")
	}
}

func TestHBCSEPapers(t *testing.T) {
	page := `<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/inmo-2025.pdf">INMO 2025</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/inmosol-25.pdf">INMO 2025 Solutions</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/rmo-2004.pdf">RMO 2004</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/crmo-13-2.pdf">CRMO 2013 Paper 2</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/crmo-15-5.pdf">CRMO 2015 Paper 5</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/rmo-2018-kl-tn.pdf">RMO 2018 (Kerala and Tamil Nadu)</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/rmo-2018.pdf">RMO 2018</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/rmo-2023-kv.pdf">RMO 2023 KV and JNV</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/rmo-2023-non-kv.pdf">RMO 2023 Non KV</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/prmo-2019.pdf">PRMO 2019</a>
<a href="https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/ioqm-2024.pdf">IOQM 2024</a>
<a href="/olympiads/wp-content/uploads/2016/09/inmo-2008.pdf">INMO 2008</a>
<a href="http://158.144.44.253/rmo-sol-2003.pdf">RMO 2003</a>`
	got := HBCSEPapers(page)
	const u = "https://olympiads.hbcse.tifr.res.in/wp-content/uploads/2026/10/"
	want := []National{
		{"in-inmo", 2025, "", u + "inmo-2025.pdf", Lang{}},
		{"in-rmo", 2004, "", u + "rmo-2004.pdf", Lang{}},
		{"in-rmo", 2013, "paper2", u + "crmo-13-2.pdf", Lang{}},
		{"in-rmo", 2015, "mumbai", u + "crmo-15-5.pdf", Lang{}},
		{"in-rmo", 2018, "paper1", u + "rmo-2018.pdf", Lang{}},
		{"in-rmo", 2018, "paper2", u + "rmo-2018-kl-tn.pdf", Lang{}},
		{"in-rmo", 2023, "paper1", u + "rmo-2023-non-kv.pdf", Lang{}},
		{"in-rmo", 2023, "paper2", u + "rmo-2023-kv.pdf", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestIMOJPPapers(t *testing.T) {
	page := `<a href="../archive/mo2026/jmo/problems/jmo36yqa.pdf"> 予選 </a>
<a href="../archive/mo2026/jmo/problems/jmo36hq.pdf">本選</a>
<a href="../archive/mo2026/selection_camp/problems/2026.pdf">代表選考合宿</a>
<a href="../archive/mo2016/jmo2016/problems/jmo26mq.html">本選</a>
<a href="../archive/mo2016/jmo2016/problems/jmo26mq.html">本選</a>
<a href="../archive/mo2009/jjmo2009/problems/jjmo7mq.html">本選</a>
<a href="../archive/mo2003/jjmo2003/problems/jjmo1q.html">問題</a>`
	got := IMOJPPapers(page)
	const u = "https://www.imojp.org/archive/"
	want := []National{
		{"jp-jjmo", 2009, "", u + "mo2009/jjmo2009/problems/jjmo7mq.html", Lang{}},
		{"jp-jmo", 2016, "", u + "mo2016/jmo2016/problems/jmo26mq.html", Lang{}},
		{"jp-jmo", 2026, "", u + "mo2026/jmo/problems/jmo36hq.pdf", Lang{}},
		{"jp-jmo", 2026, "prelim", u + "mo2026/jmo/problems/jmo36yqa.pdf", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestJapaneseFormulas(t *testing.T) {
	got := PaperFormulas("1. nを2以上の整数とする. 三角形 ABC の外心を O とする. ａ＋ｂ＝１０", "ja")
	for _, k := range []string{"n", "2", "ABC", "O", "a", "b", "10"} {
		if !got[k] {
			t.Errorf("no token %q in %v", k, got)
		}
	}
	for k := range got {
		if japanese([]rune(k)[0]) {
			t.Errorf("a Japanese word is a token: %q", k)
		}
	}
}

func TestReadNotPDF(t *testing.T) {
	if _, err := Read(t.Context(), "imojp", []byte(`<html><img src="./jmo26mq1.jpg"></html>`)); err != ErrNoText {
		t.Errorf("an HTML page of scans: got %v, want ErrNoText", err)
	}
	if hasLetters("1. 2. 3.") || !hasLetters(strings.Repeat("問題", 60)) {
		t.Errorf("hasLetters")
	}
}

func TestSplitPageBreak(t *testing.T) {
	segs := Split("1. a first problem\n2. a second problem\n\f3. a problem at the top of page 2\n")
	if len(segs) != 3 || segs[2].Page != 2 {
		t.Fatalf("got %+v, want p3 on page 2", segs)
	}
}

func TestAMTPapers(t *testing.T) {
	page := `<a href="https://amt.edu.au/wp-content/uploads/2020/05/AMO-2020-paper-and-solutions.pdf">here</a>
<a href="https://amt.edu.au/wp-content/uploads/2019/04/2017-AIMO-paper-solutions-v2019.pdf">here</a>
<a href="https://www.amt.edu.au/wp-content/uploads/2019/02/2016-AMO-Paper-and-Solutions.pdf">here</a>
<a href="https://amt.edu.au/wp-content/uploads/2019/02/2016-AMO-Paper-and-Solutions.pdf">here</a>
<a href="https://amt.edu.au/wp-content/uploads/2019/05/AMC-practice-problems-solutions-Set1-SEN.pdf">here</a>`
	got := AMTPapers(page)
	const u = "https://amt.edu.au/wp-content/uploads/"
	want := []National{
		{"au-amo", 2016, "", u + "2019/02/2016-AMO-Paper-and-Solutions.pdf", Lang{}},
		{"au-amo", 2020, "", u + "2020/05/AMO-2020-paper-and-solutions.pdf", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestVTRMCPapers(t *testing.T) {
	page := `<p><a href="http://intranet.math.vt.edu/vtrmc/exams/42nd%20VTRMC.pdf">2022</a></p>
<p><a href="http://intranet.math.vt.edu/vtrmc/exams/33th%20VTRMC.pdf">2011</a></p>
<p><a href="http://intranet.math.vt.edu/vtrmc/exams/32nd%20VTRMC.pdf">2010</a></p>
<p><a href="http://intranet.math.vt.edu/vtrmc/exams/32nd%20VTRMC.pdf">2009</a></p>
<p><a href="http://intranet.math.vt.edu/vtrmc/exams/1st%20VTRMC.pdf">1979&nbsp;</a></p>
<p><a href="http://intranet.math.vt.edu/vtrmc/solutions/33th%20VTRMC,%202011,%20Solutions.pdf">2011</a></p>`
	got := VTRMCPapers(page)
	const u = "https://intranet.math.vt.edu/vtrmc/exams/"
	want := []National{
		{"vtrmc", 1979, "", u + "1st%20VTRMC.pdf", Lang{}},
		{"vtrmc", 2009, "", u + "31st%20VTRMC.pdf", Lang{}},
		{"vtrmc", 2010, "", u + "32nd%20VTRMC.pdf", Lang{}},
		{"vtrmc", 2011, "", u + "33rd%20VTRMC.pdf", Lang{}},
		{"vtrmc", 2022, "", u + "42nd%20VTRMC.pdf", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	for n, w := range map[int]string{1: "1st", 2: "2nd", 3: "3rd", 4: "4th", 11: "11th", 12: "12th", 13: "13th", 21: "21st", 33: "33rd", 42: "42nd"} {
		if g := ordinal(n); g != w {
			t.Errorf("ordinal(%d) = %s, want %s", n, g, w)
		}
	}
}

func TestIMCPapers(t *testing.T) {
	page := `<a href="/archive/problems/1994/IMC_1994_Problems_and_Solutions.pdf">Open PDF</a>
<a href="/archive/problems/2012/IMC_2012_Day_1_Problems_and_Solutions.pdf">Open PDF</a>
<a href="/archive/problems/2012/IMC_2012_Day_1_Problems.pdf">Open PDF</a>
<a href="/archive/problems/2016/IMC_2016_Day_2_Problems.pdf">Open PDF</a>
<a href="/archive/problems/2016/IMC_2016_Day_2_Solutions.pdf">Open PDF</a>
<a href="/archive/problems/2016/IMC_2016_Day_2_Questions.pdf">Open PDF</a>
<a href="/archive/problems/2025/IMC_2025_Day_1_Solutions.pdf">Open PDF</a>`
	got := IMCPapers(page)
	const u = "https://www.imc-math.org.uk/archive/problems/"
	want := []National{
		{"imc", 1994, "", u + "1994/IMC_1994_Problems_and_Solutions.pdf", Lang{}},
		{"imc", 2012, "day1", u + "2012/IMC_2012_Day_1_Problems.pdf", Lang{}},
		{"imc", 2016, "day2", u + "2016/IMC_2016_Day_2_Questions.pdf", Lang{}},
		{"imc", 2025, "day1", u + "2025/IMC_2025_Day_1_Solutions.pdf", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
	if l := IMCListings(1995); len(l) != 2 || l[1] != "https://www.imc-math.org.uk/?year=1995&item=problems" {
		t.Errorf("listings: %v", l)
	}
}

const pamoPage = `<main id="main">
<section class="paper" id="pamo-2026" aria-label="PAMO 2026">
<h2 class="paper-title">PAMO 2026 Problems</h2>
<div class="paper-day"><h3>Day 1 <span>30 June 2026</span></h3>
<ol class="problem-list" start="1">
<li class="problem" data-n="1"><p>Let $ABC$ be a triangle.<br>Show that <span class="nobr">$P(n)&gt;0$.</span></p><p>Find all $n$.</p></li>
<li class="problem" data-n="2"><p>Find all primes $p$.</p></li>
</ol></div>
</section>
<section class="paper" id="pamo-2011" aria-label="PAMO 2011">
<h2 class="paper-title">PAMO 2011 Problems</h2>
</section>
<section class="paper" id="pamo-2000" aria-label="PAMO 2000">
<li class="problem" data-n="1"><p>Solve $x^2 = 1$.</p></li>
</section>
</main>`

func TestPAMOPapers(t *testing.T) {
	got := PAMOPapers(pamoPage)
	want := []National{
		{"pamo", 2000, "", "https://pamoofficial.org/problems/#pamo-2000", Lang{}},
		{"pamo", 2026, "", "https://pamoofficial.org/problems/#pamo-2026", Lang{}},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("paper %d: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestPAMOText(t *testing.T) {
	text := PAMOText(pamoPage, "pamo-2026")
	want := "Problem 1. Let $ABC$ be a triangle.\nShow that $P(n)>0$.\nFind all $n$.\n\nProblem 2. Find all primes $p$.\n\n"
	if text != want {
		t.Errorf("got %q, want %q", text, want)
	}
	segs := Split(text)
	if len(segs) != 2 || segs[0].Position != "p1" || segs[1].Position != "p2" || !strings.Contains(segs[1].Text, "primes") {
		t.Errorf("segments: %+v", segs)
	}
	if PAMOText(pamoPage, "pamo-2011") != "" || PAMOText(pamoPage, "pamo-1999") != "" {
		t.Error("a section with no problem gives text")
	}
}

func TestTuymaadaPapers(t *testing.T) {
	page := `<li><a href="http://tuymaada.lensky-kray.ru/wp-content/uploads/2026/07/T26-booklet-day2-eng.pdf" class="mtli_attachment mtli_pdf">Mathematics (second day) 2026</a></li>
<li><a href="http://tuymaada.lensky-kray.ru/wp-content/uploads/2026/07/T26-booklet-day1-eng.pdf" class="mtli_attachment mtli_pdf">Mathematics (first day) 2026</a></li>
<li><a href="http://tuymaada.lensky-kray.ru/wp-content/uploads/2026/07/T26-physics.pdf">Physics (first day) 2026</a></li>
<li><a href="http://tuymaada.lensky-kray.ru/wp-content/uploads/2017/07/Matematika.pdf">Математика 2017</a></li>`
	got := TuymaadaPapers(page)
	d1 := "https://tuymaada.lensky-kray.ru/wp-content/uploads/2026/07/T26-booklet-day1-eng.pdf"
	d2 := "https://tuymaada.lensky-kray.ru/wp-content/uploads/2026/07/T26-booklet-day2-eng.pdf"
	want := []National{
		{Competition: "tuymaada", Year: 2026, Paper: "senior", URL: d1},
		{Competition: "tuymaada", Year: 2026, Paper: "junior", URL: d1},
		{Competition: "tuymaada", Year: 2026, Paper: "senior", URL: d2},
		{Competition: "tuymaada", Year: 2026, Paper: "junior", URL: d2},
	}
	if !slices.Equal(got, want) {
		t.Errorf("TuymaadaPapers = %v, want %v", got, want)
	}
}

func TestTuymaadaPart(t *testing.T) {
	text := "XXXII Tuymaada\nSecond day\nSenior league\n5. Prove that 1. is not 2.\n6. Find all n.\n7. A square.\n8. A graph.\nJunior League\n5. Prove it.\n6. See problem 6 of Senior league.\n7. A board.\n8. A set.\nSOLUTIONS\n5. Solution.\n"
	var got []string
	for _, s := range Split(TuymaadaPart(text, "senior")) {
		got = append(got, s.Position)
	}
	if want := []string{"p5", "p6", "p7", "p8"}; !slices.Equal(got, want) {
		t.Errorf("senior positions = %v, want %v", got, want)
	}
	j := TuymaadaPart(text, "junior")
	if strings.Contains(j, "Solution") || !strings.Contains(j, "A set.") {
		t.Errorf("junior part = %q", j)
	}
	if TuymaadaPart("no leagues", "senior") != "" {
		t.Error("a text with no league gives a part")
	}
}

func TestGeorgMohrPapers(t *testing.T) {
	page := `<table class="table-gen" style="text-align:center"
paths="gm$$opg.pdf gm$$pb.pdf gm$$loes.pdf gm$$retvejl.pdf"
  from="1991 2000 1991 2008" to="2026"
>`
	got := GeorgMohrPapers(page)
	if len(got) != 36 {
		t.Fatalf("got %d papers, want 36", len(got))
	}
	first, en, last := got[0], got[9], got[35]
	if first.Year != 1991 || first.URL != GeorgMohrListing+"gm91opg.pdf" || first.Lang != Danish {
		t.Errorf("first = %+v", first)
	}
	if en.Year != 2000 || en.URL != GeorgMohrListing+"gm00pb.pdf" || en.Lang.Code != "" {
		t.Errorf("2000 = %+v", en)
	}
	if last.Year != 2026 || last.URL != GeorgMohrListing+"gm26pb.pdf" || last.Competition != "dk-georgmohr" {
		t.Errorf("last = %+v", last)
	}
	if GeorgMohrPapers("<table>") != nil {
		t.Error("a page with no table gives papers")
	}
}

func TestSplitOpgave(t *testing.T) {
	segs := Split("Opgaver · 1991\n\nOpgave 1. Beskriv mængden.\n\nOpgave 2. Bevis at x > 0.\n")
	if len(segs) != 2 || segs[1].Position != "p2" || segs[1].Text != "Bevis at x > 0." {
		t.Errorf("segments = %+v", segs)
	}
}

func TestOPMPapers(t *testing.T) {
	page := `<a href="/Downloads/Ficheiros/Provas/XXXI%20OPM(2012-13)/XXXI_f1b.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XXXI%20OPM(2012-13)/XXXI_f1bs.pdf" target="_blank">Soluções</a>
<a href="/Downloads/Ficheiros/Provas/XXXI%20OPM(2012-13)/XXXI_f2b.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XXXI%20OPM(2012-13)/XXXI_f1a.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XXXI%20OPM(2012-13)/XXXI_1b.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XXXIX%20OPM(2020-21)/XXXIX_fb.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XXVI%20OPM%20(2007-08)/XXVI_f12bps.pdf" target="_blank">Enunciado e Soluções</a>
<a href="/Downloads/Ficheiros/Provas/XXIV%20OPM%20(2005-06)/XXIV_fdia2b.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XXI%20OPM%20(2002-03)/catBfinaldia1.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XXI%20OPM%20(2002-03)/catAfinaldia1.pdf" target="_blank">Enunciado</a>
<a href="/Downloads/Ficheiros/Provas/XII%20OPM%20(1993-94)/XII_fb2.pdf" target="_blank">Final Nacional (Dia 2)</a>
<a href="/Downloads/Ficheiros/Provas/XII%20OPM%20(1993-94)/XII_fb2s.pdf" target="_blank">Soluções.</a>
<a href="/Downloads/Ficheiros/Provas/XLIV OPM(2025-26)/mopm2526prova3ano.pdf" target="_blank">Mini</a>`
	const u = "https://olimpiadas.spm.pt/Downloads/Ficheiros/Provas/"
	want := []National{
		{"pt-opm", 1994, "day2", u + "XII%20OPM%20(1993-94)/XII_fb2.pdf", Lang{}},
		{"pt-opm", 2003, "day1", u + "XXI%20OPM%20(2002-03)/catBfinaldia1.pdf", Lang{}},
		{"pt-opm", 2006, "day2", u + "XXIV%20OPM%20(2005-06)/XXIV_fdia2b.pdf", Lang{}},
		{"pt-opm", 2008, "", u + "XXVI%20OPM%20(2007-08)/XXVI_f12bps.pdf", Lang{}},
		{"pt-opm", 2013, "day1", u + "XXXI%20OPM(2012-13)/XXXI_f1b.pdf", Lang{}},
		{"pt-opm", 2013, "day2", u + "XXXI%20OPM(2012-13)/XXXI_f2b.pdf", Lang{}},
		{"pt-opm", 2021, "", u + "XXXIX%20OPM(2020-21)/XXXIX_fb.pdf", Lang{}},
	}
	if got := OPMPapers(page); !slices.Equal(got, want) {
		t.Errorf("OPMPapers =\n%v\nwant\n%v", got, want)
	}
}

func TestOPMPart(t *testing.T) {
	text := "XXXI OPM - Final - 2o Dia\n\n4. Qual é o menor n?\n\n5. A Liliana quer:\n 1. azul;\n 2. preto.\n\n6. Em cada lado.\n"
	segs := Split(OPMPart(text, "day2"))
	if len(segs) != 3 || segs[0].Position != "p4" || segs[2].Position != "p6" || segs[1].Text != "A Liliana quer: 1. azul; 2. preto." {
		t.Errorf("segments = %+v", segs)
	}
}

func TestMOPapers(t *testing.T) {
	page := `<a href="/media/1/a57i.pdf">A</a><a href="/media/2/a57iii.pdf">A</a><a href="/media/3/a57iiiv.pdf">list</a>` +
		`<a href="/media/4/a73iii.pdf">A</a><a href="/media/5/a73iv.pdf">A</a><a href="/media/6/b73iv.pdf">B</a>` +
		`<a href="/media/7/a68iii-web.pdf">A</a><a href="/media/8/a68list-cz.pdf">list</a>`
	got := MOPapers(page)
	want := []National{
		{Competition: "cz-sk-mo", Year: 2008, URL: "https://www.matematickaolympiada.cz/media/2/a57iii.pdf"},
		{Competition: "cz-sk-mo", Year: 2019, URL: "https://www.matematickaolympiada.cz/media/7/a68iii-web.pdf"},
		{Competition: "cz-sk-mo", Year: 2024, URL: "https://www.matematickaolympiada.cz/media/5/a73iv.pdf"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("MOPapers = %+v", got)
	}
	if l := MOListings(1996); len(l) != 2 || !strings.HasSuffix(l[1], "/44-rocnik") {
		t.Errorf("MOListings = %v", l)
	}
}

func TestMOPart(t *testing.T) {
	text := "\f50. ročník matematické olympiády, III. kolo kategorie A\n\n\f 1. Určete P. (P. Calábek)\nŘešení. Je-li\n     2. jinak.\n 2. V rovině je dán bod. (J. Šimša)\nŘešení. Označme\n     1. Každé slovo.\n 3. Najděte čísla.\n"
	segs := Split(MOPart(text, ""))
	if len(segs) != 3 || segs[0].Text != "Určete P. (P. Calábek)" || segs[1].Text != "V rovině je dán bod. (J. Šimša)" || segs[2].Position != "p3" {
		t.Errorf("segments = %+v", segs)
	}
}

func TestELMOPapers(t *testing.T) {
	page := `<a href="https://web.evanchen.cc/exams/ELMO-2019.pdf">P</a><a href="https://web.evanchen.cc/exams/ELMO-2019-SL.pdf">SL</a>` +
		`<a href="https://web.evanchen.cc/exams/ELMO-2010.pdf">P</a><a href="https://web.evanchen.cc/exams/ELMO-2010-sols.pdf">S</a>` +
		`<a href="https://web.evanchen.cc/exams/ELSMO-2019.pdf">E</a><a href="https://web.evanchen.cc/exams/ELMO-blank-2024.pdf">B</a>` +
		`<a href="https://services.artofproblemsolving.com/download.php?id=x&rn=RUxNT18yMDE1X0ZpbmFsLnBkZg==">2015</a><a href="https://web.evanchen.cc/exams/ELMO-2019.pdf">P</a>`
	got := ELMOPapers(page)
	want := []National{
		{Competition: "elmo", Year: 2010, URL: "https://web.evanchen.cc/exams/ELMO-2010.pdf"},
		{Competition: "elmo", Year: 2019, URL: "https://web.evanchen.cc/exams/ELMO-2019.pdf"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("ELMOPapers = %+v", got)
	}
}

func TestDMSPapers(t *testing.T) {
	u := "https://dms.rs/wp-content/uploads/"
	page := `<a href="` + u + `2016/12/zadaci_smo_2008-1.pdf">Z</a><a href="` + u + `2016/12/resenja_smo_2008-1.pdf">R</a>` +
		`<a href="` + u + `2016/12/zadaci-smo-2014.pdf">Z</a><a href="` + u + `2016/12/rezultati-smo-2014.pdf">R</a>` +
		`<a href="` + u + `2016/12/zadaci_i_resenja_smo_2016.pdf">Z</a><a href="` + u + `2017/04/2017_smo.pdf">Z</a><a href="` + u + `2017/04/2017_smo_resenja.pdf">R</a>` +
		`<a href="` + u + `2019/04/SMO2019.pdf">Z</a><a href="` + u + `2022/04/SMO_2022_ZADACI.pdf">Z</a><a href="` + u + `2022/04/SMO_2022_RESENJA.pdf">R</a>` +
		`<a href="` + u + `2023/04/SMO_2023_SVE.pdf">Z</a><a href="` + u + `2023/04/SMO_2023_RESENJA_SVE.pdf">R</a><a href="` + u + `2021/05/2021_smo_rezultati.pdf">R</a>`
	got := DMSPapers(page)
	want := []National{
		{Competition: "rs-smo", Year: 2008, URL: u + "2016/12/zadaci_smo_2008-1.pdf"},
		{Competition: "rs-smo", Year: 2014, URL: u + "2016/12/zadaci-smo-2014.pdf"},
		{Competition: "rs-smo", Year: 2016, URL: u + "2016/12/zadaci_i_resenja_smo_2016.pdf"},
		{Competition: "rs-smo", Year: 2017, URL: u + "2017/04/2017_smo.pdf"},
		{Competition: "rs-smo", Year: 2019, URL: u + "2019/04/SMO2019.pdf"},
		{Competition: "rs-smo", Year: 2022, URL: u + "2022/04/SMO_2022_ZADACI.pdf"},
		{Competition: "rs-smo", Year: 2023, URL: u + "2023/04/SMO_2023_SVE.pdf"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("DMSPapers = %+v", got)
	}
}

func TestAWBPapers(t *testing.T) {
	d := "/fileadmin/Mathe-Wettbewerbe/AIMO/Aufgaben_und_Loesungen_AIMO/"
	page := `<a href="` + d + `2021_IMO_Auswahlklausuren_HP.pdf">A</a><a href="` + d + `2021_IMO_Auswahlklausuren_Lsg_HP.pdf">L</a>` +
		`<a href="` + d + `aufgaben_awb_00.pdf">A</a><a href="` + d + `aufgaben_awb_13.pdf">A</a><a href="` + d + `loes_awkl_13.pdf">L</a>` +
		`<a href="` + d + `loes_awkl1_02.pdf">L</a><a href="` + d + `aufgaben_awb_13.pdf">A</a>` +
		`<a href="/fileadmin/Mathe-Wettbewerbe/Bundeswettbewerb_Mathematik/Dokumente/Aufgaben_und_Loesungen_BWM/aufgaben_00_1.pdf">B</a>`
	u := "https://www.mathe-wettbewerbe.de" + d
	want := []National{
		{Competition: "de-tst", Year: 2000, URL: u + "aufgaben_awb_00.pdf"},
		{Competition: "de-tst", Year: 2013, URL: u + "aufgaben_awb_13.pdf"},
		{Competition: "de-tst", Year: 2021, URL: u + "2021_IMO_Auswahlklausuren_HP.pdf"},
	}
	if got := AWBPapers(page); !slices.Equal(got, want) {
		t.Errorf("AWBPapers = %+v", got)
	}
}

func TestOKTVPapers(t *testing.T) {
	d := "/pub_bin/dload/kozoktatas/tanulmanyi_versenyek/oktv/"
	u := "https://www.oktatas.hu" + d
	var page string
	for _, f := range []string{"oktv2018_2019_donto/mat2_flap_d_oktv_1819", "oktv2018_2019_donto/mat2_javut_d_oktv_1819", "oktv2018_2019_1ford/mat2_flap1f_oktv_1819", "oktv2018_2019_1ford/mat2_flap1f_oktv_1819",
		"oktv2014_2015donto/mat1_flap_d_oktv_1415", "oktv_2010_2011/mat3_flap1f_oktv1011", "oktv_2009_2010/oktv0910_mat1_d_flut", "oktv_2005_2006/oktv0506_mat3_d_fl_ut",
		"oktv_2004_2005/matematika3_fela", "oktv_2004_2005/matematika3_mego", "oktv_2004_2005/mat2_2foktv0405_fl", "oktv_2004_2005/mat2_2foktv0405_ut", "oktv2017_2018_1ford/mat3_flap1f_0708"} {
		page += `<a href="` + d + f + `.pdf">x</a>`
	}
	got := OKTVPapers(page)
	want := []National{
		{Competition: "hu-oktv", Year: 2005, Paper: "cat2-round2", URL: u + "oktv_2004_2005/mat2_2foktv0405_fl.pdf"},
		{Competition: "hu-oktv", Year: 2005, Paper: "cat3-round1", URL: u + "oktv_2004_2005/matematika3_fela.pdf"},
		{Competition: "hu-oktv", Year: 2006, Paper: "cat3-final", URL: u + "oktv_2005_2006/oktv0506_mat3_d_fl_ut.pdf"},
		{Competition: "hu-oktv", Year: 2010, Paper: "cat1-final", URL: u + "oktv_2009_2010/oktv0910_mat1_d_flut.pdf"},
		{Competition: "hu-oktv", Year: 2011, Paper: "cat3-round1", URL: u + "oktv_2010_2011/mat3_flap1f_oktv1011.pdf"},
		{Competition: "hu-oktv", Year: 2015, Paper: "cat1-final", URL: u + "oktv2014_2015donto/mat1_flap_d_oktv_1415.pdf"},
		{Competition: "hu-oktv", Year: 2018, Paper: "cat3-round1", URL: u + "oktv2017_2018_1ford/mat3_flap1f_0708.pdf"},
		{Competition: "hu-oktv", Year: 2019, Paper: "cat2-final", URL: u + "oktv2018_2019_donto/mat2_flap_d_oktv_1819.pdf"},
		{Competition: "hu-oktv", Year: 2019, Paper: "cat2-round1", URL: u + "oktv2018_2019_1ford/mat2_flap1f_oktv_1819.pdf"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("OKTVPapers = %+v", got)
	}
}

func TestSSMRPapers(t *testing.T) {
	var page string
	for _, u := range []string{"http://ssmr.ro/files/onm2013/faza_nationala/subiecte/subiecte_10.pdf", "http://ssmr.ro/files/onm2013/faza_nationala/subiecte/subiecte_7_HU.pdf", "http://ssmr.ro/files/onm2013/faza_nationala/subiecte/subiecte_7.pdf",
		"http://ssmr.ro/files/onm2013/faza_nationala/subiecte/solutii_8.pdf", "http://ssmr.ro/files/onm2013/faza_nationala/subiecte/subiecte_juniori.pdf", "http://ssmr.ro/files/onm2013/faza_nationala/subiecte/subiecte_5.pdf",
		"http://ssmr.ro/files/onm2013/faza_judet/subiecte_9.pdf", "http://onm2015.ssmr.ro/files/onm2015/subiecte/12_2015_solutii.pdf", "http://onm2015.ssmr.ro/files/onm2015/subiecte/06_2015_solutii.pdf",
		"http://ssmr.ro/files/onm2016/subiecte/ONM_2016_Clasa_VIII.pdf", "http://ssmr.ro/files/onm2016/subiecte/ONM_2016_Clasa_V.pdf", "http://ssmr.ro/files/onm2016/subiecte/ONM_2016_seniori.pdf", "http://ssmr.ro/files/onm2016/subiecte/ONM_2016_Clasa_VIII.pdf",
		"https://ssmr.ro/files/onm2021/etapa_3/subiecte/Subiecte_cl_9.pdf", "https://ssmr.ro/files/onm2021/etapa_3/subiecte/Barem_solutii_9.pdf", "https://ssmr.ro/files/onm2021/etapa_2/cl_9_enunturi.pdf"} {
		page += `<a href="` + u + `">x</a>`
	}
	got := SSMRPapers(page)
	want := []National{
		{Competition: "ro-nmo", Year: 2013, Paper: "grade7", URL: "https://ssmr.ro/files/onm2013/faza_nationala/subiecte/subiecte_7.pdf"},
		{Competition: "ro-nmo", Year: 2013, Paper: "grade10", URL: "https://ssmr.ro/files/onm2013/faza_nationala/subiecte/subiecte_10.pdf"},
		{Competition: "ro-nmo", Year: 2015, Paper: "grade12", URL: "https://onm2015.ssmr.ro/files/onm2015/subiecte/12_2015_solutii.pdf"},
		{Competition: "ro-nmo", Year: 2016, Paper: "grade8", URL: "https://ssmr.ro/files/onm2016/subiecte/ONM_2016_Clasa_VIII.pdf"},
		{Competition: "ro-nmo", Year: 2021, Paper: "grade9", URL: "https://ssmr.ro/files/onm2021/etapa_3/subiecte/Subiecte_cl_9.pdf"},
	}
	if !slices.Equal(got, want) {
		t.Errorf("SSMRPapers = %+v", got)
	}
}
