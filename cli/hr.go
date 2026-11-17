package cli

import (
	"bytes"
	"context"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/ocr"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"hr", "read the papers of the Croatian competitions from natjecanja.math.hr and write their problems to hr-city, hr-county, hr-state, hr-mo, hr-tst, hr-jmo, hr-girls, hr-cadets, hr-cadets-r1 and hr-medo (om hr -write)", runHr})
}

const (
	// hrSource is the site of the competitions of the Croatian Mathematical Society.
	hrSource = "natjecanja-math-hr"
	hrSite   = "https://natjecanja.math.hr/"
)

// hrCodes are the competitions that om hr writes.
var hrCodes = []string{"hr-city", "hr-county", "hr-state", "hr-mo", "hr-tst", "hr-jmo", "hr-girls", "hr-cadets", "hr-cadets-r1", "hr-medo"}

// hrPaper is one paper PDF of the site: kind is the competition of the page that links it (hr-city, hr-county, hr-state, hr-mo, hr-jmo, hr-girls, hmok or medo), year its year, or the year in which the season of MEDO starts, and variant the column of the table (o for the primary school, a and b for the variants of the secondary school) or the round (prvo, drugo, ...). A paper that a zip file of a site holds has the url of the zip file and the name of the PDF in it as its member.
type hrPaper struct {
	kind, variant, url, member string
	year                       int
}

var (
	// hrHeadingRE splits a page at its headings; hrRowRE, hrCellRE and hrHrefRE read the rows of its tables.
	hrHeadingRE = regexp.MustCompile(`<h[1-6][^>]*>`)
	hrRowRE     = regexp.MustCompile(`(?s)<tr[^>]*>(.*?)</tr>`)
	hrCellRE    = regexp.MustCompile(`(?s)<t[dh][^>]*>(.*?)</t[dh]>`)
	hrHrefRE    = regexp.MustCompile(`href="([^"]+\.pdf)"`)
	hrTagRE     = regexp.MustCompile(`<[^>]+>`)
	hrYearRE    = regexp.MustCompile(`^\s*(\d{4})`)
	// hrRoundRE is the file of a paper of the olympiad for cadets or of MEDO, as HMOK-2022-prvo_kolo-zadaci.pdf or MEDO-2024_2025-drugo_kolo-zadaci.pdf.
	hrRoundRE = regexp.MustCompile(`/(HMOK|MEDO)-(\d{4})(?:_\d{4})?-([a-z]+)_kolo-zadaci\.pdf$`)
)

// hrPlain is the text of a fragment of a page.
func hrPlain(s string) string {
	return strings.TrimSpace(html.UnescapeString(hrTagRE.ReplaceAllString(s, " ")))
}

// hrArchive reads the papers of the school, county and state rounds from the archive page: a table for each round, after its heading, with a row for each year and the columns of the primary school and of the A and B variants.
func hrArchive(page string) []hrPaper {
	var out []hrPaper
	kind := ""
	for _, part := range hrHeadingRE.Split(page, -1) {
		h := hrPlain(part[:min(len(part), 200)])
		switch {
		case strings.HasPrefix(h, "Školska"):
			kind = "hr-city"
		case strings.HasPrefix(h, "Županijska"):
			kind = "hr-county"
		case strings.HasPrefix(h, "Državna"):
			kind = "hr-state"
		case strings.HasPrefix(h, "Slika"):
			kind = ""
		}
		if kind == "" {
			continue
		}
		for _, row := range hrRowRE.FindAllStringSubmatch(part, -1) {
			cells := hrCellRE.FindAllStringSubmatch(row[1], -1)
			if len(cells) < 2 {
				continue
			}
			y := hrYearRE.FindStringSubmatch(hrPlain(cells[0][1]))
			if y == nil {
				continue
			}
			year, _ := strconv.Atoi(y[1])
			for k, v := range []string{"o", "a", "b"} {
				if k+1 >= len(cells) {
					break
				}
				if m := hrHrefRE.FindStringSubmatch(cells[k+1][1]); m != nil {
					out = append(out, hrPaper{kind: kind, year: year, variant: v, url: html.UnescapeString(m[1])})
				}
			}
		}
	}
	return out
}

