package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/review"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/site"
)

func init() {
	register(command{"import", "make problem records from the seed rows: choose the canonical statements, merge duplicates and write the review queue", runImport})
}

func runImport(env Env, args []string) int {
	fs := newFlags("import", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	comps := fs.String("comp", "", "a comma-separated list of competition codes; the default is all competitions")
	dry := fs.Bool("dry", false, "print the plan and write no file")
	maxDoubt := fs.Float64("max-doubt", canon.MaxDoubt, "skip a competition whose review doubts per appearance are above this share")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om import: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	al, err := seed.LoadAliases(*root, corpus)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om import: %v\n", err)
		return ExitFailure
	}
	mk, err := site.NewMarkup(nil)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om import: %v\n", err)
		return ExitFailure
	}
	s := &canon.Seeds{Corpus: corpus, Aliases: al, ThisYear: time.Now().Year(), Accept: func(t string) bool {
		_, errs := mk.HTML(t)
		return len(errs) == 0
	}}
	if *comps != "" {
		s.Comps = strings.Split(*comps, ",")
	}
	sources, err := seed.Sources(*dir)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om import: %v\n", err)
		return ExitFailure
	}
	for _, src := range sources {
		if err := seed.Read(*dir, src, func(r seed.Row) error { s.Add(r); return nil }); err != nil {
			fmt.Fprintf(env.Stderr, "om import: %s: %v\n", src, err)
			return ExitFailure
		}
	}
	cands := s.Candidates()
	ats := make([]string, 0, len(cands))
	for at := range cands {
		ats = append(ats, at)
	}
	sort.Strings(ats)
	var choices []canon.Choice
	for _, at := range ats {
		comp := corpus.Competition(strings.SplitN(at, "/", 2)[0])
		if ch, ok := canon.Choose(at, cands[at], original(comp)); ok {
			choices = append(choices, ch)
		}
	}
	choices, gated := canon.Gate(choices, *maxDoubt)
	for _, g := range gated {
		fmt.Fprintf(env.Stderr, "om import: %s skipped: %d doubts for %d appearances.\n", g.Comp, g.Doubts, g.Appearances)
	}
	recs, items := canon.Plan(choices)

	owner := map[string]*content.Problem{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			owner[a.At] = p
		}
	}
	var created, updated, merged int
	var mergeRows []string
	var newRecs []*canon.Record
	for _, r := range recs {
		if p := owner[r.Primary.At]; p != nil {
			comp := corpus.Competition(strings.SplitN(r.Primary.At, "/", 2)[0])
			add := canon.Add(p, r, original(comp))
			if add.Better != nil {
				items = append(items, canon.Item{ID: "better-" + p.Tag, Kind: "canonical", At: r.Primary.At, Source: add.Better.Source, Ref: add.Better.Ref, Text: p.Statement, Other: add.Better.Text})
			}
			if len(add.Statements) == 0 {
				continue
			}
			updated++
			if *dry {
				continue
			}
			full := filepath.Join(*root, filepath.FromSlash(p.Path))
			b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
			if err == nil {
				b, err = canon.Apply(b, add)
			}
			if err == nil {
				err = os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
			}
			if err != nil {
				fmt.Fprintf(env.Stderr, "om import: %s: %v\n", p.Path, err)
				return ExitFailure
			}
			continue
		}
		// An appearance that already belongs to a problem does not merge into a new problem.
		r.Merged = slices.DeleteFunc(r.Merged, func(m canon.Merge) bool { return owner[m.At] != nil })
		newRecs = append(newRecs, r)
	}
	created = len(newRecs)
	if !*dry && created > 0 {
		tags, err := mintTags(corpus, created)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		for i, r := range newRecs {
			comp := strings.SplitN(r.Primary.At, "/", 2)[0]
			b, err := canon.Render(r, tags[i], canon.TypeOf(comp, r.Primary.Answer))
			if err != nil {
				fmt.Fprintf(env.Stderr, "om import: %s: %v\n", r.Primary.At, err)
				return ExitFailure
			}
			if err := writeNew(*root, corpus, r, tags[i], b); err != nil {
				fmt.Fprintf(env.Stderr, "om import: %s: %v\n", r.Primary.At, err)
				return ExitFailure
			}
			for _, m := range r.Merged {
				merged++
				mergeRows = append(mergeRows, fmt.Sprintf("%s\t%s\t%s\t%.3f\t%s\t%s\t-", m.At, m.Into, keyName(m), m.Signals.T, answerName(m), cmpOr(m.Rule, "text")))
			}
		}
		if err := appendLines(*root, review.MergesFile, "# Each appearance that an import merged into a problem of another appearance: the appearance, the primary appearance, the signals K, T and A, the rule and the review. Rule D03 reads this file.\n", mergeRows); err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
	} else {
		for _, r := range newRecs {
			merged += len(r.Merged)
		}
	}
	if !*dry {
		if err := writeQueue(*root, items); err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
	}
	var reasons []string
	for k, n := range s.Skipped {
		reasons = append(reasons, fmt.Sprintf("%s %d", k, n))
	}
	sort.Strings(reasons)
	fmt.Fprintf(env.Stdout, "%d appearances with a candidate. %d new problems, %d merged appearances, %d problems with new links, %d review items. Rows skipped: %s.\n", len(choices), created, merged, updated, len(items), strings.Join(reasons, ", "))
	return ExitOK
}

// original gives the original language of a competition, or "" when the register does not say.
func original(c *content.Competition) string {
	if c == nil || len(c.Languages) == 0 {
		return ""
	}
	return c.Languages[0]
}

func writeNew(root string, corpus *content.Corpus, r *canon.Record, t string, b []byte) error {
	rel := ""
	for _, at := range r.Appearances() {
		if _, err := newEdition(root, corpus, at); err != nil {
			return err
		}
		if rel == "" {
			rel = strings.TrimSuffix(at, "/") + ".md"
		}
	}
	full := filepath.Join(root, "problems", filepath.FromSlash(rel))
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%s exists", rel)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
		return err
	}
	if err := os.WriteFile(full, b, 0o644); err != nil { // #nosec G306 G703 -- content files are public.
		return err
	}
	return addTag(root, t, "problem")
}

func writeQueue(root string, items []canon.Item) error {
	full := filepath.Join(root, filepath.FromSlash(review.QueueFile))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
		return err
	}
	var b strings.Builder
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	for _, it := range items {
		if err := enc.Encode(it); err != nil {
			return err
		}
	}
	return os.WriteFile(full, []byte(b.String()), 0o644) // #nosec G306 -- the queue is public.
}

func appendLines(root, rel, header string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	_, err := os.Stat(full)
	isNew := os.IsNotExist(err)
	f, err := os.OpenFile(full, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // #nosec G302 G304 -- a public register inside the content root.
	if err != nil {
		return err
	}
	if isNew {
		if _, err := f.WriteString(header); err != nil {
			_ = f.Close()
			return err
		}
	}
	if _, err := f.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func keyName(m canon.Merge) string {
	return [...]string{"none", "weak", "strong"}[m.Signals.K]
}

func answerName(m canon.Merge) string {
	return [...]string{"none", "match", "mismatch"}[m.Signals.A]
}

func cmpOr(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
