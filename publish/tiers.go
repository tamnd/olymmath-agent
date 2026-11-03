// Package publish splits the built site into its outlets and uploads the pages that are not static assets to R2 (spec 10.4).
package publish

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

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

// Split takes out of the site directory dir the files that are not static assets. It deletes each redirect page of redirects.json, because the Worker answers these paths. It moves to out, at the same relative path, each file of a competition that is not in tier 1 and each machine copy of a problem page and each API record. The index pages stay. It gives the number of deleted and moved files.
func Split(dir, out string, corpus *content.Corpus, t Tiers) (deleted, moved int, err error) {
	b, err := os.ReadFile(filepath.Join(dir, "redirects.json")) // #nosec G304 -- the path is inside the site directory.
	if err != nil {
		return 0, 0, err
	}
	redirects, err := redirectFiles(b)
	if err != nil {
		return 0, 0, err
	}
	files := make([]string, 0, len(redirects))
	for _, f := range redirects {
		files = append(files, filepath.Join(dir, filepath.FromSlash(f)))
	}
	var gone atomic.Int64
	err = each(len(files), func(i int) error {
		err := os.Remove(files[i])
		if err == nil {
			gone.Add(1)
		} else if os.IsNotExist(err) {
			err = nil
		}
		return err
	})
	deleted = int(gone.Load())
	if err != nil {
		return deleted, moved, err
	}
	rules := Rules(corpus, t)
	for _, f := range slices.Sorted(maps.Keys(rules)) {
		n, err := moveFiles(dir, out, f, rules[f])
		moved += n
		if err != nil {
			return deleted, moved, err
		}
	}
	return deleted, moved, nil
}

// redirectFiles gives the slash path of the page of each redirect in redirects.json.
func redirectFiles(b []byte) ([]string, error) {
	var redirects map[string]string
	if err := json.Unmarshal(b, &redirects); err != nil {
		return nil, fmt.Errorf("redirects.json: %w", err)
	}
	files := make([]string, 0, len(redirects))
	for from := range redirects {
		files = append(files, strings.TrimPrefix(from, "/")+".html")
	}
	return files, nil
}

// Rule tells which files of one top folder of the site go to R2.
type Rule int

const (
	// Keep moves no file.
	Keep Rule = iota
	// Copies moves the machine copies of the pages.
	Copies
	// All moves each file.
	All
)

// Moves tells if the rule moves the file with the base name name. An index page never moves.
func (r Rule) Moves(name string) bool {
	if strings.HasPrefix(name, "index.") {
		return false
	}
	switch r {
	case All:
		return true
	case Copies:
		return Copy(name)
	}
	return false
}

// Rules gives the rule of Split for each top folder: All for a competition that is not in tier 1 and for the API records, and Copies for a competition in tier 1 and for the tag pages under p. The other folders keep their files.
func Rules(corpus *content.Corpus, t Tiers) map[string]Rule {
	rules := map[string]Rule{}
	add := func(f string, r Rule) { rules[f] = max(rules[f], r) }
	for _, c := range corpus.Competitions {
		if t.First(c) {
			add(c.Code, Copies)
		} else {
			add(c.Code, All)
		}
	}
	add("p", Copies)
	// The API records go to R2 too, but the index of the API stays.
	add("api", All)
	return rules
}

// moveFiles moves the files under dir/sub that the rule moves to out, at the same relative path.
func moveFiles(dir, out, sub string, rule Rule) (int, error) {
	all := rule == All
	moved := 0
	made := map[string]bool{}
	mkdir := func(d string) error {
		if made[d] {
			return nil
		}
		if err := os.MkdirAll(d, 0o755); err != nil { // #nosec G301 -- the site is public.
			return err
		}
		made[d] = true
		return nil
	}
	root := filepath.Join(dir, sub)
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		to := filepath.Join(out, rel)
		if d.IsDir() {
			if !all || p == root {
				return nil
			}
			// A folder with no copy in out moves with one rename, and its index pages come back. Most files of a folder are pages, so this is much faster than a rename of each file.
			if _, err := os.Lstat(to); !os.IsNotExist(err) {
				return err
			}
			if err := mkdir(filepath.Dir(to)); err != nil {
				return err
			}
			n, err := moveDir(p, to)
			moved += n
			if err != nil {
				return err
			}
			return filepath.SkipDir
		}
		if !rule.Moves(d.Name()) {
			return nil
		}
		if err := mkdir(filepath.Dir(to)); err != nil {
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

// moveDir renames the folder from to to, which does not exist. It then puts back the index pages and makes again each folder at from, and deletes the folders at to that have no file. Thus the result is the same as a move of each file that is not an index page. It gives the number of moved files.
func moveDir(from, to string) (int, error) {
	if err := os.Rename(from, to); err != nil { // #nosec G122 -- the build directory is our own output.
		return 0, err
	}
	moved := 0
	var dirs []string
	err := filepath.WalkDir(to, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(to, p)
		if err != nil {
			return err
		}
		back := filepath.Join(from, rel)
		if d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return err
			}
			dirs = append(dirs, p)
			return os.Mkdir(back, info.Mode().Perm())
		}
		if !All.Moves(d.Name()) {
			return os.Rename(p, back) // #nosec G122 -- the build directory is our own output.
		}
		moved++
		return nil
	})
	// A folder with only index pages had no move, so it goes. os.Remove does not delete a folder with files.
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i])
	}
	return moved, err
}