// hrOlympiads reads the papers of the olympiads from the page of the selection competitions: a row for each year, with the columns of the HMO, of the olympiad for girls and of the junior olympiad.
func hrOlympiads(page string) []hrPaper {
	var out []hrPaper
	for _, row := range hrRowRE.FindAllStringSubmatch(page, -1) {
		cells := hrCellRE.FindAllStringSubmatch(row[1], -1)
		if len(cells) < 2 {
			continue
		}
		y := hrYearRE.FindStringSubmatch(hrPlain(cells[0][1]))
		if y == nil {
			continue
		}
		year, _ := strconv.Atoi(y[1])
		for k, kind := range []string{"hr-mo", "hr-girls", "hr-jmo"} {
			if k+1 >= len(cells) {
				break
			}
			if m := hrHrefRE.FindStringSubmatch(cells[k+1][1]); m != nil {
				out = append(out, hrPaper{kind: kind, year: year, variant: "p", url: html.UnescapeString(m[1])})
			}
		}
	}
	return out
}

// hrRounds reads the papers of the olympiad for cadets or of MEDO from their pages: the files of the problems of each round.
func hrRounds(page string) []hrPaper {
	var out []hrPaper
	for _, m := range hrHrefRE.FindAllStringSubmatch(page, -1) {
		u := html.UnescapeString(m[1])
		r := hrRoundRE.FindStringSubmatch(u)
		if r == nil || slices.ContainsFunc(out, func(p hrPaper) bool { return p.url == u }) {
			continue
		}
		year, _ := strconv.Atoi(r[2])
		kind := "hmok"
		if r[1] == "MEDO" {
			kind = "medo"
		}
		out = append(out, hrPaper{kind: kind, year: year, variant: r[3], url: u})
	}
	return out
}

