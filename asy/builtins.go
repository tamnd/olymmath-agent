package asy

import (
	"math"
)

var constants = map[string]value{
	"pi": math.Pi, "N": Pair{0, 1}, "S": Pair{0, -1}, "E": Pair{1, 0}, "W": Pair{-1, 0},
	"NE": dirDeg(45), "NW": dirDeg(135), "SW": dirDeg(225), "SE": dirDeg(315),
	"NNE": dirDeg(67.5), "ENE": dirDeg(22.5), "NNW": dirDeg(112.5), "WNW": dirDeg(157.5),
	"SSW": dirDeg(247.5), "WSW": dirDeg(202.5), "SSE": dirDeg(292.5), "ESE": dirDeg(337.5),
	"up": Pair{0, 1}, "down": Pair{0, -1}, "left": Pair{-1, 0}, "right": Pair{1, 0}, "origin": Pair{},
	"cm": 72 / 2.54, "mm": 72 / 25.4, "inch": 72.0, "inches": 72.0, "pt": 72 / 72.27, "bp": 1.0,
	"unitcircle": circle(Pair{}, 1), "unitsquare": square(),
	"Arrow": arrowKind(1), "EndArrow": arrowKind(1), "ArcArrow": arrowKind(1), "EndArcArrow": arrowKind(1),
	"Arrows": arrowKind(2), "ArcArrows": arrowKind(2), "BeginArrow": arrowKind(3), "MidArrow": arrowKind(0),
	"None": arrowKind(0), "NoMargin": arrowKind(0), "black": colors["black"],
	"CCW": true, "CW": false, "markscalefactor": 0.03, "Fill": arrowKind(0), "NoFill": arrowKind(0),
	"roundcap": Pen{}, "squarecap": Pen{}, "extendcap": Pen{}, "roundjoin": Pen{}, "miterjoin": Pen{}, "beveljoin": Pen{},
	"keepAspect": true, "IgnoreAspect": false, "Aspect": true,
}

func square() *Path {
	p := &Path{Knots: []Pair{{0, 0}, {1, 0}, {1, 1}, {0, 1}}, Cycle: true}
	straighten(p)
	return p
}

var builtins map[string]builtin

