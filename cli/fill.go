package cli

import (
	"cmp"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/ocr"
	"github.com/tamnd/olymmath-agent/official"
)

func init() {
	register(command{"fill", "give a text to the link-only records of a Russian olympiad from its official pages, at the same position, when the problems.ru text of the record agrees (om fill -keys -write ru-moscow)", runFill})
}

// ruLane is the official source of a Russian olympiad whose records hold only a link to problems.ru.
type ruLane struct {
	// source is the official source in the register.
	source string
	// node is the olympiad node of the source catalogue of problems.ru.
	node int
	// read gives the official problems of an edition of the content.
	read func(ctx context.Context, fc *fetch.Client, source, edition string) ([]official.RuProblem, error)
	// holds tells if the year node name of problems.ru holds the edition.
	holds func(name, edition string) bool
	// final tells that the problems of a year node are in its child node of the final round.
	final bool
}

var ruLanes = map[string]ruLane{
	"ru-moscow": {source: "mccme-mmo", node: prMoscow, read: mmoRead, holds: prSameYear},
}

// fillReview is the file of the records that om fill did not fill.
func fillReview(comp string) string { return "reports/review/fill-" + comp + ".tsv" }

const fillHeader = "# The link-only records that om fill did not fill: the position, the problem, the problems.ru number, the official page, the word signal of the official text at the position with the problems.ru text, the best other position of the edition and its signal, and the reason. The problems.ru text is a key in memory only and is not kept.\n"

// fillTarget is one appearance of a link-only record.
type fillTarget struct {
	p                *content.Problem
	at, edition, pos string
	// pr is the number of the problem on problems.ru, and link the id of the statement that links to it.
	pr   int
	link string
}