var (
	// hrDate is the date of a round, as "1. ožujka 2023.", which ends the header of a paper: a date of the first of a month looks like the heading of a problem 1.
	hrDate = `\d{1,2}\.\s*(?:siječnja|veljače|ožujka|travnja|svibnja|lipnja|srpnja|kolovoza|rujna|listopada|studenoga?|prosinca)\s+\d{4}\.?$`
	// hrTail is a note on the points or the rules that a paper sets on the line of the end of a problem.
	hrTail = regexp.MustCompile(`\s*(?:Prvih \S+ zadataka (?:vrijedi|boduje)|Svaki (?:se )?zadatak (?:se )?(?:vrijedi|boduje)|Nije dopuštena uporaba|MINISTARSTVO ZNANOSTI|Okreni(?:te)? list)`)
	// hrGradeRules shape the papers of the school, county and state rounds: a section for each grade, "4. razred – osnovna škola" or "1. razred – srednja škola – A varijanta", whose problems are "1." to "5.". Some papers set the š of škola as ö.
	hrGradeRules = official.HrRules{
		First:   true,
		Start:   regexp.MustCompile(`^(\d{1,2})\.(?:\s|$)`),
		Section: regexp.MustCompile(`^(\d)\.\s*razred\s*[-–]\s*(?:osnovna|srednja)\s+(?:škola|ökola)`),
		Stop:    regexp.MustCompile(`^(?:Svaki (?:od \S+ )?zadat(?:ak|ka) (?:vrijedi|boduje|donosi)|Zadat?ci za \d+ bod|Zadat?ci (?:od|vrijede)|Osim kona[čc]n|Ukoliko|U svim zadatcima|ŠKOLSKO|GRADSKO|ŽUPANIJSKO|DRŽAVNO|Ministarstvo|MINISTARSTVO|Agencija za odgoj|Hrvatsko matemati[čc]ko|ZADAT?CI|Svaki se zadatak|Prvih \S+ zadataka|Nije dopušten|\d{1,2}\.\s*[–-]\s*\d{1,2}\.|` + hrDate + `)`),
		Skip:    regexp.MustCompile(`^(?:Okreni(?:te)? (?:list|stranicu|papir)\s*!?|\d{1,2}|\d+\s*/\s*\d+)$`),
		Tail:    hrTail,
	}
	// hrHMORules shape the books of the HMO, which give each problem as "Zadatak 3." on a line of its own, and then its solutions; the parts are the days or the tests, as "Prvi dan" or "Završni test za izbor IMO ekipe".
	hrHMORules = official.HrRules{
		First:   true,
		Start:   regexp.MustCompile(`^Zadatak (\d{1,2})\.?$`),
		Section: regexp.MustCompile(`^(?:(?:Prvi|Drugi|Treći|Četvrti|prvi|drugi|treći|četvrti) (?:dan|test)|Završni test za izbor (?:IMO|MEMO) ekipe|IZBORNO NATJECANJE \d{4}\.\s*[–-]\s*(?:prvi dio|drugi dio|Izbor IMO ekipe|Izbor MEMO ekipe))$`),
		Stop:    regexp.MustCompile(`^(?:(?:Prvo|Drugo|Treće|Četvrto|Peto|Alternativno|Drugačije)?\s*[Rr]ješenje|Rješenja|Napomena|Komentar|Uputa|HRVATSKA MATEMATIČKA OLIMPIJADA|IZBORNO NATJECANJE)`),
		Skip:    regexp.MustCompile(`^(?:Hrvatska matematička olimpijada|Izborno natjecanje|Izborna natjecanja)\s.*\d+\s*/\s*\d+$|^\d+\s*/\s*\d+$|^\d{1,2}$`),
	}
	// hrJMORules shape the papers of the junior olympiad and of the olympiad for girls: problems "1." to "4.", in one test or in two days, as "Prvi dan" or "Drugi test".
	hrJMORules = official.HrRules{
		First:   true,
		Start:   regexp.MustCompile(`^(\d)\.(?:\s|$)`),
		Section: regexp.MustCompile(`^(?:Test za izbor JBMO ekipe\s*[–-]\s*)?(?:Prvi|Drugi|prvi|drugi) (?:dan|test)$`),
		Stop:    regexp.MustCompile(`^(?:Vrijeme rješavanja|Svaki zadatak vrijedi|Redoslijed zadataka|Zadatke nije|\d*\.?\s*HRVATSKA|\d+\.\s*Hrvatska|Hrvatsko matematičko|` + hrDate + `)`),
		Skip:    regexp.MustCompile(`^(?:Okreni(?:te)? (?:list|stranicu)\s*!?|\d{1,2})$`),
		Tail:    hrTail,
	}
	// hrCadetRules shape the papers of the olympiad for cadets: a section for each grade or group, "Zadatci za 4. razred" or "Zadatci za grupu A (4./5. razred)", whose problems are "1." to "10.".
	hrCadetRules = official.HrRules{
		First:   true,
		Start:   regexp.MustCompile(`^(\d{1,2})\.(?:\s|$)`),
		Section: regexp.MustCompile(`^Zadatci za (?:(\d)\. razred|grupu ([A-Z]))`),
		Stop:    regexp.MustCompile(`^(?:Hrvatsko(?: matematičko)?(?: društvo)?$|matematičko$|društvo$|HRVATSKA MATEMATIČKA OLIMPIJADA|(?:prvo|drugo) kolo|Svaki zadatak|Osim konačn|sva rješenja|Rješenje svakog|Vrijeme rješavanja|U svakom zadatku|` + hrDate + `)`),
		Skip:    regexp.MustCompile(`^(?:Okreni(?:te)? list\s*!?|\d{1,2})$`),
		Tail:    hrTail,
	}
	// hrMEDORules shape the papers of MEDO: "Zadatak J-2.3. [30 bodova]", the problem 3 of the round 2 of the juniors, with a part for each level.
	hrMEDORules = official.HrRules{
		First:   true,
		Start:   regexp.MustCompile(`^Zadatak ([KJS]-\d\.(\d{1,2}))\.\s*\[\d+ bod(?:ova|a)\]`),
		Section: regexp.MustCompile(`^MEDO (\d{4})\./`),
		Stop:    regexp.MustCompile(`^(?:(?:Prvo|Drugo|Treće|Četvrto|Peto) kolo$|\d{1,2}\.\s*\d{1,2}\.\s*\d{4}\.$|(?:Kadeti|Juniori|Seniori) \(|Zadaci$)`),
	}
	// hrMEDOLabelRE reads the label of a problem of MEDO: the level, the round and the number.
	hrMEDOLabelRE = regexp.MustCompile(`^([KJS])-(\d)\.(\d{1,2})$`)
	// hrTestRE is a part of the book of the HMO that is a selection test for a team.
	hrTestRE = regexp.MustCompile(`(IMO|MEMO) ekipe`)
)

