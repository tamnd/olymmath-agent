package asy

import (
	"math"
	"strings"
	"testing"
)

func near(a, b Pair) bool { return a.sub(b).abs() < 1e-3 }

func TestRun(t *testing.T) {
	src := `size(100); defaultpen(linewidth(0.8)+fontsize(10));
pair A=(0,0), B=(4,0), C=(0,3);
path c = circle((2,0), 2);
pair P = intersectionpoints(c, A--C+(1,0))[0];
draw(A--B--C--cycle); dot("$A$", A, SW); label("$5$", B--C, NE);
for (int i = 1; i < 3; ++i) { dot((i,0)); }
real f(real x) { return 2x; }
draw((f(1),1)..(3,2)..(4,1), dashed);`
	pic, err := Run(src)
	if err != nil {
		t.Fatal(err)
	}
	if len(pic.Items) != 5 || len(pic.Labels) != 2 || pic.Size != (Pair{100, 0}) {
		t.Fatalf("%d items, %d labels, size %v", len(pic.Items), len(pic.Labels), pic.Size)
	}
	if !near(pic.Labels[1].At, Pair{2, 1.5}) || pic.Labels[0].Pen.Font != 10 {
		t.Errorf("labels %+v", pic.Labels)
	}
	f, err := pic.Layout("x")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(f.SVG, "<svg") || !strings.Contains(f.SVG, `stroke-dasharray="6.4 6.4"`) || f.Width < minWidth {
		t.Errorf("%v %s", f.Width, f.SVG)
	}
}

func TestComments(t *testing.T) {
	// The source lost its line breaks: the comment ends at its semicolon, or before code.
	pic, err := Run(`// a figure //draw((0,0)); real r = 1; //from a thread pair A = (r,0); draw(A--(0,1));`)
	if err != nil || len(pic.Items) != 1 {
		t.Fatalf("%v %+v", err, pic)
	}
}

func TestGeometry(t *testing.T) {
	if p := extension(Pair{0, 0}, Pair{1, 1}, Pair{0, 2}, Pair{2, 0}); !near(p, Pair{1, 1}) {
		t.Errorf("extension %v", p)
	}
	if p := foot(Pair{1, 5}, Pair{0, 0}, Pair{4, 0}); !near(p, Pair{1, 0}) {
		t.Errorf("foot %v", p)
	}
	if p := circumcenter(Pair{0, 0}, Pair{4, 0}, Pair{0, 2}); !near(p, Pair{2, 1}) {
		t.Errorf("circumcenter %v", p)
	}
	c := circle(Pair{}, 1)
	if p := c.point(0.5); math.Abs(p.abs()-1) > 1e-3 {
		t.Errorf("the circle is not round: %v", p)
	}
	ts := intersect(c, line(Pair{-2, 0}, Pair{2, 0}))
	if len(ts) != 2 || !near(c.point(ts[0][0]), Pair{1, 0}) || !near(c.point(ts[1][0]), Pair{-1, 0}) {
		t.Errorf("intersections %v", ts)
	}
	// Four knots on a circle joined with .. give the circle.
	pic, err := Run(`draw((1,0)..(0,1)..(-1,0)..(0,-1)..cycle);`)
	if err != nil {
		t.Fatal(err)
	}
	if p := pic.Items[0].Path.point(0.5); math.Abs(p.abs()-1) > 0.01 {
		t.Errorf("the smooth path is not round: %v", p.abs())
	}
	if p := rotate(90, Pair{1, 0}).pair(Pair{2, 0}); !near(p, Pair{1, 1}) {
		t.Errorf("rotate %v", p)
	}
}

func TestErrors(t *testing.T) {
	for _, src := range []string{`import three; draw((0,0,0)--(1,1,1));`, `picture p; add(p);`, `draw(;`, `while (true) {}`, `draw(nosuch);`} {
		if _, err := Run(src); err == nil {
			t.Errorf("%q gave no error", src)
		}
	}
}
