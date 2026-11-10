// Package export writes the Parquet export of the corpus for Hugging Face (spec 10.7 and 11.5).
//
// The export has one config for each licence zone: z1-text, z2-facts and z3-links. Each config is one Parquet file with one row for each problem whose statement is in the zone. A row carries the facts of the problem, its appearances and its solutions, and the source, the zone and the licence of each text. The text of a source with ai_train: no, or with do_not_republish, is never in the export: the row keeps the facts and the link, and says why the text is left out.
package export

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/parquet-go/parquet-go"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

// Config is one config of the export.
type Config struct {
	Name, Zone, Summary string
}

// Configs are the configs of the export, one for each licence zone (spec 11.5).
var Configs = []Config{
	{"z1-text", "Z1", "problems with an open licence or a granted permission, with the full text"},
	{"z2-facts", "Z2", "problems quoted as a reference, with the facts, the statement where the source allows it, and the links"},
	{"z3-links", "Z3", "problems that olymmath only links to, with the facts and the links and no text"},
}

// The reasons that a text is left out.
const (
	LeftAITrain   = "ai-train=no"
	LeftNoRepub   = "do-not-republish"
	LeftLinkOnly  = "link-only"
	LeftExcerpted = "excerpt-only"
)

// Solution is one solution in a row.
type Solution struct {
	Tag     string `parquet:"tag"`
	Origin  string `parquet:"origin"`
	Author  string `parquet:"author,optional"`
	Source  string `parquet:"source"`
	Zone    string `parquet:"zone"`
	Licence string `parquet:"licence"`
	URL     string `parquet:"url,optional"`
	// Text is empty when it is left out. LeftOut says why.
	Text    string `parquet:"text,optional"`
	LeftOut string `parquet:"left_out,optional"`
	// Model names the model of an AI solution (spec 11.7).
	Model string `parquet:"model,optional"`
}

// Row is one problem of the export.
type Row struct {
	Tag         string   `parquet:"tag"`
	Locator     string   `parquet:"locator,optional"`
	URL         string   `parquet:"url"`
	Title       string   `parquet:"title"`
	Competition string   `parquet:"competition,optional"`
	Year        int32    `parquet:"year,optional"`
	Position    string   `parquet:"position,optional"`
	Type        string   `parquet:"type,optional"`
	Topics      []string `parquet:"topics,list"`
	Band        int32    `parquet:"band,optional"`
	Answer      string   `parquet:"answer,optional"`
	// Appearances are the locators of the problem, the primary first.
	Appearances []string `parquet:"appearances,list"`
	// Source, Zone and Licence are those of the statement.
	Source    string `parquet:"source"`
	Zone      string `parquet:"zone"`
	Licence   string `parquet:"licence"`
	SourceURL string `parquet:"source_url,optional"`
	Origin    string `parquet:"origin,optional"`
	Statement string `parquet:"statement,optional"`
	// Figures are the URLs of the figures of the statement on the site, as https://olymmath.tamnd.com/imo/2024/p1.fig1.svg. A figure is part of the statement, so a row whose statement is left out has none. The statement links to a figure by the name of its file.
	Figures   []string   `parquet:"figures,list"`
	LeftOut   string     `parquet:"left_out,optional"`
	Solutions []Solution `parquet:"solutions,list"`
}

// Stats count the rows of each config and the texts left out for each reason.
type Stats struct {
	Rows    map[string]int
	LeftOut map[string]int
	// Sources are the sources whose texts are left out for ai-train=no.
	Sources []string
}

// Options control an export.
type Options struct {
	URL    string // the public URL of the site, with no slash at the end
	Commit string // the commit of the content repository, for the card
}

// Reference is the licence of a third-party text with no stated licence (spec 11.3).
const Reference = "all rights reserved, quoted as reference"

type exporter struct {
	corpus *content.Corpus
	api    *site.API
	opts   Options
	stats  Stats
	noAI   map[string]bool
}

