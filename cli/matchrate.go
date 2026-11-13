package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/similar"
	"github.com/tamnd/olymmath-agent/site"
)

func init() {
	register(command{"matchrate", "write the match rate report: for each dataset, the seed rows that match a problem record by K, E and A and the rows that can be new problems (om matchrate [-derived] -seed DIR)", runMatchrate})
}

// MatchRate is the match rate of the seed rows of one dataset.
type MatchRate struct {
	Source string `json:"source"`
	// Rows are the rows with a statement.
	Rows int `json:"rows"`
	// Placed is the match against the records that are placed in a competition.
	Placed Against `json:"placed"`
	// All is the match against all the records, the placed ones and the loose ones under problems/_loose.
	All Against `json:"all"`
}

// Against counts the rows of a dataset against one set of records.
type Against struct {
	// Matched are the rows whose appearance has a record, or whose capture is a statement of a record, or whose nearest record has E of at least match.EStrict.
	Matched int `json:"matched"`
	// ByKey are the matched rows whose appearance has a record.
	ByKey int `json:"by_key"`
	// ByLink are the matched rows whose capture is a statement of a record: an import took the row into that record.
	ByLink int `json:"by_link"`
	// Variant are the rows whose nearest record has E of at least match.EStrict but another answer (signal A). They can be variants, so a person must look at them.
	Variant int `json:"variant"`
	// Near are the rows whose nearest record has E from DupLeast to match.EStrict. A person must look at them to tell.
	Near int `json:"near"`
	// New are the rows with no record near them.
	New int `json:"new"`
	// NewProblems are the new rows counted once for each group of new rows of the dataset that are one problem by E. A dataset often has the same problem in several rows.
	NewProblems int `json:"new_problems"`
	// NewLocated are the new rows whose labels name a competition of the register. They can become problem records.
	NewLocated int `json:"new_located"`
	// Why counts the new rows by the reason that the loose import made no record of them. Only All has it.
	Why map[string]int `json:"why,omitempty"`
}

// Rate gives the part of the rows that can be new problems, from 0 to 1.
func (a Against) Rate(rows int) float64 { return float64(a.New) / float64(max(rows, 1)) }

// Per10k gives the new problems for each 10,000 rows. The exit of milestone 6 asks for less than 100, that is less than 1%.
func (a Against) Per10k(rows int) float64 {
	return 10000 * float64(a.NewProblems) / float64(max(rows, 1))
}

func (a *Against) add(o Against) {
	a.Matched += o.Matched
	a.ByKey += o.ByKey
	a.ByLink += o.ByLink
	a.Variant += o.Variant
	a.Near += o.Near
	a.New += o.New
	a.NewProblems += o.NewProblems
	a.NewLocated += o.NewLocated
	for k, v := range o.Why {
		if a.Why == nil {
			a.Why = map[string]int{}
		}
		a.Why[k] += v
	}
}

func (m *MatchRate) add(o MatchRate) {
	m.Rows += o.Rows
	m.Placed.add(o.Placed)
	m.All.add(o.All)
}

// count counts one row. keyed tells that the appearance of the row has a record, and linked that its capture is a statement of a record. n is the nearest record and answer its answer. It tells whether the row is new.
func (a *Against) count(keyed, linked bool, n similar.Match, rowAnswer, answer string) bool {
	switch {
	case keyed:
		a.Matched++
		a.ByKey++
	case linked:
		a.Matched++
		a.ByLink++
	case n.Score >= match.EStrict && match.Answers(seedAnswer(rowAnswer), answer) == match.AnswerMismatch:
		a.Variant++
	case n.Score >= match.EStrict:
		a.Matched++
	case n.Score >= DupLeast:
		a.Near++
	default:
		a.New++
		return true
	}
	return false
}

// matchBatch is the number of seed rows that the workers match at once. The rows are read as a stream, so a large dataset is never in memory as a whole.
const matchBatch = 4096

// nearPair holds the nearest record of a row among all the records and among the placed records.
type nearPair struct{ all, placed similar.Match }

