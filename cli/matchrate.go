package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/similar"
)

func init() {
	register(command{"matchrate", "write the match rate report: for each dataset, the seed rows that match a problem record by K, E and A and the rows that can be new problems (om matchrate [-derived] -seed DIR)", runMatchrate})
}

// MatchRate is the match rate of the seed rows of one dataset.
type MatchRate struct {
	Source string `json:"source"`
	// Rows are the rows with a statement.
	Rows int `json:"rows"`
	// Matched are the rows whose appearance has a record, or whose nearest record has E of at least match.EStrict.
	Matched int `json:"matched"`
	// ByKey are the matched rows whose appearance has a record.
	ByKey int `json:"by_key"`
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
}

// Rate gives the part of the rows that can be new problems, from 0 to 1.
func (m MatchRate) Rate() float64 { return float64(m.New) / float64(max(m.Rows, 1)) }

// Per10k gives the new problems for each 10,000 rows. The exit of milestone 6 asks for less than 100, that is less than 1%.
func (m MatchRate) Per10k() float64 { return 10000 * float64(m.NewProblems) / float64(max(m.Rows, 1)) }

func (m *MatchRate) add(o MatchRate) {
	m.Rows += o.Rows
	m.Matched += o.Matched
	m.ByKey += o.ByKey
	m.Variant += o.Variant
	m.Near += o.Near
	m.New += o.New
	m.NewProblems += o.NewProblems
	m.NewLocated += o.NewLocated
}

