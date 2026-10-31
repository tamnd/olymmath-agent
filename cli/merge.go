package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/review"
	"github.com/tamnd/olymmath-agent/similar"
)

func init() {
	register(command{"merge", "join two problem records of one problem (om merge imo/2019/p1 isl/2019/a1), or each pair of a known duplicate pattern with two strict signals (om merge -pattern)", runMerge})
}

// mergePlan is the merge of the record from into the record into.
type mergePlan struct {
	into, from *content.Problem
	s          match.Signals
	rule       string
}

func runMerge(env Env, args []string) int {
	fs := newFlags("merge", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dry := fs.Bool("dry", false, "print the merges and write no file")
	pat := fs.Bool("pattern", false, "merge each pair of records of a known duplicate pattern (spec 06.5) with two strict signals")
	rev := fs.String("review", "", "the id of a review that accepts the merge, for a merge with fewer than two strict signals")
	pos, err := parseMixed(fs, args)
	if err != nil || *pat == (len(pos) == 2) || (!*pat && len(pos) != 2) {
		fmt.Fprintln(env.Stderr, "om merge: give two appearances, or -pattern")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om merge: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	byAt := map[string]*content.Problem{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			byAt[a.At] = p
		}
	}
	ex := newEIndex(corpus.All())
	var plans []mergePlan
	if *pat {
		var weak []mergePlan
		plans, weak = patternMerges(corpus.All(), ex)
		for _, m := range weak {
			fmt.Fprintf(env.Stderr, "om merge: %s into %s needs a review: K %s, T %.3f, A %s, E %.3f\n", primaryAt(m.from), primaryAt(m.into), sigKey(m.s.K), m.s.T, sigAnswer(m.s.A), m.s.E)
		}
		fmt.Fprintf(env.Stderr, "om merge: %d pairs of a known pattern have two strict signals. %d other pairs agree by text but need a review.\n", len(plans), len(weak))
	} else {
		a, b := byAt[pos[0]], byAt[pos[1]]
		switch {
		case a == nil || b == nil:
			fmt.Fprintf(env.Stderr, "om merge: no problem has the appearance %s\n", map[bool]string{true: pos[0], false: pos[1]}[a == nil])
			return ExitUsage
		case a == b:
			fmt.Fprintf(env.Stderr, "om merge: %s and %s are already one problem\n", pos[0], pos[1])
			return ExitUsage
		}
		m := planMerge(a, b, ex)
		if !match.Strict(m.s) {
			r := mustReview(*root, *rev)
			if r == nil {
				fmt.Fprintf(env.Stderr, "om merge: the merge of %s into %s has fewer than two strict signals (K %s, T %.3f, A %s, E %.3f). Give -review with the id of a review that accepts it.\n", primaryAt(m.from), primaryAt(m.into), sigKey(m.s.K), m.s.T, sigAnswer(m.s.A), m.s.E)
				return ExitUsage
			}
		}
		plans = append(plans, m)
	}
	failed := 0
	for _, m := range plans {
		fmt.Fprintf(env.Stdout, "%s %s into %s %s: K %s, T %.3f, A %s, E %.3f, rule %s\n", m.from.Tag, primaryAt(m.from), m.into.Tag, primaryAt(m.into), sigKey(m.s.K), m.s.T, sigAnswer(m.s.A), m.s.E, m.rule)
		if *dry {
			continue
		}
		review := "-"
		if !match.Strict(m.s) {
			review = *rev
		}
		if err := applyMerge(*root, m, review); err != nil {
			fmt.Fprintf(env.Stderr, "om merge: %s into %s: %v\n", primaryAt(m.from), primaryAt(m.into), err)
			failed++
		}
	}
	if failed > 0 {
		return ExitFailure
	}
	return ExitOK
}

// mustReview gives the review with the id when it accepts a merge, or nil.
func mustReview(root, id string) *review.Review {
	if id == "" {
		return nil
	}
	revs, _ := review.Load(root)
	if r := revs[id]; r != nil && r.Decision == "accept" {
		return r
	}
	return nil
}

// planMerge gives the merge of two records: the record whose primary appearance goes first stays.
func planMerge(a, b *content.Problem, ex *eIndex) mergePlan {
	if canon.First(primaryAt(b), primaryAt(a)) {
		a, b = b, a
	}
	m := mergePlan{into: a, from: b, rule: "text"}
	m.s = match.Signals{T: match.Jaccard(match.Shingles(a.Statement), match.Shingles(b.Statement)), A: match.Answers(a.Answer, b.Answer), E: ex.E(a, b)}
	if r := canon.Pattern(primaryAt(a), primaryAt(b)); r != "" {
		m.s.K, m.rule = match.StrongKey, r
	}
	return m
}

// patternMerges finds the pairs of records of a known duplicate pattern, for example a shortlist problem and the contest problem of the same year, whose texts agree. It gives the merges with two strict signals and the other pairs with T of at least match.Doubt or E of at least match.EStrict. Each record joins its best partner only.
func patternMerges(ps []*content.Problem, ex *eIndex) (strict, weak []mergePlan) {
	byEdition := map[string][]*content.Problem{}
	year := func(p *content.Problem) string {
		at := primaryAt(p)
		comp, rest, _ := strings.Cut(at, "/")
		return strings.TrimSuffix(comp, "-sl") + "/" + rest[:min(4, len(rest))]
	}
	for _, p := range ps {
		if strings.TrimSpace(p.Statement) == "" || len(p.Appearances) == 0 {
			continue
		}
		k := year(p)
		if strings.HasPrefix(k, "isl/") {
			k = "imo/" + strings.TrimPrefix(k, "isl/")
		}
		byEdition[k] = append(byEdition[k], p)
	}
	keys := make([]string, 0, len(byEdition))
	for k := range byEdition {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	var out []mergePlan
	used := map[*content.Problem]bool{}
	for _, k := range keys {
		group := byEdition[k]
		var cands []mergePlan
		for i, a := range group {
			for _, b := range group[i+1:] {
				if canon.Pattern(primaryAt(a), primaryAt(b)) == "" {
					continue
				}
				if m := planMerge(a, b, ex); (m.s.T >= match.Doubt || m.s.E >= match.EStrict) && m.s.A != match.AnswerMismatch {
					cands = append(cands, m)
				}
			}
		}
		slices.SortStableFunc(cands, func(x, y mergePlan) int {
			if x.s.E != y.s.E {
				if x.s.E > y.s.E {
					return -1
				}
				return 1
			}
			return strings.Compare(primaryAt(x.from), primaryAt(y.from))
		})
		for _, m := range cands {
			if used[m.into] || used[m.from] {
				continue
			}
			if !match.Strict(m.s) {
				weak = append(weak, m)
				continue
			}
			used[m.into], used[m.from] = true, true
			out = append(out, m)
		}
	}
	return out, weak
}

// applyMerge writes one merge: the joined problem file, the solutions of the record that goes, the alias of its tag and the row of the merges register.
func applyMerge(root string, m mergePlan, rev string) error {
	full := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }
	into, err := os.ReadFile(full(m.into.Path)) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	from, err := os.ReadFile(full(m.from.Path)) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	joined, err := canon.MergeFiles(into, from)
	if err != nil {
		return err
	}
	dir := "solutions/" + strings.TrimSuffix(strings.TrimPrefix(m.into.Path, "problems/"), ".md")
	for _, s := range m.from.Solutions {
		b, err := os.ReadFile(full(s.Path)) // #nosec G304 -- the path is inside the content root.
		if err != nil {
			return err
		}
		b = []byte(strings.Replace(string(b), "\nproblem: "+m.from.Tag+"\n", "\nproblem: "+m.into.Tag+"\n", 1))
		to := dir + "/" + s.Tag + ".md"
		if err := os.MkdirAll(filepath.Dir(full(to)), 0o755); err != nil { // #nosec G301 -- content is public.
			return err
		}
		if err := os.WriteFile(full(to), b, 0o644); err != nil { // #nosec G306 G703 -- content files are public.
			return err
		}
		if err := os.Remove(full(s.Path)); err != nil {
			return err
		}
	}
	if err := os.WriteFile(full(m.into.Path), joined, 0o644); err != nil { // #nosec G306 G703 -- content files are public.
		return err
	}
	if err := os.Remove(full(m.from.Path)); err != nil {
		return err
	}
	if err := appendLine(full("registers/aliases.tsv"), fmt.Sprintf("%s\t%s\t%s is the same problem as %s", m.from.Tag, m.into.Tag, primaryAt(m.from), primaryAt(m.into))); err != nil {
		return err
	}
	for _, a := range m.from.Appearances {
		row := strings.Join([]string{a.At, primaryAt(m.into), sigKey(m.s.K), fmt.Sprintf("%.3f", m.s.T), sigAnswer(m.s.A), m.rule, rev, fmt.Sprintf("%.3f", m.s.E)}, "\t")
		if err := appendLine(full(review.MergesFile), row); err != nil {
			return err
		}
	}
	return nil
}

func appendLine(path, line string) error {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644) // #nosec G302 G304 -- a register of the content repository.
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(f, line); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func primaryAt(p *content.Problem) string {
	if a := p.Primary(); a != nil {
		return a.At
	}
	return dupWhere(p)
}

func sigKey(k match.Key) string {
	return map[match.Key]string{match.NoKey: "none", match.WeakKey: "weak", match.StrongKey: "strong"}[k]
}

func sigAnswer(a match.Answer) string {
	return map[match.Answer]string{match.NoAnswer: "none", match.AnswerMatch: "match", match.AnswerMismatch: "mismatch"}[a]
}

// eIndex gives the signal E of two problems: the cosine of the TF-IDF weights of the words and the TeX of their statements.
type eIndex struct {
	at   map[*content.Problem]int
	near [][]similar.Match
}

// eNear is the number of nearest problems that eIndex keeps for each problem.
const eNear = 5

func newEIndex(ps []*content.Problem) *eIndex {
	x := &eIndex{at: map[*content.Problem]int{}}
	var texts []string
	for _, p := range ps {
		if strings.TrimSpace(p.Statement) == "" {
			continue
		}
		x.at[p] = len(texts)
		texts = append(texts, dupText(p))
	}
	x.near = similar.Find(texts, eNear, 0)
	return x
}

// E gives the signal E of a and b when one of the two is the nearest problem of the other, or 0. A text that is closer to a third problem gets no E, because then the pair can be a near miss.
func (x *eIndex) E(a, b *content.Problem) float64 {
	i, ok := x.at[a]
	j, ok2 := x.at[b]
	if !ok || !ok2 {
		return 0
	}
	for _, p := range [][2]int{{i, j}, {j, i}} {
		if n := x.near[p[0]]; len(n) > 0 && n[0].Doc == p[1] {
			return n[0].Score
		}
	}
	return 0
}
