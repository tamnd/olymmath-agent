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
		{"bold marks in a line", "Compute the remainder when $(10!)^{20}$ is divided by $2024$ .**p14.** A right square pyramid with height $12$ is inscribed in a sphere $S$.", "", Glued},
		{"next number", "3. Prove that $n^2+n$ is even for every integer $n$ and every prime.\n4. Find all real $x$ with $x^2=2x$.", "", Glued},
		{"answer under a heading", "# 3. Answer: $\\frac{2}{5}$ or $\\frac{11}{40}$ for the two cases of the problem.", "", Solution},
		{"a step of a solution", "Since the right side of the equation $(x+y)^{2}=49(3 x+5 y)$ is divisible by 49, then $x+y=7 k$.", "", Solution},
		{"by the condition", "By the condition, $a_{k}=\\frac{120-a_{k}}{k+1}$ for all $k=1,2, \\ldots, n-1$, so we find $n$.", "", Solution},
		{"a step after a number", "5. Since the expressions on the left and right are even functions, it is sufficient to consider the case $x \\geqslant 0$.", "", Solution},
		{"shortlist after a number", "8. A2 (GBR) Let $a, b, c$ be positive integers satisfying the conditions $b>2 a$ and $c>2 b$. Show that there exists a real number $t$.", "", Shortlist},
		{"steps of a solution", "Consider the equation as a quadratic in terms of $a$. Then its discriminant is equal to $x^{2}(49-40 x^{2})$, therefore $x=0$ or $x= \\pm 1$.", "", Solution},
		{"a formula lost before a sentence", "Positive numbers $ a $, $ b $, $ c $, $ d $ satisfy the system of equations\n\nProve that $ a = b = c = d $.", "", Lost},
		{"economics", "In a perfectly competitive market, the demand function for a good is $Q_d(p)=150-p$. Find the equilibrium price.", "", Economics},
		{"a claim lost at the end", "Positive numbers $ a, b, c $ satisfy the condition $ ab+bc+ca = abc $. Prove that", "", Lost},
		{"a list lost after a colon", "Point $ P $ lies inside triangle $ ABC $ and satisfies the conditions:  #", "", Lost},
		{"a formula lost after and", "In a convex hexagon $ ABCDEF $ all sides are of equal length and\n\nProve that the diagonals $ AD $, $ BE $ and $ CF $ meet at one point.", "", Lost},
		{"a financial literacy task", "Full score -8\n\nVasya took a loan of 100,000 rubles from a financial institution for three years. How much will he pay back?", "", Economics},
		{"a task of informatics", "Text of the task. At a polar station, the decryption device has malfunctioned. Find the number of messages that can still be read.", "", Informatics},
		{"a problem statement in words", "Find the least $n$ for which the problem statement holds for every set of $n$ points.", "Find the least $n$ for which the problem statement holds for every set of $n$ points.", ""},
		{"density", "(15 points) A wooden cube with edge 30 cm floats in a lake. The density of wood is 750 kg/m^3. Find the depth of the cube under the water.", "", Science},
		{"a sales tax", "A coat costs 100 dollars and is put on sale for 25% off. If a 10% sales tax is added to the sale price, what is the total price?", "A coat costs 100 dollars and is put on sale for 25% off. If a 10% sales tax is added to the sale price, what is the total price?", ""},
		{"a prompt left at the end", "Find the first 2000 digits after the decimal point of the number $(9+4 \\sqrt{5})^{2021}$.\n\nTranslate the text above into English, please keep the original text's line breaks and format, and output the translation result directly.", "Find the first 2000 digits after the decimal point of the number $(9+4 \\sqrt{5})^{2021}$.", ""},
		{"template", "Given a regular param1. In how many ways can three of its vertices be chosen so that they form a scalene triangle?", "", Template},
		{"lost formula", "Prove that if the equation\n\nhas three real roots, then\n\nIs the converse statement true?", "", Lost},
		{"translator note", "Solve in integers the equation\n\nNote: The equation is not provided in the original text, so it is not included.", "", Lost},
		{"physics", "(15 points) Two identical resistors $R_{0}$ are connected in series to a source of constant voltage. Find the power.", "", Science},
		{"a club of physics", "The robotics club accepts only those who know mathematics, physics, or programming. How many members know all three?", "The robotics club accepts only those who know mathematics, physics, or programming. How many members know all three?", ""},
		{"shortlist country", "N7 (GBR) A wobbly number is a positive integer whose digits are alternately nonzero and zero. Find all positive integers that do not divide any wobbly number.", "", Shortlist},
		{"imo note", "(GDR 2) ${ }^{\\mathrm{IMO} 5}$ Is it possible to put 1987 points in the plane such that the distance between each pair of points is irrational?", "", Shortlist},
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
		{"a heading that starts the problem", "# 3. Solve the system of equations\n\n$$\nx^{5}+3 x^{4}=2, \\quad x^{3}+x=1\n$$", "Solve the system of equations\n\n$$\nx^{5}+3 x^{4}=2, x^{3}+x=1\n$$", ""},
		{"a heading of two words", "# 3. In the equation\n\n$$\nx^{2}+p x+q=0\n$$\n\nthe coefficients $p$ and $q$ are odd integers. Prove that it has no rational root.", "In the equation\n\n$$\nx^{2}+p x+q=0\n$$\n\nthe coefficients $p$ and $q$ are odd integers. Prove that it has no rational root.", ""},
		{"a title", "## 26. Gold Coins\n\nThe king is waiting for each of his 30 vassals to present him with 30 gold coins. How many coins does he get?", "The king is waiting for each of his 30 vassals to present him with 30 gold coins. How many coins does he get?", ""},
		{"points first", "(3 points) Find the distance between the curves $y=e^{3 x+5}$ and $y=(\\ln x-5) / 3$.", "Find the distance between the curves $y=e^{3 x+5}$ and $y=(\\ln x-5) / 3$.", ""},
		{"points after the number", "1. [4 points] Find the number of eight-digit numbers, the product of whose digits equals 700.", "Find the number of eight-digit numbers, the product of whose digits equals 700.", ""},
		{"points and condition lines", "# Problem 5. (3 points)\n\n## Condition:\n\nIn triangle $ABC$, $AB = 3 \\cdot BC$ and $M$ is the midpoint of $AB$. Find the angle $MDB$.\n\n(10 points)", "In triangle $ABC$, $AB = 3 \\cdot BC$ and $M$ is the midpoint of $AB$. Find the angle $MDB$.", ""},
		{"a section number", "10.2 Plot on the coordinate plane the set of points satisfying the equation $2 x^{2}+3 x y+y^{2}+x=1$.", "Plot on the coordinate plane the set of points satisfying the equation $2 x^{2}+3 x y+y^{2}+x=1$.", ""},
		{"a lone hash at the end", "What is the maximum number of sides a polygon can have if each of its angles is either $172^{\\circ}$ or $173^{\\circ}$?\n\n#", "What is the maximum number of sides a polygon can have if each of its angles is either $172^{\\circ}$ or $173^{\\circ}$?", ""},
		{"the form of the answer", "Find the sum of all natural numbers $n$ with $n^2 < 2024$ that are divisible by 7.\n\nThe answer to the task should be an integer.", "Find the sum of all natural numbers $n$ with $n^2 < 2024$ that are divisible by 7.", ""},
		{"points for the problem", "Find the sum of all natural numbers $n$ with $n^2 < 2024$ that are divisible by 7.\n\nPoints for the problem: 13.", "Find the sum of all natural numbers $n$ with $n^2 < 2024$ that are divisible by 7.", ""},
		{"one variant", "## Variant 1\n\nFind the sum of all natural numbers $n$ with $n^2 < 2024$ that are divisible by 7.", "Find the sum of all natural numbers $n$ with $n^2 < 2024$ that are divisible by 7.", ""},
		{"two variants", "## Variant 1\n\nFind the sum of all natural numbers $n$ with $n^2 < 2024$.\n\n## Variant 2\n\nFind the sum of all natural numbers $n$ with $n^2 < 2025$.", "", Glued},
		{"grading", "Find all real $x$ with $x^2 = 2x + 3$ and $x > 0$.\n\nEvaluation criteria: 7 points for a full solution.", "", Grading},
		{"a note on the form", "Find all real $x$ with $x^2 = 2x + 3$ and $x > 0$.\n\n---\n\nNote: The translation preserves the original text's line breaks and formatting.", "Find all real $x$ with $x^2 = 2x + 3$ and $x > 0$.", ""},
		{"a note on a lost part", "Find all real $x$ with $x^2 = 2x + 3$ and $x > 0$.\n\nNote: The translation is provided as requested. However, the mathematical expressions are not complete.", "", Grading},
		{"a tournament", "In a chess tournament, 29 people are playing. One point is awarded for a win, half a point for a draw, and zero for a loss. How many points can the winner have?", "In a chess tournament, 29 people are playing. One point is awarded for a win, half a point for a draw, and zero for a loss. How many points can the winner have?", ""},
		{"an example heading", "## Example 5. Given the function\n\n$$\nf(x)=c e^{-x^{2}}\n$$\n\nFor what value of the constant $c$ is the function $f(x)$ a probability density?", "Given the function\n\n$$\nf(x)=c e^{-x^{2}}\n$$\n\nFor what value of the constant $c$ is the function $f(x)$ a probability density?", ""},
		{"a mark before a formula", "## Problem 3\n\n$D$ is a point inside the equilateral triangle $ABC$ and $E$ is a point inside $DBC$. Show that $DB+DC>EB+EC$.", "$D$ is a point inside the equilateral triangle $ABC$ and $E$ is a point inside $DBC$. Show that $DB+DC>EB+EC$.", ""},
		{"a title before a capital", "## On the Café Terrace\n\nOne beautiful spring evening, Dupont and Durand were playing dice. If the sum was 7, Durand won a point. On whom would you bet?", "One beautiful spring evening, Dupont and Durand were playing dice. If the sum was 7, Durand won a point. On whom would you bet?", ""},
		{"a prompt that preserves", "Find the first 2000 digits after the decimal point of the number $(9+4 \\sqrt{5})^{2021}$.\n\nTranslate the above text into English, preserving the original text's line breaks and format.", "Find the first 2000 digits after the decimal point of the number $(9+4 \\sqrt{5})^{2021}$.", ""},
		{"given twice", "Solve the system of inequalities:\n\n$$\n\\left\\{\\begin{array}{l} 13 x^{2}-4 x y+4 y^{2} \\leq 2 \\\\ 2 x-4 y \\leq-3 \\end{array}\\right.\n$$\n\nSolve the following system of inequalities:\n\n$$\n\\left\\{\\begin{array}{l} 13 x^{2}-4 x y+4 y^{2} \\leq 2 \\\\ 2 x-4 y \\leq-3 \\end{array}\\right.\n$$", "Solve the system of inequalities:\n\n$$\n\\left\\{\\begin{array}{l} 13 x^{2}-4 x y+4 y^{2} \\leq 2 \\\\ 2 x-4 y \\leq-3 \\end{array}\\right.\n$$", ""},
		{"two parts", "a) Prove that the number $n^{3}+2n$ is divisible by 3 for every integer $n$.\n\nb) Prove that the number $n^{5}+4n$ is divisible by 5 for every integer $n$.", "a) Prove that the number $n^{3}+2n$ is divisible by 3 for every integer $n$.\n\nb) Prove that the number $n^{5}+4n$ is divisible by 5 for every integer $n$.", ""},
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

