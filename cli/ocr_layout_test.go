package cli

import (
	"testing"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/ocr"
)

func TestLinkLayout(t *testing.T) {
	pages := []ocr.Page{{N: 2, Flags: []string{ocr.FlagLayout}}, {N: 3}}
	news := []newScan{
		{at: "nz-nzmo/2019/p11", scan: canon.Scan{Page: 2, Text: "Prove that aa + bb + cc ≥ 3.", Extraction: "native", Confidence: 0.9}, why: "text"},
		{at: "nz-nzmo/2019/p12", scan: canon.Scan{Page: 3, Text: "Prove it.", Extraction: "native", Confidence: 0.9}, why: "text"},
	}
	linkLayout(news, pages)
	if s := news[0].scan; s.Text != "" || s.Extraction != "" || s.Confidence != 0 || news[0].why == "text" {
		t.Errorf("page 2: got %+v, %q", s, news[0].why)
	}
	if s := news[1].scan; s.Text != "Prove it." || news[1].why != "text" {
		t.Errorf("page 3: got %+v, %q", s, news[1].why)
	}
}