func runMatchrate(env Env, args []string) int {
	fs := newFlags("matchrate", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	out := fs.String("out", "", "the directory of the report; the default is reports/coverage in the content repository")
	derived := fs.Bool("derived", false, "match the derived datasets of the source register (derived: true) and write derived.md, derived.json and derived-new.tsv")
	workers := fs.Int("workers", runtime.NumCPU(), "the number of workers that match the rows")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	fail := func(err error) int {
		fmt.Fprintf(env.Stderr, "om matchrate: %v\n", err)
		return ExitFailure
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om matchrate: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	al, err := seed.LoadAliases(*root, corpus)
	if err != nil {
		return fail(err)
	}
	sources := pos
	// missing are the derived datasets that have no seed rows yet.
	var missing []string
	if len(sources) == 0 {
		have, err := seed.Sources(*dir)
		if err != nil {
			return fail(err)
		}
		for _, s := range corpus.Sources {
			switch {
			case s.Kind != "dataset" || (*derived && !s.Derived):
			case slices.Contains(have, s.ID):
				sources = append(sources, s.ID)
			case *derived:
				missing = append(missing, s.ID)
			}
		}
	}
	have := map[string]bool{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			have[a.At] = true
		}
	}
	// The docs are the placed records first and then the other records, so the placed records are the docs before placed. captures maps the capture of each statement to whether its record is placed.
	var docs, answers []string
	captures := map[string]bool{}
	seen := map[*content.Problem]bool{}
	addDoc := func(p *content.Problem, isPlaced bool) {
		if seen[p] {
			return
		}
		seen[p] = true
		for _, s := range p.Statements {
			if s.Capture != "" {
				captures[s.Capture] = captures[s.Capture] || isPlaced
			}
		}
		if strings.TrimSpace(p.Statement) != "" {
			docs = append(docs, dupText(p))
			answers = append(answers, p.Answer)
		}
	}
	for _, p := range corpus.Problems() {
		addDoc(p, true)
	}
	placed := len(docs)
	for _, p := range corpus.All() {
		addDoc(p, false)
	}
	seen = nil
	index := similar.NewIndex(docs, 1)
	docs = nil
	why, err := newLooseWhy(*root, corpus, al)
	if err != nil {
		return fail(err)
	}
	var rates []MatchRate
	var fresh strings.Builder
	fresh.WriteString("# source\tyear\tnumber\tlabels\tE\twhy\tstatement\n")
	for _, src := range sources {
		m := MatchRate{Source: src, All: Against{Why: map[string]int{}}}
		// cache holds the nearest records of each text of the dataset, so a text that the dataset has in many rows is matched once. The new texts are kept once each, for the groups.
		cache := map[uint64]nearPair{}
		newPlaced, newAll := map[uint64]string{}, map[uint64]string{}
		var batch []seed.Row
		flush := func() {
			keys := make([]uint64, len(batch))
			texts := make([]string, len(batch))
			var todo []int
			queued := map[uint64]bool{}
			for i, r := range batch {
				texts[i] = r.Statement + mathGrams(r.Statement)
				keys[i] = textKey(texts[i])
				if _, ok := cache[keys[i]]; !ok && !queued[keys[i]] {
					queued[keys[i]] = true
					todo = append(todo, i)
				}
			}
			got := make([]nearPair, len(todo))
			var next atomic.Int64
			var wg sync.WaitGroup
			for range max(min(*workers, len(todo)), 1) {
				wg.Go(func() {
					s := index.Searcher()
					for {
						j := int(next.Add(1)) - 1
						if j >= len(todo) {
							return
						}
						got[j].all, got[j].placed = s.Nearest(texts[todo[j]], placed)
					}
				})
			}
			wg.Wait()
			for j, i := range todo {
				cache[keys[i]] = got[j]
			}
			for i, r := range batch {
				n := cache[keys[i]]
				code := al.Code(r.Labels)
				keyed := have[rowAt(corpus, code, r)]
				onPlaced, linked := captures[r.Source+"/"+r.File+"#row="+strconv.Itoa(r.Row)]
				m.Rows++
				if m.Placed.count(keyed, linked && onPlaced, n.placed, r.Answer, docAnswer(answers, n.placed)) {
					newPlaced[keys[i]] = texts[i]
					if code != "" && code != "-" {
						m.Placed.NewLocated++
					}
				}
				if m.All.count(keyed, linked, n.all, r.Answer, docAnswer(answers, n.all)) {
					newAll[keys[i]] = texts[i]
					w := why.why(r)
					m.All.Why[w]++
					if code != "" && code != "-" {
						m.All.NewLocated++
						fmt.Fprintf(&fresh, "%s\t%d\t%s\t%s\t%.3f\t%s\t%s\n", src, r.Year, r.Number, strings.Join(r.Labels, "; "), n.all.Score, w, oneLine(r.Statement))
					}
				}
			}
			batch = batch[:0]
		}
		err := seed.Read(*dir, src, func(r seed.Row) error {
			if strings.TrimSpace(r.Statement) == "" {
				return nil
			}
			if batch = append(batch, r); len(batch) == matchBatch {
				flush()
			}
			return nil
		})
		if err != nil {
			return fail(fmt.Errorf("%s: %w", src, err))
		}
		flush()
		cache = nil
		m.Placed.NewProblems = groups(slices.Collect(maps.Values(newPlaced)), *workers)
		newPlaced = nil
		m.All.NewProblems = groups(slices.Collect(maps.Values(newAll)), *workers)
		fmt.Fprintf(env.Stdout, "%s: %d rows. Against the placed records: %s. Against all the records: %s.\n", src, m.Rows, m.Placed.line(m.Rows), m.All.line(m.Rows))
		rates = append(rates, m)
	}
	if *out == "" {
		*out = filepath.Join(*root, "reports", "coverage")
	}
	if err := os.MkdirAll(*out, 0o750); err != nil {
		return fail(err)
	}
	name, report := "match", matchReport(placed, index.Len(), rates)
	if *derived {
		name, report = "derived", derivedReport(placed, index.Len(), rates, missing)
	}
	js, err := json.MarshalIndent(rates, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(*out, name+".json"), append(js, '\n'), 0o600)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(*out, name+"-new.tsv"), []byte(fresh.String()), 0o600)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(*out, name+".md"), []byte(report), 0o600)
	}
	if err != nil {
		return fail(err)
	}
	return ExitOK
}

