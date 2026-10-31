package site

import "testing"

func TestFormulaTokens(t *testing.T) {
	cases := map[string]string{
		`\frac{a}{b}`:                   "frac a b",
		`\dfrac{a}{b}`:                  "frac a b",
		`x^2+y^{2} = z^2`:               "x 2 + y 2 = z 2",
		`\left( a \leq 3.5 \right)`:     "( a le 3 . 5 )",
		`a \geqslant b, \, \{1\}`:       "a ge b , { 1 }",
		`\lfloor n\alpha \rfloor`:       "lfloor n alpha rfloor",
		`f(x) \ne 0 \text{ for } x ≥ 1`: "f ( x ) ne 0 f o r x ≥ 1",
	}
	for in, want := range cases {
		if got := FormulaTokens(in); got != want {
			t.Errorf("FormulaTokens(%q) = %q, want %q", in, got, want)
		}
	}
	if got := formulas("Let $x^2$ and $$\\frac12$$ be."); got != "x 2 ; frac 1 2" {
		t.Errorf("formulas %q", got)
	}
}
