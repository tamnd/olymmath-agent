package asy

import (
	"fmt"
	"math"
	"strconv"
)

// Pen is a color, a line width, a dash pattern and a font size. Each field has a flag that tells if the pen sets it, so that the sum of two pens keeps the fields of both.
type Pen struct {
	R, G, B, Opacity   float64
	Width, Font        float64
	Dash               string
	HasColor, HasOpac  bool
	HasWidth, HasFont  bool
	HasDash, Invisible bool
}

func (p Pen) plus(q Pen) Pen {
	if q.HasColor {
		p.R, p.G, p.B, p.HasColor = q.R, q.G, q.B, true
	}
	if q.HasOpac {
		p.Opacity, p.HasOpac = q.Opacity, true
	}
	if q.HasWidth {
		p.Width, p.HasWidth = q.Width, true
	}
	if q.HasFont {
		p.Font, p.HasFont = q.Font, true
	}
	if q.HasDash {
		p.Dash, p.HasDash = q.Dash, true
	}
	p.Invisible = p.Invisible || q.Invisible
	return p
}

func color(r, g, b float64) Pen { return Pen{R: r, G: g, B: b, HasColor: true} }

var colors = map[string]Pen{
	"black": color(0, 0, 0), "white": color(1, 1, 1), "red": color(1, 0, 0), "green": color(0, 1, 0), "blue": color(0, 0, 1),
	"yellow": color(1, 1, 0), "cyan": color(0, 1, 1), "magenta": color(1, 0, 1), "fuchsia": color(1, 0, 1),
	"gray": color(.5, .5, .5), "grey": color(.5, .5, .5), "lightgray": color(.9, .9, .9), "lightgrey": color(.9, .9, .9),
	"mediumgray": color(.75, .75, .75), "mediumgrey": color(.75, .75, .75), "darkgray": color(.25, .25, .25), "darkgrey": color(.25, .25, .25),
	"heavygray": color(.25, .25, .25), "palegray": color(.95, .95, .95), "orange": color(1, .5, 0), "purple": color(.5, 0, .5),
	"pink": color(1, .75, .75), "brown": color(.5, .25, 0), "olive": color(.5, .5, 0), "royalblue": color(.25, .41, .88),
	"lightred": color(1, .5, .5), "lightgreen": color(.5, 1, .5), "lightblue": color(.5, .5, 1), "lightyellow": color(1, 1, .5),
	"lightcyan": color(.5, 1, 1), "palered": color(1, .75, .75), "paleblue": color(.75, .75, 1), "palegreen": color(.75, 1, .75),
	"paleyellow": color(1, 1, .75), "heavyred": color(.75, 0, 0), "heavyblue": color(0, 0, .75), "heavygreen": color(0, .75, 0),
	"deepred": color(.5, 0, 0), "deepblue": color(0, 0, .5), "deepgreen": color(0, .5, 0), "darkblue": color(0, 0, .5),
	"darkgreen": color(0, .5, 0), "darkred": color(.5, 0, 0), "darkcyan": color(0, .5, .5), "lightolive": color(.8, .8, .5),
	"chartreuse": color(.5, 1, 0), "springgreen": color(0, 1, .5), "purple1": color(.5, 0, 1), "orangered": color(1, .27, 0),
}

var dashes = map[string]string{"solid": "", "dashed": "8 8", "dotted": "0 4", "longdashed": "24 8", "dashdotted": "8 8 0 8", "longdashdotted": "24 8 0 8", "Dotted": "0 8"}

// Picture holds what a program draws, in the coordinates of the program.
type Picture struct {
	Items  []Item
	Labels []Label
	// Unit is the size of one unit in PostScript points when the program sets it with unitsize, or else 0.
	Unit Pair
	// Size is the width and height that the program asks for with size, or 0.
	Size Pair
	Clip *Path
}