func runFill(env Env, args []string) int {
	fs := newFlags("fill", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 0, "the first year")
	to := fs.Int("to", 9999, "the last year")
	keys := fs.Bool("keys", false, "read the problems.ru text of each record, as a key in memory, and compare it with the official text")
	write := fs.Bool("write", false, "with -keys, fill the records that agree and write the review file")
	explain := fs.Bool("explain", false, "with -keys, print the signals of each appearance; the problems.ru text is not printed")
	figures := fs.Bool("figures", false, "fill a problem whose official text has a picture too; the picture must then be added with om figure")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 {
		fmt.Fprintln(env.Stderr, "usage: om fill [-from YEAR] [-to YEAR] [-keys [-write]] ru-moscow")
		return ExitUsage
	}
	lane, ok := ruLanes[pos[0]]
	if !ok {
		fmt.Fprintf(env.Stderr, "om fill: no official lane for %s\n", pos[0])
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om fill: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	comp, src := corpus.Competition(pos[0]), corpus.Source(lane.source)
	if comp == nil || src == nil {
		fmt.Fprintf(env.Stderr, "om fill: the content has no competition %s or no source %s\n", pos[0], lane.source)
		return ExitUsage
	}
	targets := fillTargets(corpus, pos[0], *from, *to)
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om fill: %v\n", err)
		return ExitFailure
	}
	fc := fetch.New(st)
	fc.Agent = fmt.Sprintf("olymmath/%s (+https://olymmath.tamnd.com/about/)", env.Version)
	ctx := context.Background()
	var pr *prKeys
	if *keys {
		pr = &prKeys{fc: fc, lane: lane, log: env.Stderr, cursor: map[int]*prCursor{}, text: map[int]string{}}
	}
	code := ExitOK
	var editions []string
	for ed := range targets {
		editions = append(editions, ed)
	}
	slices.Sort(editions)
	var review []string
	filled, total := 0, 0
	for _, ed := range editions {
		ts := targets[ed]
		total += len(ts)
		probs, err := lane.read(ctx, fc, lane.source, ed)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om fill: %s/%s: %v\n", pos[0], ed, err)
			code = ExitFailure
			continue
		}
		byPos := map[string]official.RuProblem{}
		for _, p := range probs {
			byPos[p.Position] = p
		}
		if pr == nil {
			at, fig, flagged := 0, 0, 0
			for _, t := range ts {
				p, ok := byPos[t.pos]
				if !ok {
					continue
				}
				at++
				if p.Figure {
					fig++
				}
				if len(p.Flags) > 0 {
					flagged++
				}
			}
			fmt.Fprintf(env.Stdout, "%s/%s: %d link-only, %d official problems, %d at the position (%d with a picture, %d flagged)\n", pos[0], ed, len(ts), len(probs), at, fig, flagged)
			continue
		}
		var ids []int
		for _, t := range ts {
			if _, ok := byPos[t.pos]; ok {
				ids = append(ids, t.pr)
			}
		}
		if err := pr.read(ctx, ed, ids); err != nil {
			fmt.Fprintf(env.Stderr, "om fill: %s/%s: problems.ru: %v\n", pos[0], ed, err)
			code = ExitFailure
		}
		n := 0
		// The appearances of one problem are judged together: the problem is filled from its first appearance whose position agrees, and only when no appearance disagrees.
		byProblem := map[*content.Problem][]fillTarget{}
		var order []*content.Problem
		for _, t := range ts {
			if _, ok := byProblem[t.p]; !ok {
				order = append(order, t.p)
			}
			byProblem[t.p] = append(byProblem[t.p], t)
		}
		for _, p := range order {
			var use *official.RuProblem
			var rows []string
			for _, t := range byProblem[p] {
				o, ok := byPos[t.pos]
				row := func(k official.RuCheck, why string) string {
					return strings.Join([]string{t.at, p.Tag, strconv.Itoa(t.pr), o.URL, fmt.Sprintf("%.2f", k.J), cmpOr(k.Rival, "-"), fmt.Sprintf("%.2f", k.RivalJ), why}, "\t")
				}
				if !ok {
					rows = append(rows, row(official.RuCheck{}, "the official page has no problem at the position"))
					continue
				}
				key, ok := pr.text[t.pr]
				if !ok {
					rows = append(rows, row(official.RuCheck{}, "problems.ru gave no text for the number"))
					continue
				}
				k := official.RuConfirm(o, probs, key)
				if *explain {
					fmt.Fprintf(env.Stdout, "%s\t%d\tj=%.2f c=%.2f rival=%s %.2f figure=%v flags=%v\n", t.at, t.pr, k.J, k.C, cmpOr(k.Rival, "-"), k.RivalJ, o.Figure, o.Flags)
				}
				switch {
				case !official.RuAgrees(k.J, k.C):
					rows = append(rows, row(k, "the official text at the position does not agree with the problems.ru text"))
				case !k.OK():
					rows = append(rows, row(k, "another official text of the edition agrees nearly as well"))
				case o.Figure && !*figures:
					rows = append(rows, row(k, "the official problem has a picture"))
				case len(o.Flags) > 0:
					rows = append(rows, row(k, "the official text is flagged: "+strings.Join(o.Flags, ", ")))
				default:
					if use == nil {
						use = &o
					}
				}
			}
			if use == nil || len(rows) > 0 {
				review = append(review, rows...)
				continue
			}
			flags := append(ocr.Check(use.Text), use.Flags...)
			conf := ocr.Confidence(flags)
			if conf < ocr.Threshold {
				review = append(review, strings.Join([]string{byProblem[p][0].at, p.Tag, strconv.Itoa(byProblem[p][0].pr), use.URL, "-", "-", "-", "the text is flagged: " + strings.Join(flags, ", ")}, "\t"))
				continue
			}
			n++
			if !*write {
				continue
			}
			if err := fillOne(*root, p, byProblem[p][0].at, comp, src, *use, conf); err != nil {
				fmt.Fprintf(env.Stderr, "om fill: %s: %v\n", p.Path, err)
				code = ExitFailure
				n--
			}
		}
		filled += n
		fmt.Fprintf(env.Stdout, "%s/%s: %d link-only, %d official problems, %d filled\n", pos[0], ed, len(ts), len(probs), n)
	}
	if pr != nil && !*write {
		for _, r := range review {
			fmt.Fprintln(env.Stdout, r)
		}
	}
	if pr != nil && *write {
		if err := writeFillReview(*root, pos[0], editions, review); err != nil {
			fmt.Fprintf(env.Stderr, "om fill: %v\n", err)
			return ExitFailure
		}
	}
	if pr != nil {
		verb := "agree"
		if *write {
			verb = "filled"
		}
		fmt.Fprintf(env.Stdout, "%d link-only appearances in %d editions, %d problems %s, %d appearances for review.\n", total, len(editions), filled, verb, len(review))
	}
	return code
}