func init() {
	builtins = map[string]builtin{
		"size":       bSize,
		"unitsize":   bUnitsize,
		"defaultpen": func(e *evaluator, a []value, _ []string) value { e.defPen = e.defPen.plus(penOf(a, 0)); return nil },
		"linewidth": func(_ *evaluator, a []value, _ []string) value {
			return Pen{Width: num(arg(a, 0)), HasWidth: true}
		},
		"fontsize": func(_ *evaluator, a []value, _ []string) value { return Pen{Font: num(arg(a, 0)), HasFont: true} },
		"linetype": func(_ *evaluator, a []value, _ []string) value {
			s, _ := arg(a, 0).(string)
			return Pen{Dash: s, HasDash: true}
		},
		"rgb": func(_ *evaluator, a []value, _ []string) value {
			p := color(num(arg(a, 0)), num(arg(a, 1)), num(arg(a, 2)))
			if p.R > 1 || p.G > 1 || p.B > 1 {
				p.R, p.G, p.B = p.R/255, p.G/255, p.B/255
			}
			return p
		},
		"opacity": func(_ *evaluator, a []value, _ []string) value {
			return Pen{Opacity: num(arg(a, 0)), HasOpac: true}
		},
		"usepackage":  none,
		"texpreamble": none,
		"import":      none,
		"draw":        bDraw,
		"fill":        func(e *evaluator, a []value, n []string) value { return e.fill(a, n, false, false) },
		"filldraw":    func(e *evaluator, a []value, n []string) value { return e.fill(a, n, true, false) },
		"unfill":      func(e *evaluator, a []value, n []string) value { return e.fill(a, n, false, true) },
		"dot":         bDot,
		"label":       bLabel,
		"clip": func(e *evaluator, a []value, _ []string) value {
			e.pic.Clip = pathOf(arg(a, 0))
			return nil
		},
		"dir": func(_ *evaluator, a []value, _ []string) value {
			switch v := arg(a, 0).(type) {
			case float64:
				return dirDeg(v)
			case Pair:
				return v.unit()
			case *Path:
				if len(a) == 1 {
					return v.dir(float64(v.segments()))
				}
				return v.dir(num(arg(a, 1)))
			}
			fail("dir needs a number, a pair or a path")
			return nil
		},
		"sqrt":  math1(math.Sqrt),
		"sin":   math1(math.Sin),
		"cos":   math1(math.Cos),
		"tan":   math1(math.Tan),
		"asin":  math1(math.Asin),
		"acos":  math1(math.Acos),
		"exp":   math1(math.Exp),
		"log":   math1(math.Log),
		"floor": math1(math.Floor),
		"ceil":  math1(math.Ceil),
		"round": math1(math.Round),
		"Sin":   math1(func(x float64) float64 { return math.Sin(x * math.Pi / 180) }),
		"Cos":   math1(func(x float64) float64 { return math.Cos(x * math.Pi / 180) }),
		"Tan":   math1(func(x float64) float64 { return math.Tan(x * math.Pi / 180) }),
		"aSin":  math1(func(x float64) float64 { return math.Asin(x) * 180 / math.Pi }),
		"aCos":  math1(func(x float64) float64 { return math.Acos(x) * 180 / math.Pi }),
		"aTan":  math1(func(x float64) float64 { return math.Atan(x) * 180 / math.Pi }),
		"atan": func(_ *evaluator, a []value, _ []string) value {
			return math.Atan(num(arg(a, 0)))
		},
		"atan2": func(_ *evaluator, a []value, _ []string) value { return math.Atan2(num(arg(a, 0)), num(arg(a, 1))) },
		"abs": func(_ *evaluator, a []value, _ []string) value {
			if p, ok := arg(a, 0).(Pair); ok {
				return p.abs()
			}
			return math.Abs(num(arg(a, 0)))
		},
		"length": func(_ *evaluator, a []value, _ []string) value {
			switch v := arg(a, 0).(type) {
			case Pair:
				return v.abs()
			case *Path:
				return float64(v.segments())
			}
			fail("length needs a pair or a path")
			return nil
		},
		"min":   minMax(-1),
		"max":   minMax(1),
		"unit":  func(_ *evaluator, a []value, _ []string) value { return pairOf(arg(a, 0)).unit() },
		"conj":  func(_ *evaluator, a []value, _ []string) value { p := pairOf(arg(a, 0)); return Pair{p.X, -p.Y} },
		"xpart": func(_ *evaluator, a []value, _ []string) value { return pairOf(arg(a, 0)).X },
		"ypart": func(_ *evaluator, a []value, _ []string) value { return pairOf(arg(a, 0)).Y },
		"degrees": func(_ *evaluator, a []value, _ []string) value {
			if p, ok := arg(a, 0).(Pair); ok {
				d := math.Atan2(p.Y, p.X) * 180 / math.Pi
				if d < 0 {
					d += 360
				}
				return d
			}
			return num(arg(a, 0)) * 180 / math.Pi
		},
		"radians": math1(func(x float64) float64 { return x * math.Pi / 180 }),
		"angle": func(_ *evaluator, a []value, _ []string) value {
			p := pairOf(arg(a, 0))
			return math.Atan2(p.Y, p.X)
		},
		"arclength": func(_ *evaluator, a []value, _ []string) value { return pathOf(arg(a, 0)).arclength() },
		"point": func(_ *evaluator, a []value, _ []string) value {
			return pathOf(arg(a, 0)).point(num(arg(a, 1)))
		},
		"relpoint": func(_ *evaluator, a []value, _ []string) value {
			p := pathOf(arg(a, 0))
			return p.point(p.arctime(num(arg(a, 1)) * p.arclength()))
		},
		"midpoint": func(_ *evaluator, a []value, _ []string) value {
			if len(a) == 2 {
				return pairOf(a[0]).add(pairOf(a[1])).scale(0.5)
			}
			p := pathOf(arg(a, 0))
			return p.point(p.arctime(p.arclength() / 2))
		},
		"interp": func(_ *evaluator, a []value, _ []string) value {
			p, q, t := pairOf(arg(a, 0)), pairOf(arg(a, 1)), num(arg(a, 2))
			return p.add(q.sub(p).scale(t))
		},
		"reverse": func(_ *evaluator, a []value, _ []string) value { return pathOf(arg(a, 0)).reverse() },
		"subpath": func(_ *evaluator, a []value, _ []string) value {
			return pathOf(arg(a, 0)).subpath(num(arg(a, 1)), num(arg(a, 2)))
		},
		"extension": func(_ *evaluator, a []value, _ []string) value {
			return extension(pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2)), pairOf(arg(a, 3)))
		},
		"foot": func(_ *evaluator, a []value, _ []string) value {
			return foot(pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2)))
		},
		"centroid": func(_ *evaluator, a []value, _ []string) value {
			return pairOf(arg(a, 0)).add(pairOf(arg(a, 1))).add(pairOf(arg(a, 2))).scale(1.0 / 3)
		},
		"circumcenter": func(_ *evaluator, a []value, _ []string) value {
			return circumcenter(pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2)))
		},
		"circumradius": func(_ *evaluator, a []value, _ []string) value {
			p := pairOf(arg(a, 0))
			return circumcenter(p, pairOf(arg(a, 1)), pairOf(arg(a, 2))).sub(p).abs()
		},
		"circumcircle": func(_ *evaluator, a []value, _ []string) value {
			p := pairOf(arg(a, 0))
			c := circumcenter(p, pairOf(arg(a, 1)), pairOf(arg(a, 2)))
			return circle(c, c.sub(p).abs())
		},
		"incenter": func(_ *evaluator, a []value, _ []string) value {
			c, _ := incircle(pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2)))
			return c
		},
		"inradius": func(_ *evaluator, a []value, _ []string) value {
			_, r := incircle(pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2)))
			return r
		},
		"incircle": func(_ *evaluator, a []value, _ []string) value {
			c, r := incircle(pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2)))
			return circle(c, r)
		},
		"orthocenter": func(_ *evaluator, a []value, _ []string) value {
			p, q, r := pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2))
			return p.add(q).add(r).sub(circumcenter(p, q, r).scale(2))
		},
		"circle": bCircle,
		"Circle": bCircle,
		"CR":     bCircle,
		"ellipse": func(_ *evaluator, a []value, _ []string) value {
			c := pairOf(arg(a, 0))
			return shift(c).then(Transform{XX: num(arg(a, 1)), YY: num(arg(a, 2))}).path(circle(Pair{}, 1))
		},
		"arc": bArc,
		"intersectionpoint": func(_ *evaluator, a []value, _ []string) value {
			ts := intersect(pathOf(arg(a, 0)), pathOf(arg(a, 1)))
			if len(ts) == 0 {
				fail("the paths do not meet")
			}
			return pathOf(a[0]).point(ts[0][0])
		},
		"intersectionpoints": func(_ *evaluator, a []value, _ []string) value {
			p := pathOf(arg(a, 0))
			out := &array{}
			for _, t := range intersect(p, pathOf(arg(a, 1))) {
				out.elems = append(out.elems, p.point(t[0]))
			}
			return out
		},
		"shift": func(_ *evaluator, a []value, _ []string) value {
			if len(a) == 2 {
				return shift(Pair{num(a[0]), num(a[1])})
			}
			return shift(pairOf(arg(a, 0)))
		},
		"rotate": func(_ *evaluator, a []value, _ []string) value {
			c := Pair{}
			if len(a) > 1 {
				c = pairOf(a[1])
			}
			return rotate(num(arg(a, 0)), c)
		},
		"scale": func(_ *evaluator, a []value, _ []string) value {
			if len(a) == 2 {
				return Transform{XX: num(a[0]), YY: num(a[1])}
			}
			k := num(arg(a, 0))
			return Transform{XX: k, YY: k}
		},
		"xscale":  func(_ *evaluator, a []value, _ []string) value { return Transform{XX: num(arg(a, 0)), YY: 1} },
		"yscale":  func(_ *evaluator, a []value, _ []string) value { return Transform{XX: 1, YY: num(arg(a, 0))} },
		"reflect": func(_ *evaluator, a []value, _ []string) value { return reflect(pairOf(arg(a, 0)), pairOf(arg(a, 1))) },
		"string": func(_ *evaluator, a []value, _ []string) value {
			if s, ok := arg(a, 0).(string); ok {
				return s
			}
			return fmtNum(num(arg(a, 0)))
		},
		"polygon": func(_ *evaluator, a []value, _ []string) value {
			n := int(num(arg(a, 0)))
			p := &Path{Cycle: true}
			for i := range n {
				p.Knots = append(p.Knots, dirDeg(90+360*float64(i)/float64(n)))
			}
			straighten(p)
			return p
		},
		"rightanglemark": func(_ *evaluator, a []value, _ []string) value {
			p, q, r := pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2))
			s := 2.0
			if len(a) > 3 {
				s = num(a[3])
			}
			s *= 0.03 * math.Max(p.sub(q).abs(), r.sub(q).abs()) / 2
			u, v := p.sub(q).unit().scale(s), r.sub(q).unit().scale(s)
			m := &Path{Knots: []Pair{q.add(u), q.add(u).add(v), q.add(v)}}
			straighten(m)
			return m
		},
		"markangle":      bMark(false),
		"markrightangle": bMark(true),
		// The commands of the module cse5.
		"D":  bD,
		"MP": bMP,
		"IP": func(e *evaluator, a []value, n []string) value { return builtins["intersectionpoint"](e, a, n) },
		"IPs": func(e *evaluator, a []value, n []string) value {
			return builtins["intersectionpoints"](e, a, n)
		},
		"CP": func(_ *evaluator, a []value, _ []string) value {
			c := pairOf(arg(a, 0))
			return circle(c, pairOf(arg(a, 1)).sub(c).abs())
		},
	}
	for _, k := range []string{"Arrow", "Arrows", "EndArrow", "BeginArrow", "ArcArrow", "ArcArrows", "MidArrow"} {
		kind := constants[k].(arrowKind)
		builtins[k] = func(*evaluator, []value, []string) value { return kind }
	}
}

