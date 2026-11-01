package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/normalize"
	"github.com/tamnd/olymmath-agent/official"
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
	anchor := fs.Bool("anchor", false, "give a position to each IMO row with no number from the official papers in the raw store, and to each other row with no number from the texts of zone Z3")
	dir2, remote := storeFlags(fs)
	doubts := fs.Bool("doubts", false, "print each doubt with its source and the source of the canonical candidate, to show why a competition is skipped")
	promote := fs.Bool("promote", false, "make a candidate of class H1 or H2 the canonical statement of an existing problem when it ranks above the canonical statement and its text agrees with it")
	explain := fs.String("explain", "", "write each seed row of a known competition to this TSV file with the appearance that it gives, or the reason why it gives none")
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
		for _, e := range errs {
			if !errors.Is(e, site.ErrFigure) {
				return false
			}
		}
		return true
	}}
	if *comps != "" {
		s.Comps = strings.Split(*comps, ",")
	}
	var traces []rowTrace
	if *explain != "" {
		s.Trace = func(r seed.Row, code, at, reason string) {
			if code != "" && code != "-" && reason != "other competition" {
				traces = append(traces, rowTrace{code: code, year: r.Year, number: r.Number, source: r.Source, ref: r.File + "#" + strconv.Itoa(r.Row), at: at, reason: reason})
			}
		}
	}
	sources, err := seed.Sources(*dir)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om import: %v\n", err)
		return ExitFailure
	}
	if *anchor {
		st, err := openStore(*dir2, *remote)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		papers, err := official.Load(context.Background(), st, "imo-official", official.IMO(1959, time.Now().Year()))
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		keys, err := zoneKeys(*dir, sources, corpus, al)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		fmt.Fprintf(env.Stderr, "om import: %d official papers, %d editions with texts of zone Z3.\n", len(papers), len(keys))
		s.Locate = func(code string, year int, text string) string {
			ed := code + "/" + strconv.Itoa(year)
			if code == "imo" {
				return papers.Locate(ed, text)
			}
			return keys.Locate(ed, text)
		}
	}
	if slices.Contains(sources, "hf-aops-wiki") {
		s.Topics = &canon.Topics{Aliases: al, Corpus: corpus}
		if err := seed.Read(*dir, "hf-aops-wiki", func(r seed.Row) error { s.Topics.Add(r); return nil }); err != nil {
			fmt.Fprintf(env.Stderr, "om import: hf-aops-wiki: %v\n", err)
			return ExitFailure
		}
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
	choices = canon.Confirm(choices)
	if *doubts {
		for _, ch := range choices {
			for _, d := range ch.Doubts {
				fmt.Fprintf(env.Stdout, "doubt %s: %s %s against %s %s, T %.2f, E %.2f\n", ch.At, ch.Best.Source, ch.Best.Ref, d.C.Source, d.C.Ref, d.T, d.E)
			}
		}
	}
	choices, gated := canon.Gate(choices, *maxDoubt)
	for _, g := range gated {
		fmt.Fprintf(env.Stderr, "om import: %s skipped: %d doubts for %d appearances.\n", g.Comp, g.Doubts, g.Appearances)
	}
	if *explain != "" {
		if err := writeExplain(*explain, traces, choices, gated); err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
	}
	recs, items := canon.Plan(choices)

	owner := map[string]*content.Problem{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			owner[a.At] = p
		}
	}
	var created, updated, merged, promoted int
	var mergeRows []string
	var newRecs []*canon.Record
	for _, r := range recs {
		if p := owner[r.Primary.At]; p != nil {
			comp := corpus.Competition(strings.SplitN(r.Primary.At, "/", 2)[0])
			add := canon.Add(p, r, original(comp))
			up := add.Better != nil && *promote && canon.Agrees(add.Better, p)
			if add.Better != nil && !up {
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
			if err == nil && up {
				_, rule := canon.Rank(add.Better, original(comp))
				b, err = canon.Promote(b, p.Canonical.Statement, add.ID(add.Better), add.Better.Text, rule)
				promoted++
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
		// A merge that the register has already does not get a second row.
		known := map[string]bool{}
		olds, err := review.Merges(*root)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		for _, m := range olds {
			known[m.At+"\x00"+m.Into] = true
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
				if known[m.At+"\x00"+m.Into] {
					continue
				}
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
		// The items of om anchor stay, because an import does not make them. The items of the competitions that this import did not read also stay.
		old, err := review.Queue(*root)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		for _, it := range old {
			comp, _, _ := strings.Cut(it.At, "/")
			if it.Kind == "official" || (len(s.Comps) > 0 && !slices.Contains(s.Comps, comp)) {
				items = append(items, it)
			}
		}
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
	if *anchor {
		fmt.Fprintf(env.Stdout, "%d rows got a position from an official paper or from a text of zone Z3.\n", s.Located)
	}
	if s.Numbered > 0 {
		fmt.Fprintf(env.Stdout, "%d rows got a position from the number of their AoPS topic.\n", s.Numbered)
	}
	fmt.Fprintf(env.Stdout, "%d appearances with a candidate. %d new problems, %d merged appearances, %d problems with new links, %d promoted statements, %d review items. Rows skipped: %s.\n", len(choices), created, merged, updated, promoted, len(items), strings.Join(reasons, ", "))
	return ExitOK
}

// rowTrace is one seed row for the -explain file.
type rowTrace struct {
	code, number, source, ref, at, reason string
	year                                  int
}

// writeExplain writes one line for each traced row: the competition, the year, the number, the source, the reference of the row, the appearance and the reason. A row that gives a candidate has the reason "gated" when Gate removed its competition, "doubt" when its text does not agree with the canonical candidate, "no choice" when no candidate of its appearance can be canonical, and "ok" in the other cases.
func writeExplain(path string, traces []rowTrace, choices []canon.Choice, gated []canon.Gated) error {
	drop := map[string]bool{}
	for _, g := range gated {
		drop[g.Comp] = true
	}
	chosen := map[string]bool{}
	doubt := map[string]bool{}
	for _, ch := range choices {
		chosen[ch.At] = true
		for _, d := range ch.Doubts {
			doubt[d.C.Ref] = true
		}
	}
	var b strings.Builder
	b.WriteString("code\tyear\tnumber\tsource\tref\tat\treason\n")
	for _, t := range traces {
		reason := t.reason
		if reason == "" {
			switch {
			case drop[t.code]:
				reason = "gated"
			case !chosen[t.at]:
				reason = "no choice"
			case doubt[t.ref]:
				reason = "doubt"
			default:
				reason = "ok"
			}
		}
		fmt.Fprintf(&b, "%s\t%d\t%s\t%s\t%s\t%s\t%s\n", t.code, t.year, t.number, t.source, t.ref, t.at, reason)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644) // #nosec G306 -- a diagnostic file.
}

// zoneKeys gives the texts of zone Z3 with a known position, by edition. A text of zone Z3 is a link only, so the import does not publish it, but it can give the position of a row with no number, for example an AoPS row of a USAMO problem.
func zoneKeys(dir string, sources []string, corpus *content.Corpus, al *seed.Aliases) (official.Papers, error) {
	z3 := func(id string) bool {
		src := corpus.Source(id)
		return src != nil && src.Zone.Statement == "Z3"
	}
	keys := official.Papers{}
	for _, src := range sources {
		err := seed.Read(dir, src, func(r seed.Row) error {
			if !z3(r.Source) && !z3(r.Author) {
				return nil
			}
			code, pos := al.Code(r.Labels), seed.Position(r.Number)
			if code == "" || code == "-" || r.Year == 0 || pos == "" {
				return nil
			}
			ed := code + "/" + strconv.Itoa(r.Year)
			keys[ed] = append(keys[ed], official.Segment{Position: pos, Text: normalize.Statement(r.Statement, pos)})
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src, err)
		}
	}
	return keys, nil
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
