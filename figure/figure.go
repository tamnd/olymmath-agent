// Package figure finds the figure of a problem on a page of a PDF and crops it (spec 4.3). The text layer of the page gives the box of each word. The figure is the ink of the page image that no word covers, in the band of the page between two texts of the problem, with the short labels next to it.
package figure

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

// Box is a part of a page in points from the top left corner.
type Box struct {
	X0, Y0, X1, Y1 float64
}

// W is the width of the box.
func (b Box) W() float64 { return b.X1 - b.X0 }

// H is the height of the box.
func (b Box) H() float64 { return b.Y1 - b.Y0 }

// Empty tells if the box has no area.
func (b Box) Empty() bool { return b.X1 <= b.X0 || b.Y1 <= b.Y0 }

// String gives the box as -box takes it: x0,y0,x1,y1.
func (b Box) String() string {
	f := func(v float64) string { return strconv.FormatFloat(v, 'f', 1, 64) }
	return f(b.X0) + "," + f(b.Y0) + "," + f(b.X1) + "," + f(b.Y1)
}

// Union is the smallest box that holds b and c. The union with an empty box is the other box.
func (b Box) Union(c Box) Box {
	if b.Empty() {
		return c
	}
	if c.Empty() {
		return b
	}
	return Box{math.Min(b.X0, c.X0), math.Min(b.Y0, c.Y0), math.Max(b.X1, c.X1), math.Max(b.Y1, c.Y1)}
}

// Grow gives the box with d more on each side.
func (b Box) Grow(d float64) Box { return Box{b.X0 - d, b.Y0 - d, b.X1 + d, b.Y1 + d} }

// Overlaps tells if the two boxes share a point.
func (b Box) Overlaps(c Box) bool {
	return b.X0 <= c.X1 && c.X0 <= b.X1 && b.Y0 <= c.Y1 && c.Y0 <= b.Y1
}

// Within tells if c holds b.
func (b Box) Within(c Box) bool {
	return b.X0 >= c.X0 && b.X1 <= c.X1 && b.Y0 >= c.Y0 && b.Y1 <= c.Y1
}

// Clip gives the part of b inside c.
func (b Box) Clip(c Box) Box {
	return Box{math.Max(b.X0, c.X0), math.Max(b.Y0, c.Y0), math.Min(b.X1, c.X1), math.Min(b.Y1, c.Y1)}
}

// Out gives the box in whole points that holds b, for a tool that takes whole numbers.
func (b Box) Out() Box {
	return Box{math.Floor(b.X0), math.Floor(b.Y0), math.Ceil(b.X1), math.Ceil(b.Y1)}
}

// ParseBox reads a box written as x0,y0,x1,y1 in points.
func ParseBox(s string) (Box, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return Box{}, fmt.Errorf("the box %q is not x0,y0,x1,y1", s)
	}
	var v [4]float64
	for i, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return Box{}, fmt.Errorf("the box %q is not x0,y0,x1,y1: %w", s, err)
		}
		v[i] = f
	}
	b := Box{v[0], v[1], v[2], v[3]}
	if b.Empty() || b.X0 < 0 || b.Y0 < 0 {
		return Box{}, fmt.Errorf("the box %q has no area", s)
	}
	return b, nil
}

// Word is one word of the text layer.
type Word struct {
	Box
	Text string
}

// Line is one line of the text layer.
type Line struct {
	Box
	Words []Word
}

// Text is the words of the line with a space between them.
func (l Line) Text() string {
	ws := make([]string, len(l.Words))
	for i, w := range l.Words {
		ws[i] = w.Text
	}
	return strings.Join(ws, " ")
}

// Page is the text layer of one page.
type Page struct {
	Width, Height float64
	Lines         []Line
}

// Bounds is the box of the whole page.
func (p Page) Bounds() Box { return Box{0, 0, p.Width, p.Height} }

