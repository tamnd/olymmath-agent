// Package review reads and writes the review records of spec 09.5 and the review queue that om import writes.
package review

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/canon"
	"github.com/tamnd/olymmath-agent/match"
)

// Dir is the directory of the reviews in the content repository.
const Dir = "reviews"

// QueueFile is the review queue in the content repository. Each import writes it again.
const QueueFile = "reports/review/queue.jsonl"

// MergesFile records each appearance that an import merged into a problem of another appearance, with its signals.
const MergesFile = "registers/merges.tsv"

// Decisions are the decisions of a review. For a pair: same (one problem), different (two problems) or variant (two problems with a variant_of edge). For a doubt or a dispute: keep (the canonical text stays), replace (the other text becomes canonical) or reject (the other text is wrong or is not this problem). For a merge: accept or split.
var Decisions = []string{"same", "different", "variant", "keep", "replace", "reject", "accept", "split"}

// Review is one decision of a person.
type Review struct {
	ID       string   `yaml:"id"`
	Kind     string   `yaml:"kind"`
	Records  []string `yaml:"records"`
	Decision string   `yaml:"decision"`
	By       string   `yaml:"by"`
	Date     string   `yaml:"date"`
	Note     string   `yaml:"note,omitempty"`
}

var idRE = regexp.MustCompile(`^[0-9a-f]{12}$`)

// Validate tells what is wrong with a review, or returns nil.
func (r *Review) Validate() error {
	switch {
	case !idRE.MatchString(r.ID):
		return fmt.Errorf("the id %q is not 12 hexadecimal digits", r.ID)
	case !slices.Contains(Decisions, r.Decision):
		return fmt.Errorf("the decision %q is not one of %s", r.Decision, strings.Join(Decisions, ", "))
	case r.By == "":
		return fmt.Errorf("the review has no reviewer")
	case len(r.Records) == 0:
		return fmt.Errorf("the review has no records")
	}
	if _, err := time.Parse(time.DateOnly, r.Date); err != nil {
		return fmt.Errorf("the date %q is not yyyy-mm-dd", r.Date)
	}
	return nil
}

// Path gives the file of a review, relative to the content root.
func (r *Review) Path() string {
	return filepath.ToSlash(filepath.Join(Dir, r.Date[:4], r.Date[5:7], r.ID+".yaml"))
}

// Load reads all reviews under root. It returns the reviews by id, and an error for each file that is not valid.
func Load(root string) (map[string]*Review, []error) {
	out := map[string]*Review{}
	var errs []error
	_ = filepath.WalkDir(filepath.Join(root, Dir), func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return nil
		}
		b, err := os.ReadFile(p) // #nosec G122 G304 -- the path is inside the content root.
		if err != nil {
			errs = append(errs, err)
			return nil
		}
		var r Review
		if err := yaml.Unmarshal(b, &r); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
			return nil
		}
		if err := r.Validate(); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", p, err))
			return nil
		}
		out[r.ID] = &r
		return nil
	})
	return out, errs
}

// Write writes a review under root.
func Write(root string, r *Review) error {
	if err := r.Validate(); err != nil {
		return err
	}
	full := filepath.Join(root, filepath.FromSlash(r.Path()))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil { // #nosec G301 -- reviews are public.
		return err
	}
	b, err := yaml.Marshal(r)
	if err != nil {
		return err
	}
	return os.WriteFile(full, b, 0o644) // #nosec G306 -- reviews are public.
}

// Queue reads the review queue under root. A missing queue is empty.
func Queue(root string) ([]canon.Item, error) {
	f, err := os.Open(filepath.Join(root, filepath.FromSlash(QueueFile))) // #nosec G304 -- the path is inside the content root.
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	var out []canon.Item
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<20), 1<<24)
	for sc.Scan() {
		var it canon.Item
		if err := json.Unmarshal(sc.Bytes(), &it); err != nil {
			return nil, fmt.Errorf("%s: %w", QueueFile, err)
		}
		out = append(out, it)
	}
	return out, sc.Err()
}

// Merge is one row of the merges register.
type Merge struct {
	Line   int
	At     string
	Into   string
	K      string
	T      float64
	A      string
	Rule   string
	Review string
	// E is the signal E of the merge, or 0 for a row with no E column.
	E float64
}

// ID gives the id of the review of the merge.
func (m Merge) ID() string { return canon.ItemID("merge", m.Into, m.At) }

// Strict tells if the merge has two strict signals (spec 06.4).
func (m Merge) Strict() bool {
	s := match.Signals{T: m.T, E: m.E}
	if m.K == "strong" {
		s.K = match.StrongKey
	}
	if m.A == "match" {
		s.A = match.AnswerMatch
	}
	return match.Strict(s)
}

// Merges reads the merges register under root. A missing register is empty. A row has seven columns, or eight when it has the signal E.
func Merges(root string) ([]Merge, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(MergesFile))) // #nosec G304 -- the path is inside the content root.
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Merge
	for i, line := range strings.Split(string(b), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 7 && len(f) != 8 {
			return nil, fmt.Errorf("%s:%d: %d columns, want 7 or 8", MergesFile, i+1, len(f))
		}
		m := Merge{Line: i + 1, At: f[0], Into: f[1], K: f[2], A: f[4], Rule: f[5], Review: f[6]}
		if _, err := fmt.Sscanf(f[3], "%g", &m.T); err != nil {
			return nil, fmt.Errorf("%s:%d: the signal T %q is not a number", MergesFile, i+1, f[3])
		}
		if len(f) == 8 {
			if _, err := fmt.Sscanf(f[7], "%g", &m.E); err != nil {
				return nil, fmt.Errorf("%s:%d: the signal E %q is not a number", MergesFile, i+1, f[7])
			}
		}
		out = append(out, m)
	}
	return out, nil
}
