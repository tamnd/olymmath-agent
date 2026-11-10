package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/keymatch"
	"github.com/tamnd/olymmath-agent/lane"
	"github.com/tamnd/olymmath-agent/review"
	"github.com/tamnd/olymmath-agent/seed"
	"github.com/tamnd/olymmath-agent/store"
)

// prIDRE reads the id of a problem from a link to its page on problems.ru.
var prIDRE = regexp.MustCompile(`view_problem_details_new\.php\?id=(\d+)`)

// prTexts reads the listings of problems.ru in the raw store and gives the text of each problem by its id. The texts are a private match key: they never go into a file of the content.
func prTexts(ctx context.Context, st *store.Store) (map[string]string, error) {
	cs, err := st.Captures(problemsRu)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, c := range cs {
		rc, err := st.Get(ctx, c.Hash)
		if err != nil {
			// A capture that is neither on the disk nor in the bucket gives no key.
			continue
		}
		b, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		for _, p := range archive.ParsePRListing(string(b)) {
			if p.Err == nil {
				out[strconv.Itoa(p.ID)] = p.Text
			}
		}
	}
	return out, nil
}

// matchKeys gives a key for each problem of comps with no text whose problems.ru page has a text in texts.
func matchKeys(corpus *content.Corpus, texts map[string]string, comps []string) []keymatch.Key {
	var out []keymatch.Key
	for _, p := range corpus.All() {
		if p.Statement != "" || len(p.Appearances) == 0 {
			continue
		}
		a := p.Appearances[0]
		code := strings.SplitN(a.At, "/", 2)[0]
		if len(comps) > 0 && !slices.Contains(comps, code) {
			continue
		}
		comp := corpus.Competition(code)
		parts := strings.Split(a.At, "/")
		if comp == nil || len(parts) != 3 || len(parts[1]) < 4 {
			continue
		}
		year, err := strconv.Atoi(parts[1][:4])
		if err != nil {
			continue
		}
		for _, s := range p.Statements {
			m := prIDRE.FindStringSubmatch(s.URL)
			if s.Source != problemsRu || m == nil || texts[m[1]] == "" {
				continue
			}
			out = append(out, keymatch.Key{At: a.At, Comp: code, Year: year, Season: comp.EditionForm == "year-season", F: keymatch.Extract(texts[m[1]])})
			break
		}
	}
	return out
}

// licensed tells if the statements of a source can be copied: a source of the register with a zone other than Z3, not model text and not marked do-not-republish.
func licensed(corpus *content.Corpus, r seed.Row) bool {
	src := corpus.Source(r.Source)
	if src == nil || src.Origin == "M" || src.DoNotRepublish || src.Zone.Statement == "Z3" || src.Zone.Statement == "" {
		return false
	}
	a := corpus.Source(r.Author)
	return a == nil || a.Zone.Statement != "Z3"
}

// matchItemID is the id of the review of a doubtful pair of the content match.
func matchItemID(at, ref string) string { return canon.ItemID("match", at, ref) }

// matchQuestion is the prompt of the second check of a doubtful pair. The answer is one word.
func matchQuestion(a, b string) string {
	return "Here are two statements of olympiad problems. One can be a translation of the other. Do they state the same problem, with the same numbers and the same question? A problem and its variant with other numbers or another question are not the same problem. Answer with one word: yes or no.\n\nStatement 1:\n" + a + "\n\nStatement 2:\n" + b + "\n"
}

// yesNo reads the answer of the second check: true for yes, false for no, and ok false for any other answer.
func yesNo(s string) (same, ok bool) {
	w := strings.ToLower(strings.Trim(strings.TrimSpace(s), ".!*\"' \n"))
	switch w {
	case "yes":
		return true, true
	case "no":
		return false, true
	}
	return false, false
}

// matchDecide decides the pairs. A pair with a review of kind match keeps its decision: accept places it and reject does not. A strong pair with no review is placed. A doubtful pair with no review is asked of the lane when ask is not nil, one question at a time, and the answer is written as a review of the agent. A pair with no decision is not placed. keyText gives the key text of an appearance; it goes only to the lane.
func matchDecide(ctx context.Context, root string, pairs []keymatch.Pair, revs map[string]*review.Review, ask lane.Asker, limit int, date string, keyText func(at string) string, log io.Writer) (placed []keymatch.Pair, asked int, err error) {
	for _, p := range pairs {
		id := matchItemID(p.At, p.Ref)
		if r := revs[id]; r != nil {
			if r.Decision == "accept" {
				placed = append(placed, p)
			}
			continue
		}
		if p.Class == keymatch.ClassStrong {
			placed = append(placed, p)
			continue
		}
		if ask == nil || asked >= limit {
			continue
		}
		row, key := p.Text, keyText(p.At)
		if row == "" || key == "" {
			continue
		}
		asked++
		ans, err := ask.Ask(ctx, lane.Question{Prompt: matchQuestion(row, key)})
		if err != nil {
			return placed, asked, err
		}
		same, ok := yesNo(ans.Text)
		if !ok {
			fmt.Fprintf(log, "om import: %s: the lane gave no yes or no\n", p.At)
			continue
		}
		dec, said := "reject", "no"
		if same {
			dec, said = "accept", "yes"
		}
		rv := &review.Review{ID: id, Kind: "match", Records: []string{p.At}, Decision: dec, By: "agent", Date: date,
			Note: fmt.Sprintf("Agent decision, not a person's review. The content match put the seed row %s at this position with a score of %.3f (the next key of the row %.3f, the next row of the key %.3f), below the strong threshold. Asked whether the row and the problems.ru text of the record state the same problem, the %s lane said %s.", p.Ref, p.Score, p.RowNext, p.KeyNext, ans.Lane, said)}
		if err := review.Write(root, rv); err != nil {
			return placed, asked, err
		}
		revs[id] = rv
		if same {
			placed = append(placed, p)
		}
	}
	return placed, asked, nil
}

