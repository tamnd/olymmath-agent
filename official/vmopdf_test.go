package official

import (
	"slices"
	"testing"
)

// vmoXML is a small output of pdftohtml -xml: a sheet of grade 9 and a sheet of grade 10 on one page in portrait.
const vmoXML = `<?xml version="1.0" encoding="UTF-8"?>
<pdf2xml producer="poppler" version="26.10.0">
<page number="1" position="absolute" top="0" left="0" height="892" width="631">
	<fontspec id="0" size="18" family="Times" color="#000000"/>
	<fontspec id="1" size="18" family="WVPAVI+CMMI12" color="#000000"/>
	<fontspec id="2" size="18" family="AOZYFN+CMR12" color="#000000"/>
	<fontspec id="3" size="12" family="KDIXXB+CMR8" color="#000000"/>
<text top="106" left="278" width="60" height="0" font="0">9 класс</text>
<text top="166" left="54" width="27" height="0" font="0">9.1.</text>
<text top="166" left="87" width="130" height="0" font="0">Докажите, что</text>
<text top="153" left="227" width="9" height="18" font="1">x</text>
<text top="150" left="237" width="6" height="12" font="3">2</text>
<text top="153" left="250" width="12" height="18" font="2">+</text>
<text top="153" left="268" width="9" height="18" font="2">1</text>
<text top="166" left="285" width="120" height="0" font="0">больше нуля.</text>
<text top="196" left="54" width="27" height="0" font="0">9.2.</text>
<text top="196" left="87" width="70" height="0" font="0">Найдите</text>
<text top="176" left="170" width="6" height="12" font="3">1</text>
<text top="190" left="170" width="6" height="12" font="3">2</text>
<text top="196" left="185" width="40" height="0" font="0">числа.</text>
<text top="256" left="278" width="60" height="0" font="0">10 класс</text>
<text top="286" left="54" width="36" height="0" font="0">10.5.</text>
<text top="286" left="96" width="70" height="0" font="0">Пусть</text>
<text top="273" left="176" width="9" height="18" font="1">n</text>
<text top="286" left="190" width="200" height="0" font="0">натуральное число.</text>
</page>
</pdf2xml>
`

func TestRuPDFXML(t *testing.T) {
	ps := RuPDFXML([]byte(vmoXML), nil, "2014", "https://example.org/s.pdf")
	want := []struct {
		pos, text string
		layout    bool
	}{
		{"p91", "Докажите, что $x^2 + 1$ больше нуля.", false},
		// Two small pieces one above the other with no letter before them are a fraction, which goes to review.
		{"p92", "Найдите ${}_2^1$ числа.", true},
		{"p105", "Пусть $n$ натуральное число.", false},
	}
	if len(ps) != len(want) {
		t.Fatalf("got %d problems, want %d: %+v", len(ps), len(want), ps)
	}
	for i, w := range want {
		p := ps[i]
		if p.Position != w.pos || p.Text != w.text || slices.Contains(p.Flags, "layout") != w.layout {
			t.Errorf("problem %d: got %s %q %v, want %s %q layout=%v", i, p.Position, p.Text, p.Flags, w.pos, w.text, w.layout)
		}
		if p.Page != 1 || p.Tool != RuPDFTool || p.Edition != "2014" || p.Figure {
			t.Errorf("problem %d: page %d tool %q edition %q figure %v", i, p.Page, p.Tool, p.Edition, p.Figure)
		}
	}
	if got := RuExtraction(ps[0]); got != "native/pdftohtml ("+RuVersion+")" {
		t.Errorf("extraction %q", got)
	}
	if got := RuExtraction(RuProblem{Page: 2}); got != "native/pdftotext ("+RuVersion+")" {
		t.Errorf("extraction with no tool %q", got)
	}
}

func TestRuPDFXMLLandscape(t *testing.T) {
	// A page in landscape holds two copies of a sheet side by side; only the left half is read.
	xml := `<page number="1" position="absolute" top="0" left="0" height="600" width="1200">
	<fontspec id="0" size="18" family="Times" color="#000000"/>
<text top="100" left="50" width="27" height="0" font="0">9.1.</text>
<text top="100" left="90" width="200" height="0" font="0">Левая половина.</text>
<text top="100" left="650" width="27" height="0" font="0">9.1.</text>
<text top="100" left="690" width="200" height="0" font="0">Правая половина.</text>
</page>`
	ps := RuPDFXML([]byte(xml), nil, "2013", "u")
	if len(ps) != 1 || ps[0].Text != "Левая половина." {
		t.Fatalf("got %+v", ps)
	}
}
