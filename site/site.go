// Package site writes the static site of olymmath from a corpus.
//
// Each page is one HTML file. The math is rendered by KaTeX at build time, thus a page shows its math with JavaScript off. The same build serves two outlets: Cloudflare Workers static assets at the root of olymmath.tamnd.com, and the GitHub Pages mirror under a path prefix. Options.Base gives the prefix.
package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/coverage"
	"github.com/tamnd/olymmath-agent/katex"
	"github.com/tamnd/olymmath-agent/tag"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

// Options control a build.
type Options struct {
	Out     string    // the output directory
	URL     string    // the public URL of the site, for canonical links and the sitemap, with no slash at the end
	Base    string    // the path prefix of each link, "/" for a site at the root of its host
	Commit  string    // the commit of the content repository
	Version string    // the version of om
	Date    time.Time // the build date
	Repo    string    // the URL of the content repository, for the edit links
}

// Stats are the counts that a build reports.
type Stats struct {
	Pages     int
	Files     int
	Problems  int
	Redirects int
	Errors    []error
}

// Build writes the site of corpus to opts.Out. The directory is replaced.
func Build(corpus *content.Corpus, opts Options) (Stats, error) {
	b, err := newBuilder(corpus, opts)
	if err != nil {
		return Stats{}, err
	}
	if err := os.RemoveAll(opts.Out); err != nil {
		return Stats{}, err
	}
	if err := b.all(); err != nil {
		return b.stats, err
	}
	return b.stats, nil
}

type builder struct {
	corpus *content.Corpus
	opts   Options
	markup *Markup
	tmpl   *template.Template
	stats  Stats
	pages  []string // the URL paths for the sitemap
	asset  string   // the version suffix of the assets, for cache busting
}

func newBuilder(corpus *content.Corpus, opts Options) (*builder, error) {
	if opts.Base == "" {
		opts.Base = "/"
	}
	if !strings.HasPrefix(opts.Base, "/") || !strings.HasSuffix(opts.Base, "/") {
		return nil, fmt.Errorf("site: the base %q must start and end with a slash", opts.Base)
	}
	opts.URL = strings.TrimSuffix(opts.URL, "/")
	if opts.Date.IsZero() {
		opts.Date = time.Now().UTC()
	}
	if opts.Repo == "" {
		opts.Repo = "https://github.com/tamnd/olymmath"
	}
	m, err := NewMarkup(nil)
	if err != nil {
		return nil, err
	}
	b := &builder{corpus: corpus, opts: opts, markup: m}
	b.asset = shortHash(opts.Version + opts.Commit + opts.Date.Format(time.RFC3339))
	b.tmpl, err = template.New("").Funcs(b.funcs()).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		return nil, err
	}
	return b, nil
}

func shortHash(s string) string {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}

func (b *builder) all() error {
	steps := []func() error{
		b.assets,
		b.home,
		b.competitions,
		b.editions,
		b.problems,
		b.redirects,
		b.static,
		b.coverage,
		b.searchIndex,
		b.machine,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return err
		}
	}
	return nil
}

// link returns the path of a page with the base prefix.
func (b *builder) link(p string) string {
	return b.opts.Base + strings.TrimPrefix(p, "/")
}

func (b *builder) funcs() template.FuncMap {
	return template.FuncMap{
		"link":  b.link,
		"asset": func(p string) string { return b.link("assets/"+p) + "?v=" + b.asset },
		"topic": topicName,
		"root":  content.TopicRoot,
		"year":  func() int { return b.opts.Date.Year() },
		"short": func(s string) string {
			if len(s) > 7 {
				return s[:7]
			}
			return s
		},
		"date": func(s string) string {
			t, err := time.Parse("2006-01-02", s)
			if err != nil {
				return s
			}
			return t.Format("2 January 2006")
		},
		"ordinal":   ordinal,
		"poslabel":  positionLabel,
		"poscell":   positionCell,
		"add":       func(a, b int) int { return a + b },
		"origin":    originName,
		"zone":      zoneName,
		"country":   countryName,
		"problems":  competitionProblems,
		"solorigin": solutionOrigin,
		"lower":     strings.ToLower,
	}
}

func ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
}

func originName(o string) string {
	switch o {
	case "H1":
		return "Official text"
	case "H2":
		return "Curated text"
	case "H3":
		return "Community text"
	case "D":
		return "Dataset text"
	}
	return "Unknown origin"
}

// topicName returns the name of a topic. A subtopic such as A.inequality shows as "Algebra: inequality".
func topicName(t string) string {
	root, sub, ok := strings.Cut(t, ".")
	if !ok {
		return content.Topics[root]
	}
	return content.Topics[root] + ": " + strings.ReplaceAll(sub, "-", " ")
}

func solutionOrigin(o string) string {
	switch o {
	case "official":
		return "Official solution"
	case "human":
		return "Human solution"
	case "human+ai":
		return "Human solution with AI help"
	case "ai":
		return "AI solution"
	}
	return "Solution"
}

func zoneName(z string) string {
	switch z {
	case "Z1":
		return "Open licence"
	case "Z2":
		return "Quoted with source"
	case "Z3":
		return "Link only"
	}
	return ""
}

// page is the data of one page.
type page struct {
	Title       string
	Description string
	Path        string // the URL path, with no base prefix
	Canonical   string
	Section     string
	Site        *builder
	Data        any
}

func (b *builder) Commit() string          { return b.opts.Commit }
func (b *builder) Version() string         { return b.opts.Version }
func (b *builder) Repo() string            { return b.opts.Repo }
func (b *builder) Built() string           { return b.opts.Date.Format("2 January 2006") }
func (b *builder) Corpus() *content.Corpus { return b.corpus }

// render writes the template name to the file of the URL path p. A path that ends with a slash gets index.html. Any other path gets .html, and the host serves it with no extension.
func (b *builder) render(name string, pg page) error {
	pg.Site = b
	pg.Canonical = b.opts.URL + "/" + strings.TrimPrefix(pg.Path, "/")
	var buf bytes.Buffer
	if err := b.tmpl.ExecuteTemplate(&buf, name, pg); err != nil {
		return fmt.Errorf("site: page %s: %w", pg.Path, err)
	}
	file := strings.TrimPrefix(pg.Path, "/")
	switch {
	case file == "" || strings.HasSuffix(file, "/"):
		file += "index.html"
	case !strings.Contains(path.Base(file), "."):
		file += ".html"
	}
	if err := b.write(file, buf.Bytes()); err != nil {
		return err
	}
	b.stats.Pages++
	if !strings.HasSuffix(file, "404.html") {
		b.pages = append(b.pages, pg.Path)
	}
	return nil
}

func (b *builder) write(rel string, data []byte) error {
	full := filepath.Join(b.opts.Out, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o750); err != nil {
		return err
	}
	b.stats.Files++
	return os.WriteFile(full, data, 0o600)
}

func (b *builder) html(src, where string) template.HTML {
	out, errs := b.markup.HTML(src)
	for _, err := range errs {
		b.stats.Errors = append(b.stats.Errors, fmt.Errorf("%s: %w", where, err))
	}
	return template.HTML(out) // #nosec G203 -- goldmark escapes raw HTML and KaTeX escapes its input.
}

func (b *builder) assets() error {
	err := fs.WalkDir(staticFS, "static", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := staticFS.ReadFile(p)
		if err != nil {
			return err
		}
		return b.write("assets/"+strings.TrimPrefix(p, "static/"), data)
	})
	if err != nil {
		return err
	}
	return fs.WalkDir(katex.Assets(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := fs.ReadFile(katex.Assets(), p)
		if err != nil {
			return err
		}
		return b.write("assets/katex/"+p, data)
	})
}