// fillTargets gives the appearances of the link-only records of comp in the years from to to, by edition. A record is link-only when it has no text and links to problems.ru.
func fillTargets(corpus *content.Corpus, comp string, from, to int) map[string][]fillTarget {
	out := map[string][]fillTarget{}
	for _, p := range corpus.All() {
		if strings.TrimSpace(p.Statement) != "" {
			continue
		}
		var link string
		var id int
		for _, s := range p.Statements {
			if m := prIDRE.FindStringSubmatch(s.URL); m != nil && s.Source == problemsRu && s.Text == "" {
				link = s.ID
				id, _ = strconv.Atoi(m[1])
			}
		}
		if link == "" {
			continue
		}
		for _, a := range p.Appearances {
			parts := strings.Split(a.At, "/")
			if len(parts) != 3 || parts[0] != comp {
				continue
			}
			y, err := strconv.Atoi(parts[1][:min(4, len(parts[1]))])
			if err != nil || y < from || y > to {
				continue
			}
			out[parts[1]] = append(out[parts[1]], fillTarget{p: p, at: a.At, edition: parts[1], pos: parts[2], pr: id, link: link})
		}
	}
	return out
}

// fillOne gives the official text o to the record p at the appearance at, as a new statement that becomes the canonical statement.
func fillOne(root string, p *content.Problem, at string, comp *content.Competition, src *content.SourceRecord, o official.RuProblem, conf float64) error {
	full := filepath.Join(root, filepath.FromSlash(p.Path))
	b, err := os.ReadFile(full) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	kind := "official page"
	if o.Page > 0 {
		kind = "official paper"
	}
	c := canon.Candidate{At: at, Source: src.ID, Origin: src.Origin, Zone: src.Zone.Statement, Lang: "ru", Name: fmt.Sprintf("%s %s %s", comp.Name, o.Edition, kind), URL: o.URL, Text: o.Text}
	add := canon.AddLink(p, &c, o.Page)
	add.Statements[0].Extraction = official.RuExtraction(o)
	add.Statements[0].Confidence = conf
	_, rule := canon.Rank(&c, original(comp))
	if b, err = canon.Apply(b, add); err != nil {
		return err
	}
	if b, err = canon.Fill(b, add.Statements[0].ID, o.Text, rule, "captured"); err != nil {
		return err
	}
	return os.WriteFile(full, b, 0o644) // #nosec G306 G703 -- content files are public.
}

// writeFillReview writes the review rows of the editions of this run, and keeps the rows of the other editions.
func writeFillReview(root, comp string, editions, rows []string) error {
	full := filepath.Join(root, filepath.FromSlash(fillReview(comp)))
	ran := map[string]bool{}
	for _, ed := range editions {
		ran[comp+"/"+ed] = true
	}
	var keep []string
	if b, err := os.ReadFile(full); err == nil { // #nosec G304 -- the path is inside the content root.
		for _, l := range strings.Split(string(b), "\n") {
			if l == "" || strings.HasPrefix(l, "#") {
				continue
			}
			at, _, _ := strings.Cut(l, "\t")
			if !ran[at[:strings.LastIndex(at, "/")]] {
				keep = append(keep, l)
			}
		}
	}
	keep = append(keep, rows...)
	if len(keep) == 0 {
		if err := os.Remove(full); err != nil && !os.IsNotExist(err) {
			return err
		}
		return nil
	}
	slices.SortFunc(keep, func(a, b string) int { return cmp.Compare(a, b) })
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
		return err
	}
	return os.WriteFile(full, []byte(fillHeader+strings.Join(keep, "\n")+"\n"), 0o644) // #nosec G306 G703 -- the review file is public.
}

