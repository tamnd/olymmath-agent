package katex

import (
	"bytes"
	"compress/flate"
	"io"
	"strings"
)

// dictTeX is the formula whose HTML starts the dictionary of the compressor.
const dictTeX = `\frac{a+b}{\sqrt{x^2+1}}=\sum_{i=1}^{n}\alpha_i\le\left(\int_0^1 f(t)\,dt\right)`

// squeeze compresses the HTML of a formula for the cache. The HTML of a formula is about 2 KB, and a large site has some 250 000 formulas, so the plain HTML of the cache is about 600 MB. With flate and the dictionary each formula keeps about one byte in seven.
func (r *Renderer) squeeze(html string) ([]byte, error) {
	if html == "" {
		return nil, nil
	}
	var buf bytes.Buffer
	w, _ := r.writers.Get().(*flate.Writer)
	if w == nil {
		var err error
		if w, err = flate.NewWriterDict(&buf, flate.BestSpeed, r.dict); err != nil {
			return nil, err
		}
	} else {
		w.Reset(&buf)
	}
	if _, err := io.WriteString(w, html); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	r.writers.Put(w)
	return bytes.Clone(buf.Bytes()), nil
}

// expand gives the HTML of a result of the cache.
func (r *Renderer) expand(res result) (string, error) {
	if res.size == 0 {
		return "", nil
	}
	src := bytes.NewReader(res.html)
	fr, _ := r.readers.Get().(io.ReadCloser)
	if fr == nil {
		fr = flate.NewReaderDict(src, r.dict)
	} else if err := fr.(flate.Resetter).Reset(src, r.dict); err != nil {
		return "", err
	}
	var b strings.Builder
	b.Grow(res.size)
	// The cache knows the length of the HTML, so the copy stops there.
	if _, err := io.CopyN(&b, fr, int64(res.size)); err != nil {
		return "", err
	}
	r.readers.Put(fr)
	return b.String(), nil
}
