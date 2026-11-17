package cli

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"unibo", "read the papers of the Italian olympiad from olimpiadi.dm.unibo.it and write their problems to it-cesenatico, it-archimede, it-febbraio and it-classi-prime (om unibo -write)", runUnibo})
}

// uniboSource is the site of the Olimpiadi della Matematica of the Unione Matematica Italiana.
const uniboSource = "olimpiadi-unibo"

// uniboCodes are the competitions that om unibo writes.
var uniboCodes = []string{"it-cesenatico", "it-archimede", "it-febbraio", "it-classi-prime"}

// uniboPages are the pages of the site that link the papers of each competition.
var uniboPages = []struct{ url, code string }{
	{"https://olimpiadi.dm.unibo.it/le-gare/gara-nazionale/", "it-cesenatico"},
	{"https://olimpiadi.dm.unibo.it/le-gare/giochi-di-archimede/", "it-archimede"},
	{"https://olimpiadi.dm.unibo.it/le-gare/gare-distrettuali/", "it-febbraio"},
	{"https://olimpiadi.dm.unibo.it/le-gare/gara-classi-prime/", "it-classi-prime"},
}

var (
	// uniboFileRE is a file of a page: a PDF or a zip file of the uploads of the site.
	uniboFileRE = regexp.MustCompile(`href="(https?://olimpiadi\.dm\.unibo\.it/wp-content/uploads/\d{4}/\d{2}/([^"/]+\.(?:pdf|zip)))"`)
	// uniboNotFileRE is a file of a page that holds no paper of the competition: the data of the olympiad, a rule book, and the papers of the make-up round of the Giochi di Archimede of 2024 (archimede2024rec.zip), whose positions the register has not.
	uniboNotFileRE = regexp.MustCompile(`(?i)dati-olimpiadi|regolamento|rec\.zip$`)
	// uniboNotMemberRE is a PDF of a zip file that is no paper of the competition: the answer grids, the letters to the schools, the English version, the papers of the public and team competitions of Cesenatico, the preliminary notes, the online papers of 2021 and the papers of the other versions of 2014.
	uniboNotMemberRE = regexp.MustCompile(`(?i)risp|griglie|lettera|tabella|english|pubblico|squadre|semifinal|finale_|prel|online|_rec|rec\.pdf$| [2-4]\.pdf$`)
	// uniboFebbraioRE is a paper of February, as the one in the zip file of Cesenatico 2005: a paper of another competition than that of the zip file, unless the zip file is of February.
	uniboFebbraioRE = regexp.MustCompile(`(?i)febb`)
	// uniboSolRE is a PDF that holds the solutions, with the problems or without them.
	uniboSolRE = regexp.MustCompile(`(?i)sol`)
	// uniboVariantRE is the paper of the Giochi di Archimede that a file holds.
	uniboVariantRE = regexp.MustCompile(`(?i)(bienn|trienn)`)
	// uniboYear4RE and uniboYear2RE are the year in the name of a file: four digits, as Archimede2022.zip, or the last two digits, as ces16-sol.pdf, SolFeb25.pdf or GaraPrime1617.pdf, the paper of February 2017.
	uniboYear4RE = regexp.MustCompile(`(?:^|[^0-9])((?:19|20)\d\d)(?:[^0-9]|$)`)
	uniboYear2RE = regexp.MustCompile(`(\d\d)\D*$`)
)

// uniboNames are the names of the papers in the names of the statements.
var uniboNames = map[string]string{"it-cesenatico": "Olimpiadi della Matematica, Gara nazionale", "it-archimede": "Olimpiadi della Matematica, Giochi di Archimede", "it-febbraio": "Olimpiadi della Matematica, Gara di febbraio", "it-classi-prime": "Olimpiadi della Matematica, Gara nazionale classi prime"}

