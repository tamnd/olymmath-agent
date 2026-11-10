package cli

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/locator"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/review"
	"github.com/tamnd/olymmath-agent/seed"
)

func init() {
	register(command{"links", "add a link-only problem for each seed row of a source of zone Z3 that has a position (om links -comp ru-moscow problems-ru), or with -appear an appearance of the problem that has the link already", runLinks})
}

func runLinks(env Env, args []string) int {
	fs := newFlags("links", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	comps := fs.String("comp", "", "a comma-separated list of competition codes; the default is all competitions")
	dry := fs.Bool("dry", false, "print the plan and write no file")
	appear := fs.Bool("appear", false, "give a row whose page the content links at another position one more appearance of that problem, with a merge review of the agent, instead of leaving it out")
	today := fs.String("date", time.Now().Format("2006-01-02"), "the date of the reviews of -appear")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 {
		fmt.Fprintln(env.Stderr, "usage: om links [-comp CODES] [-dry] SOURCE")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om links: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	src := corpus.Source(pos[0])
	if src == nil {
		fmt.Fprintf(env.Stderr, "om links: the register has no source %s\n", pos[0])
		return ExitUsage
	}
	// A source whose texts the content may hold goes through om import, which compares the texts.
	if src.Zone.Statement != "Z3" {
		fmt.Fprintf(env.Stderr, "om links: %s is in zone %s, so use om import\n", src.ID, src.Zone.Statement)
		return ExitUsage
	}
	al, err := seed.LoadAliases(*root, corpus)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om links: %v\n", err)
		return ExitFailure
	}
	var only []string
	if *comps != "" {
		only = strings.Split(*comps, ",")
	}
	owners := official.Owners(corpus)
	// A problem that two papers share, such as a problem of grades 8 and 9, has one page on the source. It gets one record with an appearance in each paper.
	linked := map[string]*content.Problem{}
	for _, p := range corpus.All() {
		for _, st := range p.Statements {
			if st.Source == src.ID && st.URL != "" {
				linked[st.URL] = p
			}
		}
	}
	// adds are the appearances that -appear gives to the problems that link to the page of a row, by the problem.
	adds := map[*content.Problem][]string{}
	byURL := map[string]*journalLink{}
	taken := map[string]bool{}
	byComp := map[string][]*journalLink{}
	skipped := map[string]int{}
	err = seed.Read(*dir, src.ID, func(r seed.Row) error {
		code := al.Code(r.Labels)
		comp := corpus.Competition(code)
		switch {
		case comp == nil:
			skipped["no competition"]++
			return nil
		case len(only) > 0 && !slices.Contains(only, code):
			return nil
		case r.Year == 0 || (r.Edition == "" && seed.Position(r.Number) == "") || r.Number == "" || !strings.HasPrefix(r.Ref, "https://"):
			skipped["no year, position or link"]++
			return nil
		}
		at := fmt.Sprintf("%s/%d/%s", code, r.Year, seed.Position(r.Number))
		// A row that names its edition gives the exact position in it, as tot/2024-fall/ja3.
		if r.Edition != "" {
			if !strings.HasPrefix(r.Edition, strconv.Itoa(r.Year)) || !locator.ValidPosition(r.Number) {
				skipped["not a locator"]++
				return nil
			}
			at = code + "/" + r.Edition + "/" + r.Number
		}
		if _, err := locator.Parse(at); err != nil {
			skipped["not a locator"]++
			return nil
		}
		if owners[at] != nil || taken[at] {
			skipped["position in the content"]++
			return nil
		}
		if p := linked[r.Ref]; p != nil {
			if !*appear || len(p.Appearances) == 0 {
				skipped["problem in the content at another position"]++
				return nil
			}
			taken[at] = true
			adds[p] = append(adds[p], at)
			return nil
		}
		taken[at] = true
		if n := byURL[r.Ref]; n != nil {
			n.also = append(n.also, at)
			return nil
		}
		lang := r.Lang
		if lang == "" {
			lang = "en"
		}
		name := src.Name
		if r.ID != "" {
			name += ", problem " + r.ID
		}
		c := canon.Candidate{Source: src.ID, Origin: src.Origin, Zone: "Z3", Lang: lang, Name: name, URL: r.Ref}
		n := &journalLink{at: at, cand: c}
		byURL[r.Ref] = n
		byComp[code] = append(byComp[code], n)
		return nil
	})
	if err != nil {
		fmt.Fprintf(env.Stderr, "om links: %v\n", err)
		return ExitFailure
	}
	var news []journalLink
	for _, code := range slices.Sorted(maps.Keys(byComp)) {
		for _, n := range byComp[code] {
			news = append(news, *n)
		}
	}
	var reasons []string
	for k, v := range skipped {
		reasons = append(reasons, fmt.Sprintf("%s %d", k, v))
	}
	slices.Sort(reasons)
	added := 0
	for _, ats := range adds {
		added += len(ats)
	}
	fmt.Fprintf(env.Stdout, "%s: %d new link-only problems, %d appearances added to problems in the content. Rows skipped: %s.\n", src.ID, len(news), added, strings.Join(reasons, ", "))
	if *dry {
		for _, n := range news {
			fmt.Fprintf(env.Stdout, "%s\t%s\t%s\n", strings.Join(append([]string{n.at}, n.also...), ","), n.cand.Name, n.cand.URL)
		}
		for _, p := range sortedProblems(adds) {
			fmt.Fprintf(env.Stdout, "%s\tadd to %s\n", strings.Join(adds[p], ","), p.Locator())
		}
		return ExitOK
	}
	if err := writeJournal(*root, corpus, news); err != nil {
		fmt.Fprintf(env.Stderr, "om links: %v\n", err)
		return ExitFailure
	}
	if err := writeAppearances(*root, corpus, src, adds, *today); err != nil {
		fmt.Fprintf(env.Stderr, "om links: %v\n", err)
		return ExitFailure
	}
	return ExitOK
}

// sortedProblems gives the problems of adds in the order of their paths.
func sortedProblems(adds map[*content.Problem][]string) []*content.Problem {
	ps := slices.Collect(maps.Keys(adds))
	slices.SortFunc(ps, func(a, b *content.Problem) int { return strings.Compare(a.Path, b.Path) })
	return ps
}

// writeAppearances gives each problem of adds its new appearances. The source gives one page for the problem at each of them, so the page is one problem and its link is not written twice. Each appearance gets a row in the merges register and a merge review of the agent, because the merge has no text to compare.
func writeAppearances(root string, corpus *content.Corpus, src *content.SourceRecord, adds map[*content.Problem][]string, date string) error {
	var rows []string
	for _, p := range sortedProblems(adds) {
		for _, at := range adds[p] {
			if _, err := newEdition(root, corpus, at); err != nil {
				return fmt.Errorf("%s: %w", at, err)
			}
		}
		full := filepath.Join(root, filepath.FromSlash(p.Path))
		b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
		if err != nil {
			return err
		}
		if b, err = canon.AddAppearances(b, adds[p]...); err != nil {
			return fmt.Errorf("%s: %w", p.Path, err)
		}
		if err := os.WriteFile(full, b, 0o644); err != nil { // #nosec G306 G703 -- content files are public.
			return err
		}
		into := p.Locator()
		page := ""
		for _, st := range p.Statements {
			if st.Source == src.ID && st.URL != "" {
				page = strings.TrimPrefix(st.Name, src.Name+", ")
				break
			}
		}
		for _, at := range adds[p] {
			id := canon.ItemID("merge", into, at)
			rv := &review.Review{ID: id, Kind: "merge", Records: []string{p.Tag}, Decision: "accept", By: "agent", Date: date,
				Note: fmt.Sprintf("Agent decision, not a person's review. %s gives one page, %s, for the problem at %s and at %s, so the two positions are one problem. The record had a link to the page already, so it gets the appearance %s and no second link. There is no text to compare.", src.Name, page, into, at, at)}
			if err := review.Write(root, rv); err != nil {
				return err
			}
			rows = append(rows, strings.Join([]string{at, into, "none", "0.000", "none", "link", id, "0.000"}, "\t"))
		}
	}
	return appendLines(root, review.MergesFile, "# Each appearance that an import merged into a problem of another appearance: the appearance, the primary appearance, the signals K, T and A, the rule and the review. Rule D03 reads this file.\n", rows)
}
