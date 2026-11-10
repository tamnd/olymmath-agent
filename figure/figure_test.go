package figure

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// layout is a page of 200 by 200 points: the line of problem 1, a label A next to the figure, and the line of problem 2.
const layout = `<!DOCTYPE html>
<html xmlns="http://www.w3.org/1999/xhtml">
<head><title></title></head>
<body>
<doc>
  <page width="200.000000" height="200.000000">
    <flow><block xMin="10" yMin="10" xMax="190" yMax="20">
      <line xMin="10" yMin="10" xMax="190" yMax="20">
        <word xMin="10" yMin="10" xMax="18" yMax="20">1.</word>
        <word xMin="20" yMin="10" xMax="60" yMax="20">Draw</word>
        <word xMin="62" yMin="10" xMax="190" yMax="20">this&amp;that</word>
      </line>
    </block></flow>
    <flow><block xMin="102" yMin="92" xMax="108" yMax="100">
      <line xMin="102" yMin="92" xMax="108" yMax="100">
        <word xMin="102" yMin="92" xMax="108" yMax="100">A</word>
      </line>
    </block></flow>
    <flow><block xMin="10" yMin="150" xMax="190" yMax="160">
      <line xMin="10" yMin="150" xMax="190" yMax="160">
        <word xMin="10" yMin="150" xMax="18" yMax="160">2.</word>
        <word xMin="20" yMin="150" xMax="190" yMax="160">Next</word>
      </line>
    </block></flow>
  </page>
</doc>
</body>
</html>`

// pageImage draws the page at 72 dpi: the words as black blocks, a square from 50 to 100 points and a thin rule under it.
func pageImage() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, 200, 200))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	fill := func(x0, y0, x1, y1 int) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				img.SetGray(x, y, color.Gray{})
			}
		}
	}
	fill(10, 11, 190, 19)
	fill(103, 93, 107, 99)
	fill(10, 151, 190, 159)
	fill(50, 50, 100, 51)
	fill(50, 99, 100, 100)
	fill(50, 50, 51, 100)
	fill(99, 50, 100, 100)
	fill(20, 125, 180, 126)
	return img
}

func TestBox(t *testing.T) {
	b, err := ParseBox("1.5, 2,30,40.25")
	if err != nil || b != (Box{1.5, 2, 30, 40.25}) || b.String() != "1.5,2.0,30.0,40.2" {
		t.Errorf("%v %v", b, err)
	}
	if o := b.Out(); o != (Box{1, 2, 30, 41}) {
		t.Errorf("out %v", o)
	}
	for _, bad := range []string{"1,2,3", "a,b,c,d", "5,5,5,9", "-1,0,4,4"} {
		if _, err := ParseBox(bad); err == nil {
			t.Errorf("ParseBox(%q) gives no error", bad)
		}
	}
}

func TestBand(t *testing.T) {
	p, err := ParseLayout([]byte(layout))
	if err != nil {
		t.Fatal(err)
	}
	if p.Width != 200 || len(p.Lines) != 3 || p.Lines[0].Text() != "1. Draw this&that" {
		t.Fatalf("page %+v", p)
	}
	for _, c := range []struct {
		after, before string
		want          Box
	}{
		{"1. Draw", "2. Next", Box{0, 10, 200, 150}},
		{"Draw this&that A", "", Box{0, 10, 200, 200}},
		{"", "A", Box{0, 0, 200, 92}},
	} {
		got, err := p.Band(c.after, c.before)
		if err != nil || got != c.want {
			t.Errorf("Band(%q, %q) = %v %v", c.after, c.before, got, err)
		}
	}
	if _, err := p.Band("3.", ""); err == nil {
		t.Error("no error for a text that is not on the page")
	}
	// The search for before starts after the text after.
	if _, err := p.Band("Next", "1."); err == nil {
		t.Error("no error for a band that ends before it starts")
	}
}

func TestFind(t *testing.T) {
	p, err := ParseLayout([]byte(layout))
	if err != nil {
		t.Fatal(err)
	}
	band, _ := p.Band("1.", "2.")
	// The square and the label A are the figure. The words are masked, and the thin rule is not a drawing.
	got, err := Find(p, pageImage(), band, Options{})
	if err != nil || got != (Box{48, 48, 110, 102}) {
		t.Errorf("Find = %v %v", got, err)
	}
	// The band over the text only has no drawing.
	if _, err := Find(p, pageImage(), Box{0, 0, 200, 30}, Options{}); err == nil {
		t.Error("a drawing in the band of the words")
	}
}

type fake struct {
	out   map[string][]byte
	calls [][]string
}