func none(*evaluator, []value, []string) value { return nil }

func arg(a []value, i int) value {
	if i >= len(a) {
		fail("too few arguments")
	}
	return a[i]
}

func penOf(a []value, i int) Pen {
	if p, ok := arg(a, i).(Pen); ok {
		return p
	}
	fail("want a pen")
	return Pen{}
}

func math1(f func(float64) float64) builtin {
	return func(_ *evaluator, a []value, _ []string) value { return f(num(arg(a, 0))) }
}

func minMax(sign float64) builtin {
	return func(_ *evaluator, a []value, _ []string) value {
		if len(a) == 1 {
			if arr, ok := a[0].(*array); ok {
				a = arr.elems
			}
		}
		best := num(arg(a, 0))
		for _, v := range a[1:] {
			if x := num(v); (x-best)*sign > 0 {
				best = x
			}
		}
		return best
	}
}

func extension(a, b, c, d Pair) Pair {
	r, s := b.sub(a), d.sub(c)
	den := r.cross(s)
	if math.Abs(den) < 1e-12 {
		fail("the lines are parallel")
	}
	return a.add(r.scale(c.sub(a).cross(s) / den))
}

func foot(p, a, b Pair) Pair {
	d := b.sub(a)
	return a.add(d.scale(p.sub(a).dot(d) / d.dot(d)))
}

