package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/review"
)

func init() {
	register(command{"place", "move each loose problem whose unplaced entry names an edition with official papers of a source to the position where the papers place its text (om place nzmoc)", runPlace})
}

// placeMove is one loose problem that moves to a position.
type placeMove struct {
	p       *content.Problem
	edition string
	at      string
	label   string
	page    int
}

func runPlace(env Env, args []string) int {
	fs := newFlags("place", env.Stderr)
	root := fs.String("content", ".", "the root of the content repository")
	dir, remote := storeFlags(fs)
	from := fs.Int("from", 1959, "the first year")
	to := fs.Int("to", time.Now().Year(), "the last year")
	dry := fs.Bool("dry", false, "print the moves and write no file")
	date := fs.String("date", time.Now().UTC().Format(time.DateOnly), "the date of the placement reviews")
	relocate := fs.Bool("relocate", false, "first move each problem whose one appearance is in an edition of the papers, and whose text the papers place at another position of that edition by a clear margin, to that position, when no other problem keeps it")
	pos, err := parseMixed(fs, args)
	if err != nil {
		return ExitUsage
	}
	if len(pos) != 1 {
		fmt.Fprintln(env.Stderr, "usage: om place [-from YEAR] [-to YEAR] [-dry] SOURCE\nSOURCE is a source whose papers om papers records and om fetch gets into the store, such as nzmoc.")
		return ExitUsage
	}
	corpus, findings := content.Load(*root)
	if findings.Errors() > 0 {
		fmt.Fprintf(env.Stderr, "om place: the content has %d errors. Run om check first.\n", findings.Errors())
		return ExitFindings
	}
	src := corpus.Source(pos[0])
	if src == nil {
		fmt.Fprintf(env.Stderr, "om place: %q is not in the source register\n", pos[0])
		return ExitUsage
	}
	var comps []string
	for _, c := range corpus.Competitions {
		for _, e := range c.Editions {
			if e.Year >= *from && e.Year <= *to && slices.ContainsFunc(e.Official, func(o content.OfficialPaper) bool { return o.Source == src.ID }) {
				comps = append(comps, c.Code)
				break
			}
		}
	}
	if len(comps) == 0 {
		fmt.Fprintf(env.Stderr, "om place: no edition has an official paper of %s. Run om papers %s first.\n", src.ID, src.ID)
		return ExitUsage
	}
	st, err := openStore(*dir, *remote)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om place: %v\n", err)
		return ExitFailure
	}
	papers, _, err := official.LoadEditions(context.Background(), st, corpus, comps)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om place: %v\n", err)
		return ExitFailure
	}
	owners := official.Owners(corpus)
	var rels []placeRelocation
	if *relocate {
		var held []string
		rels, held = planRelocate(corpus, papers, owners, comps, *from, *to)
		for _, h := range held {
			fmt.Fprintln(env.Stdout, h)
		}
		for _, r := range rels {
			fmt.Fprintf(env.Stdout, "%s: %s moves from %s to %s (page %d)\n", r.edition, r.p.Tag, r.from, r.at, r.page)
		}
		owners = relocatedOwners(owners, rels)
	}
	moves, kept := planPlace(corpus, papers, owners, comps, *from, *to)
	for _, k := range kept {
		fmt.Fprintln(env.Stdout, k)
	}
	for _, m := range moves {
		fmt.Fprintf(env.Stdout, "%s: %s moves to %s (page %d)\n", m.edition, m.p.Tag, m.at, m.page)
	}
	if !*dry {
		if err := moveRelocated(*root, rels, *date); err != nil {
			fmt.Fprintf(env.Stderr, "om place: %v\n", err)
			return ExitFailure
		}
		for _, m := range moves {
			if err := movePlaced(*root, m, *date); err != nil {
				fmt.Fprintf(env.Stderr, "om place: %s: %v\n", m.p.Path, err)
				return ExitFailure
			}
		}
	}
	if *relocate {
		fmt.Fprintf(env.Stdout, "%d placed problems moved to another position.\n", len(rels))
	}
	fmt.Fprintf(env.Stdout, "%d loose problems placed, %d left loose.\n", len(moves), len(kept))
	return ExitOK
}

