package cli

import (
	"testing"

	"github.com/tamnd/olymmath-agent/canon"
)

func TestLinkUnlisted(t *testing.T) {
	mk := func() []newScan {
		return []newScan{
			{at: "purple-comet/2005/h11", scan: canon.Scan{Page: 2, Text: "How many minutes?", Extraction: "native", Confidence: 0.9}, why: "text"},
			{at: "purple-comet/2005/h12", scan: canon.Scan{Page: 2, Text: "Find n for which is a prime.", Extraction: "native", Confidence: 0.9}, why: "text"},
			{at: "purple-comet/2005/h13", scan: canon.Scan{Page: 3}, why: "a link only: the checks found layout"},
		}
	}
	news := mk()
	linkUnlisted(news, "h11, h13")
	if s := news[0].scan; s.Text != "How many minutes?" || news[0].why != "text" {
		t.Errorf("h11: got %+v, %q", s, news[0].why)
	}
	if s := news[1].scan; s.Text != "" || s.Extraction != "" || s.Confidence != 0 || news[1].why == "text" {
		t.Errorf("h12: got %+v, %q", s, news[1].why)
	}
	if news[2].why != "a link only: the checks found layout" {
		t.Errorf("h13: got %q", news[2].why)
	}
	news = mk()
	linkUnlisted(news, "")
	if news[1].scan.Text == "" {
		t.Errorf("an empty -text changed h12")
	}
}