func runMatchrate(env Env, args []string) int {
	fs := newFlags("matchrate", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	out := fs.String("out", "", "the directory of the report; the default is reports/coverage in the content repository")
	derived := fs.Bool("derived", false, "match the derived datasets of the source register (derived: true) and write derived.md, derived.json and derived-new.tsv")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om matchrate: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	al, err := seed.LoadAliases(*root, corpus)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om matchrate: %v\n", err)
		return ExitFailure
	}
	sources := pos
	// missing are the derived datasets that have no seed rows yet.
	var missing []string
	if len(sources) == 0 {
		have, err := seed.Sources(*dir)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om matchrate: %v\n", err)
			return ExitFailure
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
	var docs, answers []string
	for _, p := range corpus.Problems() {
		if strings.TrimSpace(p.Statement) != "" {
			docs = append(docs, dupText(p))
			answers = append(answers, p.Answer)
		}
	}
	var rates []MatchRate
	var fresh strings.Builder
	fresh.WriteString("# source\tyear\tnumber\tlabels\tE\tstatement\n")
	for _, src := range sources {
		var texts []string
		var located, keyed []bool
		var rowAnswers []string
		var rows []seed.Row
		err := seed.Read(*dir, src, func(r seed.Row) error {
			if strings.TrimSpace(r.Statement) == "" {
				return nil
			}
			texts = append(texts, r.Statement+mathGrams(r.Statement))
			code := al.Code(r.Labels)
			located = append(located, code != "" && code != "-")
			keyed = append(keyed, have[rowAt(corpus, code, r)])
			rowAnswers = append(rowAnswers, r.Answer)
			rows = append(rows, r)
			return nil
		})
		if err != nil {
			fmt.Fprintf(env.Stderr, "om matchrate: %s: %v\n", src, err)
			return ExitFailure
		}
		m := MatchRate{Source: src, Rows: len(texts)}
		var newTexts []string
		for i, n := range similar.Nearest(docs, texts) {
			switch {
			case keyed[i]:
				m.Matched++
				m.ByKey++
			case n.Score >= match.EStrict && match.Answers(seedAnswer(rowAnswers[i]), answers[n.Doc]) == match.AnswerMismatch:
				m.Variant++
			case n.Score >= match.EStrict:
				m.Matched++
			case n.Score >= DupLeast:
				m.Near++
			default:
				m.New++
				newTexts = append(newTexts, texts[i])
				if located[i] {
					m.NewLocated++
					r := rows[i]
					fmt.Fprintf(&fresh, "%s\t%d\t%s\t%s\t%.3f\t%s\n", src, r.Year, r.Number, strings.Join(r.Labels, "; "), n.Score, oneLine(r.Statement))
				}
			}
		}
		m.NewProblems = groups(newTexts)
		fmt.Fprintf(env.Stdout, "%s: %d rows, %d matched, %d variants, %d near, %d new (%.2f%%), %d new problems (%.0f for each 10,000 rows), %d new with a competition\n", src, m.Rows, m.Matched, m.Variant, m.Near, m.New, 100*m.Rate(), m.NewProblems, m.Per10k(), m.NewLocated)
		rates = append(rates, m)
	}
	if *out == "" {
		*out = filepath.Join(*root, "reports", "coverage")
	}
	if err := os.MkdirAll(*out, 0o750); err != nil {
		fmt.Fprintf(env.Stderr, "om matchrate: %v\n", err)
		return ExitFailure
	}
	name, report := "match", matchReport(len(docs), rates)
	if *derived {
		name, report = "derived", derivedReport(len(docs), rates, missing)
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
		fmt.Fprintf(env.Stderr, "om matchrate: %v\n", err)
		return ExitFailure
	}
	return ExitOK
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

// groups counts the groups of texts that are one problem: two texts with E of at least match.EStrict are in one group.
func groups(texts []string) int {
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
	for i, ms := range similar.Find(texts, 5, match.EStrict) {
		for _, m := range ms {
			if a, b := root(i), root(m.Doc); a != b {
				parent[a] = b
				n--
			}
		}
	}
	return n
}

// matchMethod tells how a row is matched, for the reports.
func matchMethod(records int) string {
	return fmt.Sprintf("A seed row is matched by key (K) when its competition, year and position give an appearance that has a record. Else it is compared with the %d problem records with a statement by E, the cosine of the TF-IDF weights of their words and TeX, and a row whose nearest record has E of at least %.1f is matched, unless the row and the record both have an answer and the answers differ after a light normalization (signal A). Such a row is a variant: it can be another problem, or the same answer in another form, so a person must look at it. A row with E from %.1f to %.1f is near, and a person must look at it too. The other rows are new. New rows of one dataset that have E of at least %.1f with each other are one new problem, so a problem that the dataset has in several rows counts once.", records, match.EStrict, DupLeast, match.EStrict, match.EStrict)
}

func rateTable(p func(string, ...any), rates []MatchRate) {
	p("| Dataset | Rows | Matched | By key | Variants | Near | New rows | New rate | New problems | New per 10,000 rows | New with a competition |\n|---|---:|---:|---:|---:|---:|---:|---:|---:|---:|---:|\n")
	var t MatchRate
	for _, m := range rates {
		p("| %s | %d | %d | %d | %d | %d | %d | %.2f%% | %d | %.0f | %d |\n", m.Source, m.Rows, m.Matched, m.ByKey, m.Variant, m.Near, m.New, 100*m.Rate(), m.NewProblems, m.Per10k(), m.NewLocated)
		t.add(m)
	}
	p("| All | %d | %d | %d | %d | %d | %d | %.2f%% | %d | %.0f | %d |\n", t.Rows, t.Matched, t.ByKey, t.Variant, t.Near, t.New, 100*t.Rate(), t.NewProblems, t.Per10k(), t.NewLocated)
}

func matchReport(records int, rates []MatchRate) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Match rate\n\n")
	p("This report is made by `om matchrate` (spec 05.4). %s The report of the derived datasets alone is `derived.md`.\n\n", matchMethod(records))
	rateTable(p, rates)
	return b.String()
}

// derivedReport is the match rate report of the derived datasets (spec 05.1), for the exit of milestone 6: less than 1% new problems for each 10,000 rows, that is less than 100.
func derivedReport(records int, rates []MatchRate, missing []string) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Match rate of the derived datasets\n\n")
	p("This report is made by `om matchrate -derived` (spec 05.4 and 06.5). It reads every seed row of the datasets that the source register marks `derived: true`: rows copied from other sources, sometimes changed or written by a model (spec 05.1). %s\n\n", matchMethod(records))
	p("The exit of milestone 6 asks for less than 1%% new problems for each 10,000 rows, that is less than 100 new problems for each 10,000 rows of a dataset.\n\n")
	rateTable(p, rates)
	var t MatchRate
	var over []string
	for _, m := range rates {
		t.add(m)
		if m.Per10k() >= 100 {
			over = append(over, m.Source)
		}
	}
	p("\n")
	if len(over) == 0 && len(rates) > 0 {
		p("Every dataset gives less than 100 new problems for each 10,000 rows, so the exit holds for the datasets above.\n")
	} else {
		p("%d of %d datasets give 100 or more new problems for each 10,000 rows: %s. The exit does not hold yet.\n", len(over), len(rates), strings.Join(over, ", "))
	}
	if len(missing) > 0 {
		p("\nThese derived datasets have no seed rows yet, so they are not in the table: %s.\n", strings.Join(missing, ", "))
	}
	return b.String()
}
