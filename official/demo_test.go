package official

import (
	"slices"
	"testing"
)

// demoFixture is a part of the second page of the paper of the final of grade 10 of the 34th olympiad, 34104_a.pdf, as pdftohtml -xml -i gives it.
const demoFixture = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE pdf2xml SYSTEM "pdf2xml.dtd">

<pdf2xml producer="poppler" version="24.02.0">
<page number="2" position="absolute" top="0" left="0" height="1262" width="892">
	<fontspec id="0" size="15" family="IDCCAA+LMRoman10" color="#000000"/>
	<fontspec id="2" size="22" family="YEJDFL+LMRoman12" color="#000000"/>
	<fontspec id="3" size="18" family="UITWGP+Times" color="#000000"/>
	<fontspec id="4" size="15" family="QNQJWH+LMRoman10" color="#000000"/>
	<fontspec id="5" size="15" family="NBBQIW+LMMathItalic10" color="#000000"/>
	<fontspec id="6" size="10" family="RJRSVG+LMRoman7" color="#000000"/>
	<fontspec id="7" size="15" family="VNRXCH+LMMathSymbols10" color="#000000"/>
	<fontspec id="8" size="10" family="NZDWFD+LMMathItalic7" color="#000000"/>
<text top="101" left="469" width="253" height="13" font="0">http://www.olympiade-mathematik.de</text>
<text top="223" left="165" width="36" height="12" font="3">OJM</text>
<text top="205" left="362" width="253" height="19" font="2">34. Mathematik-Olympiade</text>
<text top="232" left="381" width="215" height="19" font="2">4. Stufe (Bundesrunde)</text>
<text top="259" left="445" width="87" height="19" font="2">Klasse 10</text>
<text top="285" left="444" width="88" height="19" font="2">Aufgaben</text>
<text top="346" left="95" width="54" height="13" font="0">Hinweis:</text>
<text top="346" left="156" width="637" height="13" font="4"><i>Der Lösungsweg mit Begründungen und Nebenrechnungen soll deutlich erkennbar in logisch und</i></text>
<text top="364" left="95" width="697" height="13" font="4"><i>grammatikalisch einwandfreien Sätzen dargestellt werden. Zur Lösungsgewinnung herangezogene Aussagen</i></text>
<text top="382" left="95" width="697" height="13" font="4"><i>sind zu beweisen. Nur wenn eine so zu verwendende Aussage aus dem Schulunterricht oder aus Arbeitsge-</i></text>
<text top="400" left="95" width="642" height="13" font="4"><i>meinschaften bekannt ist, genügt es ohne Beweisangabe, sie als bekannten Sachverhalt anzuführen.</i></text>
<text top="437" left="95" width="108" height="13" font="0">Aufgabe 341041:</text>
<text top="460" left="95" width="157" height="13" font="0">Zeigen Sie, daß die Zahl</text>
<text top="460" left="257" width="7" height="13" font="5"><i>z</i></text>
<text top="460" left="269" width="49" height="13" font="0">= 7 + 7</text>
<text top="458" left="318" width="6" height="9" font="6">3</text>
<text top="460" left="328" width="22" height="13" font="0">+ 7</text>
<text top="458" left="350" width="6" height="9" font="6">5</text>
<text top="460" left="360" width="22" height="13" font="0">+ 7</text>
<text top="458" left="383" width="6" height="9" font="6">7</text>
<text top="460" left="393" width="12" height="13" font="0">+</text>
<text top="460" left="408" width="12" height="13" font="5"><i>...</i></text>
<text top="460" left="424" width="22" height="13" font="0">+ 7</text>
<text top="458" left="446" width="12" height="9" font="6">93</text>
<text top="460" left="462" width="22" height="13" font="0">+ 7</text>
<text top="458" left="484" width="12" height="9" font="6">95</text>
<text top="460" left="502" width="137" height="13" font="0">durch 336 teilbar ist!</text>
<text top="497" left="95" width="174" height="13" font="0">Aufgabe 341042 = 340943:</text>
<text top="518" left="95" width="86" height="13" font="0">Auf der Seite</text>
<text top="518" left="186" width="23" height="13" font="5"><i>AB</i></text>
<text top="518" left="214" width="93" height="13" font="0">des Quadrates</text>
<text top="518" left="312" width="47" height="13" font="5"><i>ABCD</i></text>
<text top="518" left="365" width="107" height="13" font="0">werde ein Punkt</text>
<text top="518" left="477" width="12" height="13" font="5"><i>X</i></text>
<text top="518" left="495" width="0" height="14" font="7">6</text>
<text top="518" left="495" width="12" height="13" font="0">=</text>
<text top="518" left="511" width="11" height="13" font="5"><i>A</i></text>
<text top="518" left="527" width="266" height="13" font="0">gewählt. Dann werde das Quadrat durch</text>
<text top="536" left="95" width="79" height="13" font="0">die Strecken</text>
<text top="536" left="180" width="22" height="13" font="5"><i>AC</i></text>
<text top="536" left="207" width="25" height="13" font="0">und</text>
<text top="536" left="237" width="26" height="13" font="5"><i>XD</i></text>
<text top="536" left="269" width="169" height="13" font="0">in vier Teilflächen zerlegt.</text>
<text top="564" left="95" width="310" height="13" font="0">Ermitteln Sie alle Möglichkeiten, die Wahl von</text>
<text top="564" left="411" width="12" height="13" font="5"><i>X</i></text>
<text top="564" left="430" width="253" height="13" font="0">so zu treffen, daß es natürliche Zahlen</text>
<text top="564" left="689" width="8" height="13" font="5"><i>p</i></text>
<text top="564" left="697" width="4" height="13" font="0">,</text>
<text top="564" left="706" width="7" height="13" font="5"><i>q</i></text>
<text top="564" left="719" width="25" height="13" font="0">und</text>
<text top="564" left="750" width="7" height="13" font="5"><i>r</i></text>
<text top="564" left="763" width="30" height="13" font="0">gibt,</text>
<text top="582" left="95" width="567" height="13" font="0">für die die Flächeninhalte dieser Teilflächen in geeigneter Reihenfolge im Verhältnis 1 :</text>
<text top="582" left="666" width="8" height="13" font="5"><i>p</i></text>
<text top="582" left="678" width="4" height="13" font="0">:</text>
<text top="582" left="686" width="7" height="13" font="5"><i>q</i></text>
<text top="582" left="698" width="4" height="13" font="0">:</text>
<text top="582" left="706" width="7" height="13" font="5"><i>r</i></text>
<text top="582" left="718" width="46" height="13" font="0">stehen!</text>
<text top="619" left="95" width="108" height="13" font="0">Aufgabe 341043:</text>
<text top="640" left="95" width="275" height="13" font="0">Beweisen Sie, daß für jede natürliche Zahl</text>
<text top="1152" left="95" width="401" height="13" font="0">34. Olympiade - Bundesrunde - Klasse 10 - Saison 1994/1995</text>
<text top="1152" left="785" width="7" height="13" font="0">2</text>
</page>
</pdf2xml>
`

func TestDemoFile(t *testing.T) {
	p, ok := DemoFile("34104_a.pdf")
	if !ok || p != (DemoPaper{Olympiad: 34, Grade: 10, Round: 4}) {
		t.Fatalf("DemoFile(34104_a.pdf) = %+v, %v", p, ok)
	}
	if _, ok := DemoFile("34104_l.pdf"); ok {
		t.Error("DemoFile takes a solution file")
	}
	p, n, ok := DemoCode("011213")
	if !ok || n != "3" || p != (DemoPaper{Olympiad: 1, Grade: 12, Round: 1}) {
		t.Errorf("DemoCode(011213) = %+v, %s, %v", p, n, ok)
	}
	p, n, ok = DemoCode("101246B")
	if !ok || n != "6b" || p != (DemoPaper{Olympiad: 10, Grade: 12, Round: 4}) {
		t.Errorf("DemoCode(101246B) = %+v, %s, %v", p, n, ok)
	}
	for _, c := range []string{"10124", "101240", "101246C", "1012461"} {
		if _, _, ok := DemoCode(c); ok {
			t.Errorf("DemoCode(%s) reads", c)
		}
	}
}

func TestDemoPDF(t *testing.T) {
	ps := DemoPDF(ParsePDFXML([]byte(demoFixture)), nil)
	if len(ps) != 3 {
		t.Fatalf("DemoPDF gives %d problems, want 3", len(ps))
	}
	want := []struct {
		code string
		same []string
		text string
	}{
		{"341041", nil, `Zeigen Sie, daß die Zahl $z = 7 + 7^3 + 7^5 + 7^7 + \ldots + 7^{93} + 7^{95}$ durch 336 teilbar ist!`},
		{"341042", []string{"340943"}, `Auf der Seite $AB$ des Quadrates $ABCD$ werde ein Punkt $X \ne A$ gewählt. Dann werde das Quadrat durch die Strecken $AC$ und $XD$ in vier Teilflächen zerlegt.

