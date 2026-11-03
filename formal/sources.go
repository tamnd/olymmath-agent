package formal

import (
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Item is one formal statement of a source, before it is mapped to a problem.
type Item struct {
	// Rel is the Lean file, relative to the root of the source.
	Rel    string
	Module string
	// Native is the name of the problem in the source, for example Imo2024P1.
	Native string
	// Locator is the appearance that the source names, for example imo/2024/p1. It is empty when the source names a competition that olymmath does not map.
	Locator string
	Part    string
	Decl    Decl
	// Answer is the definition of the answer, or nil.
	Answer *Decl
	// Proved is true when the file gives a proof of the statement with no sorry. The check decides if the proof holds.
	Proved bool
}

// Source describes how to read one Lean source.
type Source struct {
	ID      string
	Name    string
	Repo    string
	Licence string
	// Rank is the import rank of spec 08.1. The statement of the best source comes first on the site.
	Rank int
	// Lib is the Lean library of the vendored files, and Roots are its root modules.
	Lib   string
	Roots []string
	// Keep reports whether a file of the source goes to the vendor tree.
	Keep func(rel string) bool
	// Items reads the statements of one file.
	Items func(rel, src string) []Item
	// Options are the Lean options of the library that differ from the package defaults.
	Options map[string]string
	// InMathlib means that the files are part of the pinned Mathlib, so the package builds them from the Mathlib dependency and the lakefile has no library for them. The vendored copy keeps the text that the records quote.
	InMathlib bool
}

// Sources are the Lean sources that om imports, by id.
var Sources = map[string]*Source{
	"compfiles": {
		ID: "compfiles", Name: "Compfiles", Repo: "https://github.com/dwrensha/compfiles", Licence: "Apache-2.0", Rank: 1,
		Lib: "Compfiles", Roots: []string{"ProblemExtraction", "Compfiles"},
		Keep: func(rel string) bool {
			return rel == "LICENSE" || rel == "ProblemExtraction.lean" || strings.HasPrefix(rel, "ProblemExtraction/") || (strings.HasPrefix(rel, "Compfiles/") && strings.HasSuffix(rel, ".lean"))
		},
		Items: compfilesItems,
	},
	"imosllean4": {
		ID: "imosllean4", Name: "IMOSLLean4", Repo: "https://github.com/mortarsanjaya/IMOSLLean4", Licence: "Apache-2.0", Rank: 2,
		Lib: "IMOSLLean4", Roots: []string{"IMOSLLean4"},
		Keep: func(rel string) bool {
			return rel == "LICENSE" || (strings.HasPrefix(rel, "IMOSLLean4/") && strings.HasSuffix(rel, ".lean")) || rel == "IMOSLLean4.lean"
		},
		Items:   imoslItems,
		Options: map[string]string{"autoImplicit": "true"},
	},
	"mathlib-archive-imo": {
		ID: "mathlib-archive-imo", Name: "Mathlib Archive/Imo", Repo: "https://github.com/leanprover-community/mathlib4", Licence: "Apache-2.0", Rank: 3,
		Lib: "Archive", Roots: []string{"Archive"},
		Keep: func(rel string) bool {
			return rel == "LICENSE" || (strings.HasPrefix(rel, "Archive/Imo/") && strings.HasSuffix(rel, ".lean"))
		},
		Items:     archiveItems,
		InMathlib: true,
	},
	"formal-imo": {
		ID: "formal-imo", Name: "formal-imo", Repo: "https://github.com/google-deepmind/formal-imo", Licence: "Apache-2.0", Rank: 4,
		Lib: "Imo", Roots: []string{"Imo"},
		Keep: func(rel string) bool {
			return rel == "LICENSE" || (strings.HasPrefix(rel, "Imo/") && strings.HasSuffix(rel, ".lean"))
		},
		Items: formalImoItems,
	},
	"putnambench": {
		ID: "putnambench", Name: "PutnamBench", Repo: "https://github.com/trishullab/PutnamBench", Licence: "Apache-2.0", Rank: 5,
		Lib: "PutnamBench", Roots: []string{"src"},
		Keep: func(rel string) bool {
			return rel == "LICENSE" || (strings.HasPrefix(rel, "src/") && strings.HasSuffix(rel, ".lean"))
		},
		Items: putnamItems,
	},
}

// Order is the order of the sources by rank.
func Order() []*Source {
	var out []*Source
	for _, s := range Sources {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Rank < out[j].Rank })
	return out
}

