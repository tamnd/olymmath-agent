package cli

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"sample", "compare a random sample of canonical statements with the official papers (om sample -n 200 imo-official)", runSample})
}

// sampleItem is one problem of a sample, with the signals of its canonical text against the official paper.
type sampleItem struct {
	at      string
	order   float64
	numbers bool
	verdict string
	extra   []string
	// paper and text are the two texts, kept for the problems that a person checks.
	paper, text string
}

func runSample(env Env, args []string) int {
	fs := newFlags("sample", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	n := fs.Int("n", 200, "the size of the sample")
	seed := fs.Uint64("seed", 1, "the seed of the sample, so that a run can be made again")
	out := fs.String("out", "", "the report file (default reports/sample/SOURCE.md in the content root)")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 || pos[0] != "imo-official" {
		fmt.Fprintln(env.Stderr, "usage: om sample [-n N] [-seed S] imo-official")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om sample: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om sample: %v\n", err)
		return ExitFailure
	}
	year := time.Now().Year()
	papers := append(official.IMO(1959, year), official.Shortlist(official.ShortlistFirst, year-1)...)
	segs, err := official.Load(context.Background(), st, pos[0], papers)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om sample: %v\n", err)
		return ExitFailure
	}
	paperText := map[string]string{}
	for ed, ss := range segs {
		for _, s := range ss {
			paperText[ed+"/"+s.Position] = s.Text
		}
	}
	// The frame is each problem whose position has a text in an official paper, in the order of the locators.
	type framed struct {
		at string
		p  *content.Problem
	}
	var frame []framed
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			if _, ok := paperText[a.At]; ok && strings.TrimSpace(p.Statement) != "" {
				frame = append(frame, framed{a.At, p})
				break
			}
		}
	}
	slices.SortFunc(frame, func(a, b framed) int { return strings.Compare(a.at, b.at) })
	r := rand.New(rand.NewPCG(*seed, 0)) // #nosec G404 -- a sample that a run can make again, not a secret.
	r.Shuffle(len(frame), func(i, j int) { frame[i], frame[j] = frame[j], frame[i] })
	frame = frame[:min(*n, len(frame))]
	slices.SortFunc(frame, func(a, b framed) int { return strings.Compare(a.at, b.at) })
	var items []sampleItem
	count := map[string]int{}
	for _, f := range frame {
		paper := paperText[f.at]
		it := sampleItem{at: f.at, order: official.Order(paper, f.p.Statement), numbers: official.SameNumbers(paper, f.p.Statement)}
		switch {
		case it.order >= official.TraceLeast && it.numbers:
			it.verdict = "exact"
		case it.order >= SampleAgree && it.numbers:
			it.verdict = "agrees"
		default:
			it.verdict = "check"
			it.extra = official.NumberDiff(paper, f.p.Statement)
			it.paper, it.text = paper, f.p.Statement
		}
		count[it.verdict]++
		items = append(items, it)
	}
	if *out == "" {
		*out = filepath.Join(*root, "reports", "sample", pos[0]+".md")
	}
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil { // #nosec G301 -- reports are public.
		fmt.Fprintf(env.Stderr, "om sample: %v\n", err)
		return ExitFailure
	}
	if err := os.WriteFile(*out, []byte(sampleReport(pos[0], *seed, len(frame), items, count)), 0o644); err != nil { // #nosec G306 G703 -- reports are public.
		fmt.Fprintf(env.Stderr, "om sample: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d problems: %d exact, %d agree, %d to check.\n", len(items), count["exact"], count["agrees"], count["check"])
	return ExitOK
}

// SampleAgree is the least order signal for which a canonical text agrees with the paper when the numbers are the same. Below it, a person checks the text.
const SampleAgree = 0.6

func sampleReport(source string, seed uint64, n int, items []sampleItem, count map[string]int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Sample of canonical statements against %s\n\n", source)
	fmt.Fprintf(&b, "Made by `om sample -n %d -seed %d %s`. The frame is each IMO and ISL problem whose position has a text in an official English paper. The sample has %d problems.\n\n", n, seed, source, n)
	b.WriteString("For each problem, the report compares the canonical text with the text of the paper. The order signal is the share of the words that occur in the same order in both texts. The numbers agree when the two texts have the same set of numbers.\n\n")
	b.WriteString("- exact: the order signal is at least 0.9 and the numbers agree.\n")
	fmt.Fprintf(&b, "- agrees: the order signal is at least %.1f and the numbers agree.\n", SampleAgree)
	b.WriteString("- check: a person compares the two texts.\n\n")
	fmt.Fprintf(&b, "| Verdict | Problems |\n|---|---|\n| exact | %d |\n| agrees | %d |\n| check | %d |\n\n", count["exact"], count["agrees"], count["check"])
	b.WriteString("## Problems\n\n| Problem | Order | Numbers | Verdict | Numbers in one text only |\n|---|---|---|---|---|\n")
	for _, it := range items {
		nums := "agree"
		if !it.numbers {
			nums = "differ"
		}
		fmt.Fprintf(&b, "| %s | %.2f | %s | %s | %s |\n", it.at, it.order, nums, it.verdict, strings.Join(it.extra, " "))
	}
	b.WriteString("\n## Texts to check\n\nThe text of the paper comes from the text layer of the PDF, so it can lose superscripts, symbols and letters.\n")
	for _, it := range items {
		if it.verdict != "check" {
			continue
		}
		fmt.Fprintf(&b, "\n### %s\n\nPaper:\n\n```text\n%s\n```\n\nCanonical text:\n\n```text\n%s\n```\n", it.at, strings.TrimSpace(it.paper), strings.TrimSpace(it.text))
	}
	return b.String()
}
