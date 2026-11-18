package cli

import (
	"context"
	"fmt"

	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"naboj", "read the problems of Náboj and Náboj Junior from the archives of math.naboj.org and junior.naboj.org and write them to naboj and naboj-junior (om naboj -write)", runNaboj})
}

// nabojSource is the source of the Náboj sites.
const nabojSource = "naboj"

// nabojSites are the archives that om naboj reads: Náboj Math in English, the language of the international set, and Náboj Junior in Slovak, as the junior site has no English pages.
var nabojSites = []struct {
	code, host, archive, lang, language, name string
}{
	{"naboj", "https://math.naboj.org", "/gb/en/archive", "en", "English", "Náboj Math"},
	{"naboj-junior", "https://junior.naboj.org", "/sk/sk/archive", "sk", "Slovak", "Náboj Junior"},
}

// nabojPapers lists the pages of problems of the archives of the sites of code. A year whose page has no problems, only a paper PDF, is no paper here: om naboj reads the HTML only.
func nabojPapers(code string) func(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error) {
	return func(ctx context.Context, fc *fetch.Client, opt fetch.Options) ([]hrPaper, error) {
		var out []hrPaper
		for _, s := range nabojSites {
			if s.code != code {
				continue
			}
			b, err := hrGet(ctx, fc, nabojSource, s.host+s.archive, opt)
			if err != nil {
				return nil, err
			}
			for _, y := range official.NabojArchive(string(b)) {
				if y.Problems != "" {
					out = append(out, hrPaper{kind: s.code, url: s.host + y.Problems, year: y.Year})
				}
			}
		}
		return out, nil
	}
}

// nabojRead reads the problems of the page of problems of a paper.
func nabojRead(ctx context.Context, fc *fetch.Client, p hrPaper, opt fetch.Options) ([]official.HrProblem, error) {
	b, err := hrGet(ctx, fc, nabojSource, p.url, opt)
	if err != nil {
		return nil, err
	}
	return official.NabojProblems(string(b)), nil
}

// nabojPlace places problem N of the competition of a year at pN.
func nabojPlace(p hrPaper, probs []official.HrProblem) (ats, names []string) {
	name := p.kind
	for _, s := range nabojSites {
		if s.code == p.kind {
			name = s.name
		}
	}
	for _, pr := range probs {
		at := ""
		if pr.N > 0 {
			at = fmt.Sprintf("%s/%d/p%d", p.kind, p.year, pr.N)
		}
		ats = append(ats, at)
		names = append(names, fmt.Sprintf("%s %d, problems and solutions", name, p.year))
	}
	return ats, names
}

func nabojHarvest(i int) *paperHarvest {
	s := nabojSites[i]
	return &paperHarvest{
		cmd:        "naboj",
		source:     nabojSource,
		codes:      []string{s.code},
		lang:       s.lang,
		language:   s.language,
		papers:     nabojPapers(s.code),
		place:      nabojPlace,
		read:       nabojRead,
		extraction: "native/html (" + official.NabojVersion + ")",
	}
}

// runNaboj reads Náboj Math and then Náboj Junior, each in its language.
func runNaboj(env Env, args []string) int {
	code := ExitOK
	for i := range nabojSites {
		if c := nabojHarvest(i).run(env, args); c != ExitOK {
			code = c
		}
	}
	return code
}
