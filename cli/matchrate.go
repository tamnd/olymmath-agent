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
	register(command{"matchrate", "write the match rate report: for each dataset, the seed rows that match a problem record by E and the rows that can be new problems (om matchrate -seed DIR)", runMatchrate})
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
	// Near are the rows whose nearest record has E from DupLeast to match.EStrict. A person must look at them to tell.
	Near int `json:"near"`
	// New are the rows with no record near them.
	New int `json:"new"`
	// NewLocated are the new rows whose labels name a competition of the register. They can become problem records.
	NewLocated int `json:"new_located"`
}

// Rate gives the part of the rows that can be new problems, from 0 to 1.
func (m MatchRate) Rate() float64 { return float64(m.New) / float64(max(m.Rows, 1)) }

func runMatchrate(env Env, args []string) int {
	fs := newFlags("matchrate", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir := fs.String("seed", seed.DefaultDir(), "the directory of the seed rows")
	out := fs.String("out", "", "the directory of the report; the default is reports/coverage in the content repository")
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
	if len(sources) == 0 {
		have, err := seed.Sources(*dir)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om matchrate: %v\n", err)
			return ExitFailure
		}
		for _, s := range corpus.Sources {
			if s.Kind == "dataset" && slices.Contains(have, s.ID) {
				sources = append(sources, s.ID)
			}
		}
	}
	have := map[string]bool{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			have[a.At] = true
		}
	}
	var docs []string
	for _, p := range corpus.Problems() {
		if strings.TrimSpace(p.Statement) != "" {
			docs = append(docs, dupText(p))
		}
	}
	var rates []MatchRate
	var fresh strings.Builder
	fresh.WriteString("# source\tyear\tnumber\tlabels\tE\tstatement\n")
	for _, src := range sources {
		var texts []string
		var located, keyed []bool
		var rows []seed.Row
		err := seed.Read(*dir, src, func(r seed.Row) error {
			if strings.TrimSpace(r.Statement) == "" {
				return nil
			}
			texts = append(texts, r.Statement+mathGrams(r.Statement))
			code := al.Code(r.Labels)
			located = append(located, code != "" && code != "-")
			keyed = append(keyed, have[rowAt(corpus, code, r)])
			rows = append(rows, r)
			return nil
		})
		if err != nil {
			fmt.Fprintf(env.Stderr, "om matchrate: %s: %v\n", src, err)
			return ExitFailure
		}
		m := MatchRate{Source: src, Rows: len(texts)}
		for i, n := range similar.Nearest(docs, texts) {
			switch {
			case keyed[i]:
				m.Matched++
				m.ByKey++
			case n.Score >= match.EStrict:
				m.Matched++
			case n.Score >= DupLeast:
				m.Near++
			default:
				m.New++
				if located[i] {
					m.NewLocated++
					r := rows[i]
					fmt.Fprintf(&fresh, "%s\t%d\t%s\t%s\t%.3f\t%s\n", src, r.Year, r.Number, strings.Join(r.Labels, "; "), n.Score, oneLine(r.Statement))
				}
			}
		}
		fmt.Fprintf(env.Stdout, "%s: %d rows, %d matched, %d near, %d new (%.2f%%), %d new with a competition\n", src, m.Rows, m.Matched, m.Near, m.New, 100*m.Rate(), m.NewLocated)
		rates = append(rates, m)
	}
	if *out == "" {
		*out = filepath.Join(*root, "reports", "coverage")
	}
	if err := os.MkdirAll(*out, 0o750); err != nil {
		fmt.Fprintf(env.Stderr, "om matchrate: %v\n", err)
		return ExitFailure
	}
	js, err := json.MarshalIndent(rates, "", "  ")
	if err == nil {
		err = os.WriteFile(filepath.Join(*out, "match.json"), append(js, '\n'), 0o600)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(*out, "match-new.tsv"), []byte(fresh.String()), 0o600)
	}
	if err == nil {
		err = os.WriteFile(filepath.Join(*out, "match.md"), []byte(matchReport(len(docs), rates)), 0o600)
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

func matchReport(records int, rates []MatchRate) string {
	var b strings.Builder
	p := func(format string, a ...any) { fmt.Fprintf(&b, format, a...) }
	p("# Match rate\n\n")
	p("This report is made by `om matchrate` (spec 05.4). A seed row is matched when its competition, year and position give an appearance that has a record. Else it is compared with the %d problem records with a statement by E, the cosine of the TF-IDF weights of their words and TeX, and a row whose nearest record has E of at least %.1f is matched. A row with E from %.1f to %.1f is near, and a person must look at it to tell. The other rows can be new problems. The acceptance of milestone 6 asks for less than 1%% new rows.\n\n", records, match.EStrict, DupLeast, match.EStrict)
	p("| Dataset | Rows | Matched | By key | Near | New | New rate | New with a competition |\n|---|---:|---:|---:|---:|---:|---:|---:|\n")
	var t MatchRate
	for _, m := range rates {
		p("| %s | %d | %d | %d | %d | %d | %.2f%% | %d |\n", m.Source, m.Rows, m.Matched, m.ByKey, m.Near, m.New, 100*m.Rate(), m.NewLocated)
		t.Rows += m.Rows
		t.Matched += m.Matched
		t.ByKey += m.ByKey
		t.Near += m.Near
		t.New += m.New
		t.NewLocated += m.NewLocated
	}
	p("| All | %d | %d | %d | %d | %d | %.2f%% | %d |\n", t.Rows, t.Matched, t.ByKey, t.Near, t.New, 100*t.Rate(), t.NewLocated)
	return b.String()
}
