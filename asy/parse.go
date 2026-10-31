// Package asy runs a subset of the Asymptote language and draws the picture as SVG. It knows the commands that olympiad figures use most: points, paths, circles, pens, labels, loops and small functions. A program that uses more than the subset gives an error, and the caller shows the source instead.
package asy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type token struct {
	kind  byte // 'i' identifier, 'n' number, 's' string, 'p' punctuation, 0 end
	text  string
	num   float64
	space bool // a space comes before the token
}

var puncts = []string{"---", "--", "..", "^^", "::", "==", "!=", "<=", ">=", "&&", "||", "++", "+=", "-=", "*=", "/=", "**"}

func lex(src string) ([]token, error) {
	var ts []token
	i := 0
	space := false
	// Some sources lost their line breaks. Then a comment ends where the code goes on.
	oneLine := !strings.Contains(strings.TrimSpace(src), "\n")
	for i < len(src) {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
			space = true
			continue
		case strings.HasPrefix(src[i:], "//") && oneLine:
			i = commentEnd(src, i+2)
			space = true
			continue
		case strings.HasPrefix(src[i:], "//"):
			for i < len(src) && src[i] != '\n' {
				i++
			}
			space = true
			continue
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return nil, fmt.Errorf("the comment has no end")
			}
			i += end + 4
			space = true
			continue
		}
		t := token{space: space}
		space = false
		switch {
		case isLetter(c):
			j := i
			for j < len(src) && (isLetter(src[j]) || isDigit(src[j])) {
				j++
			}
			t.kind, t.text = 'i', src[i:j]
			i = j
		case isDigit(c) || (c == '.' && i+1 < len(src) && isDigit(src[i+1])):
			j := i
			for j < len(src) && isDigit(src[j]) {
				j++
			}
			if j < len(src) && src[j] == '.' && !strings.HasPrefix(src[j:], "..") {
				j++
				for j < len(src) && isDigit(src[j]) {
					j++
				}
			}
			if j < len(src) && (src[j] == 'e' || src[j] == 'E') {
				k := j + 1
				if k < len(src) && (src[k] == '+' || src[k] == '-') {
					k++
				}
				if k < len(src) && isDigit(src[k]) {
					j = k
					for j < len(src) && isDigit(src[j]) {
						j++
					}
				}
			}
			v, err := strconv.ParseFloat(src[i:j], 64)
			if err != nil {
				return nil, err
			}
			t.kind, t.text, t.num = 'n', src[i:j], v
			i = j
		case c == '"' || c == '\'':
			var b strings.Builder
			j := i + 1
			for j < len(src) && src[j] != c {
				if src[j] == '\\' && j+1 < len(src) && (src[j+1] == c || src[j+1] == '\\') {
					if src[j+1] == c {
						j++
					} else {
						b.WriteByte('\\')
						j++
					}
				}
				b.WriteByte(src[j])
				j++
			}
			if j >= len(src) {
				return nil, fmt.Errorf("the string has no end")
			}
			t.kind, t.text = 's', b.String()
			i = j + 1
		default:
			t.kind, t.text = 'p', string(c)
			for _, p := range puncts {
				if strings.HasPrefix(src[i:], p) {
					t.text = p
					break
				}
			}
			i += len(t.text)
		}
		ts = append(ts, t)
	}
	return append(ts, token{}), nil
}

var codeStart = regexp.MustCompile(`^(?:(?:real|int|pair|path|pen|bool|string|guide|point)\s+[A-Za-z_]\w*\s*[=,;\[]|(?:draw|label|dot|fill|filldraw|size|unitsize|defaultpen|clip)\s*\(|for\s*\(|import\s)`)

// commentEnd finds the end of a comment in a source with no line breaks: after the first semicolon, before a second comment, or before a word that starts code.
func commentEnd(src string, i int) int {
	for i < len(src) && src[i] == ' ' {
		i++
	}
	for i < len(src) && isLetter(src[i]) {
		i++
	}
	for k := i; k < len(src); k++ {
		switch {
		case strings.HasPrefix(src[k:], "//"):
			return k
		case src[k] == ';':
			return k + 1
		case (k == 0 || !isLetter(src[k-1]) && !isDigit(src[k-1])) && codeStart.MatchString(src[k:]):
			return k
		}
	}
	return len(src)
}

func isLetter(c byte) bool { return c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }
func isDigit(c byte) bool  { return c >= '0' && c <= '9' }

