package asy

import (
	"fmt"
	"math"
	"strings"
)

// Figure is a picture laid out for a page: an SVG drawing and the labels to put over it.
type Figure struct {
	SVG string
	// Width and Height are the size of the figure in CSS pixels.
	Width, Height float64
	Labels        []Placed
}

// Placed is a label with its place in the figure. Left and Top are percents of the width and the height. Size is the font size as a percent of the width. DX and DY move the text box by a share of its own size, and MX and MY by a share of its font size.
type Placed struct {
	Text           string
	Left, Top      float64
	Size           float64
	DX, DY, MX, MY float64
	Color          string
}

const (
	minWidth = 240.0
	maxWidth = 560.0
	pxPerBp  = 4.0 / 3
	// defaultSize is the size in points of a figure that gives no size and has small units.
	defaultSize = 200.0
)

type box struct {
	lo, hi Pair
	ok     bool
}

func (b *box) add(p Pair, pad float64) {
	lo, hi := Pair{p.X - pad, p.Y - pad}, Pair{p.X + pad, p.Y + pad}
	if !b.ok {
		b.lo, b.hi, b.ok = lo, hi, true
		return
	}
	b.lo = Pair{math.Min(b.lo.X, lo.X), math.Min(b.lo.Y, lo.Y)}
	b.hi = Pair{math.Max(b.hi.X, hi.X), math.Max(b.hi.Y, hi.Y)}
}