type xmlBox struct {
	XMin float64 `xml:"xMin,attr"`
	YMin float64 `xml:"yMin,attr"`
	XMax float64 `xml:"xMax,attr"`
	YMax float64 `xml:"yMax,attr"`
}

func (b xmlBox) box() Box { return Box{b.XMin, b.YMin, b.XMax, b.YMax} }

type xmlPage struct {
	Width  float64 `xml:"width,attr"`
	Height float64 `xml:"height,attr"`
	Flows  []struct {
		Blocks []struct {
			Lines []struct {
				xmlBox
				Words []struct {
					xmlBox
					Text string `xml:",chardata"`
				} `xml:"word"`
			} `xml:"line"`
		} `xml:"block"`
	} `xml:"flow"`
}

// ParseLayout reads the output of pdftotext -bbox-layout for one page.
func ParseLayout(b []byte) (Page, error) {
	// A word of a PDF can hold a control character, which XML does not allow.
	b = bytes.Map(func(r rune) rune {
		if r < ' ' && r != '\t' && r != '\n' && r != '\r' {
			return ' '
		}
		return r
	}, b)
	d := xml.NewDecoder(bytes.NewReader(b))
	d.Strict = false
	d.AutoClose = xml.HTMLAutoClose
	d.Entity = xml.HTMLEntity
	for {
		t, err := d.Token()
		if err != nil {
			return Page{}, errors.New("the layout has no page")
		}
		se, ok := t.(xml.StartElement)
		if !ok || se.Name.Local != "page" {
			continue
		}
		var xp xmlPage
		if err := d.DecodeElement(&xp, &se); err != nil {
			return Page{}, fmt.Errorf("the layout: %w", err)
		}
		p := Page{Width: xp.Width, Height: xp.Height}
		for _, f := range xp.Flows {
			for _, bl := range f.Blocks {
				for _, l := range bl.Lines {
					line := Line{Box: l.box()}
					for _, w := range l.Words {
						line.Words = append(line.Words, Word{Box: w.box(), Text: strings.TrimSpace(w.Text)})
					}
					p.Lines = append(p.Lines, line)
				}
			}
		}
		if p.Width <= 0 || p.Height <= 0 {
			return Page{}, errors.New("the layout gives no page size")
		}
		return p, nil
	}
}

var space = regexp.MustCompile(`\s+`)

func squash(s string) string { return strings.TrimSpace(space.ReplaceAllString(s, " ")) }

// key gives the text for the search of Band: the accents go and the spaces are squashed. A PDF can give "número" as u and a combining accent, or as the accent before the u, and the text of a flag has the letter with its accent.
func key(s string) string {
	s = strings.Map(func(r rune) rune {
		switch {
		case unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Sk, r):
			return -1
		case unicode.IsControl(r):
			return ' '
		}
		return r
	}, norm.NFD.String(s))
	return squash(s)
}

// Band gives the part of the page from the top of the line where the text after starts to the top of the line where the text before starts. The search for before starts after the text after. An empty after starts the band at the top of the page, and an empty before ends it at the bottom. A text can run over more than one line.
func (p Page) Band(after, before string) (Box, error) {
	var all strings.Builder
	starts := make([]int, len(p.Lines))
	for i, l := range p.Lines {
		if i > 0 {
			all.WriteByte(' ')
		}
		starts[i] = all.Len()
		all.WriteString(key(l.Text()))
	}
	text := all.String()
	lineAt := func(off int) int {
		i := 0
		for j, s := range starts {
			if s <= off {
				i = j
			}
		}
		return i
	}
	band := p.Bounds()
	from := 0
	if after = key(after); after != "" {
		k := strings.Index(text, after)
		if k < 0 {
			return Box{}, fmt.Errorf("the page has no text %q", after)
		}
		band.Y0 = p.Lines[lineAt(k)].Y0
		from = k + len(after)
	}
	if before = key(before); before != "" {
		k := strings.Index(text[from:], before)
		if k < 0 {
			return Box{}, fmt.Errorf("the page has no text %q after the start of the band", before)
		}
		band.Y1 = p.Lines[lineAt(from+k)].Y0
	}
	if band.Empty() {
		return Box{}, fmt.Errorf("the band from %q to %q is empty", after, before)
	}
	return band, nil
}

