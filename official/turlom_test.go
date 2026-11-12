package official

import (
	"slices"
	"testing"
)

func TestTurlomHTML(t *testing.T) {
	page := `<H2>Физика</H2><P>1. Задача по физике.
<H2>Конкурс по математике</H2>
<H3>Задачи</H3>
<H4>7-9 классы</H4>
<OL>
<LI>(7-9) Докажите, что a<SUP>2</SUP> &ge; 0.
<LI>Найдите x, если 2x = 4.
</OL>
<H4>10-11 классы</H4>
<P><B>1.</B> (10&ndash;11) Петя заплатил 5$ за книгу.
<P><B>2.</B> (10, 11) На рисунке дан квадрат ABCD.
<PRE>
+--+
</PRE>
<H4>Дополнительные задачи (для всех классов)</H4>
<OL><LI>Есть таблица:<TABLE><TR><TD>1</TD></TR></TABLE></OL>
<H3>Решения задач конкурса по математике</H3>
<OL><LI>Решение.</OL>`
	got := TurlomHTML(page, "1996", "u")
	want := []RuProblem{
		{Edition: "1996", Position: "p1", Text: `Докажите, что $a^2 \ge 0$.`, URL: "u"},
		{Edition: "1996", Position: "p2", Text: `Найдите $x$, если $2x = 4$.`, URL: "u"},
		{Edition: "1996", Position: "p3", Text: `Петя заплатил 5\$ за книгу.`, URL: "u"},
		{Edition: "1996", Position: "p4", Text: "На рисунке дан квадрат $ABCD$. +--+", URL: "u", Figure: true},
		{Edition: "1996", Position: "p5", Text: "Есть таблица:\n\n1", URL: "u", Flags: []string{"layout"}},
	}
	if len(got) != len(want) {
		t.Fatalf("TurlomHTML gave %d problems, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Position != w.Position || g.Text != w.Text || g.Figure != w.Figure || !slices.Equal(g.Flags, w.Flags) || g.URL != w.URL || g.Edition != w.Edition {
			t.Errorf("problem %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestTurlomHTMLNoMath(t *testing.T) {
	if got := TurlomHTML(`<H2>Физика</H2><OL><LI>Задача.</OL>`, "2000", "u"); len(got) != 0 {
		t.Errorf("TurlomHTML of a page with no mathematics = %+v", got)
	}
}

func TestTurlomPDF(t *testing.T) {
	pages := []string{
		"XLVIII Турнир имени М.В.Ломоносова\nКонкурс по математике\n28 сентября 2025 года\nлист 1 из 2\nВ скобках указано, каким классам рекомендуется задача.\n" +
			"1 (6; ответ). Сколько солнеч-\nных дней —в сентябре?\n" +
			"2 (6–7; ответ). Жора хочет разбить куб на 𝑛3\nкубиков.",
		"XLVIII Турнир имени М.В.Ломоносова\nКонкурс по математике\n28 сентября 2025 года\nлист 2 из 2\n" +
			"3. (8–11) Отрезки AB =\n= CD. Найдите x, как на рис. 1.\nРис. 1\n(См. также условия задач на обороте)\n" +
			"Не забудьте подписать свою работу.\n1. (6–7) Копия.",
	}
	got := TurlomPDF(pages, "2025", "u")
	want := []RuProblem{
		{Edition: "2025", Position: "p1", Text: "Сколько солнечных дней — в сентябре?", URL: "u", Page: 1},
		{Edition: "2025", Position: "p2", Text: "Жора хочет разбить куб на $n3$ кубиков.", URL: "u", Page: 1},
		{Edition: "2025", Position: "p3", Text: "Отрезки $AB = CD$. Найдите $x$, как на рис. 1.", URL: "u", Page: 2, Figure: true},
	}
	if len(got) != len(want) {
		t.Fatalf("TurlomPDF gave %d problems, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Position != w.Position || g.Text != w.Text || g.Figure != w.Figure || g.Page != w.Page {
			t.Errorf("problem %d = %+v, want %+v", i, g, w)
		}
	}
}

func TestPDFText(t *testing.T) {
	for _, c := range []struct {
		in, want string
		figure   bool
	}{
		{"Докажите, что PQ \x01 AC.", "Докажите, что $PQ$ [?] $AC$.", false},
		{"Найдите угол.\nA B\nT\nD C\nE F", "Найдите угол.", true},
		{"Угол равен 45◦\n, а AB ̸= BC.", `Угол равен $45^\circ$, а $AB \ne BC$.`, false},
	} {
		got, fig := PDFText(c.in)
		if got != c.want || fig != c.figure {
			t.Errorf("PDFText(%q) = %q, %v; want %q, %v", c.in, got, fig, c.want, c.figure)
		}
	}
}

func TestSharyginFinal(t *testing.T) {
	pages := []string{
		"IX Олимпиада по геометрии им. И.Ф.Шарыгина\nФинал. Первый день. 8 класс\nРатмино, 1 августа 2013 г.\n" +
			"1. (Б.Френкин) Точки A1 и B1 лежат на ω1. Отношение равно\n√\n3 :\n2. На сторонах ABC отмечены точки.\n2. Дан квадрат ABCD.",
		"IX Олимпиада по геометрии им. И.Ф.Шарыгина\nФинал. Второй день. 8 класс\nРатмино, 2 августа 2013 г.\n" +
			"5. Дан угол A12. Ответ. 5.\nРешение. Да.\n6. Последняя задача.",
		"Финальный тур, Ратмино, 2017 г., 31 июля\n10 класс. Первый день\n10.1. Задача десятого класса.\n" +
			"Финальный тур, Ратмино, 2017 г., 31 июля\n10 класс. Первый день\n10.1. Копия.",
	}
	got := SharyginFinal(pages, "2013", "u")
	want := []RuProblem{
		{Position: "p81", Text: `Точки $A_1$ и $B_1$ лежат на $\omega_1$. Отношение равно $\sqrt$ 3: 2. На сторонах $ABC$ отмечены точки.`, Page: 1},
		{Position: "p82", Text: "Дан квадрат $ABCD$.", Page: 1},
		{Position: "p85", Text: "Дан угол $A_{12}$.", Page: 2},
		{Position: "p86", Text: "Последняя задача.", Page: 2},
		{Position: "p101", Text: "Задача десятого класса.", Page: 3},
	}
	if len(got) != len(want) {
		t.Fatalf("SharyginFinal gave %d problems, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		g, w := got[i], want[i]
		if g.Position != w.Position || g.Text != w.Text || g.Page != w.Page || g.Edition != "2013" || g.URL != "u" {
			t.Errorf("problem %d = %+v, want %+v", i, g, w)
		}
	}
}
