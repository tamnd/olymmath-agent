package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/tamnd/olymmath-agent/archive"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/seed"
)

// problemsRu is the source id of problems.ru in the source register.
const problemsRu = "problems-ru"

// prMoscow is the node of the Moscow Mathematical Olympiad in the source catalogue of problems.ru.
const prMoscow = 164898

// prPosition gives the position of problem n of the paper of grade g and day d. Problem N of grade G is pG followed by N, as in ru-moscow/2025/p81. The second day of a grade numbers its problems from 1 again, so its problem N is dG followed by N, as in ru-moscow/2025/d113. A year with no grade papers gives pN.
func prPosition(g, d, n int) string {
	switch {
	case g == 0:
		return "p" + strconv.Itoa(n)
	case d == 2:
		return fmt.Sprintf("d%d%d", g, n)
	}
	return fmt.Sprintf("p%d%d", g, n)
}

// seedProblemsRu walks the node of one olympiad of problems.ru, its years from from to to, and the paper of each grade, and writes one seed row for each problem whose paper gives its number. The label of the rows is the name of the olympiad node, and the position comes from prPosition. A paper whose name gives no grade is left out, and so is a year in which two problems would get the same position, because then the numbers start again in a way that the names do not tell. A page of the raw store that is younger than maxAge is used with no request.
func seedProblemsRu(ctx context.Context, fc *fetch.Client, dir string, node, from, to int, maxAge time.Duration, log io.Writer) (kept, skipped int, err error) {
	get := func(url string) (string, error) {
		res, err := fc.Get(ctx, problemsRu, url, fetch.Options{MaxAge: maxAge})
		if err != nil {
			return "", err
		}
		rc, err := fc.Store.Get(ctx, res.Capture.Hash)
		if err != nil {
			return "", err
		}
		defer func() { _ = rc.Close() }()
		b, err := io.ReadAll(rc)
		return string(b), err
	}
	root, err := get(archive.PRNodeURL(node, 0))
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
			page, err := get(archive.PRNodeURL(yn.ID, 0))
			if err != nil {
				fmt.Fprintf(log, "%s: %s %d: %v\n", problemsRu, label, year, err)
				continue
			}
			papers := archive.PRNodes(page)
			if len(papers) == 0 {
				papers = []archive.PRNode{{ID: yn.ID, Name: "", Count: yn.Count}}
			}
			var rows []seed.Row
			for _, pn := range papers {
				grade, day := 0, 0
				if pn.ID != yn.ID {
					if grade, day, ok = archive.PRPaper(pn.Name); !ok {
						fmt.Fprintf(log, "%s: %s %d: the paper %q gives no grade, so its %d problems are left out\n", problemsRu, label, year, pn.Name, pn.Count)
						skipped += pn.Count
						continue
					}
				}
				rs, s := readPRPaper(get, label, year, grade, day, pn, log)
				rows, skipped = append(rows, rs...), skipped+s
			}
			seen := map[string]bool{}
			for _, r := range rows {
				if seen[r.Number] {
					fmt.Fprintf(log, "%s: %s %d: two problems have the position %s, so the %d problems of the year are left out\n", problemsRu, label, year, r.Number, len(rows))
					skipped += len(rows)
					rows = nil
					break
				}
				seen[r.Number] = true
			}
			for _, r := range rows {
				if err := enc.Encode(r); err != nil {
					return err
				}
				kept++
			}
		}
		return nil
	})
	return kept, skipped, err
}

// readPRPaper reads the listing of one paper node, page by page, and gives a seed row for each problem with a number.
func readPRPaper(get func(string) (string, error), label string, year, grade, day int, pn archive.PRNode, log io.Writer) (rows []seed.Row, skipped int) {
	for start := 0; ; {
		page, err := get(archive.PRNodeURL(pn.ID, start))
		if err != nil {
			fmt.Fprintf(log, "%s: %s %d %s: %v\n", problemsRu, label, year, pn.Name, err)
			return rows, skipped
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
			rows = append(rows, seed.Row{Source: problemsRu, File: "parent=" + strconv.Itoa(pn.ID), Row: len(rows), ID: strconv.Itoa(p.ID), Labels: []string{label}, Year: year, Number: prPosition(grade, day, p.Number), Lang: "ru", Statement: p.Text, Ref: archive.PRProblemURL(p.ID)})
		}
		start += len(ps)
		if len(ps) == 0 || start >= archive.PRTotal(page) {
			return rows, skipped
		}
	}
}