func circumcenter(a, b, c Pair) Pair {
	m1, m2 := a.add(b).scale(0.5), a.add(c).scale(0.5)
	r1, r2 := b.sub(a), c.sub(a)
	return extension(m1, m1.add(Pair{-r1.Y, r1.X}), m2, m2.add(Pair{-r2.Y, r2.X}))
}

func incircle(a, b, c Pair) (Pair, float64) {
	la, lb, lc := b.sub(c).abs(), c.sub(a).abs(), a.sub(b).abs()
	s := la + lb + lc
	i := a.scale(la / s).add(b.scale(lb / s)).add(c.scale(lc / s))
	return i, math.Abs(b.sub(a).cross(c.sub(a))) / s
}

func bSize(e *evaluator, a []value, _ []string) value {
	w := num(arg(a, 0))
	h := w
	if len(a) > 1 {
		if f, ok := a[1].(float64); ok {
			h = f
		}
	}
	if len(a) == 1 {
		h = 0
	}
	e.pic.Size = Pair{w, h}
	return nil
}

func bUnitsize(e *evaluator, a []value, _ []string) value {
	u := num(arg(a, 0))
	v := u
	if len(a) > 1 {
		v = num(a[1])
	}
	e.pic.Unit = Pair{u, v}
	return nil
}

