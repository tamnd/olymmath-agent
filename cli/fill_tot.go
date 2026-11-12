package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

// prTot is the node of the Tournament of Towns in the source catalogue of problems.ru.
const prTot = 187098

func init() {
	ruLanes["tot"] = ruLane{source: "turgor", node: prTot, read: totRead, holds: totHolds}
}

var (
	// totEditionRE is an edition of the Tournament, as 2008-fall.
	totEditionRE = regexp.MustCompile(`^(\d{4})-(fall|spring)$`)
	// totHref is a link of a page.
	totHref = regexp.MustCompile(`(?i)href\s*=\s*["']?([^"'\s>]+)`)
	// totTeXFile is the TeX file of the papers of a tour, as os30os.tex: os for the autumn tour, vs for the spring tour.
	totTeXFile = regexp.MustCompile(`^(os|vs)(\d+)(?:os|sl|baz)\.tex$`)
	// totPDFFile is the PDF of one level of a tour, as os-36-baz-avt.pdf, os32os-avt.pdf or os31sl.pdf: baz is the base level and sl or os the hard level.
	totPDFFile = regexp.MustCompile(`^(os|vs)-?(\d+)-?(baz|sl|os)(?:-avt)?\.pdf$`)
)

// totTour gives the number of the tournament of an edition and the start of the names of its files: the autumn tour of 2008 is in tournament 30 (2008/2009), and the spring tour of 2009 too.
func totTour(edition string) (n int, prefix string, ok bool) {
	m := totEditionRE.FindStringSubmatch(edition)
	if m == nil {
		return 0, "", false
	}
	y, _ := strconv.Atoi(m[1])
	if m[2] == "fall" {
		return y - 1978, "os", true
	}
	return y - 1979, "vs", true
}

// totHolds tells if a year node of problems.ru, as "30 турнир (2008/2009 год)", holds the edition: the autumn tour is in the first year and the spring tour in the second.
func totHolds(name, edition string) bool {
	first, last, ok := archive.PRSpan(name)
	m := totEditionRE.FindStringSubmatch(edition)
	if !ok || m == nil {
		return false
	}
	y, _ := strconv.Atoi(m[1])
	if m[2] == "fall" {
		return first == y
	}
	return last == y
}

// totRead reads the problems of a tour of the Tournament of Towns from the page of its tournament on turgor.ru. The TeX file of the papers comes first, then the PDFs of the two levels, then the text of the page; a position takes its text from the first that has it.
func totRead(ctx context.Context, fc *fetch.Client, source, edition string) ([]official.RuProblem, error) {
	n, prefix, ok := totTour(edition)
	if !ok {
		return nil, fmt.Errorf("tot: edition %q is not a tour", edition)
	}
	u := fmt.Sprintf("https://turgor.ru/problems/%d/index.php", n)
	raw, err := get(ctx, fc, source, u)
	if err != nil {
		return nil, err
	}
	page, err := official.RuDecode([]byte(raw))
	if err != nil {
		return nil, err
	}
	base, err := url.Parse(u)
	if err != nil {
		return nil, err
	}
	var tex, pdfs []string
	seen := map[string]bool{}
	for _, m := range totHref.FindAllStringSubmatch(page, -1) {
		ref, err := base.Parse(m[1])
		if err != nil || ref.Host != base.Host || seen[ref.String()] {
			continue
		}
		seen[ref.String()] = true
		name := path.Base(ref.Path)
		if t := totTeXFile.FindStringSubmatch(name); t != nil && t[1] == prefix && t[2] == strconv.Itoa(n) {
			tex = append(tex, ref.String())
		}
		if t := totPDFFile.FindStringSubmatch(name); t != nil && t[1] == prefix && t[2] == strconv.Itoa(n) {
			pdfs = append(pdfs, ref.String())
		}
	}
	var out []official.RuProblem
	have := map[string]bool{}
	add := func(ps []official.RuProblem) {
		for _, p := range ps {
			if p.Edition == edition && !have[p.Position] {
				have[p.Position] = true
				out = append(out, p)
			}
		}
	}
	for _, t := range tex {
		src, err := get(ctx, fc, source, t)
		if err != nil {
			return nil, err
		}
		add(official.TotTeX(official.TotTeXDecode([]byte(src)), edition, t))
	}
	for _, p := range pdfs {
		tier := byte('a')
		if strings.Contains(path.Base(p), "baz") {
			tier = 'o'
		}
		ps, err := totPDFRead(ctx, fc, source, p, edition, tier)
		if err != nil {
			return nil, err
		}
		add(ps)
	}
	add(official.TotPage(page, n, u))
	return out, nil
}

// totPDFRead gets a PDF of the papers of one level into the store and reads its problems from the runs of text of pdftohtml.
func totPDFRead(ctx context.Context, fc *fetch.Client, source, u, edition string, tier byte) ([]official.RuProblem, error) {
	raw, err := get(ctx, fc, source, u)
	if err != nil {
		return nil, err
	}
	dir, err := os.MkdirTemp("", "om-fill-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	f := filepath.Join(dir, "paper.pdf")
	if err := os.WriteFile(f, []byte(raw), 0o600); err != nil {
		return nil, err
	}
	b, err := figureRunner.Run(ctx, official.RuPDFTool, "-xml", "-i", "-stdout", "-q", f)
	if err != nil {
		return nil, err
	}
	ps := official.TotPDF(official.ParsePDFXML(b), edition, u, tier)
	for k := range ps {
		ps[k].Tool = official.RuPDFTool
	}
	return ps, nil
}
