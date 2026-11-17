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
	register(command{"ukmt", "read the free past papers of the UKMT from ukmt.org.uk and write their problems to uk-jmc, uk-imc, uk-smc, uk-jmo, uk-imoc and uk-mog (om ukmt -write)", runUkmt})
}

const (
	// ukmtSource is the free past papers of the UK Mathematics Trust; ukmtSitemap lists their pages, one page for each paper.
	ukmtSource  = "ukmt-free"
	ukmtSitemap = "https://ukmt.org.uk/free-past-papers-sitemap.xml"
)

// ukmtCodes are the competitions that om ukmt writes.
var ukmtCodes = []string{"uk-jmc", "uk-imc", "uk-smc", "uk-jmo", "uk-imoc", "uk-mog"}

var (
	// ukmtLocRE is a page of the sitemap.
	ukmtLocRE = regexp.MustCompile(`<loc>(https://ukmt\.org\.uk/free-past-papers/([a-z0-9-]+?)/?)</loc>`)
	// ukmtChallengeRE, ukmtOlympiadRE and ukmtGirlsRE are the names of the pages of the challenges and the junior olympiad, of the Cayley, Hamilton and Maclaurin olympiads, and of the olympiad for girls, as junior-mathematical-challenge-2024, cayley-mathematical-olympiad-2025 or mathematical-olympiad-for-girls-2014-2.
	ukmtChallengeRE = regexp.MustCompile(`^(junior|intermediate|senior)-mathematical-(challenge|olympiad)-(\d{4})(?:-\d)?$`)
	ukmtOlympiadRE  = regexp.MustCompile(`^(cayley|hamilton|maclaurin)(?:-mathematical)?(?:-olympiad)?-(\d{4})(?:-no-files)?$`)
	ukmtGirlsRE     = regexp.MustCompile(`^mathematical-olympiad-for-girls-(\d{4})(?:-\d)?$`)
	// ukmtPDFRE is a PDF of a page; ukmtNotPaperRE is a PDF that is no question paper: the solutions, the extended solutions, the reports of the markers and the samples.
	ukmtPDFRE      = regexp.MustCompile(`href="(https://ukmt\.org\.uk/wp-content/uploads/[^"]+\.pdf)"`)
	ukmtNotPaperRE = regexp.MustCompile(`(?i)sol|extended|report|sample|answer|mark|[-_]s(?:-\d+)?\.pdf$`)
	// ukmtYearRE is the year in the name of a PDF, which the page of the olympiad for girls of 2014 needs: it holds the paper of 2013.
	ukmtYearRE = regexp.MustCompile(`(?:^|[^0-9])((?:19|20)\d\d)(?:[^0-9]|$)`)
)

// ukmtChallenges are the competitions of the pages of the challenges and of the junior olympiad.
var ukmtChallenges = map[string]string{"junior-challenge": "uk-jmc", "intermediate-challenge": "uk-imc", "senior-challenge": "uk-smc", "junior-olympiad": "uk-jmo"}

// ukmtNames are the names of the papers in the names of the statements.
var ukmtNames = map[string]string{"uk-jmc": "Junior Mathematical Challenge", "uk-imc": "Intermediate Mathematical Challenge", "uk-smc": "Senior Mathematical Challenge", "uk-jmo": "Junior Mathematical Olympiad", "uk-mog": "Mathematical Olympiad for Girls", "cayley": "Intermediate Mathematical Olympiad, Cayley paper", "hamilton": "Intermediate Mathematical Olympiad, Hamilton paper", "maclaurin": "Intermediate Mathematical Olympiad, Maclaurin paper"}

// ukmtPaperOf reads the competition and the year of a page of the sitemap; the paper of the olympiads is the variant.
func ukmtPaperOf(slug string) (hrPaper, bool) {
	if m := ukmtChallengeRE.FindStringSubmatch(slug); m != nil {
		code, ok := ukmtChallenges[m[1]+"-"+m[2]]
		y, _ := strconv.Atoi(m[3])
		return hrPaper{kind: code, year: y}, ok
	}
	if m := ukmtOlympiadRE.FindStringSubmatch(slug); m != nil {
		y, _ := strconv.Atoi(m[2])
		return hrPaper{kind: "uk-imoc", variant: m[1], year: y}, true
	}
	if m := ukmtGirlsRE.FindStringSubmatch(slug); m != nil {
		y, _ := strconv.Atoi(m[1])
		return hrPaper{kind: "uk-mog", year: y}, true
	}
	return hrPaper{}, false
}

// ukmtPaperPDF is the question paper of a page: its one PDF that is no solutions, report or sample, or "" when it has none or more than one. The older papers of the challenges are images of the questions on the page, with no PDF.
func ukmtPaperPDF(page string) string {
	var out []string
	for _, m := range ukmtPDFRE.FindAllStringSubmatch(page, -1) {
		name := m[1][strings.LastIndex(m[1], "/")+1:]
		if ukmtNotPaperRE.MatchString(name) || slices.Contains(out, m[1]) {
			continue
		}
		out = append(out, m[1])
	}
	if len(out) != 1 {
		return ""
	}
	return out[0]
}

