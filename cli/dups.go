package cli

import (
	"bufio"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/similar"
)

func init() {
	register(command{"dups", "measure the duplicate rate: the nearest problems by E of a random sample, with the decisions of a person (om dups -n 500)", runDups})
}

// dupPair is a problem of the sample and one of its nearest problems.
type dupPair struct {
	a, b    *content.Problem
	e       float64
	verdict string
}

// DupLeast is the least E of a pair that a person must look at. Below it, the texts share too few rare words to be one problem.
const DupLeast = 0.5

func runDups(env Env, args []string) int {
	fs := newFlags("dups", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	n := fs.Int("n", 500, "the size of the sample")
	k := fs.Int("k", 5, "the number of nearest problems of each problem of the sample")
	seed := fs.Uint64("seed", 1, "the seed of the sample, so that a run can be made again")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om dups: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	var ps []*content.Problem
	for _, p := range corpus.Problems() {
		if strings.TrimSpace(p.Statement) != "" {
			ps = append(ps, p)
		}
	}
	slices.SortFunc(ps, func(a, b *content.Problem) int { return strings.Compare(a.Tag, b.Tag) })
	texts := make([]string, len(ps))
	for i, p := range ps {
		texts[i] = dupText(p)
	}
	near := similar.Find(texts, *k, 0)
	order := make([]int, len(ps))
	for i := range order {
		order[i] = i
	}
	r := rand.New(rand.NewPCG(*seed, 0)) // #nosec G404 -- a sample that a run can make again, not a secret.
	r.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	order = order[:min(*n, len(order))]
	dir := filepath.Join(*root, "reports", "sample")
	decided, err := readDecisions(filepath.Join(dir, "duplicates.tsv"))
	if err != nil {
		fmt.Fprintf(env.Stderr, "om dups: %v\n", err)
		return ExitFailure
	}
	var pairs []dupPair
	same, open := map[string]bool{}, 0
	for _, i := range order {
		for _, m := range near[i] {
			if m.Score < DupLeast {
				continue
			}
			pr := dupPair{a: ps[i], b: ps[m.Doc], e: m.Score, verdict: decided[pairKey(ps[i].Tag, ps[m.Doc].Tag)]}
			switch pr.verdict {
			case "same":
				same[pr.a.Tag] = true
			case "":
				open++
			}
			pairs = append(pairs, pr)
		}
	}
	if err := os.MkdirAll(dir, 0o750); err != nil {
		fmt.Fprintf(env.Stderr, "om dups: %v\n", err)
		return ExitFailure
	}
	out := filepath.Join(dir, "duplicates.md")
	if err := os.WriteFile(out, []byte(dupsReport(len(order), len(ps), *k, *seed, pairs, len(same), open)), 0o600); err != nil {
		fmt.Fprintf(env.Stderr, "om dups: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintf(env.Stdout, "%d problems in the sample, %d pairs with E of at least %.1f, %d without a decision. %d missed duplicates: %.2f%%. The report is in %s.\n", len(order), len(pairs), DupLeast, open, len(same), 100*float64(len(same))/float64(max(len(order), 1)), out)
	return ExitOK
}

// dupMath finds the TeX of a text.
var dupMath = regexp.MustCompile(`(?s)\$\$?(.+?)\$?\$`)

// dupText gives the text of a problem for the signal E: its statement and the runs of its TeX.
func dupText(p *content.Problem) string { return p.Statement + mathGrams(p.Statement) }

// mathGrams gives the runs of four characters of the TeX of a text, with no spaces, as words. The words of the text alone do not tell two inequalities apart, because their words are only the names of the TeX commands. A run is written in hex, so that it stays one word.
func mathGrams(text string) string {
	var b strings.Builder
	for _, m := range dupMath.FindAllStringSubmatch(text, -1) {
		t := strings.Join(strings.Fields(m[1]), "")
		for i := 0; i+4 <= len(t); i++ {
			fmt.Fprintf(&b, " m%x", t[i:i+4])
		}
	}
	return b.String()
}

// pairKey gives the key of a pair of tags, in the same order whatever the order of the two.
func pairKey(a, b string) string {
	if b < a {
		a, b = b, a
	}
	return a + " " + b
}

// readDecisions reads the decisions of a person on pairs: one line for each pair with the two tags and same, different or variant, separated by tabs. A variant is a changed form of the other problem. It is not a missed duplicate. A line that starts with # is a comment.
func readDecisions(path string) (map[string]string, error) {
	out := map[string]string{}
	f, err := os.Open(path) // #nosec G304 -- a file of the content repository.
	if os.IsNotExist(err) {
		return out, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	sc := bufio.NewScanner(f)
	for line := 1; sc.Scan(); line++ {
		t := strings.TrimSpace(sc.Text())
		if t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		cols := strings.Split(t, "\t")
		if len(cols) < 3 || (cols[2] != "same" && cols[2] != "different" && cols[2] != "variant") {
			return nil, fmt.Errorf("%s:%d: want two tags and same, different or variant", path, line)
		}
		out[pairKey(cols[0], cols[1])] = cols[2]
	}
	return out, sc.Err()
}

func dupsReport(n, total, k int, seed uint64, pairs []dupPair, same, open int) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Duplicate rate\n\n")
	p("This report is made by `om dups` (spec 09.4). It takes %d random problems of the %d problems with a statement (seed %d) and finds the %d nearest problems of each by E, the cosine of the TF-IDF weights of their words. A person decides on each pair with E of at least %.1f in `reports/sample/duplicates.tsv`. A problem of the sample with a pair decided as `same` is a missed duplicate.\n\n", n, total, seed, k, DupLeast)
	p("| Sample | Pairs to decide | Without a decision | Missed duplicates | Rate |\n|---:|---:|---:|---:|---:|\n")
	p("| %d | %d | %d | %d | %.2f%% |\n\n", n, len(pairs), open, same, 100*float64(same)/float64(max(n, 1)))
	if len(pairs) == 0 {
		return b.String()
	}
	slices.SortStableFunc(pairs, func(x, y dupPair) int {
		if x.e != y.e {
			if x.e > y.e {
				return -1
			}
			return 1
		}
		return strings.Compare(x.a.Tag, y.a.Tag)
	})
	p("## Pairs\n\n| E | Problem | Nearest | Decision |\n|---:|---|---|---|\n")
	for _, pr := range pairs {
		v := pr.verdict
		if v == "" {
			v = "open"
		}
		p("| %.2f | %s %s | %s %s | %s |\n", pr.e, pr.a.Tag, dupWhere(pr.a), pr.b.Tag, dupWhere(pr.b), v)
	}
	return b.String()
}

// dupWhere gives the primary appearance of a problem, or loose for a problem with no appearance.
func dupWhere(p *content.Problem) string {
	for _, a := range p.Appearances {
		if a.Primary {
			return a.At
		}
	}
	if len(p.Appearances) > 0 {
		return p.Appearances[0].At
	}
	return "loose"
}