// prKeys reads the texts of problems.ru numbers from the listings of the year nodes of one olympiad. The texts stay in memory: they are keys, and the content never holds them.
type prKeys struct {
	fc     *fetch.Client
	lane   ruLane
	log    io.Writer
	years  []archive.PRNode
	cursor map[int]*prCursor
	text   map[int]string
}

// prCursor is the next problem of the listing of a node, and done tells that the listing has no more.
type prCursor struct {
	start int
	done  bool
}

func (k *prKeys) page(ctx context.Context, u string) (string, error) {
	b, err := k.fc.Read(ctx, u)
	return string(b), err
}

// read reads the listings of the year nodes that hold the edition until it has the text of each number of ids.
func (k *prKeys) read(ctx context.Context, edition string, ids []int) error {
	missing := func() bool {
		for _, id := range ids {
			if _, ok := k.text[id]; !ok {
				return true
			}
		}
		return false
	}
	if !missing() {
		return nil
	}
	if k.years == nil {
		root, err := k.page(ctx, archive.PRNodeURL(k.lane.node, 0))
		if err != nil {
			return err
		}
		k.years = archive.PRNodes(root)
	}
	for _, yn := range k.years {
		if !k.lane.holds(yn.Name, edition) {
			continue
		}
		node := yn.ID
		if k.lane.final {
			page, err := k.page(ctx, archive.PRNodeURL(node, 0))
			if err != nil {
				return err
			}
			node = 0
			for _, c := range archive.PRNodes(page) {
				if archive.PRFinal(c.Name) {
					node = c.ID
				}
			}
			if node == 0 {
				continue
			}
		}
		c := k.cursor[node]
		if c == nil {
			c = &prCursor{}
			k.cursor[node] = c
		}
		for !c.done && missing() {
			page, err := k.page(ctx, archive.PRNodeURL(node, c.start))
			if err != nil {
				return err
			}
			ps := archive.ParsePRListing(page)
			for _, p := range ps {
				if p.Err == nil {
					k.text[p.ID] = p.Text
				}
			}
			c.start += len(ps)
			c.done = len(ps) == 0 || c.start >= archive.PRTotal(page)
		}
	}
	return nil
}

// prSameYear tells if a year node such as "2003 год" holds the edition of that year.
func prSameYear(name, edition string) bool {
	y, ok := archive.PRYear(name)
	return ok && strconv.Itoa(y) == edition
}

// mmoRead reads the problems of a year of the Moscow olympiad from its page on the site of the olympiad.
func mmoRead(ctx context.Context, fc *fetch.Client, source, edition string) ([]official.RuProblem, error) {
	y, err := strconv.Atoi(edition)
	if err != nil {
		return nil, err
	}
	var u string
	switch {
	case y == 1999:
		u = "https://mmo.mccme.ru/1999/index.htm"
	case y >= 2000 && y <= 2008:
		u = fmt.Sprintf("https://mmo.mccme.ru/%d/mmo%d.htm", y, y)
	case y >= 2019:
		u = fmt.Sprintf("https://mmo.mccme.ru/%d/", y)
	default:
		return nil, nil
	}
	raw, err := get(ctx, fc, source, u)
	if err != nil {
		return nil, err
	}
	page, err := official.RuDecode([]byte(raw))
	if err != nil {
		return nil, err
	}
	return official.RuPage(page, edition, u, func(g int) bool { return g >= 7 && g <= 11 }), nil
}