Ermitteln Sie alle Möglichkeiten, die Wahl von $X$ so zu treffen, daß es natürliche Zahlen $p, q$ und $r$ gibt, für die die Flächeninhalte dieser Teilflächen in geeigneter Reihenfolge im Verhältnis $1 : p : q : r$ stehen!`},
		{"341043", nil, "Beweisen Sie, daß für jede natürliche Zahl"},
	}
	for i, w := range want {
		p := ps[i]
		if p.Code != w.code || !slices.Equal(p.Same, w.same) || p.Page != 2 || p.Figure || len(p.Flags) > 0 {
			t.Errorf("problem %d = %s %v page %d figure %v flags %v", i, p.Code, p.Same, p.Page, p.Figure, p.Flags)
		}
		if p.Text != w.text {
			t.Errorf("%s text:\n got %q\nwant %q", w.code, p.Text, w.text)
		}
	}
	// Ink in the rows of 341042 marks a figure there and nowhere else.
	ps = DemoPDF(ParsePDFXML([]byte(demoFixture)), func(page, top, bottom int) bool { return page == 2 && top <= 540 && bottom >= 540 })
	if ps[0].Figure || !ps[1].Figure || ps[2].Figure {
		t.Errorf("figures %v %v %v, want false true false", ps[0].Figure, ps[1].Figure, ps[2].Figure)
	}
}

func TestDemoHyphen(t *testing.T) {
	for _, c := range []struct {
		cur, next string
		want      bool
	}{
		{"die Gesamt-", "länge", true},
		{"der DDR-", "Olympiade", false},
		{"a -", "b", false},
		{"kein Strich", "da", false},
	} {
		if got := demoHyphen(c.cur, c.next); got != c.want {
			t.Errorf("demoHyphen(%q, %q) = %v", c.cur, c.next, got)
		}
	}
}

func TestDemoSymbols(t *testing.T) {
	items := ParsePDFXML([]byte(`<?xml version="1.0" encoding="UTF-8"?>