func TestDecideNumbers(t *testing.T) {
	a := "Let $a_{n, m}$ be the number written with all numbers from $n$ down to $m$, where $n > m \\geq 1$. For example, $a_{4,2}=432$. Prove that one of these numbers is divisible by 2021."
	b := strings.Replace(a, "2021", "2022", 1)
	rows := []Doc{{Text: a}, {Text: b}, {Text: b}, {Text: a + " "}}
	ds := Decide(filler(40), rows, Options{Neighbors: 8})
	want := []Kind{New, Hold, Hold, Join}
	for i, w := range want {
		if ds[i].Kind != w {
			t.Errorf("row %d: %v (%+v), want %v", i, ds[i].Kind, ds[i], w)
		}
	}
	if ds[1].Reason != "other numbers" {
		t.Errorf("row 1: reason %q", ds[1].Reason)
	}
}

func TestDecideLinkNumbers(t *testing.T) {
	a := "Let $a_{n, m}$ be the number written with all numbers from $n$ down to $m$, where $n > m \\geq 1$. For example, $a_{4,2}=432$. Prove that one of these numbers is divisible by 2021."
	b := strings.Replace(a, "2021", "2022", 1)
	problems := append(filler(40), Doc{Text: a, Derived: true}, Doc{Text: a})
	ds := Decide(problems[:41], []Doc{{Text: b}, {Text: a}}, Options{Neighbors: 8})
	if ds[0].Kind != Hold || ds[0].Reason != "other numbers" {
		t.Errorf("a row with other numbers than a derived loose problem: %v %q, want a hold", ds[0].Kind, ds[0].Reason)
	}
	if ds[1].Kind != Link || ds[1].Target != 40 {
		t.Errorf("a row with the same numbers: %v %d, want a link to 40", ds[1].Kind, ds[1].Target)
	}
	// A problem that is not from a derived dataset keeps the old rule: the text signals decide.
	ds = Decide(append(filler(40), problems[41]), []Doc{{Text: b}}, Options{Neighbors: 8})
	if ds[0].Kind != Link {
		t.Errorf("a row near a placed problem: %v %q, want a link", ds[0].Kind, ds[0].Reason)
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
		{"What is the area of the union of the regions enclosed by the square and the circle?\n\n$(\\text{A})200+25\\pi(\\text{B})100+75\\pi$", "What is the area of the union of the regions enclosed by the square and the circle?"},
		{"What is the area of the union of the regions enclosed by the square and the circle? $(\\mathrm {A}) 200+25\\pi (\\mathrm {B}) 100+75\\pi$", "What is the area of the union of the regions enclosed by the square and the circle?"},
		{"What is the probability that he lands less than 1 unit away? $\\mathbf{(A)}\\frac{1}{6}\\mathbf{(B)}\\frac{1}{5}$", "What is the probability that he lands less than 1 unit away?"},
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
		{"$\\overline{ID} \\perp \\overline{BC}$, $\\angle{BAE} = \\measuredangle ACD$ and $\\triangle{CFE}$", "$ID\\perp BC$, $\\angle BAE=\\angle ACD$ and $\\triangle CFE$"},
		// A command keeps its space before a letter, an empty group is nothing, and \langle is <.
		{"$0\\le AB\\le 1$, $75_{}^{}$ and $\\langle n\\rangle$", "$0\\le AB\\le1$, 75 and $<n>$"},
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

func TestDecideWords(t *testing.T) {
	problems := filler(40)
	// AMC 12 2012A problem 21, as the content has it.
	problems[5].Text = alignDollars
	// The same problem with a bare align, other spacing and other choices.
	ds := Decide(problems, []Doc{{Text: alignBare}}, Options{Neighbors: 8})
	if ds[0].Kind != Link || ds[0].Target != 5 || !strings.HasPrefix(ds[0].Reason, "words") {
		t.Errorf("other formulas: %v %d (%s) %+v, want a link to 5 by the words", ds[0].Kind, ds[0].Target, ds[0].Reason, ds[0].Signals)
	}
	for _, c := range []Signals{{W: 1, M: 1, Words: 20, Numbers: true}, {W: 1, M: 1, O: 0.8, Words: 12, Numbers: true}} {
		if !c.byWords() {
			t.Errorf("%+v: not the same by the words", c)
		}
	}
	for _, c := range []Signals{{W: 1, M: 1, Words: 20}, {W: 1, M: 1, O: 1, Words: 8, Numbers: true}, {W: 1, M: 0.6, Words: 20, Numbers: true}, {W: 0.7, M: 1, Words: 20, Numbers: true}, {W: 1, M: 0.9, O: 0.6, Words: 12, Numbers: true}} {
		if c.byWords() {
			t.Errorf("%+v: the same by the words", c)
		}
	}
}

const (
	alignDollars = "Let $a$, $b$, and $c$ be positive integers with $a\\ge$ $b\\ge$ $c$ such that\n$$\n\\begin{align*}a^{2}-b^{2}-c^{2}+ab&=2011\\text{ and}\\\\\na^{2}+3b^{2}+3c^{2}-3ab-2ac-2bc&=-1997\n\\end{align*}\n$$\nWhat is $a$?\n\n\\textbf{(A)}\\ 249 \\textbf{(B)}\\ 250 \\textbf{(C)}\\ 251 \\textbf{(D)}\\ 252 \\textbf{(E)}\\ 253"
	alignBare    = "Let $a,b,$ and $c$ be positive integers with $a\\ge b\\ge c$ such that \\begin{align*} a^2-b^2-c^2+ab&=2011\\text{ and}\\\\ a^2+3b^2+3c^2-3ab-2ac-2bc&=-1997\\end{align*} What is $a$ ? $ \\textbf{(A)}\\ 249 \\qquad\\textbf{(B)}\\ 250 \\qquad\\textbf{(C)}\\ 251 \\qquad\\textbf{(D)}\\ 252 \\qquad\\textbf{(E)}\\ 253$"
)

func TestShape(t *testing.T) {
	a, b := shapeOf(matchText(alignBare)), shapeOf(matchText(alignDollars))
	if w, m := bagJaccard(a.words, b.words), bagJaccard(a.formulas, b.formulas); w < WordSame || m < FormulaSame || a.numbers != b.numbers {
		t.Errorf("a bare align and one in dollars: words %.2f, formulas %.2f, numbers %q and %q", w, m, a.numbers, b.numbers)
	}
	if a.n < MinWords {
		t.Errorf("words: %d", a.n)
	}
	// PAMO 2013 problem 2 and another equation in the same words: the same letters in another order.
	f, g := shapeOf(matchText("Find all functions $f:\\mathbb{R}\\to\\mathbb{R}$ such that $$f(xy)=\\max\\{f(x+y),f(x) f(y)\\} $$ for all real numbers $x$ and $y$ .")), shapeOf(matchText("Find all functions $f:\\mathbb{R}\\to\\mathbb{R}$ such that $f(x)f(y)+f(x+y)=xy$ for all real numbers $x$ and $y$ ."))
	s := Signals{W: bagJaccard(f.words, g.words), M: bagJaccard(f.formulas, g.formulas), O: bagJaccard(f.order, g.order), Words: min(f.n, g.n), Numbers: f.numbers == g.numbers}
	if s.byWords() {
		t.Errorf("two functional equations: the same by the words, %+v", s)
	}
	// Two texts with no formulas have the same formulas.
	if m := bagJaccard(map[string]int{}, map[string]int{}); m != 1 {
		t.Errorf("two empty multisets: %.2f, want 1", m)
	}
}

func TestPolishMark(t *testing.T) {
	cases := []struct {
		in    string
		ok    bool
		want  Mark
		after string
	}{
		{"XXXVIII OM - I - Problem 11\n\nProve that a sphere can be lit.", true, Mark{38, "I", 11, "XXXVIII OM - I - Problem 11"}, "Prove that a sphere can be lit."},
		{"XXV - I - Problem 3  Prove that the bisectors are parallel.", true, Mark{25, "I", 3, "XXV - I - Problem 3"}, "Prove that the bisectors are parallel."},
		{"LIX OM - III -Zadanie 2\n\nFunkcja $f$ trzech zmiennych.", true, Mark{59, "III", 2, "LIX OM - III -Zadanie 2"}, "Funkcja $f$ trzech zmiennych."},
		{"I OM - B - Task 4\n\nFind two natural numbers.", true, Mark{1, "B", 4, "I OM - B - Task 4"}, "Find two natural numbers."},
		{"IIII OM - I - Problem 1 Find $x$.", false, Mark{}, ""},
		{"Find all functions $f$ such that $f(x)=x$.", false, Mark{}, ""},
	}
	for _, c := range cases {
		m, after, ok := PolishMark(c.in)
		if ok != c.ok || (ok && (m != c.want || after != c.after)) {
			t.Errorf("PolishMark(%q) = %+v, %q, %v", c.in, m, after, ok)
		}
	}
	if y := (Mark{Olympiad: 41}).Year(); y != 1990 {
		t.Errorf("year of the 41st olympiad: %d", y)
	}
}
