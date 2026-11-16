package cli

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/text/encoding/charmap"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"skmo", "read the Slovak papers of the Czech and Slovak olympiad from skmo.sk and write their problems to cz-sk-mo, cz-sk-mo-r1, cz-sk-mo-r2 and cz-sk-mo-z (om skmo -write)", runSkmo})
}

const (
	// skmoSource is the site of the Slovak commission of the olympiad.
	skmoSource = "skmo"
	skmoSite   = "https://skmo.sk/"
	// skmoFirst is the first olympiad whose page has papers: the pages of the 30th and the 40th olympiads have none, the 45th and the 48th a leaflet, and the 49th (1999/2000) the papers of categories A, B and C.
	skmoFirst = 45
)

// skmoCodes are the competitions that om skmo writes.
var skmoCodes = []string{"cz-sk-mo", "cz-sk-mo-r1", "cz-sk-mo-r2", "cz-sk-mo-z"}

var (
	// skmoOptionRE is an olympiad on the page of the documents: in the list of the older ones, as <option value="66">, or a link to one of the newer ones, as dokumenty.php?rocnik=75.
	skmoOptionRE = regexp.MustCompile(`<option value="(\d+)"|dokumenty\.php\?rocnik=(\d+)`)
	skmoTableRE  = regexp.MustCompile(`(?s)<table class="terminy">(.*?)</table>`)
	skmoHeadRE   = regexp.MustCompile(`(?s)<th([^>]*)>(.*?)</th>`)
	skmoCellRE   = regexp.MustCompile(`(?s)<td([^>]*)>(.*?)</td>`)
	skmoSpanRE   = regexp.MustCompile(`colspan="(\d+)"`)
	// skmoLinkRE is a paper in a cell, with the note on its language that follows it, if any.
	skmoLinkRE = regexp.MustCompile(`(?s)<a href="(dokument\.php\?id=\d+)">[^<]*</a>(\s*\[<span class="tooltip">\?<span class="tooltiptext">([^<]*)</span>)?`)
	// skmoCategoryRE is the category of a row: A, B, C, or Z5 to Z9 of the primary school. Z4 is no part of cz-sk-mo-z.
	skmoCategoryRE = regexp.MustCompile(`^(?:[ABC]|Z[5-9])$`)
)

// skmoRounds are the rounds of the head of the table, as the page names them.
var skmoRounds = map[string]string{"domáce": "home", "školské": "school", "okresné": "district", "krajské": "regional", "celoštátne": "national"}

// skmoRoundNames are the Slovak names of the rounds, for the names of the statements.
var skmoRoundNames = map[string]string{"home": "domáce kolo", "school": "školské kolo", "district": "okresné kolo", "regional": "krajské kolo", "national": "celoštátne kolo"}

// skmoTable reads the papers of an olympiad from its page: a row for each category, and for each round of the head a group of columns whose first column has the problems. A cell has the paper in Slovak and the paper in Hungarian, each with a note on its language; om skmo takes the Slovak one. The year is the end year of the school year, 1951 plus the number of the olympiad.
func skmoTable(page string, olympiad int) []hrPaper {
	m := skmoTableRE.FindStringSubmatch(page)
	if m == nil {
		return nil
	}
	rows := hrRowRE.FindAllStringSubmatch(m[1], -1)
	if len(rows) < 3 {
		return nil
	}
	// The first row of the head has the rounds, after the cell of the categories.
	starts := map[int]string{}
	col := 0
	for k, h := range skmoHeadRE.FindAllStringSubmatch(rows[0][1], -1) {
		if k == 0 {
			continue
		}
		if r, ok := skmoRounds[hrPlain(h[2])]; ok {
			starts[col] = r
		}
		col += skmoSpan(h[1])
	}
	var out []hrPaper
	for _, row := range rows[2:] {
		hs := skmoHeadRE.FindStringSubmatch(row[1])
		if hs == nil || !skmoCategoryRE.MatchString(hrPlain(hs[2])) {
			continue
		}
		cat := hrPlain(hs[2])
		col := 0
		for _, c := range skmoCellRE.FindAllStringSubmatch(row[1], -1) {
			if round, ok := starts[col]; ok {
				for _, l := range skmoLinkRE.FindAllStringSubmatch(c[2], -1) {
					if l[3] == "" || strings.Contains(l[3], "sloven") {
						out = append(out, hrPaper{kind: cat, variant: round, url: skmoSite + l[1], year: 1951 + olympiad})
						break
					}
				}
			}
			col += skmoSpan(c[1])
		}
	}
	return out
}

