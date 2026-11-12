package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"papers", "record the official papers of each edition (om papers imo-official|usa-tst-archive|om-sem|bundeswettbewerb|ukmt|hbcse|imojp|amt|vtrmc-official|imc-official|pamo-official|tuymaada-official|georgmohr|spm-opm|mo-cz|elmo-official|dms-rs|aimo-awb|oktatas-oktv|ssmr-onm|dmfa-massa)", runPapers})
}

func runPapers(env Env, args []string) int {
	fs := newFlags("papers", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 1959, "the first year")
	to := fs.Int("to", time.Now().Year(), "the last year")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 || !slices.Contains([]string{"imo-official", "usa-tst-archive", "om-sem", "bundeswettbewerb", "ukmt", "hbcse", "imojp", "amt", "vtrmc-official", "imc-official", "pamo-official", "tuymaada-official", "georgmohr", "spm-opm", "mo-cz", "elmo-official", "dms-rs", "aimo-awb", "oktatas-oktv", "ssmr-onm", "dmfa-massa"}, pos[0]) {
		fmt.Fprintln(env.Stderr, "usage: om papers [-from YEAR] [-to YEAR] imo-official|usa-tst-archive|om-sem|bundeswettbewerb|ukmt|hbcse|imojp|amt|vtrmc-official|imc-official|pamo-official|tuymaada-official|georgmohr|spm-opm|mo-cz|elmo-official|dms-rs|aimo-awb|oktatas-oktv|ssmr-onm|dmfa-massa")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om papers: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	if corpus.Source(pos[0]) == nil {
		fmt.Fprintf(env.Stderr, "om papers: the content has no source %s\n", pos[0])
		return ExitUsage
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om papers: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	ctx := context.Background()
	switch pos[0] {
	case "usa-tst-archive":
		return tstPapers(ctx, env, fc, corpus, *root, *from, *to)
	case "om-sem", "bundeswettbewerb", "ukmt", "hbcse", "imojp", "amt", "vtrmc-official", "imc-official", "pamo-official", "tuymaada-official", "georgmohr", "spm-opm", "mo-cz", "elmo-official", "dms-rs", "aimo-awb", "oktatas-oktv", "ssmr-onm", "dmfa-massa":
		return nationalPapers(ctx, env, fc, corpus, *root, pos[0], *from, *to)
	}
	comp := corpus.Competition("imo")
	if comp == nil {
		fmt.Fprintln(env.Stderr, "om papers: the content has no competition imo")
		return ExitUsage
	}
	code := ExitOK
	var editions, papers int
	for _, e := range comp.Editions {
		if e.Year < *from || e.Year > *to {
			continue
		}
		page, err := get(ctx, fc, pos[0], official.YearPage(e.Year))
		if err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", e.Locator(), err)
			code = ExitFailure
			continue
		}
		langs := official.Languages(page)
		if len(langs) == 0 {
			continue
		}
		if err := setOfficial(filepath.Join(*root, filepath.FromSlash(e.Path)), langs); err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", e.Path, err)
			return ExitFailure
		}
		editions++
		papers += len(langs)
	}
	fmt.Fprintf(env.Stdout, "%d editions, %d official papers.\n", editions, papers)
	return code
}

// tstPapers records the problems file of each test of the USA TST Archive as the official English paper of its edition. It writes the edition file when the corpus does not have the edition.
func tstPapers(ctx context.Context, env Env, fc *fetch.Client, corpus *content.Corpus, root string, from, to int) int {
	listing, err := get(ctx, fc, "usa-tst-archive", archive.TSTContents)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om papers: %v\n", err)
		return ExitFailure
	}
	ps, err := archive.TSTPapers([]byte(listing))
	if err != nil {
		fmt.Fprintf(env.Stderr, "om papers: the listing: %v\n", err)
		return ExitFailure
	}
	var editions, created int
	for _, p := range ps {
		if p.Year < from || p.Year > to {
			continue
		}
		if corpus.Competition(p.Competition) == nil {
			fmt.Fprintf(env.Stderr, "om papers: the content has no competition %s\n", p.Competition)
			return ExitUsage
		}
		rel, err := newEdition(root, corpus, fmt.Sprintf("%s/%d/p1", p.Competition, p.Year))
		if err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s/%d: %v\n", p.Competition, p.Year, err)
			return ExitFailure
		}
		if rel != "" {
			created++
		}
		file := filepath.Join(root, "editions", p.Competition, fmt.Sprintf("%d.yaml", p.Year))
		if err := setOfficial(file, []content.OfficialPaper{{Lang: "English", Code: "en", URL: p.URL}}); err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", file, err)
			return ExitFailure
		}
		editions++
	}
	fmt.Fprintf(env.Stdout, "%d editions, %d new, %d official papers.\n", editions, created, editions)
	return ExitOK
}