// line gives the counts for the output of om matchrate.
func (a Against) line(rows int) string {
	return fmt.Sprintf("%d matched (%d by key, %d by link), %d variants, %d near, %d new (%.2f%%), %d new problems (%.0f for each 10,000 rows), %d new with a competition", a.Matched, a.ByKey, a.ByLink, a.Variant, a.Near, a.New, 100*a.Rate(rows), a.NewProblems, a.Per10k(rows), a.NewLocated)
}

// docAnswer gives the answer of the nearest record, or "" when there is none.
func docAnswer(answers []string, n similar.Match) string {
	if n.Doc < 0 {
		return ""
	}
	return answers[n.Doc]
}

// textKey gives a hash of a text, so that the same text in two rows is matched once.
func textKey(t string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(t))
	return h.Sum64()
}

// rowAt gives the appearance of a seed row in the competition code, such as aime/2007b/p8, or "" when the row has no year or no position.
func rowAt(corpus *content.Corpus, code string, r seed.Row) string {
	c := corpus.Competition(code)
	pos := seed.Position(r.Number)
	if c == nil || r.Year == 0 || pos == "" {
		return ""
	}
	ed := strconv.Itoa(r.Year)
	if c.EditionForm == "year-letter" {
		ed += canon.Letter(r.Labels)
	}
	return code + "/" + ed + "/" + pos
}

// seedAnswer gives the answer of a seed row, or "" when the row says that it has none, as the NuminaMath rows of proof problems do.
func seedAnswer(a string) string {
	switch strings.ToLower(strings.TrimSpace(a)) {
	case "proof", "notfound", "none", "n/a", "-":
		return ""
	}
	return a
}