// hrRulesOf gives the rules of the papers of a kind.
func hrRulesOf(p hrPaper) official.HrRules {
	switch p.kind {
	case "hr-mo":
		return hrHMORules
	case "hr-jmo", "hr-girls":
		return hrJMORules
	case "hmok":
		r := hrCadetRules
		r.Titled = p.variant == "prvo"
		return r
	case "medo":
		return hrMEDORules
	}
	return hrGradeRules
}

// hrTitles are the names of the papers of each kind on the site.
var hrTitles = map[string]string{
	"hr-city":   "Školsko/gradsko natjecanje iz matematike",
	"hr-county": "Županijsko natjecanje iz matematike",
	"hr-state":  "Državno natjecanje iz matematike",
	"hr-mo":     "Hrvatska matematička olimpijada",
	"hr-jmo":    "Hrvatska juniorska matematička olimpijada",
	"hr-girls":  "Hrvatska matematička olimpijada za djevojke",
	"hmok":      "Hrvatska matematička olimpijada za kadete",
}

// hrRead is one problem that om hr read, with its position, the paper and the name of its part.
type hrRead struct {
	at    string
	paper hrPaper
	name  string
	prob  official.HrProblem
	flags []string
	conf  float64
}

// hrPlace gives the positions of the problems of a paper, or "" for a problem with no place, and the name of the statement of each. The parts of a book of the HMO are its days, a to d in their order, and the selection tests for the IMO and the MEMO team, which are hr-tst; a problem of a part is counted from 1.
func hrPlace(p hrPaper, probs []official.HrProblem) (ats, names []string) {
	title := hrTitles[p.kind]
	var days []string
	count := map[string]int{}
	for _, pr := range probs {
		at, name := "", ""
		count[pr.Section]++
		k := count[pr.Section]
		switch p.kind {
		case "hr-city", "hr-county", "hr-state":
			if g, err := strconv.Atoi(pr.Section); err == nil && pr.Code == strconv.Itoa(pr.N) {
				at = fmt.Sprintf("%s/%d/%s%d%d", p.kind, p.year, p.variant, g, pr.N)
				school := "osnovna škola"
				if p.variant != "o" {
					school = "srednja škola, " + strings.ToUpper(p.variant) + " varijanta"
				}
				name = fmt.Sprintf("%s %d, %d. razred, %s", title, p.year, g, school)
			}
		case "hr-mo":
			if m := hrTestRE.FindStringSubmatch(pr.Section); m != nil {
				at = fmt.Sprintf("hr-tst/%d/%s%d", p.year, strings.ToLower(m[1][:1]), k)
			} else {
				if !slices.Contains(days, pr.Section) {
					days = append(days, pr.Section)
				}
				at = fmt.Sprintf("hr-mo/%d/%c%d", p.year, 'a'+slices.Index(days, pr.Section), k)
			}
			name = fmt.Sprintf("%s %d, %s", title, p.year, pr.Section)
		case "hr-jmo", "hr-girls":
			name = fmt.Sprintf("%s %d", title, p.year)
			if pr.Section == "" {
				at = fmt.Sprintf("%s/%d/p%d", p.kind, p.year, k)
				break
			}
			if !slices.Contains(days, pr.Section) {
				days = append(days, pr.Section)
			}
			at = fmt.Sprintf("%s/%d/%c%d", p.kind, p.year, 'a'+slices.Index(days, pr.Section), k)
			name += ", " + pr.Section
		case "hmok":
			code := "hr-cadets"
			if p.variant == "prvo" {
				code = "hr-cadets-r1"
			}
			name = fmt.Sprintf("%s %d, %s kolo", title, p.year, p.variant)
			switch {
			case len(pr.Section) == 1 && pr.Section >= "0" && pr.Section <= "9":
				// The first round has ten problems, so its numbers have two digits, as o401 to o410 for grade 4.
				num := strconv.Itoa(pr.N)
				if p.variant == "prvo" {
					num = fmt.Sprintf("%02d", pr.N)
				}
				at = fmt.Sprintf("%s/%d/o%s%s", code, p.year, pr.Section, num)
				name += ", " + pr.Section + ". razred"
			case len(pr.Section) == 1:
				at = fmt.Sprintf("%s/%d/%s%d", code, p.year, strings.ToLower(pr.Section), pr.N)
				name += ", grupa " + pr.Section
			}
		case "medo":
			if m := hrMEDOLabelRE.FindStringSubmatch(pr.Code); m != nil {
				at = fmt.Sprintf("hr-medo/%d/%s%s%s", p.year, strings.ToLower(m[1]), m[2], m[3])
				name = fmt.Sprintf("MEDO %d./%d., %s. kolo, zadatak %s", p.year, p.year+1, m[2], pr.Code)
			}
		}
		ats = append(ats, at)
		names = append(names, "natjecanja.math.hr, "+name)
	}
	return ats, names
}

