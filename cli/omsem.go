package cli

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"omsem", "read the problems of the first and second rounds of the Polish olympiad from the papers of om.sem.edu.pl and write them to pl-om-r1 and pl-om-r2 (om omsem -write)", runOMSem})
}

// omSemSource is the source of the papers of the Polish olympiad.
const omSemSource = "om-sem"

// omSemCodes are the competitions of the first and the second round.
var omSemCodes = []string{"pl-om-r1", "pl-om-r2"}

// omSemPapers lists the papers of the first and the second round of each olympiad on the listing page. A round whose problems are only in the file with the solutions, as the second round of the 51st and 52nd olympiads, has no paper here.
func omSemPapers(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error) {
	b, err := hrGet(ctx, fc, omSemSource, official.OMSemListing, opt)
	if err != nil {
		return nil, err
	}
	var out []hrPaper
	for _, n := range official.OMSemPapers(string(b)) {
		switch n.Paper {
		case "stage1":
			out = append(out, hrPaper{kind: "pl-om-r1", url: n.URL, year: n.Year})
		case "stage2":
			out = append(out, hrPaper{kind: "pl-om-r2", url: n.URL, year: n.Year})
		}
	}
	return out, nil
}

// omSemRules read the papers, set in the PL fonts of the Polish TeX users: the problems are numbered through the paper, the series of the first round and the days of the second round each start with the header of the olympiad, and the second round ends with the information for the participants, whose points are numbered too.
var omSemRules = official.HrRules{
	First:   true,
	Start:   regexp.MustCompile(`^(\d{1,2})\.(?:\s|$)`),
	Stop:    regexp.MustCompile(`^(?:[LXVI]+\s+Olimpiada Matematyczna|Zadania konkursowe|Zawody (?:pierwszego|drugiego|trzeciego) stopnia|zawodów stopnia|Teksty zadań|[IV]+\s+seria|Rozwiązania (?:powyższych )?zadań|Informacje dla uczestnika|\(decyduje|Każde rozwiązanie|\d{1,2}\s+\p{L}+\s+\d{4}\s*r\.)`),
	Skip:    regexp.MustCompile(`^(?:\d{1,2}|–\s*\d+\s*–)$`),
	Not:     regexp.MustCompile(`^\d\.\s+(?:Czas trwania zawodów|W przypadku (?:konieczności|stwierdzenia)|W czasie zawodów|Należy (?:pisać wy|dokładnie przestrzegać)|Przed rozpoczęciem rozwiązywania)`),
	Letters: "ąćęłńóśźżĄĆĘŁŃÓŚŹŻ–—„”…·°×",
	Math:    regexp.MustCompile(`^(?:[A-Z]{6}\+)?PLMath(?:Italic|Symbols|Extension)\d*(?:-\w+)?$`),
	Text:    regexp.MustCompile(`^(?:[A-Z]{6}\+)?PL(?:Roman|Sans|Typewriter|RomanSlant|Bold|Italic|Capitals)\w*\d*(?:-\w+)?$`),
}

// omSemColumns are the papers whose pages hold two pages of A5 side by side, as two columns: the papers of the second round from 2015, but 2016 and 2021, and the first round of 2021.
var omSemColumns = map[string][]int{"pl-om-r1": {2021}, "pl-om-r2": {2015, 2017, 2018, 2019, 2020, 2022, 2023, 2024, 2025, 2026}}

// omSemLossy are the papers whose text layer loses the ą and the ę ("sa" for "są", "bedzie" for "będzie"), so that their problems keep their links only.
var omSemLossy = map[string][]int{"pl-om-r1": {1998, 1999, 2001, 2010, 2016, 2017, 2018}, "pl-om-r2": {1998, 1999, 2015, 2016, 2017}}

// omSemAnyFont matches every font, for the rule Lossy of a paper of omSemLossy.
var omSemAnyFont = regexp.MustCompile(`^`)

// omSemAngleRE is the sign of an angle of the papers, "<" set over ")", which the text layer keeps as the two glyphs, as "<)APB" for the angle APB.
var omSemAngleRE = regexp.MustCompile(`<\s?\)\s*([A-Z\\])`)

// omSemPaperRules gives the rules of a paper: omSemRules, in two columns for a paper of omSemColumns, and with every font lossy for a paper of omSemLossy.
func omSemPaperRules(p hrPaper) official.HrRules {
	r := omSemRules
	r.Columns = slices.Contains(omSemColumns[p.kind], p.year)
	if slices.Contains(omSemLossy[p.kind], p.year) {
		r.Lossy = omSemAnyFont
	}
	return r
}

// omSemPlace places problem N of the paper of a round at pN of its edition.
func omSemPlace(p hrPaper, probs []official.HrProblem) (ats, names []string) {
	round := "first"
	if p.kind == "pl-om-r2" {
		round = "second"
	}
	for _, pr := range probs {
		at := ""
		if pr.N > 0 {
			at = fmt.Sprintf("%s/%d/p%d", p.kind, p.year, pr.N)
		}
		ats = append(ats, at)
		names = append(names, fmt.Sprintf("Polish Mathematical Olympiad %d, %s round paper", p.year, round))
	}
	return ats, names
}

var omSemHarvest = &paperHarvest{
	cmd:      "omsem",
	source:   omSemSource,
	codes:    omSemCodes,
	lang:     "pl",
	language: "Polish",
	papers:   omSemPapers,
	rules:    omSemPaperRules,
	place:    omSemPlace,
	fix: func(p *official.HrProblem) {
		p.Text = skmoHyphenRE.ReplaceAllString(p.Text, "$1$2")
		p.Text = omSemAngleRE.ReplaceAllString(p.Text, `\angle $1`)
	},
}

func runOMSem(env Env, args []string) int {
	return omSemHarvest.run(env, args)
}