// groups counts the groups of texts that are one problem: two texts with E of at least match.EStrict are in one group. The texts of a group are the 5 best matches of each text, as in similar.Find, but the index holds no copy of the texts, so a large set of new rows needs little memory.
func groups(texts []string, workers int) int {
	slices.Sort(texts)
	index := similar.NewIndex(texts, 2)
	ms := make([][]similar.Match, len(texts))
	var next atomic.Int64
	var wg sync.WaitGroup
	for range max(min(workers, len(texts)), 1) {
		wg.Go(func() {
			s := index.Searcher()
			for {
				i := int(next.Add(1)) - 1
				if i >= len(texts) {
					return
				}
				ms[i] = s.Above(texts[i], i, 5, match.EStrict)
			}
		})
	}
	wg.Wait()
	parent := make([]int, len(texts))
	for i := range parent {
		parent[i] = i
	}
	root := func(i int) int {
		for parent[i] != i {
			parent[i] = parent[parent[i]]
			i = parent[i]
		}
		return i
	}
	n := len(texts)
	for i := range ms {
		for _, m := range ms[i] {
			if a, b := root(i), root(m.Doc); a != b {
				parent[a] = b
				n--
			}
		}
	}
	return n
}

// matchMethod tells how a row is matched, for the reports.
func matchMethod(placed, records int) string {
	return fmt.Sprintf("A seed row is matched by key (K) when its competition, year and position give an appearance that has a record. It is matched by link when its capture (the file and the row of the dataset) is a statement of a record: an import took the row into that record. Else it is compared by E, the cosine of the TF-IDF weights of the words and TeX, with the canonical statements of the records. A row whose nearest record has E of at least %.1f is matched, unless the row and the record both have an answer and the answers differ after a light normalization (signal A). Such a row is a variant: it can be another problem, or the same answer in another form, so a person must look at it. A row with E from %.1f to %.1f is near, and a person must look at it too. The other rows are new. New rows of one dataset that have E of at least %.1f with each other are one new problem, so a problem that the dataset has in several rows counts once.\n\nEach row is matched twice: against the %d placed records (with an appearance in a competition), and against all %d records with a statement, the loose records under problems/_loose included. The weights of the words come from all the records in both cases.", match.EStrict, DupLeast, match.EStrict, match.EStrict, placed, records)
}

func rateTable(p func(string, ...any), rates []MatchRate, side func(MatchRate) Against) {
	p("| Dataset | Rows | Matched | By key | By link | Variants | Near | New rows | New rate | New problems | New per 10,000 rows | New with a competition |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	var t MatchRate
	row := func(name string, m MatchRate) {
		a := side(m)
		p("| %s | %d | %d | %d | %d | %d | %d | %d | %.2f%% | %d | %.0f | %d |\n", name, m.Rows, a.Matched, a.ByKey, a.ByLink, a.Variant, a.Near, a.New, 100*a.Rate(m.Rows), a.NewProblems, a.Per10k(m.Rows), a.NewLocated)
	}
	for _, m := range rates {
		row(m.Source, m)
		t.add(m)
	}
	row("All", t)
}

func placedSide(m MatchRate) Against { return m.Placed }

func allSide(m MatchRate) Against { return m.All }

// rateTables writes the short table of both rates, the tables of the two matches and the reasons of the new rows.
func rateTables(p func(string, ...any), rates []MatchRate) {
	p("## New problems for each 10,000 rows\n\n| Dataset | Rows | Against the placed records | Against all the records |\n|---|---:|---:|---:|\n")
	var t MatchRate
	for _, m := range rates {
		p("| %s | %d | %.0f | %.0f |\n", m.Source, m.Rows, m.Placed.Per10k(m.Rows), m.All.Per10k(m.Rows))
		t.add(m)
	}
	p("| All | %d | %.0f | %.0f |\n\n", t.Rows, t.Placed.Per10k(t.Rows), t.All.Per10k(t.Rows))
	p("## Against the placed records\n\n")
	rateTable(p, rates, placedSide)
	p("\n## Against all the records\n\n")
	rateTable(p, rates, allSide)
	p("\n## Why the new rows have no record\n\nFor each row that is new against all the records, the reason that the loose import (`om import -loose`) made no record of it: the check that dropped the row, %q for a row that the import left for a review (%s), %q for a dataset that the loose import does not read, and %q for a row that passed the checks. Such a row can be one that the import linked or joined by its other signals, or a new problem. The removed rows are in %s.\n\n", whyHeld, HeldFile, whyNotRead, whyPassed, RemovedFile)
	reasons := map[string]int{}
	for _, m := range rates {
		for k, v := range m.All.Why {
			reasons[k] += v
		}
	}
	keys := slices.Collect(maps.Keys(reasons))
	sort.Slice(keys, func(i, j int) bool {
		if reasons[keys[i]] != reasons[keys[j]] {
			return reasons[keys[i]] > reasons[keys[j]]
		}
		return keys[i] < keys[j]
	})
	p("| Reason |")
	for _, m := range rates {
		p(" %s |", m.Source)
	}
	p(" All |\n|---|%s---:|\n", strings.Repeat("---:|", len(rates)))
	for _, k := range keys {
		p("| %s |", k)
		for _, m := range rates {
			p(" %d |", m.All.Why[k])
		}
		p(" %d |\n", reasons[k])
	}
}

func matchReport(placed, records int, rates []MatchRate) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Match rate\n\n")
	p("This report is made by `om matchrate` (spec 05.4). %s The report of the derived datasets alone is `derived.md`.\n\n", matchMethod(placed, records))
	rateTables(p, rates)
	return b.String()
}