// Layout scales the picture and gives the figure. The id must be unique on the page, as the clip path uses it.
func (pic *Picture) Layout(id string) (*Figure, error) {
	var user box
	for _, it := range pic.Items {
		if pic.Clip != nil {
			break
		}
		if lo, hi, ok := it.Path.bounds(); ok {
			user.add(lo, 0)
			user.add(hi, 0)
		}
	}
	if pic.Clip != nil {
		lo, hi, _ := pic.Clip.bounds()
		user.add(lo, 0)
		user.add(hi, 0)
	}
	if !user.ok {
		return nil, fmt.Errorf("the picture is empty")
	}
	w, h := user.hi.X-user.lo.X, user.hi.Y-user.lo.Y
	sx, sy := 1.0, 1.0
	switch {
	case pic.Unit != (Pair{}):
		sx, sy = pic.Unit.X, pic.Unit.Y
	case pic.Size != (Pair{}):
		s := math.Inf(1)
		if pic.Size.X > 0 && w > 0 {
			s = pic.Size.X / w
		}
		if pic.Size.Y > 0 && h > 0 {
			s = math.Min(s, pic.Size.Y/h)
		}
		if !math.IsInf(s, 1) {
			sx, sy = s, s
		}
	case math.Max(w, h) < defaultSize/4:
		// With no size, one unit is one point, and a figure in small units is too small to read. Asymptote users then see a dot, but the source means a figure of usual size.
		sx = defaultSize / math.Max(w, h)
		sy = sx
	}
	at := func(p Pair) Pair { return Pair{p.X * sx, -p.Y * sy} }
	var b box
	for _, it := range pic.Items {
		pad := it.Pen.Width / 2
		if it.Dot {
			pad = it.R
		}
		if pic.Clip != nil {
			continue
		}
		ps, _ := it.Path.flat()
		for _, p := range ps {
			b.add(at(p), pad)
		}
	}
	if pic.Clip != nil {
		ps, _ := pic.Clip.flat()
		for _, p := range ps {
			b.add(at(p), 0)
		}
	}
	for _, l := range pic.Labels {
		c := at(l.At)
		fw, fh := textSize(l.Text, font(l.Pen))
		u := l.Align
		if m := math.Max(math.Abs(u.X), math.Abs(u.Y)); m > 0 {
			u = u.scale(1 / m)
		}
		g := gap(l)
		mid := Pair{c.X + u.X*fw/2 + g.X, c.Y - u.Y*fh/2 + g.Y}
		b.add(Pair{mid.X - fw/2, mid.Y - fh/2}, 0.4*font(l.Pen))
		b.add(Pair{mid.X + fw/2, mid.Y + fh/2}, 0.4*font(l.Pen))
	}
	bw, bh := b.hi.X-b.lo.X, b.hi.Y-b.lo.Y
	if bw <= 0 || bh <= 0 || bw > 1e6 || bh > 1e6 {
		return nil, fmt.Errorf("the picture has no area")
	}
	k := pxPerBp
	k = math.Max(k, minWidth/bw)
	k = math.Min(k, maxWidth/bw)
	k = math.Min(k, math.Max(maxWidth/bh, minWidth/bw/2))
	f := &Figure{Width: bw * k, Height: bh * k}
	var s strings.Builder
	fmt.Fprintf(&s, `<svg xmlns="http://www.w3.org/2000/svg" viewBox="%s %s %s %s" width="%s" height="%s" role="img">`, n(b.lo.X), n(b.lo.Y), n(bw), n(bh), n(f.Width), n(f.Height))
	if pic.Clip != nil {
		fmt.Fprintf(&s, `<clipPath id="%s"><path d="%s"/></clipPath><g clip-path="url(#%s)">`, id, svgPath(pic.Clip, at), id)
	}
	s.WriteString(`<g fill="none" stroke-linecap="round" stroke-linejoin="round">`)
	for _, it := range pic.Items {
		if it.Pen.Invisible {
			continue
		}
		if it.Mark {
			s.WriteString(mark(it, at))
			continue
		}
		if it.Dot {
			c := at(it.Path.Knots[0])
			fmt.Fprintf(&s, `<circle cx="%s" cy="%s" r="%s" fill="%s"%s/>`, n(c.X), n(c.Y), n(it.R), rgb(it.Pen), opac(it.Pen, "fill"))
			continue
		}
		d := svgPath(it.Path, at)
		if it.Fill {
			fmt.Fprintf(&s, `<path d="%s" fill="%s" stroke="none"%s/>`, d, rgb(it.FillPen), opac(it.FillPen, "fill"))
		}
		if it.Stroke {
			fmt.Fprintf(&s, `<path d="%s" stroke="%s" stroke-width="%s"%s%s/>`, d, rgb(it.Pen), n(width(it.Pen)), dash(it.Pen), opac(it.Pen, "stroke"))
			if it.Arrow == 1 || it.Arrow == 2 {
				s.WriteString(arrow(it.Path, it.Pen, at, false))
			}
			if it.Arrow == 2 || it.Arrow == 3 {
				s.WriteString(arrow(it.Path, it.Pen, at, true))
			}
		}
	}
	s.WriteString(`</g>`)
	if pic.Clip != nil {
		s.WriteString(`</g>`)
	}
	s.WriteString(`</svg>`)
	f.SVG = s.String()
	for _, l := range pic.Labels {
		c := at(l.At)
		u := l.Align
		m := math.Max(math.Abs(u.X), math.Abs(u.Y))
		p := Placed{Text: l.Text, Left: 100 * (c.X - b.lo.X) / bw, Top: 100 * (c.Y - b.lo.Y) / bh, Size: 100 * font(l.Pen) / bw, DX: -0.5, DY: -0.5}
		if m > 0 {
			p.DX += 0.5 * u.X / m
			p.DY -= 0.5 * u.Y / m
			g := gap(l)
			p.MX, p.MY = g.X/font(l.Pen), g.Y/font(l.Pen)
		}
		if l.Pen.HasColor && (l.Pen.R != 0 || l.Pen.G != 0 || l.Pen.B != 0) {
			p.Color = rgb(l.Pen)
		}
		f.Labels = append(f.Labels, p)
	}
	return f, nil
}

// gap gives the move in points from the point to the text box, in SVG directions. It is 0.3 of the font size, longer for a longer align vector, plus the gap of the label.
func gap(l Label) Pair {
	if l.Align == (Pair{}) {
		return Pair{}
	}
	g := 0.3*font(l.Pen)*l.Align.abs() + l.Gap
	v := l.Align.unit()
	return Pair{g * v.X, -g * v.Y}
}

func font(p Pen) float64 {
	if p.Font > 0 {
		return p.Font
	}
	return 12
}

func width(p Pen) float64 {
	if p.Width > 0 {
		return p.Width
	}
	return 0.5
}

