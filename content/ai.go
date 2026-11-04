package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Judgement is one grade of one grader for a solution (spec 04.4 and 07.4). A grade in the grades list of a solution names its rubric and its date. A grader in the graders list of a jury grade takes them from the jury grade when it does not name them.
type Judgement struct {
	// By is the grader: model:<model id> for a model, human:reviewer/<person id> for a person.
	By    string `yaml:"by" json:"by"`
	Score int    `yaml:"score" json:"score"`
	// Max is the top score of the scale. It is 7 when it is 0.
	Max    int    `yaml:"max,omitempty" json:"max,omitempty"`
	Rubric string `yaml:"rubric,omitempty" json:"rubric,omitempty"`
	At     string `yaml:"at,omitempty" json:"at,omitempty"`
	// NotesHash is the SHA-256 of the notes of the grader. The notes are in the transcript of the run.
	NotesHash string `yaml:"notes_hash,omitempty" json:"notes_hash,omitempty"`
	// Revised is true when the grader changed the score after it read the notes of the other graders.
	Revised bool `yaml:"revised,omitempty" json:"revised,omitempty"`
}

// Grade is the grade of a jury (spec 07.4). The score is the result of the rule, for example the minimum of the scores of the graders.
type Grade struct {
	Scale   string      `yaml:"scale" json:"scale"`
	Score   int         `yaml:"score" json:"score"`
	Bucket  string      `yaml:"bucket" json:"bucket"`
	Rubric  string      `yaml:"rubric" json:"rubric"`
	At      string      `yaml:"at" json:"at"`
	Graders []Judgement `yaml:"graders" json:"graders"`
	Rule    string      `yaml:"rule" json:"rule"`
	// Queue is true when the graders still differ by more than one point after the revision, so a person must grade the solution.
	Queue bool `yaml:"queue,omitempty" json:"queue,omitempty"`
	// RunID is the run of om grade that made the grade.
	RunID string `yaml:"run_id,omitempty" json:"run_id,omitempty"`
}

// Scales are the grading scales, with their top score.
var Scales = map[string]int{"imo-7": 7}

// GradeRules are the rules that make the score of a jury from the scores of its graders.
var GradeRules = []string{"min-of-jury-v1"}

// Buckets are the buckets of a grade, best first (spec 07.4).
var Buckets = []string{"correct", "almost", "partial", "incorrect"}

// Bucket gives the bucket of a score out of top: correct for full marks, almost for 5 or 6 out of 7, partial for 1 to 4 out of 7, and incorrect for 0. A score on another scale is first scaled to 7.
func Bucket(score, top int) string {
	if top <= 0 {
		top = 7
	}
	s := Seven(score, top)
	switch {
	case score >= top:
		return "correct"
	case s >= 5:
		return "almost"
	case s >= 1:
		return "partial"
	}
	return "incorrect"
}

// Seven scales a score out of top to a score out of 7, rounded down, so that a score below full marks never becomes 7.
func Seven(score, top int) int {
	if top <= 0 || top == 7 {
		return score
	}
	if score >= top {
		return 7
	}
	return score * 7 / top
}

// IsModel tells if a grader is a model.
func IsModel(by string) bool { return strings.HasPrefix(by, "model:") }

// HumanGrader gives the person id of a grader of the form human:reviewer/<id>, or "".
func HumanGrader(by string) string {
	id, ok := strings.CutPrefix(by, "human:reviewer/")
	if !ok {
		return ""
	}
	return id
}

// All returns each judgement of the solution: the grades list, then the graders of the jury grade. A grader of the jury grade gets the rubric and the date of the jury grade when it has none.
func (s *Solution) All() []Judgement {
	js := append([]Judgement(nil), s.Grades...)
	if g := s.Grade; g != nil {
		for _, j := range g.Graders {
			if j.Rubric == "" {
				j.Rubric = g.Rubric
			}
			if j.At == "" {
				j.At = g.At
			}
			if j.Max == 0 {
				j.Max = Scales[g.Scale]
			}
			js = append(js, j)
		}
	}
	return js
}

// HumanGraded tells if a person graded the solution. Only then can the site show it as verified (spec 07.4).
func (s *Solution) HumanGraded() bool {
	for _, j := range s.All() {
		if HumanGrader(j.By) != "" {
			return true
		}
	}
	return false
}

var rubricRefRE = regexp.MustCompile(`^(.+)@r([1-9][0-9]*)$`)

// ParseRubricRef cuts a rubric reference such as imo/2024/p1@r2 into the locator of the rubric and its version.
func ParseRubricRef(ref string) (loc string, version int, err error) {
	m := rubricRefRE.FindStringSubmatch(ref)
	if m == nil {
		return "", 0, fmt.Errorf("the rubric %q is not of the form <locator>@r<version>", ref)
	}
	v, _ := strconv.Atoi(m[2])
	return m[1], v, nil
}

