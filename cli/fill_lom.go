package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/pdfsrc"
)

func init() {
	ruLanes["ru-lomonosov"] = ruLane{source: "turlom", node: prLomonosov, read: lomRead, holds: prSameYear}
}

// prLomonosov is the node of the Lomonosov Tournament in the source catalogue of problems.ru.
const prLomonosov = 111442

// lomPages are the pages of the problems of the Lomonosov Tournament on its old site, by year. A year of lomPapers has a PDF paper.
var lomPages = map[int]string{
	1995: "index.htm",
	1996: "index.htm",
	1997: "index.htm",
	1998: "tl98koi8.htm",
	1999: "tl1999.htm",
	2000: "turlom00.htm",
	2001: "zadanija.htm",
	2002: "zadanija.htm",
	2003: "zadanija/zadanija.htm",
	2007: "zadanija/zadanija.htm",
	2009: "zadanija/html/",
}

var lomPapers = map[int]string{
	2008: "https://olympiads.mccme.ru/turlom/2008/zadanija/tl2008mat.pdf",
	2010: "https://olympiads.mccme.ru/turlom/2010/zadanija/tl2010mat.pdf",
	2011: "https://olympiads.mccme.ru/turlom/2011/zadanija/tl2011mat.pdf",
	2012: "https://olympiads.mccme.ru/turlom/2012/zadanija/tl2012mat.pdf",
	2013: "https://olympiads.mccme.ru/turlom/2013/zadanija/tl2013mat.pdf",
	2014: "https://olympiads.mccme.ru/turlom/2014/zadanija/tl2014mat.pdf",
	2025: "https://xn--k1acfgjg.xn--l1afu.xn--p1ai/wp-content/uploads/2025/10/tl48_math.pdf",
}

// lomRead reads the problems of mathematics of a year of the Lomonosov Tournament from the site of the tournament. The years 2015 to 2022 have no problems of mathematics there.
func lomRead(ctx context.Context, fc *fetch.Client, source, edition string) ([]official.RuProblem, error) {
	y, err := strconv.Atoi(edition)
	if err != nil {
		return nil, err
	}
	if u, ok := lomPapers[y]; ok {
		pages, err := pdfPages(ctx, fc, source, u)
		if err != nil {
			return nil, err
		}
		return official.TurlomPDF(pages, edition, u), nil
	}
	name, ok := lomPages[y]
	if !ok {
		return nil, nil
	}
	u := fmt.Sprintf("https://olympiads.mccme.ru/turlom/%d/%s", y, name)
	raw, err := get(ctx, fc, source, u)
	if err != nil {
		return nil, err
	}
	page, err := official.RuDecode([]byte(raw))
	if err != nil {
		return nil, err
	}
	return official.TurlomHTML(page, edition, u), nil
}

// pdfPages gets a PDF into the store and gives the text of each page, as pdftotext -raw gives it: in the order of the text, which keeps the columns of a page apart.
func pdfPages(ctx context.Context, fc *fetch.Client, source, url string) ([]string, error) {
	data, err := get(ctx, fc, source, url)
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "om-fill-*.pdf")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	_, werr := f.WriteString(data)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return nil, werr
	}
	info, err := pdfsrc.ReadInfo(ctx, pdfsrc.Exec, path)
	if err != nil {
		return nil, err
	}
	var pages []string
	for i := 1; i <= info.Pages; i++ {
		p := strconv.Itoa(i)
		out, err := pdfsrc.Exec.Run(ctx, "pdftotext", "-raw", "-f", p, "-l", p, path, "-")
		if err != nil {
			return nil, err
		}
		pages = append(pages, strings.TrimRight(string(out), "\f\n "))
	}
	return pages, nil
}