// each calls f for each i from 0 to n-1, with one worker for each CPU, and gives the first error.
func each(n int, f func(i int) error) error {
	var next atomic.Int64
	var wg sync.WaitGroup
	var mu sync.Mutex
	var first error
	for range min(max(runtime.NumCPU(), 1), n) {
		wg.Go(func() {
			for {
				i := int(next.Add(1)) - 1
				if i >= n {
					return
				}
				if err := f(i); err != nil {
					mu.Lock()
					if first == nil {
						first = err
					}
					mu.Unlock()
					return
				}
			}
		})
	}
	wg.Wait()
	return first
}

// Fit keeps the number of static assets in dir at most limit, because a Worker version can have a limited number of asset files. When there are more, it moves the problem pages of the tier 1 competitions to out, the competition with the most files first, until the rest fits. The index pages stay. It gives the number of moved files and the moved competitions.
func Fit(dir, out string, corpus *content.Corpus, limit int) (int, []string, error) {
	total, _, err := countFiles(dir)
	if err != nil || total <= limit {
		return 0, nil, err
	}
	var folders []Folder
	for _, c := range corpus.Competitions {
		n, m, err := countFiles(filepath.Join(dir, c.Code))
		if err != nil {
			return 0, nil, err
		}
		folders = append(folders, Folder{c.Code, n, m})
	}
	codes, ferr := FitOrder(total, folders, limit)
	moved := 0
	for _, code := range codes {
		n, err := moveFiles(dir, out, code, All)
		moved += n
		if err != nil {
			return moved, codes, err
		}
	}
	return moved, codes, ferr
}

// Folder is the folder of a competition in the site after Split, for Fit.
type Folder struct {
	Code string
	// Files is the number of files in the folder, and Moves the number of files that All moves.
	Files, Moves int
}

// FitOrder gives the competitions that Fit moves to R2, so that at most limit of the total files stay. It takes the folder with the most files first, and it skips the empty folders. It gives an error when the files do not fit after the move of all the folders.
func FitOrder(total int, folders []Folder, limit int) ([]string, error) {
	if total <= limit {
		return nil, nil
	}
	var fs []Folder
	for _, f := range folders {
		if f.Files > 0 {
			fs = append(fs, f)
		}
	}
	slices.SortFunc(fs, func(a, b Folder) int {
		if a.Files != b.Files {
			return b.Files - a.Files
		}
		return strings.Compare(a.Code, b.Code)
	})
	moved := 0
	var codes []string
	for _, f := range fs {
		if total-moved <= limit {
			break
		}
		moved += f.Moves
		codes = append(codes, f.Code)
	}
	if total-moved > limit {
		return codes, fmt.Errorf("the site has %d static files after the move, more than %d", total-moved, limit)
	}
	return codes, nil
}

// countFiles counts the files under dir, and the files under dir that All moves. A missing dir has 0 files.
func countFiles(dir string) (files, moves int, err error) {
	err = filepath.WalkDir(dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		if !d.IsDir() {
			files++
			if All.Moves(d.Name()) {
				moves++
			}
		}
		return nil
	})
	return files, moves, err
}

// Copy tells if a file is a machine copy of a page: a .md or a .json file.
func Copy(name string) bool {
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".json")
}

// Key gives the R2 key of a file at the relative path rel: the slash form of the path.
func Key(rel string) string { return path.Clean(filepath.ToSlash(rel)) }