// Item is a path that the picture strokes, fills, or both.
type Item struct {
	Path         *Path
	Stroke, Fill bool
	Pen, FillPen Pen
	Dot          bool    // a dot at Path.Knots[0] with radius R in points
	R            float64 // dot radius
	Arrow        int     // 1 at the end, 2 at both ends, 3 at the start
	// Mark is an angle mark at Path.Knots[0] from angle A1 to angle A2 in degrees, with size R in points. Right makes it a square.
	Mark, Right bool
	A1, A2      float64
}

// Label is a text at a point. Align is the direction from the point to the text.
type Label struct {
	Text  string
	At    Pair
	Align Pair
	Pen   Pen
	// Gap is a distance in points to keep between the point and the text, such as the radius of a dot.
	Gap float64
}

type array struct{ elems []value }

type value any

type userFunc struct {
	decl funcDecl
	env  *env
}

type builtin func(e *evaluator, args []value, names []string) value

type arrowKind int

type env struct {
	vars   map[string]*variable
	parent *env
}

type variable struct {
	typ string
	v   value
}

func (e *env) find(name string) *variable {
	for s := e; s != nil; s = s.parent {
		if v, ok := s.vars[name]; ok {
			return v
		}
	}
	return nil
}

type evaluator struct {
	pic       Picture
	global    *env
	steps     int
	defPen    Pen
	dotfactor float64
}

type runError struct{ msg string }

func (e runError) Error() string { return e.msg }

func fail(f string, a ...any) { panic(runError{fmt.Sprintf(f, a...)}) }

// control flow inside a function or a loop.
type (
	flowReturn struct{ v value }
	flowBreak  struct{}
	flowCont   struct{}
)

const maxSteps = 200000

// Run runs src and gives the picture it draws.
func Run(src string) (pic *Picture, err error) {
	prog, err := parse(src)
	if err != nil {
		return nil, err
	}
	e := &evaluator{global: &env{vars: map[string]*variable{}}, dotfactor: 6}
	e.defPen = Pen{Width: 0.5, Font: 12, HasColor: true, HasWidth: true, HasFont: true, Opacity: 1}
	// The pens of the module cse5.
	e.global.vars["pathpen"] = &variable{"pen", Pen{}}
	e.global.vars["pointpen"] = &variable{"pen", Pen{}}
	defer func() {
		if r := recover(); r != nil {
			if re, ok := r.(runError); ok {
				err = re
				return
			}
			panic(r)
		}
	}()
	for _, s := range prog {
		if f := e.exec(s, e.global); f != nil {
			break
		}
	}
	return &e.pic, nil
}

func (e *evaluator) tick() {
	e.steps++
	if e.steps > maxSteps {
		fail("the program runs too long")
	}
}

// exec runs a statement. It gives a flow value when the statement returns, breaks or continues.
func (e *evaluator) exec(s node, en *env) any {
	e.tick()
	switch s := s.(type) {
	case blockStmt:
		inner := &env{vars: map[string]*variable{}, parent: en}
		for _, t := range s.body {
			if f := e.exec(t, inner); f != nil {
				return f
			}
		}
	case exprStmt:
		e.eval(s.x, en)
	case declStmt:
		for i, n := range s.names {
			var v value
			if s.vals[i] != nil {
				v = coerce(s.typ, e.eval(s.vals[i], en))
			} else {
				v = zero(s.typ)
			}
			en.vars[n] = &variable{s.typ, v}
		}
	case funcDecl:
		en.vars[s.name] = &variable{"func", &userFunc{s, en}}
	case ifStmt:
		if truth(e.eval(s.c, en)) {
			return e.exec(s.then, en)
		} else if s.els != nil {
			return e.exec(s.els, en)
		}
	case whileStmt:
		for truth(e.eval(s.c, en)) {
			switch f := e.exec(s.body, en).(type) {
			case flowBreak:
				return nil
			case flowReturn:
				return f
			}
		}
	case forStmt:
		inner := &env{vars: map[string]*variable{}, parent: en}
		if s.init != nil {
			e.exec(s.init, inner)
		}
		for s.c == nil || truth(e.eval(s.c, inner)) {
			f := e.exec(s.body, inner)
			if _, ok := f.(flowBreak); ok {
				return nil
			}
			if r, ok := f.(flowReturn); ok {
				return r
			}
			if s.post != nil {
				e.exec(s.post, inner)
			}
		}
	case eachStmt:
		a, ok := e.eval(s.over, en).(*array)
		if !ok {
			fail("for each needs an array")
		}
		for _, v := range a.elems {
			inner := &env{vars: map[string]*variable{s.name: {s.typ, v}}, parent: en}
			f := e.exec(s.body, inner)
			if _, ok := f.(flowBreak); ok {
				return nil
			}
			if r, ok := f.(flowReturn); ok {
				return r
			}
		}
	case returnStmt:
		var v value
		if s.x != nil {
			v = e.eval(s.x, en)
		}
		return flowReturn{v}
	case breakStmt:
		return flowBreak{}
	case contStmt:
		return flowCont{}
	default:
		fail("unknown statement %T", s)
	}
	return nil
}