// matchRow gives the row of the content match for a seed row, or false when the row is not one to match: its source is not licensed, or its labels name no competition of comps and are not one of open.
func matchRow(corpus *content.Corpus, al *seed.Aliases, comps, open []string, r seed.Row) (keymatch.Row, bool) {
	if !licensed(corpus, r) {
		return keymatch.Row{}, false
	}
	code := al.Code(r.Labels)
	switch {
	case slices.Contains(comps, code):
	case (code == "" || code == "-") && len(r.Labels) > 0 && slices.Contains(open, r.Labels[0]):
		code = ""
	default:
		return keymatch.Row{}, false
	}
	f := keymatch.Extract(r.Statement)
	if f.Empty() {
		return keymatch.Row{}, false
	}
	return keymatch.Row{Ref: canon.MatchRef(r), Comp: code, Year: r.Year, F: f, Text: r.Statement}, true
}

// matchOptions are the flags of the content match of om import.
type matchOptions struct {
	open  []string
	out   string
	lane  string
	asks  int
	root  string
	dir   string
	today string
}

// contentMatch runs the content match before an import. It reads the rows once with the rules of the import, to know which rows and which appearances already get a candidate, and compares the other rows of licensed sources with the private keys of the link-only problems of the competitions. It gives the appearance of each row to place, by the reference of the row.
func contentMatch(ctx context.Context, env Env, s *canon.Seeds, st *store.Store, sources []string, o matchOptions) (map[string]string, error) {
	if len(s.Comps) == 0 {
		return nil, fmt.Errorf("-match needs -comp")
	}
	texts, err := prTexts(ctx, st)
	if err != nil {
		return nil, err
	}
	keys := matchKeys(s.Corpus, texts, s.Comps)
	keyText := map[string]string{}
	for _, p := range s.Corpus.All() {
		if len(p.Appearances) == 0 || p.Statement != "" {
			continue
		}
		for _, st := range p.Statements {
			if m := prIDRE.FindStringSubmatch(st.URL); st.Source == problemsRu && m != nil && texts[m[1]] != "" {
				keyText[p.Appearances[0].At] = texts[m[1]]
				break
			}
		}
	}
	mt := keymatch.NewMatcher(keys)
	first := &canon.Seeds{Corpus: s.Corpus, Aliases: s.Aliases, Comps: s.Comps, ThisYear: s.ThisYear, Accept: s.Accept, Locate: s.Locate, Confirm: s.Confirm, Topics: s.Topics}
	rows := 0
	for _, src := range sources {
		err := seed.Read(o.dir, src, func(r seed.Row) error {
			first.Add(r)
			if mr, ok := matchRow(s.Corpus, s.Aliases, s.Comps, o.open, r); ok {
				rows++
				mt.Add(mr)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("%s: %w", src, err)
		}
	}
	placed, taken := map[string]bool{}, map[string]bool{}
	first.Trace = func(r seed.Row, _, at, reason string) {
		if reason == "" && at != "" {
			placed[canon.MatchRef(r)] = true
			taken[at] = true
		}
	}
	first.Candidates()
	var pairs []keymatch.Pair
	ruled := 0
	for _, p := range mt.Pairs() {
		// A row that the rules place, and an appearance that gets a candidate from them, are left to the rules.
		if placed[p.Ref] || taken[p.At] {
			ruled++
			continue
		}
		pairs = append(pairs, p)
	}
	revs, _ := review.Load(o.root)
	var ask lane.Asker
	if o.lane != "" {
		ls, err := lane.Select(lane.Defaults(), o.lane)
		if err != nil {
			return nil, err
		}
		if len(ls) != 1 {
			return nil, fmt.Errorf("-match-lane: name one lane")
		}
		ask = ls[0]
	}
	chosen, asked, err := matchDecide(ctx, o.root, pairs, revs, ask, o.asks, o.today, func(at string) string { return keyText[at] }, env.Stderr)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om import: the lane stopped: %v\n", err)
	}
	out := map[string]string{}
	for _, p := range chosen {
		out[p.Ref] = p.At
	}
	strong := 0
	for _, p := range pairs {
		if p.Class == keymatch.ClassStrong {
			strong++
		}
	}
	fmt.Fprintf(env.Stdout, "content match: %d keys, %d rows, %d pairs left to the rules, %d pairs (%d strong, %d doubtful), %d asked of the lane, %d placed.\n", len(keys), rows, ruled, len(pairs), strong, len(pairs)-strong, asked, len(out))
	if o.out != "" {
		var b strings.Builder
		b.WriteString("at\tref\tclass\tscore\tshared\ttokens\trow_next\tkey_next\tplaced\n")
		for _, p := range pairs {
			fmt.Fprintf(&b, "%s\t%s\t%s\t%.3f\t%.1f\t%d\t%.3f\t%.3f\t%t\n", p.At, p.Ref, p.Class, p.Score, p.Shared, p.Tokens, p.RowNext, p.KeyNext, out[p.Ref] == p.At)
		}
		if err := os.WriteFile(o.out, []byte(b.String()), 0o644); err != nil { // #nosec G306 -- a diagnostic file with no text.
			return nil, err
		}
	}
	return out, nil
}
