package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/calibrate"
	"github.com/tamnd/olymmath-agent/grade"
	"github.com/tamnd/olymmath-agent/model"
)

func init() {
	register(command{"calibrate", "measure the grading jury against the human grades of IMO-GradingBench, the Open Proof Corpus and ProofBench (om calibrate fetch, then om calibrate)", runCalibrate})
}

// maxCalibrationFile is the largest data file that om calibrate fetch downloads.
const maxCalibrationFile = 32 << 20

// calibrationGrade is one line of a grades file: the jury score of an item, by the id set/item.
type calibrationGrade struct {
	ID    string `json:"id"`
	Score int    `json:"score"`
	Queue bool   `json:"queue,omitempty"`
}

func calibrationSets(names string) ([]*calibrate.Set, error) {
	var out []*calibrate.Set
	for n := range strings.SplitSeq(names, ",") {
		s := calibrate.SetByName(strings.TrimSpace(n))
		if s == nil {
			return nil, fmt.Errorf("no calibration set %q; the sets are gradingbench, proofbench and opc", n)
		}
		out = append(out, s)
	}
	return out, nil
}

func runCalibrate(env Env, args []string) int {
	if len(args) > 0 && args[0] == "fetch" {
		return runCalibrateFetch(env, args[1:])
	}
	fs := newFlags("calibrate", env.Stderr)
	sets := fs.String("set", "gradingbench,proofbench,opc", "the calibration sets, by commas")
	work := fs.String("work", defaultWork(), "the work directory; the data is in its calibration directory")
	limit := fs.Int("limit", 0, "the most items of each set, spread over the set, or 0 for all")
	grades := fs.String("grades", "", "a JSON Lines file of jury scores {\"id\": \"set/item\", \"score\": N}; om compares them and calls no model")
	dry := fs.Bool("dry-run", false, "print the plan and the cost estimate, and call no model")
	budget := fs.Float64("budget", 0, "the most the run may cost, in US dollars")
	jurors := fs.String("jurors", strings.Join(grade.DefaultJurors, ","), "the models of the jury, by commas")
	batch := fs.Bool("batch", true, "use the Batch API, at half the price")
	out := fs.String("out", "", "the report file, for example reports/calibration.md in the content repository; empty for the standard output")
	if _, err := parseMixed(fs, args); err != nil {
		return ExitUsage
	}
	ss, err := calibrationSets(*sets)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
		return ExitUsage
	}
	dir := filepath.Join(*work, "calibration")
	items := map[string][]calibrate.Item{}
	var tasks []grade.Task
	js := strings.Split(*jurors, ",")
	for _, s := range ss {
		its, err := readCalibrationSet(s, dir)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
			return ExitUsage
		}
		its = spread(its, *limit)
		items[s.Name] = its
		for _, it := range its {
			tasks = append(tasks, it.Task(js))
		}
	}
	scores := map[string]int{}
	source := ""
	switch {
	case *grades != "":
		if scores, err = readCalibrationGrades(*grades); err != nil {
			fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
			return ExitUsage
		}
		source = "the file " + filepath.Base(*grades)
	default:
		cost, calls, err := grade.Estimate(tasks, grade.Options{}, *batch)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
			return ExitUsage
		}
		fmt.Fprintf(env.Stdout, "%d proofs to grade, about %d calls and $%.2f.\n", len(tasks), calls, cost)
		if *dry {
			return ExitOK
		}
		if *budget <= 0 {
			fmt.Fprintln(env.Stderr, "om calibrate: set -budget in US dollars, -grades FILE, or -dry-run")
			return ExitUsage
		}
		client, err := newClient(*batch)
		if err != nil {
			fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
			return ExitUsage
		}
		b := &model.Budget{Client: client, Limit: *budget, Batch: *batch}
		results, err := grade.Run(context.Background(), &model.Cache{Client: b, Dir: filepath.Join(dir, "calls")}, tasks, grade.Options{})
		if err != nil {
			fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
			return ExitFailure
		}
		path := filepath.Join(dir, "grades-"+time.Now().UTC().Format("20060102-150405")+".jsonl")
		if err := writeCalibrationGrades(path, results, scores); err != nil {
			fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
			return ExitFailure
		}
		fmt.Fprintf(env.Stdout, "%d proofs graded for $%.2f; the scores are in %s.\n", len(scores), b.Spent(), path)
		source = "a jury run of " + time.Now().UTC().Format(time.DateOnly)
	}
	var rows []calibrate.Agreement
	jury := map[string]int{}
	for _, s := range ss {
		for _, it := range items[s.Name] {
			if v, ok := scores[s.Name+"/"+it.ID]; ok {
				jury[it.ID] = v
			}
		}
		rows = append(rows, calibrate.Compare(s, items[s.Name], jury))
		clear(jury)
	}
	w := env.Stdout
	if *out != "" {
		f, err := os.Create(*out) // #nosec G304 -- the report file that the user names.
		if err != nil {
			fmt.Fprintf(env.Stderr, "om calibrate: %v\n", err)
			return ExitFailure
		}
		defer func() { _ = f.Close() }()
		w = f
	}
	calibrate.Report(w, rows, grade.Jurors(js, ""), source)
	for _, a := range rows {
		if a.N == 0 || !(a.Kappa >= calibrate.Target) {
			return ExitFindings
		}
	}
	return ExitOK
}