// The nodes of the syntax tree.
type (
	numLit   struct{ v float64 }
	strLit   struct{ s string }
	ident    struct{ name string }
	pairLit  struct{ x, y node }
	arrayLit struct{ elems []node }
	callExpr struct {
		fn    node
		args  []node
		names []string // the name of each argument, or ""
	}
	binary struct {
		op   string
		l, r node
	}
	unary struct {
		op string
		x  node
	}
	index  struct{ x, i node }
	member struct {
		x    node
		name string
	}
	assign struct {
		op     string
		target node
		v      node
	}
	cond   struct{ c, a, b node }
	incdec struct {
		op string
		x  node
	}
	cast struct {
		typ string
		x   node
	}
	newArr struct {
		typ  string
		size node
		init []node
	}
	pathExpr struct {
		parts []node   // operands; a nil part is cycle
		joins []string // "--", "..", "---" or "::" between parts
		dirs  []node   // a direction at each part, or nil
		ctrls [][2]node
	}

	declStmt struct {
		typ   string
		names []string
		vals  []node
	}
	funcDecl struct {
		typ    string
		name   string
		params []param
		body   []node
	}
	exprStmt  struct{ x node }
	blockStmt struct{ body []node }
	ifStmt    struct{ c, then, els node }
	forStmt   struct{ init, c, post, body node }
	eachStmt  struct {
		typ, name string
		over      node
		body      node
	}
	whileStmt  struct{ c, body node }
	returnStmt struct{ x node }
	breakStmt  struct{}
	contStmt   struct{}
)

type node any

type param struct {
	typ, name string
	def       node
}

var units = map[string]bool{"cm": true, "mm": true, "inch": true, "inches": true, "pt": true, "bp": true}

// aliases are the types of the module geometry that the subset treats as plain types.
var aliases = map[string]string{"point": "pair", "circle": "path", "vector": "pair"}

var types = map[string]bool{"point": true, "circle": true, "vector": true, "real": true, "int": true, "pair": true, "path": true, "guide": true, "pen": true, "bool": true, "string": true, "transform": true, "void": true}

type parser struct {
	ts []token
	at int
}

func parse(src string) (prog []node, err error) {
	ts, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{ts: ts}
	defer func() {
		if r := recover(); r != nil {
			if e, ok := r.(parseError); ok {
				err = e
				return
			}
			panic(r)
		}
	}()
	for p.peek().kind != 0 {
		prog = append(prog, p.stmt())
	}
	return prog, err
}

type parseError struct{ msg string }

func (e parseError) Error() string { return e.msg }

func (p *parser) peek() token { return p.ts[p.at] }
func (p *parser) peekAt(n int) token {
	if p.at+n < len(p.ts) {
		return p.ts[p.at+n]
	}
	return token{}
}
func (p *parser) next() token {
	t := p.ts[p.at]
	if t.kind != 0 {
		p.at++
	}
	return t
}
func (p *parser) is(s string) bool {
	t := p.peek()
	return (t.kind == 'p' || t.kind == 'i') && t.text == s
}
func (p *parser) accept(s string) bool {
	if p.is(s) {
		p.at++
		return true
	}
	return false
}
func (p *parser) expect(s string) {
	if !p.accept(s) {
		p.fail("want %q, got %q", s, p.peek().text)
	}
}
func (p *parser) fail(f string, a ...any) { panic(parseError{fmt.Sprintf(f, a...)}) }
func (p *parser) name() string {
	t := p.next()
	if t.kind != 'i' {
		p.fail("want a name, got %q", t.text)
	}
	return t.text
}

// declType reads a type at the cursor when a declaration starts there, and returns "" when it does not.
func (p *parser) declType() string {
	t := p.peek()
	if t.kind != 'i' || !types[t.text] {
		return ""
	}
	n := 1
	typ := t.text
	if a, ok := aliases[typ]; ok {
		typ = a
	}
	for p.peekAt(n).text == "[" && p.peekAt(n+1).text == "]" {
		typ += "[]"
		n += 2
	}
	if p.peekAt(n).kind != 'i' {
		return ""
	}
	p.at += n
	return typ
}

