package cli

import (
	"bytes"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/locator"
	"github.com/tamnd/olymmath-agent/tag"
)

func init() {
	register(command{"fmt", "put the content files in their canonical form (om fmt -check to only report)", runFmt})
	register(command{"new", "write a new problem or solution file with a new tag (om new problem imo/2027/p1)", runNew})
	register(command{"tag", "make new tags or check a tag (om tag new, om tag check 0HVHSDNNMY)", runTag})
}

func runFmt(env Env, args []string) int {
	fs := newFlags("fmt", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	check := fs.Bool("check", false, "report the files that are not in canonical form, and change no file")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	var changed, failed int
	for _, dir := range []string{"problems", "solutions"} {
		err := filepath.WalkDir(filepath.Join(*root, dir), func(p string, d os.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() || !strings.HasSuffix(p, ".md") || d.Name() == "README.md" {
				return nil
			}
			b, err := os.ReadFile(p) // #nosec G304 G122 -- the path is inside the content root.
			if err != nil {
				return err
			}
			out, err := content.Format(b)
			if err != nil {
				fmt.Fprintf(env.Stderr, "%s: %v\n", p, err)
				failed++
				return nil
			}
			if bytes.Equal(b, out) {
				return nil
			}
			changed++
			fmt.Fprintln(env.Stdout, p)
			if *check {
				return nil
			}
			return os.WriteFile(p, out, 0o644) // #nosec G306 G122 G703 -- content files are public.
		})
		if err != nil {
			fmt.Fprintf(env.Stderr, "om fmt: %v\n", err)
			return ExitFailure
		}
	}
	if failed > 0 || (*check && changed > 0) {
		return ExitFindings
	}
	return ExitOK
}

// parseMixed parses flags that come before or after the positional arguments, for example om new problem imo/2027/p1 -source imo-official. It returns the positional arguments.
func parseMixed(fs *flag.FlagSet, args []string) ([]string, error) {
	var pos []string
	for {
		if err := fs.Parse(args); err != nil {
			return nil, err
		}
		if fs.NArg() == 0 {
			return pos, nil
		}
		pos = append(pos, fs.Arg(0))
		args = fs.Args()[1:]
	}
}

// mintTags makes n new tags that are not in registers/tags.tsv.
func mintTags(corpus *content.Corpus, n int) ([]string, error) {
	var out []string
	used := func(t string) bool {
		if _, ok := corpus.Tags[t]; ok {
			return true
		}
		for _, o := range out {
			if o == t {
				return true
			}
		}
		return false
	}
	for range n {
		t, err := tag.Mint(rand.Reader, used)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

// addTag appends a tag with its kind to registers/tags.tsv.
func addTag(root, t, kind string) error {
	f, err := os.OpenFile(filepath.Join(root, "registers", "tags.tsv"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644) // #nosec G302 G304 -- a public register inside the content root.
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(f, "%s\t%s\n", t, kind); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func runTag(env Env, args []string) int {
	if len(args) == 0 {
		fmt.Fprintln(env.Stderr, "om tag: use om tag new [-n N] or om tag check TAG")
		return ExitUsage
	}
	switch args[0] {
	case "new":
		fs := newFlags("tag new", env.Stderr)
		root := fs.String("content", ".", "the root of the content repository")
		n := fs.Int("n", 1, "the number of tags")
		if err := fs.Parse(args[1:]); err != nil {
			return ExitUsage
		}
		corpus, _ := content.Load(*root)
		tags, err := mintTags(corpus, *n)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om tag new: %v\n", err)
			return ExitFailure
		}
		for _, t := range tags {
			fmt.Fprintln(env.Stdout, t)
		}
		return ExitOK
	case "check":
		if len(args) != 2 {
			fmt.Fprintln(env.Stderr, "om tag check: give one tag")
			return ExitUsage
		}
		t, err := tag.Normalize(args[1])
		if err != nil || !tag.IsTag(t) {
			fmt.Fprintf(env.Stdout, "%s is not a valid tag\n", args[1])
			return ExitFindings
		}
		fmt.Fprintf(env.Stdout, "%s is a valid tag (%s), file %s if it is loose\n", t, tag.Display(t), tag.LoosePath(t))
		return ExitOK
	}
	fmt.Fprintf(env.Stderr, "om tag: unknown action %q\n", args[0])
	return ExitUsage
}

func runNew(env Env, args []string) int {
	if len(args) == 0 || (args[0] != "problem" && args[0] != "solution") {
		fmt.Fprintln(env.Stderr, "om new: use om new problem LOCATOR, om new problem -loose, or om new solution LOCATOR|TAG")
		return ExitUsage
	}
	what := args[0]
	fs := newFlags("new "+what, env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	source := fs.String("source", "", "the id of the source in registers/sources.yaml")
	url := fs.String("url", "", "the URL of the text at the source")
	loose := fs.Bool("loose", false, "a problem with no known appearance")
	origin := fs.String("origin", "human", "the origin of a solution: official, human, human+ai or ai")
	pos, err := parseMixed(fs, args[1:])
	if err != nil {
		return ExitUsage
	}
	if len(pos) > 1 {
		fmt.Fprintln(env.Stderr, "om new: give one locator or tag")
		return ExitUsage
	}
	arg := ""
	if len(pos) == 1 {
		arg = pos[0]
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om new: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	if what == "solution" && !slices.Contains(content.SolutionOrigins, *origin) {
		fmt.Fprintf(env.Stderr, "om new: the origin %q is not one of %s\n", *origin, strings.Join(content.SolutionOrigins, ", "))
		return ExitUsage
	}
	tags, err := mintTags(corpus, 1)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om new: %v\n", err)
		return ExitFailure
	}
	t := tags[0]
	var rel, text string
	switch what {
	case "problem":
		rel, text, err = newProblem(corpus, t, arg, *loose, *source, *url)
	case "solution":
		rel, text, err = newSolution(corpus, t, arg, *origin, *source, *url)
	}
	if err != nil {
		fmt.Fprintf(env.Stderr, "om new: %v\n", err)
		return ExitUsage
	}
	if out, err := content.Format([]byte(text)); err == nil {
		text = string(out)
	}
	full := filepath.Join(*root, filepath.FromSlash(rel))
	if _, err := os.Stat(full); err == nil {
		fmt.Fprintf(env.Stderr, "om new: %s exists\n", rel)
		return ExitUsage
	} else if !os.IsNotExist(err) {
		fmt.Fprintf(env.Stderr, "om new: %v\n", err)
		return ExitFailure
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
		fmt.Fprintf(env.Stderr, "om new: %v\n", err)
		return ExitFailure
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil { // #nosec G306 G122 G703 -- content files are public.
		fmt.Fprintf(env.Stderr, "om new: %v\n", err)
		return ExitFailure
	}
	if what == "problem" && !*loose {
		ed, err := newEdition(*root, corpus, arg)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om new: %v\n", err)
			return ExitFailure
		}
		if ed != "" {
			fmt.Fprintln(env.Stdout, ed)
		}
	}
	if err := addTag(*root, t, what); err != nil {
		fmt.Fprintf(env.Stderr, "om new: %v\n", err)
		return ExitFailure
	}
	fmt.Fprintln(env.Stdout, rel)
	return ExitOK
}

func sourceFields(corpus *content.Corpus, id, kind string) (origin, zone string, err error) {
	if id == "" {
		return "", "", fmt.Errorf("give the source with -source")
	}
	src := corpus.Source(id)
	if src == nil {
		return "", "", fmt.Errorf("the source %q is not in registers/sources.yaml", id)
	}
	zone = src.Zone.Statement
	if kind == "solution" {
		zone = src.Zone.Solution
	}
	return src.Origin, zone, nil
}

func newProblem(corpus *content.Corpus, t, at string, loose bool, source, url string) (string, string, error) {
	var rel, appearance string
	switch {
	case loose && at != "":
		return "", "", fmt.Errorf("give a locator or -loose, not both")
	case loose:
		rel = tag.LoosePath(t)
	default:
		l, err := locator.Parse(at)
		if err != nil {
			return "", "", err
		}
		if l.Kind() != locator.Appearance {
			return "", "", fmt.Errorf("%s is not the locator of a problem, for example imo/2027/p1", at)
		}
		if corpus.Competition(l.Comp) == nil {
			return "", "", fmt.Errorf("the competition %q is not in registers/competitions.yaml", l.Comp)
		}
		rel = l.Path()
		appearance = fmt.Sprintf("appearances:\n  - at: %s\n    primary: true\n", l)
	}
	origin, zone, err := sourceFields(corpus, source, "problem")
	if err != nil {
		return "", "", err
	}
	rule := map[string]string{"H1": "official-original", "H2": "curated", "H3": "community", "D": "derived"}[origin]
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntag: %s\nkind: problem\n%stype: proof\ntopics: []\ncanonical:\n  statement: S1\n  rule: %s\nstatements:\n  - id: S1\n    lang: en\n    source: %s\n", t, appearance, rule, source)
	if url != "" {
		fmt.Fprintf(&b, "    url: %s\n", url)
	}
	fmt.Fprintf(&b, "    origin: %s\n    zone: %s\n    kind: transcription\n---\nWrite the statement here.\n", origin, zone)
	return rel, b.String(), nil
}

// newEdition writes the edition file of the locator when it does not exist. The papers come from the layout of the competition in the register. It returns the path of the new file, or "" when the file exists.
func newEdition(root string, corpus *content.Corpus, at string) (string, error) {
	l, err := locator.Parse(at)
	if err != nil {
		return "", err
	}
	rel := "editions/" + l.Comp + "/" + l.Edition + ".yaml"
	full := filepath.Join(root, filepath.FromSlash(rel))
	if _, err := os.Stat(full); err == nil {
		return "", nil
	}
	var b strings.Builder
	year := 0
	for _, r := range l.Edition {
		if r < '0' || r > '9' || year >= 1000 {
			break
		}
		year = year*10 + int(r-'0')
	}
	fmt.Fprintf(&b, "year: %d\n", year)
	if c := corpus.Competition(l.Comp); c != nil && len(c.Shape(year).Layout) > 0 {
		b.WriteString("papers:\n")
		for _, p := range c.Shape(year).Layout {
			fmt.Fprintf(&b, "  - name: %s\n    problems: [%s]\n", p.Name, strings.Join(p.Positions, ", "))
		}
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
		return "", err
	}
	return rel, os.WriteFile(full, []byte(b.String()), 0o644) // #nosec G306 G122 G703 -- content files are public.
}

func newSolution(corpus *content.Corpus, t, ref, origin, source, url string) (string, string, error) {
	p := corpus.ByTag(ref)
	if p == nil {
		for _, q := range corpus.All() {
			if q.Locator() == strings.Trim(ref, "/") {
				p = q
				break
			}
		}
	}
	if p == nil {
		return "", "", fmt.Errorf("no problem has the tag or the locator %q", ref)
	}
	zone := "Z1"
	if source != "" {
		var err error
		if _, zone, err = sourceFields(corpus, source, "solution"); err != nil {
			return "", "", err
		}
	}
	rel := "solutions/" + strings.TrimSuffix(strings.TrimPrefix(p.Path, "problems/"), ".md") + "/" + t + ".md"
	var b strings.Builder
	fmt.Fprintf(&b, "---\ntag: %s\nkind: solution\nproblem: %s\norigin: %s\nauthor:\n  display: Your name\n", t, p.Tag, origin)
	if source != "" {
		fmt.Fprintf(&b, "source: %s\n", source)
	}
	if url != "" {
		fmt.Fprintf(&b, "url: %s\n", url)
	}
	fmt.Fprintf(&b, "zone: %s\nlang: en\ncomplete: true\n", zone)
	if origin == "ai" || origin == "human+ai" {
		b.WriteString("run:\n  model: \n  prompt: \n  date: \n  run_id: \n  transcript: \n")
	}
	b.WriteString("---\nWrite the solution here.\n")
	return rel, b.String(), nil
}
