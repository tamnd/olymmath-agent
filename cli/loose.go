package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/loose"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/site"
	"github.com/tamnd/olymmath-agent/tag"
)

// LooseSources are the seed sources of om import -loose, in the order of the import (spec 06.8).
var LooseSources = []string{"hf-ai-mo-aops", "hf-numina-olympiads", "hf-numinamath-1-5", "hf-mathnet"}

// LinkOnly are the competitions whose texts the project only links (document 11). The loose import takes no row of them, and adds no row to a problem of them.
var LinkOnly = []string{"au-amo", "in-rmo", "in-inmo", "uk-bmo1", "uk-bmo2", "kr-kmo", "kr-fkmo", "de-mo", "ca-cmo", "ca-cjmo", "bg-mo"}

// NuminaLabels are the labels of the olympiad rows of hf-numinamath-1-5. Its other rows come from school sets and synthetic sets.
var NuminaLabels = []string{"numina:olympiads", "numina:aops_forum", "numina:amc_aime"}

// HeldFile lists the rows that the loose import held for a review.
const HeldFile = "reports/review/loose-held.tsv"

// MaxAnswer is the longest answer of a row, in bytes, that a loose problem keeps. A longer answer is prose, not a value.
const MaxAnswer = 100

// looseRow is a row that the loose import may take.
type looseRow struct {
	row      seed.Row
	cand     *canon.Candidate
	unplaced []content.Unplaced
}