// problemView is a problem with its rendered parts.
type problemView struct {
	*content.Problem
	Title     string
	Link      string
	Statement template.HTML
	Answer    template.HTML
	Prev      *problemView
	Next      *problemView
	Solutions []solutionView
	// Also lists the other appearances of the problem, with links to their editions.
	Also []alsoView
	// Label names the position in its edition, for example "Problem 3" or "G1".
	Label string
	// Primary is the title of the primary appearance when the view shows the problem in another edition.
	Primary string
}

// alsoView is one other appearance of a problem.
type alsoView struct {
	Title string
	Link  string
}

// appearanceTitle returns the title of an appearance, for example "ISL 2006 G1" or "IMO 2006 Problem 1".
func appearanceTitle(c *content.Competition, edition, pos string) string {
	q := &content.Problem{Position: pos}
	if n := q.Number(); n > 0 && strings.HasPrefix(pos, "p") {
		return fmt.Sprintf("%s %s Problem %d", c.Short, edition, n)
	}
	return fmt.Sprintf("%s %s %s", c.Short, edition, strings.ToUpper(pos))
}

// solutionView is a solution with its rendered body.
type solutionView struct {
	*content.Solution
	Body template.HTML
	// SourceName is the name of the source in the source register, or the source id.
	SourceName string
}

func (b *builder) view(p *content.Problem) *problemView {
	v := &problemView{
		Problem:   p,
		Title:     problemTitle(p),
		Link:      b.link(problemPath(p)),
		Statement: b.html(p.Statement, p.Path),
		Label:     positionLabel(p.Position),
	}
	if p.Answer != "" {
		v.Answer = b.html(p.Answer, p.Path)
	}
	for _, s := range p.Solutions {
		sv := solutionView{Solution: s, Body: b.html(s.Body, s.Path), SourceName: s.Source}
		if r := b.corpus.Source(s.Source); r != nil {
			sv.SourceName = r.Name
		}
		v.Solutions = append(v.Solutions, sv)
	}
	for _, a := range p.Appearances {
		if a.Primary {
			continue
		}
		if c := b.corpus.Competition(a.Locator.Comp); c != nil {
			v.Also = append(v.Also, alsoView{Title: appearanceTitle(c, a.Locator.Edition, a.Locator.Position), Link: b.link("/" + a.Locator.Comp + "/" + a.Locator.Edition + "/#" + a.Locator.Position)})
		}
	}
	return v
}

// alsoIn returns a view of a problem as it appears in an edition that is not its primary edition. The view links to the page of the problem.
func (b *builder) alsoIn(p *content.Problem, e *content.Edition) *problemView {
	q := *p
	q.Position = p.PositionIn(e)
	q.Edition = e
	q.Paper = nil
	for i := range e.Papers {
		if slices.Contains(e.Papers[i].Problems, q.Position) {
			q.Paper = &e.Papers[i]
		}
	}
	v := b.view(&q)
	v.Link = b.link(problemPath(p))
	v.Primary = problemTitle(p)
	return v
}

// problemTitle returns the title of a problem, for example "IMO 2024 Problem 1".
func problemTitle(p *content.Problem) string {
	switch {
	case p.Edition == nil && p.Title != "":
		return p.Title
	case p.Edition == nil:
		return "Problem " + tag.Display(p.Tag)
	}
	return fmt.Sprintf("%s %s %s", p.Edition.Competition.Short, p.Edition.ID, positionLabel(p.Position))
}

// problemPath returns the URL path of the page of a problem. A loose problem has its page at /p/<tag>.
func problemPath(p *content.Problem) string {
	if p.Edition == nil {
		return "/p/" + p.Tag
	}
	return "/" + p.Locator()
}

type homeData struct {
	Problems     int
	Competitions int
	Editions     int
	Latest       []*content.Edition
	Daily        *problemView
}