// Rubric is the grading rubric of one problem (spec 07.4): the checkpoints with their points, the items that get no credit and the usual deductions. The file is rubrics/<locator>.md, and its body holds the notes for the graders.
type Rubric struct {
	Problem string `yaml:"problem"`
	// Version goes up by one at each change of the checkpoints. A grade names the version that it used.
	Version int    `yaml:"version"`
	Scale   string `yaml:"scale"`
	// Source is official when the rubric is a public marking scheme, drafted when a model drafted it, and written when a person wrote it.
	Source string `yaml:"source"`
	URL    string `yaml:"url,omitempty"`
	// DraftedBy names the model that drafted the rubric.
	DraftedBy string `yaml:"drafted_by,omitempty"`
	// Status is draft until a person reviews the rubric. A draft rubric is not used at scale.
	Status      string       `yaml:"status"`
	ReviewedBy  string       `yaml:"reviewed_by,omitempty"`
	Checkpoints []Checkpoint `yaml:"checkpoints"`
	ZeroCredit  []string     `yaml:"zero_credit"`
	Deductions  []Checkpoint `yaml:"deductions"`

	Notes string `yaml:"-"`
	Path  string `yaml:"-"`
	// Locator is the locator of the problem, or its tag for a loose problem.
	Locator string `yaml:"-"`
}

// Checkpoint is one item of a rubric with its points. A deduction has negative points.
type Checkpoint struct {
	Points int    `yaml:"points"`
	Text   string `yaml:"text"`
}

// RubricSources and RubricStatuses are the valid values of the fields source and status of a rubric.
var (
	RubricSources  = []string{"official", "drafted", "written"}
	RubricStatuses = []string{"draft", "reviewed"}
)

// Ref gives the reference of the rubric for a grade, for example imo/2024/p1@r2.
func (r *Rubric) Ref() string { return r.Locator + "@r" + strconv.Itoa(r.Version) }

// Max gives the top score of the rubric.
func (r *Rubric) Max() int { return Scales[r.Scale] }

// RunRecord is the record of one run of the solver or the grader (spec 07.6): its models, its budget, its token counts and its cost. The file is runs/<id>.json. The transcripts are not in git; each solution names the hash of its transcript.
type RunRecord struct {
	ID string `json:"id"`
	// Kind is solve or grade.
	Kind     string `json:"kind"`
	Pipeline string `json:"pipeline"`
	Started  string `json:"started"`
	Ended    string `json:"ended,omitempty"`
	// Status is running, done, budget (the run stopped at its budget) or failed.
	Status string  `json:"status"`
	Budget float64 `json:"budget_usd"`
	Cost   float64 `json:"cost_usd"`
	Batch  bool    `json:"batch,omitempty"`
	Tokens Tokens  `json:"tokens"`
	// Items are the problems of a solve run, or the solutions of a grade run.
	Items []RunItem `json:"items"`

	Path string `json:"-"`
}

// RunItem is one problem or one solution of a run.
type RunItem struct {
	Problem  string `json:"problem"`
	Solution string `json:"solution,omitempty"`
	Band     int    `json:"band,omitempty"`
	Model    string `json:"model,omitempty"`
	// Stage is the last stage that ended, for example S3. A resumed run starts after it.
	Stage   string  `json:"stage,omitempty"`
	Outcome string  `json:"outcome,omitempty"`
	Tokens  Tokens  `json:"tokens"`
	Cost    float64 `json:"cost_usd"`
	Calls   int     `json:"calls"`
	// Transcript is the SHA-256 of the transcript of the item.
	Transcript string `json:"transcript,omitempty"`
	Note       string `json:"note,omitempty"`
}

// Tokens counts the tokens of model calls.
type Tokens struct {
	Input      int `json:"input" yaml:"input"`
	Output     int `json:"output" yaml:"output"`
	CacheRead  int `json:"cache_read,omitempty" yaml:"cache_read,omitempty"`
	CacheWrite int `json:"cache_write,omitempty" yaml:"cache_write,omitempty"`
}

// Add adds the counts of u to t.
func (t *Tokens) Add(u Tokens) {
	t.Input += u.Input
	t.Output += u.Output
	t.CacheRead += u.CacheRead
	t.CacheWrite += u.CacheWrite
}

// Person is one row of registers/people.yaml: a problem author, a solver, a grader or a contributor.
type Person struct {
	ID     string   `yaml:"id"`
	Name   string   `yaml:"name"`
	GitHub string   `yaml:"github,omitempty"`
	Roles  []string `yaml:"roles"`
	Notes  string   `yaml:"notes,omitempty"`
}

// Has tells if the person has the role.
func (p *Person) Has(role string) bool { return in(p.Roles, role) }

