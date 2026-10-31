package asy

import (
	"math"
	"sort"
)

// Pair is a point or a vector.
type Pair struct{ X, Y float64 }

func (a Pair) add(b Pair) Pair      { return Pair{a.X + b.X, a.Y + b.Y} }
func (a Pair) sub(b Pair) Pair      { return Pair{a.X - b.X, a.Y - b.Y} }
func (a Pair) scale(k float64) Pair { return Pair{a.X * k, a.Y * k} }
func (a Pair) mul(b Pair) Pair      { return Pair{a.X*b.X - a.Y*b.Y, a.X*b.Y + a.Y*b.X} }
func (a Pair) abs() float64         { return math.Hypot(a.X, a.Y) }
func (a Pair) dot(b Pair) float64   { return a.X*b.X + a.Y*b.Y }
func (a Pair) cross(b Pair) float64 { return a.X*b.Y - a.Y*b.X }
func (a Pair) unit() Pair {
	if l := a.abs(); l > 0 {
		return a.scale(1 / l)
	}
	return Pair{}
}

func (a Pair) div(b Pair) Pair {
	d := b.X*b.X + b.Y*b.Y
	return Pair{(a.X*b.X + a.Y*b.Y) / d, (a.Y*b.X - a.X*b.Y) / d}
}

func dirDeg(deg float64) Pair {
	r := deg * math.Pi / 180
	return Pair{math.Cos(r), math.Sin(r)}
}

// Path is a list of cubic Bezier segments. Segment i goes from Knots[i] to the next knot with the controls Pre and Post: Post[i] after knot i and Pre[i+1] before the next.
type Path struct {
	Knots     []Pair
	Post, Pre []Pair
	Cycle     bool
}

func (p *Path) segments() int {
	if len(p.Knots) == 0 {
		return 0
	}
	if p.Cycle {
		return len(p.Knots)
	}
	return len(p.Knots) - 1
}

// seg gives the four points of segment i.
func (p *Path) seg(i int) (a, b, c, d Pair) {
	j := (i + 1) % len(p.Knots)
	return p.Knots[i], p.Post[i], p.Pre[j], p.Knots[j]
}

func bezier(a, b, c, d Pair, t float64) Pair {
	s := 1 - t
	return a.scale(s * s * s).add(b.scale(3 * s * s * t)).add(c.scale(3 * s * t * t)).add(d.scale(t * t * t))
}

// point gives the point at time t, as Asymptote does: knot k is at time k.
func (p *Path) point(t float64) Pair {
	n := p.segments()
	if n == 0 {
		if len(p.Knots) == 1 {
			return p.Knots[0]
		}
		return Pair{}
	}
	if p.Cycle {
		t = math.Mod(t, float64(n))
		if t < 0 {
			t += float64(n)
		}
	} else {
		t = math.Max(0, math.Min(t, float64(n)))
	}
	i := int(t)
	if i == n {
		i--
	}
	a, b, c, d := p.seg(i)
	return bezier(a, b, c, d, t-float64(i))
}

// dir gives the unit tangent at time t.
func (p *Path) dir(t float64) Pair {
	const h = 1e-4
	lo, hi := t-h, t+h
	if !p.Cycle {
		lo = math.Max(lo, 0)
		hi = math.Min(hi, float64(p.segments()))
	}
	return p.point(hi).sub(p.point(lo)).unit()
}

const flatSteps = 48

// flat gives points of the path and their times.
func (p *Path) flat() ([]Pair, []float64) {
	if p.segments() == 0 {
		return p.Knots, make([]float64, len(p.Knots))
	}
	var ps []Pair
	var ts []float64
	for i := 0; i < p.segments(); i++ {
		a, b, c, d := p.seg(i)
		steps := flatSteps
		if b == a.add(d.sub(a).scale(1.0/3)) && c == a.add(d.sub(a).scale(2.0/3)) {
			steps = 1
		}
		for k := 0; k < steps; k++ {
			u := float64(k) / float64(steps)
			ps = append(ps, bezier(a, b, c, d, u))
			ts = append(ts, float64(i)+u)
		}
	}
	n := p.segments()
	ps = append(ps, p.point(float64(n)))
	ts = append(ts, float64(n))
	return ps, ts
}

func (p *Path) arclength() float64 {
	ps, _ := p.flat()
	l := 0.0
	for i := 1; i < len(ps); i++ {
		l += ps[i].sub(ps[i-1]).abs()
	}
	return l
}

// arctime gives the time at which the arc length from the start is l.
func (p *Path) arctime(l float64) float64 {
	ps, ts := p.flat()
	for i := 1; i < len(ps); i++ {
		d := ps[i].sub(ps[i-1]).abs()
		if d >= l && d > 0 {
			return ts[i-1] + (ts[i]-ts[i-1])*l/d
		}
		l -= d
	}
	return ts[len(ts)-1]
}

