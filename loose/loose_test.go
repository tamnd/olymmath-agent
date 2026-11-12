package loose

import (
	"strings"
	"testing"
)

func TestClean(t *testing.T) {
	cases := []struct {
		name, in, want, why string
	}{
		{"plain", "Let $ABC$ be an acute triangle with altitude $AD$. Show that $AC = BC$ if and only if $\\angle AGC = 90^{\\circ}$.", "Let $ABC$ be an acute triangle with altitude $AD$. Show that $AC = BC$ if and only if $\\angle AGC = 90^{\\circ}$.", ""},
		{"lead mark", "Problem 1. Find all prime numbers $p$ for which $x^p+y^p+z^p-x-y-z$ is a product of exactly three distinct primes.", "Find all prime numbers $p$ for which $x^p+y^p+z^p-x-y-z$ is a product of exactly three distinct primes.", ""},
		{"heading", "# Task 3.\n\nVintik drew a rectangle $9 \\times 11$ cm and marked two hundred points in it. Prove that some square contains three points.", "Vintik drew a rectangle $9 \\times 11$ cm and marked two hundred points in it. Prove that some square contains three points.", ""},
		{"answer cut", "3. Five runners ran a relay. If the first runner ran twice as fast, they would have spent $5\\%$ less time. By what percentage less time would they have spent?\n\nOTBET: by $8\\%$", "Five runners ran a relay. If the first runner ran twice as fast, they would have spent $5\\%$ less time. By what percentage less time would they have spent?", ""},
		{"too short", "Find $x$ if $x+1=2$.", "", TooShort},
		{"solution", "Solution. Since $a+b=c$ we get the claim, and the proof is done by induction on $n$.", "", Solution},
		{"glued", "Problem 1. Prove that $n^2+n$ is even for every integer $n$.\nProblem 2. Find all real $x$ with $x^2=2x$.", "", Glued},
		{"next number", "3. Prove that $n^2+n$ is even for every integer $n$ and every prime.\n4. Find all real $x$ with $x^2=2x$.", "", Glued},
		{"stub", "Prove the lemma.", "", TooShort},
		{"stub long", "Prove Lemma 2 from the paper of Erdos and Szekeres on convex polygons.", "", Stub},
		{"request", "Can someone help me with this one: prove that $a^2+b^2 \\ge 2ab$ for all real numbers $a$ and $b$?", "", Request},
		{"context", "In the previous problem, replace $5$ by $7$ and find the number of ways to color the board.", "", Context},
		{"figure", "In the figure below, $ABCD$ is a square of side $1$. Find the area of the shaded region.", "", Figure},
		{"image", "Find the area of the region [img]https://latex.artofproblemsolving.com/a.png[/img] if $x=1$ and $y = 2$.", "", Figure},
		{"contents", "Contents\n1 Algebra ........ 3\n2 Geometry ........ 7\n3 Number theory ....... 9", "", Contents},
		{"broken tex", "Let $x$ be a real number such that $x^2 + {1 = 3$. Find all possible values of $x$.", "", BrokenTeX},
		{"title", "USAMO 2022 Problem 1", "", TooShort},
		{"asy", "Find the area of the shaded region.\n[asy]draw((0,0)--(1,1));[/asy]", "", Figure},
		{"link", "Prove the claim of this [problem](http://www.mathlinks.ro/Forum/viewtopic.php?p=845756) for every $n$.", "", WebLink},
		{"details", "Prove that given $n$ points in the plane, not all on a line, some line passes through exactly two of them.\n\n<details><summary>original wording</summary>Demostrar que dados n puntos</details>", "Prove that given $n$ points in the plane, not all on a line, some line passes through exactly two of them.", ""},
		{"bbcode", "Find all [i]good[/i] numbers $n$ such that <u>every</u> divisor $d$ of $n$ has $d+1$ prime.[list][*]$n$ is odd.[/list]", "Find all *good* numbers $n$ such that every divisor $d$ of $n$ has $d+1$ prime.\n\n- $n$ is odd.", ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, why := Clean(c.in)
			if got != c.want || why != c.why {
				t.Errorf("Clean(%q) = %q, %q; want %q, %q", c.in, got, why, c.want, c.why)
			}
		})
	}
}

func TestLang(t *testing.T) {
	cases := []struct{ text, named, want string }{
		{"Let $ABC$ be a triangle and let $M$ be the midpoint of $BC$. Prove that the lines are concurrent.", "", "en"},
		{"Determinar todos los pares $(a, b)$ de enteros positivos para los cuales la fracción es un número entero.", "español", "es"},
		{"Sea $A$ uno de los dos puntos de intersección de dos círculos distintos con centros en la recta.", "", "es"},
		{"Найдите все натуральные $n$, для которых число $n^2+1$ делится на $n+1$.", "Russian", "ru"},
		{"Найдите все натуральные $n$, для которых число $n^2+1$ делится на $n+1$.", "", ""},
		{"求所有正整数 $n$，使得 $n^2+1$ 整除 $n!$。", "Chinese", "zh"},
		{"$x^2+y^2=z^2$, $x,y,z \\in \\mathbb{Z}$?", "en", "en"},
		{"Let $ABC$ be a triangle and let $M$ be the midpoint of $BC$. Prove that the lines are concurrent.", "Spanish", "en"},
	}
	for _, c := range cases {
		if got := Lang(c.text, c.named); got != c.want {
			t.Errorf("Lang(%q, %q) = %q, want %q", c.text, c.named, got, c.want)
		}
	}
}