// moduleOf gives the module of a Lean file, for example Compfiles.Imo2024P1 for Compfiles/Imo2024P1.lean. A part that does not start with a letter or _ goes in « and », as in Imo.«2005».P3.
func moduleOf(rel string) string {
	parts := strings.Split(strings.TrimSuffix(rel, ".lean"), "/")
	for i, p := range parts {
		if p != "" && !(p[0] == '_' || (p[0] >= 'a' && p[0] <= 'z') || (p[0] >= 'A' && p[0] <= 'Z')) {
			parts[i] = "«" + p + "»"
		}
	}
	return strings.Join(parts, ".")
}

// compfilesComps maps the prefix of a Compfiles name to a competition code. Only the prefixes whose numbers are the positions of the papers are here.
var compfilesComps = map[string]string{"Imo": "imo", "Usa": "usamo", "Egmo": "egmo", "Canada": "ca-cmo", "UK": "uk-bmo1"}

var compfilesRE = regexp.MustCompile(`^([A-Za-z]+?)(\d{4})(R1)?P(\d+)$`)

func compfilesItems(rel, src string) []Item {
	if !strings.HasPrefix(rel, "Compfiles/") || strings.Count(rel, "/") != 1 {
		return nil
	}
	native := strings.TrimSuffix(filepath.Base(rel), ".lean")
	loc := ""
	if m := compfilesRE.FindStringSubmatch(native); m != nil {
		if code, ok := compfilesComps[m[1]]; ok && (m[3] == "") == (m[1] != "UK") {
			loc = code + "/" + m[2] + "/p" + m[4]
		}
	}
	ds := Decls(src)
	var answer *Decl
	var problems []Decl
	for i := range ds {
		switch ds[i].Keyword {
		case "determine":
			if answer == nil {
				answer = &ds[i]
			}
		case "problem":
			problems = append(problems, ds[i])
		}
	}
	return withParts(rel, native, loc, problems, answer, src)
}

// withParts gives one item for each statement of a file. A file with several statements gives parts a, b and so on, or the letter at the end of each name.
func withParts(rel, native, loc string, stmts []Decl, answer *Decl, src string) []Item {
	var out []Item
	for i, d := range stmts {
		it := Item{Rel: rel, Module: moduleOf(rel), Native: native, Locator: loc, Decl: d, Answer: answer, Proved: !HasSorry(d.Body)}
		if len(stmts) > 1 {
			it.Part = partOf(d.Short, i)
		}
		out = append(out, it)
	}
	_ = src
	return out
}

var partRE = regexp.MustCompile(`(?:_|\.)(?:part)?([a-z]|\d)$`)

func partOf(name string, i int) string {
	if m := partRE.FindStringSubmatch(strings.ToLower(name)); m != nil {
		return m[1]
	}
	return string("abcdefghijklmnopqrstuvwxyz"[i%26])
}

// imoslRE matches both layouts of IMOSLLean4: Main/IMO2020/A4.lean and Main/IMO2020/A3/A3.lean.
var imoslRE = regexp.MustCompile(`^IMOSLLean4/Main/IMO(\d{4})/([ACGN]\d+)(?:/[ACGN]\d+)?\.lean$`)

