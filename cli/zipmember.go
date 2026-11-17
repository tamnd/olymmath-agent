package cli

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"strings"
)

// zipMaxMember is the largest file that zipMember reads out of a zip file.
const zipMaxMember = 64 << 20

// zipNames gives the names of the files of a zip file, without its directories and the files that macOS adds, as __MACOSX/._a.pdf and .DS_Store.
func zipNames(raw []byte) ([]string, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	var out []string
	for _, f := range zr.File {
		base := f.Name[strings.LastIndex(f.Name, "/")+1:]
		if f.FileInfo().IsDir() || strings.HasPrefix(f.Name, "__MACOSX/") || strings.HasPrefix(base, "._") || base == ".DS_Store" {
			continue
		}
		out = append(out, f.Name)
	}
	return out, nil
}

// zipMember reads the file name out of a zip file.
func zipMember(raw []byte, name string) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw)))
	if err != nil {
		return nil, err
	}
	for _, f := range zr.File {
		if f.Name != name {
			continue
		}
		if f.UncompressedSize64 > zipMaxMember {
			return nil, fmt.Errorf("%s is larger than %d bytes", name, zipMaxMember)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		defer func() { _ = rc.Close() }()
		return io.ReadAll(io.LimitReader(rc, zipMaxMember))
	}
	return nil, fmt.Errorf("the zip file has no %s", name)
}
