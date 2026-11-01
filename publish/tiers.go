// Package publish splits the built site into its outlets and uploads the pages that are not static assets to R2 (spec 10.4).
package publish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
)

// Tiers is registers/site-tiers.yaml. A competition is in tier 1 when its group, its country or its code is in the lists. The problem pages of tier 1 are static assets. The problem pages of the other competitions go to R2.
type Tiers struct {
	Groups    []string `yaml:"groups"`
	Countries []string `yaml:"countries"`
	Codes     []string `yaml:"codes"`
}

// LoadTiers reads the tiers file.
func LoadTiers(file string) (Tiers, error) {
	var t Tiers
	b, err := os.ReadFile(file) // #nosec G304 -- the path comes from the user.
	if err != nil {
		return t, err
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(&t); err != nil {
		return t, fmt.Errorf("%s: %w", file, err)
	}
	return t, nil
}

// First tells if the competition is in tier 1.
func (t Tiers) First(c *content.Competition) bool {
	return slices.Contains(t.Groups, c.Group) || (c.Country != "" && slices.Contains(t.Countries, c.Country)) || slices.Contains(t.Codes, c.Code)
}

// Split takes out of the site directory dir the files that are not static assets. It deletes each redirect page of redirects.json, because the Worker answers these paths. It moves to out, at the same relative path, each file of a competition that is not in tier 1 and each machine copy of a problem page. The index pages stay. It gives the number of deleted and moved files.
func Split(dir, out string, corpus *content.Corpus, t Tiers) (deleted, moved int, err error) {
	b, err := os.ReadFile(filepath.Join(dir, "redirects.json")) // #nosec G304 -- the path is inside the site directory.
	if err != nil {
		return 0, 0, err
	}
	var redirects map[string]string
	if err := json.Unmarshal(b, &redirects); err != nil {
		return 0, 0, fmt.Errorf("redirects.json: %w", err)
	}
	for from := range redirects {
		file := filepath.Join(dir, filepath.FromSlash(strings.TrimPrefix(from, "/")+".html"))
		if err := os.Remove(file); err == nil {
			deleted++
		} else if !os.IsNotExist(err) {
			return deleted, moved, err
		}
	}
	for _, c := range corpus.Competitions {
		n, err := moveFiles(dir, out, c.Code, !t.First(c))
		moved += n
		if err != nil {
			return deleted, moved, err
		}
	}
	n, err := moveFiles(dir, out, "p", false)
	return deleted, moved + n, err
}

// moveFiles moves the files under dir/sub to out, at the same relative path. The index pages stay. When all is false, it moves only the machine copies of the pages.
func moveFiles(dir, out, sub string, all bool) (int, error) {
	moved := 0
	err := filepath.WalkDir(filepath.Join(dir, sub), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() || strings.HasPrefix(d.Name(), "index.") || (!all && !Copy(d.Name())) {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		to := filepath.Join(out, rel)
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil { // #nosec G301 -- the site is public.
			return err
		}
		if err := os.Rename(p, to); err != nil { // #nosec G122 -- the build directory is our own output.
			return err
		}
		moved++
		return nil
	})
	return moved, err
}

// Fit keeps the number of static assets in dir at most limit, because a Worker version can have a limited number of asset files. When there are more, it moves the problem pages of the tier 1 competitions to out, the competition with the most files first, until the rest fits. The index pages stay. It gives the number of moved files and the moved competitions.
func Fit(dir, out string, corpus *content.Corpus, limit int) (int, []string, error) {
	total, err := countFiles(dir)
	if err != nil || total <= limit {
		return 0, nil, err
	}
	type comp struct {
		code string
		n    int
	}
	var comps []comp
	for _, c := range corpus.Competitions {
		n, err := countFiles(filepath.Join(dir, c.Code))
		if err != nil {
			return 0, nil, err
		}
		if n > 0 {
			comps = append(comps, comp{c.Code, n})
		}
	}
	slices.SortFunc(comps, func(a, b comp) int {
		if a.n != b.n {
			return b.n - a.n
		}
		return strings.Compare(a.code, b.code)
	})
	moved := 0
	var codes []string
	for _, c := range comps {
		if total-moved <= limit {
			break
		}
		n, err := moveFiles(dir, out, c.code, true)
		moved += n
		if err != nil {
			return moved, codes, err
		}
		codes = append(codes, c.code)
	}
	if total-moved > limit {
		return moved, codes, fmt.Errorf("the site has %d static files after the move, more than %d", total-moved, limit)
	}
	return moved, codes, nil
}

// countFiles counts the files under dir. A missing dir has 0 files.
func countFiles(dir string) (int, error) {
	n := 0
	err := filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			n++
		}
		return nil
	})
	return n, err
}

// Copy tells if a file is a machine copy of a page: a .md or a .json file.
func Copy(name string) bool {
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".json")
}

// Key gives the R2 key of a file at the relative path rel: the slash form of the path.
func Key(rel string) string { return path.Clean(filepath.ToSlash(rel)) }