func zero(typ string) value {
	switch typ {
	case "real", "int":
		return 0.0
	case "pair":
		return Pair{}
	case "bool":
		return false
	case "string":
		return ""
	case "pen":
		return Pen{}
	case "transform":
		return identity
	case "path", "guide":
		return &Path{}
	}
	if len(typ) > 2 && typ[len(typ)-2:] == "[]" {
		return &array{}
	}
	return nil
}

// coerce changes v to the type typ where Asymptote does so on its own.
func coerce(typ string, v value) value {
	switch typ {
	case "pair":
		if f, ok := v.(float64); ok {
			return Pair{f, 0}
		}
	case "path", "guide":
		if p, ok := v.(Pair); ok {
			return &Path{Knots: []Pair{p}, Pre: []Pair{p}, Post: []Pair{p}}
		}
	case "int":
		if f, ok := v.(float64); ok {
			return math.Trunc(f)
		}
	}
	return v
}

func truth(v value) bool {
	switch v := v.(type) {
	case bool:
		return v
	case float64:
		return v != 0
	}
	fail("want a bool, got %T", v)
	return false
}

func num(v value) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	fail("want a number, got %s", typeName(v))
	return 0
}

func pairOf(v value) Pair {
	switch v := v.(type) {
	case Pair:
		return v
	case float64:
		return Pair{v, 0}
	}
	fail("want a pair, got %s", typeName(v))
	return Pair{}
}

func pathOf(v value) *Path {
	switch v := v.(type) {
	case *Path:
		return v
	case Pair:
		return &Path{Knots: []Pair{v}, Pre: []Pair{v}, Post: []Pair{v}}
	}
	fail("want a path, got %s", typeName(v))
	return nil
}

func typeName(v value) string {
	switch v.(type) {
	case float64:
		return "real"
	case Pair:
		return "pair"
	case *Path:
		return "path"
	case []*Path:
		return "path[]"
	case Pen:
		return "pen"
	case string:
		return "string"
	case bool:
		return "bool"
	case Transform:
		return "transform"
	case *array:
		return "array"
	case nil:
		return "nothing"
	}
	return fmt.Sprintf("%T", v)
}

func (e *evaluator) lookup(name string, en *env) value {
	if v := en.find(name); v != nil {
		return v.v
	}
	if v, ok := constants[name]; ok {
		return v
	}
	if p, ok := colors[name]; ok {
		return p
	}
	if d, ok := dashes[name]; ok {
		return Pen{Dash: d, HasDash: true}
	}
	switch name {
	case "defaultpen", "currentpen":
		return e.defPen
	case "dotfactor":
		return e.dotfactor
	case "invisible":
		return Pen{Invisible: true}
	}
	if f, ok := builtins[name]; ok {
		return f
	}
	fail("%s is not in the subset", name)
	return nil
}