func bCircle(_ *evaluator, a []value, _ []string) value {
	c := pairOf(arg(a, 0))
	if q, ok := arg(a, 1).(Pair); ok {
		// circle(A, B, C) is the circle through three points in some modules.
		if len(a) > 2 {
			o := circumcenter(c, q, pairOf(a[2]))
			return circle(o, o.sub(c).abs())
		}
		fail("circle needs a radius")
	}
	return circle(c, num(arg(a, 1)))
}

func bArc(_ *evaluator, a []value, _ []string) value {
	c := pairOf(arg(a, 0))
	if p, ok := arg(a, 1).(Pair); ok {
		// arc(c, z1, z2, direction) goes from z1 to the ray toward z2.
		q := pairOf(arg(a, 2))
		ccw := true
		if len(a) > 3 {
			ccw = truth(a[3])
		}
		a1 := math.Atan2(p.Y-c.Y, p.X-c.X) * 180 / math.Pi
		a2 := math.Atan2(q.Y-c.Y, q.X-c.X) * 180 / math.Pi
		if ccw && a2 < a1 {
			a2 += 360
		}
		if !ccw && a2 > a1 {
			a2 -= 360
		}
		return arc(c, p.sub(c).abs(), a1, a2)
	}
	r, a1, a2 := num(arg(a, 1)), num(arg(a, 2)), num(arg(a, 3))
	if len(a) > 4 && !truth(a[4]) && a2 > a1 {
		a2 -= 360
	}
	return arc(c, r, a1, a2)
}

// drawArgs sorts the arguments of a drawing command: a text, the paths, a direction for the text, a pen and an arrow.
type drawArgs struct {
	text    string
	hasText bool
	paths   []*Path
	align   Pair
	pen     Pen
	pens    []Pen
	arrow   arrowKind
}

func (e *evaluator) sort(a []value, names []string) drawArgs {
	var d drawArgs
	geom := false
	for i, v := range a {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		switch v := v.(type) {
		case string:
			d.text, d.hasText = v, true
		case Pair:
			if geom || name == "align" {
				d.align = v
			} else {
				d.paths = append(d.paths, pathOf(v))
				geom = true
			}
		case *Path:
			d.paths = append(d.paths, v)
			geom = true
		case []*Path:
			d.paths = append(d.paths, v...)
			geom = true
		case *array:
			for _, el := range v.elems {
				d.paths = append(d.paths, pathOf(el))
			}
			geom = true
		case Pen:
			d.pens = append(d.pens, v)
		case arrowKind:
			d.arrow = v
		case float64, bool, nil:
		default:
			fail("%s is not an argument of a drawing command", typeName(v))
		}
	}
	d.pen = e.defPen
	if len(d.pens) > 0 {
		d.pen = e.defPen.plus(d.pens[0])
	}
	return d
}

