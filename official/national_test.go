package official

import (
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
		{"pl-om", 1996, "", "https://om.sem.edu.pl/static/app_main/problems/om47_3.pdf"},
		{"pl-om", 2018, "", "https://om.sem.edu.pl/static/app_main/problems/om69_3.pdf"},
		{"pl-om", 2018, "stage1", "https://om.sem.edu.pl/static/app_main/problems/om69_1.pdf"},
		{"pl-om", 2018, "stage2", "https://om.sem.edu.pl/static/app_main/problems/om69_2.pdf"},
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
		{"uk-bmo1", 1975, "", "https://bmos.ukmt.org.uk/home/bmo-1975.pdf"},
		{"uk-bmo1", 2015, "", "https://bmos.ukmt.org.uk/home/bmo1-2015.pdf"},
		{"uk-bmo2", 1975, "", "https://bmos.ukmt.org.uk/home/fist-1975.pdf"},
		{"uk-bmo2", 2015, "", "https://bmos.ukmt.org.uk/home/bmo2-2015.pdf"},
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
		{"in-inmo", 2025, "", u + "inmo-2025.pdf"},
		{"in-rmo", 2004, "", u + "rmo-2004.pdf"},
		{"in-rmo", 2013, "paper2", u + "crmo-13-2.pdf"},
		{"in-rmo", 2015, "mumbai", u + "crmo-15-5.pdf"},
		{"in-rmo", 2018, "paper1", u + "rmo-2018.pdf"},
		{"in-rmo", 2018, "paper2", u + "rmo-2018-kl-tn.pdf"},
		{"in-rmo", 2023, "paper1", u + "rmo-2023-non-kv.pdf"},
		{"in-rmo", 2023, "paper2", u + "rmo-2023-kv.pdf"},
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
		{"jp-jjmo", 2009, "", u + "mo2009/jjmo2009/problems/jjmo7mq.html"},
		{"jp-jmo", 2016, "", u + "mo2016/jmo2016/problems/jmo26mq.html"},
		{"jp-jmo", 2026, "", u + "mo2026/jmo/problems/jmo36hq.pdf"},
		{"jp-jmo", 2026, "prelim", u + "mo2026/jmo/problems/jmo36yqa.pdf"},
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
		{"au-amo", 2016, "", u + "2019/02/2016-AMO-Paper-and-Solutions.pdf"},
		{"au-amo", 2020, "", u + "2020/05/AMO-2020-paper-and-solutions.pdf"},
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
		{"vtrmc", 1979, "", u + "1st%20VTRMC.pdf"},
		{"vtrmc", 2009, "", u + "31st%20VTRMC.pdf"},
		{"vtrmc", 2010, "", u + "32nd%20VTRMC.pdf"},
		{"vtrmc", 2011, "", u + "33rd%20VTRMC.pdf"},
		{"vtrmc", 2022, "", u + "42nd%20VTRMC.pdf"},
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
		{"imc", 1994, "", u + "1994/IMC_1994_Problems_and_Solutions.pdf"},
		{"imc", 2012, "day1", u + "2012/IMC_2012_Day_1_Problems.pdf"},
		{"imc", 2016, "day2", u + "2016/IMC_2016_Day_2_Questions.pdf"},
		{"imc", 2025, "day1", u + "2025/IMC_2025_Day_1_Solutions.pdf"},
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
		{"pamo", 2000, "", "https://pamoofficial.org/problems/#pamo-2000"},
		{"pamo", 2026, "", "https://pamoofficial.org/problems/#pamo-2026"},
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
