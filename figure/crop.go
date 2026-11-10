package figure

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"math"
	"regexp"
	"strconv"

	"github.com/tamnd/olymmath-agent/pdfsrc"
)

// FindDPI is the resolution of the page image that Locate reads. The boxes of the words come from the text layer, so the image only has to show where the ink is.
const FindDPI = 100

// Locate finds the box of the figure on a page of the PDF at path, in the band from the text after to the text before (Page.Band).
func Locate(ctx context.Context, r pdfsrc.Runner, path string, page int, after, before string, o Options) (Box, error) {
	layout, err := pdfsrc.Layout(ctx, r, path, page)
	if err != nil {
		return Box{}, err
	}
	p, err := ParseLayout(layout)
	if err != nil {
		return Box{}, err
	}
	band, err := p.Band(after, before)
	if err != nil {
		return Box{}, err
	}
	data, err := pdfsrc.Image(ctx, r, path, page, FindDPI, pdfsrc.Crop{})
	if err != nil {
		return Box{}, err
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return Box{}, fmt.Errorf("the page image: %w", err)
	}
	return Find(p, img, band, o)
}

// Formats of a figure file. Auto is SVG for a drawing and PNG for a figure with a raster image in it.
const (
	Auto = "auto"
	SVG  = "svg"
	PNG  = "png"
)

// Figure is a cropped figure: the box on the page, the format and the bytes of the file.
type Figure struct {
	Box    Box
	Format string
	Data   []byte
}

// Crop crops the box of a page of the PDF at path. With format Auto it gives an SVG when the box has only vector drawing, and a PNG at dpi when it has a raster image. The box is widened to whole points. The words in erase, boxes on the page, are taken out of the figure, so a line of the text of the problem next to the figure does not show in it.
func Crop(ctx context.Context, r pdfsrc.Runner, path string, page int, box Box, format string, dpi int, erase []Box) (Figure, error) {
	box = box.Out()
	if box.Empty() {
		return Figure{}, fmt.Errorf("the box %s is empty", box)
	}
	f := Figure{Box: box, Format: format}
	if format == Auto || format == SVG {
		svg, err := pdfsrc.SVG(ctx, r, path, page, pdfsrc.Crop{X: int(box.X0), Y: int(box.Y0), W: int(box.W()), H: int(box.H())})
		if err != nil {
			return f, err
		}
		if format == SVG || !HasImage(svg) {
			f.Format = SVG
			f.Data = SmallSVG(EraseSVG(svg, box, erase))
			return f, nil
		}
	}
	if format != Auto && format != PNG {
		return f, fmt.Errorf("the format %q is not auto, svg or png", format)
	}
	s := float64(dpi) / 72
	c := pdfsrc.Crop{X: int(math.Floor(box.X0 * s)), Y: int(math.Floor(box.Y0 * s)), W: int(math.Ceil(box.W() * s)), H: int(math.Ceil(box.H() * s))}
	data, err := pdfsrc.Image(ctx, r, path, page, dpi, c)
	if err != nil {
		return f, err
	}
	if len(erase) > 0 {
		if data, err = erasePNG(data, box, s, erase); err != nil {
			return f, err
		}
	}
	f.Format = PNG
	f.Data, err = SmallPNG(data)
	return f, err
}

// glyphUse is a glyph that pdftocairo puts on the page, at the start of its baseline.
var glyphUse = regexp.MustCompile(`<use xlink:href="#glyph[^"]*" x="([-0-9.]+)" y="([-0-9.]+)"/>\n?`)

// EraseSVG takes out of an SVG cropped at box each glyph that starts in one of the words in erase. The words are boxes on the page.
func EraseSVG(svg []byte, box Box, erase []Box) []byte {
	if len(erase) == 0 {
		return svg
	}
	return glyphUse.ReplaceAllFunc(svg, func(m []byte) []byte {
		sm := glyphUse.FindSubmatch(m)
		x, errx := strconv.ParseFloat(string(sm[1]), 64)
		y, erry := strconv.ParseFloat(string(sm[2]), 64)
		if errx != nil || erry != nil {
			return m
		}
		at := Box{box.X0 + x, box.Y0 + y, box.X0 + x, box.Y0 + y}
		for _, w := range erase {
			if at.Within(w.Grow(1)) {
				return nil
			}
		}
		return m
	})
}

// erasePNG paints white the words in erase on a PNG cropped at box with s pixels to a point.
func erasePNG(data []byte, box Box, s float64, erase []Box) ([]byte, error) {
	src, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	img := image.NewRGBA(src.Bounds())
	draw.Draw(img, img.Bounds(), src, src.Bounds().Min, draw.Src)
	for _, w := range erase {
		w = w.Grow(1)
		r := image.Rect(int(math.Floor((w.X0-box.X0)*s)), int(math.Floor((w.Y0-box.Y0)*s)), int(math.Ceil((w.X1-box.X0)*s)), int(math.Ceil((w.Y1-box.Y0)*s)))
		draw.Draw(img, r.Add(img.Bounds().Min).Intersect(img.Bounds()), image.White, image.Point{}, draw.Src)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