// textSize guesses the size of a label in points from its TeX source.
func textSize(s string, size float64) (float64, float64) {
	chars := 0
	skip := false
	for _, r := range s {
		switch {
		case r == '\\':
			skip = true
		case skip && (r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'):
		case r == '$' || r == '{' || r == '}' || r == '^' || r == '_' || r == ' ':
			skip = false
		default:
			skip = false
			chars++
		}
	}
	h := size
	if strings.Contains(s, `\frac`) || strings.Contains(s, `\dfrac`) {
		h *= 2
	}
	return math.Max(1, float64(chars)) * size * 0.6, h
}

func n(f float64) string {
	s := fmt.Sprintf("%.2f", f)
	s = strings.TrimRight(strings.TrimRight(s, "0"), ".")
	if s == "-0" {
		return "0"
	}
	return s
}

func svgPath(p *Path, at func(Pair) Pair) string {
	var s strings.Builder
	k := at(p.Knots[0])
	fmt.Fprintf(&s, "M%s %s", n(k.X), n(k.Y))
	for i := 0; i < p.segments(); i++ {
		a, b, c, d := p.seg(i)
		if b == a.add(d.sub(a).scale(1.0/3)) && c == a.add(d.sub(a).scale(2.0/3)) {
			if p.Cycle && i == p.segments()-1 {
				break
			}
			q := at(d)
			fmt.Fprintf(&s, "L%s %s", n(q.X), n(q.Y))
			continue
		}
		b, c, d = at(b), at(c), at(d)
		fmt.Fprintf(&s, "C%s %s %s %s %s %s", n(b.X), n(b.Y), n(c.X), n(c.Y), n(d.X), n(d.Y))
	}
	if p.Cycle {
		s.WriteString("Z")
	}
	return s.String()
}

func rgb(p Pen) string {
	c := func(f float64) int { return int(math.Round(255 * math.Max(0, math.Min(1, f)))) }
	return fmt.Sprintf("#%02x%02x%02x", c(p.R), c(p.G), c(p.B))
}

func opac(p Pen, what string) string {
	if p.HasOpac && p.Opacity < 1 {
		return fmt.Sprintf(` %s-opacity="%s"`, what, n(p.Opacity))
	}
	return ""
}

func dash(p Pen) string {
	if !p.HasDash || strings.TrimSpace(p.Dash) == "" {
		return ""
	}
	var out []string
	for _, f := range strings.Fields(p.Dash) {
		var v float64
		if _, err := fmt.Sscan(f, &v); err != nil {
			return ""
		}
		// Asymptote scales the dash pattern by the line width.
		out = append(out, n(math.Max(v*width(p), 0.01)))
	}
	return fmt.Sprintf(` stroke-dasharray="%s"`, strings.Join(out, " "))
}

func arrow(p *Path, pen Pen, at func(Pair) Pair, begin bool) string {
	t := float64(p.segments())
	if begin {
		t = 0
	}
	tip := at(p.point(t))
	d := p.dir(t)
	if begin {
		d = d.scale(-1)
	}
	d = Pair{d.X, -d.Y}.unit()
	size := math.Max(15*width(pen), 6)
	back := tip.sub(d.scale(size))
	side := Pair{-d.Y, d.X}.scale(size * math.Tan(15*math.Pi/180))
	a, b := back.add(side), back.sub(side)
	return fmt.Sprintf(`<path d="M%s %sL%s %sL%s %sZ" fill="%s" stroke="none"/>`, n(tip.X), n(tip.Y), n(a.X), n(a.Y), n(b.X), n(b.Y), rgb(pen))
}

func mark(it Item, at func(Pair) Pair) string {
	c := at(it.Path.Knots[0])
	// The picture turns upside down in SVG, so the angles change sign.
	pt := func(deg, r float64) Pair { d := dirDeg(-deg); return c.add(d.scale(r)) }
	var d string
	if it.Right {
		a, b := pt(it.A1, it.R), pt(it.A2, it.R)
		m := a.add(b).sub(c)
		d = fmt.Sprintf("M%s %sL%s %sL%s %s", n(a.X), n(a.Y), n(m.X), n(m.Y), n(b.X), n(b.Y))
	} else {
		a, b := pt(it.A1, it.R), pt(it.A2, it.R)
		large := 0
		if it.A2-it.A1 > 180 {
			large = 1
		}
		d = fmt.Sprintf("M%s %sA%s %s 0 %d 0 %s %s", n(a.X), n(a.Y), n(it.R), n(it.R), large, n(b.X), n(b.Y))
	}
	return fmt.Sprintf(`<path d="%s" stroke="%s" stroke-width="%s"/>`, d, rgb(it.Pen), n(width(it.Pen)))
}