func runImportLoose(env Env, args []string) int {
	fs := newFlags("import -loose", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	sources := fs.String("sources", strings.Join(LooseSources, ","), "a comma-separated list of the seed sources to read, in order")
	located := fs.Bool("located", false, "take only the rows whose labels name a competition of the register")
	labels := fs.String("labels", strings.Join(NuminaLabels, ","), "a comma-separated list of the labels of the hf-numinamath-1-5 rows to take")
	block := fs.String("block", strings.Join(LinkOnly, ","), "a comma-separated list of the competitions whose rows are left out")
	limit := fs.Int("limit", 0, "stop after this number of new problems; 0 is no limit")
	neighbors := fs.Int("neighbors", 8, "compare each row with this number of its nearest texts")
	dry := fs.Bool("dry", false, "print the counts and write no file")
	explain := fs.String("explain", "", "write the decision for each row to this TSV file, with the text")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	fail := func(err error) int {
		fmt.Fprintf(env.Stderr, "om import -loose: %v\n", err)
		return ExitFailure
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om import -loose: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	al, err := seed.LoadAliases(*root, corpus)
	if err != nil {
		return fail(err)
	}
	if _, err := site.NewMarkup(nil); err != nil {
		return fail(err)
	}
	newAccept := func() func(string) bool {
		mk, _ := site.NewMarkup(nil)
		return func(t string) bool {
			_, errs := mk.HTML(t)
			for _, e := range errs {
				if !errors.Is(e, site.ErrFigure) {
					return false
				}
			}
			return true
		}
	}
	r := &looseReader{corpus: corpus, al: al, located: *located, labels: strings.Split(*labels, ","), block: strings.Split(*block, ","), thisYear: time.Now().Year(), dropped: map[string]int{}}
	r.used = map[string]bool{}
	if r.removed, err = loadRemoved(*root); err != nil {
		return fail(err)
	}
	for _, p := range corpus.All() {
		for _, s := range p.Statements {
			if s.Capture != "" {
				r.used[s.Capture] = true
			}
		}
	}
	for _, src := range strings.Split(*sources, ",") {
		if err := seed.Read(*dir, src, r.add); err != nil {
			return fail(fmt.Errorf("%s: %w", src, err))
		}
	}
	r.finish(newAccept)
	// The docs of the problems are their texts: the canonical text and the other texts, each owned by its problem.
	var docs []loose.Doc
	var owners []*content.Problem
	for _, p := range corpus.All() {
		var eds []string
		for _, a := range p.Appearances {
			eds = append(eds, a.Locator.EditionOf().String())
		}
		var named []string
		for _, u := range p.Unplaced {
			if strings.Contains(u.At, "/") {
				named = append(named, u.At)
			}
		}
		for _, s := range p.Statements {
			if s.Text != "" {
				docs = append(docs, loose.Doc{Text: s.Text, Answer: p.Answer, Editions: eds, Named: named, ID: p.Tag, Derived: len(p.Appearances) == 0 && s.Origin == "D", Capture: s.Capture})
				owners = append(owners, p)
			}
		}
	}
	rows := make([]loose.Doc, len(r.rows))
	for i, x := range r.rows {
		rows[i] = loose.Doc{Text: x.cand.Text, Answer: x.cand.Answer, Capture: x.cand.Capture}
		for _, u := range x.unplaced {
			if strings.Contains(u.At, "/") {
				rows[i].Editions = append(rows[i].Editions, u.At)
			}
		}
	}
	fmt.Fprintf(env.Stderr, "om import -loose: %d texts of %d problems, %d rows to match.\n", len(docs), len(corpus.All()), len(rows))
	ds := loose.Decide(docs, rows, loose.Options{Limit: *limit, Neighbors: *neighbors, Complete: completeEditions(corpus)})

	// groups holds the rows of each new problem, by the index of its first row. links holds the rows of each problem of the content.
	groups := map[int][]int{}
	var heads []int
	links := map[*content.Problem][]int{}
	var linked []*content.Problem
	counts := map[loose.Kind]int{}
	var held []string
	for i, d := range ds {
		counts[d.Kind]++
		switch d.Kind {
		case loose.New:
			heads = append(heads, i)
			groups[i] = []int{i}
		case loose.Join:
			groups[d.Target] = append(groups[d.Target], i)
		case loose.Link:
			p := owners[d.Target]
			if inBlock(p, r.block) {
				// The import adds nothing to a problem of a link-only competition, not even a second statement.
				r.dropped["same as a problem of a link-only competition"]++
				r.droppedN++
				counts[d.Kind]--
				continue
			}
			if links[p] == nil {
				linked = append(linked, p)
			}
			links[p] = append(links[p], i)
		case loose.Hold:
			with := ""
			if d.Target >= 0 {
				with = owners[d.Target].Tag
			} else {
				with = r.rows[-1-d.Target].cand.Capture
			}
			held = append(held, fmt.Sprintf("%s\t%s\t%.3f\t%.3f\t%.3f\t%s\t%s", r.rows[i].cand.Capture, with, d.E, d.T, d.F, answerWord(d), d.Reason))
		}
	}
	if *explain != "" {
		if err := writeLooseExplain(*explain, r.rows, ds, owners); err != nil {
			return fail(err)
		}
	}
	// adds gives the statements and the unplaced entries that each problem of the content gets: one statement for each source that it does not have.
	type addition struct {
		p        *content.Problem
		cands    []*canon.Candidate
		unplaced []content.Unplaced
	}
	var adds []addition
	for _, p := range linked {
		have := map[string]bool{}
		for _, s := range p.Statements {
			have[s.Source] = true
		}
		a := addition{p: p}
		for _, i := range links[p] {
			x := r.rows[i]
			if !have[x.cand.Source] {
				have[x.cand.Source] = true
				a.cands = append(a.cands, x.cand)
			}
			if p.Primary() == nil {
				a.unplaced = append(a.unplaced, x.unplaced...)
			}
		}
		if len(a.cands) > 0 || len(a.unplaced) > 0 {
			adds = append(adds, a)
		}
	}
	statements := 0
	for _, a := range adds {
		statements += len(a.cands)
	}
	fmt.Fprintf(env.Stderr, "om import -loose: %d rows read, %d already imported, %d dropped.\n", r.read, r.already, r.droppedN)
	keys := make([]string, 0, len(r.dropped))
	for k := range r.dropped {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return r.dropped[keys[i]] > r.dropped[keys[j]] })
	for _, k := range keys {
		fmt.Fprintf(env.Stderr, "  dropped, %s: %d\n", k, r.dropped[k])
	}
	fmt.Fprintf(env.Stderr, "om import -loose: %d new problems from %d rows, %d rows link %d problems of the content (%d new statements), %d rows held, %d rows after the limit.\n",
		len(heads), counts[loose.New]+counts[loose.Join], counts[loose.Link], len(adds), statements, counts[loose.Hold], counts[loose.Skip])
	if *dry {
		return ExitOK
	}
	tags, err := mintTags(corpus, len(heads))
	if err != nil {
		return fail(err)
	}
	var tagLines []string
	for k, h := range heads {
		var cands []*canon.Candidate
		var us []content.Unplaced
		for _, i := range groups[h] {
			cands = append(cands, r.rows[i].cand)
			us = mergeUnplaced(us, r.rows[i].unplaced)
		}
		first := r.rows[h].cand
		b, err := canon.RenderLoose(tags[k], canon.TypeOf("", first.Answer), first.Answer, cands, us)
		if err != nil {
			return fail(fmt.Errorf("%s: %w", first.Capture, err))
		}
		full := filepath.Join(*root, filepath.FromSlash(tag.LoosePath(tags[k])))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
			return fail(err)
		}
		if err := os.WriteFile(full, b, 0o644); err != nil { // #nosec G306 G703 -- content files are public.
			return fail(err)
		}
		tagLines = append(tagLines, tags[k]+"\tproblem")
	}
	if err := appendLines(*root, "registers/tags.tsv", "", tagLines); err != nil {
		return fail(err)
	}
	for _, a := range adds {
		full := filepath.Join(*root, filepath.FromSlash(a.p.Path))
		b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
		if err != nil {
			return fail(err)
		}
		var add canon.Additions
		n := len(a.p.Statements)
		for _, c := range a.cands {
			n++
			add = add.With(n, c)
		}
		if b, err = canon.Apply(b, add); err == nil {
			b, err = canon.AddUnplaced(b, a.unplaced)
		}
		if err == nil {
			err = os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
		}
		if err != nil {
			return fail(fmt.Errorf("%s: %w", a.p.Path, err))
		}
	}
	if err := writeHeld(*root, held); err != nil {
		return fail(err)
	}
	return ExitOK
}