// Rows gives the rows of each config, in the order of the paths of the problem files.
func Rows(corpus *content.Corpus, opts Options) (map[string][]Row, Stats) {
	x := &exporter{corpus: corpus, api: site.NewAPI(corpus, opts.URL), opts: opts, stats: Stats{Rows: map[string]int{}, LeftOut: map[string]int{}}, noAI: map[string]bool{}}
	x.opts.URL = strings.TrimSuffix(x.opts.URL, "/")
	out := map[string][]Row{}
	for _, p := range corpus.All() {
		r := x.row(p)
		name := configOf(r.Zone)
		out[name] = append(out[name], r)
		x.stats.Rows[name]++
	}
	for s := range x.noAI {
		x.stats.Sources = append(x.stats.Sources, s)
	}
	sort.Strings(x.stats.Sources)
	return out, x.stats
}

// configOf gives the config of a zone. A text with no zone is treated as link only.
func configOf(zone string) string {
	for _, c := range Configs {
		if c.Zone == zone {
			return c.Name
		}
	}
	return "z3-links"
}

// leftOut gives why the text of a source in a zone is left out, or "". full is true for a text that the zone carries in full, as a statement in Z1 or Z2 or a solution in Z1.
func (x *exporter) leftOut(source, zone string, full bool) string {
	r := x.corpus.Source(source)
	switch {
	case r != nil && r.DoNotRepublish:
		return LeftNoRepub
	case r != nil && r.AITrain == "no":
		x.noAI[source] = true
		return LeftAITrain
	case zone == "Z3" || zone == "":
		return LeftLinkOnly
	case !full:
		return LeftExcerpted
	}
	return ""
}

func (x *exporter) licence(source string) string {
	if r := x.corpus.Source(source); r != nil && r.Licence != "" {
		return r.Licence
	}
	return Reference
}

func (x *exporter) row(p *content.Problem) Row {
	rec := x.api.Problem(p)
	r := Row{
		Tag: p.Tag, Locator: rec.Locator, URL: rec.URL, Title: rec.Title, Competition: p.Competition,
		Year: int32(p.Year), Position: p.Position, Type: p.Type, Topics: p.Topics, // #nosec G115 -- a year fits in 32 bits.
		Band: int32(rec.Band), Answer: p.Answer, Zone: p.Source.Zone, SourceURL: p.Source.URL, Origin: p.Source.Origin, // #nosec G115 -- a band is 1 to 10.
		Solutions: []Solution{}, Figures: []string{},
	}
	if r.Topics == nil {
		r.Topics = []string{}
	}
	for _, a := range p.Appearances {
		if a.Primary {
			r.Appearances = append([]string{a.Locator.String()}, r.Appearances...)
		} else {
			r.Appearances = append(r.Appearances, a.Locator.String())
		}
	}
	if r.Appearances == nil {
		r.Appearances = []string{}
	}
	if st := p.CanonicalStatement(); st != nil {
		r.Source = st.Source
	}
	r.Licence = x.licence(r.Source)
	if r.LeftOut = x.leftOut(r.Source, r.Zone, true); r.LeftOut == "" {
		r.Statement = p.Statement
		r.Figures = append(r.Figures, rec.Figures...)
	} else {
		x.stats.LeftOut[r.LeftOut]++
	}
	for _, s := range p.Solutions {
		zone := s.Zone
		if src := x.corpus.Source(s.Source); zone == "" && src != nil {
			zone = src.Zone.Solution
		}
		sol := Solution{Tag: s.Tag, Origin: s.Origin, Author: s.Author.Display, Source: s.Source, Zone: zone, Licence: x.licence(s.Source), URL: s.URL}
		if s.Source == "" {
			// A solution with no source is written for olymmath (spec 11.3).
			sol.Licence = "CC-BY-SA-4.0"
			if s.Origin == "ai" {
				sol.Licence = "CC-BY-4.0"
			}
		}
		if s.Run != nil {
			sol.Model = s.Run.Model
		}
		if sol.LeftOut = x.leftOut(s.Source, zone, zone == "Z1"); sol.LeftOut == "" {
			sol.Text = s.Body
		} else {
			x.stats.LeftOut[sol.LeftOut]++
		}
		r.Solutions = append(r.Solutions, sol)
	}
	return r
}