func (e *evaluator) eval(x node, en *env) value {
	e.tick()
	switch x := x.(type) {
	case numLit:
		return x.v
	case strLit:
		return x.s
	case ident:
		switch x.name {
		case "true":
			return true
		case "false":
			return false
		}
		return e.lookup(x.name, en)
	case pairLit:
		return Pair{num(e.eval(x.x, en)), num(e.eval(x.y, en))}
	case arrayLit:
		a := &array{}
		for _, el := range x.elems {
			a.elems = append(a.elems, e.eval(el, en))
		}
		return a
	case newArr:
		a := &array{}
		if x.size != nil {
			n := int(num(e.eval(x.size, en)))
			for range n {
				a.elems = append(a.elems, zero(x.typ[:len(x.typ)-2]))
			}
		}
		for _, el := range x.init {
			a.elems = append(a.elems, e.eval(el, en))
		}
		return a
	case cast:
		v := e.eval(x.x, en)
		if x.typ == "int" {
			return math.Trunc(num(v))
		}
		return coerce(x.typ, v)
	case unary:
		v := e.eval(x.x, en)
		if x.op == "!" {
			return !truth(v)
		}
		return e.binop("*", -1.0, v)
	case binary:
		switch x.op {
		case "&&":
			return truth(e.eval(x.l, en)) && truth(e.eval(x.r, en))
		case "||":
			return truth(e.eval(x.l, en)) || truth(e.eval(x.r, en))
		}
		return e.binop(x.op, e.eval(x.l, en), e.eval(x.r, en))
	case cond:
		if truth(e.eval(x.c, en)) {
			return e.eval(x.a, en)
		}
		return e.eval(x.b, en)
	case assign:
		v := e.eval(x.v, en)
		if x.op != "=" {
			v = e.binop(x.op[:1], e.eval(x.target, en), v)
		}
		e.store(x.target, v, en)
		return v
	case incdec:
		v := e.binop("+", e.eval(x.x, en), 1.0)
		e.store(x.x, v, en)
		return v
	case index:
		a, ok := e.eval(x.x, en).(*array)
		if !ok {
			fail("only an array has an index")
		}
		i := int(num(e.eval(x.i, en)))
		if i < 0 || i >= len(a.elems) {
			fail("the index %d is out of the array", i)
		}
		return a.elems[i]
	case member:
		return e.member(e.eval(x.x, en), x.name)
	case callExpr:
		if m, ok := x.fn.(member); ok {
			if a, ok := e.eval(m.x, en).(*array); ok && m.name == "push" && len(x.args) == 1 {
				a.elems = append(a.elems, e.eval(x.args[0], en))
				return nil
			}
		}
		var fn value
		if id, ok := x.fn.(ident); ok && en.find(id.name) == nil && builtins[id.name] != nil {
			fn = builtins[id.name]
		} else {
			fn = e.eval(x.fn, en)
		}
		args := make([]value, len(x.args))
		for i, a := range x.args {
			args[i] = e.eval(a, en)
		}
		return e.call(fn, args, x.names)
	case pathExpr:
		return e.buildPath(x, en)
	}
	fail("unknown expression %T", x)
	return nil
}

func (e *evaluator) member(v value, name string) value {
	switch v := v.(type) {
	case Pair:
		switch name {
		case "x":
			return v.X
		case "y":
			return v.Y
		}
	case *array:
		if name == "length" {
			return float64(len(v.elems))
		}
	}
	fail("%s has no member %s", typeName(v), name)
	return nil
}

func (e *evaluator) store(target node, v value, en *env) {
	switch t := target.(type) {
	case ident:
		if t.name == "dotfactor" {
			e.dotfactor = num(v)
			return
		}
		if t.name == "currentpen" {
			e.defPen = e.defPen.plus(v.(Pen))
			return
		}
		va := en.find(t.name)
		if va == nil {
			fail("%s has no declaration", t.name)
		}
		va.v = coerce(va.typ, v)
	case index:
		a, ok := e.eval(t.x, en).(*array)
		if !ok {
			fail("only an array has an index")
		}
		i := int(num(e.eval(t.i, en)))
		if i < 0 || i > 10000 {
			fail("the index %d is out of range", i)
		}
		for len(a.elems) <= i {
			a.elems = append(a.elems, nil)
		}
		a.elems[i] = v
	default:
		fail("cannot assign to %T", target)
	}
}

