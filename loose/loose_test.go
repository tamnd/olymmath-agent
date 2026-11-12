package loose

import (
	"slices"
	"strings"
	"testing"

	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/similar"
)

func TestClean(t *testing.T) {
	cases := []struct {
		name, in, want, why string
	}{
		{"plain", "Let $ABC$ be an acute triangle with altitude $AD$. Show that $AC = BC$ if and only if $\\angle AGC = 90^{\\circ}$.", "Let $ABC$ be an acute triangle with altitude $AD$. Show that $AC = BC$ if and only if $\\angle AGC = 90^{\\circ}$.", ""},
		{"lead mark", "Problem 1. Find all prime numbers $p$ for which $x^p+y^p+z^p-x-y-z$ is a product of exactly three distinct primes.", "Find all prime numbers $p$ for which $x^p+y^p+z^p-x-y-z$ is a product of exactly three distinct primes.", ""},
		{"heading", "# Task 3.\n\nVintik drew a rectangle $9 \\times 11$ cm and marked two hundred points in it. Prove that some square contains three points.", "Vintik drew a rectangle $9 \\times 11$ cm and marked two hundred points in it. Prove that some square contains three points.", ""},
		{"proof stands for prove", "Given positive odd number $m$ and integer ${a}.$\nProof: For any real number $c,$ the set has at most $2+\\log_2 m$ elements.", "", NotStated},
		{"a quick solution", "A quick solution:\nLet R be the foot of the perpendicular from X to BC. Then P, R, Q are collinear by the Simson line.", "", Solution},
		{"forum news", "The national olympiad is taking place this week and I am part of the big event. The subjects will be posted on the site soon, as for the contest the subjects will be posted on monday.", "", NotStated},
		{"claim with no formula", "If a triangle has two equal angles then it has two equal sides, and the converse holds too.", "If a triangle has two equal angles then it has two equal sides, and the converse holds too.", ""},
		{"the solution of", "The solution of the equation $7^{x+7}=8^x$ can be expressed in the form $x=\\log_b 7^7$. What is $b$?", "The solution of the equation $7^{x+7}=8^x$ can be expressed in the form $x=\\log_b 7^7$. What is $b$?", ""},
		{"construction", "Restore a triangle by one of its vertices, the circumcenter and the Lemoine point of the triangle.", "Restore a triangle by one of its vertices, the circumcenter and the Lemoine point of the triangle.", ""},
		{"a puzzle told by I", "I have five different pairs of socks. Every day for five days, I pick two socks at random. Find the probability that I wear a matching pair on the fifth day.", "I have five different pairs of socks. Every day for five days, I pick two socks at random. Find the probability that I wear a matching pair on the fifth day.", ""},
		{"answer cut", "3. Five runners ran a relay. If the first runner ran twice as fast, they would have spent $5\\%$ less time. By what percentage less time would they have spent?\n\nOTBET: by $8\\%$", "Five runners ran a relay. If the first runner ran twice as fast, they would have spent $5\\%$ less time. By what percentage less time would they have spent?", ""},
		{"too short", "Find $x$ if $x+1=2$.", "", TooShort},
		{"glued number", "3-There is given a trapezoid $ABCD$ in the plane with $BC\\parallel AD$. Prove that $AB=CD$.", "There is given a trapezoid $ABCD$ in the plane with $BC\\parallel AD$. Prove that $AB=CD$.", ""},
		{"glued dot", "1.Let $ABC$ be a triangle and $D$ the foot of the altitude from $A$. Prove that $BD<AB$.", "Let $ABC$ be a triangle and $D$ the foot of the altitude from $A$. Prove that $BD<AB$.", ""},
		{"a number in the text", "2-colorings of a $3\\times 3$ board are counted up to rotation. How many are there in total?", "2-colorings of a $3\\times 3$ board are counted up to rotation. How many are there in total?", ""},
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

func TestDecideEdition(t *testing.T) {
	ed := []string{"x/2008"}
	problems := append([]Doc{{Text: tri, ID: "x/2008/p1", Editions: ed}, {Text: fn, ID: "x/2008/p2", Editions: ed}}, filler(40)...)
	renamed := "Find all functions $g:\\mathbb{R}\\to\\mathbb{R}$ with $g(a+b)+g(ab)=g(a)g(b)+1$ for all reals $a$ and $b$."
	rows := []Doc{
		{Text: renamed, Editions: ed}, // 0: problem 1 with other names, in the edition
		{Text: renamed},               // 1: the same text with no edition
		{Text: nt, Editions: ed},      // 2: not a problem of the edition
	}
	ds := Decide(problems, rows, Options{Neighbors: 8})
	// With other names T and the keyword match are low, so E alone holds the row and does not link it.
	if ds[0].Kind != Hold || ds[0].Target != 1 {
		t.Errorf("row 0: %v %d %q (%+v), want a hold on 1", ds[0].Kind, ds[0].Target, ds[0].Reason, ds[0].Signals)
	}
	if ds[1].Edition {
		t.Errorf("row 1 has no edition but links by edition: %+v", ds[1])
	}
	if ds[2].Kind != New {
		t.Errorf("row 2: %v %d %q, want new", ds[2].Kind, ds[2].Target, ds[2].Reason)
	}
}

func TestDecideChoices(t *testing.T) {
	choices := " $\\textbf{(A)}\\ 3 \\qquad\\textbf{(B)}\\ 4 \\qquad\\textbf{(C)}\\ 5 \\qquad\\textbf{(D)}\\ 6 \\qquad\\textbf{(E)}\\ 7$"
	cube := "A wooden cube $n$ units on a side is painted red on all six faces and then cut into $n^3$ unit cubes. Exactly one-fourth of the total number of faces of the unit cubes are red. What is $n$?"
	digits := "Let $n$ be the smallest positive integer such that $n$ is divisible by $20$, $n^2$ is a perfect cube, and $n^3$ is a perfect square. What is the number of digits of $n$?"
	problems := append([]Doc{{Text: cube + choices}}, filler(40)...)
	rows := []Doc{
		{Text: digits + choices}, // 0: another problem with the same choices
		{Text: strings.Replace(cube, "  ", " ", 1) + " " + strings.ReplaceAll(choices, "\\qquad", "")}, // 1: the problem with other spacing
	}
	area := "What is the area of the region enclosed by the graph of the equation $x^2+y^2=|x|+|y|?$"
	problems = append(problems, Doc{Text: area + "\n\n\\textbf{(A)}\\ \\pi+\\sqrt{2} \\textbf{(B)}\\ \\pi+2 \\textbf{(C)}\\ \\pi+2\\sqrt{2} \\textbf{(D)}\\ 2\\pi+\\sqrt{2} \\textbf{(E)}\\ 2\\pi+2\\sqrt{2}"})
	// 2: the same problem with its choices in a formula
	rows = append(rows, Doc{Text: area + " $\\textbf{(A)}\\ \\pi+\\sqrt{2} \\textbf{(B)}\\ \\pi+2 \\textbf{(C)}\\ \\pi+2\\sqrt{2} \\textbf{(D)}\\ 2\\pi+\\sqrt{2} \\textbf{(E)}\\ 2\\pi+2\\sqrt{2}$"})
	// 3: a short stem that many problems share, with other choices
	largest := "Find the largest number among the following numbers: "
	problems = append(problems, Doc{Text: largest + "$ \\textbf{(A) }30^{30} \\textbf{(B) }50^{10} \\textbf{(C) }40^{20} \\textbf{(D) }45^{15} \\textbf{(E) }5^{60}$"})
	rows = append(rows, Doc{Text: largest + "$ \\textbf{(A) }\\tan47^{\\circ}+\\cos47^{\\circ} \\textbf{(B) }\\cot 47^{\\circ}+\\sqrt{2}\\sin 47^{\\circ} \\textbf{(C) }\\sqrt{2}\\cos47^{\\circ}+\\sin47^{\\circ} \\textbf{(D) }\\tan47^{\\circ}+\\cot47^{\\circ} \\textbf{(E) }\\cos47^{\\circ}+\\sqrt{2}\\sin47^{\\circ}$"})
	ds := Decide(problems, rows, Options{Neighbors: 8})
	if ds[3].Kind == Link || ds[3].Kind == Join {
		t.Errorf("row 3 is placed on its stem alone: %v %d (%+v)", ds[3].Kind, ds[3].Target, ds[3].Signals)
	}
	if ds[0].Kind == Link {
		t.Errorf("row 0 links on its choices: %+v", ds[0])
	}
	if ds[2].Kind != Link || ds[2].Target != len(problems)-2 {
		t.Errorf("row 2: %v %d (%+v), want a link to %d", ds[2].Kind, ds[2].Target, ds[2].Signals, len(problems)-2)
	}
	if ds[1].Kind != Link || ds[1].Target != 0 {
		t.Errorf("row 1: %v %d (%+v), want a link to 0", ds[1].Kind, ds[1].Target, ds[1].Signals)
	}
}

func TestDecideSibling(t *testing.T) {
	square := `Baron Munchausen discovered the following theorem: "For any positive integers $a$ and $b$ there exists a positive integer $n$ such that $an$ is a perfect square, while $bn$ is a perfect cube". Determine if the statement of Baron's theorem is correct.`
	cube := strings.Replace(strings.Replace(square, "perfect cube", "perfect fifth power", 1), "perfect square", "perfect cube", 1)
	ed := []string{"caucasus/2018"}
	problems := filler(40)
	cases := []struct {
		name     string
		problems []Doc
		rows     []Doc
		want     Kind
	}{
		{"two rows of one edition", problems, []Doc{{Text: square, Editions: ed}, {Text: cube, Editions: ed}}, Hold},
		{"two rows of no edition", problems, []Doc{{Text: square}, {Text: cube}}, Join},
		{"a row and a loose problem of its edition", append([]Doc{{Text: square, Named: ed}}, problems...), []Doc{{Text: filler(41)[40].Text}, {Text: cube, Editions: ed}}, Hold},
		{"the same text twice in one edition", problems, []Doc{{Text: square, Editions: ed}, {Text: square + " ", Editions: ed}}, Join},
	}
	for _, c := range cases {
		ds := Decide(c.problems, c.rows, Options{Neighbors: 8})
		if ds[1].Kind != c.want {
			t.Errorf("%s: %v (%+v), want %v", c.name, ds[1].Kind, ds[1].Signals, c.want)
		}
	}
}

func TestSimText(t *testing.T) {
	stem := "What number is one third of the way from $\\frac14$ to $\\frac34$?"
	math := stem + " $\\textbf{(A)}\\ \\frac{1}{3} \\textbf{(B)}\\ \\frac{5}{12} \\textbf{(C)}\\ \\frac{1}{2}$"
	plain := stem + "\n\n\\textbf{(A)}\\ \\frac {1}{3} \\textbf{(B)}\\ \\frac {5}{12} \\textbf{(C)}\\ \\frac {1}{2}"
	a, b := similar.Words(simText(matchText(math))), similar.Words(simText(matchText(plain)))
	if !slices.Equal(a, b) {
		t.Errorf("the choices count for E:\n%v\n%v", a, b)
	}
}

func TestSameEdition(t *testing.T) {
	floors := Signals{Tokens: 5, Shared: 10}
	cases := []struct {
		best, next   float64
		s            Signals
		same, unsure bool
	}{
		{0.7, 0.2, floors, true, false},
		{0.7, 0.6, floors, false, true},
		{0.4, 0.1, floors, false, true},
		{0.2, 0.1, floors, false, false},
		{0.9, 0.1, Signals{Tokens: 2, Shared: 10}, false, false},
		{0.9, 0.1, Signals{Tokens: 5, Shared: 10, A: match.AnswerMismatch}, false, true},
	}
	for _, c := range cases {
		same, unsure := sameEdition(c.best, c.next, c.s)
		if same != c.same || unsure != c.unsure {
			t.Errorf("%v %v %+v: %v %v, want %v %v", c.best, c.next, c.s, same, unsure, c.same, c.unsure)
		}
	}
	if !sameEditionText(0.6, 0.3, Signals{T: 0.4}) || !sameEditionText(0.6, 0.3, Signals{F: 0.6}) || !sameEditionText(0.85, 0.3, Signals{}) || sameEditionText(0.6, 0.3, Signals{T: 0.1, F: 0.2}) || sameEditionText(0.6, 0.5, Signals{T: 0.4}) || sameEditionText(0.45, 0.1, Signals{T: 0.4}) || sameEditionText(0.9, 0.1, Signals{A: match.AnswerMismatch}) {
		t.Errorf("sameEditionText does not keep its bars")
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

func TestStem(t *testing.T) {
	cases := [][2]string{
		{"What is the number of digits of the least such $n$? $\\textbf{(A)}3\\textbf{(B)}4\\textbf{(C)}5$", "What is the number of digits of the least such $n$?"},
		{"Find $x$ such that $x^2=4$ and $x>0$, and prove it.", "Find $x$ such that $x^2=4$ and $x>0$, and prove it."},
		{"How many triangles? \\mathrm{(A)}1", "How many triangles? \\mathrm{(A)}1"},
		{"What is the area of the region enclosed by the graph of $x^2+y^2=|x|+|y|?$\n\n\\textbf{(A)}\\ \\pi \\textbf{(B)}\\ 2", "What is the area of the region enclosed by the graph of $x^2+y^2=|x|+|y|?$"},
		{"What is the area of the region enclosed by the graph of $x^2+y^2=|x|+|y|?$ $\\textbf{(A)}\\ \\pi \\textbf{(B)}\\ 2$", "What is the area of the region enclosed by the graph of $x^2+y^2=|x|+|y|?$"},
	}
	for _, c := range cases {
		if got := stem(c[0]); got != c[1] {
			t.Errorf("stem(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestMatchText(t *testing.T) {
	cases := [][2]string{
		{"In triangle $A B C, A B=2$ and $x_{1} \\geq 2$, radius $2$.", "In triangle $ABC,AB=2$ and $x_1\\ge2$, radius 2."},
		{"Find $\\left(\\dfrac{1}{2}\\right)$.\n\n*Proposed by John Cuya, Peru*", "Find $(\\frac{1}{2})$.\n"},
		{"$\\overline{ID} \\perp \\overline{BC}$, $\\angle{BAE} = \\measuredangle ACD$ and $\\triangle{CFE}$", "$ID\\perpBC$, $\\angleBAE=\\angleACD$ and $\\triangleCFE$"},
	}
	for _, c := range cases {
		if got := matchText(c[0]); got != c[1] {
			t.Errorf("matchText(%q) = %q, want %q", c[0], got, c[1])
		}
	}
}

func TestDecideComplete(t *testing.T) {
	problems := filler(40)
	problems[3].Editions = []string{"imo/1978"}
	// IMO 1978 problem 2 in other words, which the text match does not find.
	row := Doc{Text: "We consider a fixed point $P$ in the interior of a fixed sphere. We construct three segments $PA, PB, PC$, perpendicular two by two, with the vertexes $A, B, C$ on the sphere. Find the locus of the vertex $Q$ opposite to $P$ in the parallelepiped with $PA, PB, PC$ as edges.", Editions: []string{"imo/1978"}}
	if ds := Decide(problems, []Doc{row}, Options{Neighbors: 8}); ds[0].Kind != New {
		t.Fatalf("an edition that is not complete: %v, want new", ds[0].Kind)
	}
	ds := Decide(problems, []Doc{row}, Options{Neighbors: 8, Complete: map[string]bool{"imo/1978": true}})
	if ds[0].Kind != Hold || ds[0].Target != 3 {
		t.Errorf("a complete edition: %v %d (%s), want a hold with 3", ds[0].Kind, ds[0].Target, ds[0].Reason)
	}
}