func (f *fake) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if o, ok := f.out[name]; ok {
		return o, nil
	}
	return nil, errors.New(name + ": no output")
}

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestLocateAndCrop(t *testing.T) {
	ctx := context.Background()
	svg := `<svg xmlns="http://www.w3.org/2000/svg" width="63pt" height="55pt"><path d="M0 0"/></svg>`
	f := &fake{out: map[string][]byte{"pdftotext": []byte(layout), "pdftoppm": encode(t, pageImage()), "pdftocairo": []byte(svg)}}
	box, err := Locate(ctx, f, "a.pdf", 1, "1.", "2.", Options{})
	if err != nil || box != (Box{48, 48, 110, 102}) {
		t.Fatalf("Locate = %v %v", box, err)
	}
	fig, err := Crop(ctx, f, "a.pdf", 1, Box{47.5, 48, 110, 102.2}, Auto, 144, nil)
	if err != nil || fig.Format != SVG || string(fig.Data) != svg || fig.Box != (Box{47, 48, 110, 103}) {
		t.Fatalf("Crop = %+v %v", fig, err)
	}
	if got := strings.Join(f.calls[len(f.calls)-1], " "); got != "pdftocairo -svg -f 1 -l 1 -x 47 -y 48 -W 63 -H 55 -paperw 63 -paperh 55 a.pdf -" {
		t.Errorf("pdftocairo call: %s", got)
	}
	// A figure with a raster image in it is a PNG.
	f.out["pdftocairo"] = []byte(`<svg><image href="data:image/png;base64,AAAA"/></svg>`)
	fig, err = Crop(ctx, f, "a.pdf", 1, Box{48, 48, 110, 102}, Auto, 144, nil)
	if err != nil || fig.Format != PNG || !bytes.HasPrefix(fig.Data, []byte("\x89PNG")) {
		t.Fatalf("Crop of a raster = %+v %v", fig.Format, err)
	}
	if got := strings.Join(f.calls[len(f.calls)-1], " "); got != "pdftoppm -r 144 -png -f 1 -l 1 -singlefile -x 96 -y 96 -W 124 -H 108 a.pdf" {
		t.Errorf("pdftoppm call: %s", got)
	}
	if _, err := Crop(ctx, f, "a.pdf", 1, Box{}, Auto, 144, nil); err == nil {
		t.Error("no error for an empty box")
	}
}

func TestSmallPNG(t *testing.T) {
	// An image in colour with only grey pixels is grey after SmallPNG.
	img := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for i := range img.Pix {
		img.Pix[i] = 255
	}
	data := encode(t, img)
	small, err := SmallPNG(data)
	if err != nil {
		t.Fatal(err)
	}
	out, err := png.Decode(bytes.NewReader(small))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := out.(*image.Gray); !ok || len(small) > len(data) {
		t.Errorf("SmallPNG gives %T of %d bytes from %d", out, len(small), len(data))
	}
	if _, err := SmallPNG([]byte("not a png")); err == nil {
		t.Error("no error for bytes that are not a PNG")
	}
}

func TestBandAccents(t *testing.T) {
	// The PDF gives the accent of "número" as a mark after the u and has a control character in a word, and the flag has the letter with its accent.
	page := strings.Replace(layout, "<word xMin=\"20\" yMin=\"10\" xMax=\"60\" yMax=\"20\">Draw</word>", "<word xMin=\"20\" yMin=\"10\" xMax=\"60\" yMax=\"20\">número\b</word>", 1)
	p, err := ParseLayout([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	if got, err := p.Band("1. número", "2. Next"); err != nil || got != (Box{0, 10, 200, 150}) {
		t.Errorf("Band = %v %v", got, err)
	}
	if k := key("  Vé¨  ta\tb "); k != "Ve ta b" {
		t.Errorf("key = %q", k)
	}
}

func TestTextWords(t *testing.T) {
	p, err := ParseLayout([]byte(layout))
	if err != nil {
		t.Fatal(err)
	}
	// The box covers the end of the line of problem 1 and the label A. The label is on a short line, so it stays.
	got := p.TextWords(Box{50, 0, 200, 100})
	if len(got) != 2 || got[0] != (Box{20, 10, 60, 20}) || got[1] != (Box{62, 10, 190, 20}) {
		t.Errorf("TextWords = %v", got)
	}
	if got := p.TextWords(Box{0, 170, 200, 200}); len(got) != 0 {
		t.Errorf("TextWords under the text = %v", got)
	}
}

func TestEraseSVG(t *testing.T) {
	svg := []byte(`<g fill="#000">
<use xlink:href="#glyph-0-1" x="5" y="8"/>
<use xlink:href="#glyph-0-2" x="40" y="45"/>
</g>`)
	// The figure is cropped at 10,10, so the first glyph starts at 15,18 on the page, inside the word, and the second at 50,55, outside it.
	got := string(EraseSVG(svg, Box{10, 10, 100, 100}, []Box{{12, 12, 30, 20}}))
	if strings.Contains(got, "glyph-0-1") || !strings.Contains(got, "glyph-0-2") {
		t.Errorf("EraseSVG = %s", got)
	}
	if got := EraseSVG(svg, Box{10, 10, 100, 100}, nil); !bytes.Equal(got, svg) {
		t.Errorf("EraseSVG with no words = %s", got)
	}
}

func TestFindWordWithNoArea(t *testing.T) {
	// Some PDFs have a word with no area. Next to the figure, it must not keep the search for labels going for ever.
	page := strings.Replace(layout, "</page>", `<flow><block xMin="112" yMin="60" xMax="112" yMax="60">
      <line xMin="112" yMin="60" xMax="112" yMax="60">
        <word xMin="112" yMin="60" xMax="112" yMax="60">x</word>
      </line>
    </block></flow>
  </page>`, 1)
	p, err := ParseLayout([]byte(page))
	if err != nil {
		t.Fatal(err)
	}
	band, _ := p.Band("1.", "2.")
	if got, err := Find(p, pageImage(), band, Options{}); err != nil || got != (Box{48, 48, 110, 102}) {
		t.Errorf("Find = %v %v", got, err)
	}
}