func (e *evaluator) call(fn value, args []value, names []string) value {
	switch f := fn.(type) {
	case builtin:
		return f(e, args, names)
	case *userFunc:
		inner := &env{vars: map[string]*variable{}, parent: f.env}
		for i, p := range f.decl.params {
			var v value
			switch {
			case i < len(args):
				v = args[i]
			case p.def != nil:
				v = e.eval(p.def, f.env)
			default:
				fail("%s needs %d arguments", f.decl.name, len(f.decl.params))
			}
			inner.vars[p.name] = &variable{p.typ, coerce(p.typ, v)}
		}
		for _, s := range f.decl.body {
			if r, ok := e.exec(s, inner).(flowReturn); ok {
				return coerce(f.decl.typ, r.v)
			}
		}
		return nil
	case Pen:
		// gray(0.5) and the like: a named pen used as a function.
		if len(args) == 1 {
			g := num(args[0])
			return color(g, g, g)
		}
	case Transform:
		if len(args) == 1 {
			return e.binop("*", f, args[0])
		}
	}
	fail("%s is not a function", typeName(fn))
	return nil
}

func (e *evaluator) binop(op string, a, b value) value {
	switch x := a.(type) {
	case float64:
		switch y := b.(type) {
		case float64:
			switch op {
			case "+":
				return x + y
			case "-":
				return x - y
			case "*":
				return x * y
			case "/":
				return x / y
			case "#":
				return math.Floor(x / y)
			case "%":
				return x - y*math.Floor(x/y)
			case "^":
				return math.Pow(x, y)
			case "<":
				return x < y
			case ">":
				return x > y
			case "<=":
				return x <= y
			case ">=":
				return x >= y
			case "==":
				return x == y
			case "!=":
				return x != y
			}
		case Pair:
			return e.binop(op, Pair{x, 0}, y)
		case Pen:
			if op == "+" {
				// A number in a pen sum is a line width.
				return y.plus(Pen{Width: x, HasWidth: true})
			}
			if op == "*" {
				// A number times a pen scales the color toward black.
				y.R, y.G, y.B = y.R*x, y.G*x, y.B*x
				return y
			}
		case string:
			if op == "*" {
				return y
			}
		}
	case Pair:
		var y Pair
		switch b := b.(type) {
		case Pair:
			y = b
		case float64:
			y = Pair{b, 0}
			if op == "^" {
				if b == 2 {
					return x.mul(x)
				}
				fail("a power of a pair is not in the subset")
			}
		default:
			fail("cannot use %s with pair and %s", op, typeName(b))
		}
		switch op {
		case "+":
			return x.add(y)
		case "-":
			return x.sub(y)
		case "*":
			return x.mul(y)
		case "/":
			return x.div(y)
		case "==":
			return x == y
		case "!=":
			return x != y
		}
	case Pen:
		if y, ok := b.(Pen); ok && op == "+" {
			return x.plus(y)
		}
		if y, ok := b.(float64); ok && op == "*" {
			return e.binop("*", y, x)
		}
		if y, ok := b.(float64); ok && op == "+" {
			return x.plus(Pen{Width: y, HasWidth: true})
		}
	case Transform:
		if op == "*" {
			switch y := b.(type) {
			case Transform:
				return x.then(y)
			case Pair:
				return x.pair(y)
			case float64:
				return x.pair(Pair{y, 0})
			case *Path:
				return x.path(y)
			case []*Path:
				out := make([]*Path, len(y))
				for i, p := range y {
					out[i] = x.path(p)
				}
				return out
			case string:
				return y
			}
		}
	case string:
		switch y := b.(type) {
		case string:
			switch op {
			case "+":
				return x + y
			case "==":
				return x == y
			case "!=":
				return x != y
			}
		case float64:
			if op == "+" {
				return x + fmtNum(y)
			}
		}
	case bool:
		if y, ok := b.(bool); ok {
			switch op {
			case "==":
				return x == y
			case "!=":
				return x != y
			}
		}
	case *Path, []*Path:
		if op == "^^" {
			return append(paths(a), paths(b)...)
		}
	}
	if op == "^^" {
		return append(paths(a), paths(b)...)
	}
	fail("cannot use %s with %s and %s", op, typeName(a), typeName(b))
	return nil
}

