// Package calibrate measures the grading jury against the human grades of public labelled sets (spec 07.4): IMO-GradingBench, the Open Proof Corpus and ProofBench. It reads the sets, gives the grading tasks, and compares the jury scores with the human grades by Cohen's kappa.
package calibrate

import (
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/grade"
	"github.com/tamnd/olymmath-agent/seed"
)

// Set is a labelled set of graded proofs.
type Set struct {
	Name string
	// Title is the name of the set in the report.
	Title string
	// Licence is the SPDX id of the licence of the data. A set whose licence does not allow redistribution stays out of git (Keep is false).
	Licence string
	Keep    bool
	// Home is the page of the set, and Files the URLs of its data files. Only small files are named.
	Home  string
	Files []string
	// Binary is true when the human grade is pass or fail, not points out of 7.
	Binary bool
	cols   []string
	item   func(seed.Record) (Item, bool)
}

// Item is one human-graded proof.
type Item struct {
	Set string `json:"set"`
	ID  string `json:"id"`
	// Problem, Solution, Reference and Guidelines are the texts of the item. Reference and Guidelines are empty when the set has none.
	Problem    string `json:"problem"`
	Solution   string `json:"solution"`
	Reference  string `json:"reference,omitempty"`
	Guidelines string `json:"guidelines,omitempty"`
	// Human is the human grade: points from 0 to 7, or 1 for pass and 0 for fail in a binary set.
	Human int `json:"human"`
	// Label is the bucket that the human graders gave, when the set has one.
	Label string `json:"label,omitempty"`
	// Model is the model that wrote the proof, when the set names it.
	Model string `json:"model,omitempty"`
}

// Sets are the calibration sets. The licences were read from the pages of the sets on 2026-11-04.
var Sets = []*Set{
	{
		Name: "gradingbench", Title: "IMO-GradingBench", Licence: "CC-BY-4.0", Keep: true,
		Home:  "https://github.com/google-deepmind/superhuman/tree/main/imobench",
		Files: []string{"https://raw.githubusercontent.com/google-deepmind/superhuman/main/imobench/gradingbench.csv"},
		cols:  []string{"Grading ID", "Problem", "Solution", "Grading guidelines", "Response", "Points", "Reward"},
		item: func(r seed.Record) (Item, bool) {
			p, err := strconv.Atoi(strings.TrimSpace(r.Get("Points")))
			if err != nil || p < 0 || p > 7 {
				return Item{}, false
			}
			return Item{
				ID: r.Get("Grading ID"), Problem: r.Get("Problem"), Solution: r.Get("Response"),
				Reference: r.Get("Solution"), Guidelines: r.Get("Grading guidelines"),
				Human: p, Label: strings.ToLower(strings.TrimSpace(r.Get("Reward"))),
			}, true
		},
	},
	{
		Name: "proofbench", Title: "ProofBench", Licence: "MIT", Keep: true,
		Home:  "https://huggingface.co/datasets/wenjiema02/ProofBench",
		Files: []string{"https://huggingface.co/datasets/wenjiema02/ProofBench/resolve/main/data/train-00000-of-00001.parquet"},
		cols:  []string{"problem_id", "problem", "model_solution", "reference_solution", "marking_scheme", "expert_rating", "generator", "response_number"},
		item: func(r seed.Record) (Item, bool) {
			p, err := strconv.Atoi(strings.TrimSpace(r.Get("expert_rating")))
			if err != nil || p < 0 || p > 7 {
				return Item{}, false
			}
			return Item{
				ID:      r.Get("problem_id") + "/" + r.Get("generator") + "/" + r.Get("response_number"),
				Problem: r.Get("problem"), Solution: r.Get("model_solution"),
				Reference: r.Get("reference_solution"), Guidelines: r.Get("marking_scheme"), Human: p, Model: r.Get("generator"),
			}, true
		},
	},
	{
		// The Open Proof Corpus is CC BY-NC-SA 4.0: om reads it in the work directory and keeps it out of git.
		Name: "opc", Title: "Open Proof Corpus", Licence: "CC-BY-NC-SA-4.0", Keep: false, Binary: true,
		Home: "https://huggingface.co/datasets/INSAIT-Institute/OPC",
		Files: []string{
			"https://huggingface.co/datasets/INSAIT-Institute/OPC/resolve/main/data/test-00000-of-00001.parquet",
			"https://huggingface.co/datasets/INSAIT-Institute/OPC/resolve/main/data/matharena-00000-of-00001.parquet",
		},
		cols: []string{"problem_id", "problem", "solution", "score", "model_id", "ground_truth_solution"},
		item: func(r seed.Record) (Item, bool) {
			// score has one judgement for each human judge, 1 for a correct proof. The human grade is the majority; a tie is not used.
			pass, n := 0, 0
			for _, s := range r["score"] {
				v, err := strconv.Atoi(strings.TrimSpace(s))
				if err != nil {
					continue
				}
				n++
				if v > 0 {
					pass++
				}
			}
			if n == 0 || 2*pass == n {
				return Item{}, false
			}
			h := 0
			if 2*pass > n {
				h = 1
			}
			return Item{
				ID: r.Get("problem_id") + "/" + r.Get("model_id"), Problem: r.Get("problem"), Solution: r.Get("solution"),
				Reference: r.Get("ground_truth_solution"), Human: h, Model: r.Get("model_id"),
			}, true
		},
	},
}