func answerWord(d loose.Decision) string {
	return [...]string{"none", "match", "mismatch"}[d.A]
}

// mergeUnplaced adds the entries of b to a, with no entry twice.
func mergeUnplaced(a, b []content.Unplaced) []content.Unplaced {
	for _, u := range b {
		if !slices.ContainsFunc(a, func(x content.Unplaced) bool { return x.At == u.At && x.Source == u.Source }) {
			a = append(a, u)
		}
	}
	return a
}

// writeHeld writes the list of the held rows. The import writes it again each time, with the rows that it held in this run.
func writeHeld(root string, lines []string) error {
	full := filepath.Join(root, filepath.FromSlash(HeldFile))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
		return err
	}
	head := "# The seed rows that om import -loose held for a review: the row, the problem or the row that it is near, the signals E, T and the keyword match, the answer signal and the reason (spec 06.8).\n"
	return os.WriteFile(full, []byte(head+strings.Join(lines, "\n")+"\n"), 0o644) // #nosec G306 -- the list is public.
}

// completeEditions gives the editions that list their papers and have a problem of the content at each position.
func completeEditions(corpus *content.Corpus) map[string]bool {
	out := map[string]bool{}
	for _, e := range corpus.Editions() {
		want := e.Positions()
		if len(want) == 0 {
			continue
		}
		have := map[string]bool{}
		for _, p := range append(slices.Clone(e.Problems), e.Also...) {
			have[p.PositionIn(e)] = true
		}
		if !slices.ContainsFunc(want, func(pos string) bool { return !have[pos] }) {
			out[e.Locator()] = true
		}
	}
	return out
}

func writeLooseExplain(path string, rows []looseRow, ds []loose.Decision, owners []*content.Problem) error {
	f, err := os.Create(path) // #nosec G304 -- a path that the user gives.
	if err != nil {
		return err
	}
	w := io.Writer(f)
	for i, d := range ds {
		with := ""
		switch {
		case d.Kind == loose.Link || (d.Kind == loose.Hold && d.Target >= 0):
			with = owners[d.Target].Tag
		case d.Kind == loose.Join:
			with = rows[d.Target].cand.Capture
		case d.Kind == loose.Hold:
			with = rows[-1-d.Target].cand.Capture
		}
		x := rows[i]
		fmt.Fprintf(w, "%s\t%s\t%s\t%.3f\t%.3f\t%.3f\t%v\t%s\t%s\n", d.Kind, x.cand.Capture, with, d.E, d.T, d.F, d.Exact, d.Reason, strings.ReplaceAll(x.cand.Text, "\n", "\\n"))
	}
	return f.Close()
}

