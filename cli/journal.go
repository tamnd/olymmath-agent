package cli

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"journal", "add a link-only problem for each problem number of the issues of a journal (om journal crux|mathreflections)", runJournal})
}

// journalLink is a new problem of a journal: its locator and the statement that links to the PDF of its issue.
type journalLink struct {
	at   string
	cand canon.Candidate
	page int
}

func runJournal(env Env, args []string) int {
	fs := newFlags("journal", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 0, "the first year")
	to := fs.Int("to", time.Now().Year(), "the last year")
	dry := fs.Bool("dry", false, "print the plan and write no file")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	names := make([]string, 0, len(archive.Journals))
	for k := range archive.Journals {
		names = append(names, k)
	}
	slices.Sort(names)
	if len(pos) != 1 || archive.Journals[pos[0]].Comp == "" {
		fmt.Fprintf(env.Stderr, "usage: om journal [-from YEAR] [-to YEAR] [-dry] %s\n", strings.Join(names, "|"))
		return ExitUsage
	}
	id, j := pos[0], archive.Journals[pos[0]]
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om journal: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	src := corpus.Source(id)
	if src == nil || corpus.Competition(j.Comp) == nil {
		fmt.Fprintf(env.Stderr, "om journal: the content needs the source %s and the competition %s\n", id, j.Comp)
		return ExitUsage
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om journal: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	if src.CrawlDelay > 0 {
		fc.Delay = time.Duration(src.CrawlDelay) * time.Second
	}
	ctx := context.Background()
	listing, err := get(ctx, fc, id, j.Listing)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om journal: %v\n", err)
		return ExitFailure
	}
	owners := official.Owners(corpus)
	var news []journalLink
	var issues, scanned, failed int
	for _, is := range j.Issues(listing) {
		if is.Year < *from || is.Year > *to {
			continue
		}
		issues++
		if j.Scanned(is) {
			fmt.Fprintf(env.Stderr, "om journal: %s %d no. %d: a scan, so its numbers need the vision lane\n", id, is.Year, is.Number)
			scanned++
			continue
		}
		marks, pdf, err := journalIssue(ctx, fc, id, j, &is)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om journal: %s %d no. %d: %v\n", id, is.Year, is.Number, err)
			failed++
			continue
		}
		for _, m := range marks {
			at := fmt.Sprintf("%s/%d/%s", j.Comp, is.Year, m.Position)
			if owners[at] != nil || slices.ContainsFunc(news, func(n journalLink) bool { return n.at == at }) {
				continue
			}
			// The record only links to the issue, so its statement is in zone Z3 whatever the zone of the source (spec 11.2).
			c := canon.Candidate{Source: id, Origin: src.Origin, Zone: "Z3", Lang: "en", Name: j.Name(is, m), URL: pdf}
			news = append(news, journalLink{at: at, cand: c, page: m.Page})
		}
	}
	fmt.Fprintf(env.Stdout, "%s: %d issues, %d scans left for the vision lane, %d issues with no numbers, %d new problems.\n", id, issues, scanned, failed, len(news))
	if *dry {
		for _, n := range news {
			fmt.Fprintf(env.Stdout, "%s\t%s\tpage %d\n", n.at, n.cand.Name, n.page)
		}
		return ExitOK
	}
	if err := writeJournal(*root, corpus, news); err != nil {
		fmt.Fprintf(env.Stderr, "om journal: %v\n", err)
		return ExitFailure
	}
	return ExitOK
}

// errNoNumbers tells that the text of an issue gave no problem number.
var errNoNumbers = errors.New("the text gives no problem number")

// journalIssue gets the PDF of an issue and reads the problem numbers from its text layer. It gives the numbers and the link to the PDF.
func journalIssue(ctx context.Context, fc *fetch.Client, id string, j archive.Journal, is *archive.Issue) ([]archive.Mark, string, error) {
	pdf := is.PDF
	if pdf == "" {
		page, err := get(ctx, fc, id, is.Page)
		if err != nil {
			return nil, "", err
		}
		if pdf = j.PDF(page); pdf == "" {
			return nil, "", fmt.Errorf("%s links no PDF", is.Page)
		}
	}
	body, err := get(ctx, fc, id, pdf)
	if err != nil {
		return nil, "", err
	}
	text, err := official.Text(ctx, []byte(body))
	if err != nil {
		return nil, "", err
	}
	marks := j.Problems(text)
	if len(marks) == 0 {
		return nil, "", errNoNumbers
	}
	return marks, pdf, nil
}

// writeJournal writes a link-only problem for each new position. The rule of each statement comes from the original language of its competition.
func writeJournal(root string, corpus *content.Corpus, news []journalLink) error {
	if len(news) == 0 {
		return nil
	}
	tags, err := mintTags(corpus, len(news))
	if err != nil {
		return err
	}
	for i, n := range news {
		comp, _, _ := strings.Cut(n.at, "/")
		_, rule := canon.Rank(&canon.Candidate{Origin: n.cand.Origin, Lang: n.cand.Lang}, original(corpus.Competition(comp)))
		b, err := canon.RenderLink(n.at, tags[i], &n.cand, n.page, rule)
		if err != nil {
			return err
		}
		if err := writeNew(root, corpus, &canon.Record{Primary: canon.Choice{At: n.at}}, tags[i], b); err != nil {
			return fmt.Errorf("%s: %w", n.at, err)
		}
	}
	return nil
}