// skmoSpan is the number of columns of a cell.
func skmoSpan(attr string) int {
	if m := skmoSpanRE.FindStringSubmatch(attr); m != nil {
		if n, err := strconv.Atoi(m[1]); err == nil && n > 0 {
			return n
		}
	}
	return 1
}

// skmoLetters are the letters and the signs beyond ASCII of the Slovak papers, with the Czech letters of the names of the authors.
const skmoLetters = "áäčďéíĺľňóôŕšťúýžÁÄČĎÉÍĹĽŇÓÔŔŠŤÚÝŽěřůĚŘŮöüÖÜ–—„“”‘’…°€§«»·×−±≤≥•"

// skmoRules shape the Slovak papers: the problems are "1." to "6.", or "1" in bold in the papers of XeLaTeX, and each paper has the head of the olympiad, the round and the category, and at its end the rules of the round and the names of the authors.
var skmoRules = official.HrRules{
	First:   true,
	Start:   regexp.MustCompile(`^(\d)\.?(?:\s|$)`),
	Stop:    regexp.MustCompile(`^(?:SK\s|MATEMATICK[AÁ]|OLYMPI[AÁ]DA|\d{4}/\d{4}$|\d+\.\s*ročník|Zadania|\(?Termín|druhá trojica|\d\.\s*časť|Za každú|Každú úlohu|Každá úloha|Počas súťaže|Na riešenie|Riešenie|Návod|Úlohy|Pri riešení|Súťažiaci|Čas na|(?:•\s*)?(?:vydali|a[u]tor|recenzent|preklad)|Slovenská komisia|Kolo sa|Domáce kolo|Školské kolo|Okresné kolo|Krajské kolo|\p{L}+ kolo MO\b)`),
	Skip:    regexp.MustCompile(`^(?:skmo\.sk|\d{1,2}|–\s*\d+\s*–)$`),
	Drop:    skmoDropRE,
	Not:     regexp.MustCompile(`^\d\s\d{3}\b`),
	Lossy:   skmoLossy,
	Letters: skmoLetters,
}

// skmoLossy is Cambria, the font of the papers from the 71st olympiad (2021/22), made with XeLaTeX: their text loses the f before i, l and the other tall letters ("ci ier" for cifier), the accents of the capitals ("Skolské" for Školské) and the ĺ ("dlžka" for dĺžka), so their problems keep their links only.
var skmoLossy = regexp.MustCompile(`^(?:[A-Z]{6}\+)?Cambria`)

// skmoAuthorRE is the name of the authors of a problem at its end, as "(P. Černek)" or "(Vojtech Bálint, Michal Rolínek)".
var skmoAuthorRE = regexp.MustCompile(`\s*\((?:\p{Lu}[\p{L}'-]*\.?\s*)+(?:,\s*(?:\p{Lu}[\p{L}'-]*\.?\s*)+)*\)\s*$`)

// skmoDropRE is the name of the authors of a problem, at the right margin after the last line of the problem or on a line of its own.
var skmoDropRE = regexp.MustCompile(`^\((?:\p{Lu}[\p{L}'-]*\.?\s*)+(?:,\s*(?:\p{Lu}[\p{L}'-]*\.?\s*)+)*\)$`)

// skmoText is the text of a problem without the names of its authors, with the words that a line broke joined, as troj- uholník, and with its bullets as a list.
func skmoText(s string) string {
	if m := skmoAuthorRE.FindStringIndex(s); m != nil && (strings.Contains(s[m[0]:], ".") || strings.Contains(strings.TrimSpace(s[m[0]:]), " ")) {
		s = strings.TrimSpace(s[:m[0]])
	}
	s = skmoHyphenRE.ReplaceAllString(s, "$1$2")
	// The bullets of a list that a line holds become the items of a list of Markdown.
	if strings.Contains(s, "•") {
		s = skmoBulletRE.ReplaceAllString(s, "\n- ")
		s = strings.Replace(s, "\n- ", "\n\n- ", 1)
		s = strings.TrimPrefix(strings.TrimPrefix(s, "\n"), "\n")
	}
	return s
}