// polishOlympiad is the code of the Polish olympiad, whose problems some rows mark in their text (loose.PolishMark).
const polishOlympiad = "pl-om"

// looseReader reads the seed rows for om import -loose.
type looseReader struct {
	corpus   *content.Corpus
	al       *seed.Aliases
	located  bool
	labels   []string
	block    []string
	thisYear int
	used     map[string]bool
	// removed are the rows that a review took out of the content (registers/loose-removed.tsv).
	removed map[string]bool

	pending                 []pendingLoose
	rows                    []looseRow
	read, already, droppedN int
	dropped                 map[string]int
}

// RemovedFile lists the rows that a review took out of the content: the capture and the reason, separated by a tab. A loose problem that is removed leaves no capture, so without this file the next batch would bring its row back.
const RemovedFile = "registers/loose-removed.tsv"

// loadRemoved reads the captures of RemovedFile. A missing file is an empty list.
func loadRemoved(root string) (map[string]bool, error) {
	b, err := os.ReadFile(filepath.Join(root, RemovedFile)) // #nosec G304 -- a register of the content root that the user names.
	if errors.Is(err, os.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for i, line := range strings.Split(string(b), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		capture, reason, ok := strings.Cut(line, "\t")
		if !ok || !strings.Contains(capture, "#row=") || strings.TrimSpace(reason) == "" {
			return nil, fmt.Errorf("%s:%d: want a capture, a tab and a reason", RemovedFile, i+1)
		}
		out[capture] = true
	}
	return out, nil
}

func (r *looseReader) drop(reason string) error {
	r.dropped[reason]++
	r.droppedN++
	return nil
}

// add reads one row. It keeps the row when its source is usable, its competition is not link-only, it has no position that the import by position can use, and its text is one clear problem.
func (r *looseReader) add(row seed.Row) error {
	r.read++
	capture := row.Source + "/" + row.File + "#row=" + strconv.Itoa(row.Row)
	if r.used[capture] {
		r.already++
		return nil
	}
	if r.removed[capture] {
		return r.drop("removed by a review")
	}
	src := r.corpus.Source(row.Source)
	if src == nil || src.Origin == "M" || src.DoNotRepublish || (src.Zone.Statement != "Z1" && src.Zone.Statement != "Z2") {
		return r.drop("source not usable")
	}
	if a := r.corpus.Source(row.Author); a != nil && a.Zone.Statement == "Z3" {
		return r.drop("text in zone Z3")
	}
	if row.Source == "hf-numinamath-1-5" && !slices.ContainsFunc(row.Labels, func(l string) bool { return slices.Contains(r.labels, l) }) {
		return r.drop("not an olympiad set")
	}
	code := r.al.Code(row.Labels)
	if slices.Contains(r.block, code) {
		return r.drop("link-only competition")
	}
	comp := r.corpus.Competition(code)
	if r.located && comp == nil {
		return r.drop("no competition")
	}
	if comp != nil && row.Number != "" && row.Year > 0 {
		pos := seed.Position(row.Number)
		ps := comp.Shape(row.Year).Positions()
		if pos != "" && (len(ps) == 0 || slices.Contains(ps, pos)) {
			return r.drop("has a position")
		}
	}
	r.pending = append(r.pending, pendingLoose{row: row, comp: comp, capture: capture, src: src})
	return nil
}

// pendingLoose is a row that passed the checks of its metadata. finish checks its text.
type pendingLoose struct {
	row     seed.Row
	comp    *content.Competition
	src     *content.SourceRecord
	capture string
}

// finish checks the texts of the pending rows on all the CPUs, and keeps the rows in their order. Each worker has its own markup, because a markup is not safe for two workers at once.
func (r *looseReader) finish(newAccept func() func(string) bool) {
	out := make([]*looseRow, len(r.pending))
	why := make([]string, len(r.pending))
	var next atomic.Int64
	var wg sync.WaitGroup
	for range max(runtime.NumCPU(), 1) {
		accept := newAccept()
		wg.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= len(r.pending) {
					return
				}
				out[i], why[i] = r.check(r.pending[i], accept)
			}
		})
	}
	wg.Wait()
	for i, x := range out {
		if x == nil {
			_ = r.drop(why[i])
			continue
		}
		r.rows = append(r.rows, *x)
	}
	r.pending = nil
}

