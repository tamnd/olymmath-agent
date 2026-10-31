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

// Split takes out of the site directory dir the files that are not static assets. It deletes each redirect page of redirects.json, because the Worker answers these paths. It moves each file of a competition that is not in tier 1 to out, except the index pages, at the same relative path. It gives the number of deleted and moved files.
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
		if t.First(c) {
			continue
		}
		root := filepath.Join(dir, c.Code)
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if os.IsNotExist(err) {
					return filepath.SkipDir
				}
				return err
			}
			if d.IsDir() || strings.HasPrefix(d.Name(), "index.") {
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
		if err != nil {
			return deleted, moved, err
		}
	}
	return deleted, moved, nil
}

// Key gives the R2 key of a file at the relative path rel: the slash form of the path.
func Key(rel string) string { return path.Clean(filepath.ToSlash(rel)) }
