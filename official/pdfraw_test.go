package official

import (
	"bytes"
	"compress/zlib"
	"strings"
	"testing"
)

// rawPDF makes a small PDF with a Type 3 font whose ToUnicode table maps the code of ф, 0xf4, to a control code, so that the text layer loses it. The content is compressed when deflate is true.
func rawPDF(content string, deflate bool) []byte {
	stream, filter := []byte(content), ""
	if deflate {
		var b bytes.Buffer
		w := zlib.NewWriter(&b)
		_, _ = w.Write(stream)
		_ = w.Close()
		stream, filter = b.Bytes(), " /Filter /FlateDecode"
	}
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n")
	b.WriteString("1 0 obj\n<< /Type /Font /Subtype /Type3 /CharProcs 5 0 R /ToUnicode 2 0 R >>\nendobj\n")
	b.WriteString("2 0 obj\n<< >>\nstream\nbeginbfchar\n<f4> <000C>\nendbfchar\nbeginbfrange\n<61> <7a> <0061>\nendbfrange\nendstream\nendobj\n")
	b.WriteString("3 0 obj\n<< /Type /Page /Font << /F1 1 0 R >> /Contents 4 0 R >>\nendobj\n")
	b.WriteString("4 0 obj\n<<" + filter + " >>\nstream\n")
	b.Write(stream)
	b.WriteString("\nendstream\nendobj\n")
	b.WriteString("5 0 obj\n<< >>\nendobj\n")
	return b.Bytes()
}

func textLines(texts ...[]string) []pdfLine {
	var out []pdfLine
	for _, l := range texts {
		var fs []pdfFrag
		for _, t := range l {
			fs = append(fs, pdfFrag{text: t, kind: 't'})
		}
		out = append(out, pdfLine{frags: fs})
	}
	return out
}

func lineTexts(lines []pdfLine) string {
	var b strings.Builder
	for _, l := range lines {
		for i, f := range l.frags {
			if i > 0 {
				b.WriteString("|")
			}
			b.WriteString(f.text)
		}
		b.WriteString("\n")
	}
	return b.String()
}

func TestPdfLostLetter(t *testing.T) {
	for _, deflate := range []bool{false, true} {
		set := pdfLost(rawPDF(`BT /F1 10 Tf (abc\364def) Tj ET`, deflate))
		if set == nil {
			t.Fatalf("deflate %v: no lost letters", deflate)
		}
		s := set.signs[pdfLostKey{"abc", "def"}]
		if s == nil || s.text != "ф" || !s.letter || s.n != 1 {
			t.Fatalf("deflate %v: got %+v", deflate, set.signs)
		}
		lines := textLines([]string{"abcdef"})
		if missed := pdfRepair(lines, set); len(missed) != 0 {
			t.Errorf("missed %v", missed)
		}
		if got := lineTexts(lines); got != "abcфdef\n" {
			t.Errorf("got %q", got)
		}
	}
}

func TestPdfLostDash(t *testing.T) {
	// The code 0x16 is an em dash in T2A, and the ToUnicode table does not map it.
	set := pdfLost(rawPDF(`BT /F1 10 Tf [(word)-300(\026)-300(next)] TJ ET`, false))
	lines := textLines([]string{"word", "next"})
	if missed := pdfRepair(lines, set); len(missed) != 0 {
		t.Errorf("missed %v", missed)
	}
	if got := lineTexts(lines); got != "word|—|next\n" {
		t.Errorf("got %q", got)
	}
}

func TestPdfLostAmbiguous(t *testing.T) {
	// The text layer has the context of the lost letter twice, but the letter is lost once: no place is sure, so the letter is missed.
	set := pdfLost(rawPDF(`BT /F1 10 Tf (abc\364def) Tj ET`, false))
	lines := textLines([]string{"abcdef"}, []string{"abcdef"})
	missed := pdfRepair(lines, set)
	if len(missed) != 1 || missed[0] != (pdfLostKey{"abc", "def"}) {
		t.Errorf("missed %v", missed)
	}
	if got := lineTexts(lines); got != "abcdef\nabcdef\n" {
		t.Errorf("got %q", got)
	}
}

func TestPdfLostNone(t *testing.T) {
	// A PDF with no Type 3 font loses nothing.
	pdf := bytes.ReplaceAll(rawPDF(`BT /F1 10 Tf (abc) Tj ET`, false), []byte("/CharProcs"), []byte("/Widths"))
	if set := pdfLost(pdf); set != nil {
		t.Errorf("got %+v", set)
	}
	if missed := pdfRepair(textLines([]string{"abc"}), nil); missed != nil {
		t.Errorf("missed %v", missed)
	}
	// A Type 3 font that loses no letter on the page gives nil too.
	if set := pdfLost(rawPDF(`BT /F1 10 Tf (abc) Tj ET`, false)); set != nil {
		t.Errorf("got %+v", set)
	}
}

func TestPdfLiteral(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{`(a(b)c)`, "a(b)c"},
		{`(a\)b)`, "a)b"},
		{`(\101\n)`, "A\n"},
		{"(a\\\nb)", "ab"},
	} {
		got, n := pdfLiteral([]byte(c.in))
		if string(got) != c.want || n != len(c.in) {
			t.Errorf("%q: got %q %d", c.in, got, n)
		}
	}
}

func TestPdfContentRunes(t *testing.T) {
	lost := func(name string) map[byte]bool {
		if name == "F1" {
			return map[byte]bool{0xf4: true}
		}
		return nil
	}
	// The text of F2 is not a Type 3 font and reads as Latin-1; a hex string and an inline image are read too.
	rs := pdfContentRunes([]byte("BT /F2 9 Tf (x) Tj /F1 10 Tf <61f4> Tj ET BI /W 1 ID \x00\x01 EI"), lost)
	var got []string
	for _, r := range rs {
		s := string(r.r)
		if r.lost {
			s += "!"
		}
		got = append(got, s)
	}
	if strings.Join(got, " ") != "x a ф!" {
		t.Errorf("got %v", got)
	}
}