func (b *builder) home() error {
	ps := b.corpus.Problems()
	d := homeData{Problems: len(ps), Competitions: len(b.corpus.Competitions), Editions: len(b.corpus.Editions())}
	es := b.corpus.Editions()
	if len(es) > 6 {
		es = es[:6]
	}
	d.Latest = es
	if len(ps) > 0 {
		// The problem of the day changes each day, and each build of one day shows the same problem.
		day := int(b.opts.Date.Unix() / 86400)
		d.Daily = b.view(ps[day%len(ps)])
	}
	return b.render("home.html", page{
		Title:       "olymmath: olympiad mathematics problems and solutions",
		Description: "All olympiad mathematics problems and solutions in one place. Human-written statements first, each with its source.",
		Path:        "/",
		Section:     "home",
		Data:        d,
	})
}

// competitionProblems returns the number of problems of a competition on olymmath.
func competitionProblems(c *content.Competition) int {
	n := 0
	for _, e := range c.Editions {
		n += len(e.Problems)
	}
	return n
}

type countryView struct {
	Name string
	Rows []*content.Competition
}

type groupView struct {
	ID, Title string
	Count     int
	Countries []countryView
}

type competitionsView struct {
	Total, WithProblems int
	Groups              []groupView
}

var groupTitles = map[string]string{
	"international": "International",
	"undergraduate": "Undergraduate",
	"usa":           "United States",
	"national":      "National olympiads",
	"journal":       "Journals",
}

// competitionsData puts the competitions in the groups of the register. The national group is split by country. In each list, the competitions with problems come first, and then the order of the register.
func (b *builder) competitionsData() competitionsView {
	v := competitionsView{Total: len(b.corpus.Competitions)}
	byGroup := map[string][]*content.Competition{}
	for _, c := range b.corpus.Competitions {
		byGroup[c.Group] = append(byGroup[c.Group], c)
		if competitionProblems(c) > 0 {
			v.WithProblems++
		}
	}
	groups := append([]string{}, content.Groups...)
	for g := range byGroup {
		if _, ok := groupTitles[g]; !ok {
			groups = append(groups, g)
		}
	}
	for _, g := range groups {
		cs := byGroup[g]
		if len(cs) == 0 {
			continue
		}
		sort.SliceStable(cs, func(i, j int) bool { return competitionProblems(cs[i]) > 0 && competitionProblems(cs[j]) == 0 })
		title := groupTitles[g]
		if title == "" {
			title = g
		}
		gv := groupView{ID: g, Title: title, Count: len(cs)}
		if g != "national" {
			gv.Countries = []countryView{{Rows: cs}}
		} else {
			byCountry := map[string][]*content.Competition{}
			for _, c := range cs {
				name := countryName(c.Country)
				if c.Country == "" {
					name = "Several countries"
				}
				byCountry[name] = append(byCountry[name], c)
			}
			names := make([]string, 0, len(byCountry))
			for n := range byCountry {
				names = append(names, n)
			}
			sort.Strings(names)
			for _, n := range names {
				gv.Countries = append(gv.Countries, countryView{Name: n, Rows: byCountry[n]})
			}
		}
		v.Groups = append(v.Groups, gv)
	}
	return v
}

func (b *builder) competitions() error {
	if err := b.render("competitions.html", page{
		Title:       "Competitions",
		Description: "All the mathematical olympiads in the register of olymmath, by group and by country.",
		Path:        "/c/",
		Section:     "competitions",
		Data:        b.competitionsData(),
	}); err != nil {
		return err
	}
	for _, c := range b.corpus.Competitions {
		rows, total := coverage.Grid(c, b.opts.Date.Year())
		if err := b.render("competition.html", page{
			Title:       c.Name,
			Description: c.Description,
			Path:        "/c/" + c.Code + "/",
			Section:     "competitions",
			Data:        competitionView{Competition: c, Rows: rows, Total: total, Measures: measures(total)},
		}); err != nil {
			return err
		}
	}
	return nil
}

// competitionView is a competition with its year grid (spec 10.3).
type competitionView struct {
	*content.Competition
	Rows     []coverage.Row
	Total    coverage.Count
	Measures []measureView
}