// check reads the text of a pending row. It gives the row, or nil and the reason why the text is not one clear problem.
func (r *looseReader) check(pr pendingLoose, accept func(string) bool) (*looseRow, string) {
	row, comp, src, capture := pr.row, pr.comp, pr.src, pr.capture
	text, why := loose.Clean(row.Statement)
	if why != "" {
		return nil, why
	}
	mark := ""
	if m, rest, ok := loose.PolishMark(text); ok {
		// The text names its olympiad, round and number. The content has the final, round III, by position, so a problem of the final is left to the import by position. A problem of round I or II has no position, and the mark gives its competition and year.
		pl := r.corpus.Competition(polishOlympiad)
		if pl == nil {
			return nil, "no competition"
		}
		if m.Round == "III" && slices.Contains(pl.Shape(m.Year()).Positions(), "p"+strconv.Itoa(m.Number)) {
			return nil, "has a position"
		}
		if utf8.RuneCountInString(rest) < canon.MinText {
			return nil, loose.TooShort
		}
		text, comp, mark = rest, pl, m.Label
		row.Year = m.Year()
	}
	lang := loose.Lang(text, row.Lang)
	if lang == "" {
		return nil, loose.NoLang
	}
	if !accept(text) {
		return nil, loose.BrokenTeX
	}
	answer := canon.Answer(row.Answer)
	if len(answer) > MaxAnswer || strings.Contains(answer, "\n") {
		answer = ""
	}
	url := src.URL
	if t, ok := strings.CutPrefix(row.Ref, "aops-topic:"); ok && t != "" {
		url = "https://artofproblemsolving.com/community/c6h" + t
	} else if strings.HasPrefix(row.Ref, "https://") {
		url = row.Ref
	}
	name := ""
	label := ""
	if len(row.Labels) > 0 && !strings.HasPrefix(row.Labels[len(row.Labels)-1], "numina:") {
		label = row.Labels[len(row.Labels)-1]
		if row.Source == "hf-ai-mo-aops" {
			name = label
		}
	}
	if mark != "" {
		label = mark
	}
	x := looseRow{row: row, cand: &canon.Candidate{
		Source: row.Source, Origin: src.Origin, Zone: src.Zone.Statement, Lang: lang, Name: name, URL: url,
		Ref: row.File + "#" + strconv.Itoa(row.Row), Capture: capture, Text: text, Answer: answer, Reworded: src.Reworded,
	}}
	if comp != nil && !wrongList(comp, label) {
		at := comp.Code
		if (comp.EditionForm == "" || comp.EditionForm == "year") && row.Year > 0 && row.Year >= comp.FirstYear && row.Year <= r.thisYear && !yearRangeRE.MatchString(label) {
			at += "/" + strconv.Itoa(row.Year)
		}
		x.unplaced = []content.Unplaced{{At: at, Source: row.Source, Label: label}}
	}
	return &x, ""
}

// yearRangeRE finds a range of years in a label, as in "Shortlist 1992 - 2010". The year of such a row is the first year of the range, not the year of the problem.
var yearRangeRE = regexp.MustCompile(`\b(?:19|20)[0-9]{2}\s*[-–]\s*(?:(?:19|20)[0-9]{2}\b|$)|\b(?:19|20)[0-9]{2}\+`)

// wrongList tells if the label names a shortlist or a longlist and the competition is not one. The problems of the Iberoamerican shortlist were not set at the Iberoamerican olympiad, for example, so the row says nothing about where the problem was set.
func wrongList(comp *content.Competition, label string) bool {
	l := strings.ToLower(label)
	if !strings.Contains(l, "shortlist") && !strings.Contains(l, "longlist") {
		return false
	}
	n := strings.ToLower(comp.Name)
	return !strings.Contains(n, "shortlist") && !strings.Contains(n, "longlist")
}

// inBlock reports whether a problem appears in one of the link-only competitions.
func inBlock(p *content.Problem, block []string) bool {
	if slices.Contains(block, p.Competition) && p.Competition != "" {
		return true
	}
	return slices.ContainsFunc(p.Appearances, func(a content.Appearance) bool {
		return a.Locator.Comp != "" && slices.Contains(block, a.Locator.Comp)
	})
}