func paths(v value) []*Path {
	switch v := v.(type) {
	case []*Path:
		return v
	case *array:
		var out []*Path
		for _, el := range v.elems {
			out = append(out, pathOf(el))
		}
		return out
	}
	return []*Path{pathOf(v)}
}

func fmtNum(f float64) string {
	return strconv.FormatFloat(f, 'g', 6, 64)
}

// buildPath joins the parts of a path expression. A straight join gives a line. A curved join gets its controls from smooth.
func (e *evaluator) buildPath(x pathExpr, en *env) value {
	p := &Path{}
	var curved []bool
	var dirs []Pair
	var fixed []bool
	type ctrl struct{ a, b Pair }
	var given []*ctrl
	for i, part := range x.parts {
		var d Pair
		if x.dirs[i] != nil {
			d = pairOf(e.eval(x.dirs[i], en))
		}
		join := ""
		if i > 0 {
			join = x.joins[i-1]
		}
		if part == nil {
			p.Cycle = true
			curved = append(curved, join == ".." || join == "::")
			fixed = append(fixed, false)
			given = append(given, nil)
			if d != (Pair{}) && dirs[0] == (Pair{}) {
				dirs[0] = d
			}
			if join == "controls" {
				given[len(given)-1] = &ctrl{pairOf(e.eval(x.ctrls[i-1][0], en)), pairOf(e.eval(x.ctrls[i-1][1], en))}
			}
			break
		}
		v := e.eval(part, en)
		var sub *Path
		switch v := v.(type) {
		case *Path:
			sub = v
		case Pair:
			sub = &Path{Knots: []Pair{v}, Pre: []Pair{v}, Post: []Pair{v}}
		case float64:
			q := Pair{v, 0}
			sub = &Path{Knots: []Pair{q}, Pre: []Pair{q}, Post: []Pair{q}}
		default:
			fail("a path cannot have a %s", typeName(v))
		}
		if len(sub.Knots) == 0 {
			continue
		}
		if len(p.Knots) > 0 {
			curved = append(curved, join == ".." || join == "::")
			fixed = append(fixed, false)
			given = append(given, nil)
			if join == "controls" {
				given[len(given)-1] = &ctrl{pairOf(e.eval(x.ctrls[i-1][0], en)), pairOf(e.eval(x.ctrls[i-1][1], en))}
			}
		}
		for k, q := range sub.Knots {
			p.Knots = append(p.Knots, q)
			p.Pre = append(p.Pre, sub.Pre[k])
			p.Post = append(p.Post, sub.Post[k])
			dirs = append(dirs, Pair{})
			if k < len(sub.Knots)-1 {
				curved = append(curved, false)
				fixed = append(fixed, true)
				given = append(given, nil)
			}
		}
		if sub.Cycle {
			fail("a cyclic path inside a path is not in the subset")
		}
		if d != (Pair{}) {
			dirs[len(dirs)-1] = d
		}
	}
	n := len(p.Knots)
	if n == 0 {
		return p
	}
	segs := p.segments()
	curved = curved[:segs]
	for i := 0; i < segs; i++ {
		j := (i + 1) % n
		a, b := p.Knots[i], p.Knots[j]
		switch {
		case fixed[i]:
		case given[i] != nil:
			p.Post[i], p.Pre[j] = given[i].a, given[i].b
		default:
			p.Post[i] = a.add(b.sub(a).scale(1.0 / 3))
			p.Pre[j] = a.add(b.sub(a).scale(2.0 / 3))
		}
	}
	for i := range curved {
		if fixed[i] || given[i] != nil {
			curved[i] = false
		}
	}
	smooth(p, curved, dirs)
	return p
}
