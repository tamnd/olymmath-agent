package official

import (
	"slices"
	"testing"

	"golang.org/x/text/encoding/charmap"
)

func TestRuHTML(t *testing.T) {
	for _, c := range []struct {
		in, want string
		figure   bool
	}{
		{`Докажите, что a<sup>2</sup> + b<sup>2</sup> &ge; 2ab.`, `Докажите, что $a^2 + b^2 \ge 2ab$.`, false},
		{`В треугольнике <i>АВС</i> угол <u>/</u>A равен 60<sup>o</sup>. (А. Шаповалов)`, `В треугольнике $ABC$ угол $\angle A$ равен $60^\circ$.`, false},
		{`Найдите <font face="symbol">a</font> + <font face="symbol">b</font>, если x<sub>1</sub> = 5.`, `Найдите $\alpha + \beta$, если $x_1 = 5$.`, false},
		{`Прямые <i>AB</i> <u>&nbsp;|&nbsp;</u> <i>CD</i>. <img src="a.gif"> Доска 8*8.`, `Прямые $AB \parallel CD$. Доска $8 \times 8$.`, true},
		{`<ul><li>первое;<li>второе.</ul>`, "- первое;\n- второе.", false},
		{`Числа $x$ и y — целые.`, `Числа $x$ и $y$ — целые.`, false},
		{`Точка Е лежит на стороне ВС. СССР.`, `Точка $E$ лежит на стороне $BC$. СССР.`, false},
		{`Известно, что (a+b+c)c<0. Докажите, что b<sup>2</sup>>4ac и a<b.`, `Известно, что $(a+b+c)c<0$. Докажите, что $b^2>4ac$ и $a<b$.`, false},
		{`Обозначим её {x<sub>n</sub>}. (<I>Фольклор</I>) <A HREF="#11kl2">Решение</A>`, `Обозначим её $\{x_n\}$.`, false},
		{`Известно, что (a+b+c)c< 0.`, `Известно, что $(a+b+c)c< 0$.`, false},
		{`Найдите $\{x\}$, где {x} - дробная часть.`, `Найдите $\{x\}$, где $\{x\}$ - дробная часть.`, false},
		{`<p>Первый абзац.</p><p>Второй<br>абзац.</p>`, "Первый абзац.\n\nВторой\nабзац.", false},
		{`Числа cos a<sub>1</sub> и 2cos x лежат на отрезке [0; 3&pi;/2]. Прямые A<sub>3</sub>B<sub>4</sub> || AB.`, `Числа $\cos a_1$ и $2\cos x$ лежат на отрезке [0; $3\pi/2$]. Прямые $A_3B_4 \parallel AB$.`, false},
		{`Докажите, что $\operatorname{tg} x$ и tg y равны.`, `Докажите, что $\operatorname{tg} x$ и $\operatorname{tg} y$ равны.`, false},
		{`Докажите, что $В\sqrt{Р + Т}\geqslant 2Р$.`, `Докажите, что $B\sqrt{P + T}\geqslant 2P$.`, false},
		{`Решите уравнение $$\operatorname{tg}\pi x = [x]$$, где x - число.`, `Решите уравнение $$\operatorname{tg}\pi x = [x]$$, где $x$ - число.`, false},
		{`<p>Докажите это.</p><p>Из трех последних задач 11 класса в зачет идут две.</p>`, "Докажите это.", false},
	} {
		got, fig := RuHTML(c.in)
		if got != c.want || fig != c.figure {
			t.Errorf("RuHTML(%q) = %q, %v; want %q, %v", c.in, got, fig, c.want, c.figure)
		}
	}
}

func TestRuLayout(t *testing.T) {
	if !RuLayout(`<table><tr><td>x</td></tr></table>`) {
		t.Error("a table is not layout")
	}
	if !RuLayout(`<font face=symbol>¾¾</font>`) {
		t.Error("a fraction bar of the Symbol font is not layout")
	}
	if RuLayout(`Докажите, что a<sup>2</sup> &gt; 0.`) {
		t.Error("a superscript is layout")
	}
}

const ruPage = `<html><body>
<h2>Условия задач</h2>
<h3>8 класс</h3>
<p><b>1.</b> Можно ли разрезать квадрат 5x5 на 3 части?
<p><b>2.</b> Докажите, что 1 + 2 = 3. Число 3. не начинает задачу.
<h3>11 класс (1 день)</h3>
<p>Задача 1. Найдите все x, для которых x<sup>2</sup> = 1.
<h3>11 класс (2 день)</h3>
<p>1. Задача второго дня. <img src="f.png">
<p>2. <table><tr><td>формула</td></tr></table>
<h3>11 класс, вариант А</h3>
<p>1. Задача варианта.
<h2>Решения</h2>
<h3>8 класс</h3>
<p><b>1.</b> Решение, которое не задача.
</body></html>`

