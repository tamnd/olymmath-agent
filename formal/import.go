package formal

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/tag"
)

// The toolchain and the Mathlib version that the package pins (spec 08.3).
const (
	Toolchain = "leanprover/lean4:v4.35.0"
	Mathlib   = "v4.35.0"
)

// ShortCommit is the length of the commit in the name of a vendor directory.
const ShortCommit = 12

// VendorDir gives the vendor directory of a source at a commit, relative to the content root.
func VendorDir(source, commit string) string {
	if len(commit) > ShortCommit {
		commit = commit[:ShortCommit]
	}
	return path.Join(content.LeanDir, "vendor", source, commit)
}

// Row is one line of the import report of a source.
type Row struct {
	Native  string
	Decl    string
	Part    string
	Locator string
	// Status is mapped, no-locator or no-problem.
	Status string
	Tag    string
	Proved bool
}

// Report is the result of an import.
type Report struct {
	// Records is false when the import wrote no record.
	Records bool
	Source  string
	Commit  string
	Files   int
	Rows    []Row
	Mapped  int
	Proofs  int
	NewTags int
}

// PortPath is the diff of the port of a source at a commit, relative to the content root.
func PortPath(source, commit string) string {
	if len(commit) > ShortCommit {
		commit = commit[:ShortCommit]
	}
	return path.Join(content.LeanDir, "ports", source, commit+".diff")
}

// ReportPath is the import report of a source in the content.
func ReportPath(source string) string { return path.Join("reports", "formal", source+".tsv") }

// Import copies the Lean files of a source at dir to the vendor tree of the content at root, and writes a formal record for each statement that maps to a problem, and a proof record for each proof. A record that exists for the same source, declaration and part keeps its tag. When records is false, Import copies the files and writes the report, but it writes no record and mints no tag. That is for a source whose files have not compiled yet, since F01 needs a passing check for each record.
func Import(root string, s *Source, dir, commit string, records bool) (*Report, error) {
	corpus, fs := content.Load(root)
	if fs.Errors() > 0 {
		return nil, fmt.Errorf("the content has errors:\n%v", fs)
	}
	items, files, err := Scan(s, dir, func(p string) (string, error) {
		b, err := os.ReadFile(p) // #nosec G304 -- the path is inside the clone of the source.
		return string(b), err
	})
	if err != nil {
		return nil, err
	}
	vendor := VendorDir(s.ID, commit)
	var diff strings.Builder
	for _, rel := range files {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel))) // #nosec G304 -- the path is inside the clone of the source.
		if err != nil {
			return nil, err
		}
		if patch := s.Patches[rel]; patch != nil {
			nb := []byte(patch(string(b)))
			diff.WriteString(Diff(path.Join(vendor, rel), string(b), string(nb)))
			b = nb
		}
		out := filepath.Join(root, filepath.FromSlash(vendor), filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out, b, 0o600); err != nil { // #nosec G703 -- rel comes from a walk of the clone, so it has no .. element.
			return nil, err
		}
	}
	port := ""
	if diff.Len() > 0 {
		port = PortPath(s.ID, commit)
		out := filepath.Join(root, filepath.FromSlash(port))
		if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
			return nil, err
		}
		if err := os.WriteFile(out, []byte(diff.String()), 0o600); err != nil {
			return nil, err
		}
	}
	byLoc := map[string]*content.Problem{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			byLoc[a.At] = p
		}
	}
	type key struct{ source, decl, part string }
	old := map[key]*content.Formal{}
	for _, f := range corpus.Formals {
		old[key{f.Origin.Source, f.Decl, f.Part}] = f
	}
	oldProof := map[string]*content.Proof{}
	for _, pr := range corpus.Proofs {
		oldProof[pr.Formal] = pr
	}
	used := func(t string) bool { _, ok := corpus.Tags[t]; return ok }
	mint := func(kind string) (string, error) {
		t, err := tag.Mint(rand.Reader, used)
		if err != nil {
			return "", err
		}
		corpus.Tags[t] = kind
		return t, nil
	}
	rep := &Report{Source: s.ID, Commit: commit, Files: len(files), Records: records}
	var newTags []string
	for _, it := range items {
		row := Row{Native: it.Native, Decl: it.Decl.Name, Part: it.Part, Locator: it.Locator, Proved: it.Proved}
		p := byLoc[it.Locator]
		switch {
		case it.Locator == "":
			row.Status = "no-locator"
		case p == nil:
			row.Status = "no-problem"
		default:
			row.Status = "mapped"
		}
		if p == nil || !records {
			if p != nil {
				rep.Mapped++
				if it.Proved {
					rep.Proofs++
				}
			}
			rep.Rows = append(rep.Rows, row)
			continue
		}
		f := old[key{s.ID, it.Decl.Name, it.Part}]
		if f == nil {
			t, err := mint("formal")
			if err != nil {
				return nil, err
			}
			newTags = append(newTags, t+"\tformal")
			f = &content.Formal{Tag: t, Fidelity: content.Fidelity{Status: content.Unreviewed}}
		}
		fillFormal(f, s, it, p, vendor, commit)
		f.PortDiff = port
		if it.Proved {
			pr := oldProof[f.Tag]
			if pr == nil {
				t, err := mint("proof")
				if err != nil {
					return nil, err
				}
				newTags = append(newTags, t+"\tproof")
				pr = &content.Proof{Tag: t}
			}
			pr.Kind, pr.Formal, pr.Origin = "proof", f.Tag, f.Origin
			pr.Path, pr.Module, pr.Decl = f.Path, f.Module, f.Decl
			f.Proofs = []string{pr.Tag}
			if err := writeYAML(root, path.Join(path.Dir(content.FormalPath(p, f.Tag)), pr.Tag+".yaml"), pr); err != nil {
				return nil, err
			}
			rep.Proofs++
		}
		if err := writeYAML(root, content.FormalPath(p, f.Tag), f); err != nil {
			return nil, err
		}
		row.Tag = f.Tag
		rep.Mapped++
		rep.Rows = append(rep.Rows, row)
	}
	if err := appendTags(root, newTags); err != nil {
		return nil, err
	}
	rep.NewTags = len(newTags)
	return rep, writeReport(root, rep)
}

