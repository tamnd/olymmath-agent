// Package serve answers the read methods of the API over HTTP, and the same methods as the tools of an MCP server (spec 10.8). It gives the records of site.API, so its answers are the bytes of the static files that om site build writes under /api/.
package serve

import (
	"math/rand/v2"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/site"
)

// Server holds the records of a corpus and a search text for each problem.
type Server struct {
	corpus  *content.Corpus
	api     *site.API
	version string
	// problems are all the problems, in the order of the search results: the problems with an appearance first, then the loose problems.
	problems []*content.Problem
	// text is the lowercase search text of each problem of problems.
	text []string
}

// New returns a server of the corpus. url is the public URL of the site, which the records link to. version names om in the MCP handshake.
func New(corpus *content.Corpus, url, version string) *Server {
	s := &Server{corpus: corpus, api: site.NewAPI(corpus, url), version: version}
	s.problems = append(corpus.Problems(), corpus.Loose...)
	s.text = make([]string, len(s.problems))
	for i, p := range s.problems {
		r := s.api.Problem(p)
		s.text[i] = strings.ToLower(strings.Join(append([]string{r.Title, r.Locator, r.Tag, site.Plain(p.Statement)}, p.Topics...), " "))
	}
	return s
}

// MaxResults is the most results of one search.
const MaxResults = 100

// SearchResult is the answer of a search: the number of matches and the records of the first matches.
type SearchResult struct {
	Query   string               `json:"query"`
	Total   int                  `json:"total"`
	Results []site.ProblemRecord `json:"results"`
}

// Search gives the problems whose search text has each word of the query. A query that is a tag or a locator gives that problem first. limit is at most MaxResults, and 20 when it is 0 or less.
func (s *Server) Search(q string, limit int) SearchResult {
	if limit <= 0 {
		limit = 20
	}
	limit = min(limit, MaxResults)
	res := SearchResult{Query: q, Results: []site.ProblemRecord{}}
	words := strings.Fields(strings.ToLower(q))
	if len(words) == 0 {
		return res
	}
	q = strings.TrimSpace(q)
	var exact *content.Problem
	if p := s.api.Lookup(strings.ToUpper(q)); p != nil {
		exact = p
	}
	for _, p := range s.problems {
		if exact == nil && p.Locator() == strings.ToLower(strings.Trim(q, "/")) {
			exact = p
		}
	}
	if exact != nil {
		res.Total++
		res.Results = append(res.Results, s.api.Problem(exact))
	}
	for i, p := range s.problems {
		if p == exact || !containsAll(s.text[i], words) {
			continue
		}
		res.Total++
		if len(res.Results) < limit {
			res.Results = append(res.Results, s.api.Problem(p))
		}
	}
	return res
}

func containsAll(text string, words []string) bool {
	for _, w := range words {
		if !strings.Contains(text, w) {
			return false
		}
	}
	return true
}

// Filter chooses the problems of a random pick. An empty field matches each problem.
type Filter struct {
	Competition string
	// Topic matches a topic or a subtopic of it: N matches N and N.floor.
	Topic            string
	Type             string
	BandMin, BandMax int
	YearFrom, YearTo int
}

// Match tells if the problem passes the filter.
func (f Filter) Match(p *content.Problem) bool {
	if f.Competition != "" && p.Competition != f.Competition {
		return false
	}
	if f.Type != "" && p.Type != f.Type {
		return false
	}
	if f.Topic != "" && !slices.ContainsFunc(p.Topics, func(t string) bool { return t == f.Topic || strings.HasPrefix(t, f.Topic+".") }) {
		return false
	}
	band := p.Band()
	if (f.BandMin > 0 && band < f.BandMin) || (f.BandMax > 0 && band > f.BandMax) {
		return false
	}
	return (f.YearFrom == 0 || p.Year >= f.YearFrom) && (f.YearTo == 0 || p.Year <= f.YearTo)
}

// Random gives a random problem that passes the filter. A seed other than 0 gives the same problem each time. It gives false when no problem passes.
func (s *Server) Random(f Filter, seed uint64) (site.ProblemRecord, bool) {
	var pool []*content.Problem
	for _, p := range s.problems {
		if f.Match(p) {
			pool = append(pool, p)
		}
	}
	if len(pool) == 0 {
		return site.ProblemRecord{}, false
	}
	var i int
	if seed != 0 {
		i = rand.New(rand.NewPCG(seed, 0)).IntN(len(pool)) // #nosec G404 -- a practice pick needs no secure random.
	} else {
		i = rand.IntN(len(pool)) // #nosec G404 -- a practice pick needs no secure random.
	}
	return s.api.Problem(pool[i]), true
}

// Handler gives the HTTP handler: the read methods under /api/ and the MCP server at /mcp. A path of a record answers with or without .json, as the static file does.
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/index.json", func(w http.ResponseWriter, _ *http.Request) { s.reply(w, s.api.Index()) })
	mux.HandleFunc("GET /api/index", func(w http.ResponseWriter, _ *http.Request) { s.reply(w, s.api.Index()) })
	mux.HandleFunc("GET /api/problem/{tag}", func(w http.ResponseWriter, r *http.Request) {
		if p := s.api.Lookup(strings.TrimSuffix(r.PathValue("tag"), ".json")); p != nil {
			s.reply(w, s.api.Problem(p))
			return
		}
		notFound(w, "no problem has this tag")
	})
	mux.HandleFunc("GET /api/solutions/{tag}", func(w http.ResponseWriter, r *http.Request) {
		if p := s.api.Lookup(strings.TrimSuffix(r.PathValue("tag"), ".json")); p != nil {
			s.reply(w, s.api.Solutions(p))
			return
		}
		notFound(w, "no problem has this tag")
	})
	mux.HandleFunc("GET /api/edition/{comp}/{edition}", func(w http.ResponseWriter, r *http.Request) {
		if e := s.api.Edition(r.PathValue("comp"), strings.TrimSuffix(r.PathValue("edition"), ".json")); e != nil {
			s.reply(w, s.api.EditionRecord(e))
			return
		}
		notFound(w, "no edition has this locator")
	})
	mux.HandleFunc("GET /api/search", func(w http.ResponseWriter, r *http.Request) {
		limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
		s.reply(w, s.Search(r.URL.Query().Get("q"), limit))
	})
	mux.HandleFunc("GET /api/random", func(w http.ResponseWriter, r *http.Request) {
		f, seed := filterOf(r.URL.Query().Get)
		if p, ok := s.Random(f, seed); ok {
			s.reply(w, p)
			return
		}
		notFound(w, "no problem passes the filters")
	})
	mux.HandleFunc("POST /mcp", s.serveMCP)
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { notFound(w, "no such method") })
	return mux
}

// filterOf reads the filters of a random pick from query values.
func filterOf(get func(string) string) (Filter, uint64) {
	num := func(k string) int {
		n, _ := strconv.Atoi(get(k))
		return n
	}
	seed, _ := strconv.ParseUint(get("seed"), 10, 64)
	return Filter{
		Competition: get("competition"), Topic: get("topic"), Type: get("type"),
		BandMin: num("band_min"), BandMax: num("band_max"), YearFrom: num("year_from"), YearTo: num("year_to"),
	}, seed
}

func (s *Server) reply(w http.ResponseWriter, v any) {
	b, err := site.Marshal(v)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write(b) // #nosec G705 -- the body is JSON with the type application/json.
}

func notFound(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}` + "\n"))
}
