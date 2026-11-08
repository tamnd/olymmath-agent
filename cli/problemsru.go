package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/seed"
)

// problemsRu is the source id of problems.ru in the source register.
const problemsRu = "problems-ru"

// prMoscow is the node of the Moscow Mathematical Olympiad in the source catalogue of problems.ru.
const prMoscow = 164898

// seedProblemsRu walks the node of one olympiad of problems.ru, its years from from to to, and the paper of each grade, and writes one seed row for each problem whose paper gives its number. The label of the rows is the name of the olympiad node, and the position of problem N of grade G is pG followed by N, as in ru-moscow/2025/p81. A year with no paper nodes gives the positions pN. The second day of a grade has no position yet, so it is left out.
func seedProblemsRu(ctx context.Context, fc *fetch.Client, dir string, node, from, to int, log io.Writer) (kept, skipped int, err error) {
	root, err := get(ctx, fc, problemsRu, archive.PRNodeURL(node, 0))
	if err != nil {
		return 0, 0, err
	}
	label := archive.PRName(root)
	if label == "" {
		return 0, 0, fmt.Errorf("the node %d has no name", node)
	}
	err = seed.WriteFile(dir, problemsRu, func(w io.Writer) error {
		enc := json.NewEncoder(w)
		enc.SetEscapeHTML(false)
		for _, yn := range archive.PRNodes(root) {
			year, ok := archive.PRYear(yn.Name)
			if !ok || year < from || year > to {
				continue
			}
			page, err := get(ctx, fc, problemsRu, archive.PRNodeURL(yn.ID, 0))
			if err != nil {
				fmt.Fprintf(log, "%s: %s %d: %v\n", problemsRu, label, year, err)
				continue
			}
			papers := archive.PRNodes(page)
			if len(papers) == 0 {
				papers = []archive.PRNode{{ID: yn.ID, Name: "", Count: yn.Count}}
			}
			for _, pn := range papers {
				grade, day := 0, 0
				if pn.ID != yn.ID {
					if grade, day, ok = archive.PRPaper(pn.Name); !ok || day == 2 {
						fmt.Fprintf(log, "%s: %s %d: the paper %q has no position, so its %d problems are left out\n", problemsRu, label, year, pn.Name, pn.Count)
						skipped += pn.Count
						continue
					}
				}
				k, s, err := seedPRPaper(ctx, fc, enc, label, year, grade, pn, log)
				kept, skipped = kept+k, skipped+s
				if err != nil {
					return err
				}
			}
		}
		return nil
	})
	return kept, skipped, err
}

// seedPRPaper writes the seed rows of the listing of one paper node, page by page.
func seedPRPaper(ctx context.Context, fc *fetch.Client, enc *json.Encoder, label string, year, grade int, pn archive.PRNode, log io.Writer) (kept, skipped int, err error) {
	row := 0
	for start := 0; ; {
		page, err := get(ctx, fc, problemsRu, archive.PRNodeURL(pn.ID, start))
		if err != nil {
			fmt.Fprintf(log, "%s: %s %d %s: %v\n", problemsRu, label, year, pn.Name, err)
			return kept, skipped, nil
		}
		ps := archive.ParsePRListing(page)
		for _, p := range ps {
			switch {
			case p.Err != nil:
				fmt.Fprintf(log, "%s: %d: %v\n", problemsRu, p.ID, p.Err)
				skipped++
				continue
			case p.Number == 0:
				fmt.Fprintf(log, "%s: %d: the listing gives no number\n", problemsRu, p.ID)
				skipped++
				continue
			}
			number := "p" + strconv.Itoa(p.Number)
			if grade > 0 {
				number = fmt.Sprintf("p%d%d", grade, p.Number)
			}
			r := seed.Row{Source: problemsRu, File: "parent=" + strconv.Itoa(pn.ID), Row: row, ID: strconv.Itoa(p.ID), Labels: []string{label}, Year: year, Number: number, Lang: "ru", Statement: p.Text, Ref: archive.PRProblemURL(p.ID)}
			row++
			if err := enc.Encode(r); err != nil {
				return kept, skipped, err
			}
			kept++
		}
		start += len(ps)
		if len(ps) == 0 || start >= archive.PRTotal(page) {
			return kept, skipped, nil
		}
	}
}
