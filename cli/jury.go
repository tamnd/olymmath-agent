package cli

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/tamnd/olymmath-agent/grade"
	"github.com/tamnd/olymmath-agent/model"
)

// newCLIClient gives the client of the cli: jurors, which runs the local command-line clients through the lanes. stateDir keeps the rests of the lanes. Tests replace it with a fake.
var newCLIClient = func(stateDir string) model.Client { return model.NewCLI(stateDir) }

// taskJurors gives the jurors of the tasks, each once, in order.
func taskJurors(tasks []grade.Task) []string {
	var out []string
	for _, t := range tasks {
		for _, j := range t.Jurors {
			if !slices.Contains(out, j) {
				out = append(out, j)
			}
		}
	}
	return out
}

// checkJurors gives an error for a cli: juror that names no known client.
func checkJurors(jurors []string) error {
	for _, j := range jurors {
		if model.IsCLI(j) {
			if _, _, err := model.ParseCLI(j); err != nil {
				return err
			}
		}
	}
	return nil
}

// juryNote gives a note when a task has a jury with no grader of another model family than its generator (spec 07.4), or "".
func juryNote(tasks []grade.Task) string {
	n := 0
	for _, t := range tasks {
		if !grade.OtherFamily(t.Jurors, t.Generator) {
			n++
		}
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("Note: %d of the juries have no grader of another model family than the generator. Spec 07.4 asks for one. Add a juror such as cli:gemini or cli:codex to -jurors.", n)
}

// allCLI reports whether each model is a cli: model. The calls of such a run go one at a time, so its cache keeps each response as it comes (model.Cache.Each).
func allCLI(models []string) bool {
	for _, m := range models {
		if !model.IsCLI(m) {
			return false
		}
	}
	return len(models) > 0
}

// juryClient gives the client of a jury: the Messages API for the Claude jurors, and the local command-line clients for the cli: jurors. It needs an API key only when a juror is not a cli: model. work is the directory of the lane state.
func juryClient(jurors []string, batch bool, work string) (model.Client, error) {
	api, cli := false, false
	for _, j := range jurors {
		if model.IsCLI(j) {
			cli = true
		} else {
			api = true
		}
	}
	var c model.Client
	if api {
		var err error
		if c, err = newClient(batch); err != nil {
			return nil, err
		}
	}
	if !cli {
		return c, nil
	}
	return &model.Split{API: c, CLI: newCLIClient(filepath.Join(work, "lanes"))}, nil
}