// intersect gives the times on p and q of the points where the paths meet, in the order of the times on p.
func intersect(p, q *Path) [][2]float64 {
	ap, at := p.flat()
	bp, bt := q.flat()
	var out [][2]float64
	for i := 1; i < len(ap); i++ {
		for j := 1; j < len(bp); j++ {
			r := ap[i].sub(ap[i-1])
			s := bp[j].sub(bp[j-1])
			den := r.cross(s)
			if math.Abs(den) < 1e-12 {
				continue
			}
			w := bp[j-1].sub(ap[i-1])
			u := w.cross(s) / den
			v := w.cross(r) / den
			if u < -1e-9 || u > 1+1e-9 || v < -1e-9 || v > 1+1e-9 {
				continue
			}
			out = append(out, [2]float64{at[i-1] + (at[i]-at[i-1])*u, bt[j-1] + (bt[j]-bt[j-1])*v})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i][0] < out[j][0] })
	// Two segments that meet at a shared end give the same point twice.
	var uniq [][2]float64
next:
	for _, o := range out {
		for _, u := range uniq {
			if p.point(o[0]).sub(p.point(u[0])).abs() < 1e-7 {
				continue next
			}
		}
		uniq = append(uniq, o)
	}
	return uniq
}

func (p *Path) bounds() (lo, hi Pair, ok bool) {
	ps, _ := p.flat()
	for i, q := range ps {
		if i == 0 {
			lo, hi = q, q
			continue
		}
		lo = Pair{math.Min(lo.X, q.X), math.Min(lo.Y, q.Y)}
		hi = Pair{math.Max(hi.X, q.X), math.Max(hi.Y, q.Y)}
	}
	return lo, hi, len(ps) > 0
}

func (p *Path) reverse() *Path {
	n := len(p.Knots)
	r := &Path{Cycle: p.Cycle}
	if !p.Cycle {
		for i := n - 1; i >= 0; i-- {
			r.Knots = append(r.Knots, p.Knots[i])
			r.Post = append(r.Post, p.Pre[i])
			r.Pre = append(r.Pre, p.Post[i])
		}
		return r
	}
	for k := 0; k < n; k++ {
		i := (n - k) % n
		r.Knots = append(r.Knots, p.Knots[i])
		r.Post = append(r.Post, p.Pre[i])
		r.Pre = append(r.Pre, p.Post[i])
	}
	return r
}

// subpath gives the part of p from time a to time b, with a < b.
func (p *Path) subpath(a, b float64) *Path {
	if b < a {
		return p.subpath(b, a).reverse()
	}
	ps, ts := p.flat()
	r := &Path{}
	add := func(q Pair) {
		if len(r.Knots) > 0 && r.Knots[len(r.Knots)-1] == q {
			return
		}
		r.Knots = append(r.Knots, q)
	}
	add(p.point(a))
	for i, t := range ts {
		if t > a && t < b {
			add(ps[i])
		}
	}
	add(p.point(b))
	straighten(r)
	return r
}

func straighten(p *Path) {
	n := len(p.Knots)
	p.Post = make([]Pair, n)
	p.Pre = make([]Pair, n)
	for i := 0; i < p.segments(); i++ {
		j := (i + 1) % n
		a, d := p.Knots[i], p.Knots[j]
		p.Post[i] = a.add(d.sub(a).scale(1.0 / 3))
		p.Pre[j] = a.add(d.sub(a).scale(2.0 / 3))
	}
}

func line(a, b Pair) *Path {
	p := &Path{Knots: []Pair{a, b}}
	straighten(p)
	return p
}

// arc gives the arc of the circle with center c and radius r from angle a1 to angle a2 in degrees. It goes counterclockwise when a2 > a1.
func arc(c Pair, r, a1, a2 float64) *Path {
	n := int(math.Ceil(math.Abs(a2-a1) / 90))
	if n == 0 {
		n = 1
	}
	step := (a2 - a1) / float64(n)
	k := 4.0 / 3 * math.Tan(step*math.Pi/180/4) * r
	p := &Path{}
	for i := 0; i <= n; i++ {
		a := a1 + step*float64(i)
		q := c.add(dirDeg(a).scale(r))
		t := dirDeg(a + 90).scale(k)
		p.Knots = append(p.Knots, q)
		p.Pre = append(p.Pre, q.sub(t))
		p.Post = append(p.Post, q.add(t))
	}
	return p
}

func circle(c Pair, r float64) *Path {
	p := arc(c, r, 0, 360)
	p.Knots = p.Knots[:4]
	p.Pre = p.Pre[:4]
	p.Post = p.Post[:4]
	p.Cycle = true
	return p
}

// Transform maps (x, y) to (X + XX x + XY y, Y + YX x + YY y), as in Asymptote.
type Transform struct{ X, Y, XX, XY, YX, YY float64 }

var identity = Transform{XX: 1, YY: 1}

