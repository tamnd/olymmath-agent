// Package pdfsrc reads PDF files with the poppler tools. It gives the page count and the page size, the text layer of a page, and a PNG image of a page for a vision model. All calls go through Runner, so a test can give the output of the tools and run no command.
package pdfsrc

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Runner runs one command and gives its standard output.
type Runner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...) // #nosec G204 -- the callers give the fixed names of the poppler tools, and the arguments are flags, numbers and the path of a paper.
	cmd.Stdin = nil
	out, err := cmd.Output()
	if err != nil {
		// The poppler tools write the cause on stderr. The exit status alone does not tell it.
		var ee *exec.ExitError
		if errors.As(err, &ee) && len(ee.Stderr) > 0 {
			return out, fmt.Errorf("%s: %w: %s", name, err, strings.TrimSpace(string(ee.Stderr)))
		}
		return out, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// Exec is the Runner that runs the real commands.
var Exec Runner = execRunner{}

// Info is what pdfinfo tells about a file.
type Info struct {
	Pages    int
	Width    float64 // the width of the first page in points
	Height   float64 // the height of the first page in points
	Producer string
	Creator  string
}

// Landscape is true when the first page is wider than it is high.
func (in Info) Landscape() bool { return in.Width > in.Height }

// ParseInfo reads the output of pdfinfo.
func ParseInfo(b []byte) Info {
	var in Info
	s := bufio.NewScanner(strings.NewReader(string(b)))
	s.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for s.Scan() {
		k, v, ok := strings.Cut(s.Text(), ":")
		if !ok {
			continue
		}
		v = strings.TrimSpace(v)
		switch strings.TrimSpace(k) {
		case "Pages":
			in.Pages, _ = strconv.Atoi(v)
		case "Producer":
			in.Producer = v
		case "Creator":
			in.Creator = v
		case "Page size":
			// The form is "595.276 x 841.89 pts (A4)".
			f := strings.Fields(v)
			if len(f) >= 3 && f[1] == "x" {
				in.Width, _ = strconv.ParseFloat(f[0], 64)
				in.Height, _ = strconv.ParseFloat(f[2], 64)
			}
		}
	}
	return in
}

// ReadInfo runs pdfinfo on the file at path.
func ReadInfo(ctx context.Context, r Runner, path string) (Info, error) {
	out, err := r.Run(ctx, "pdfinfo", path)
	if err != nil {
		return Info{}, err
	}
	in := ParseInfo(out)
	if in.Pages <= 0 {
		return in, fmt.Errorf("pdfinfo %s: no page count", path)
	}
	return in, nil
}

// Text gives the text layer of one page, in the order of the layout.
func Text(ctx context.Context, r Runner, path string, page int) (string, error) {
	p := strconv.Itoa(page)
	out, err := r.Run(ctx, "pdftotext", "-layout", "-f", p, "-l", p, path, "-")
	if err != nil {
		return "", err
	}
	return strings.TrimRight(string(out), "\f\n "), nil
}

// Words gives the words of all pages of the file at path with their boxes, in the order of the content of each page, as the output of pdftotext -raw -bbox.
func Words(ctx context.Context, r Runner, path string) ([]byte, error) {
	return r.Run(ctx, "pdftotext", "-raw", "-bbox", path, "-")
}

// MinDPI and MaxDPI are the limits of Render. Below 100 dpi a model cannot read the subscripts. Above 400 dpi the image is too large to send.
const (
	MinDPI = 100
	MaxDPI = 400
)

// Render writes one page of the file at path as a PNG image into dir and gives the path of the image. The name of the image tells the page and the resolution, so two resolutions of a page can be in the same dir.
func Render(ctx context.Context, r Runner, path string, page, dpi int, dir string) (string, error) {
	if page < 1 {
		return "", fmt.Errorf("render: page %d is not a page", page)
	}
	if dpi < MinDPI || dpi > MaxDPI {
		return "", fmt.Errorf("render: %d dpi is not in %d..%d", dpi, MinDPI, MaxDPI)
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	base := filepath.Join(dir, fmt.Sprintf("p%03d-%d", page, dpi))
	p := strconv.Itoa(page)
	if _, err := r.Run(ctx, "pdftoppm", "-r", strconv.Itoa(dpi), "-png", "-f", p, "-l", p, "-singlefile", path, base); err != nil {
		return "", err
	}
	img := base + ".png"
	if fi, err := os.Stat(img); err != nil || fi.Size() == 0 {
		return "", fmt.Errorf("render: pdftoppm wrote no image for page %d", page)
	}
	return img, nil
}

// Layout gives the text layer of one page with the box of each block, line and word, as the XHTML of pdftotext -bbox-layout. The boxes are in points from the top left corner of the page.
func Layout(ctx context.Context, r Runner, path string, page int) ([]byte, error) {
	if page < 1 {
		return nil, fmt.Errorf("layout: page %d is not a page", page)
	}
	p := strconv.Itoa(page)
	return r.Run(ctx, "pdftotext", "-bbox-layout", "-f", p, "-l", p, path, "-")
}

// Crop is a part of a page in pixels, from the top left corner. A Crop with no width is the whole page.
type Crop struct {
	X, Y, W, H int
}

// Image gives one page, or the part of it in crop, as a PNG image at dpi. It writes no file.
func Image(ctx context.Context, r Runner, path string, page, dpi int, crop Crop) ([]byte, error) {
	if page < 1 {
		return nil, fmt.Errorf("image: page %d is not a page", page)
	}
	if dpi < 36 || dpi > MaxDPI {
		return nil, fmt.Errorf("image: %d dpi is not in 36..%d", dpi, MaxDPI)
	}
	p := strconv.Itoa(page)
	args := []string{"-r", strconv.Itoa(dpi), "-png", "-f", p, "-l", p, "-singlefile"}
	if crop.W > 0 && crop.H > 0 {
		args = append(args, "-x", strconv.Itoa(crop.X), "-y", strconv.Itoa(crop.Y), "-W", strconv.Itoa(crop.W), "-H", strconv.Itoa(crop.H))
	}
	out, err := r.Run(ctx, "pdftoppm", append(args, path)...)
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("image: pdftoppm gave no image for page %d", page)
	}
	return out, nil
}

// SVG gives the part of one page in crop as SVG, with crop in points. The paper of the SVG has the size of the crop, so the SVG shows only the crop: pdftocairo puts the crop in the middle of the paper, and leaves out what is outside it.
func SVG(ctx context.Context, r Runner, path string, page int, crop Crop) ([]byte, error) {
	if page < 1 {
		return nil, fmt.Errorf("svg: page %d is not a page", page)
	}
	if crop.W <= 0 || crop.H <= 0 {
		return nil, fmt.Errorf("svg: the crop %dx%d is empty", crop.W, crop.H)
	}
	p := strconv.Itoa(page)
	out, err := r.Run(ctx, "pdftocairo", "-svg", "-f", p, "-l", p, "-x", strconv.Itoa(crop.X), "-y", strconv.Itoa(crop.Y), "-W", strconv.Itoa(crop.W), "-H", strconv.Itoa(crop.H), "-paperw", strconv.Itoa(crop.W), "-paperh", strconv.Itoa(crop.H), path, "-")
	if err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("svg: pdftocairo gave no SVG for page %d", page)
	}
	return out, nil
}