// SetByName gives the set with the name, or nil.
func SetByName(name string) *Set {
	for _, s := range Sets {
		if s.Name == name {
			return s
		}
	}
	return nil
}

// Read reads the items of a data file of the set.
func (s *Set) Read(path string) ([]Item, error) {
	f, err := os.Open(path) // #nosec G304 -- a data file that the caller names.
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	st, err := f.Stat()
	if err != nil {
		return nil, err
	}
	t, err := seed.Open(path, f, st.Size())
	if err != nil {
		return nil, err
	}
	var out []Item
	seen := map[string]int{}
	err = t.Each(s.cols, func(_ int, r seed.Record) error {
		it, ok := s.item(r)
		if !ok || strings.TrimSpace(it.Solution) == "" {
			return nil
		}
		it.Set = s.Name
		// An id that comes twice gets a number, so each item has its own id.
		if n := seen[it.ID]; n > 0 {
			seen[it.ID] = n + 1
			it.ID = fmt.Sprintf("%s#%d", it.ID, n+1)
		} else {
			seen[it.ID] = 1
		}
		out = append(out, it)
		return nil
	})
	return out, err
}

// Task gives the grading task of an item. A juror that wrote the proof is taken out of the jury.
func (it Item) Task(jurors []string) grade.Task {
	return grade.Task{
		ID: it.Set + "/" + it.ID, Statement: it.Problem, Rubric: it.Guidelines, Reference: it.Reference,
		Solution: it.Solution, Generator: it.Model, Jurors: grade.Jurors(jurors, it.Model),
	}
}

// Pass is the least jury score that counts as a pass in a binary set.
const Pass = 7

// Agreement compares the jury scores with the human grades of the items.
type Agreement struct {
	Set string
	N   int
	// Exact is the part of the items where the jury and the humans agree: the same points, or the same pass or fail.
	Exact float64
	// Kappa is Cohen's kappa on the four buckets, or on pass and fail for a binary set.
	Kappa float64
	// Quadratic is the kappa with quadratic weights on the points from 0 to 7. It is NaN for a binary set.
	Quadratic float64
	// Within1 is the part of the items where the points differ by at most 1.
	Within1 float64
	// Low and High are the bounds of the 95% interval of Kappa, from a bootstrap over the items (Interval). They are NaN when the kappa is.
	Low, High float64
}

// Sample keeps n items of the list, drawn at random with the seed, in the order of the list. The same seed gives the same items. It keeps all items when n is 0 or not less than their number.
func Sample(items []Item, n int, seed uint64) []Item {
	if n <= 0 || n >= len(items) {
		return items
	}
	r := rand.New(rand.NewPCG(seed, 0)) // #nosec G404 -- a reproducible sample, not a secret.
	idx := r.Perm(len(items))[:n]
	slices.Sort(idx)
	out := make([]Item, 0, n)
	for _, i := range idx {
		out = append(out, items[i])
	}
	return out
}

// Bootstrap is the number of resamples of Interval.
const Bootstrap = 2000