// measureView is one coverage measure with its value.
type measureView struct {
	Code, Name, Value string
}

// measures gives the measures C0 to C8 of a count.
func measures(c coverage.Count) []measureView {
	out := make([]measureView, coverage.Measures)
	for m := range out {
		out[m] = measureView{Code: "C" + strconv.Itoa(m), Name: coverage.Names[m], Value: c.Percent(m)}
	}
	return out
}

type paperView struct {
	content.Paper
	Problems []*problemView
}

type editionView struct {
	*content.Edition
	Title  string
	Papers []paperView
	Other  []*problemView // the problems that are on no paper
}

func (b *builder) editions() error {
	for _, c := range b.corpus.Competitions {
		for _, e := range c.Editions {
			ev := editionView{Edition: e, Title: c.Short + " " + e.ID}
			byPos := map[string]*problemView{}
			for _, p := range e.Problems {
				byPos[p.Position] = b.view(p)
			}
			var also []*problemView
			for _, p := range e.Also {
				if v := b.alsoIn(p, e); byPos[v.Position] == nil {
					byPos[v.Position] = v
					also = append(also, v)
				}
			}
			used := map[string]bool{}
			for _, paper := range e.Papers {
				pv := paperView{Paper: paper}
				for _, pos := range paper.Problems {
					if v := byPos[pos]; v != nil {
						pv.Problems = append(pv.Problems, v)
						used[pos] = true
					}
				}
				ev.Papers = append(ev.Papers, pv)
			}
			for _, p := range e.Problems {
				if !used[p.Position] {
					ev.Other = append(ev.Other, byPos[p.Position])
				}
			}
			for _, v := range also {
				if !used[v.Position] {
					ev.Other = append(ev.Other, v)
				}
			}
			slices.SortStableFunc(ev.Other, func(x, y *problemView) int { return comparePositions(x.Position, y.Position) })
			if err := b.render("edition.html", page{
				Title:       ev.Title,
				Description: fmt.Sprintf("The problems of %s %d.", c.Name, e.Year),
				Path:        "/" + e.Locator() + "/",
				Section:     "competitions",
				Data:        ev,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (b *builder) problems() error {
	for _, c := range b.corpus.Competitions {
		for _, e := range c.Editions {
			views := make([]*problemView, len(e.Problems))
			for i, p := range e.Problems {
				views[i] = b.view(p)
			}
			for i, v := range views {
				if i > 0 {
					v.Prev = views[i-1]
				}
				if i+1 < len(views) {
					v.Next = views[i+1]
				}
				if err := b.problem(v); err != nil {
					return err
				}
			}
		}
	}
	for _, p := range b.corpus.Loose {
		if err := b.problem(b.view(p)); err != nil {
			return err
		}
	}
	return nil
}

func (b *builder) problem(v *problemView) error {
	p := problemPath(v.Problem)
	if err := b.render("problem.html", page{
		Title:       v.Title,
		Description: truncate(Plain(v.Problem.Statement), 180),
		Path:        p,
		Section:     "competitions",
		Data:        v,
	}); err != nil {
		return err
	}
	b.stats.Problems++
	for _, a := range v.Appearances {
		if !a.Primary && v.Edition != nil {
			if err := b.redirect("/"+a.Locator.String(), p); err != nil {
				return err
			}
		}
	}
	if v.Edition != nil {
		// The tag URL of a problem with a locator goes to the locator URL.
		return b.redirect("/p/"+v.Tag, p)
	}
	return nil
}

// redirects writes a page for each old locator in registers/redirects.tsv and for each old tag in registers/aliases.tsv. A withdrawn tag, with the target "-", gets no page.
func (b *builder) redirects() error {
	tags := make([]string, 0, len(b.corpus.Aliases))
	for old, to := range b.corpus.Aliases {
		if to != content.Withdrawn && b.corpus.ByTag(to) != nil {
			tags = append(tags, old)
		}
	}
	sort.Strings(tags)
	for _, old := range tags {
		if err := b.redirect("/p/"+old, "/p/"+b.corpus.Aliases[old]); err != nil {
			return err
		}
	}
	olds := make([]string, 0, len(b.corpus.Redirects))
	for old := range b.corpus.Redirects {
		olds = append(olds, old)
	}
	sort.Strings(olds)
	for _, old := range olds {
		if err := b.redirect("/"+old, "/"+b.corpus.Redirects[old]); err != nil {
			return err
		}
	}
	return nil
}

// redirect writes a small page that sends the reader from one path to another. A static host has no server rules, so the page uses a meta refresh and a canonical link.
func (b *builder) redirect(from, to string) error {
	target := b.opts.URL + "/" + strings.TrimPrefix(to, "/")
	local := b.link(to)
	page := fmt.Sprintf("<!doctype html>\n<html lang=\"en\"><head><meta charset=\"utf-8\"><title>Moved</title><link rel=\"canonical\" href=\"%s\"><meta name=\"robots\" content=\"noindex\"><meta http-equiv=\"refresh\" content=\"0; url=%s\"></head><body><p>This page moved to <a href=\"%s\">%s</a>.</p></body></html>\n",
		html.EscapeString(target), html.EscapeString(local), html.EscapeString(local), html.EscapeString(target))
	file := strings.TrimPrefix(from, "/") + ".html"
	if err := b.write(file, []byte(page)); err != nil {
		return err
	}
	b.stats.Redirects++
	return nil
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "…"
}

func (b *builder) static() error {
	pages := []struct{ tmpl, title, desc, path, section string }{
		{"about.html", "About", "What olymmath is, and the rules that it follows.", "/about/", "about"},
		{"sources.html", "Sources", "The sources of the texts on olymmath, and their licence zones.", "/sources/", "sources"},
		{"search.html", "Search", "Search the problems of olymmath.", "/search/", "search"},
		{"404.html", "Page not found", "This page is not on olymmath.", "/404.html", ""},
	}
	for _, p := range pages {
		if err := b.render(p.tmpl, page{Title: p.title, Description: p.desc, Path: p.path, Section: p.section}); err != nil {
			return err
		}
	}
	return nil
}

// coverageRow is one competition on the coverage page.
type coverageRow struct {
	Competition *content.Competition
	Count       coverage.Count
	Measures    []measureView
}

// coverageGroup is one group of competitions on the coverage page.
type coverageGroup struct {
	Name string
	Rows []coverageRow
}

// coverageView is the data of the coverage page (spec 14.3).
type coverageView struct {
	Names    []measureView
	Total    []measureView
	Expected int
	Groups   []coverageGroup
}

// coverage writes the coverage page: the measures C0 to C8 of each competition and of the whole corpus.
func (b *builder) coverage() error {
	v := coverageView{}
	for m := range coverage.Measures {
		v.Names = append(v.Names, measureView{Code: "C" + strconv.Itoa(m), Name: coverage.Names[m]})
	}
	var total coverage.Count
	byGroup := map[string][]coverageRow{}
	for _, c := range b.corpus.Competitions {
		_, n := coverage.Grid(c, b.opts.Date.Year())
		if n.Expected == 0 {
			continue
		}
		total.Add(n)
		byGroup[c.Group] = append(byGroup[c.Group], coverageRow{Competition: c, Count: n, Measures: measures(n)})
	}
	for _, g := range content.Groups {
		if rows := byGroup[g]; len(rows) > 0 {
			name := groupTitles[g]
			if name == "" {
				name = g
			}
			v.Groups = append(v.Groups, coverageGroup{Name: name, Rows: rows})
		}
	}
	v.Total, v.Expected = measures(total), total.Expected
	return b.render("coverage.html", page{
		Title:       "Coverage",
		Description: "How much of each competition olymmath has: positions, human texts, official texts and solutions.",
		Path:        "/coverage/",
		Section:     "coverage",
		Data:        v,
	})
}

// searchEntry is one problem in search.json. The keys are short because the file goes to each reader.
type searchEntry struct {
	Title   string   `json:"t"`
	Link    string   `json:"u"`
	Text    string   `json:"x"`
	Year    int      `json:"y"`
	Comp    string   `json:"c"`
	Topics  []string `json:"g,omitempty"`
	Locator string   `json:"l"`
}

func (b *builder) searchIndex() error {
	var entries []searchEntry
	for _, p := range append(b.corpus.Problems(), b.corpus.Loose...) {
		entries = append(entries, searchEntry{
			Title:   problemTitle(p),
			Link:    b.link(problemPath(p)),
			Text:    Plain(p.Statement),
			Year:    p.Year,
			Comp:    p.Competition,
			Topics:  p.Topics,
			Locator: p.Locator(),
		})
	}
	data, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	return b.write("search.json", data)
}

// machine writes the files for crawlers and programs: robots.txt, sitemap.xml, llms.txt and the headers file of Cloudflare.
func (b *builder) machine() error {
	sort.Strings(b.pages)
	var sm strings.Builder
	sm.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" + `<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n")
	for _, p := range b.pages {
		fmt.Fprintf(&sm, "  <url><loc>%s/%s</loc></url>\n", b.opts.URL, strings.TrimPrefix(p, "/"))
	}
	sm.WriteString("</urlset>\n")
	if err := b.write("sitemap.xml", []byte(sm.String())); err != nil {
		return err
	}
	robots := fmt.Sprintf("User-agent: *\nAllow: /\n\nSitemap: %s/sitemap.xml\n", b.opts.URL)
	if err := b.write("robots.txt", []byte(robots)); err != nil {
		return err
	}
	var llms strings.Builder
	fmt.Fprintf(&llms, "# olymmath\n\n> All olympiad mathematics problems and solutions in one place. Human-written statements first, each with its source.\n\n## Competitions\n\n")
	for _, c := range b.corpus.Competitions {
		fmt.Fprintf(&llms, "- [%s](%s/c/%s/): %s\n", c.Name, b.opts.URL, c.Code, c.Description)
	}
	fmt.Fprintf(&llms, "\n## Data\n\n- [Search index](%s/search.json): each problem with its locator and its text.\n", b.opts.URL)
	if err := b.write("llms.txt", []byte(llms.String())); err != nil {
		return err
	}
	headers := `/assets/*
  Cache-Control: public, max-age=31536000, immutable
/*
  X-Content-Type-Options: nosniff
  Referrer-Policy: strict-origin-when-cross-origin
  Permissions-Policy: camera=(), microphone=(), geolocation=()
  Content-Security-Policy: default-src 'self'; img-src 'self' data:; style-src 'self' 'unsafe-inline'; font-src 'self'; script-src 'self'; connect-src 'self'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'
`
	return b.write("_headers", []byte(headers))
}

// comparePositions orders positions by their letters, then by their numbers, so that g2 comes before g10.
func comparePositions(a, b string) int {
	la := strings.TrimRight(a, "0123456789")
	lb := strings.TrimRight(b, "0123456789")
	if la != lb {
		return strings.Compare(la, lb)
	}
	na, _ := strconv.Atoi(a[len(la):])
	nb, _ := strconv.Atoi(b[len(lb):])
	return na - nb
}

// positionCell names a position in the grid of a competition: "3" for p3, and "G1" for g1.
func positionCell(pos string) string {
	return strings.TrimPrefix(positionLabel(pos), "Problem ")
}

// positionLabel names a position for a list of the problems of an edition: "Problem 3" for p3, and "G1" for g1.
func positionLabel(pos string) string {
	if n, err := strconv.Atoi(strings.TrimPrefix(pos, "p")); err == nil && pos != "" {
		return "Problem " + strconv.Itoa(n)
	}
	return strings.ToUpper(pos)
}