// hrHarvest is the harvest of natjecanja.math.hr.
var hrHarvest = &paperHarvest{
	cmd:      "hr",
	source:   hrSource,
	codes:    hrCodes,
	lang:     "hr",
	language: "Croatian",
	papers: func(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error) {
		var papers []hrPaper
		for _, pg := range []struct {
			path string
			read func(string) []hrPaper
		}{{"arhiva-domacih-natjecanja/", hrArchive}, {"izborna-natjecanja-i-olimpijade/", hrOlympiads}, {"hmok/", hrRounds}, {"medo/", hrRounds}} {
			b, err := hrGet(ctx, fc, hrSource, hrSite+pg.path, opt)
			if err != nil {
				return nil, err
			}
			papers = append(papers, pg.read(string(b))...)
		}
		return papers, nil
	},
	rules: hrRulesOf,
	place: hrPlace,
	fix: func(p *official.HrProblem) {
		p.Text = hrPointsRE.ReplaceAllString(p.Text, "")
	},
}

// hrPointsRE is the points of a problem before its text, as "(10 bodova)" in the papers of the B variant.
var hrPointsRE = regexp.MustCompile(`^\(\d+ bod(?:ova|a)\)\s*`)

func runHr(env Env, args []string) int {
	return hrHarvest.run(env, args)
}

// paperHarvest is a harvest of the paper PDFs of a site whose problems HrPDF reads: the command, the source, the competitions it writes, the language of the papers, and the functions that list the papers, give the rules of a paper and place its problems.
type paperHarvest struct {
	cmd, source, lang, language string
	codes                       []string
	papers                      func(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error)
	rules                       func(hrPaper) official.HrRules
	place                       func(hrPaper, []official.HrProblem) (ats, names []string)
	// fix, if any, mends the text of each problem that HrPDF read.
	fix func(*official.HrProblem)
}

