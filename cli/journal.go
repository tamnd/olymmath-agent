package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
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

// journalUsage tells the three forms of om journal.
const journalUsage = `usage:
  om journal [-from YEAR] [-to YEAR] [-dry] JOURNAL                  read the numbers from the text layer of each issue
  om journal [-from YEAR] [-to YEAR] -scans FILE JOURNAL             get the PDF of each issue that the text lane cannot read, and list it in FILE
  om journal -marks FILE -extraction EXTRACTION [-dry] JOURNAL       add the numbers that a model read from the page images
JOURNAL is one of: %s
`

// journalLink is a new problem of a journal: its locator and the statement that links to the PDF of its issue.
type journalLink struct {
	at   string
	cand canon.Candidate
	page int
	// extraction names the model that read the number and the page from the page image of a scan. It is "" when they come from the text layer.
	extraction string
	// also are the other appearances of the problem, such as the paper of another grade that has the same problem.
	also []string
}

func runJournal(env Env, args []string) int {
	fs := newFlags("journal", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 0, "the first year")
	to := fs.Int("to", time.Now().Year(), "the last year")
	dry := fs.Bool("dry", false, "print the plan and write no file")
	scans := fs.String("scans", "", "get the PDF of each issue that the text lane cannot read (a scan, a text layer that is not readable, or a text with no number) and write a TSV line for it to this file: year, volume, number, month, PDF link and the hash of the PDF in the store. It writes no record")
	marks := fs.String("marks", "", "add the problems of this TSV file of numbers that a model read from the page images: year, volume, number, month, PDF link, problem number and page. It needs no network")
	extraction := fs.String("extraction", "", "the model that read the numbers of -marks, as vision/claude@2.1.323 (claude-opus-5-5)")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	names := make([]string, 0, len(archive.Journals))
	for k := range archive.Journals {
		names = append(names, k)
	}
	slices.Sort(names)
	if len(pos) != 1 || archive.Journals[pos[0]].Comp == "" || (*marks != "" && (*extraction == "" || *scans != "")) {
		fmt.Fprintf(env.Stderr, journalUsage, strings.Join(names, "|"))
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
	if *marks != "" {
		return journalMarks(env, *root, corpus, id, j, *marks, *extraction, *dry)
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
	var rows []string
	for _, is := range j.Issues(listing) {
		if is.Year < *from || is.Year > *to {
			continue
		}
		issues++
		if *scans != "" {
			row, err := journalScan(ctx, fc, id, j, &is)
			if err != nil {
				fmt.Fprintf(env.Stderr, "om journal: %s %d no. %d: %v\n", id, is.Year, is.Number, err)
				failed++
			} else if row != "" {
				rows = append(rows, row)
			}
			continue
		}
		if j.Scanned(is) {
			fmt.Fprintf(env.Stderr, "om journal: %s %d no. %d: a scan, so its numbers need the vision lane\n", id, is.Year, is.Number)
			scanned++
			continue
		}
		marks, pdf, _, err := journalIssue(ctx, fc, id, j, &is)
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
	if *scans != "" {
		fmt.Fprintf(env.Stdout, "%s: %d issues, %d for the vision lane, %d failed.\n", id, issues, len(rows), failed)
		if err := os.WriteFile(*scans, []byte(strings.Join(rows, "")), 0o600); err != nil {
			fmt.Fprintf(env.Stderr, "om journal: %v\n", err)
			return ExitFailure
		}
		return ExitOK
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

// issuePDF gets the PDF of an issue into the store. It gives the link to the PDF, its bytes and the hash of its capture.
func issuePDF(ctx context.Context, fc *fetch.Client, id string, j archive.Journal, is *archive.Issue) (pdf, body, hash string, err error) {
	pdf = is.PDF
	if pdf == "" {
		page, err := get(ctx, fc, id, is.Page)
		if err != nil {
			return "", "", "", err
		}
		if pdf = j.PDF(page); pdf == "" {
			return "", "", "", fmt.Errorf("%s links no PDF", is.Page)
		}
	}
	body, hash, err = getCapture(ctx, fc, id, pdf)
	return pdf, body, hash, err
}

// journalIssue gets the PDF of an issue and reads the problem numbers from its text layer. It gives the numbers, the link to the PDF and the hash of the PDF in the store.
func journalIssue(ctx context.Context, fc *fetch.Client, id string, j archive.Journal, is *archive.Issue) ([]archive.Mark, string, string, error) {
	pdf, body, hash, err := issuePDF(ctx, fc, id, j, is)
	if err != nil {
		return nil, "", "", err
	}
	text, err := official.Text(ctx, []byte(body))
	if err != nil {
		return nil, pdf, hash, err
	}
	marks := j.Problems(text)
	if len(marks) == 0 {
		return nil, pdf, hash, errNoNumbers
	}
	return marks, pdf, hash, nil
}

// journalScan gets the PDF of an issue that the text lane cannot read: a scan, a PDF whose text layer is not readable, or a PDF whose text gives no number. It gives the TSV line of the issue for the vision lane, or "" when the text lane reads the issue.
func journalScan(ctx context.Context, fc *fetch.Client, id string, j archive.Journal, is *archive.Issue) (string, error) {
	var pdf, hash string
	var err error
	if j.Scanned(*is) {
		pdf, _, hash, err = issuePDF(ctx, fc, id, j, is)
	} else {
		_, pdf, hash, err = journalIssue(ctx, fc, id, j, is)
		switch {
		case err == nil:
			return "", nil
		case errors.Is(err, errNoNumbers), errors.Is(err, official.ErrNoText):
			err = nil
		}
	}
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d\t%d\t%d\t%s\t%s\t%s\n", is.Year, is.Volume, is.Number, is.Month, pdf, hash), nil
}

// journalMark is one line of a -marks file: an issue, and a problem number with its page that a model read from the page image.
type journalMark struct {
	issue archive.Issue
	mark  archive.Mark
	line  int
}

// readMarks reads a -marks file. Each line has the year, the volume, the number and the month of the issue, the link to its PDF, the problem number and the page of the PDF, split by tabs. An empty line and a line that starts with # are left out.
func readMarks(file string) ([]journalMark, error) {
	b, err := os.ReadFile(file) // #nosec G304 -- the path comes from the command line.
	if err != nil {
		return nil, err
	}
	var out []journalMark
	for i, line := range strings.Split(string(b), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 7 {
			return nil, fmt.Errorf("%s:%d: %d fields, and a line needs 7", file, i+1, len(f))
		}
		var n [4]int
		for k, s := range []string{f[0], f[1], f[2], f[6]} {
			if n[k], err = strconv.Atoi(s); err != nil || n[k] <= 0 {
				return nil, fmt.Errorf("%s:%d: %q is not a number above 0", file, i+1, s)
			}
		}
		label := f[5]
		if !markRE.MatchString(label) || f[3] == "" || !strings.HasPrefix(f[4], "https://") {
			return nil, fmt.Errorf("%s:%d: a bad problem number, month or PDF link", file, i+1)
		}
		pos := strings.ToLower(label)
		if label[0] >= '0' && label[0] <= '9' {
			pos = "p" + label
		}
		out = append(out, journalMark{
			issue: archive.Issue{Year: n[0], Volume: n[1], Number: n[2], Month: f[3], PDF: f[4]},
			mark:  archive.Mark{Position: pos, Label: label, Page: n[3]},
			line:  i + 1,
		})
	}
	return out, nil
}

// markRE is a problem number of a -marks file: digits, or a letter and digits as J745.
var markRE = regexp.MustCompile(`^[A-Z]?[0-9]{1,5}$`)

// markGaps checks the numbers of a -marks file against each other. The numbers of a journal run on from issue to issue, so the numbers of an issue must be above the numbers of the issues before it. It gives a line for each number that is there twice, each number that is lower than a number of an earlier issue, and each gap in the run.
func markGaps(ms []journalMark) (dups, gaps []string) {
	type key struct{ vol, num int }
	seen := map[string]int{}
	byIssue := map[key][]int{}
	var keys []key
	for _, m := range ms {
		if l, ok := seen[m.mark.Label]; ok {
			dups = append(dups, fmt.Sprintf("problem %s is on line %d and on line %d", m.mark.Label, l, m.line))
			continue
		}
		seen[m.mark.Label] = m.line
		n, err := strconv.Atoi(m.mark.Label)
		if err != nil {
			continue
		}
		k := key{m.issue.Volume, m.issue.Number}
		if byIssue[k] == nil {
			keys = append(keys, k)
		}
		byIssue[k] = append(byIssue[k], n)
	}
	slices.SortFunc(keys, func(a, b key) int {
		if a.vol != b.vol {
			return a.vol - b.vol
		}
		return a.num - b.num
	})
	last, lastKey := 0, key{}
	for _, k := range keys {
		ns := byIssue[k]
		slices.Sort(ns)
		if last > 0 && ns[0] <= last {
			dups = append(dups, fmt.Sprintf("vol. %d no. %d starts at %d, which is not above %d of vol. %d no. %d", k.vol, k.num, ns[0], last, lastKey.vol, lastKey.num))
		}
		prev := last
		for i, n := range ns {
			if i > 0 {
				prev = ns[i-1]
			}
			if prev > 0 && n > prev+1 {
				gaps = append(gaps, fmt.Sprintf("vol. %d no. %d: no number from %d to %d", k.vol, k.num, prev+1, n-1))
			}
		}
		last, lastKey = ns[len(ns)-1], k
	}
	return dups, gaps
}

// journalMarks adds the problems of a -marks file. A number that is there twice stops the run, since one of its pages is wrong. A gap is only told: a journal can skip a number, and an issue can be missing from the archive.
func journalMarks(env Env, root string, corpus *content.Corpus, id string, j archive.Journal, file, extraction string, dry bool) int {
	ms, err := readMarks(file)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om journal: %v\n", err)
		return ExitUsage
	}
	dups, gaps := markGaps(ms)
	for _, g := range gaps {
		fmt.Fprintf(env.Stderr, "om journal: gap: %s\n", g)
	}
	for _, d := range dups {
		fmt.Fprintf(env.Stderr, "om journal: duplicate: %s\n", d)
	}
	if len(dups) > 0 {
		return ExitFindings
	}
	src := corpus.Source(id)
	owners := official.Owners(corpus)
	var news []journalLink
	have := 0
	for _, m := range ms {
		at := fmt.Sprintf("%s/%d/%s", j.Comp, m.issue.Year, m.mark.Position)
		if owners[at] != nil {
			have++
			continue
		}
		c := canon.Candidate{Source: id, Origin: src.Origin, Zone: "Z3", Lang: "en", Name: j.Name(m.issue, m.mark), URL: m.issue.PDF}
		news = append(news, journalLink{at: at, cand: c, page: m.mark.Page, extraction: extraction})
	}
	fmt.Fprintf(env.Stdout, "%s: %d numbers, %d already in the content, %d gaps, %d new problems.\n", id, len(ms), have, len(gaps), len(news))
	if dry {
		for _, n := range news {
			fmt.Fprintf(env.Stdout, "%s\t%s\tpage %d\n", n.at, n.cand.Name, n.page)
		}
		return ExitOK
	}
	if err := writeJournal(root, corpus, news); err != nil {
		fmt.Fprintf(env.Stderr, "om journal: %v\n", err)
		return ExitFailure
	}
	return ExitOK
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
		b, err := canon.RenderLink(n.at, tags[i], &n.cand, n.page, rule, n.also...)
		if n.extraction != "" {
			b, err = canon.RenderReadLink(n.at, tags[i], &n.cand, n.page, rule, n.extraction, n.also...)
		}
		if err != nil {
			return err
		}
		rec := &canon.Record{Primary: canon.Choice{At: n.at}}
		for _, a := range n.also {
			rec.Merged = append(rec.Merged, canon.Merge{At: a})
		}
		if err := writeNew(root, corpus, rec, tags[i], b); err != nil {
			return fmt.Errorf("%s: %w", n.at, err)
		}
	}
	return nil
}