// skmoBulletRE is a bullet of a list in the text of a problem, with the spaces around it.
var skmoBulletRE = regexp.MustCompile(`[ \t]*•\s*`)

// skmoHyphenRE is a word that the end of a line broke, as "troj- uholník": a hyphen between two lower case letters with a space after it.
var skmoHyphenRE = regexp.MustCompile(`(\p{Ll})- (\p{Ll})`)

// skmoPlace gives the positions of the problems of a Slovak paper and the names of their statements. Categories A, B and C have a home round, haN, and a school round, saN, in cz-sk-mo-r1, a regional round in cz-sk-mo-r2, and category A the national round, cz-sk-mo; categories Z5 to Z9 have a home round, hGN, a district round, dGN, and Z9 a regional round, r9N, in cz-sk-mo-z.
func skmoPlace(p hrPaper, probs []official.HrProblem) (ats, names []string) {
	cat := strings.ToLower(p.kind)
	at := ""
	switch {
	case cat[0] != 'z' && p.variant == "home":
		at = fmt.Sprintf("cz-sk-mo-r1/%d/h%s", p.year, cat)
	case cat[0] != 'z' && p.variant == "school":
		at = fmt.Sprintf("cz-sk-mo-r1/%d/s%s", p.year, cat)
	case cat[0] != 'z' && p.variant == "regional":
		at = fmt.Sprintf("cz-sk-mo-r2/%d/%s", p.year, cat)
	case cat == "a" && p.variant == "national":
		at = fmt.Sprintf("cz-sk-mo/%d/p", p.year)
	case cat[0] == 'z' && p.variant == "home":
		at = fmt.Sprintf("cz-sk-mo-z/%d/h%c", p.year, cat[1])
	case cat[0] == 'z' && p.variant == "district":
		at = fmt.Sprintf("cz-sk-mo-z/%d/d%c", p.year, cat[1])
	case cat == "z9" && p.variant == "regional":
		at = fmt.Sprintf("cz-sk-mo-z/%d/r9", p.year)
	}
	name := fmt.Sprintf("skmo.sk, Matematická olympiáda %d/%d, %s, kategória %s", p.year-1, p.year, skmoRoundNames[p.variant], p.kind)
	for _, pr := range probs {
		if at == "" {
			ats = append(ats, "")
		} else {
			ats = append(ats, at+strconv.Itoa(pr.N))
		}
		names = append(names, name)
	}
	return ats, names
}

// skmoHarvest is the harvest of skmo.sk.
var skmoHarvest = &paperHarvest{
	cmd:      "skmo",
	source:   skmoSource,
	codes:    skmoCodes,
	lang:     "sk",
	language: "Slovak",
	papers:   skmoPapers,
	rules:    func(hrPaper) official.HrRules { return skmoRules },
	place:    skmoPlace,
	fix: func(p *official.HrProblem) {
		p.Text = skmoText(p.Text)
	},
}

// skmoPapers reads the list of the olympiads from the page of the documents and the papers from the page of each olympiad. The pages are in windows-1250.
func skmoPapers(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error) {
	b, err := hrGet(ctx, fc, skmoSource, skmoSite+"dokumenty.php", opt)
	if err != nil {
		return nil, err
	}
	last := 0
	for _, m := range skmoOptionRE.FindAllStringSubmatch(string(b), -1) {
		if n, err := strconv.Atoi(m[1] + m[2]); err == nil && n > last {
			last = n
		}
	}
	var out []hrPaper
	for n := skmoFirst; n <= last; n++ {
		b, err := hrGet(ctx, fc, skmoSource, fmt.Sprintf("%sdokumenty.php?rocnik=%d", skmoSite, n), opt)
		if err != nil {
			return nil, err
		}
		page, err := charmap.Windows1250.NewDecoder().Bytes(b)
		if err != nil {
			return nil, err
		}
		out = append(out, skmoTable(string(page), n)...)
	}
	return out, nil
}

func runSkmo(env Env, args []string) int {
	return skmoHarvest.run(env, args)
}