// Interval gives the 95% interval of Cohen's kappa of two ratings with k categories: the 2.5 and 97.5 percentiles of the kappa of Bootstrap resamples of the items, drawn with a fixed seed. A resample with a kappa that is NaN is left out.
func Interval(a, b []int, k int, quadratic bool) (lo, hi float64) {
	n := len(a)
	if n == 0 || n != len(b) || math.IsNaN(Kappa(a, b, k, quadratic)) {
		return math.NaN(), math.NaN()
	}
	r := rand.New(rand.NewPCG(1, 2)) // #nosec G404 -- a reproducible bootstrap, not a secret.
	ks := make([]float64, 0, Bootstrap)
	ra, rb := make([]int, n), make([]int, n)
	for range Bootstrap {
		for i := range n {
			j := r.IntN(n)
			ra[i], rb[i] = a[j], b[j]
		}
		if v := Kappa(ra, rb, k, quadratic); !math.IsNaN(v) {
			ks = append(ks, v)
		}
	}
	if len(ks) == 0 {
		return math.NaN(), math.NaN()
	}
	slices.Sort(ks)
	at := func(q float64) float64 { return ks[min(len(ks)-1, int(q*float64(len(ks))))] }
	return at(0.025), at(0.975)
}

// Pass reports whether a human grade of an item is a pass: 1 in a binary set, 7 points out of 7 in the other sets.
func (s *Set) Pass(human int) bool {
	if s.Binary {
		return human == 1
	}
	return human == 7
}

// Overall gives the agreement on pass and fail over the items of all sets, so the sets with points and the binary set count together. A pass is 7 points, or a pass of a binary set; a jury pass is a score of Pass. Exact is the part of the items with the same pass or fail.
func Overall(sets []*Set, items map[string][]Item, jury map[string]int) Agreement {
	a := Agreement{Set: "all", Quadratic: math.NaN()}
	var hum, mac []int
	same := 0
	for _, s := range sets {
		for _, it := range items[s.Name] {
			j, ok := jury[s.Name+"/"+it.ID]
			if !ok {
				continue
			}
			h, m := 0, 0
			if s.Pass(it.Human) {
				h = 1
			}
			if j >= Pass {
				m = 1
			}
			hum, mac = append(hum, h), append(mac, m)
			if h == m {
				same++
			}
		}
	}
	a.N = len(hum)
	a.Kappa = Kappa(hum, mac, 2, false)
	a.Low, a.High = Interval(hum, mac, 2, false)
	if a.N > 0 {
		a.Exact = float64(same) / float64(a.N)
	}
	return a
}

// Compare gives the agreement of the jury scores (by item id) with the items. Items with no score are left out.
func Compare(set *Set, items []Item, jury map[string]int) Agreement {
	a := Agreement{Set: set.Name, Quadratic: math.NaN(), Low: math.NaN(), High: math.NaN()}
	var hum, mac, hp, mp []int
	exact, within := 0, 0
	for _, it := range items {
		j, ok := jury[it.ID]
		if !ok {
			continue
		}
		a.N++
		if set.Binary {
			m := 0
			if j >= Pass {
				m = 1
			}
			hum, mac = append(hum, it.Human), append(mac, m)
			if m == it.Human {
				exact++
			}
			continue
		}
		hum = append(hum, bucketIndex(it.Human))
		mac = append(mac, bucketIndex(j))
		hp, mp = append(hp, it.Human), append(mp, j)
		if j == it.Human {
			exact++
		}
		if j-it.Human <= 1 && it.Human-j <= 1 {
			within++
		}
	}
	if a.N == 0 {
		a.Kappa = math.NaN()
		return a
	}
	a.Exact = float64(exact) / float64(a.N)
	if set.Binary {
		a.Kappa = Kappa(hum, mac, 2, false)
		a.Low, a.High = Interval(hum, mac, 2, false)
		return a
	}
	a.Kappa = Kappa(hum, mac, len(content.Buckets), false)
	a.Low, a.High = Interval(hum, mac, len(content.Buckets), false)
	a.Quadratic = Kappa(hp, mp, 8, true)
	a.Within1 = float64(within) / float64(a.N)
	return a
}

