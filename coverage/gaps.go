package coverage

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
)

// GapFile is the path of the gap register in the content repository. Each entry gives the reason why some expected positions of a competition have no record.
const GapFile = "registers/gaps.yaml"

// The reasons of an expected position with no record. A position that no entry of the gap register names is open.
const (
	// LinkOnly is a competition that the project only links by rule, and the official site gives no list to link.
	LinkOnly = "link-only"
	// Robots is a competition whose source robots.txt blocks.
	Robots = "robots"
	// NoSource is a position with no allowed source that a person or an agent could find.
	NoSource = "no-source"
	// Open is a position that can still get a record.
	Open = "open"
)

// Reasons are the reasons that the gap register can give.
var Reasons = []string{LinkOnly, Robots, NoSource}

// Gap is one entry of the gap register.
type Gap struct {
	Competition string `yaml:"competition" json:"competition"`
	// Years are years ("2000") or ranges of years ("1970-1999"). No years is every year.
	Years []string `yaml:"years" json:"years,omitempty"`
	// Positions are the positions of the years, as p4. No positions is every position.
	Positions []string `yaml:"positions" json:"positions,omitempty"`
	Reason    string   `yaml:"reason" json:"reason"`
	// Note gives the evidence for the reason.
	Note string `yaml:"note" json:"note"`
	// Count is the number of expected positions with no record that the entry gives the reason of.
	Count int `yaml:"-" json:"count"`

	ranges [][2]int
}

// Gaps is the gap register.
type Gaps struct {
	List   []*Gap
	byComp map[string][]*Gap
}

// LoadGaps reads the gap register of the content repository at root. A repository with no gap register has no gaps. Each competition must be in the corpus, each reason must be one of Reasons, and two entries must not name the same position.
func LoadGaps(root string, corpus *content.Corpus) (*Gaps, error) {
	b, err := os.ReadFile(filepath.Join(root, GapFile)) // #nosec G304 -- the path is inside the content root.
	if errors.Is(err, fs.ErrNotExist) {
		return &Gaps{}, nil
	}
	if err != nil {
		return nil, err
	}
	g, err := ParseGaps(b, corpus)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", GapFile, err)
	}
	return g, nil
}

// ParseGaps reads the gap register from b.
func ParseGaps(b []byte, corpus *content.Corpus) (*Gaps, error) {
	return parseGaps(b, func(code string) bool { return corpus.Competition(code) != nil })
}

// parseGaps reads the gap register from b. known tells if a code is in the competition register.
func parseGaps(b []byte, known func(code string) bool) (*Gaps, error) {
	var list []*Gap
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&list); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}
	g := &Gaps{List: list, byComp: map[string][]*Gap{}}
	for i, e := range list {
		if !known(e.Competition) {
			return nil, fmt.Errorf("entry %d: %q is not in the competition register", i+1, e.Competition)
		}
		if !slices.Contains(Reasons, e.Reason) {
			return nil, fmt.Errorf("entry %d (%s): the reason %q is not one of %s", i+1, e.Competition, e.Reason, strings.Join(Reasons, ", "))
		}
		if strings.TrimSpace(e.Note) == "" {
			return nil, fmt.Errorf("entry %d (%s): the note with the evidence is empty", i+1, e.Competition)
		}
		for _, y := range e.Years {
			r, err := yearRange(y)
			if err != nil {
				return nil, fmt.Errorf("entry %d (%s): %w", i+1, e.Competition, err)
			}
			e.ranges = append(e.ranges, r)
		}
		for _, o := range g.byComp[e.Competition] {
			if overlap(o, e) {
				return nil, fmt.Errorf("entry %d (%s): it names a position that an earlier entry names", i+1, e.Competition)
			}
		}
		g.byComp[e.Competition] = append(g.byComp[e.Competition], e)
	}
	return g, nil
}

func yearRange(s string) ([2]int, error) {
	a, b, found := strings.Cut(strings.TrimSpace(s), "-")
	from, err := strconv.Atoi(strings.TrimSpace(a))
	if err != nil {
		return [2]int{}, fmt.Errorf("the year %q is not a year or a range of years", s)
	}
	to := from
	if found {
		if to, err = strconv.Atoi(strings.TrimSpace(b)); err != nil || to < from {
			return [2]int{}, fmt.Errorf("the year %q is not a year or a range of years", s)
		}
	}
	return [2]int{from, to}, nil
}

func (e *Gap) hasYear(y int) bool {
	if len(e.ranges) == 0 {
		return true
	}
	for _, r := range e.ranges {
		if y >= r[0] && y <= r[1] {
			return true
		}
	}
	return false
}

func (e *Gap) hasPosition(p string) bool {
	return len(e.Positions) == 0 || slices.Contains(e.Positions, p)
}

// overlap tells if two entries of one competition name a common position.
func overlap(a, b *Gap) bool {
	years := len(a.ranges) == 0 || len(b.ranges) == 0
	for _, x := range a.ranges {
		for _, y := range b.ranges {
			if x[0] <= y[1] && y[0] <= x[1] {
				years = true
			}
		}
	}
	if !years {
		return false
	}
	if len(a.Positions) == 0 || len(b.Positions) == 0 {
		return true
	}
	for _, p := range a.Positions {
		if slices.Contains(b.Positions, p) {
			return true
		}
	}
	return false
}

// entry gives the entry for a position of a competition in a year, or nil when no entry names it.
func (g *Gaps) entry(code string, year int, pos string) *Gap {
	if g == nil {
		return nil
	}
	for _, e := range g.byComp[code] {
		if e.hasYear(year) && e.hasPosition(pos) {
			return e
		}
	}
	return nil
}