// nationals gives the listing page of each national source and the function that reads the papers from it.
var nationals = map[string]struct {
	listings []string
	lang     content.OfficialPaper
	papers   func(string) []official.National
}{
	"om-sem":            {[]string{official.OMSemListing}, content.OfficialPaper{Lang: "Polish", Code: "pl"}, official.OMSemPapers},
	"bundeswettbewerb":  {[]string{official.BWMListing}, content.OfficialPaper{Lang: "German", Code: "de"}, official.BWMPapers},
	"ukmt":              {[]string{official.UKMTListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.UKMTPapers},
	"hbcse":             {[]string{official.HBCSEListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.HBCSEPapers},
	"imojp":             {official.IMOJPListings, content.OfficialPaper{Lang: "Japanese", Code: "ja"}, official.IMOJPPapers},
	"amt":               {[]string{official.AMTListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.AMTPapers},
	"vtrmc-official":    {[]string{official.VTRMCListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.VTRMCPapers},
	"imc-official":      {official.IMCListings(time.Now().Year()), content.OfficialPaper{Lang: "English", Code: "en"}, official.IMCPapers},
	"pamo-official":     {[]string{official.PAMOListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.PAMOPapers},
	"tuymaada-official": {[]string{official.TuymaadaListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.TuymaadaPapers},
	"georgmohr":         {[]string{official.GeorgMohrListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.GeorgMohrPapers},
	"spm-opm":           {[]string{official.OPMListing}, content.OfficialPaper{Lang: "Portuguese", Code: "pt"}, official.OPMPapers},
	"mo-cz":             {official.MOListings(time.Now().Year()), content.OfficialPaper{Lang: "Czech", Code: "cs"}, official.MOPapers},
	"elmo-official":     {[]string{official.ELMOListing}, content.OfficialPaper{Lang: "English", Code: "en"}, official.ELMOPapers},
	"dms-rs":            {[]string{official.DMSListing}, content.OfficialPaper{Lang: "Serbian", Code: "sr"}, official.DMSPapers},
	"aimo-awb":          {[]string{official.BWMListing}, content.OfficialPaper{Lang: "German", Code: "de"}, official.AWBPapers},
	"oktatas-oktv":      {official.OKTVListings, content.OfficialPaper{Lang: "Hungarian", Code: "hu"}, official.OKTVPapers},
	"ssmr-onm":          {official.SSMRListings, content.OfficialPaper{Lang: "Romanian", Code: "ro"}, official.SSMRPapers},
	"dmfa-massa":        {[]string{official.DMFAListing}, content.OfficialPaper{Lang: "Slovenian", Code: "sl"}, official.DMFAPapers},
}

// nationalPapers records the papers of a national source from its listing page. It writes the edition file when the corpus does not have the edition. The papers of one edition replace the papers of the source in the edition file, and the papers of other sources stay. Then om fetch SOURCE gets the papers into the store.
func nationalPapers(ctx context.Context, env Env, fc *fetch.Client, corpus *content.Corpus, root, source string, from, to int) int {
	n := nationals[source]
	var listing string
	for _, u := range n.listings {
		page, err := get(ctx, fc, source, u)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %v\n", err)
			return ExitFailure
		}
		listing += page + "\n"
	}
	byEdition := map[string][]content.OfficialPaper{}
	var keys []string
	for _, p := range n.papers(listing) {
		if p.Year < from || p.Year > to {
			continue
		}
		if corpus.Competition(p.Competition) == nil {
			fmt.Fprintf(env.Stderr, "om papers: the content has no competition %s\n", p.Competition)
			return ExitUsage
		}
		key := fmt.Sprintf("%s/%d", p.Competition, p.Year)
		if byEdition[key] == nil {
			keys = append(keys, key)
		}
		o := n.lang
		o.URL, o.Paper, o.Source = p.URL, p.Paper, source
		if p.Lang.Code != "" {
			o.Lang, o.Code = p.Lang.Lang, p.Lang.Code
		}
		byEdition[key] = append(byEdition[key], o)
	}
	var created, papers int
	for _, key := range keys {
		rel, err := newEdition(root, corpus, key+"/p1")
		if err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", key, err)
			return ExitFailure
		}
		if rel != "" {
			created++
		}
		file := filepath.Join(root, "editions", filepath.FromSlash(key)+".yaml")
		old, err := officialOf(file)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", file, err)
			return ExitFailure
		}
		list := slices.DeleteFunc(old, func(o content.OfficialPaper) bool { return o.Source == source })
		if err := setOfficial(file, append(list, byEdition[key]...)); err != nil {
			fmt.Fprintf(env.Stderr, "om papers: %s: %v\n", file, err)
			return ExitFailure
		}
		papers += len(byEdition[key])
	}
	fmt.Fprintf(env.Stdout, "%d editions, %d new, %d official papers.\n", len(keys), created, papers)
	return ExitOK
}

// officialOf reads the list of official papers of an edition file.
func officialOf(file string) ([]content.OfficialPaper, error) {
	b, err := os.ReadFile(file) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return nil, err
	}
	var e content.Edition
	if err := yaml.Unmarshal(b, &e); err != nil {
		return nil, err
	}
	return e.Official, nil
}

// get gets a page into the store and returns its text.
func get(ctx context.Context, fc *fetch.Client, source, url string) (string, error) {
	text, _, err := getCapture(ctx, fc, source, url)
	return text, err
}

// getCapture gets a page into the store and returns its text and the hash of its capture.
func getCapture(ctx context.Context, fc *fetch.Client, source, url string) (text, hash string, err error) {
	res, err := fc.Get(ctx, source, url, fetch.Options{})
	if err != nil {
		return "", "", err
	}
	rc, err := fc.Store.Get(ctx, res.Capture.Hash)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = rc.Close() }()
	b, err := io.ReadAll(rc)
	return string(b), res.Capture.Hash, err
}

// setOfficial sets the list of official papers of an edition file. The other fields keep their order.
func setOfficial(file string, langs []content.OfficialPaper) error {
	b, err := os.ReadFile(file) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(b, &doc); err != nil {
		return err
	}
	var list yaml.Node
	if err := list.Encode(langs); err != nil {
		return err
	}
	m := doc.Content[0]
	found := false
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == "official" {
			m.Content[i+1], found = &list, true
		}
	}
	if !found {
		m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "official"}, &list)
	}
	f, err := os.Create(file) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		_ = f.Close()
		return err
	}
	if err := enc.Close(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
