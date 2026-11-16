package cli

import (
	"slices"

	"github.com/tamnd/olymmath-agent/ocr"
)

// linkLayout makes each new problem of a page with ocr.FlagLayout a link only. planScans gives a problem only the flags of its page that hold for the whole page, and the layout flag does not, because one exponent on a page does not spoil the other problems. Some text layers lose every exponent and index, as the NZMO papers give a^a + b^b + c^c as "aa + bb + cc", and then no problem of a flagged page can keep its text.
func linkLayout(news []newScan, pages []ocr.Page) {
	flagged := map[int]bool{}
	for _, p := range pages {
		if slices.Contains(p.Flags, ocr.FlagLayout) {
			flagged[p.N] = true
		}
	}
	for i := range news {
		if news[i].scan.Text == "" || !flagged[news[i].scan.Page] {
			continue
		}
		news[i].scan.Text, news[i].scan.Extraction, news[i].scan.Confidence = "", "", 0
		news[i].why = "a link only: the page has the flag " + ocr.FlagLayout + " (-layout-link)"
	}
}
