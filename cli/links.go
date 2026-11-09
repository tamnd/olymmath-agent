package cli

import (
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/locator"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/seed"
)

func init() {
	register(command{"links", "add a link-only problem for each seed row of a source of zone Z3 that has a position (om links -comp ru-moscow problems-ru)", runLinks})
}

func runLinks(env Env, args []string) int {
	fs := newFlags("links", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	comps := fs.String("comp", "", "a comma-separated list of competition codes; the default is all competitions")
	dry := fs.Bool("dry", false, "print the plan and write no file")
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
	linked := map[string]bool{}
	for _, p := range corpus.All() {
		for _, st := range p.Statements {
			if st.Source == src.ID && st.URL != "" {
				linked[st.URL] = true
			}
		}
	}
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
		if linked[r.Ref] {
			skipped["problem in the content at another position"]++
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
	fmt.Fprintf(env.Stdout, "%s: %d new link-only problems. Rows skipped: %s.\n", src.ID, len(news), strings.Join(reasons, ", "))
	if *dry {
		for _, n := range news {
			fmt.Fprintf(env.Stdout, "%s\t%s\t%s\n", strings.Join(append([]string{n.at}, n.also...), ","), n.cand.Name, n.cand.URL)
		}
		return ExitOK
	}
	if err := writeJournal(*root, corpus, news); err != nil {
		fmt.Fprintf(env.Stderr, "om links: %v\n", err)
		return ExitFailure
	}
	return ExitOK
}