// spread keeps n items spread evenly over the list, so a limit does not keep only the first problems.
func spread(items []calibrate.Item, n int) []calibrate.Item {
	if n <= 0 || n >= len(items) {
		return items
	}
	out := make([]calibrate.Item, 0, n)
	for i := range n {
		out = append(out, items[i*len(items)/n])
	}
	return out
}

func readCalibrationSet(s *calibrate.Set, dir string) ([]calibrate.Item, error) {
	var out []calibrate.Item
	for _, u := range s.Files {
		p := filepath.Join(dir, s.Name, path.Base(u))
		if _, err := os.Stat(p); err != nil {
			return nil, fmt.Errorf("the set %s has no file %s; run om calibrate fetch -set %s", s.Name, p, s.Name)
		}
		its, err := s.Read(p)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", p, err)
		}
		out = append(out, its...)
	}
	return out, nil
}

func readCalibrationGrades(path string) (map[string]int, error) {
	f, err := os.Open(path) // #nosec G304 -- the grades file that the user names.
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := map[string]int{}
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var g calibrationGrade
		if err := json.Unmarshal([]byte(line), &g); err != nil || g.ID == "" || g.Score < 0 || g.Score > 7 {
			return nil, fmt.Errorf("%s:%d: not a grade {\"id\": \"set/item\", \"score\": 0 to 7}", path, n)
		}
		out[g.ID] = g.Score
	}
	return out, sc.Err()
}

func writeCalibrationGrades(path string, results []*grade.Result, scores map[string]int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	var b strings.Builder
	for _, r := range results {
		if r.Err != nil {
			continue
		}
		scores[r.Task.ID] = r.Score
		line, _ := json.Marshal(calibrationGrade{ID: r.Task.ID, Score: r.Score, Queue: r.Queue})
		b.Write(line)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func runCalibrateFetch(env Env, args []string) int {
	fs := newFlags("calibrate fetch", env.Stderr)
	sets := fs.String("set", "gradingbench,proofbench,opc", "the calibration sets, by commas")
	work := fs.String("work", defaultWork(), "the work directory; the data goes to its calibration directory, out of git")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	ss, err := calibrationSets(*sets)
	if err != nil {
		fmt.Fprintf(env.Stderr, "om calibrate fetch: %v\n", err)
		return ExitUsage
	}
	client := &http.Client{Timeout: 5 * time.Minute}
	code := ExitOK
	for _, s := range ss {
		for _, u := range s.Files {
			p := filepath.Join(*work, "calibration", s.Name, path.Base(u))
			if _, err := os.Stat(p); err == nil {
				fmt.Fprintf(env.Stdout, "%s: %s is there\n", s.Name, p)
				continue
			}
			n, err := download(client, u, p)
			if err != nil {
				fmt.Fprintf(env.Stderr, "om calibrate fetch: %s: %v\n", u, err)
				code = ExitFailure
				continue
			}
			fmt.Fprintf(env.Stdout, "%s: %s, %d bytes, licence %s\n", s.Name, p, n, s.Licence)
		}
	}
	return code
}

// download writes a file of at most maxCalibrationFile bytes.
func download(client *http.Client, u, p string) (int64, error) {
	resp, err := client.Get(u) // #nosec G107 -- the URLs are the fixed files of the calibration sets.
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("the server answered %s", resp.Status)
	}
	if resp.ContentLength > maxCalibrationFile {
		return 0, fmt.Errorf("the file has %d bytes, more than %d", resp.ContentLength, maxCalibrationFile)
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o750); err != nil {
		return 0, err
	}
	tmp := p + ".part"
	f, err := os.Create(tmp) // #nosec G304 -- a file under the work directory.
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, io.LimitReader(resp.Body, maxCalibrationFile+1))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > maxCalibrationFile {
		err = errors.New("the file is too large")
	}
	if err != nil {
		_ = os.Remove(tmp)
		return 0, err
	}
	return n, os.Rename(tmp, p)
}
