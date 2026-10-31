// Package site writes the static site of olymmath from a corpus.
//
// Each page is one HTML file. The math is rendered by KaTeX at build time, thus a page shows its math with JavaScript off. The same build serves two outlets: Cloudflare Workers static assets at the root of olymmath.tamnd.com, and the GitHub Pages mirror under a path prefix. Options.Base gives the prefix.
package site

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/katex"
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
	Pages    int
	Files    int
	Problems int
	Errors   []error
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
		b.static,
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
		"topic": func(t string) string { return content.Topics[t] },
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
		"ordinal": ordinal,
		"add":     func(a, b int) int { return a + b },
		"origin":  originName,
		"zone":    zoneName,
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
}

func (b *builder) view(p *content.Problem) *problemView {
	v := &problemView{
		Problem:   p,
		Title:     fmt.Sprintf("%s %d Problem %d", p.Edition.Competition.Short, p.Year, p.Number()),
		Link:      b.link(p.Locator()),
		Statement: b.html(p.Statement, p.Path),
	}
	if p.Answer != "" {
		v.Answer = b.html(p.Answer, p.Path)
	}
	return v
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

func (b *builder) competitions() error {
	if err := b.render("competitions.html", page{
		Title:       "Competitions",
		Description: "All the mathematical olympiads on olymmath.",
		Path:        "/c/",
		Section:     "competitions",
		Data:        b.corpus.Competitions,
	}); err != nil {
		return err
	}
	for _, c := range b.corpus.Competitions {
		if err := b.render("competition.html", page{
			Title:       c.Name,
			Description: c.Description,
			Path:        "/c/" + c.Code + "/",
			Section:     "competitions",
			Data:        c,
		}); err != nil {
			return err
		}
	}
	return nil
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
			ev := editionView{Edition: e, Title: fmt.Sprintf("%s %d", c.Short, e.Year)}
			byPos := map[string]*problemView{}
			for _, p := range e.Problems {
				byPos[p.Position] = b.view(p)
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
			if err := b.render("edition.html", page{
				Title:       ev.Title,
				Description: fmt.Sprintf("The problems of %s %d.", c.Name, e.Year),
				Path:        fmt.Sprintf("/%s/%d/", c.Code, e.Year),
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
				if err := b.render("problem.html", page{
					Title:       v.Title,
					Description: truncate(Plain(v.Problem.Statement), 180),
					Path:        "/" + v.Locator(),
					Section:     "competitions",
					Data:        v,
				}); err != nil {
					return err
				}
				b.stats.Problems++
			}
		}
	}
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
	for _, p := range b.corpus.Problems() {
		entries = append(entries, searchEntry{
			Title:   fmt.Sprintf("%s %d Problem %d", p.Edition.Competition.Short, p.Year, p.Number()),
			Link:    b.link(p.Locator()),
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