// Write writes the export to dir: one Parquet file for each config and the card README.md. The directory is replaced. The same corpus gives the same bytes.
func Write(corpus *content.Corpus, dir string, opts Options) (Stats, error) {
	rows, stats := Rows(corpus, opts)
	if err := os.RemoveAll(dir); err != nil {
		return stats, err
	}
	for _, c := range Configs {
		data, err := Parquet(rows[c.Name])
		if err != nil {
			return stats, fmt.Errorf("export: %s: %w", c.Name, err)
		}
		if err := os.MkdirAll(filepath.Join(dir, c.Name), 0o750); err != nil {
			return stats, err
		}
		if err := os.WriteFile(filepath.Join(dir, c.Name, "train.parquet"), data, 0o600); err != nil {
			return stats, err
		}
	}
	return stats, os.WriteFile(filepath.Join(dir, "README.md"), []byte(Card(stats, opts)), 0o600)
}

// Parquet gives the rows as one Parquet file, compressed with zstd.
func Parquet(rows []Row) ([]byte, error) {
	var buf bytes.Buffer
	w := parquet.NewGenericWriter[Row](&buf, parquet.Compression(&parquet.Zstd), parquet.CreatedBy("om", "", ""))
	if _, err := w.Write(rows); err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Card gives the dataset card: the configs, the fields, and what the export leaves out and why (spec 11.5).
func Card(st Stats, opts Options) string {
	var b strings.Builder
	b.WriteString("---\nlicense: other\nlicense_name: per-row\nlicense_link: https://olymmath.tamnd.com/about/\npretty_name: olymmath\nlanguage:\n- en\ntags:\n- mathematics\n- olympiad\nconfigs:\n")
	for _, c := range Configs {
		fmt.Fprintf(&b, "- config_name: %s\n  data_files:\n  - split: train\n    path: %s/train.parquet\n", c.Name, c.Name)
	}
	b.WriteString("---\n\n# olymmath\n\n")
	b.WriteString("Olympiad mathematics problems and solutions from https://olymmath.tamnd.com, with the source of each text.")
	if opts.Commit != "" {
		fmt.Fprintf(&b, " This export is built from the commit %s of https://github.com/tamnd/olymmath.", opts.Commit)
	}
	b.WriteString("\n\n## Configs\n\nThe export has one config for each licence zone. Each row is one problem, with its appearances and its solutions. Each row and each solution carries its source, its zone and its licence.\n\n| Config | Rows | What it has |\n|---|---|---|\n")
	for _, c := range Configs {
		fmt.Fprintf(&b, "| `%s` | %d | %s |\n", c.Name, st.Rows[c.Name], c.Summary)
	}
	b.WriteString("\n## What is left out, and why\n\nThe field `left_out` of a row or a solution says why its text is not in the export:\n\n")
	fmt.Fprintf(&b, "- `%s` (%d texts): the source asks that its texts are not used to train models. The row keeps the facts and the link.", LeftAITrain, st.LeftOut[LeftAITrain])
	if len(st.Sources) > 0 {
		fmt.Fprintf(&b, " The sources are %s.", strings.Join(st.Sources, ", "))
	}
	fmt.Fprintf(&b, "\n- `%s` (%d texts): the holder asked that the text is not published.\n", LeftNoRepub, st.LeftOut[LeftNoRepub])
	fmt.Fprintf(&b, "- `%s` (%d texts): the source allows a link only.\n", LeftLinkOnly, st.LeftOut[LeftLinkOnly])
	fmt.Fprintf(&b, "- `%s` (%d texts): the site shows only a short excerpt of the solution, so the export has the link.\n", LeftExcerpted, st.LeftOut[LeftExcerpted])
	b.WriteString("\nA figure is part of its statement. The field `figures` of a row has the URLs of the figures of the statement, and it is empty when the statement is left out.\n")
	b.WriteString("\nThe facts of a problem (its competition, year, position, answer, topics and difficulty) are CC0 1.0. A text keeps the licence of its source. A text with no stated licence is quoted as a reference, and its copyright stays with its holder. AI solutions are labelled with `origin: ai` and the model.\n")
	return b.String()
}