func (l *loader) people() {
	const rel = "registers/people.yaml"
	if !l.exists(rel) {
		return
	}
	b, ok := l.read(rel)
	if !ok {
		return
	}
	var ps []*Person
	if err := strictYAML(b, &ps); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	for _, p := range ps {
		switch {
		case p.ID == "":
			l.add(rel, "", "a person has no id")
		case l.corpus.People[p.ID] != nil:
			l.add(rel, "", "the person %q is in the register two times", p.ID)
		default:
			l.corpus.People[p.ID] = p
		}
	}
}

// rubrics reads rubrics/<locator>.md. It runs after the problems, so that it can check the problem of each rubric.
func (l *loader) rubrics() {
	l.walk("rubrics", l.rubric)
}

func (l *loader) rubric(rel string) {
	b, ok := l.read(rel)
	if !ok {
		return
	}
	front, body, err := splitFront(b)
	if err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	r := &Rubric{Path: rel, Notes: strings.TrimSpace(body)}
	if err := strictYAML(front, r); err != nil {
		l.add(rel, "", "%v", err)
		return
	}
	p := l.corpus.byTag[r.Problem]
	if p == nil {
		l.add(rel, "", "the rubric is for the problem %q, which does not exist", r.Problem)
		return
	}
	r.Locator = p.Locator()
	if want := "rubrics/" + r.Locator + ".md"; rel != want {
		l.add(rel, "", "the path must be %s", want)
		return
	}
	if msg := rubricError(r); msg != "" {
		l.add(rel, "", "%s", msg)
		return
	}
	l.corpus.Rubrics[r.Locator] = r
	p.Rubric = r
}

// rubricError gives the first error of a rubric, or "".
func rubricError(r *Rubric) string {
	top, ok := Scales[r.Scale]
	switch {
	case !ok:
		return fmt.Sprintf("the scale %q is not one of imo-7", r.Scale)
	case r.Version < 1:
		return "the version must be 1 or more"
	case !in(RubricSources, r.Source):
		return fmt.Sprintf("the source %q is not one of %s", r.Source, strings.Join(RubricSources, ", "))
	case !in(RubricStatuses, r.Status):
		return fmt.Sprintf("the status %q is not one of %s", r.Status, strings.Join(RubricStatuses, ", "))
	case r.Status == "reviewed" && HumanGrader(r.ReviewedBy) == "":
		return "a reviewed rubric names its reviewer as human:reviewer/<id>"
	case r.Source == "drafted" && !IsModel(r.DraftedBy):
		return "a drafted rubric names its model as model:<id> in drafted_by"
	case len(r.Checkpoints) == 0:
		return "the rubric has no checkpoints"
	}
	sum := 0
	for _, c := range r.Checkpoints {
		if c.Points <= 0 || strings.TrimSpace(c.Text) == "" {
			return "each checkpoint has a text and positive points"
		}
		sum += c.Points
	}
	if sum != top {
		return fmt.Sprintf("the checkpoints give %d points, want %d", sum, top)
	}
	for _, d := range r.Deductions {
		if d.Points >= 0 || strings.TrimSpace(d.Text) == "" {
			return "each deduction has a text and negative points"
		}
	}
	return ""
}

// runs reads the run records in runs/.
func (l *loader) runs() {
	dir := filepath.Join(l.root, "runs")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".json") {
			return nil
		}
		rel, _ := filepath.Rel(l.root, path)
		rel = filepath.ToSlash(rel)
		b, ok := l.read(rel)
		if !ok {
			return nil
		}
		r := &RunRecord{Path: rel}
		if err := json.Unmarshal(b, r); err != nil {
			l.add(rel, "", "JSON: %v", err)
			return nil
		}
		if r.ID == "" || strings.TrimSuffix(filepath.Base(rel), ".json") != r.ID {
			l.add(rel, "", "the name of the file must be the id of the run")
			return nil
		}
		l.corpus.Runs[r.ID] = r
		return nil
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		l.add("runs", "", "%v", err)
	}
}

// RunIDs returns the ids of the run records, sorted.
func (c *Corpus) RunIDs() []string {
	ids := make([]string, 0, len(c.Runs))
	for id := range c.Runs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// ReadRun reads a run record from a file.
func ReadRun(path string) (*RunRecord, error) {
	b, err := os.ReadFile(path) // #nosec G304 -- the caller names the file.
	if err != nil {
		return nil, err
	}
	r := &RunRecord{Path: path}
	if err := json.Unmarshal(b, r); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return r, nil
}

// WriteRun writes a run record to runs/<id>.json under root.
func WriteRun(root string, r *RunRecord) error {
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Join(root, "runs")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, r.ID+".json"), append(b, '\n'), 0o600)
}
