package official

import "testing"

func TestMathMLTeX(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		// The attributes of make4ht have no quotes.
		{`<math display=inline xmlns=http://www.w3.org/1998/Math/MathML><mrow><mn>137641</mn> <mo class=MathClass-rel stretchy=false>=</mo> <mn>37</mn><msup><mrow><mn>1</mn></mrow><mrow><mn>2</mn></mrow></msup></mrow></math>`, `137641=371^2`, true},
		{`<math><mfrac><mrow><mi>a</mi></mrow><mrow><mn>2</mn></mrow></mfrac></math>`, `\frac{a}{2}`, true},
		{`<math><msqrt><mrow><mn>2</mn></mrow></msqrt></math>`, `\sqrt{2}`, true},
		{`<math><mrow><menclose notation=updiagonalstrike><mn>1</mn></menclose><mn>37</mn></mrow></math>`, `\cancel{1} 37`, true},
		{`<math><munder><mrow><munder><mrow><mn>45</mn><mo>…</mo><mn>76</mn></mrow><mo>⏟</mo></munder></mrow><mrow><mn>6623</mn><mtext> digits</mtext></mrow></munder></math>`, `\underset{6623\text{ digits}}{\underbrace{45\ldots 76}}`, true},
		{`<math><mover><mrow><mi>AB</mi></mrow><mo>→</mo></mover></math>`, `\overrightarrow{\mathrm{AB}}`, true},
		{`<math><munderover><mo>∑</mo><mrow><mi>i</mi><mo>=</mo><mn>1</mn></mrow><mrow><mi>n</mi></mrow></munderover><mi>i</mi></math>`, `\sum_{i=1}^n i`, true},
		{`<math><mi>α</mi><mo>∕</mo><mi>𝜀</mi></math>`, `\alpha /\varepsilon`, true},
		// A sign that has no TeX here makes the reading not sure.
		{`<math><mi>☃</mi></math>`, ``, false},
	} {
		got, ok := MathMLTeX(c.in)
		if ok != c.ok || c.ok && got != c.want {
			t.Errorf("MathMLTeX(%s) = %q, %v, want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}
