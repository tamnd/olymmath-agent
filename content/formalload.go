package content

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/tamnd/olymmath-agent/tag"
)

// formals reads the formal and proof records under formal/. A file holds one record. Its kind is formal or proof.
func (l *loader) formals() {
	err := filepath.WalkDir(filepath.Join(l.root, FormalDir), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return nil
		}
		rel, _ := filepath.Rel(l.root, p)
		l.formalFile(filepath.ToSlash(rel))
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		l.add(FormalDir, "", "%v", err)
	}
	byTag := map[string]*Formal{}
	for _, f := range l.corpus.Formals {
		byTag[f.Tag] = f
	}
	proofs := map[string]*Proof{}
	for _, pr := range l.corpus.Proofs {
		f := byTag[pr.Formal]
		if f == nil {
			l.add(pr.File, "F02", "the formal statement %q of the proof does not exist", pr.Formal)
			continue
		}
		if filepath.Dir(pr.File) != filepath.Dir(f.File) {
			l.add(pr.File, "T03", "the proof must be in the directory of its formal statement, %s", filepath.Dir(f.File))
			continue
		}
		proofs[pr.Tag] = pr
	}
	for _, f := range l.corpus.Formals {
		for _, t := range f.Proofs {
			pr := proofs[t]
			if pr == nil {
				l.add(f.File, "F02", "the proof %s is not in formal/", t)
				continue
			}
			if pr.Formal != f.Tag {
				l.add(f.File, "F02", "the proof %s belongs to %s", t, pr.Formal)
				continue
			}
			f.ProofRecords = append(f.ProofRecords, pr)
		}
	}
	for _, p := range l.corpus.all {
		sort.SliceStable(p.Formals, func(i, j int) bool { return p.Formals[i].Rank < p.Formals[j].Rank })
	}
}

// kindOf reads only the tag and the kind of a record.
type kindOf struct {
	Tag  string `yaml:"tag"`
	Kind string `yaml:"kind"`
}

func (l *loader) formalFile(rel string) {
	b, ok := l.read(rel)
	if !ok {
		return
	}
	var k kindOf
	if err := lenientYAML(b, &k); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	if !tag.IsTag(k.Tag) {
		l.add(rel, "T01", "the tag %q is not valid", k.Tag)
		return
	}
	if want := k.Tag + ".yaml"; filepath.Base(rel) != want {
		l.add(rel, "T03", "the name of the file must be %s", want)
		return
	}
	switch k.Kind {
	case "formal":
		f := &Formal{File: rel}
		if err := strictYAML(b, f); err != nil {
			l.add(rel, "", "%v", err)
			return
		}
		p := l.corpus.byTag[f.Problem]
		if p == nil {
			l.add(rel, "L01", "the problem %q does not exist", f.Problem)
			return
		}
		if want := FormalPath(p, f.Tag); rel != want {
			l.add(rel, "T03", "the path must be %s", want)
			return
		}
		p.Formals = append(p.Formals, f)
		l.corpus.Formals = append(l.corpus.Formals, f)
	case "proof":
		pr := &Proof{File: rel}
		if err := strictYAML(b, pr); err != nil {
			l.add(rel, "", "%v", err)
			return
		}
		l.corpus.Proofs = append(l.corpus.Proofs, pr)
	default:
		l.add(rel, "", "the kind is %q, want formal or proof", k.Kind)
	}
}

// fidelityReviews reads the review records of kind fidelity. The review package reads and checks all reviews. The site needs only these.
func (l *loader) fidelityReviews() {
	_ = filepath.WalkDir(filepath.Join(l.root, "reviews"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(p, ".yaml") {
			return nil
		}
		b, err := os.ReadFile(p) // #nosec G122 G304 -- the path is inside the content root.
		if err != nil {
			return nil
		}
		var r FidelityReview
		if lenientYAML(b, &r) != nil || r.Kind != "fidelity" || r.ID == "" {
			return nil
		}
		l.corpus.FidelityReviews[r.ID] = &r
		return nil
	})
}
