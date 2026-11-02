package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestReadPages(t *testing.T) {
	name := filepath.Join(t.TempDir(), "pages.tsv")
	if err := os.WriteFile(name, []byte("# edition and pages\nuk-bmo1/1975\t1 1 2\n\nuk-bmo2/1981\t1 1 1 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := readPages(name)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got["uk-bmo1/1975"], []int{1, 1, 2}) || len(got["uk-bmo2/1981"]) != 4 || len(got) != 2 {
		t.Errorf("got %v", got)
	}
	if err := os.WriteFile(name, []byte("uk-bmo1/1975 1 1 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPages(name); err == nil {
		t.Errorf("a line with no tab gives no error")
	}
	if err := os.WriteFile(name, []byte("uk-bmo1/1975\t1 x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readPages(name); err == nil {
		t.Errorf("a page that is not a number gives no error")
	}
}