// derivedReport is the match rate report of the derived datasets (spec 05.1), for the exit of milestone 6: less than 1% new problems for each 10,000 rows, that is less than 100. The exit counts the rate against all the records, because a loose record is a problem of the corpus too.
func derivedReport(placed, records int, rates []MatchRate, missing []string) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Match rate of the derived datasets\n\n")
	p("This report is made by `om matchrate -derived` (spec 05.4 and 06.5). It reads every seed row of the datasets that the source register marks `derived: true`: rows copied from other sources, sometimes changed or written by a model (spec 05.1). %s\n\n", matchMethod(placed, records))
	p("The exit of milestone 6 asks for less than 1%% new problems for each 10,000 rows, that is less than 100 new problems for each 10,000 rows of a dataset. The exit uses the rate against all the records.\n\n")
	var over []string
	for _, m := range rates {
		if m.All.Per10k(m.Rows) >= 100 {
			over = append(over, m.Source)
		}
	}
	if len(over) == 0 && len(rates) > 0 {
		p("Every dataset gives less than 100 new problems for each 10,000 rows against all the records, so the exit holds for the datasets below.\n\n")
	} else {
		p("%d of %d datasets give 100 or more new problems for each 10,000 rows against all the records: %s. The exit does not hold yet.\n\n", len(over), len(rates), strings.Join(over, ", "))
	}
	if len(missing) > 0 {
		p("These derived datasets have no seed rows yet, so they are not in the tables: %s.\n\n", strings.Join(missing, ", "))
	}
	rateTables(p, rates)
	return b.String()
}

// The reasons of a new row that the loose import did not drop.
const (
	whyHeld    = "held for a review"
	whyNotRead = "dataset not read by the loose import"
	whyPassed  = "passed the checks"
)

// looseWhy tells why the loose import made no record of a seed row.
type looseWhy struct {
	r      *looseReader
	accept func(string) bool
	held   map[string]bool
}

func newLooseWhy(root string, corpus *content.Corpus, al *seed.Aliases) (*looseWhy, error) {
	if _, err := site.NewMarkup(nil); err != nil {
		return nil, err
	}
	r, err := newLooseReader(root, corpus, al, false, NuminaLabels, LinkOnly)
	if err != nil {
		return nil, err
	}
	w := &looseWhy{r: r, accept: newTeXCheck(), held: map[string]bool{}}
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(HeldFile))) // #nosec G304 -- a report of the content root that the user names.
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	for _, line := range strings.Split(string(b), "\n") {
		if c, _, _ := strings.Cut(line, "\t"); c != "" && !strings.HasPrefix(c, "#") {
			w.held[c] = true
		}
	}
	return w, nil
}

// why gives the reason: the check of the loose import that drops the row, or one of whyHeld, whyNotRead and whyPassed.
func (w *looseWhy) why(row seed.Row) string {
	if !slices.Contains(LooseSources, row.Source) {
		return whyNotRead
	}
	n := len(w.r.pending)
	w.r.last = ""
	_ = w.r.add(row)
	if len(w.r.pending) == n {
		if w.r.last == "" {
			return "already imported"
		}
		return w.r.last
	}
	pr := w.r.pending[n]
	w.r.pending = w.r.pending[:n]
	if x, why := w.r.check(pr, w.accept); x == nil {
		return why
	}
	if w.held[pr.capture] {
		return whyHeld
	}
	return whyPassed
}