// planPlace gives the loose problems that move: a problem with no appearance whose unplaced entry names an edition of comps with official papers, whose canonical text Locate places at one position of the papers (with the margin of Locate), when no other problem has that position. It also gives a line for each loose problem of these editions that stays loose, with the reason.
func planPlace(corpus *content.Corpus, papers official.Papers, owners map[string]*content.Problem, comps []string, from, to int) (moves []placeMove, kept []string) {
	taken := map[string]string{}
	for _, p := range corpus.All() {
		if len(p.Appearances) > 0 || len(p.Unplaced) == 0 {
			continue
		}
		s := p.CanonicalStatement()
		for _, u := range p.Unplaced {
			comp, year, ok := strings.Cut(u.At, "/")
			if !ok || !slices.Contains(comps, comp) || len(papers[u.At]) == 0 {
				continue
			}
			var y int
			if _, err := fmt.Sscanf(year, "%d", &y); err != nil || y < from || y > to {
				continue
			}
			if s == nil || s.Text == "" {
				kept = append(kept, fmt.Sprintf("%s: %s stays loose: no text", u.At, p.Tag))
				break
			}
			pos := papers.Locate(u.At, s.Text)
			if pos == "" {
				kept = append(kept, fmt.Sprintf("%s: %s stays loose: no position of the papers agrees by a clear margin", u.At, p.Tag))
				break
			}
			at := u.At + "/" + pos
			if o := owners[at]; o != nil {
				kept = append(kept, fmt.Sprintf("%s: %s stays loose: its text is at %s, which has the problem %s", u.At, p.Tag, at, o.Tag))
				break
			}
			if t, ok := taken[at]; ok {
				kept = append(kept, fmt.Sprintf("%s: %s stays loose: its text is at %s, which the loose problem %s takes", u.At, p.Tag, at, t))
				break
			}
			taken[at] = p.Tag
			page := 0
			for _, sg := range papers[u.At] {
				if sg.Position == pos {
					page = sg.Page
					break
				}
			}
			moves = append(moves, placeMove{p: p, edition: u.At, at: at, label: u.Label, page: page})
			break
		}
	}
	sort.Slice(moves, func(i, j int) bool { return moves[i].at < moves[j].at })
	sort.Strings(kept)
	return moves, kept
}

// movePlaced writes the problem of a move at its position, with the position as its primary appearance and with no unplaced entry, removes the loose file, and writes a placement review.
func movePlaced(root string, m placeMove, date string) error {
	old := filepath.Join(root, filepath.FromSlash(m.p.Path))
	b, err := os.ReadFile(old) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	if b, err = placeFront(b, m.at); err != nil {
		return err
	}
	full := filepath.Join(root, "problems", filepath.FromSlash(m.at)+".md")
	if _, err := os.Stat(full); err == nil {
		return fmt.Errorf("%s exists", full)
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
		return err
	}
	if err := os.WriteFile(full, b, 0o644); err != nil { // #nosec G306 G703 -- content files are public.
		return err
	}
	if err := os.Remove(old); err != nil {
		return err
	}
	sum := sha256.Sum256([]byte("placement\x00" + m.at + "\x00" + m.p.Tag))
	label := ""
	if m.label != "" {
		label = fmt.Sprintf(", labeled %s,", m.label)
	}
	where := ""
	if m.page > 0 {
		where = fmt.Sprintf(" (page %d)", m.page)
	}
	r := &review.Review{
		ID: hex.EncodeToString(sum[:6]), Kind: "placement", Records: []string{m.at}, Decision: "accept", By: "agent", Date: date,
		Note: fmt.Sprintf("Agent decision, not a person's review. The text of the loose problem %s%s agrees best, by a clear margin, with the problem at %s in the official paper of %s%s (om place). The record moved from the loose problems to %s.", m.p.Tag, label, m.at, m.edition, where, m.at),
	}
	return review.Write(root, r)
}

// placeFront gives the problem file b with at as its primary appearance and with no unplaced entry. The body does not change.
func placeFront(b []byte, at string) ([]byte, error) {
	front, body, err := content.SplitFront(b)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return nil, err
	}
	m := doc.Content[0]
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == "appearances" {
			return nil, fmt.Errorf("the problem has appearances")
		}
	}
	var list yaml.Node
	if err := list.Encode([]map[string]any{{"at": at, "primary": true}}); err != nil {
		return nil, err
	}
	key := &yaml.Node{Kind: yaml.ScalarNode, Value: "appearances"}
	k := 0
	for i := 0; i+1 < len(m.Content); i += 2 {
		if v := m.Content[i].Value; v == "tag" || v == "kind" {
			k = i + 2
		}
	}
	m.Content = slices.Insert(m.Content, k, key, &list)
	// A problem with a primary appearance has no unplaced entry (rule T03), so all the entries go, also those that name only the competition.
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == "unplaced" {
			m.Content = slices.Delete(m.Content, i, i+2)
			break
		}
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	buf.WriteString("---\n")
	buf.WriteString(body)
	return content.Format(buf.Bytes())
}