// ukmtPapers reads the pages of the papers from the sitemap and the question paper from each page. A paper whose file names a year is of that year, as ukmog-questions-2013.pdf on the page of 2014; two pages that give the same paper count once.
func ukmtPapers(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error) {
	b, err := hrGet(ctx, fc, ukmtSource, ukmtSitemap, opt)
	if err != nil {
		return nil, err
	}
	var out []hrPaper
	for _, m := range ukmtLocRE.FindAllStringSubmatch(string(b), -1) {
		p, ok := ukmtPaperOf(m[2])
		if !ok {
			continue
		}
		page, err := hrGet(ctx, fc, ukmtSource, m[1], opt)
		if err != nil {
			return nil, err
		}
		if p.url = ukmtPaperPDF(string(page)); p.url == "" {
			continue
		}
		if y := ukmtYearRE.FindStringSubmatch(p.url[strings.LastIndex(p.url, "/")+1:]); y != nil {
			p.year, _ = strconv.Atoi(y[1])
		}
		if slices.ContainsFunc(out, func(q hrPaper) bool { return q.kind == p.kind && q.variant == p.variant && q.year == p.year }) {
			continue
		}
		out = append(out, p)
	}
	slices.SortFunc(out, func(a, b hrPaper) int {
		if c := strings.Compare(a.kind+a.variant, b.kind+b.variant); c != 0 {
			return c
		}
		return a.year - b.year
	})
	return out, nil
}

// ukmtRules are the lines of the papers. The first page of a paper holds its numbered instructions, so the problems start on the second page. A problem starts at its number, as "7.", "A7." in the junior olympiad, whose Section A and Section B are its parts, or "C1." in the older Cayley papers; the number may stand on a line of its own. The header and the footer of a page, the title and the date of the paper, are skipped.
var ukmtRules = official.HrRules{
	Start:     regexp.MustCompile(`^([ABCHM]?(\d{1,2}))\.(?:\s|$)`),
	Section:   regexp.MustCompile(`^Section ([AB])\s*$`),
	Stop:      regexp.MustCompile(`(?i)^(?:end of (?:the )?paper|turn over|please turn over|\(?continued|try to complete section|your solutions to section|advice to candidates)`),
	Skip:      regexp.MustCompile(`(?i)^(?:©|\(c\) |uk mathematics trust|the united kingdom mathematics trust|www\.ukmt\.org\.uk|(?:uk )?(?:junior|intermediate|senior) mathematical (?:challenge|olympiad)|mathematical olympiad for girls|(?:olympiad )?(?:cayley|hamilton|maclaurin)(?: paper)?|(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday) \d|\d{1,2}$)`),
	Letters:   "–—‘’“”×−÷°…·£€",
	First:     true,
	FirstPage: 2,
	// The options of a question, as "A 12   B 24   C 36   D 72   E 96", or "D" and "E" on a line of their own, are a paragraph, and their letters stand after wide gaps.
	Gap:   regexp.MustCompile(`^[B-E](?:\s|$)`),
	Part:  regexp.MustCompile(`^(?:(?:\(?[a-z]\)|\([ivx]+\)|-)\s|A \S.* B \S.* C \S|D \S.* E \S)`),
	Shift: regexp.MustCompile(`^(?:[A-Z]{6}\+)?txmiaX$`),
	// The math of the papers made with LaTeX is set in the NewTX fonts, with the Nimbus italic for its letters; their text is in TeX Gyre and the NewTX text italic. The older papers are set in Times with the math of RO_MathPhys.
	Math: regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:NewTXB?MI\d*|r?n?txb?mi\d*|txmiaX|txsy[a-z]*|rtxr|NimbusRomNo9L-(?:Regu|Medi)Ital|RO_MathPhys(?:\.Italic)?)$`),
	Text: regexp.MustCompile(`^(?:[A-Z]{6}\+)?(?:TeXGyre(?:Termes|Heros)X?(?:-\w+)?|ntxtm[a-z]*|NimbusRomNo9L-(?:Regu|Medi))$`),
}

// ukmtPlace gives the positions of the problems of a paper and the names of their statements. Question N of a challenge and of the olympiad for girls is pN; the junior olympiad of 2015 to 2024 has its Section A and Section B, aN and bN, and from 2025 six problems, pN; the Cayley, Hamilton and Maclaurin papers are cN, hN and mN of uk-imoc.
func ukmtPlace(p hrPaper, probs []official.HrProblem) (ats, names []string) {
	key := p.kind
	if p.kind == "uk-imoc" {
		key = p.variant
	}
	name := fmt.Sprintf("UKMT, %s %d", ukmtNames[key], p.year)
	for _, pr := range probs {
		at := ""
		switch {
		case p.kind == "uk-imoc":
			at = fmt.Sprintf("uk-imoc/%d/%c%d", p.year, p.variant[0], pr.N)
		case p.kind == "uk-jmo" && p.year <= 2024:
			if pr.Section != "" && strings.HasPrefix(pr.Code, pr.Section) {
				at = fmt.Sprintf("uk-jmo/%d/%s%d", p.year, strings.ToLower(pr.Section), pr.N)
			}
		default:
			at = fmt.Sprintf("%s/%d/p%d", p.kind, p.year, pr.N)
		}
		ats = append(ats, at)
		names = append(names, name)
	}
	return ats, names
}

// ukmtHarvest is the harvest of the free past papers of the UKMT.
var ukmtHarvest = &paperHarvest{
	cmd:      "ukmt",
	source:   ukmtSource,
	codes:    ukmtCodes,
	lang:     "en",
	language: "English",
	papers:   ukmtPapers,
	rules:    func(hrPaper) official.HrRules { return ukmtRules },
	place:    ukmtPlace,
}

func runUkmt(env Env, args []string) int {
	return ukmtHarvest.run(env, args)
}