func (p *parser) stmt() node {
	switch {
	case p.accept(";"):
		return blockStmt{}
	case p.accept("{"):
		var body []node
		for !p.accept("}") {
			if p.peek().kind == 0 {
				p.fail("the block has no end")
			}
			body = append(body, p.stmt())
		}
		return blockStmt{body}
	case p.is("import") || p.is("include") || p.is("access"):
		for !p.accept(";") && p.peek().kind != 0 {
			p.next()
		}
		return blockStmt{}
	case p.accept("if"):
		p.expect("(")
		c := p.expr()
		p.expect(")")
		s := ifStmt{c: c, then: p.stmt()}
		if p.accept("else") {
			s.els = p.stmt()
		}
		return s
	case p.accept("while"):
		p.expect("(")
		c := p.expr()
		p.expect(")")
		return whileStmt{c, p.stmt()}
	case p.accept("for"):
		p.expect("(")
		save := p.at
		if typ := p.declType(); typ != "" {
			n := p.name()
			if p.accept(":") {
				over := p.expr()
				p.expect(")")
				return eachStmt{typ, n, over, p.stmt()}
			}
		}
		p.at = save
		var s forStmt
		if !p.is(";") {
			s.init = p.simple()
		}
		p.expect(";")
		if !p.is(";") {
			s.c = p.expr()
		}
		p.expect(";")
		if !p.is(")") {
			s.post = exprStmt{p.expr()}
		}
		p.expect(")")
		s.body = p.stmt()
		return s
	case p.accept("return"):
		if p.accept(";") {
			return returnStmt{}
		}
		x := p.expr()
		p.expect(";")
		return returnStmt{x}
	case p.accept("break"):
		p.expect(";")
		return breakStmt{}
	case p.accept("continue"):
		p.expect(";")
		return contStmt{}
	}
	save := p.at
	if typ := p.declType(); typ != "" && p.peekAt(1).text == "(" {
		name := p.name()
		p.expect("(")
		var ps []param
		for !p.accept(")") {
			pt := p.declType()
			if pt == "" {
				p.fail("the parameter has no type")
			}
			pr := param{typ: pt, name: p.name()}
			if p.accept("=") {
				pr.def = p.expr()
			}
			ps = append(ps, pr)
			p.accept(",")
		}
		if p.is("{") {
			body := p.stmt().(blockStmt)
			return funcDecl{typ, name, ps, body.body}
		}
		p.fail("a function needs a body")
	}
	p.at = save
	s := p.simple()
	p.expect(";")
	return s
}

// simple reads a declaration or an expression, with no semicolon.
func (p *parser) simple() node {
	if typ := p.declType(); typ != "" {
		d := declStmt{typ: typ}
		for {
			n := p.name()
			for p.accept("[") {
				p.expect("]")
				d.typ += "[]"
			}
			var v node
			if p.accept("=") {
				v = p.expr()
			}
			d.names = append(d.names, n)
			d.vals = append(d.vals, v)
			if !p.accept(",") {
				return d
			}
		}
	}
	return exprStmt{p.expr()}
}

func (p *parser) expr() node {
	l := p.ternary()
	for _, op := range []string{"=", "+=", "-=", "*=", "/="} {
		if p.accept(op) {
			return assign{op, l, p.expr()}
		}
	}
	return l
}

func (p *parser) ternary() node {
	c := p.binary(0)
	if p.accept("?") {
		a := p.expr()
		p.expect(":")
		return cond{c, a, p.ternary()}
	}
	return c
}

var levels = [][]string{{"||"}, {"&&"}, {"==", "!="}, {"<", ">", "<=", ">="}, {"^^"}}

func (p *parser) binary(level int) node {
	if level == len(levels) {
		return p.path()
	}
	l := p.binary(level + 1)
	for {
		matched := false
		for _, op := range levels[level] {
			if p.accept(op) {
				l = binary{op, l, p.binary(level + 1)}
				matched = true
				break
			}
		}
		if !matched {
			return l
		}
	}
}

func (p *parser) isJoin() bool {
	return p.is("--") || p.is("..") || p.is("---") || p.is("::")
}

func (p *parser) path() node {
	first := p.pathPart()
	var dir node
	if p.is("{") {
		dir = p.direction()
	}
	if !p.isJoin() {
		if dir != nil {
			p.fail("a direction needs a path")
		}
		return first
	}
	pe := pathExpr{parts: []node{first}, dirs: []node{dir}}
	for p.isJoin() {
		j := p.next().text
		var ctrl [2]node
		if j == ".." && p.accept("tension") {
			p.accept("atleast")
			p.additive()
			if p.accept("and") {
				p.additive()
			}
			p.expect("..")
		} else if j == ".." && p.accept("controls") {
			ctrl[0] = p.additive()
			ctrl[1] = ctrl[0]
			if p.accept("and") {
				ctrl[1] = p.additive()
			}
			p.expect("..")
			j = "controls"
		}
		var d node
		if p.is("{") {
			d = p.direction()
		}
		var part node
		if !p.accept("cycle") {
			part = p.pathPart()
		}
		if p.is("{") {
			d = p.direction()
		}
		pe.joins = append(pe.joins, j)
		pe.ctrls = append(pe.ctrls, ctrl)
		pe.parts = append(pe.parts, part)
		pe.dirs = append(pe.dirs, d)
		if part == nil {
			break
		}
	}
	return pe
}

