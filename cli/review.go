package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/review"
)

func init() {
	register(command{"review", "show the open items of the review queue, or record a decision with -decide", runReview})
}

func runReview(env Env, args []string) int {
	fs := newFlags("review", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	kind := fs.String("kind", "", "show only the items of this kind: pair, doubt, dispute or canonical")
	limit := fs.Int("n", 20, "show at most this number of items; 0 means all items")
	decide := fs.String("decide", "", "the id of the item or the merge to decide")
	as := fs.String("as", "", "the decision: "+strings.Join(review.Decisions, ", "))
	by := fs.String("by", "", "the reviewer, a person in registers/people.yaml")
	note := fs.String("note", "", "a note on the decision")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	items, err := review.Queue(*root)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om review: %v\n", err)
		return ExitFailure
	}
	revs, errs := review.Load(*root)
	for _, e := range errs {
		fmt.Fprintf(env.Stderr, "om review: %v\n", e)
	}
	if *decide != "" {
		r := &review.Review{ID: *decide, Decision: *as, By: *by, Note: *note, Date: time.Now().UTC().Format(time.DateOnly)}
		for _, it := range items {
			if it.ID == *decide {
				r.Kind = it.Kind
				r.Records = append([]string{it.At}, nonEmpty(it.With)...)
			}
		}
		if r.Kind == "" {
			ms, err := review.Merges(*root)
			if err != nil {
				fmt.Fprintf(env.Stderr, "om review: %v\n", err)
				return ExitFailure
			}
			for _, m := range ms {
				if m.ID() == *decide {
					r.Kind = "merge"
					r.Records = []string{m.Into, m.At}
				}
			}
		}
		if r.Kind == "" {
			fmt.Fprintf(env.Stderr, "om review: no item or merge has the id %s\n", *decide)
			return ExitUsage
		}
		if err := review.Write(*root, r); err != nil {
			fmt.Fprintf(env.Stderr, "om review: %v\n", err)
			return ExitUsage
		}
		fmt.Fprintln(env.Stdout, r.Path())
		return ExitOK
	}
	open, shown := 0, 0
	for _, it := range items {
		if revs[it.ID] != nil || (*kind != "" && it.Kind != *kind) {
			continue
		}
		open++
		if *limit > 0 && shown >= *limit {
			continue
		}
		shown++
		head := fmt.Sprintf("%s %s %s", it.ID, it.Kind, it.At)
		if it.With != "" {
			head += " with " + it.With
		}
		if it.Source != "" {
			head += " from " + it.Source
		}
		fmt.Fprintf(env.Stdout, "%s (T %.3f, K %s, A %s)\n  canonical: %s\n  other:     %s\n\n", head, it.T, dash(it.K), dash(it.A), oneLine(it.Text), oneLine(it.Other))
	}
	fmt.Fprintf(env.Stdout, "%d open items, %d shown, %d reviews.\n", open, shown, len(revs))
	return ExitOK
}

func nonEmpty(s string) []string {
	if s == "" {
		return nil
	}
	return []string{s}
}

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// oneLine puts a text on one line and cuts it to 300 characters.
func oneLine(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > 300 {
		return string(r[:300]) + "..."
	}
	return s
}