// TextWords gives the words of the running text that overlap box: the words of the lines that are at least a third of the page wide. A label of a figure is a short line, so it is not in the list.
func (p Page) TextWords(box Box) []Box {
	var out []Box
	for _, l := range p.Lines {
		if l.W() < p.Width*textLine {
			continue
		}
		for _, w := range l.Words {
			if w.Overlaps(box) {
				out = append(out, w.Box)
			}
		}
	}
	return out
}

// textLine is the width of a line of running text, as a part of the width of the page. A shorter line can be a label of a figure.
const textLine = 0.3

// Options tune Find. The zero value takes the defaults.
type Options struct {
	// Gap joins the parts of a drawing that are closer than this, in points. It is 12 by default.
	Gap float64
	// Thin is the size in points under which a part is a rule or a fraction bar and not a drawing. It is 4 by default.
	Thin float64
	// Margin is how far from the drawing a label can be, in points. It is 10 by default.
	Margin float64
	// Pad is the white space around the figure, in points. It is 2 by default.
	Pad float64
}

func (o Options) defaults() Options {
	if o.Gap <= 0 {
		o.Gap = 12
	}
	if o.Thin <= 0 {
		o.Thin = 4
	}
	if o.Margin <= 0 {
		o.Margin = 10
	}
	if o.Pad <= 0 {
		o.Pad = 2
	}
	return o
}

// ink tells if a pixel is dark enough to be part of a drawing. A light grey fill counts, and a white page does not.
func ink(c color.Color) bool {
	g := color.GrayModel.Convert(c).(color.Gray)
	return g.Y < 235
}

// Find gives the box of the figure in band. img is the image of the whole page. The words of the text layer are masked out of the image, so the ink that is left is the drawing. The parts of the drawing closer than Gap are one part, and a part thinner than Thin, such as a rule or a fraction bar, is left out. The figure is all the parts that are left, with the short labels next to them and the words inside them.
func Find(p Page, img image.Image, band Box, o Options) (Box, error) {
	o = o.defaults()
	r := img.Bounds()
	sx := float64(r.Dx()) / p.Width
	sy := float64(r.Dy()) / p.Height
	if sx <= 0 || sy <= 0 {
		return Box{}, errors.New("the page image is empty")
	}
	band = band.Clip(p.Bounds())
	// The cells of the grid are about 2 points wide.
	cell := 2.0
	cw, ch := int(math.Ceil(p.Width/cell)), int(math.Ceil(p.Height/cell))
	type ext struct {
		box Box
		set bool
	}
	cells := make([]ext, cw*ch)
	var masks []Box
	for _, l := range p.Lines {
		for _, w := range l.Words {
			masks = append(masks, w.Grow(1))
		}
	}
	px0, py0 := int(band.X0*sx), int(band.Y0*sy)
	px1, py1 := min(int(math.Ceil(band.X1*sx)), r.Dx()), min(int(math.Ceil(band.Y1*sy)), r.Dy())
	for y := py0; y < py1; y++ {
		fy := (float64(y) + 0.5) / sy
		var row []Box
		for _, m := range masks {
			if m.Y0 <= fy && fy <= m.Y1 {
				row = append(row, m)
			}
		}
		for x := px0; x < px1; x++ {
			if !ink(img.At(r.Min.X+x, r.Min.Y+y)) {
				continue
			}
			fx := (float64(x) + 0.5) / sx
			masked := false
			for _, m := range row {
				if m.X0 <= fx && fx <= m.X1 {
					masked = true
					break
				}
			}
			if masked {
				continue
			}
			pix := Box{float64(x) / sx, float64(y) / sy, float64(x+1) / sx, float64(y+1) / sy}
			i := min(int(fy/cell), ch-1)*cw + min(int(fx/cell), cw-1)
			cells[i].box = cells[i].box.Union(pix)
			cells[i].set = true
		}
	}
	// The parts are the groups of cells with ink that are at most Gap apart.
	reach := int(math.Ceil(o.Gap / cell))
	seen := make([]bool, len(cells))
	var parts []Box
	for start := range cells {
		if !cells[start].set || seen[start] {
			continue
		}
		part := Box{}
		queue := []int{start}
		seen[start] = true
		for len(queue) > 0 {
			i := queue[0]
			queue = queue[1:]
			part = part.Union(cells[i].box)
			cx, cy := i%cw, i/cw
			for dy := -reach; dy <= reach; dy++ {
				for dx := -reach; dx <= reach; dx++ {
					nx, ny := cx+dx, cy+dy
					if nx < 0 || ny < 0 || nx >= cw || ny >= ch {
						continue
					}
					j := ny*cw + nx
					if cells[j].set && !seen[j] {
						seen[j] = true
						queue = append(queue, j)
					}
				}
			}
		}
		parts = append(parts, part)
	}
	fig := Box{}
	for _, part := range parts {
		if part.W() < o.Thin || part.H() < o.Thin {
			continue
		}
		fig = fig.Union(part)
	}
	if fig.Empty() {
		return Box{}, fmt.Errorf("no drawing in the band %s", band)
	}
	// The labels: each word inside the figure, and each word near it on a short line. A line of the text of the problem is long, so it stays out.
	short := p.Width * textLine
	for changed := true; changed; {
		changed = false
		for _, l := range p.Lines {
			for _, w := range l.Words {
				if w.Within(fig) || !w.Within(band) {
					continue
				}
				if w.Within(fig.Grow(1)) || (l.W() < short && w.Overlaps(fig.Grow(o.Margin))) {
					// A word with no area, which some PDFs have, does not change the box, so it must not start another round.
					if u := fig.Union(w.Box); u != fig {
						fig = u
						changed = true
					}
				}
			}
		}
	}
	return fig.Grow(o.Pad).Clip(band), nil
}