// uniboYear reads the year of the contest of a file from its name, or 0.
func uniboYear(name string) int {
	if m := uniboYear4RE.FindStringSubmatch(name); m != nil {
		y, _ := strconv.Atoi(m[1])
		return y
	}
	if m := uniboYear2RE.FindStringSubmatch(strings.TrimSuffix(strings.TrimSuffix(name, ".pdf"), ".zip")); m != nil {
		y, _ := strconv.Atoi(m[1])
		return 2000 + y
	}
	return 0
}

// uniboVariant is the paper of the Giochi di Archimede that a file name gives, biennio or triennio, or "" when the file holds both.
func uniboVariant(name string) string {
	m := uniboVariantRE.FindStringSubmatch(name)
	if m == nil {
		return ""
	}
	return strings.ToLower(m[1]) + "io"
}

// uniboMembers chooses the papers of a zip file of a competition from the names of its files: for each paper the file of its problems, or the file of its problems with their solutions when the zip file has no other. The Giochi di Archimede have a paper for the biennio and one for the triennio; a file that holds both has the variant "".
func uniboMembers(code string, names []string) []hrPaper {
	byVariant := map[string][]string{}
	var order []string
	for _, n := range names {
		base := n[strings.LastIndex(n, "/")+1:]
		if !strings.HasSuffix(strings.ToLower(base), ".pdf") || uniboNotMemberRE.MatchString(base) || code != "it-febbraio" && uniboFebbraioRE.MatchString(base) {
			continue
		}
		v := ""
		if code == "it-archimede" {
			v = uniboVariant(base)
		}
		if _, ok := byVariant[v]; !ok {
			order = append(order, v)
		}
		byVariant[v] = append(byVariant[v], n)
	}
	if len(order) > 1 {
		// A file of both papers is the letter to the schools with the papers, when the zip file has the papers apart.
		order = slices.DeleteFunc(order, func(v string) bool { return v == "" })
	}
	var out []hrPaper
	for _, v := range order {
		ms := byVariant[v]
		slices.SortStableFunc(ms, func(a, b string) int {
			sa, sb := uniboSolRE.MatchString(a[strings.LastIndex(a, "/")+1:]), uniboSolRE.MatchString(b[strings.LastIndex(b, "/")+1:])
			switch {
			case sa == sb:
				return strings.Compare(a, b)
			case sa:
				return 1
			}
			return -1
		})
		out = append(out, hrPaper{kind: code, variant: v, member: ms[0]})
	}
	return out
}

// uniboPapers reads the files of the pages of the competitions, and the papers of each zip file. Two files of a competition that give the same paper count once, the first that the page links.
func uniboPapers(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error) {
	var out []hrPaper
	for _, pg := range uniboPages {
		b, err := hrGet(ctx, fc, uniboSource, pg.url, opt)
		if err != nil {
			return nil, err
		}
		var urls []string
		for _, m := range uniboFileRE.FindAllStringSubmatch(string(b), -1) {
			u := strings.Replace(m[1], "http://", "https://", 1)
			if uniboNotFileRE.MatchString(m[2]) || slices.Contains(urls, u) {
				continue
			}
			urls = append(urls, u)
		}
		for _, u := range urls {
			name := u[strings.LastIndex(u, "/")+1:]
			y := uniboYear(name)
			if y == 0 || slices.Contains(uniboUnread[pg.code], y) {
				continue
			}
			ps := []hrPaper{{kind: pg.code, variant: "", url: u, year: y}}
			if strings.HasSuffix(name, ".zip") {
				raw, err := hrGet(ctx, fc, uniboSource, u, opt)
				if err != nil {
					return nil, err
				}
				names, err := zipNames(raw)
				if err != nil {
					return nil, fmt.Errorf("%s: %w", u, err)
				}
				ps = uniboMembers(pg.code, names)
				for i := range ps {
					ps[i].url, ps[i].year = u, y
				}
			}
			for _, p := range ps {
				if slices.ContainsFunc(out, func(q hrPaper) bool {
					return q.kind == p.kind && q.year == p.year && (q.variant == p.variant || q.variant == "" || p.variant == "")
				}) {
					continue
				}
				out = append(out, p)
			}
		}
	}
	slices.SortStableFunc(out, func(a, b hrPaper) int {
		if c := strings.Compare(a.kind, b.kind); c != 0 {
			return c
		}
		if a.year != b.year {
			return a.year - b.year
		}
		return strings.Compare(a.variant, b.variant)
	})
	return out, nil
}

