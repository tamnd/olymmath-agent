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
