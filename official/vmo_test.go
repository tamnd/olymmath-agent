package official

import "testing"

const vmoHTML = `<html><body>
<h3>9 класс</h3>
<p><b>9.1. </b><br>(<i>И. Иванов</i>) Докажите, что <i>a</i> <u>&gt;</u> <i>b</i>.
<p><b>9.2. </b>Числа 2<sup>0</sup>, 2<sup>1</sup>, 2<sup>2</sup> выписаны в ряд.
<p><b>9.3. </b>Функция <i>f</i>: <b>R</b> -&gt; <b>R</b> такова, что <i>f</i>(0) = 1.
<h3>10 класс</h3>
<p><b>9.1. </b>Задача с меткой другого класса.
<h3>11 класс</h3>
<p><b>11.1. </b>Дана табл. 2*<i>n</i>.
</body></html>`

func TestVmoPage(t *testing.T) {
	ps := VmoPage(vmoHTML, "2005", "https://olympiads.mccme.ru/vmo/31/zadachi.htm")
	// The paper of grade 10 has a mark of grade 9, so it is left out.
	want := []struct{ pos, text string }{
		{"p91", "Докажите, что $a \\ge b$."},
		{"p92", "Числа $2^0$, $2^1$, $2^2$ выписаны в ряд."},
		{"p93", "Функция $f$: $\\mathbb{R} \\to \\mathbb{R}$ такова, что $f(0) = 1$."},
		{"p111", "Дана табл. $2\\times n$."},
	}
	if len(ps) != len(want) {
		t.Fatalf("got %d problems, want %d: %+v", len(ps), len(want), ps)
	}
	for i, w := range want {
		if ps[i].Position != w.pos || ps[i].Text != w.text || len(ps[i].Flags) != 0 || ps[i].Page != 0 {
			t.Errorf("problem %d: got %s %q %v, want %s %q", i, ps[i].Position, ps[i].Text, ps[i].Flags, w.pos, w.text)
		}
	}
}

func TestVmoReals(t *testing.T) {
	for in, want := range map[string]string{
		"нет":            "нет",
		"$x \\in$ ℝ":     "$x \\in \\mathbb{R}$",
		"$f$: $ℝ\\to ℝ$": "$f$: $\\mathbb{R}\\to \\mathbb{R}$",
		"ℝ $\\to$ ℝ":     "$\\mathbb{R} \\to \\mathbb{R}$",
	} {
		if got := vmoReals(in); got != want {
			t.Errorf("%q: got %q, want %q", in, got, want)
		}
	}
}
