package cli

import (
	"cmp"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/review"
)

func init() {
	register(command{"sample", "compare a random sample of canonical statements with the official papers, or record a review of one with -decide (om sample -n 200 imo-official)", runSample})
}

// sampleItem is one problem of a sample, with the signals of its canonical text against the official paper.
type sampleItem struct {
	at      string
	order   float64
	numbers bool
	verdict string
	extra   []string
	// paper and text are the two texts.
	paper, text string
	// id is the id of the review of the text, and review is that review, or nil.
	id     string
	review *review.Review
}

// SampleDecisions are the decisions of a review of a sample: accept when the canonical text states the problem of the paper with the same mathematics, reject when it does not.
var SampleDecisions = []string{"accept", "reject"}

// SampleID gives the id of the review of the canonical text of the appearance at against a source. The id depends on the text, so a review no longer applies when the text changes.
func SampleID(source, at, text string) string {
	return canon.ItemID("sample", source, at, strings.TrimSpace(text))
}

func runSample(env Env, args []string) int {
	fs := newFlags("sample", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	n := fs.Int("n", 200, "the size of the sample")
	seed := fs.Uint64("seed", 1, "the seed of the sample, so that a run can be made again")
	out := fs.String("out", "", "the report file (default reports/sample/SOURCE.md in the content root)")
	decide := fs.String("decide", "", "the appearance of the sample to review, such as imo/1985/p4")
	as := fs.String("as", "", "the decision: "+strings.Join(SampleDecisions, " or "))
	by := fs.String("by", "", "the reviewer, a person in registers/people.yaml, or agent")
	note := fs.String("note", "", "a note on the decision. A review by agent has a note that starts with "+strconv.Quote(content.AgentNote))
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 || pos[0] != "imo-official" {
		fmt.Fprintln(env.Stderr, "usage: om sample [-n N] [-seed S] [-decide AT -as accept|reject -by WHO -note TEXT] imo-official")
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
	// The sample is the n problems of the frame with the least rank. The rank of a problem depends only on the seed and its position, so a new problem in the frame changes the sample by at most one problem, and the reviews of the sample stay useful.
	slices.SortFunc(frame, func(a, b framed) int {
		return cmp.Or(strings.Compare(sampleRank(*seed, a.at), sampleRank(*seed, b.at)), strings.Compare(a.at, b.at))
	})
	frame = frame[:min(*n, len(frame))]
	slices.SortFunc(frame, func(a, b framed) int { return strings.Compare(a.at, b.at) })
	revs, errs := review.Load(*root)
	for _, e := range errs {
		fmt.Fprintf(env.Stderr, "om sample: %v\n", e)
	}
	var items []sampleItem
	count := map[string]int{}
	for _, f := range frame {
		paper := paperText[f.at]
		it := sampleItem{at: f.at, order: official.Order(paper, f.p.Statement), numbers: official.SameNumbers(paper, f.p.Statement), paper: paper, text: f.p.Statement}
		it.id = SampleID(pos[0], f.at, f.p.Statement)
		it.review = revs[it.id]
		switch {
		case it.order >= official.TraceLeast && it.numbers:
			it.verdict = "exact"
		case it.order >= SampleAgree && it.numbers:
			it.verdict = "agrees"
		default:
			it.verdict = "check"
			it.extra = official.NumberDiff(paper, f.p.Statement)
		}
		count[it.verdict]++
		count[sampleResult(it)]++
		items = append(items, it)
	}
	if *decide != "" {
		return sampleDecide(env, *root, pos[0], items, *decide, *as, *by, *note)
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
	fmt.Fprintf(env.Stdout, "%d problems: %d exact, %d agree, %d to check. %d match the paper, %d differ, %d have no review: %.1f%% match.\n", len(items), count["exact"], count["agrees"], count["check"], count["match"], count["differ"], count["open"], 100*float64(count["match"])/float64(max(len(items), 1)))
	return ExitOK
}

// sampleRank gives the rank of the position at in the sample of the seed.
func sampleRank(seed uint64, at string) string {
	h := sha256.Sum256([]byte(strconv.FormatUint(seed, 10) + "\x00" + at))
	return hex.EncodeToString(h[:])
}

// sampleResult tells if the canonical text of an item matches the paper. A review decides. With no review, a text that is exact or agrees matches, and a text to check is open.
func sampleResult(it sampleItem) string {
	switch {
	case it.review != nil && it.review.Decision == "accept":
		return "match"
	case it.review != nil:
		return "differ"
	case it.verdict == "check":
		return "open"
	}
	return "match"
}

// sampleDecide writes the review of the canonical text of the appearance at of the sample.
func sampleDecide(env Env, root, source string, items []sampleItem, at, as, by, note string) int {
	if !slices.Contains(SampleDecisions, as) || by == "" {
		fmt.Fprintf(env.Stderr, "om sample: -decide needs -as %s and -by\n", strings.Join(SampleDecisions, " or "))
		return ExitUsage
	}
	if by == "agent" && !strings.HasPrefix(note, content.AgentNote) {
		fmt.Fprintf(env.Stderr, "om sample: a review by agent needs a note that starts with %q\n", content.AgentNote)
		return ExitUsage
	}
	for _, it := range items {
		if it.at != at {
			continue
		}
		r := &review.Review{ID: it.id, Kind: "sample", Records: []string{at}, Decision: as, By: by, Date: time.Now().UTC().Format(time.DateOnly), Note: note}
		if err := review.Write(root, r); err != nil {
			fmt.Fprintf(env.Stderr, "om sample: %v\n", err)
			return ExitUsage
		}
		fmt.Fprintln(env.Stdout, r.Path())
		return ExitOK
	}
	fmt.Fprintf(env.Stderr, "om sample: %s is not in the sample\n", at)
	return ExitUsage
}

// SampleAgree is the least order signal for which a canonical text agrees with the paper when the numbers are the same. Below it, a person checks the text.
const SampleAgree = 0.6

func sampleReport(source string, seed uint64, n int, items []sampleItem, count map[string]int) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Sample of canonical statements against %s\n\n", source)
	fmt.Fprintf(&b, "Made by `om sample -n %d -seed %d %s`. The frame is each IMO and ISL problem whose position has a text in an official English paper. The sample is the %d problems of the frame with the least rank, where the rank of a problem is a hash of the seed and its position, so a new problem in the frame changes the sample by at most one problem.\n\n", n, seed, source, n)
	b.WriteString("For each problem, the report compares the canonical text with the text of the paper. The order signal is the share of the words that occur in the same order in both texts. The numbers agree when the two texts have the same set of numbers.\n\n")
	b.WriteString("- exact: the order signal is at least 0.9 and the numbers agree.\n")
	fmt.Fprintf(&b, "- agrees: the order signal is at least %.1f and the numbers agree.\n", SampleAgree)
	b.WriteString("- check: a person compares the two texts.\n\n")
	fmt.Fprintf(&b, "A person records a review of a text with `om sample -decide AT -as accept|reject -by WHO -note TEXT %s`: accept when the text states the problem of the paper with the same mathematics, reject when it does not. The id of the review depends on the text, so a review no longer applies when the text changes. A review by `agent` is a decision of the agent of the project, not the review of a person. A text matches the paper when its review accepts it, or when it has no review and is exact or agrees.\n\n", source)
	fmt.Fprintf(&b, "| Verdict | Problems |\n|---|---|\n| exact | %d |\n| agrees | %d |\n| check | %d |\n\n", count["exact"], count["agrees"], count["check"])
	reviewed, agent := 0, 0
	for _, it := range items {
		if it.review != nil {
			reviewed++
			if it.review.By == "agent" {
				agent++
			}
		}
	}
	fmt.Fprintf(&b, "| Result | Problems |\n|---|---|\n| match | %d |\n| differ | %d |\n| open | %d |\n\n", count["match"], count["differ"], count["open"])
	fmt.Fprintf(&b, "%d of the %d texts match the paper: %.1f%%. The target of spec 09.4 is 99.5%%. %d texts have a review, %d of them by agent.\n\n", count["match"], len(items), 100*float64(count["match"])/float64(max(len(items), 1)), reviewed, agent)
	b.WriteString("## Problems\n\n| Problem | Order | Numbers | Verdict | Numbers in one text only | Review |\n|---|---|---|---|---|---|\n")
	for _, it := range items {
		nums := "agree"
		if !it.numbers {
			nums = "differ"
		}
		rv := ""
		if it.review != nil {
			rv = fmt.Sprintf("%s by %s (%s)", it.review.Decision, it.review.By, it.id)
		}
		fmt.Fprintf(&b, "| %s | %.2f | %s | %s | %s | %s |\n", it.at, it.order, nums, it.verdict, strings.Join(it.extra, " "), rv)
	}
	b.WriteString("\n## Texts to check\n\nThese are the texts to check with no review that accepts them. The text of the paper comes from the text layer of the PDF, so it can lose superscripts, symbols and letters.\n")
	for _, it := range items {
		if it.verdict != "check" || sampleResult(it) == "match" {
			continue
		}
		fmt.Fprintf(&b, "\n### %s\n\nPaper:\n\n```text\n%s\n```\n\nCanonical text:\n\n```text\n%s\n```\n", it.at, strings.TrimSpace(it.paper), strings.TrimSpace(it.text))
	}
	return b.String()
}
