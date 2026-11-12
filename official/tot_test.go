package official

import (
	"slices"
	"strings"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func TestTotHeading(t *testing.T) {
	for _, c := range []struct {
		in          string
		group, tier byte
		ok          bool
	}{
		{"8 – 9 классы, сложный вариант, 25 октября 2009 г.", 'j', 'a', true},
		{"8 -- 9 классы, сложный вариант, 26 октября 2008 г.", 'j', 'a', true},
		{"10-11 кл., тренировочный вариант, 24 февраля 2002 г.", 's', 'o', true},
		{"10 – 11 классы", 's', 0, true},
		{"Осенний тур", 0, 0, false},
	} {
		g, tier, ok := TotHeading(c.in)
		if g != c.group || tier != c.tier || ok != c.ok {
			t.Errorf("TotHeading(%q) = %c %c %v, want %c %c %v", c.in, g, tier, ok, c.group, c.tier, c.ok)
		}
	}
}

func TestTotHyphen(t *testing.T) {
	for _, c := range []struct {
		cur, next string
		want      bool
	}{
		{"одна пара про-", "тивоположных граней", true},
		{"какой-", "то угол", false},
		{"по-", "разному", false},
		{"по-", "ставили", true},
		{"во-", "первых,", false},
		{"из-", "за этого", false},
		{"кое-", "где", false},
		{"$AB$-", "угольник", false},
		{"точ-", "ка", true},
	} {
		if got := totHyphen(c.cur, c.next); got != c.want {
			t.Errorf("totHyphen(%q, %q) = %v, want %v", c.cur, c.next, got, c.want)
		}
	}
}

func TestPDFTidy(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{`a+b=c`, `a + b = c`},
		{`x\not=y`, `x \ne y`},
		{`-1<x`, `-1 < x`},
		{`a_{n+1}`, `a_{n + 1}`},
		{`2^{-k}`, `2^{-k}`},
	} {
		if got := PDFTidy(c.in); got != c.want {
			t.Errorf("PDFTidy(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// totPDFXML is the start of a paper PDF of the Tournament as pdftohtml -xml gives it: a title, a heading, a column of points and two problems, one with math.
const totPDFXML = `<page number="1" position="absolute" top="0" left="0" height="1262" width="892">
	<fontspec id="0" size="18" family="MBENXF+SFRB1200" color="#000000"/>
	<fontspec id="1" size="18" family="AKMKUQ+SFRM1200" color="#000000"/>
	<fontspec id="3" size="18" family="PHVQDD+SFTI1200" color="#000000"/>
	<fontspec id="4" size="18" family="YQEGAZ+CMR12" color="#000000"/>
	<fontspec id="5" size="18" family="SZXRFY+CMSY10" color="#000000"/>
<text top="16" left="60" width="364" height="16" font="0">ТРИДЦАТЬ ПЕРВЫЙ ТУРНИР ГОРОДОВ</text>
<text top="37" left="60" width="107" height="16" font="1">Осенний тур,</text>
<text top="59" left="60" width="409" height="16" font="1">8 – 9 классы, сложный вариант, 25 октября 2009 г.</text>
<text top="140" left="69" width="50" height="16" font="1">баллы</text>
<text top="140" left="137" width="54" height="16" font="1">задачи</text>
<text top="183" left="89" width="9" height="16" font="1">4</text>
<text top="183" left="146" width="14" height="16" font="1">1.</text>
<text top="183" left="181" width="625" height="16" font="1">У Миши есть 1000 одинаковых кубиков, у каждого из которых одна пара про-</text>
<text top="205" left="181" width="134" height="16" font="1">тивоположных граней белая. Он собрал из них куб</text>
<text top="205" left="319" width="18" height="16" font="4">10</text>
<text top="204" left="337" width="14" height="17" font="5">×</text>
<text top="205" left="352" width="18" height="16" font="4">10</text>
<text top="205" left="371" width="10" height="16" font="1">.</text>
<text top="226" left="682" width="123" height="16" font="3">М.В.Мурашкин</text>
<text top="270" left="89" width="9" height="16" font="1">6</text>
<text top="270" left="146" width="14" height="16" font="1">2.</text>
<text top="270" left="181" width="600" height="16" font="1">Докажите, что у большого куба есть одноцветная грань.</text>
</page>
`

func TestTotPDF(t *testing.T) {
	ps := TotPDF(ParsePDFXML([]byte(totPDFXML)), "2009-fall", "u", 'a')
	if len(ps) != 2 {
		t.Fatalf("TotPDF gives %d problems, want 2: %+v", len(ps), ps)
	}
	if ps[0].Position != "ja1" || ps[1].Position != "ja2" {
		t.Errorf("positions %q %q, want ja1 ja2", ps[0].Position, ps[1].Position)
	}
	want := `У Миши есть 1000 одинаковых кубиков, у каждого из которых одна пара противоположных граней белая. Он собрал из них куб $10 \times 10$.`
	if ps[0].Text != want {
		t.Errorf("text %q, want %q", ps[0].Text, want)
	}
	if ps[1].Text != "Докажите, что у большого куба есть одноцветная грань." {
		t.Errorf("text %q", ps[1].Text)
	}
}

// totPageHTML is a page of an old tournament: a title, a heading and the papers in a <pre>, with points, an author and a line cut by a hyphen.
const totPageHTML = `<html><body>
<p>ДВАДЦАТЬ ТРЕТИЙ ТУРНИР ГОРОДОВ</p>
<p>Весенний тур</p>
<p>10-11 кл., тренировочный вариант, 24 февраля 2002 г.</p>
<pre>
Задача 1.(4)
Для натуральных чисел x и y число x^2+xy+y^2 в десятичной записи оканчи-
вается нулем. Докажите, что оно оканчивается хотя бы двумя нулями.
                                         В. В. Произволов
Задача 2(5)
Найдите все n, для которых найдется n-угольник с такими углами:
а) (2 балла) все углы равны;
б) (3 балла) все углы различны.
</pre>
<hr>
</body></html>`

func TestTotPage(t *testing.T) {
	ps := TotPage(totPageHTML, 23, "u")
	if len(ps) != 2 {
		t.Fatalf("TotPage gives %d problems, want 2: %+v", len(ps), ps)
	}
	if ps[0].Edition != "2002-spring" || ps[0].Position != "so1" || ps[1].Position != "so2" {
		t.Errorf("got %s %s %s, want 2002-spring so1 so2", ps[0].Edition, ps[0].Position, ps[1].Position)
	}
	if want := "Для натуральных чисел $x$ и $y$ число $x^2+xy+y^2$ в десятичной записи оканчивается нулем. Докажите, что оно оканчивается хотя бы двумя нулями."; ps[0].Text != want {
		t.Errorf("text %q, want %q", ps[0].Text, want)
	}
	if want := "Найдите все $n$, для которых найдется $n$-угольник с такими углами:\nа) все углы равны;\nб) все углы различны."; ps[1].Text != want {
		t.Errorf("text %q, want %q", ps[1].Text, want)
	}
	for _, p := range ps {
		if len(p.Flags) > 0 || p.Figure {
			t.Errorf("%s: flags %v figure %v", p.Position, p.Flags, p.Figure)
		}
	}
}

func TestTotPageArt(t *testing.T) {
	page := strings.Replace(totPageHTML, "в десятичной записи", "в записи <PRE>  1\n-----\n 1 + x</PRE>", 1)
	page = strings.Replace(page, "<pre>", "<p>", 1)
	page = strings.Replace(page, "</pre>", "", 1)
	page = strings.ReplaceAll(page, "\nЗадача", "<br>Задача")
	ps := TotPage(page, 23, "u")
	if len(ps) != 2 {
		t.Fatalf("TotPage gives %d problems, want 2", len(ps))
	}
	if !slices.Contains(ps[0].Flags, "layout") {
		t.Errorf("a formula in a <pre> gives flags %v, want layout", ps[0].Flags)
	}
}

func TestTotLatin(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"х_1", "x_1"},
		{"АК=АО", "AK=AO"},
		{"Докажите", "Докажите"},
		{"НОД(a_1", "НОД(a_1"},
	} {
		if got := totLatin(c.in); got != c.want {
			t.Errorf("totLatin(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTotEllipsis(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"вида $a_1+$ ... $+a_k$, где", `вида $a_1+ \ldots +a_k$, где`},
		{"числа $a_1$, ... , $a_n$", "числа $a_1$, ... , $a_n$"},
	} {
		if got := totEllipsis(c.in); got != c.want {
			t.Errorf("totEllipsis(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// totTeXSrc is a paper of a TeX file of the Tournament, as in os30os.tex.
const totTeXSrc = `\noindent
{\bf ТРИДЦАТЫЙ ТУРНИР ГОРОДОВ}

\noindent
8 -- 9 классы, сложный вариант, 26 октября 2008 г.

\noindent
{\sl (Итог подводится по тр\"ем задачам.)}
4\qquad&\begin{tabular}{p{.4truecm}p{14.7truecm}}1.&
Даны числа $a_1$, $a_2$, \dots, $a_n$. Известно, что $a_1+\dots+a_n\leq\frac12.$
\end{tabular}\\
\qquad&\begin{tabular}{p{.4truecm}p{14.7truecm}}&
\hfill (С.Слободник)
\end{tabular}\\ \\
\qquad&\begin{tabular}{p{.4truecm}p{14.7truecm}}$2$.&
Докажите, что в после\-дова\-тель\-ности то \hbox{$a_n=a_{n-1}+1$,}
\end{tabular}\\
5\qquad&\begin{tabular}{p{.4truecm}p{14.7truecm}}а)&
число 1 встречается; % a note
\end{tabular}\\
5\qquad&\begin{tabular}{p{.4truecm}p{14.7truecm}}б)&
вс\"е встречается.
\end{tabular}\\
\qquad&\begin{tabular}{p{.4truecm}p{14.7truecm}}&
(Вот первые члены: 1, 2, 1, \dots)
\end{tabular}
`

func TestTotTeX(t *testing.T) {
	b, err := charmap.CodePage866.NewEncoder().String(totTeXSrc)
	if err != nil {
		t.Fatal(err)
	}
	src := TotTeXDecode([]byte(b))
	if src != totTeXSrc {
		t.Fatalf("TotTeXDecode does not give the text back")
	}
	ps := TotTeX(src, "2008-fall", "u")
	if len(ps) != 2 {
		t.Fatalf("TotTeX gives %d problems, want 2: %+v", len(ps), ps)
	}
	if want := `Даны числа $a_1$, $a_2$, ..., $a_n$. Известно, что $a_1+\dots+a_n\leq\frac12$.`; ps[0].Text != want || ps[0].Position != "ja1" {
		t.Errorf("%s %q, want ja1 %q", ps[0].Position, ps[0].Text, want)
	}
	if want := "Докажите, что в последовательности то $a_n=a_{n-1}+1$,\nа) число 1 встречается;\nб) всё встречается.\n(Вот первые члены: 1, 2, 1, ...)"; ps[1].Text != want || ps[1].Position != "ja2" {
		t.Errorf("%s %q, want ja2 %q", ps[1].Position, ps[1].Text, want)
	}
	if ps[0].Tool != "tex" || RuExtraction(ps[0]) != "native/tex (om-rutex/1)" {
		t.Errorf("extraction %q", RuExtraction(ps[0]))
	}
}
