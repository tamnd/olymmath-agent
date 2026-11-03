package formal

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
)

// Vendored is one source in the vendor tree of the content.
type Vendored struct {
	Source *Source
	// Dir is the vendor directory, relative to the lean directory, for example vendor/compfiles/1745e6ced3e6.
	Dir string
	// Roots are the root modules of the source that have a file in Dir.
	Roots []string
}

// FindVendored lists the vendor directories under root/lean/vendor, the newest commit of each source only. The newest is the directory that the most recent import wrote.
func FindVendored(root string) ([]Vendored, error) {
	var out []Vendored
	for _, s := range Order() {
		base := filepath.Join(root, content.LeanDir, "vendor", s.ID)
		ents, err := os.ReadDir(base)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		var best string
		var bestTime int64
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			info, err := e.Info()
			if err != nil {
				return nil, err
			}
			if t := info.ModTime().UnixNano(); best == "" || t > bestTime {
				best, bestTime = e.Name(), t
			}
		}
		if best != "" {
			v := Vendored{Source: s, Dir: path.Join("vendor", s.ID, best)}
			for _, r := range s.Roots {
				if _, err := os.Stat(filepath.Join(base, best, filepath.FromSlash(strings.ReplaceAll(r, ".", "/"))+".lean")); err == nil {
					v.Roots = append(v.Roots, r)
				}
			}
			out = append(out, v)
		}
	}
	return out, nil
}

// Lakefile gives the lakefile.toml of the package Olymmath for the vendored sources. The package pins Mathlib, and each source is one library whose srcDir is its vendor directory.
func Lakefile(vs []Vendored, extra []string) string {
	var b strings.Builder
	b.WriteString(`# The Lake package Olymmath (spec 08.3). om formal lakefile writes this file. Do not edit it by hand.
name = "Olymmath"
defaultTargets = [`)
	for i, v := range vs {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(&b, "%q", v.Source.Lib)
	}
	b.WriteString(`]

[leanOptions]
pp.unicode.fun = true
autoImplicit = false
relaxedAutoImplicit = false

[[require]]
name = "mathlib"
git = "https://github.com/leanprover-community/mathlib4"
rev = "` + Mathlib + `"
`)
	for _, e := range extra {
		b.WriteString("\n" + strings.TrimSpace(e) + "\n")
	}
	for _, v := range vs {
		s := v.Source
		fmt.Fprintf(&b, "\n# %s, %s. Licence: %s.\n[[lean_lib]]\nname = %q\nsrcDir = %q\n", s.Name, s.Repo, s.Licence, s.Lib, v.Dir)
		var roots, globs []string
		for _, r := range v.Roots {
			roots = append(roots, fmt.Sprintf("%q", r))
		}
		for _, r := range s.Roots {
			globs = append(globs, fmt.Sprintf("%q", r+".+"))
		}
		fmt.Fprintf(&b, "roots = [%s]\nglobs = [%s]\n", strings.Join(roots, ", "), strings.Join(globs, ", "))
		if len(s.Options) > 0 {
			keys := make([]string, 0, len(s.Options))
			for k := range s.Options {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			var kv []string
			for _, k := range keys {
				kv = append(kv, k+" = "+s.Options[k])
			}
			fmt.Fprintf(&b, "leanOptions = {%s}\n", strings.Join(kv, ", "))
		}
	}
	return b.String()
}

// Target is one declaration that the checks look at.
type Target struct {
	Module string
	Decl   string
	// Kind is statement or proof.
	Kind string
}

// TargetsFile gives the targets of a source as lines of module, declaration and kind, separated by tabs. The check script on the Lean host reads it.
func TargetsFile(ts []Target) string {
	var b strings.Builder
	for _, t := range ts {
		fmt.Fprintf(&b, "%s\t%s\t%s\n", t.Module, t.Decl, t.Kind)
	}
	return b.String()
}

// Targets lists the declarations of the formal and proof records of a source.
func Targets(corpus *content.Corpus, source string) []Target {
	var ts []Target
	for _, f := range corpus.Formals {
		if f.Origin.Source != source {
			continue
		}
		ts = append(ts, Target{Module: f.Module, Decl: f.Decl, Kind: "statement"})
		for _, pr := range f.ProofRecords {
			ts = append(ts, Target{Module: pr.Module, Decl: pr.Decl, Kind: "proof"})
		}
	}
	return ts
}
