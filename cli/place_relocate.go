package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/official"
	"github.com/tamnd/olymmath-agent/review"
)

// placeRelocation is a placed problem that moves to another position of its edition, because the official papers place its text there.
type placeRelocation struct {
	p       *content.Problem
	edition string
	from    string
	at      string
	page    int
}

// planRelocate gives the placed problems that move to another position of their edition: a problem with one appearance, in an edition of comps with official papers, whose canonical text Locate places at another position of the edition by its margin. A move goes ahead when no problem keeps the new position: the position has no problem, or its problem moves too. Two problems that go to one position both stay. It also gives a line for each problem that stays, with the reason.
func planRelocate(corpus *content.Corpus, papers official.Papers, owners map[string]*content.Problem, comps []string, from, to int) (rels []placeRelocation, held []string) {
	want := map[string][]placeRelocation{}
	for _, p := range corpus.All() {
		if len(p.Appearances) != 1 {
			continue
		}
		cur := p.Appearances[0].At
		cut := strings.LastIndex(cur, "/")
		if cut < 0 {
			continue
		}
		ed := cur[:cut]
		comp, year, ok := strings.Cut(ed, "/")
		if !ok || len(papers[ed]) == 0 || !slices.Contains(comps, comp) {
			continue
		}
		if y, err := strconv.Atoi(year); err != nil || y < from || y > to {
			continue
		}
		s := p.CanonicalStatement()
		if s == nil || s.Text == "" {
			continue
		}
		pos := papers.Locate(ed, s.Text)
		if pos == "" || ed+"/"+pos == cur {
			continue
		}
		r := placeRelocation{p: p, edition: ed, from: cur, at: ed + "/" + pos}
		for _, sg := range papers[ed] {
			if sg.Position == pos {
				r.page = sg.Page
				break
			}
		}
		want[r.at] = append(want[r.at], r)
	}
	// moving holds the problems that may move. A move whose position another problem keeps is dropped, until no more drop.
	moving := map[string]placeRelocation{}
	for at, rs := range want {
		if len(rs) > 1 {
			for _, r := range rs {
				held = append(held, fmt.Sprintf("%s: %s stays at %s: its text is at %s, which %d problems want", r.edition, r.p.Tag, r.from, at, len(rs)))
			}
			continue
		}
		moving[rs[0].from] = rs[0]
	}
	for changed := true; changed; {
		changed = false
		for k, r := range moving {
			o := owners[r.at]
			if o == nil {
				continue
			}
			if _, ok := moving[r.at]; ok {
				continue
			}
			held = append(held, fmt.Sprintf("%s: %s stays at %s: its text is at %s, which has the problem %s", r.edition, r.p.Tag, r.from, r.at, o.Tag))
			delete(moving, k)
			changed = true
		}
	}
	for _, r := range moving {
		rels = append(rels, r)
	}
	sort.Slice(rels, func(i, j int) bool { return rels[i].from < rels[j].from })
	sort.Strings(held)
	return rels, held
}

// relocatedOwners gives the owners of the positions after the relocations.
func relocatedOwners(owners map[string]*content.Problem, rels []placeRelocation) map[string]*content.Problem {
	out := maps.Clone(owners)
	for _, r := range rels {
		if out[r.from] == r.p {
			delete(out, r.from)
		}
	}
	for _, r := range rels {
		out[r.at] = r.p
	}
	return out
}

// moveRelocated writes each relocated problem at its new position, with the new position as its appearance, and writes a placement review. It reads every file before it removes any, so that two problems can swap their positions.
func moveRelocated(root string, rels []placeRelocation, date string) error {
	news := make([][]byte, len(rels))
	for i, r := range rels {
		b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(r.p.Path))) // #nosec G304 -- the path is inside the content root.
		if err != nil {
			return err
		}
		if news[i], err = relocateFront(b, r.from, r.at); err != nil {
			return fmt.Errorf("%s: %w", r.p.Path, err)
		}
	}
	for _, r := range rels {
		if err := os.Remove(filepath.Join(root, filepath.FromSlash(r.p.Path))); err != nil {
			return err
		}
	}
	for i, r := range rels {
		full := filepath.Join(root, "problems", filepath.FromSlash(r.at)+".md")
		if _, err := os.Stat(full); err == nil {
			return fmt.Errorf("%s exists", full)
		}
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- content directories are public.
			return err
		}
		if err := os.WriteFile(full, news[i], 0o644); err != nil { // #nosec G306 G703 -- content files are public.
			return err
		}
		sum := sha256.Sum256([]byte("placement\x00" + r.at + "\x00" + r.p.Tag))
		where := ""
		if r.page > 0 {
			where = fmt.Sprintf(" (page %d)", r.page)
		}
		rv := &review.Review{
			ID: hex.EncodeToString(sum[:6]), Kind: "placement", Records: []string{r.at}, Decision: "accept", By: "agent", Date: date,
			Note: fmt.Sprintf("Agent decision, not a person's review. The text of the problem %s, which was at %s, agrees best, by a clear margin, with the problem at %s in the official paper of %s%s, and not with the problem at %s (om place -relocate). The record moved from %s to %s.", r.p.Tag, r.from, r.at, r.edition, where, r.from, r.from, r.at),
		}
		if err := review.Write(root, rv); err != nil {
			return err
		}
	}
	return nil
}

// relocateFront gives the problem file b with its appearance at from changed to at. The body does not change.
func relocateFront(b []byte, from, at string) ([]byte, error) {
	front, body, err := content.SplitFront(b)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(front, &doc); err != nil {
		return nil, err
	}
	m := doc.Content[0]
	done := false
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value != "appearances" {
			continue
		}
		for _, a := range m.Content[i+1].Content {
			for j := 0; j+1 < len(a.Content); j += 2 {
				if a.Content[j].Value == "at" && a.Content[j+1].Value == from {
					a.Content[j+1].Value = at
					done = true
				}
			}
		}
	}
	if !done {
		return nil, fmt.Errorf("no appearance at %s", from)
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
