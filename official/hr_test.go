package official

import (
	"regexp"
	"slices"
	"strings"
	"testing"
)

// hrFixture is made like a page of a paper of the county round, Zupanijsko2026-SS.pdf, as pdftohtml -xml -i gives it: a header, a date, a grade whose š the caron of Computer Modern splits, two problems of the grade, the second with its heading and its text in one run and a line that starts like a heading, a footer, and the next grade.
const hrFixture = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE pdf2xml SYSTEM "pdf2xml.dtd">

<pdf2xml producer="poppler" version="24.02.0">
<page number="1" position="absolute" top="0" left="0" height="1262" width="892">
	<fontspec id="0" size="15" family="IDCCAA+LMRoman10" color="#000000"/>
	<fontspec id="5" size="15" family="NBBQIW+LMMathItalic10" color="#000000"/>
<text top="60" left="300" width="300" height="13" font="0">ŽUPANIJSKO NATJECANJE IZ MATEMATIKE</text>
<text top="80" left="300" width="140" height="13" font="0">26. veljače 2026.</text>
<text top="120" left="200" width="132" height="13" font="0">1. razred – srednja ˇ</text>
<text top="120" left="330" width="130" height="13" font="0">skola – A varijanta</text>
<text top="160" left="100" width="14" height="13" font="0">1.</text>
<text top="160" left="125" width="170" height="13" font="0">Odredi sve brojeve izmedu</text>
<text top="160" left="300" width="8" height="13" font="5"><i>a</i></text>
<text top="160" left="313" width="8" height="13" font="0">i</text>
<text top="160" left="326" width="8" height="13" font="5"><i>b</i></text>
<text top="160" left="334" width="4" height="13" font="0">.</text>
<text top="190" left="100" width="160" height="13" font="0">2. Koliko ima brojeva</text>
<text top="190" left="265" width="10" height="13" font="5"><i>n</i></text>
<text top="190" left="280" width="60" height="13" font="0">manjih od</text>
<text top="205" left="100" width="160" height="13" font="0">4. potencije broja 9?</text>
<text top="220" left="400" width="80" height="13" font="0">Okreni list!</text>
<text top="250" left="100" width="230" height="13" font="0">Svaki zadatak vrijedi 10 bodova.</text>
<text top="290" left="200" width="270" height="13" font="0">2. razred – srednja škola – A varijanta</text>
<text top="330" left="100" width="14" height="13" font="0">1.</text>
<text top="330" left="125" width="200" height="13" font="0">Dokaži da je zbroj paran.</text>
</page>
</pdf2xml>
`

// hrTestRules are the rules of the papers of the grades, as the command hr has them.
var hrTestRules = HrRules{
	Start:   regexp.MustCompile(`^(\d{1,2})\.(?:\s|$)`),
	Section: regexp.MustCompile(`^(\d)\.\s*razred\s*[-–]\s*(?:osnovna|srednja)\s+škola`),
	Stop:    regexp.MustCompile(`^(?:Svaki zadatak|ŽUPANIJSKO)`),
	Skip:    regexp.MustCompile(`^Okreni list`),
	First:   true,
}

func TestHrPDF(t *testing.T) {
	ps := HrPDF(ParsePDFXML([]byte(hrFixture)), nil, hrTestRules)
	want := []struct {
		code, section, text string
	}{
		{"1", "1", "Odredi sve brojeve između $a$ i $b$."},
		{"2", "1", "Koliko ima brojeva $n$ manjih od 4. potencije broja 9?"},
		{"1", "2", "Dokaži da je zbroj paran."},
	}
	if len(ps) != len(want) {
		t.Fatalf("HrPDF gives %d problems, want %d: %+v", len(ps), len(want), ps)
	}
	for i, w := range want {
		p := ps[i]
		if p.Code != w.code || p.Section != w.section || p.Page != 1 || p.Figure || len(p.Flags) > 0 {
			t.Errorf("problem %d = %s in %q page %d figure %v flags %v", i, p.Code, p.Section, p.Page, p.Figure, p.Flags)
		}
		if p.Text != w.text {
			t.Errorf("problem %d text:\n got %q\nwant %q", i, p.Text, w.text)
		}
	}
	// Without First the date is the heading of a problem 26.
	rules := hrTestRules
	rules.First = false
	rules.Stop = regexp.MustCompile(`^Svaki zadatak`)
	ps = HrPDF(ParsePDFXML([]byte(hrFixture)), nil, rules)
	if len(ps) == 0 || ps[0].Code != "26" {
		t.Errorf("without First the first problem is %+v, want the date", ps)
	}
}

func TestHrPDFTitled(t *testing.T) {
	items := ParsePDFXML([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<pdf2xml>
<page number="1" position="absolute" top="0" left="0" height="1262" width="892">
	<fontspec id="0" size="15" family="IDCCAA+LMRoman10" color="#000000"/>
<text top="100" left="100" width="14" height="13" font="0">1.</text>
<text top="100" left="125" width="60" height="13" font="0">Aljmez</text>
<text top="120" left="125" width="200" height="13" font="0">Ana ima tri jabuke.</text>
<text top="140" left="125" width="8" height="13" font="0">•</text>
<text top="140" left="140" width="200" height="13" font="0">Koliko ih ima Iva?</text>
</page>
</pdf2xml>
`))
	rules := HrRules{Start: regexp.MustCompile(`^(\d{1,2})\.(?:\s|$)`), Titled: true, First: true}
	ps := HrPDF(items, nil, rules)
	if len(ps) != 1 {
		t.Fatalf("HrPDF gives %d problems", len(ps))
	}
	if !strings.HasPrefix(ps[0].Text, "**Aljmez**\n\nAna ima tri jabuke.") || !strings.Contains(ps[0].Text, "- Koliko ih ima Iva?") || len(ps[0].Flags) > 0 {
		t.Errorf("titled problem %q flags %v", ps[0].Text, ps[0].Flags)
	}
}