// svgDefaults are attributes that pdftocairo writes with the value that SVG takes when the attribute is not there.
var svgDefaults = strings.NewReplacer(
	` stroke-opacity="1"`, "",
	` fill-opacity="1"`, "",
	` stroke-linecap="butt"`, "",
	` stroke-linejoin="miter"`, "",
	` fill-rule="nonzero"`, "",
	`rgb(0%, 0%, 0%)`, "#000",
	`rgb(100%, 100%, 100%)`, "#fff",
)

// longNumber is a number with more than three decimals. A thousandth of a point is far less than a pixel.
var longNumber = regexp.MustCompile(`([0-9]\.[0-9]{3})[0-9]+`)

// SmallSVG makes the SVG of pdftocairo smaller: it drops the attributes with the default value and cuts the numbers to three decimals. The drawing stays the same.
func SmallSVG(svg []byte) []byte {
	return longNumber.ReplaceAll([]byte(svgDefaults.Replace(string(svg))), []byte("$1"))
}

// HasImage tells if an SVG has a raster image in it. A figure that is a scan or a photo is better as a PNG.
func HasImage(svg []byte) bool { return bytes.Contains(svg, []byte("<image")) }

// SmallPNG encodes the image again with the best compression, in grey when it has no colour, so the file is small.
func SmallPNG(data []byte) ([]byte, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	r := img.Bounds()
	out := img
	grey := true
	for y := r.Min.Y; y < r.Max.Y && grey; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			if c.R != c.G || c.G != c.B || c.A != 255 {
				grey = false
				break
			}
		}
	}
	if grey {
		g := image.NewGray(r)
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				g.Set(x, y, img.At(x, y))
			}
		}
		out = g
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, out); err != nil {
		return nil, err
	}
	if buf.Len() >= len(data) {
		return data, nil
	}
	return buf.Bytes(), nil
}