// uniboUnread are the years of the papers that pdftohtml does not read: of Cesenatico 1997 to 2003 and 2006 and of February 1997 to 2000 the fonts are Type 3 or not embedded, so that the formulas, the ligatures and the accents are lost; February 2002 to 2005 are scans; the fonts of Archimede 2004 give no text, and the pages of Archimede 2011 are turned.
var uniboUnread = map[string][]int{
	"it-cesenatico": {1997, 1998, 1999, 2000, 2001, 2002, 2003, 2006},
	"it-febbraio":   {1997, 1998, 1999, 2000, 2002, 2003, 2004, 2005},
	"it-archimede":  {2004, 2011},
}

// uniboLetters are the letters and the signs beyond ASCII of the Italian papers.
const uniboLetters = "àèéìíîòóùúÀÈÉÌÍÒÓÙÚçäöüëï’‘“”«»–—−°×÷·…€£"

var (
	// uniboStart is the heading of a problem: "7." or, in the older papers, "7)".
	uniboStart = regexp.MustCompile(`^(\d{1,2})[.)](?:\s|$)`)
	// uniboStop are the lines that end a problem: its solution, its answer, the box for the solution of a proof problem with the name of the student, and the notes for the teachers that follow a paper.
	uniboStop = regexp.MustCompile(`(?i)^(?:problema\s+\d+\s*[–—-]?\s*soluzione|soluzion[ei]\b|risposta\s*[:(]|la risposta (?:è|corretta)|risposte\b|nome\s*:|cognome\s*:|punteggio|per ottenere un minimo|indicazioni per|griglia|quesito\s+\d)`)
	// uniboSkip are the header and the footer of a page: the names of the ministry, the union and the school, the title of the olympiad, the number of the page and the address of the site.
	uniboSkip = regexp.MustCompile(`(?i)^(?:ministero|unione matematica italiana|u\.m\.i\.|progetto olimpiadi|scuola normale|olimpiadi? (?:italiana )?di matematica|olimpiadi della matematica|con il contribut[oi]|visitate il sito|.*\bpag\.?\s*\d+\s*(?:di\s*\d+)?$|.*olimpiadi\.dm\.unibo\.it|\d{1,2}$|[ivxlc]+ olimpiade|dell.università e della ricerca|del merito)`)
	// uniboSection are the parts of the papers: the paper of the biennio or of the triennio of the Giochi di Archimede, whose title the older papers set letter by letter in small capitals, as "G ara B iennio"; the groups of the problems of the paper of February.
	uniboArchimedeSection = regexp.MustCompile(`(?i)^(?:.*\bg\s?ara\s+(b\s?iennio|t\s?riennio)\b.*|nome\b.*cognome\b.*|cognome\b.*nome\b.*)$`)
	uniboFebbraioSection  = regexp.MustCompile(`(?i)^(?:problemi\s+(?:a\s+risposta|dimostrativi)|esercizi\s+(?:a\s+risposta|dimostrativi)).*$`)
	// uniboNot are the numbered instructions of the older papers of the Giochi di Archimede and of February, as "1) La prova consiste di 25 problemi": no headings of problems.
	uniboNot = regexp.MustCompile(`(?i)^\d\)\s*(?:la prova|una sola|ogni risposta|per ciascun|il tempo|non sfogliare|nei problemi|i problemi|quando l|non è|non sono|hai |le risposte|tutte le|scrivi|ricorda)`)
	// uniboPart are the lines that are paragraphs of their own: the options of a question, as "(A) 12 (B) 24", and the parts of a problem, as "(a)" or "b)".
	uniboPart = regexp.MustCompile(`^(?:\([A-E]\)|\(?[a-z]\)|\([ivx]+\)|•|- )\s?`)
	// uniboGap is a run that may stand after a wide gap: the letter of an option.
	uniboGap = regexp.MustCompile(`^\([B-E]\)`)
)

