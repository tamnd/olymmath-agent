package official

import (
	"strconv"
	"strings"
	"testing"
)

// ruBBox makes the output of pdftotext -raw -bbox for pages of words: each word is x, y, height and text.
func ruBBox(pages ...[]ruW) []byte {
	var b strings.Builder
	b.WriteString("<doc>\n")
	for _, ws := range pages {
		b.WriteString("<page width=\"595\" height=\"842\">\n")
		for _, w := range ws {
			x1 := w.x + 6*float64(len([]rune(w.t)))
			b.WriteString("<word xMin=\"" + ruF(w.x) + "\" yMin=\"" + ruF(w.y-w.h) + "\" xMax=\"" + ruF(x1) + "\" yMax=\"" + ruF(w.y) + "\">" + w.t + "</word>\n")
		}
		b.WriteString("</page>\n")
	}
	b.WriteString("</doc>\n")
	return []byte(b.String())
}

func ruF(f float64) string { return strconv.FormatFloat(f, 'f', -1, 64) }

type ruW struct {
	x, y, h float64
	t       string
}

func TestRuPDF(t *testing.T) {
	// Word x positions leave a gap of 4 between words, so they get a space; "x" and its exponent "2" touch.
	at := func(x, y float64, ts ...string) []ruW {
		var out []ruW
		for _, s := range ts {
			out = append(out, ruW{x, y, 10, s})
			x += 6*float64(len([]rune(s))) + 4
		}
		return out
	}
	var p1 []ruW
	p1 = append(p1, at(50, 20, "8", "класс")...)
	p1 = append(p1, at(50, 40, "1.", "Сколько", "решений", "у", "урав-")...)
	p1 = append(p1, at(50, 52, "нения")...)
	p1 = append(p1, ruW{110, 52, 10, "x"}, ruW{116, 48, 6, "2"})
	p1 = append(p1, at(126, 52, "=", "4?")...)
	p1 = append(p1, at(50, 70, "2.", "Найдите", "какое-")...)
	p1 = append(p1, at(50, 82, "то", "число.")...)
	p1 = append(p1, at(50, 94, "а)", "Одно;")...)
	p1 = append(p1, at(50, 106, "б)", "два", "(рис.", "1).")...)
	p1 = append(p1, at(50, 124, "Подробнее", "на", "сайте")...)
	p1 = append(p1, at(50, 140, "8", "класс")...)
	p1 = append(p1, at(50, 160, "1.", "Копия.")...)
	var p2 []ruW
	p2 = append(p2, at(50, 20, "11", "класс")...)
	p2 = append(p2, at(50, 40, "1.", "Докажите", "это.")...)
	p2 = append(p2, at(50, 52, "3.", "Не", "по", "порядку.")...)
	p2 = append(p2, at(50, 64, "12", "класс")...)
	p2 = append(p2, at(50, 84, "1.", "Лишний", "класс.")...)

	lines, koi8 := RuPDFLines(ruBBox(p1, p2))
	if koi8 {
		t.Error("a Unicode page is not koi8")
	}
	grades := func(g int) bool { return g >= 8 && g <= 11 }
	ps := RuPDF(lines, "2015", "https://example.org/var.pdf", grades, true, false)
	var got []string
	for _, p := range ps {
		got = append(got, p.Position)
	}
	if strings.Join(got, " ") != "p81 p82 d111" {
		t.Fatalf("positions %v", got)
	}
	if p := ps[0]; p.Text != "Сколько решений у уравнения $x2 = 4$?" || p.Page != 1 || len(p.Flags) != 1 || p.Flags[0] != "layout" || p.Figure {
		t.Errorf("p81: %+v", p)
	}
	if p := ps[1]; p.Text != "Найдите какое-то число.\nа) Одно;\nб) два (рис. 1)." || !p.Figure || len(p.Flags) != 0 {
		t.Errorf("p82: %+v", p)
	}
	if p := ps[2]; p.Text != "Докажите это. 3. Не по порядку." || p.Page != 2 {
		t.Errorf("d111: %+v", p)
	}
}

func TestRuPDFLinesKOI8(t *testing.T) {
	// "Задача" in KOI8-R codes read as Latin-1, then an en dash as "{".
	koi := string([]rune{0xfa, 0xc1, 0xc4, 0xc1, 0xde, 0xc1})
	lines, koi8 := RuPDFLines(ruBBox([]ruW{{50, 20, 10, koi}, {100, 20, 10, "{"}, {110, 20, 10, "1"}}))
	if !koi8 || len(lines) != 1 || lines[0].Text != "Задача – 1" {
		t.Errorf("koi8=%v %+v", koi8, lines)
	}
}

func TestRuPDFText(t *testing.T) {
	got := RuPDFText([]string{"Задача 3. Найдите вели-", "чину угла ABC: (А.Б.Иванов)"}, true)
	if got != "Найдите величину угла $ABC$." {
		t.Errorf("got %q", got)
	}
}