func fillFormal(f *content.Formal, s *Source, it Item, p *content.Problem, vendor, commit string) {
	f.Kind, f.Problem, f.Part, f.System = "formal", p.Tag, it.Part, "lean4"
	f.Toolchain, f.Mathlib = Toolchain, Mathlib
	f.Path = path.Join(vendor, it.Rel)
	f.Module, f.Decl = it.Module, it.Decl.Name
	f.Origin = content.FormalOrigin{Kind: "human", Source: s.ID, URL: upstreamURL(s, commit, it.Rel), Commit: commit, NativeName: it.Native}
	f.Licence = s.Licence
	f.Options = s.Options
	text := it.Decl.Text
	f.Answer = &content.FormalAnswer{Style: "none"}
	if it.Answer != nil {
		text = it.Answer.Text + "\n\n" + text
		f.Answer = &content.FormalAnswer{Style: "separate-def", Decl: it.Answer.Name}
	} else if strings.Contains(text, "answer(") {
		f.Answer = &content.FormalAnswer{Style: "embedded"}
	}
	f.Text = Standalone(strings.TrimSpace(text))
	f.Rank = s.Rank
	f.Proofs = nil
}

// Diff gives a unified diff of the lines that differ between a and b, for the file name. It compares line by line, which is enough for a port that replaces lines and keeps their number. When the number of lines differs, the hunk replaces the whole file.
func Diff(name, a, b string) string {
	if a == b {
		return ""
	}
	al, bl := strings.SplitAfter(a, "\n"), strings.SplitAfter(b, "\n")
	var out strings.Builder
	fmt.Fprintf(&out, "--- a/%s\n+++ b/%s\n", name, name)
	if len(al) != len(bl) {
		fmt.Fprintf(&out, "@@ -1,%d +1,%d @@\n", len(al), len(bl))
		for _, l := range al {
			out.WriteString("-" + strings.TrimSuffix(l, "\n") + "\n")
		}
		for _, l := range bl {
			out.WriteString("+" + strings.TrimSuffix(l, "\n") + "\n")
		}
		return out.String()
	}
	for i := range al {
		if al[i] != bl[i] {
			fmt.Fprintf(&out, "@@ -%d +%d @@\n-%s\n+%s\n", i+1, i+1, strings.TrimSuffix(al[i], "\n"), strings.TrimSuffix(bl[i], "\n"))
		}
	}
	return out.String()
}

func upstreamURL(s *Source, commit, rel string) string {
	switch s.ID {
	case "putnambench":
		rel = "lean4/" + rel
	}
	return s.Repo + "/blob/" + commit + "/" + rel
}

func writeYAML(root, rel string, v any) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(v); err != nil {
		return err
	}
	out := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return err
	}
	return os.WriteFile(out, buf.Bytes(), 0o600)
}

func appendTags(root string, lines []string) error {
	if len(lines) == 0 {
		return nil
	}
	p := filepath.Join(root, "registers", "tags.tsv")
	f, err := os.OpenFile(p, os.O_APPEND|os.O_WRONLY, 0o600) // #nosec G304 -- the path is inside the content root.
	if err != nil {
		return err
	}
	if _, err := f.WriteString(strings.Join(lines, "\n") + "\n"); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeReport(root string, rep *Report) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# The import of %s at %s: %d statements, %d mapped to a problem, %d with a proof.\n", rep.Source, rep.Commit, len(rep.Rows), rep.Mapped, rep.Proofs)
	if !rep.Records {
		b.WriteString("# The import wrote no formal record, because the files have not passed a compile check at the pin yet. Rule F01 needs one.\n")
	}
	b.WriteString("# native\tdecl\tpart\tlocator\tstatus\ttag\tproved\n")
	rows := append([]Row(nil), rep.Rows...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Native < rows[j].Native })
	for _, r := range rows {
		fmt.Fprintf(&b, "%s\t%s\t%s\t%s\t%s\t%s\t%v\n", r.Native, r.Decl, r.Part, r.Locator, r.Status, r.Tag, r.Proved)
	}
	out := filepath.Join(root, filepath.FromSlash(ReportPath(rep.Source)))
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return err
	}
	return os.WriteFile(out, []byte(b.String()), 0o600)
}