func (h *paperHarvest) run(env Env, args []string) int {
	fs := newFlags(h.cmd, env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	only := fs.String("only", "", "write only the problems of these competitions, a list with commas")
	write := fs.Bool("write", false, "write the problems and the official papers of the editions")
	explain := fs.Bool("explain", false, "print each paper with the problems of its parts, and each problem with its position and its flags")
	maxAge := fs.Duration("max-age", 30*24*time.Hour, "use a capture of the raw store that is younger than this, with no request")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om %s: the content has %d errors. Run om check first.\n", h.cmd, findings.Errors())
		return ExitFindings
	}
	src := corpus.Source(h.source)
	if src == nil {
		fmt.Fprintf(env.Stderr, "om %s: the content has no source %s\n", h.cmd, h.source)
		return ExitUsage
	}
	want := map[string]bool{}
	for _, c := range strings.Split(*only, ",") {
		if c = strings.TrimSpace(c); c != "" {
			want[c] = true
		}
	}
	for _, c := range h.codes {
		if (len(want) == 0 || want[c]) && corpus.Competition(c) == nil {
			fmt.Fprintf(env.Stderr, "om %s: the content has no competition %s\n", h.cmd, c)
			return ExitUsage
		}
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om %s: %v\n", h.cmd, err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	fc.Delay = 2 * time.Second
	ctx := context.Background()
	opt := fetch.Options{MaxAge: *maxAge}
	papers, err := h.papers(ctx, fc, opt)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om %s: %v\n", h.cmd, err)
		return ExitFailure
	}
	code := ExitOK
	var reads []*hrRead
	editions := map[string][]string{}
	for _, p := range papers {
		probs, err := hrRun(ctx, fc, h.source, p, h.rules(p), opt)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om %s: %s: %v\n", h.cmd, p.url, err)
			code = ExitFailure
			continue
		}
		if h.fix != nil {
			for i := range probs {
				h.fix(&probs[i])
			}
		}
		ats, names := h.place(p, probs)
		if *explain {
			var parts []string
			for i, pr := range probs {
				if i == 0 || pr.Section != probs[i-1].Section {
					parts = append(parts, fmt.Sprintf("%q:", pr.Section))
				}
				parts[len(parts)-1] += " " + pr.Code
			}
			fmt.Fprintf(env.Stdout, "PAPER\t%s\t%d\t%s\t%s\t%d\t%s\n", p.kind, p.year, p.variant, p.url, len(probs), strings.Join(parts, " "))
		}
		seen := map[string]bool{}
		for i, pr := range probs {
			if ats[i] == "" || seen[ats[i]] {
				fmt.Fprintf(env.Stderr, "om %s: %s: problem %s of %q has no place or is twice\n", h.cmd, p.url, pr.Code, pr.Section)
				continue
			}
			seen[ats[i]] = true
			c := ats[i][:strings.Index(ats[i], "/")]
			if len(want) > 0 && !want[c] {
				continue
			}
			flags := append(ocr.Check(pr.Text), pr.Flags...)
			if pr.Figure {
				flags = append(flags, ocr.FlagFigure)
			}
			if pr.Text == "" {
				flags = append(flags, "empty")
			}
			r := &hrRead{at: ats[i], paper: p, name: names[i], prob: pr, flags: flags, conf: ocr.Confidence(flags)}
			if pr.Text == "" {
				r.conf = 0
			}
			reads = append(reads, r)
			ed := r.at[:strings.LastIndex(r.at, "/")]
			if !slices.Contains(editions[ed], p.url) {
				editions[ed] = append(editions[ed], p.url)
			}
		}
	}
	// A problem that two parts of an edition share, as the papers of two grades or two levels, is one record with both positions.
	var groups [][]*hrRead
	byText := map[string]int{}
	for _, r := range reads {
		ed := r.at[:strings.LastIndex(r.at, "/")]
		key := ed + "\x00" + strings.Join(strings.Fields(r.prob.Text), " ")
		if g, ok := byText[key]; ok && len(r.prob.Text) > 40 {
			groups[g] = append(groups[g], r)
			continue
		}
		byText[key] = len(groups)
		groups = append(groups, []*hrRead{r})
	}
	owners := official.Owners(corpus)
	allowed := textAllowed(src)
	type record struct{ ms []*hrRead }
	var news []record
	stats := map[string]*[4]int{}
	stat := func(at string) *[4]int {
		c := at[:strings.Index(at, "/")]
		if stats[c] == nil {
			stats[c] = &[4]int{}
		}
		return stats[c]
	}
	reasons := map[string]int{}
	linked := 0
	for _, ms := range groups {
		best := ms[0]
		if *explain {
			for _, m := range ms {
				fmt.Fprintf(env.Stdout, "%s\t%s\t%.2f\t%s\t%s\n", m.at, m.prob.Code, m.conf, strings.Join(m.flags, ","), strings.ReplaceAll(m.prob.Text, "\n", " ¶ "))
			}
		}
		var owned *content.Problem
		var free []string
		for _, m := range ms {
			if p := owners[m.at]; p != nil {
				if owned == nil {
					owned = p
				}
				continue
			}
			free = append(free, m.at)
		}
		if owned != nil {
			// A record at a position gets a link to the paper and the other positions; its text does not change.
			if slices.ContainsFunc(owned.Statements, func(s content.Statement) bool { return s.URL == best.paper.url }) {
				continue
			}
			linked++
			stat(best.at)[3]++
			if !*write {
				continue
			}
			if err := hrLink(*root, owned, best, src, h.lang, free); err != nil {
				fmt.Fprintf(env.Stderr, "om %s: %s: %v\n", h.cmd, owned.Path, err)
				code = ExitFailure
			}
			continue
		}
		news = append(news, record{ms: ms})
		s := stat(ms[0].at)
		if allowed && best.conf >= ocr.Threshold {
			s[0]++
		} else {
			s[1]++
			for _, f := range best.flags {
				reasons[f]++
			}
		}
		s[2] += len(ms) - 1
	}
	var cs []string
	for c := range stats {
		cs = append(cs, c)
	}
	slices.Sort(cs)
	for _, c := range cs {
		s := stats[c]
		fmt.Fprintf(env.Stdout, "%s: %d new records with text, %d link-only, %d more positions of them, %d links to existing records\n", c, s[0], s[1], s[2], s[3])
	}
	var rs []string
	for f, n := range reasons {
		rs = append(rs, fmt.Sprintf("%s %d", f, n))
	}
	slices.Sort(rs)
	fmt.Fprintf(env.Stdout, "%d papers, %d problems, %d records, %d links. The flags of the link-only records: %s\n", len(papers), len(reads), len(news), linked, strings.Join(rs, ", "))
	if !*write {
		return code
	}
	tags, err := mintTags(corpus, len(news))
	if err != nil {
		fmt.Fprintf(env.Stderr, "om %s: %v\n", h.cmd, err)
		return ExitFailure
	}
	for i, n := range news {
		if err := hrWrite(*root, corpus, src, h.lang, allowed, tags[i], n.ms); err != nil {
			fmt.Fprintf(env.Stderr, "om %s: %s: %v\n", h.cmd, n.ms[0].at, err)
			code = ExitFailure
		}
	}
	var eds []string
	for ed := range editions {
		eds = append(eds, ed)
	}
	slices.Sort(eds)
	for _, ed := range eds {
		if err := hrOfficial(*root, corpus, h.source, h.language, h.lang, ed, editions[ed]); err != nil {
			fmt.Fprintf(env.Stderr, "om %s: %s: %v\n", h.cmd, ed, err)
			code = ExitFailure
		}
	}
	return code
}

