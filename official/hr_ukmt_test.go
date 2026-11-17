package official

import (
	"regexp"
	"testing"
)

// hrChoiceFixture is a paper of two pages in the fonts of the UKMT: its first page holds numbered instructions, and its second page a multiple choice question whose options stand after wide gaps and whose = pdftohtml sets lower than its line.
const hrChoiceFixture = `<?xml version="1.0" encoding="UTF-8"?>
<pdf2xml>
<page number="1" position="absolute" top="0" left="0" height="1263" width="892">
	<fontspec id="0" size="18" family="ABCDEF+TeXGyreTermesX" color="#000000"/>
	<fontspec id="1" size="18" family="ABCDEF+NewTXMI" color="#000000"/>
	<fontspec id="2" size="18" family="ABCDEF+txmiaX" color="#000000"/>
	<fontspec id="3" size="18" family="ABCDEF+txsys" color="#000000"/>
<text top="100" left="52" width="400" height="18" font="0">1. Do not open the paper until told to do so.</text>
<text top="130" left="52" width="400" height="18" font="0">2. Time allowed: 60 minutes.</text>
</page>
<page number="2" position="absolute" top="0" left="0" height="1263" width="892">
<text top="70" left="52" width="15" height="18" font="0">1.</text>
<text top="70" left="75" width="156" height="18" font="0">Given that</text>
<text top="70" left="236" width="10" height="18" font="1">𝑥</text>
<text top="76" left="250" width="12" height="17" font="2">=</text>
<text top="70" left="266" width="9" height="18" font="0">2</text>
<text top="70" left="275" width="150" height="18" font="0">, what is the value of</text>
<text top="70" left="430" width="10" height="18" font="1">𝑥</text>
<text top="70" left="444" width="12" height="18" font="3">+</text>
<text top="70" left="460" width="9" height="18" font="0">1</text>
<text top="70" left="469" width="9" height="18" font="0">?</text>
<text top="102" left="92" width="33" height="18" font="0">A 1</text>
<text top="102" left="251" width="31" height="18" font="0">B 2</text>
<text top="102" left="408" width="31" height="18" font="0">C 3</text>
<text top="102" left="564" width="33" height="18" font="0">D 4</text>
<text top="102" left="723" width="33" height="18" font="0">E 5</text>
</page>
</pdf2xml>
`

func TestHrPDFChoices(t *testing.T) {
	rules := HrRules{
		Start:     regexp.MustCompile(`^((\d{1,2}))\.(?:\s|$)`),
		First:     true,
		FirstPage: 2,
		Letters:   "–",
		Math:      regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:NewTXMI|txmiaX|txsys)$`),
		Text:      regexp.MustCompile(`^(?:[A-Z]{6}\+)?TeXGyreTermesX$`),
		Gap:       regexp.MustCompile(`^[B-E](?:\s|$)`),
		Part:      regexp.MustCompile(`^A \S.* B \S.* C \S`),
		Shift:     regexp.MustCompile(`^(?:[A-Z]{6}\+)?txmiaX$`),
	}
	ps := HrPDF(ParsePDFXML([]byte(hrChoiceFixture)), nil, rules)
	if len(ps) != 1 {
		t.Fatalf("HrPDF gives %d problems, want 1: %+v", len(ps), ps)
	}
	want := "Given that $x = 2$, what is the value of $x + 1$?\n\nA 1 B 2 C 3 D 4 E 5"
	if p := ps[0]; p.Code != "1" || p.Page != 2 || len(p.Flags) > 0 || p.Text != want {
		t.Errorf("problem = %s page %d flags %v text:\n got %q\nwant %q", p.Code, p.Page, p.Flags, p.Text, want)
	}
	// Without the rules of the paper the instructions are problems, the options a table and TeX Gyre an unknown font.
	plain := HrRules{Start: rules.Start, First: true}
	ps = HrPDF(ParsePDFXML([]byte(hrChoiceFixture)), nil, plain)
	if len(ps) == 0 || ps[0].Page != 1 {
		t.Errorf("without FirstPage the first problem is %+v, want the instructions", ps)
	}
}
