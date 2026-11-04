package ocr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/tamnd/olymmath-agent/lane"
	"github.com/tamnd/olymmath-agent/pdfsrc"
)

// DefaultDPI is the ladder of resolutions. A page whose text at the first resolution has a flag is read again at the next one.
var DefaultDPI = []int{200, 300}

// DefaultLimit is the largest number of pages that Read reads when it is given no pages. Each page is a model call, and a booklet of solutions can have fifty pages.
const DefaultLimit = 8

// Reader reads the pages of a PDF with a model.
type Reader struct {
	Asker lane.Asker
	Cache *Cache
	// PDF runs the poppler tools. It is pdfsrc.Exec when it is nil.
	PDF pdfsrc.Runner
	// Work is the directory for the page images. Read deletes each image when it has read it.
	Work string
	// DPI is the ladder of resolutions. It is DefaultDPI when it is empty.
	DPI []int
	// Limit is the largest number of pages when Read is given no pages. It is DefaultLimit when it is 0.
	Limit int
	// Log gets a line for each page, or nothing when it is nil.
	Log io.Writer
}

// Page is the text of one page.
type Page struct {
	N    int
	Text string
	// Flags are the flags of the checks of the whole page.
	Flags   []string
	Lane    string
	Model   string
	Version string
	DPI     int
	// Key is the key of the page image in the cache.
	Key    string
	Cached bool
}

// Extraction gives the extraction of the page as spec 05.7 writes it: vision/<client>@<version>, and the model when the client tells it.
func (p Page) Extraction() string {
	s := "vision/" + p.Lane
	if p.Version != "" {
		s += "@" + p.Version
	}
	if p.Model != "" {
		s += " (" + p.Model + ")"
	}
	return s
}

func (r *Reader) runner() pdfsrc.Runner {
	if r.PDF == nil {
		return pdfsrc.Exec
	}
	return r.PDF
}

func (r *Reader) logf(format string, args ...any) {
	if r.Log != nil {
		fmt.Fprintf(r.Log, format+"\n", args...)
	}
}

// Read reads the pages of the PDF at path, or the first Limit pages when pages is empty. It stops at the first error that tells that no lane can answer, and it gives the pages that it read before it, with the error. So a batch that runs out of allowance stops and does not loop.
func (r *Reader) Read(ctx context.Context, path string, pages []int) ([]Page, error) {
	if len(pages) == 0 {
		in, err := pdfsrc.ReadInfo(ctx, r.runner(), path)
		if err != nil {
			return nil, err
		}
		limit := r.Limit
		if limit <= 0 {
			limit = DefaultLimit
		}
		for n := 1; n <= min(in.Pages, limit); n++ {
			pages = append(pages, n)
		}
	}
	var out []Page
	for _, n := range pages {
		p, err := r.Page(ctx, path, n)
		if err != nil {
			return out, fmt.Errorf("page %d: %w", n, err)
		}
		out = append(out, p)
	}
	return out, nil
}

// Page reads one page. It reads the page again at the next resolution of the ladder when the model gave no text or the text has a flag that a sharper image can clear, and it keeps the text with the fewest such flags.
func (r *Reader) Page(ctx context.Context, path string, n int) (Page, error) {
	ladder := r.DPI
	if len(ladder) == 0 {
		ladder = DefaultDPI
	}
	var best *Page
	var last error
	for _, dpi := range ladder {
		p, err := r.at(ctx, path, n, dpi)
		if errors.Is(err, lane.ErrExhausted) {
			return Page{}, err
		}
		if err != nil {
			last = err
			r.logf("page %d at %d dpi: %v", n, dpi, err)
			continue
		}
		if best == nil || retry(p.Flags) < retry(best.Flags) {
			best = &p
		}
		if retry(best.Flags) == 0 {
			break
		}
	}
	if best == nil {
		return Page{}, last
	}
	return *best, nil
}

// retry counts the flags that a read at a higher resolution can clear. A figure stays a figure at any resolution.
func retry(flags []string) int {
	n := 0
	for _, f := range flags {
		switch f {
		case FlagRefusal, FlagDollars, FlagBraces, FlagUnreadable:
			n++
		}
	}
	return n
}

// at reads one page at one resolution, from the cache when the cache has the image.
func (r *Reader) at(ctx context.Context, path string, n, dpi int) (Page, error) {
	img, err := pdfsrc.Render(ctx, r.runner(), path, n, dpi, r.Work)
	if err != nil {
		return Page{}, err
	}
	defer func() { _ = os.Remove(img) }()
	b, err := os.ReadFile(img) // #nosec G304 -- the image that Render wrote.
	if err != nil {
		return Page{}, err
	}
	key := Key(b)
	p := Page{N: n, DPI: dpi, Key: key}
	e, ok, err := r.Cache.Get(key)
	if err != nil {
		return Page{}, err
	}
	if ok {
		p.Cached = true
	} else {
		a, err := r.Asker.Ask(ctx, lane.Question{Prompt: Prompt, Images: []string{img}})
		if err != nil {
			return Page{}, err
		}
		e = Entry{Text: a.Text, Lane: a.Lane, Model: a.Model, Version: a.Version, Prompt: PromptVersion, At: now()}
		if err := r.Cache.Put(key, e); err != nil {
			return Page{}, err
		}
	}
	p.Lane, p.Model, p.Version = e.Lane, e.Model, e.Version
	text, flags := Clean(e.Text)
	if text == Empty {
		text = ""
	}
	p.Text = Repair(text)
	for _, f := range append(flags, Check(p.Text)...) {
		if !slices.Contains(p.Flags, f) {
			p.Flags = append(p.Flags, f)
		}
	}
	r.logf("page %d at %d dpi: %d characters, cached %v, flags %v", n, dpi, len(p.Text), p.Cached, p.Flags)
	return p, nil
}