// bucketIndex gives the index of the bucket of a score out of 7 in content.Buckets.
func bucketIndex(score int) int {
	b := content.Bucket(score, 7)
	for i, x := range content.Buckets {
		if x == b {
			return i
		}
	}
	return 0
}

// Kappa gives Cohen's kappa of two ratings of the same items, with k categories from 0 to k-1. With quadratic weights a disagreement costs the square of its distance. It is NaN when the expected agreement is total.
func Kappa(a, b []int, k int, quadratic bool) float64 {
	n := len(a)
	if n == 0 || n != len(b) || k < 2 {
		return math.NaN()
	}
	obs := make([][]float64, k)
	for i := range obs {
		obs[i] = make([]float64, k)
	}
	ra, rb := make([]float64, k), make([]float64, k)
	for i := range a {
		if a[i] < 0 || a[i] >= k || b[i] < 0 || b[i] >= k {
			return math.NaN()
		}
		obs[a[i]][b[i]]++
		ra[a[i]]++
		rb[b[i]]++
	}
	var po, pe float64
	for i := range k {
		for j := range k {
			w := 0.0
			if i != j {
				w = 1
				if quadratic {
					d := float64(i - j)
					w = d * d / float64((k-1)*(k-1))
				}
			}
			po += w * obs[i][j] / float64(n)
			pe += w * ra[i] * rb[j] / float64(n*n)
		}
	}
	if pe == 0 {
		return math.NaN()
	}
	return 1 - po/pe
}

// Target is the kappa that the grader must reach before it is used at scale (issue 7).
const Target = 0.8

// Report writes the agreements as Markdown. overall is the agreement on pass and fail over all sets (Overall), or nil. sample says how the items were chosen, or is empty.
func Report(w io.Writer, rows []Agreement, overall *Agreement, jurors []string, source, sample string) {
	fmt.Fprintf(w, "# Grader calibration\n\nThe jury (%s, rule %s) graded the human-graded proofs of the calibration sets. The scores came from %s. The target is a Cohen's kappa of %.1f or more on each set.\n\n", strings.Join(jurors, ", "), grade.Rule, source, Target)
	if sample != "" {
		fmt.Fprintf(w, "%s\n\n", sample)
	}
	fmt.Fprintln(w, "| Set | Licence | Proofs | Exact | Kappa | 95% interval | Quadratic kappa | Within 1 point | Target met |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|")
	for _, a := range rows {
		s := SetByName(a.Set)
		title, lic := a.Set, ""
		if s != nil {
			title, lic = s.Title, s.Licence
		}
		fmt.Fprintf(w, "| %s | %s | %d | %s | %s | %s | %s | %s | %s |\n", title, lic, a.N, pct(a.Exact, a.N), num(a.Kappa), interval(a), num(a.Quadratic), within(a), met(a))
	}
	if overall != nil {
		a := *overall
		fmt.Fprintf(w, "| All sets, pass or fail | | %d | %s | %s | %s | n/a | n/a | %s |\n", a.N, pct(a.Exact, a.N), num(a.Kappa), interval(a), met(a))
	}
	fmt.Fprintln(w, "\nKappa is on the four buckets (correct, almost, partial, incorrect), or on pass and fail for the Open Proof Corpus, where a pass is a jury score of 7. The quadratic kappa is on the points from 0 to 7. The row of all sets is on pass and fail, where a human pass is 7 points or a pass of the Open Proof Corpus. The 95% interval is the 2.5 and 97.5 percentiles of the kappa of 2,000 bootstrap resamples of the proofs.")
}

func met(a Agreement) string {
	if a.Kappa >= Target {
		return "yes"
	}
	return "no"
}

func interval(a Agreement) string {
	if math.IsNaN(a.Low) || math.IsNaN(a.High) {
		return "n/a"
	}
	return fmt.Sprintf("%.3f to %.3f", a.Low, a.High)
}

func pct(x float64, n int) string {
	if n == 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.1f%%", 100*x)
}

func num(x float64) string {
	if math.IsNaN(x) {
		return "n/a"
	}
	return fmt.Sprintf("%.3f", x)
}

func within(a Agreement) string {
	if math.IsNaN(a.Quadratic) {
		return "n/a"
	}
	return pct(a.Within1, a.N)
}