const (
	tri  = "Let $ABC$ be an acute triangle with altitude $AD$ and orthocenter $H$. The line through $C$ parallel to $AB$ meets the perpendicular bisector of $AD$ at $G$. Show that $AC = BC$ if and only if $\\angle AGC = 90^{\\circ}$."
	tri2 = "Let $ABC$ be an acute triangle with altitude $AD$ and orthocenter $H$. The line through $C$ parallel to $AB$ meets the perpendicular bisector of $AD$ at $G$. Prove that $AC = BC$ if and only if $\\angle AGC = 90^{\\circ}$."
	fn   = "Find all functions $f:\\mathbb{R}\\to\\mathbb{R}$ such that $f(x+y)+f(xy)=f(x)f(y)+1$ for all real numbers $x$ and $y$."
	nt   = "Find the least positive integer $n$ for which there exist $n$ distinct positive integers $s_1, \\ldots, s_n$ with $\\prod (1 - 1/s_i) = 51/2010$."
	nt2  = "Determine the smallest positive integer $n$ such that one can choose $n$ distinct positive integers $s_1, \\ldots, s_n$ with $\\prod (1 - 1/s_i) = 51/2010$."
)

// filler gives n texts that share no rare word with the texts of the test, so that the weights of the words are as in a large corpus.
func filler(n int) []Doc {
	var out []Doc
	for i := range n {
		w := strings.Repeat(string(rune('a'+i%26)), 2+i/26)
		out = append(out, Doc{Text: "A board of chess with coins and tokens named " + w + " is colored in many ways; count the colorings of the board."})
	}
	return out
}

func TestDecide(t *testing.T) {
	problems := append([]Doc{{Text: tri, Answer: ""}, {Text: fn, Answer: "f(x)=x+1"}}, filler(40)...)
	rows := []Doc{
		{Text: tri2},                     // 0: the same as problem 0
		{Text: nt, Answer: "39"},         // 1: new
		{Text: nt2, Answer: "39"},        // 2: the same problem as row 1, in other words
		{Text: fn, Answer: "f(x)=x^2+1"}, // 3: the text of problem 1 with another answer
		{Text: nt},                       // 4: the same text as row 1
		{Text: "Prove that there are infinitely many primes $p$ such that $p$ divides $2^{2^n}+1$ for some $n$."}, // 5: new
	}
	ds := Decide(problems, rows, Options{Neighbors: 8})
	want := []struct {
		k      Kind
		target int
	}{{Link, 0}, {New, 1}, {Join, 1}, {Hold, 1}, {Join, 1}, {New, 5}}
	for i, w := range want {
		if ds[i].Kind != w.k || ds[i].Target != w.target {
			t.Errorf("row %d: %v %d (%+v), want %v %d", i, ds[i].Kind, ds[i].Target, ds[i].Signals, w.k, w.target)
		}
	}
	if !ds[4].Exact {
		t.Errorf("row 4 is not an exact match: %+v", ds[4])
	}
	ds = Decide(problems, rows, Options{Neighbors: 8, Limit: 1})
	if ds[0].Kind != Link || ds[1].Kind != New || ds[2].Kind != Skip || ds[5].Kind != Skip {
		t.Errorf("with a limit of 1: %v %v %v", ds[1].Kind, ds[2].Kind, ds[5].Kind)
	}
}

func TestSignals(t *testing.T) {
	cases := []struct {
		s     Signals
		same  bool
		doubt string
	}{
		{Signals{T: 0.85}, true, ""},
		{Signals{T: 0.85, A: 2}, false, "answer mismatch"},
		{Signals{E: 0.85, F: 0.75, Tokens: 5, Shared: 10}, true, ""},
		{Signals{E: 0.6, F: 0.75, T: 0.1, Tokens: 5, Shared: 10}, false, "E 0.60 and keyword match 0.75"},
		{Signals{E: 0.97, F: 1, Tokens: 2, Shared: 3}, true, ""},
		{Signals{E: 0.6, F: 0.6, Tokens: 5, Shared: 10}, false, "E 0.60 and keyword match 0.60"},
		{Signals{E: 0.85, F: 0.75, Tokens: 2, Shared: 10}, false, "E 0.85"},
		{Signals{E: 0.75}, false, "E 0.75"},
		{Signals{E: 0.55, F: 0.4, Tokens: 4}, false, "E 0.55 and keyword match 0.40"},
		{Signals{E: 0.55, F: 0.2, Tokens: 4}, false, ""},
	}
	for _, c := range cases {
		if c.s.Same() != c.same || c.s.Doubt() != c.doubt {
			t.Errorf("%+v: same %v doubt %q, want %v %q", c.s, c.s.Same(), c.s.Doubt(), c.same, c.doubt)
		}
	}
}

func TestMatchText(t *testing.T) {
	cases := [][2]string{
		{"In triangle $A B C, A B=2$ and $x_{1} \\geq 2$, radius $2$.", "In triangle $ABC,AB=2$ and $x_1\\ge2$, radius 2."},
		{"Find $\\left(\\dfrac{1}{2}\\right)$.\n\n*Proposed by John Cuya, Peru*", "Find $(\\frac{1}{2})$.\n"},
	}
	for _, c := range cases {
		if got := matchText(c[0]); got != c[1] {
			t.Errorf("matchText(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}