func TestHrText(t *testing.T) {
	for in, want := range map[string]string{
		"ˇskola":       "škola",
		"ˇ Cetiri ´ce": "Četiri će",
		"ˇZ i ˇz":      "Ž i ž",
		"ﬁlm ﬂota":     "film flota",
		"ˇa ostaje":    "ˇa ostaje",
	} {
		if got := HrText(in); got != want {
			t.Errorf("HrText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHrJoinAccents(t *testing.T) {
	items := []PDFItem{
		{Page: 1, Top: 100, Left: 100, Width: 50, Text: "osnovna ˇ"},
		{Page: 1, Top: 100, Left: 149, Width: 30, Text: "skola"},
		{Page: 1, Top: 100, Left: 200, Width: 30, Text: "ˇ"},
		{Page: 1, Top: 140, Left: 200, Width: 30, Text: "sljedeći red"},
	}
	got := hrJoinAccents(items)
	if len(got) != 3 || got[0].Text != "osnovna ˇskola" || got[0].Right() != 179 || got[1].Text != "ˇ" {
		t.Errorf("hrJoinAccents gives %+v", got)
	}
}

func TestHrDj(t *testing.T) {
	if got, want := hrDj("Izmedu dva broja medu kojima je odredeno, a ne Medvjed."), "Između dva broja među kojima je određeno, a ne Medvjed."; got != want {
		t.Errorf("hrDj gives %q, want %q", got, want)
	}
}

func TestHrOdd(t *testing.T) {
	for s, want := range map[string]bool{
		"Neka je $ö$ realan broj, a ž, č, ć – „dobar”.": false,
		"Neka je ö realan broj.":                        true,
		"Cijena je 5 £.":                                true,
	} {
		if got := hrOdd(s); got != want {
			t.Errorf("hrOdd(%q) = %v", s, got)
		}
	}
}

func TestHrMath(t *testing.T) {
	if got, want := hrMath("𝑥+𝛼𝜖𝒜"), "x+αϵ𝒜"; got != want {
		t.Errorf("hrMath gives %q, want %q", got, want)
	}
	if got := hrMath("𝛺𝜗"); got != "Ωϑ" {
		t.Errorf("hrMath(𝛺𝜗) = %q", got)
	}
}

func TestHrPDFWord(t *testing.T) {
	items := ParsePDFXML([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<pdf2xml>
<page number="1" position="absolute" top="0" left="0" height="1262" width="892">
	<fontspec id="1" size="18" family="AAAAAE+Cambria" color="#000000"/>
	<fontspec id="11" size="18" family="AAAAAY+Cambria" color="#000000"/>
	<fontspec id="22" size="18" family="AAAAAQ+CambriaMath" color="#000000"/>
<text top="786" left="94" width="23" height="21" font="1"><b>1.  </b></text>
<text top="786" left="148" width="126" height="21" font="1"><b>Palindrom </b></text>
<text top="817" left="94" width="56" height="21" font="1">Broj je </text>
<text top="817" left="151" width="79" height="21" font="11"><i>palindrom</i></text>
<text top="817" left="231" width="60" height="21" font="1"> ako je</text>
<text top="817" left="295" width="34" height="21" font="22">𝐴𝐵𝐶</text>
<text top="817" left="333" width="20" height="21" font="1">.</text>
<text top="850" left="94" width="23" height="21" font="1"><b>2.  </b></text>
<text top="850" left="148" width="126" height="21" font="1"><b>Kutovi </b></text>
<text top="881" left="94" width="100" height="21" font="1">Neka je </text>
<text top="881" left="245" width="24" height="21" font="22">𝐴𝐷</text>
<text top="877" left="245" width="24" height="21" font="22">''''</text>
<text top="881" left="270" width="60" height="21" font="1"> dužina.</text>
</page>
</pdf2xml>
`))
	rules := HrRules{Start: regexp.MustCompile(`^(\d{1,2})\.(?:\s|$)`), Titled: true, First: true}
	ps := HrPDF(items, nil, rules)
	if len(ps) != 2 {
		t.Fatalf("HrPDF gives %d problems", len(ps))
	}
	if want := "**Palindrom**\n\nBroj je palindrom ako je $ABC$."; ps[0].Text != want || len(ps[0].Flags) > 0 {
		t.Errorf("Word text %q flags %v, want %q", ps[0].Text, ps[0].Flags, want)
	}
	// The bar of the segment AD is no math that a line can hold.
	if !slices.Contains(ps[1].Flags, "font") {
		t.Errorf("the bar of a segment gives flags %v, want font", ps[1].Flags)
	}
}

func TestHrLegacy(t *testing.T) {
	for _, c := range []struct{ family, in, want string }{
		{"CJCMDK+CMSY10", "0", "′"},
		{"CJCMDK+CMSY10", "k", "∥"},
		{"USIRKN+MSAM10", "^", "∢"},
		{"TZYURE+MSBM10", "N", "ℕ"},
		{"TZYURE+MSBM10", "B", "𝔹"},
		{"YJCNKO+CMMI12", "k", "k"},
	} {
		if got := hrLegacy(c.family, c.in); got != c.want {
			t.Errorf("hrLegacy(%s, %q) = %q, want %q", c.family, c.in, got, c.want)
		}
	}
}

func TestHrBroken(t *testing.T) {
	for s, want := range map[string]bool{
		"Na tabuli je napísaný zlomok . Určte číslo $q$.":     true,
		"Pre $x$ . platí $y$.":                                false,
		"Riješite jednadžbu $(a - 1)x^2 + ax+=0$.":            true,
		"Zbroj svih pet faktora iznosi $40_$.":                true,
		"Za $x^2$ i $y_{n}$ vrijedi $x -1 = y$.":              false,
		"Postupnosť $(a_n)_n^\\infty_{=0}$ celých čísel":      true,
		"Ak $P(t)=$ = 0, tak $P(t + 1) = 1$.":                 true,
		"Neka je $ABC$ trokut i $a_n = 3a_{n - 1}$.":          false,
		"Izračunaj $$\\sin x + 1,$$ za $x \\in [0,2\\pi]$.":   false,
		"Izračunaj 184 · 15 + 15 · 16 – 15 · 100 + 15?":       false,
		"Odredi vrijednost izraza + . ¶ $$x y$$":              true,
		"Zadane su točke $A ,-1$ i $B ,-1$.":                  true,
		"Djed, otac i \\sin zajedno":                          true,
		"$$--\\to - \\to$$":                                   true,
		"Koliko ima djelitelja broja $30^{()}$?":              true,
		"Koliko ima parova za koje je mn$+2$m$-2$n $=2 020$?": true,
	} {
		if got := hrBroken(s); got != want {
			t.Errorf("hrBroken(%q) = %v, want %v", s, got, want)
		}
	}
}

func TestHrPDFTail(t *testing.T) {
	rules := hrTestRules
	rules.Tail = regexp.MustCompile(`\s*Prvih \S+ zadataka vrijedi`)
	items := ParsePDFXML([]byte(strings.Replace(hrFixture, "Dokaži da je zbroj paran.", "Dokaži da je zbroj paran. Prvih pet zadataka vrijedi po 6 bodova.", 1)))
	ps := HrPDF(items, nil, rules)
	if len(ps) != 3 || ps[2].Text != "Dokaži da je zbroj paran." {
		t.Errorf("the tail stays in %+v", ps)
	}
}

func TestHrPDFDrop(t *testing.T) {
	rules := hrTestRules
	rules.Start = regexp.MustCompile(`^(\d{1,2})\.?(?:\s|$)`)
	rules.Drop = regexp.MustCompile(`^\((?:\p{Lu}[\p{L}.]*\s*)+\)$`)
	rules.Not = regexp.MustCompile(`^\d\s\d{3}\b`)
	rules.Lossy = regexp.MustCompile(`^(?:[A-Z]{6}\+)?Cambria`)
	fixture := strings.Replace(hrFixture, `<text top="160" left="334" width="4" height="13" font="0">.</text>`, `<text top="160" left="334" width="4" height="13" font="0">.</text>
<text top="160" left="700" width="80" height="13" font="0">(P. Černek)</text>
<text top="175" left="100" width="160" height="13" font="0">2 360 je broj.</text>`, 1)
	ps := HrPDF(ParsePDFXML([]byte(fixture)), nil, rules)
	if len(ps) != 3 || ps[0].Text != "Odredi sve brojeve između $a$ i $b$. 2 360 je broj." || slices.Contains(ps[0].Flags, "layout") || slices.Contains(ps[0].Flags, "glyph") {
		t.Errorf("the note stays or the number is a heading: %+v", ps)
	}
	ps = HrPDF(ParsePDFXML([]byte(strings.Replace(fixture, "IDCCAA+LMRoman10", "IDCCAA+Cambria", 1))), nil, rules)
	if len(ps) == 0 || !slices.Contains(ps[0].Flags, "glyph") {
		t.Errorf("a problem of a lossy paper has no flag glyph: %+v", ps)
	}
}

func TestPxMathFonts(t *testing.T) {
	for s, want := range map[string]struct {
		tex string
		ok  bool
	}{
		`R\{0}`:  {`R \setminus \{0\}`, true},
		`%%%%BC`: {"", false},
		`n"`:     {"", false},
	} {
		tex, ok := pxMath(s, false)
		if ok != want.ok || want.ok && tex != want.tex {
			t.Errorf("pxMath(%q) = %q, %v, want %q, %v", s, tex, ok, want.tex, want.ok)
		}
	}
}
