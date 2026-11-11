package ocr

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"slices"

	"github.com/tamnd/olymmath-agent/figure"
	"github.com/tamnd/olymmath-agent/pdfsrc"
)

// NativeLane is the name of the native lane in -lanes and in an extraction (spec 05.7). It reads the text layer of a born-digital PDF and asks no model.
const NativeLane = "native"

// FlagEncoding is the flag of a text layer whose font has no map to Unicode, so that some letters come out as private or control characters. Such a paper needs the vision lane.
const FlagEncoding = "encoding"

// FlagLayout is the flag of a page whose text layer has math that plain text cannot hold: small words above or below the line, as exponents, indices and the limits of a sum. The text layer gives A⁻¹ as A−1, so such a page needs the vision lane.
const FlagLayout = "layout"

// Native reads the text layer of the pages of a PDF: the native lane, with the TeX lane after it.
type Native struct {
	// PDF runs the poppler tools. It is pdfsrc.Exec when it is nil.
	PDF pdfsrc.Runner
	// Version is the version of pdftotext, as 26.10.0. It goes into the extraction of each page.
	Version string
	// Limit is the largest number of pages when Read is given no pages. It is DefaultLimit when it is 0.
	Limit int
	// Log gets a line for each page, or nothing when it is nil.
	Log io.Writer
}

// Read reads the pages of the PDF at path, or the first Limit pages when pages is empty. Each page is plain text from Layout and math from TeX. A page with no text layer has no text, and a page whose text layer has broken letters has FlagEncoding.
func (n *Native) Read(ctx context.Context, path string, pages []int) ([]Page, error) {
	run := n.PDF
	if run == nil {
		run = pdfsrc.Exec
	}
	if len(pages) == 0 {
		in, err := pdfsrc.ReadInfo(ctx, run, path)
		if err != nil {
			return nil, err
		}
		limit := n.Limit
		if limit <= 0 {
			limit = DefaultLimit
		}
		for i := 1; i <= min(in.Pages, limit); i++ {
			pages = append(pages, i)
		}
	}
	var out []Page
	for _, i := range pages {
		raw, err := pdfsrc.Text(ctx, run, path, i)
		if err != nil {
			return out, fmt.Errorf("page %d: %w", i, err)
		}
		p := Page{N: i, Kind: NativeLane, Lane: "pdftotext", Version: n.Version}
		plain := Layout(raw)
		p.Text = TeX(plain)
		if p.Text != plain {
			p.Model = TeXVersion
		}
		if encodingFlag(raw) {
			p.Flags = append(p.Flags, FlagEncoding)
		}
		if raw != "" {
			b, err := pdfsrc.Layout(ctx, run, path, i)
			if err != nil {
				return out, fmt.Errorf("page %d: %w", i, err)
			}
			if lp, err := figure.ParseLayout(b); err != nil || scripts(lp) {
				p.Flags = append(p.Flags, FlagLayout)
			}
		}
		for _, f := range Check(p.Text) {
			if !slices.Contains(p.Flags, f) {
				p.Flags = append(p.Flags, f)
			}
		}
		if n.Log != nil {
			fmt.Fprintf(n.Log, "page %d: %d characters from the text layer, flags %v\n", i, len(p.Text), p.Flags)
		}
		out = append(out, p)
	}
	return out, nil
}

// scripts tells if a page has small words, as exponents and indices: words whose height is less than 4/5 of the median height of the words of the page.
func scripts(p figure.Page) bool {
	var hs []float64
	for _, l := range p.Lines {
		for _, w := range l.Words {
			hs = append(hs, w.Y1-w.Y0)
		}
	}
	if len(hs) == 0 {
		return false
	}
	slices.Sort(hs)
	median := hs[len(hs)/2]
	return hs[0] < 0.8*median
}

var versionRE = regexp.MustCompile(`version ([0-9][0-9.]*)`)

// PdftotextVersion gives the version of the pdftotext on the PATH, as 26.10.0, or "" when it does not tell.
func PdftotextVersion(ctx context.Context) string {
	// pdftotext -v writes to standard error and exits with 0 or 99, so the output counts and the status does not.
	b, _ := exec.CommandContext(ctx, "pdftotext", "-v").CombinedOutput() // #nosec G204 -- a fixed command.
	if m := versionRE.FindSubmatch(b); m != nil {
		return string(m[1])
	}
	return ""
}