func TestRuPage(t *testing.T) {
	ps := RuPage(ruPage, "2003", "https://example.org/2003/", func(g int) bool { return g >= 8 })
	var got []string
	for _, p := range ps {
		got = append(got, p.Position)
	}
	if want := []string{"p81", "p82", "p111", "d111", "d112"}; !slices.Equal(got, want) {
		t.Fatalf("positions %v, want %v", got, want)
	}
	if ps[0].Text != `Можно ли разрезать квадрат $5 \times 5$ на 3 части?` {
		t.Errorf("p81 %q", ps[0].Text)
	}
	if ps[1].Text != `Докажите, что 1 + 2 = 3. Число 3. не начинает задачу.` {
		t.Errorf("p82 %q", ps[1].Text)
	}
	if ps[2].Text != `Найдите все $x$, для которых $x^2 = 1$.` {
		t.Errorf("p111 %q", ps[2].Text)
	}
	if !ps[3].Figure || len(ps[3].Flags) != 0 {
		t.Errorf("d111 figure %v flags %v", ps[3].Figure, ps[3].Flags)
	}
	if !slices.Equal(ps[4].Flags, []string{"layout"}) {
		t.Errorf("d112 flags %v", ps[4].Flags)
	}
	if ps[0].At("ru-moscow") != "ru-moscow/2003/p81" || ps[0].URL != "https://example.org/2003/" {
		t.Errorf("p81 at %s url %s", ps[0].At("ru-moscow"), ps[0].URL)
	}
}

func TestRuConfirm(t *testing.T) {
	all := []RuProblem{
		{Position: "p81", Text: "Можно ли разрезать квадрат на три равные части прямыми линиями?"},
		{Position: "p91", Text: "Можно ли разрезать квадрат на три равные части прямыми линиями?"},
		{Position: "p82", Text: "Докажите, что сумма углов треугольника равна развёрнутому углу."},
		{Position: "p83", Text: "Найдите все натуральные числа, которые делятся на сумму своих цифр."},
	}
	key := "Можно ли разрезать квадрат на три равные части прямыми?"
	k := RuConfirm(all[0], all, key)
	if !k.OK() || k.Rival == "p91" {
		t.Errorf("p81: %+v", k)
	}
	if k := RuConfirm(all[2], all, key); k.OK() || RuAgrees(k.J, k.C) {
		t.Errorf("p82 agrees with the key of p81: %+v", k)
	}
	// Two texts that differ but both agree with the key: no margin.
	twin := []RuProblem{all[0], {Position: "p84", Text: "Можно ли разрезать квадрат на три равные части прямыми линиями быстро?"}}
	if k := RuConfirm(twin[0], twin, key); k.OK() || k.Rival != "p84" {
		t.Errorf("p81 with a rival: %+v", k)
	}
	if k := RuConfirm(all[0], all, "Ёлка и ёж"); k.OK() {
		t.Errorf("an unrelated key: %+v", k)
	}
}

func TestRuDecode(t *testing.T) {
	text := `<html><head><meta http-equiv="Content-Type" content="text/html; charset=koi8-r"></head><body>Задача</body></html>`
	b, err := charmap.KOI8R.NewEncoder().Bytes([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := RuDecode(b); err != nil || got != text {
		t.Errorf("koi8-r: %q, %v", got, err)
	}
	if got, err := RuDecode([]byte(`<meta charset="utf-8">Задача`)); err != nil || got != `<meta charset="utf-8">Задача` {
		t.Errorf("utf-8: %q, %v", got, err)
	}
	if _, err := RuDecode([]byte(`<meta charset="no-such">`)); err == nil {
		t.Error("an unknown charset gives no error")
	}
}

func TestRuExtraction(t *testing.T) {
	if got := RuExtraction(RuProblem{}); got != "native/html (om-rutex/1)" {
		t.Errorf("page: %s", got)
	}
	if got := RuExtraction(RuProblem{Page: 2}); got != "native/pdftotext (om-rutex/1)" {
		t.Errorf("pdf: %s", got)
	}
	if got := RuExtraction(RuProblem{Page: 2, Model: "claude-opus-5-5"}); got != "native/pdftotext (claude-opus-5-5)" {
		t.Errorf("model: %s", got)
	}
}