func imoslItems(rel, src string) []Item {
	m := imoslRE.FindStringSubmatch(rel)
	if m == nil {
		return nil
	}
	var stmts []Decl
	for _, d := range Decls(src) {
		if (d.Keyword == "theorem" || d.Keyword == "lemma") && (d.Short == "final_solution" || strings.HasPrefix(d.Short, "final_solution_part")) {
			stmts = append(stmts, d)
		}
	}
	native := "IMO" + m[1] + m[2]
	return withParts(rel, native, "isl/"+m[1]+"/"+strings.ToLower(m[2]), stmts, nil, src)
}

var archiveRE = regexp.MustCompile(`^Archive/Imo/Imo(\d{4})Q(\d+)\.lean$`)

func archiveItems(rel, src string) []Item {
	m := archiveRE.FindStringSubmatch(rel)
	if m == nil {
		return nil
	}
	ds := Decls(src)
	want := "imo" + m[1] + "_q" + m[2]
	var main []Decl
	for _, d := range ds {
		if d.Keyword == "theorem" && (d.Short == want || strings.HasSuffix(d.Name, "."+want)) {
			main = append(main, d)
		}
	}
	if len(main) == 0 {
		for i := len(ds) - 1; i >= 0; i-- {
			if ds[i].Keyword == "theorem" {
				main = []Decl{ds[i]}
				break
			}
		}
	}
	native := "Imo" + m[1] + "Q" + m[2]
	return withParts(rel, native, "imo/"+m[1]+"/p"+m[2], main, nil, src)
}

var formalImoRE = regexp.MustCompile(`^Imo/(\d{4})/P(\d+)\.lean$`)

func formalImoItems(rel, src string) []Item {
	m := formalImoRE.FindStringSubmatch(rel)
	if m == nil {
		return nil
	}
	want := "imo_" + m[1] + "_p" + m[2]
	var main, parts []Decl
	for _, d := range Decls(src) {
		switch {
		case d.Keyword != "theorem":
		case d.Name == want:
			main = append(main, d)
		case strings.HasPrefix(d.Name, want+".parts."):
			parts = append(parts, d)
		}
	}
	if len(main) == 0 {
		main = parts
	}
	var out []Item
	for i, it := range withParts(rel, want, "imo/"+m[1]+"/p"+m[2], main, nil, src) {
		if len(main) > 1 {
			it.Part = strings.TrimPrefix(main[i].Name, want+".parts.")
		}
		it.Proved = false
		out = append(out, it)
	}
	return out
}

var putnamRE = regexp.MustCompile(`^src/putnam_(\d{4})_([ab]\d+)\.lean$`)

func putnamItems(rel, src string) []Item {
	m := putnamRE.FindStringSubmatch(rel)
	if m == nil {
		return nil
	}
	name := "putnam_" + m[1] + "_" + m[2]
	var main []Decl
	var answer *Decl
	ds := Decls(src)
	for i := range ds {
		switch {
		case ds[i].Keyword == "theorem" && ds[i].Name == name:
			main = append(main, ds[i])
		case ds[i].Keyword == "abbrev" && ds[i].Name == name+"_solution":
			answer = &ds[i]
		}
	}
	var out []Item
	for _, it := range withParts(rel, name, "putnam/"+m[1]+"/"+m[2], main, answer, src) {
		it.Proved = false
		out = append(out, it)
	}
	return out
}

// Scan reads the items of a source in the directory dir, the root of a clone of the source. It returns the items and the files to vendor, sorted.
func Scan(s *Source, dir string, read func(string) (string, error)) ([]Item, []string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == ".lake" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		rel = filepath.ToSlash(rel)
		if s.Keep(rel) {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	sort.Strings(files)
	var items []Item
	for _, rel := range files {
		if !strings.HasSuffix(rel, ".lean") {
			continue
		}
		src, err := read(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, nil, err
		}
		items = append(items, s.Items(rel, src)...)
	}
	return items, files, nil
}

// Year returns the year of a locator such as imo/2024/p1, or 0.
func Year(loc string) int {
	parts := strings.Split(loc, "/")
	if len(parts) < 2 {
		return 0
	}
	y, _ := strconv.Atoi(parts[1])
	return y
}