// hrGet gets a URL of the site through the fetch client, or from the raw store when its capture is young enough.
func hrGet(ctx context.Context, fc *fetch.Client, source, u string, opt fetch.Options) ([]byte, error) {
	res, err := fc.Get(ctx, source, u, opt)
	if err != nil {
		return nil, err
	}
	rc, err := fc.Store.Get(ctx, res.Capture.Hash)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rc.Close() }()
	var b bytes.Buffer
	_, err = b.ReadFrom(rc)
	return b.Bytes(), err
}

// hrRun reads the problems of a paper PDF with the rules of its kind, and finds its drawings on the pages that pdftoppm renders.
func hrRun(ctx context.Context, fc *fetch.Client, source string, p hrPaper, rules official.HrRules, opt fetch.Options) ([]official.HrProblem, error) {
	raw, err := hrGet(ctx, fc, source, p.url, opt)
	if err != nil {
		return nil, err
	}
	if p.member != "" {
		if raw, err = zipMember(raw, p.member); err != nil {
			return nil, err
		}
	}
	dir, err := os.MkdirTemp("", "om-hr-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	f := filepath.Join(dir, "paper.pdf")
	if err := os.WriteFile(f, raw, 0o600); err != nil {
		return nil, err
	}
	b, err := figureRunner.Run(ctx, official.RuPDFTool, "-xml", "-i", "-stdout", "-q", f)
	if err != nil {
		return nil, err
	}
	items := official.ParsePDFXML(b)
	ink := demoInk(ctx, f, items)
	if rules.Columns {
		ink = columnInk(ctx, f, items)
	}
	return official.HrPDF(items, ink, rules), nil
}

// hrCandidate is the statement of the paper of r.
func hrCandidate(r *hrRead, src *content.SourceRecord, lang string) *canon.Candidate {
	return &canon.Candidate{At: r.at, Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: lang, Name: r.name, URL: r.paper.url}
}

// hrWrite writes a new record for the problem at the positions of ms, with the text of the first when its confidence is high enough, or a link only.
func hrWrite(root string, corpus *content.Corpus, src *content.SourceRecord, lang string, allowed bool, tag string, ms []*hrRead) error {
	text := ms[0]
	at := text.at
	comp := corpus.Competition(at[:strings.Index(at, "/")])
	s := canon.Scan{Page: text.prob.Page, Text: text.prob.Text, Extraction: "native/" + official.RuPDFTool + " (" + official.HrVersion + ")", Confidence: text.conf}
	if !allowed || text.conf < ocr.Threshold {
		s = canon.Scan{Page: text.prob.Page}
	}
	_, rule := canon.Rank(&canon.Candidate{Origin: src.Origin, Lang: lang}, original(comp))
	b, err := canon.RenderScan(at, tag, hrCandidate(text, src, lang), s, rule)
	if err != nil {
		return err
	}
	rec := &canon.Record{Primary: canon.Choice{At: at}}
	var more []string
	for _, m := range ms[1:] {
		more = append(more, m.at)
		rec.Merged = append(rec.Merged, canon.Merge{At: m.at})
	}
	if len(more) > 0 {
		if b, err = canon.AddAppearances(b, more...); err != nil {
			return err
		}
	}
	return writeNew(root, corpus, rec, tag, b)
}

// hrLink adds to an existing record a link to the paper of r, with no text, and the positions free.
func hrLink(root string, p *content.Problem, r *hrRead, src *content.SourceRecord, lang string, free []string) error {
	full := filepath.Join(root, filepath.FromSlash(p.Path))
	b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	if b, err = canon.Apply(b, canon.AddLink(p, hrCandidate(r, src, lang), r.prob.Page)); err != nil {
		return err
	}
	if len(free) > 0 {
		if b, err = canon.AddAppearances(b, free...); err != nil {
			return err
		}
	}
	return os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
}

// hrOfficial adds the paper PDFs urls of the source to the official papers of an edition, as comp/year, in the language of the papers.
func hrOfficial(root string, corpus *content.Corpus, source, language, lang, ed string, urls []string) error {
	if _, err := newEdition(root, corpus, ed+"/p1"); err != nil {
		return err
	}
	file := filepath.Join(root, "editions", filepath.FromSlash(ed)+".yaml")
	list, err := officialOf(file)
	if err != nil {
		return err
	}
	comp := corpus.Competition(ed[:strings.Index(ed, "/")])
	from := ""
	if len(comp.Sources) > 0 && comp.Sources[0] != source {
		from = source
	}
	n := len(list)
	for _, u := range urls {
		if !slices.ContainsFunc(list, func(o content.OfficialPaper) bool { return o.URL == u }) {
			list = append(list, content.OfficialPaper{Lang: language, Code: lang, URL: u, Source: from})
		}
	}
	if len(list) == n {
		return nil
	}
	return setOfficial(file, list)
}
