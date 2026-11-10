package canon

import "testing"

func TestCleanHead(t *testing.T) {
	cases := []struct{ in, want string }{
		{"Bogdanov I.I.\n\nThe distance between two cells is 100.", "The distance between two cells is 100."},
		{"Golovanov A.S. $^{2}$\n\nGiven three reduced quadratic trinomials.", "Given three reduced quadratic trinomials."},
		{"Sonkin $M$.\n\nLet the incircle of triangle $ABC$ touch its sides.", "Let the incircle of triangle $ABC$ touch its sides."},
		{"A.\n\nA natural number is written on the board.", "A natural number is written on the board."},
		{"$\\underline{\\text { Capacity }}$..\n\nOn a line, 100 sets are chosen.", "On a line, 100 sets are chosen."},
		{"[ Two tangents drawn from one point ]\n\nThe centers $O_1, O_2$ of three circles.", "The centers $O_1, O_2$ of three circles."},
		{"Coloring\n[ Geometry on grid paper\n\nA grid plane is colored.", "A grid plane is colored."},
		{"10,11 [Dirichlet's Principle (finite number of points, lines, etc.).]\n\nAuthor: Akoopn A.v.\n\nCan 12 boxes be placed?", "Can 12 boxes be placed?"},
		{"Partitioning into pairs and groups; bijections\n\nAuthors: Bogdanov I.i., von der Flaass D: On a plane, a square is drawn.", "On a plane, a square is drawn."},
		{"Berlov S.L. On the arcs $AB$ and $BC$ points are chosen.", "On the arcs $AB$ and $BC$ points are chosen."},
		{"11.4. On each of 2013 cards, a number is written.", "On each of 2013 cards, a number is written."},
		{"11.1 For the tetrahedron $SABC$, the incircle touches.", "For the tetrahedron $SABC$, the incircle touches."},
		{"Problem 1. (4 points) Given is a convex 100-gon.", "Given is a convex 100-gon."},
		{"18.6.3 ${ }^{\\star}$ For an integer $n>3$, solve it.", "For an integer $n>3$, solve it."},
		{"депчинн В.A. The test consists of 30 questions.", "The test consists of 30 questions."},
		{"In triangle $ABC$, find the ratio $OM: PC$. Answer: 1:2.", "In triangle $ABC$, find the ratio $OM: PC$."},
		// Texts with no head stay as they are.
		{"Solve the equation.\n$$x^2=2$$", "Solve the equation.\n$$x^2=2$$"},
		{"The number\n$$2^{10}$$\nis even. Prove it.", "The number\n$$2^{10}$$\nis even. Prove it."},
		{"2013 cards lie on a table. Prove that one is red.", "2013 cards lie on a table. Prove that one is red."},
		{"Point A. Then the line is drawn.", "Point A. Then the line is drawn."},
	}
	for _, c := range cases {
		if got := CleanHead(c.in); got != c.want {
			t.Errorf("CleanHead(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