// uniboRules are the lines of the papers of a competition. The papers of the Giochi di Archimede are set in two columns.
func uniboRules(p hrPaper) official.HrRules {
	r := official.HrRules{Start: uniboStart, Stop: uniboStop, Skip: uniboSkip, Not: uniboNot, Letters: uniboLetters, Part: uniboPart, Gap: uniboGap, First: true, Roman: true, Accents: official.HrAccentsIt}
	switch p.kind {
	case "it-archimede":
		r.Section, r.Columns = uniboArchimedeSection, true
	case "it-febbraio":
		// A group of the problems of February starts at its own number, as 13 for the problems with a numeric answer.
		r.Section, r.First = uniboFebbraioSection, false
	case "it-classi-prime":
		// The papers of the classi prime up to 2020 are set in two columns; the later papers give the problems with their solutions across the page.
		r.Columns = p.year <= 2020
	}
	return r
}

// uniboPlace gives the positions of the problems of a paper and the names of their statements: problem N of the biennio and of the triennio of the Giochi di Archimede is bN and tN, and problem N of the other papers pN. A problem before the first part of a paper that has parts is an instruction of the paper; in the Giochi di Archimede the part named by the grid for the answers, "NOME COGNOME", keeps the paper of the part before it.
func uniboPlace(p hrPaper, probs []official.HrProblem) (ats, names []string) {
	parted := slices.ContainsFunc(probs, func(pr official.HrProblem) bool { return pr.Section != "" })
	variant := p.variant
	name := fmt.Sprintf("%s %d", uniboNames[p.kind], p.year)
	for _, pr := range probs {
		at := ""
		sec := strings.ToLower(strings.Join(strings.Fields(pr.Section), ""))
		switch {
		case parted && pr.Section == "":
		case p.kind == "it-archimede":
			if strings.HasPrefix(sec, "biennio") || strings.HasPrefix(sec, "triennio") {
				variant = sec[:strings.Index(sec, "io")+2]
			}
			if variant != "" {
				at = fmt.Sprintf("it-archimede/%d/%c%d", p.year, variant[0], pr.N)
			}
		default:
			at = fmt.Sprintf("%s/%d/p%d", p.kind, p.year, pr.N)
		}
		ats = append(ats, at)
		n := name
		if p.kind == "it-archimede" && variant != "" {
			n += ", gara " + variant
		}
		names = append(names, n)
	}
	return ats, names
}

// uniboHarvest is the harvest of the papers of the Italian olympiad.
var uniboHarvest = &paperHarvest{
	cmd:      "unibo",
	source:   uniboSource,
	codes:    uniboCodes,
	lang:     "it",
	language: "Italian",
	papers:   uniboPapers,
	rules:    uniboRules,
	place:    uniboPlace,
	fix:      uniboFix,
}

// uniboMarkRE is the star, the F of the AMS symbols, with its dot, that the papers of the classi prime set after the heading of the easier problems: no part of the problem.
var uniboMarkRE = regexp.MustCompile(`^\.?\s?\$\$?F\$\$?\s*\.?\s*`)

// uniboFix takes the star of an easier problem off the start of its text.
func uniboFix(p *official.HrProblem) {
	p.Text = uniboMarkRE.ReplaceAllString(p.Text, "")
}

func runUnibo(env Env, args []string) int {
	return uniboHarvest.run(env, args)
}
