package normalize

import (
	"strings"
	"testing"
)

func TestText(t *testing.T) {
	cases := map[string]string{
		"Let \\(x\\) be real.":             "Let $x$ be real.",
		"Show \\[a+b\\] holds.":            "Show $$a+b$$ holds.",
		"“Quoted” — text":                  `"Quoted" - text`,
		"a   b\n\n\n\nc":                   "a b\n\nc",
		"Find $\\displaystyle\\sum x$.":    "Find $ \\sum x$.",
		"Prove it. [hide=Answer]42[/hide]": "Prove it.",
		"Café":                            "Café",
	}
	for in, want := range cases {
		if got := Text(in); got != want {
			t.Errorf("Text(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestStripMarker(t *testing.T) {
	cases := []struct{ in, pos, want string }{
		{"Problem 3. Let x be real.", "p3", "Let x be real."},
		{"3. Let x be real.", "p3", "Let x be real."},
		{"P3) Let x be real.", "p3", "Let x be real."},
		{"A1. Let x be real.", "a1", "Let x be real."},
		{"**Problem 2.** Show it.", "p2", "Show it."},
		{"3. Let x be real.", "p4", "3. Let x be real."},
		{"2024 is a year.", "p2", "2024 is a year."},
	}
	for _, c := range cases {
		if got := StripMarker(c.in, c.pos); got != c.want {
			t.Errorf("StripMarker(%q, %q) = %q, want %q", c.in, c.pos, got, c.want)
		}
	}
}

func TestTokens(t *testing.T) {
	if a, b := strings.Join(Tokens("$x^2+\\frac{1}{2}$"), " "), strings.Join(Tokens("$x ^ 2 + \\frac 1 2$"), " "); a != b {
		t.Errorf("%q != %q", a, b)
	}
}

func TestGuard(t *testing.T) {
	c := Guard("Let n be a posit1ve integer with n > 2.", "Let n be a positive integer with n > 2.")
	if !c.OK(MaxShare) {
		t.Errorf("an OCR fix failed the guard: %+v", c)
	}
	c = Guard("Let n be a positive integer with n > 2.", "Let n be a positive integer with n > 3.")
	if c.OK(MaxShare) || len(c.Flags) != 2 {
		t.Errorf("a changed number passed the guard: %+v", c)
	}
	c = Guard("Prove that the triangle is acute.", "Show that every such triangle must be an acute one.")
	if c.OK(MaxShare) {
		t.Errorf("a rewrite passed the guard: %+v", c)
	}
}

func TestCutTail(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Compute the number of ways to place the weights on the pans. Answer. The number is $(2n-1)!!$.", "Compute the number of ways to place the weights on the pans."},
		{"Prove that the right pan is never heavier than the left pan. ## C2 Suppose that 1000 students stand in a circle.", "Prove that the right pan is never heavier than the left pan."},
		{"Answer: find all $n$ such that the sum is a square number for each prime.", "Answer: find all $n$ such that the sum is a square number for each prime."},
	}
	for _, c := range cases {
		if got := CutTail(c.in); got != c.want {
			t.Errorf("CutTail(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestStripCountry(t *testing.T) {
	cases := map[string]string{
		"BEL Let $ABC$ be a triangle.":                  "Let $ABC$ be a triangle.",
		"RUS Variant 1. A grasshopper jumps.":           "A grasshopper jumps.",
		"Let $ABC$ be a triangle.":                      "Let $ABC$ be a triangle.",
		"ABC is a word with three letters in it today.": "ABC is a word with three letters in it today.",
	}
	for in, want := range cases {
		if got := StripCountry(in); got != want {
			t.Errorf("StripCountry(%q) = %q, want %q", in, got, want)
		}
	}
}