func bDraw(e *evaluator, a []value, n []string) value {
	d := e.sort(a, n)
	for _, p := range d.paths {
		if len(p.Knots) == 1 {
			e.pic.Items = append(e.pic.Items, Item{Path: p, Dot: true, R: d.pen.Width / 2, Pen: d.pen})
			continue
		}
		e.pic.Items = append(e.pic.Items, Item{Path: p, Stroke: true, Pen: d.pen, Arrow: int(d.arrow)})
	}
	if d.hasText && len(d.paths) > 0 {
		p := d.paths[0]
		at := p.point(p.arctime(p.arclength() / 2))
		e.pic.Labels = append(e.pic.Labels, Label{Text: d.text, At: at, Align: d.align, Pen: d.pen})
	}
	return nil
}

func (e *evaluator) fill(a []value, n []string, stroke, white bool) value {
	d := e.sort(a, n)
	fp := d.pen
	sp := d.pen
	if len(d.pens) > 1 {
		sp = e.defPen.plus(d.pens[1])
	}
	if white {
		fp = e.defPen.plus(colors["white"])
	}
	for _, p := range d.paths {
		e.pic.Items = append(e.pic.Items, Item{Path: p, Fill: true, FillPen: fp, Stroke: stroke, Pen: sp})
	}
	return nil
}

func bDot(e *evaluator, a []value, n []string) value {
	d := e.sort(a, n)
	r := e.dotfactor * d.pen.Width / 2
	for _, p := range d.paths {
		for _, k := range p.Knots {
			e.pic.Items = append(e.pic.Items, Item{Path: pathOf(k), Dot: true, R: r, Pen: d.pen})
		}
	}
	if d.hasText && len(d.paths) > 0 {
		e.pic.Labels = append(e.pic.Labels, Label{d.text, d.paths[0].Knots[0], d.align, d.pen, r})
	}
	return nil
}

func bLabel(e *evaluator, a []value, n []string) value {
	d := e.sort(a, n)
	if !d.hasText {
		fail("label needs a text")
	}
	at := Pair{}
	if len(d.paths) > 0 {
		p := d.paths[0]
		at = p.point(p.arctime(p.arclength() / 2))
		if len(p.Knots) == 1 {
			at = p.Knots[0]
		}
	}
	e.pic.Labels = append(e.pic.Labels, Label{Text: d.text, At: at, Align: d.align, Pen: d.pen})
	return nil
}

// bD is D of cse5: it draws a path, or a dot for a pair, and gives back its argument.
func bD(e *evaluator, a []value, n []string) value {
	if p, ok := arg(a, 0).(Pair); ok {
		bDot(e, a, n)
		return p
	}
	bDraw(e, a, n)
	return a[0]
}

// bMP is MP of cse5: a label in math mode.
func bMP(e *evaluator, a []value, n []string) value {
	s, ok := arg(a, 0).(string)
	if !ok {
		fail("MP needs a text")
	}
	b := append([]value{"$" + s + "$"}, a[1:]...)
	bLabel(e, b, n)
	for _, v := range a[1:] {
		if p, ok := v.(Pair); ok {
			return p
		}
	}
	return nil
}

// bMark is markangle and markrightangle of the module geometry. The size of a mark is in points, so Layout draws it.
func bMark(right bool) builtin {
	return func(e *evaluator, a []value, names []string) value {
		p, q, r := pairOf(arg(a, 0)), pairOf(arg(a, 1)), pairOf(arg(a, 2))
		m := Item{Mark: true, Right: right, Path: pathOf(q), Pen: e.defPen, R: 8}
		if right {
			m.R = 6
		}
		for i, v := range a[3:] {
			switch v := v.(type) {
			case float64:
				if i+3 < len(names) && names[i+3] == "radius" || !right {
					m.R = v
				}
			case Pen:
				m.Pen = e.defPen.plus(v)
			}
		}
		m.A1 = math.Atan2(p.Y-q.Y, p.X-q.X) * 180 / math.Pi
		m.A2 = math.Atan2(r.Y-q.Y, r.X-q.X) * 180 / math.Pi
		if m.A2 < m.A1 {
			m.A2 += 360
		}
		e.pic.Items = append(e.pic.Items, m)
		return nil
	}
}