<pdf2xml>
<page number="1" position="absolute" top="0" left="0" height="1262" width="892">
	<fontspec id="0" size="15" family="IDCCAA+LMRoman10" color="#000000"/>
	<fontspec id="5" size="15" family="NBBQIW+LMMathItalic10" color="#000000"/>
	<fontspec id="7" size="15" family="VNRXCH+LMMathSymbols10" color="#000000"/>
<text top="437" left="95" width="108" height="13" font="0">Aufgabe 341041:</text>
<text top="460" left="95" width="80" height="13" font="0">Es sei</text>
<text top="460" left="180" width="20" height="13" font="5"><i>AB</i></text>
<text top="460" left="204" width="10" height="13" font="7">k</text>
<text top="460" left="218" width="20" height="13" font="5"><i>CD</i></text>
<text top="460" left="242" width="40" height="13" font="0">und</text>
<text top="460" left="287" width="7" height="13" font="0">4</text>
<text top="460" left="294" width="4" height="13" font="5"><i>,</i></text>
<text top="460" left="301" width="30" height="13" font="0">5 cm.</text>
</page>
</pdf2xml>
`))
	ps := DemoPDF(items, nil)
	if len(ps) != 1 {
		t.Fatalf("DemoPDF gives %d problems", len(ps))
	}
	if want := `Es sei $AB \parallel CD$ und 4,5 cm.`; ps[0].Text != want {
		t.Errorf("text %q, want %q", ps[0].Text, want)
	}
}
