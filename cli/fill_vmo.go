package cli

import (
	"context"
	"os"
	"strconv"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	// A year node of problems.ru names a school year, as 2004/2005, and holds the final round in its child node "Заключительный этап".
	ruLanes["ru-allrussian"] = ruLane{source: "mccme-vmo", node: prAllRussian, read: vmoRead, holds: prLastYear, final: true}
}

// prAllRussian is the node of the All-Russian olympiad in the source catalogue of problems.ru.
const prAllRussian = 204607

// prLastYear tells if a year node that runs over a school year, as "2004/2005", ends in the year of the edition.
func prLastYear(name, edition string) bool {
	_, last, ok := archive.PRSpan(name)
	return ok && strconv.Itoa(last) == edition
}

// vmoBase is the archive of the All-Russian olympiad on the site of MCCME.
const vmoBase = "https://olympiads.mccme.ru/vmo/"

// vmoPages are the web pages of the problems of the final round, by year.
var vmoPages = map[int]string{
	2005: "31/zadachi.htm",
}

// vmoPDFs are the PDF sheets of the problems of the final round, by year, in the order they count. The archive has no problems of the final round before 2005, and the records of the years that are not here have their text.
var vmoPDFs = map[int][]string{
	2006: {"32/sheets.pdf"},
	2007: {"33/sheets1.pdf", "33/sheets2.pdf"},
	2008: {"34/1.pdf", "34/2.pdf"},
	2012: {"2012/final/sheets1.PDF", "2012/final/sheets2.PDF"},
	2013: {"2013/final/sheets1.PDF", "2013/final/sheets2.PDF"},
	2014: {"2014/final/sheets1.PDF", "2014/final/sheets2.PDF"},
}

// vmoRead reads the problems of the final round of a year of the All-Russian olympiad from the archive on the site of MCCME.
func vmoRead(ctx context.Context, fc *fetch.Client, source, edition string) ([]official.RuProblem, error) {
	y, err := strconv.Atoi(edition)
	if err != nil {
		return nil, err
	}
	if name, ok := vmoPages[y]; ok {
		raw, err := get(ctx, fc, source, vmoBase+name)
		if err != nil {
			return nil, err
		}
		page, err := official.RuDecode([]byte(raw))
		if err != nil {
			return nil, err
		}
		return official.VmoPage(page, edition, vmoBase+name), nil
	}
	var out []official.RuProblem
	seen := map[string]bool{}
	for _, name := range vmoPDFs[y] {
		ps, err := vmoPDFRead(ctx, fc, source, vmoBase+name, edition)
		if err != nil {
			return nil, err
		}
		// The first sheet that gives a position counts.
		for _, p := range ps {
			if !seen[p.Position] {
				seen[p.Position] = true
				out = append(out, p)
			}
		}
	}
	return out, nil
}

// vmoPDFRead reads the problems of one PDF sheet: the text layer comes from pdftohtml, and the letters that it loses come from the PDF itself.
func vmoPDFRead(ctx context.Context, fc *fetch.Client, source, u, edition string) ([]official.RuProblem, error) {
	raw, err := get(ctx, fc, source, u)
	if err != nil {
		return nil, err
	}
	f, err := os.CreateTemp("", "om-fill-*.pdf")
	if err != nil {
		return nil, err
	}
	path := f.Name()
	defer func() { _ = os.Remove(path) }()
	_, werr := f.WriteString(raw)
	if cerr := f.Close(); werr == nil {
		werr = cerr
	}
	if werr != nil {
		return nil, werr
	}
	xml, err := figureRunner.Run(ctx, official.RuPDFTool, "-xml", "-i", "-stdout", "-q", path)
	if err != nil {
		return nil, err
	}
	return official.RuPDFXML(xml, []byte(raw), edition, u), nil
}
