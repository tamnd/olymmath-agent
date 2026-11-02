package cli

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tamnd/olymmath-agent/content"
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

func TestPaperURL(t *testing.T) {
	e := &content.Edition{Official: []content.OfficialPaper{
		{URL: "prelim.pdf", Code: "ja", Paper: "prelim", Source: "imojp"},
		{URL: "final.html", Code: "ja", Source: "imojp"},
		{URL: "other.pdf", Source: "other"},
	}}
	if url, lang := paperURL(e, "imojp"); url != "final.html" || lang != "ja" {
		t.Errorf("got %q %q, want the final paper in ja", url, lang)
	}
	if url, lang := paperURL(e, "other"); url != "other.pdf" || lang != "en" {
		t.Errorf("got %q %q, want other.pdf in en", url, lang)
	}
	if url, _ := paperURL(e, "none"); url != "" {
		t.Errorf("got %q, want none", url)
	}
}
