package cli

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
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
	anchor := fs.Bool("anchor", false, "give a position to each row with no number from the official papers of its edition in the raw store, or, when the edition has no official paper in the store, from the texts of zone Z3")
	wikiOrder := fs.Bool("wiki-order", false, "give a position to each amc_aime row of NuminaMath 1.5 from the order of the AoPS wiki pages, in the runs of one year that hold every problem of the year; the year comes from the rows of the other sources with the same text")
	dir2, remote := storeFlags(fs)
	doubts := fs.Bool("doubts", false, "print each doubt with its source and the source of the canonical candidate, to show why a competition is skipped")
	promote := fs.Bool("promote", false, "make a candidate of class H1 or H2 the canonical statement of an existing problem when it ranks above the canonical statement and its text agrees with it")
	explain := fs.String("explain", "", "write each seed row of a known competition to this TSV file with the appearance that it gives, or the reason why it gives none")
	fill := fs.Bool("fill", false, "give the text of the canonical candidate to an existing problem that has no text, such as a problem with only a link of zone Z3, when its appearance has no doubt and no other problem has the text; the other problems do not change")
	existing := fs.Bool("existing", false, "write no new problem: only add statements to the problems of the content")
	maxDoubt := fs.Float64("max-doubt", canon.MaxDoubt, "skip a competition whose review doubts per appearance are above this share")
	matchOn := fs.Bool("match", false, "place the rows of licensed sources that get no position by their content: compare them with the problems.ru texts in the raw store of the problems of -comp that have no text, by their numbers, formulas and letters, and place a row only when it and its problem are the best of each other by a clear margin (package keymatch). The problems.ru texts are only a match key")
	matchOpen := fs.String("match-open", "numina:olympiads", "a comma-separated list of the labels of rows that name no competition and can still be matched")
	matchOut := fs.String("match-out", "", "write the pairs of the content match to this TSV file, with their scores and no text")
	matchLane := fs.String("match-lane", "", "ask this lane, such as codex, whether each doubtful pair of the content match is one problem, one question at a time, and write the answer as a review of kind match")
	matchAsks := fs.Int("match-asks", 20, "ask the lane at most this number of questions")
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
		papers, n, err := official.LoadEditions(context.Background(), st, corpus, s.Comps)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		keys, err := zoneKeys(*dir, sources, corpus, al)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		fmt.Fprintf(env.Stderr, "om import: %d official papers of %d editions, %d editions with texts of zone Z3.\n", n, len(papers), len(keys))
		s.Locate = func(code string, year int, text string) string {
			ed := code + "/" + strconv.Itoa(year)
			if _, ok := papers[ed]; ok {
				return papers.Locate(ed, text)
			}
			return keys.Locate(ed, text)
		}
		s.Confirm = func(code string, year int, text, pos string) bool {
			return !papers.Disagrees(code+"/"+strconv.Itoa(year), text, pos)
		}
	}
	if slices.Contains(sources, "hf-aops-wiki") {
		s.Topics = &canon.Topics{Aliases: al, Corpus: corpus}
		if err := seed.Read(*dir, "hf-aops-wiki", func(r seed.Row) error { s.Topics.Add(r); return nil }); err != nil {
			fmt.Fprintf(env.Stderr, "om import: hf-aops-wiki: %v\n", err)
			return ExitFailure
		}
	}
	if *matchOn && *dry && *matchLane != "" {
		fmt.Fprintln(env.Stderr, "om import: -match-lane writes the answers as reviews, so it does not run with -dry")
		return ExitUsage
	}
	if *matchOn {
		st, err := openStore(*dir2, *remote)
		if err == nil {
			o := matchOptions{open: strings.Split(*matchOpen, ","), out: *matchOut, lane: *matchLane, asks: *matchAsks, root: *root, dir: *dir, today: time.Now().Format(time.DateOnly)}
			s.Matched, err = contentMatch(context.Background(), env, s, st, sources, o)
		}
		if err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
	}
	var ordered map[string]seed.Row
	if *wikiOrder {
		var anchored int
		if ordered, anchored, err = wikiPlaces(*dir, sources, corpus, al); err != nil {
			fmt.Fprintf(env.Stderr, "om import: %v\n", err)
			return ExitFailure
		}
		fmt.Fprintf(env.Stderr, "om import: %d rows got a position from the order of the AoPS wiki, %d of them between two anchors.\n", len(ordered), anchored)
	}
	for _, src := range sources {
		if err := seed.Read(*dir, src, func(r seed.Row) error {
			if o, ok := ordered[canon.MatchRef(r)]; ok {
				r = o
			}
			s.Add(r)
			return nil
		}); err != nil {
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
	// texts holds the canonical text of each problem, with its first appearance, as rule D01 reads them.
	texts := match.NewIndex()
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			owner[a.At] = p
		}
		if len(p.Appearances) > 0 && p.Statement != "" {
			texts.Add(p.Appearances[0].At, match.Shingles(p.Statement))
		}
	}
	var created, updated, merged, promoted, dupes, fills int
	// filled holds the problems that got a text from -fill, so that a problem with two appearances gets one text.
	filled := map[string]bool{}
	var mergeRows []string
	var newRecs []*canon.Record
	for _, r := range recs {
		if p := owner[r.Primary.At]; p != nil {
			comp := corpus.Competition(strings.SplitN(r.Primary.At, "/", 2)[0])
			add := canon.Add(p, r, original(comp))
			best := r.Primary.Best
			fillID := add.ID(best)
			if fillID == "" {
				// An earlier import can have added the link of the row already, as a statement with no text. The text fills that statement.
				fillID = canon.LinkID(p, best)
			}
			give := *fill && p.Statement == "" && !filled[p.Tag] && len(r.Primary.Doubts) == 0 && !best.Reworded && best.Text != "" && best.Zone != "Z3" && fillID != ""
			if give {
				// A text that another problem has already is not given, because rule D01 does not allow two problems with one text. A review item names the pair.
				if h, ok := duplicateOf(texts, best.Shingles()); ok {
					items = append(items, canon.Item{ID: canon.ItemID("pair", h.ID, r.Primary.At), Kind: "pair", At: h.ID, With: r.Primary.At, T: math.Round(h.T*1000) / 1000, Text: owner[h.ID].Statement, Other: best.Text})
					dupes++
					give = false
				}
			}
			if *fill && !give {
				// A fill pass changes only the problems that get a text.
				continue
			}
			if give {
				filled[p.Tag] = true
				fills++
				texts.Add(p.Appearances[0].At, best.Shingles())
			}
			up := !give && add.Better != nil && *promote && canon.Agrees(add.Better, p)
			if add.Better != nil && !up {
				items = append(items, canon.Item{ID: "better-" + p.Tag, Kind: "canonical", At: r.Primary.At, Source: add.Better.Source, Ref: add.Better.Ref, Text: p.Statement, Other: add.Better.Text})
			}
			if len(add.Statements) == 0 && !give {
				continue
			}
			if len(add.Statements) > 0 {
				updated++
			}
			if *dry {
				continue
			}
			full := filepath.Join(*root, filepath.FromSlash(p.Path))
			b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
			if err == nil {
				b, err = canon.Apply(b, add)
			}
			if err == nil && give {
				b, err = canon.Fill(b, fillID, best.Text, r.Primary.Rule, r.Primary.Status())
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
		if *existing {
			continue
		}
		// An appearance that already belongs to a problem does not merge into a new problem.
		r.Merged = slices.DeleteFunc(r.Merged, func(m canon.Merge) bool { return owner[m.At] != nil })
		// A new problem with the text of a problem in the content is not written, because rule D01 does not allow two problems with one text. One of the two appearances is wrong, so a review item names the pair.
		if h, ok := duplicateOf(texts, r.Primary.Best.Shingles()); ok {
			items = append(items, canon.Item{ID: canon.ItemID("pair", h.ID, r.Primary.At), Kind: "pair", At: h.ID, With: r.Primary.At, T: math.Round(h.T*1000) / 1000, Text: owner[h.ID].Statement, Other: r.Primary.Best.Text})
			dupes++
			continue
		}
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
	if *matchOn {
		fmt.Fprintf(env.Stdout, "%d rows got a position from the content match.\n", s.Matches)
	}
	if s.Led > 0 {
		fmt.Fprintf(env.Stdout, "%d rows got a position from the problem mark at the start of their text.\n", s.Led)
	}
	fmt.Fprintf(env.Stdout, "%d appearances with a candidate. %d new problems, %d merged appearances, %d problems with new links, %d promoted statements, %d problems that got a text, %d texts not written because they are in the content, %d review items. Rows skipped: %s.\n", len(choices), created, merged, updated, promoted, fills, dupes, len(items), strings.Join(reasons, ", "))
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

// The amc_aime rows of NuminaMath 1.5 keep the order of the AoPS wiki pages: the AHSME, the AIME, the AMC 10, the AMC 12 and the AMC 8 (with the AJHSME), each by year, by paper and by page title.
const (
	orderedSource = "hf-numinamath-1-5"
	orderedLabel  = "numina:amc_aime"
)

var orderedSections = []string{"ahsme", "aime", "amc10", "amc12", "amc8"}

// wikiPlaces reads the rows of the sources and gives the amc_aime rows that canon.Order places, with the label, the year and the number of their place, by canon.MatchRef. The other rows of the competitions of the wiki are the reference rows, and the rows among them that have a number are anchors. A row of a paper of the fall, such as the AMC 10 of fall 2021, is a reference row of its own part of the year, and it gets the papers of that part in the register.
func wikiPlaces(dir string, sources []string, corpus *content.Corpus, al *seed.Aliases) (out map[string]seed.Row, anchored int, err error) {
	o := &canon.Order{
		Sections: orderedSections,
		Shared:   map[string]string{"amc12": "amc10"},
		Papers: func(code string, year int, part string) ([]string, int) {
			c := corpus.Competition(code)
			if c == nil {
				return nil, 0
			}
			ps := c.Shape(year).Positions()
			for i, p := range ps {
				if p != "p"+strconv.Itoa(i+1) {
					return nil, 0
				}
			}
			return c.PartLetters(year, part), len(ps)
		},
	}
	var topics *canon.Topics
	if slices.Contains(sources, "hf-aops-wiki") && slices.Contains(sources, "hf-ai-mo-aops") {
		topics = &canon.Topics{Aliases: al, Corpus: corpus}
		if err := seed.Read(dir, "hf-aops-wiki", func(r seed.Row) error { topics.Add(r); return nil }); err != nil {
			return nil, 0, fmt.Errorf("hf-aops-wiki: %w", err)
		}
	}
	var rows []seed.Row
	for _, src := range sources {
		err := seed.Read(dir, src, func(r seed.Row) error {
			if r.Source == orderedSource && slices.Contains(r.Labels, orderedLabel) {
				rows = append(rows, r)
				return nil
			}
			part := ""
			if fallRE.MatchString(strings.Join(r.Labels, " ")) {
				part = "fall"
			}
			code := al.Code(r.Labels)
			o.Ref(code, r.Year, part, r.Statement)
			for _, a := range wikiAnchors(r, code, topics) {
				o.Anchor(a.code, a.year, part, a.letter, a.number, r.Statement)
			}
			return nil
		})
		if err != nil {
			return nil, 0, fmt.Errorf("%s: %w", src, err)
		}
	}
	out = map[string]seed.Row{}
	for i, p := range o.Place(rows) {
		r := rows[i]
		// Some rows hold the table of contents of the wiki page and not the problem. Such a row keeps its slot in the order, but it gets no place.
		if wikiContentsRE.MatchString(r.Statement) {
			continue
		}
		label := p.Code
		if p.Letter != "" {
			label += " " + strings.ToUpper(p.Letter)
		}
		labels := []string{label}
		if p.Part != "" {
			// The season goes in a label of its own, so that canon.Paper finds the paper of the part.
			labels = append(labels, p.Part)
		}
		r.Labels = append(labels, r.Labels...)
		r.Year, r.Number = p.Year, strconv.Itoa(p.Number)
		r.Answer = wikiAnswer(r.Answer)
		if p.Anchored {
			anchored++
			// Agent decision: an anchored row sits next to the numbered row that anchors it, which gives the choice letter. A value as the answer, such as 66, cannot match that letter, so the row keeps only a choice letter.
			if !choiceLetterRE.MatchString(r.Answer) {
				r.Answer = ""
			}
		}
		out[canon.MatchRef(r)] = r
	}
	return out, anchored, nil
}

// wikiAnchor is the number of a reference row of the wiki order.
type wikiAnchor struct {
	code   string
	year   int
	letter string
	number int
}

// wikiTitleRE reads one appearance of a title of the AoPS wiki, such as "2022 AMC 10A #1" in "2022 AMC 10A #1 / 2022 AMC 12A #1".
var wikiTitleRE = regexp.MustCompile(`^\s*((?:19|20)[0-9]{2}) (AHSME|AJHSME|AMC 8|AMC 10|AMC 12)\s*([AB])?\s*#([0-9]{1,2})\s*$`)

// wikiAnchors gives the numbers that a reference row gives itself: the number column of a row of a competition and a year, as in the AMC 12 dataset; each appearance in the title of a row of the AoPS wiki; and the position of the topic of a row of the AoPS dataset that the wiki numbers. A row of a paper with no letter in a year with two papers gives no number.
func wikiAnchors(r seed.Row, code string, topics *canon.Topics) []wikiAnchor {
	var out []wikiAnchor
	if r.Source == "hf-aops-wiki" {
		for _, t := range strings.Split(r.Title, "/") {
			m := wikiTitleRE.FindStringSubmatch(t)
			if m == nil {
				continue
			}
			y, _ := strconv.Atoi(m[1])
			n, _ := strconv.Atoi(m[4])
			c := map[string]string{"AHSME": "ahsme", "AJHSME": "amc8", "AMC 8": "amc8", "AMC 10": "amc10", "AMC 12": "amc12"}[m[2]]
			out = append(out, wikiAnchor{c, y, strings.ToLower(m[3]), n})
		}
		return out
	}
	if code == "" || code == "-" || r.Year == 0 {
		return nil
	}
	pos := seed.Position(r.Number)
	letter := canon.Letter(r.Labels)
	if pos == "" && topics != nil && r.Source == "hf-ai-mo-aops" {
		pos, letter = topics.Position(r.Ref, code, r.Year)
	}
	n, err := strconv.Atoi(strings.TrimPrefix(pos, "p"))
	if pos == "" || err != nil {
		return nil
	}
	return []wikiAnchor{{code, r.Year, letter, n}}
}

// wikiAnswer cleans the answer of an amc_aime row. These answers lost the spaces and often the letters of the variables, so that "a=3x or a=-2x" became "=3xor=-2x". The answer keeps the choice letter when it starts with one, and else it is kept only when it has no letter outside TeX commands and no sign of a lost letter, such as empty braces.
func wikiAnswer(a string) string {
	a = strings.TrimSpace(a)
	if m := choiceRE.FindStringSubmatch(a); m != nil {
		return m[1]
	}
	if a == "" || lostLetterRE.MatchString(a) || strings.ContainsAny(texCommandRE.ReplaceAllString(a, ""), "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ") {
		return ""
	}
	return a
}

var (
	choiceRE       = regexp.MustCompile(`^(?:\\textbf\{)?\(?([A-E])\)?(?:\}|$)`)
	texCommandRE   = regexp.MustCompile(`\\[a-zA-Z]+`)
	lostLetterRE   = regexp.MustCompile(`\{\}|\{[-+^_=]|\^\{\}|^[=+*/^_]|[-+*/^_=]$|\\\\`)
	choiceLetterRE = regexp.MustCompile(`^[A-E]$`)
	fallRE         = regexp.MustCompile(`(?i)\bfall\b`)
	wikiContentsRE = regexp.MustCompile(`^\s*\[1 [^\]]*\]\(https://artofproblemsolving\.com#`)
)

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
	// An item can come twice: from this import and from the old queue, when this import read the competition of its pair but not the competition of its appearance. Only the first one is kept.
	seen := map[string]bool{}
	for _, it := range items {
		if seen[it.ID] {
			continue
		}
		seen[it.ID] = true
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

// duplicateOf gives the text of the index that is most like the text with shingles sh, when its signal T is at or above match.Duplicate, the bound of rule D01. The result ok is false when the index has no such text.
func duplicateOf(texts *match.Index, sh []uint64) (h match.Hit, ok bool) {
	hits := texts.Near(sh, match.Duplicate)
	if len(hits) == 0 {
		return match.Hit{}, false
	}
	return slices.MaxFunc(hits, func(a, b match.Hit) int { return cmp.Or(cmp.Compare(a.T, b.T), strings.Compare(b.ID, a.ID)) }), true
}
