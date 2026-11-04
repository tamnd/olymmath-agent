package pdfsrc

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type fake struct {
	out   map[string]string
	calls [][]string
	write bool // write the image that pdftoppm names
}

func (f *fake) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	f.calls = append(f.calls, append([]string{name}, args...))
	if name == "pdftoppm" && f.write {
		if err := os.WriteFile(args[len(args)-1]+".png", []byte("png"), 0o644); err != nil {
			return nil, err
		}
	}
	o, ok := f.out[name]
	if !ok {
		return nil, errors.New(name + ": no output")
	}
	return []byte(o), nil
}

const info = `Title:          Zadania
Creator:        TeX
Producer:       pdfTeX-1.40.21
Pages:          3
Encrypted:      no
Page size:      841.89 x 595.276 pts (A4)
`

func TestInfo(t *testing.T) {
	f := &fake{out: map[string]string{"pdfinfo": info}}
	in, err := ReadInfo(context.Background(), f, "a.pdf")
	if err != nil {
		t.Fatal(err)
	}
	if in.Pages != 3 || in.Producer != "pdfTeX-1.40.21" || !in.Landscape() {
		t.Errorf("info = %+v", in)
	}
	f.out["pdfinfo"] = "Title: x\n"
	if _, err := ReadInfo(context.Background(), f, "a.pdf"); err == nil {
		t.Error("no page count gave no error")
	}
}

func TestText(t *testing.T) {
	f := &fake{out: map[string]string{"pdftotext": "1. Prove it.\n\f"}}
	got, err := Text(context.Background(), f, "a.pdf", 2)
	if err != nil || got != "1. Prove it." {
		t.Errorf("text = %q, %v", got, err)
	}
	want := []string{"pdftotext", "-layout", "-f", "2", "-l", "2", "a.pdf", "-"}
	if !slices.Equal(f.calls[0], want) {
		t.Errorf("argv = %v", f.calls[0])
	}
}

func TestRender(t *testing.T) {
	dir := t.TempDir()
	f := &fake{out: map[string]string{"pdftoppm": ""}, write: true}
	img, err := Render(context.Background(), f, "a.pdf", 2, 200, dir)
	if err != nil {
		t.Fatal(err)
	}
	if img != filepath.Join(dir, "p002-200.png") {
		t.Errorf("image = %s", img)
	}
	if got := strings.Join(f.calls[0], " "); !strings.Contains(got, "-r 200 -png -f 2 -l 2 -singlefile a.pdf") {
		t.Errorf("argv = %s", got)
	}
	for _, c := range []struct{ page, dpi int }{{0, 200}, {1, 50}, {1, 600}} {
		if _, err := Render(context.Background(), f, "a.pdf", c.page, c.dpi, dir); err == nil {
			t.Errorf("page %d at %d dpi gave no error", c.page, c.dpi)
		}
	}
	f.write = false
	if _, err := Render(context.Background(), f, "a.pdf", 3, 200, dir); err == nil {
		t.Error("a missing image gave no error")
	}
}