func (t Transform) pair(p Pair) Pair {
	return Pair{t.X + t.XX*p.X + t.XY*p.Y, t.Y + t.YX*p.X + t.YY*p.Y}
}

func (t Transform) then(u Transform) Transform {
	// t*u: first u, then t.
	return Transform{
		X: t.X + t.XX*u.X + t.XY*u.Y, Y: t.Y + t.YX*u.X + t.YY*u.Y,
		XX: t.XX*u.XX + t.XY*u.YX, XY: t.XX*u.XY + t.XY*u.YY,
		YX: t.YX*u.XX + t.YY*u.YX, YY: t.YX*u.XY + t.YY*u.YY,
	}
}

func (t Transform) path(p *Path) *Path {
	r := &Path{Cycle: p.Cycle}
	for i := range p.Knots {
		r.Knots = append(r.Knots, t.pair(p.Knots[i]))
		r.Pre = append(r.Pre, t.pair(p.Pre[i]))
		r.Post = append(r.Post, t.pair(p.Post[i]))
	}
	return r
}

func shift(p Pair) Transform { return Transform{X: p.X, Y: p.Y, XX: 1, YY: 1} }

func rotate(deg float64, c Pair) Transform {
	d := dirDeg(deg)
	r := Transform{XX: d.X, XY: -d.Y, YX: d.Y, YY: d.X}
	return shift(c).then(r).then(shift(c.scale(-1)))
}

func reflect(a, b Pair) Transform {
	d := b.sub(a).unit()
	r := Transform{XX: d.X*d.X - d.Y*d.Y, XY: 2 * d.X * d.Y, YX: 2 * d.X * d.Y, YY: d.Y*d.Y - d.X*d.X}
	return shift(a).then(r).then(shift(a.scale(-1)))
}

// hobby gives the length factor of a control point for the angles theta and phi between the chord and the tangents, from the rule of John Hobby with tension 1.
func hobby(theta, phi float64) float64 {
	st, ct := math.Sincos(theta)
	sp, cp := math.Sincos(phi)
	num := 2 + math.Sqrt2*(st-sp/16)*(sp-st/16)*(ct-cp)
	den := 3 * (1 + (math.Sqrt(5)-1)/2*ct + (3-math.Sqrt(5))/2*cp)
	return num / den
}

// smooth sets the controls of the curved segments of p. Each knot gets a tangent, from its direction when given or else from its neighbours, and the control lengths come from hobby.
func smooth(p *Path, curved []bool, dirs []Pair) {
	n := len(p.Knots)
	tan := make([]Pair, n)
	for i := 0; i < n; i++ {
		if dirs[i] != (Pair{}) {
			tan[i] = dirs[i].unit()
			continue
		}
		prev, next := i-1, i+1
		if p.Cycle {
			prev, next = (i+n-1)%n, (i+1)%n
		}
		inCurve := prev >= 0 && curved[prev%max(1, p.segments())]
		outCurve := next < n && i < p.segments() && curved[i]
		switch {
		case inCurve && outCurve && prev >= 0 && next < n:
			a := p.Knots[i].sub(p.Knots[prev])
			b := p.Knots[next].sub(p.Knots[i])
			// The tangent of the circle through the three points.
			tan[i] = a.unit().scale(b.abs()).add(b.unit().scale(a.abs())).unit()
		case outCurve:
			tan[i] = Pair{}
		case inCurve:
			tan[i] = Pair{}
		}
	}
	// An end with no direction gets the tangent of a circular arc: the mirror of the tangent at the other end of its segment.
	for i := 0; i < p.segments(); i++ {
		if !curved[i] {
			continue
		}
		j := (i + 1) % n
		ch := p.Knots[j].sub(p.Knots[i])
		if tan[i] == (Pair{}) && tan[j] != (Pair{}) {
			tan[i] = mirror(tan[j], ch)
		}
		if tan[j] == (Pair{}) && tan[i] != (Pair{}) {
			tan[j] = mirror(tan[i], ch)
		}
	}
	for i := 0; i < p.segments(); i++ {
		if !curved[i] {
			continue
		}
		j := (i + 1) % n
		a, d := p.Knots[i], p.Knots[j]
		ch := d.sub(a)
		l := ch.abs()
		t0, t1 := tan[i], tan[j]
		if t0 == (Pair{}) {
			t0 = ch.unit()
		}
		if t1 == (Pair{}) {
			t1 = ch.unit()
		}
		theta := math.Atan2(ch.cross(t0), ch.dot(t0))
		phi := math.Atan2(t1.cross(ch), ch.dot(t1))
		p.Post[i] = a.add(t0.scale(l * hobby(theta, phi)))
		p.Pre[j] = d.sub(t1.scale(l * hobby(phi, theta)))
	}
}

func mirror(t, ch Pair) Pair {
	u := ch.unit()
	return u.scale(2 * t.dot(u)).sub(t)
}