func (p *parser) direction() node {
	p.expect("{")
	d := p.expr()
	p.expect("}")
	return d
}

func (p *parser) pathPart() node { return p.additive() }

func (p *parser) additive() node {
	l := p.multiplicative()
	for {
		switch {
		case p.accept("+"):
			l = binary{"+", l, p.multiplicative()}
		case p.is("-") && !p.is("--"):
			p.next()
			l = binary{"-", l, p.multiplicative()}
		default:
			return l
		}
	}
}

func (p *parser) multiplicative() node {
	l := p.unaryExpr()
	for {
		switch {
		case p.accept("*"):
			l = binary{"*", l, p.unaryExpr()}
		case p.accept("/"):
			l = binary{"/", l, p.unaryExpr()}
		case p.accept("%"):
			l = binary{"%", l, p.unaryExpr()}
		case p.accept("#"):
			l = binary{"#", l, p.unaryExpr()}
		default:
			return l
		}
	}
}

func (p *parser) unaryExpr() node {
	switch {
	case p.accept("-"):
		return unary{"-", p.unaryExpr()}
	case p.accept("+"):
		return p.unaryExpr()
	case p.accept("!"):
		return unary{"!", p.unaryExpr()}
	case p.accept("++"):
		return incdec{"+=", p.unaryExpr()}
	case p.is("(") && types[p.peekAt(1).text] && p.peekAt(2).text == ")":
		p.next()
		typ := p.next().text
		p.next()
		return cast{typ, p.unaryExpr()}
	}
	return p.power()
}

func (p *parser) power() node {
	b := p.postfix()
	if p.accept("^") || p.accept("**") {
		return binary{"^", b, p.unaryExpr()}
	}
	return b
}

func (p *parser) postfix() node {
	x := p.primary()
	for {
		switch {
		case p.is("(") && !p.peek().space || p.is("(") && isCallee(x):
			p.next()
			c := callExpr{fn: x}
			for !p.accept(")") {
				name := ""
				if p.peek().kind == 'i' && p.peekAt(1).text == "=" {
					name = p.next().text
					p.next()
				}
				c.args = append(c.args, p.expr())
				c.names = append(c.names, name)
				if !p.accept(",") && !p.is(")") {
					p.fail("want , or ) in a call, got %q", p.peek().text)
				}
			}
			x = c
		case p.accept("["):
			i := p.expr()
			p.expect("]")
			x = index{x, i}
		case p.is(".") && p.peekAt(1).kind == 'i':
			p.next()
			x = member{x, p.next().text}
		case p.accept("++"):
			x = incdec{"+=", x}
		default:
			return x
		}
	}
}

func isCallee(x node) bool {
	switch x.(type) {
	case ident, member:
		return true
	}
	return false
}

func (p *parser) primary() node {
	t := p.next()
	switch t.kind {
	case 'n':
		var x node = numLit{t.num}
		// A number before a name or a bracket multiplies it, as in 2A, 3cm or 0.5(A+B).
		if n := p.peek(); !n.space && (n.kind == 'i' || n.text == "(") || units[n.text] {
			return binary{"*", x, p.power()}
		}
		return x
	case 's':
		return strLit{t.text}
	case 'i':
		if t.text == "new" {
			typ := p.name()
			n := newArr{typ: typ + "[]"}
			p.expect("[")
			if !p.is("]") {
				n.size = p.expr()
			}
			p.expect("]")
			if p.is("{") {
				n.init = p.braceList()
			}
			return n
		}
		if t.text == "true" || t.text == "false" {
			return ident{t.text}
		}
		return ident{t.text}
	}
	switch t.text {
	case "(":
		x := p.expr()
		if p.accept(",") {
			y := p.expr()
			if p.accept(",") {
				p.fail("a triple is not in the subset")
			}
			p.expect(")")
			return pairLit{x, y}
		}
		p.expect(")")
		return x
	case "{":
		p.at--
		return arrayLit{p.braceList()}
	}
	p.fail("unexpected %q", t.text)
	return nil
}

func (p *parser) braceList() []node {
	p.expect("{")
	var es []node
	for !p.accept("}") {
		es = append(es, p.expr())
		if !p.accept(",") && !p.is("}") {
			p.fail("want , or } in a list")
		}
	}
	return es
}
