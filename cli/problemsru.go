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
	return prPrefix(g, d) + strconv.Itoa(n)
}

func prPrefix(g, d int) string {
	switch {
	case g == 0:
		return "p"
	case d == 2:
		return fmt.Sprintf("d%d", g)
	}
	return fmt.Sprintf("p%d", g)
}

// prPlace tells where the problems of one paper go.
type prPlace struct {
	// year is the year of the edition. edition is the edition part of the locator when the year alone does not name it, as in tot/2024-fall.
	year    int
	edition string
	// prefix is the start of each position: problem N is prefix followed by N.
	prefix string
	// grade and day are the grade and the day of a grade paper, or 0.
	grade, day int
}

// key is the edition part of the locator.
func (p prPlace) key() string {
	if p.edition != "" {
		return p.edition
	}
	return strconv.Itoa(p.year)
}

// prPaperPlace gives the place of the paper named name of a year node that runs from the year first to the year last. The empty name is a year node with no papers. A grade paper belongs to the last year, which is the year of the olympiad when the node is a school year, as "1992-1993" of the All-Russian olympiad. A paper of the Tournament of Towns gives the edition of its tour: the autumn tour of "46 турнир (2024/2025 год)" is 2024-fall and its spring tour is 2025-spring, and its tier and level give the prefix, as in ja for the junior A-level paper.
func prPaperPlace(first, last int, name string) (prPlace, bool) {
	if name == "" {
		return prPlace{year: last, prefix: "p"}, first == last
	}
	if g, d, ok := archive.PRPaper(name); ok {
		return prPlace{year: last, prefix: prPrefix(g, d), grade: g, day: d}, true
	}
	if t, ok := archive.PRTourPaper(name); ok {
		if t.Spring {
			return prPlace{year: last, edition: fmt.Sprintf("%d-spring", last), prefix: t.Prefix()}, true
		}
		return prPlace{year: first, edition: fmt.Sprintf("%d-fall", first), prefix: t.Prefix()}, true
	}
	return prPlace{}, false
}

// prYears reads the years of a year node: one year, as "2025 год", or a school year, as "46 турнир (2024/2025 год)".
func prYears(name string) (first, last int, ok bool) {
	if y, ok := archive.PRYear(name); ok {
		return y, y, true
	}
	return archive.PRSpan(name)
}

// seedProblemsRu walks the node of one olympiad of problems.ru, its year nodes whose last year is in from to to, and the paper of each grade, and writes one seed row for each problem whose paper gives its number. The label of the rows is the name of the olympiad node, and the position comes from prPaperPlace. A paper whose name gives no grade and no tour of the Tournament of Towns is left out, and so is a year node in which two problems would get the same position, because then the numbers start again in a way that the names do not tell. The final round of the All-Russian olympiad is a node whose children are the grade papers; its other rounds are left out. A page of the raw store that is younger than maxAge is used with no request. With details, a problem of a one-day grade paper whose listing gives no number takes its number from its own page, at one request a problem.
func seedProblemsRu(ctx context.Context, fc *fetch.Client, dir string, node, from, to int, maxAge time.Duration, details bool, log io.Writer) (kept, skipped int, err error) {
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
			first, last, ok := prYears(yn.Name)
			if !ok || last < from || last > to {
				continue
			}
			page, err := get(archive.PRNodeURL(yn.ID, 0))
			if err != nil {
				fmt.Fprintf(log, "%s: %s %s: %v\n", problemsRu, label, yn.Name, err)
				continue
			}
			papers := archive.PRNodes(page)
			if len(papers) == 0 {
				papers = []archive.PRNode{{ID: yn.ID, Name: "", Count: yn.Count}}
			}
			var rows []seed.Row
			for i := 0; i < len(papers); i++ {
				pn := papers[i]
				if archive.PRFinal(pn.Name) {
					fp, err := get(archive.PRNodeURL(pn.ID, 0))
					if err != nil {
						fmt.Fprintf(log, "%s: %s %s %s: %v\n", problemsRu, label, yn.Name, pn.Name, err)
						continue
					}
					papers = append(papers, archive.PRNodes(fp)...)
					continue
				}
				pl, ok := prPaperPlace(first, last, pn.Name)
				if !ok {
					fmt.Fprintf(log, "%s: %s %s: the paper %q gives no grade, so its %d problems are left out\n", problemsRu, label, yn.Name, pn.Name, pn.Count)
					skipped += pn.Count
					continue
				}
				var detail func(string) (string, error)
				if details && pl.day == 0 && pl.edition == "" {
					detail = get
				}
				rs, s := readPRPaper(get, detail, node, label, pl, pn, log)
				rows, skipped = append(rows, rs...), skipped+s
			}
			seen := map[string]bool{}
			for _, r := range rows {
				k := r.Edition + "/" + strconv.Itoa(r.Year) + "/" + r.Number
				if seen[k] {
					fmt.Fprintf(log, "%s: %s %s: two problems have the position %s, so the %d problems of the year are left out\n", problemsRu, label, yn.Name, r.Number, len(rows))
					skipped += len(rows)
					rows = nil
					break
				}
				seen[k] = true
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

// readPRPaper reads the listing of one paper node, page by page, and gives a seed row for each problem with a number. When detail is not nil, a problem whose listing gives no number takes it from its own page.
func readPRPaper(get, detail func(string) (string, error), node int, label string, pl prPlace, pn archive.PRNode, log io.Writer) (rows []seed.Row, skipped int) {
	for start := 0; ; {
		page, err := get(archive.PRNodeURL(pn.ID, start))
		if err != nil {
			fmt.Fprintf(log, "%s: %s %d %s: %v\n", problemsRu, label, pl.year, pn.Name, err)
			return rows, skipped
		}
		ps := archive.ParsePRListing(page)
		for _, p := range ps {
			if p.Err == nil && p.Number == 0 && detail != nil {
				p.Number = prDetailNumber(detail, node, pl.year, pl.grade, p.ID, log)
			}
			switch {
			case p.Err != nil:
				fmt.Fprintf(log, "%s: %d: %v\n", problemsRu, p.ID, p.Err)
				skipped++
				continue
			case p.Number == 0:
				fmt.Fprintf(log, "%s: %d: the listing gives no number\n", problemsRu, p.ID)
				skipped++
				continue
			case (p.Year != 0 && p.Year != pl.year) || (p.Grade != 0 && p.Grade != pl.grade):
				fmt.Fprintf(log, "%s: %d: the number of year %d and grade %d does not fit the paper %q of %s\n", problemsRu, p.ID, p.Year, p.Grade, pn.Name, pl.key())
				skipped++
				continue
			}
			rows = append(rows, seed.Row{Source: problemsRu, File: "parent=" + strconv.Itoa(pn.ID), Row: len(rows), ID: strconv.Itoa(p.ID), Labels: []string{label}, Year: pl.year, Edition: pl.edition, Number: pl.prefix + strconv.Itoa(p.Number), Lang: "ru", Statement: p.Text, Ref: archive.PRProblemURL(p.ID)})
		}
		start += len(ps)
		if len(ps) == 0 || start >= archive.PRTotal(page) {
			return rows, skipped
		}
	}
}

// prDetailNumber reads the number of problem id from the table of sources of its own page. The table must give the problem one number in the year and the grade of its paper and nothing more, such as a day or a variant. It gives 0 otherwise.
func prDetailNumber(get func(string) (string, error), node, year, grade, id int, log io.Writer) int {
	page, err := get(archive.PRProblemURL(id))
	if err != nil {
		fmt.Fprintf(log, "%s: %d: %v\n", problemsRu, id, err)
		return 0
	}
	n := 0
	for _, s := range archive.ParsePRSources(page, node) {
		if s.Year != year || s.Grade != grade || s.Number == 0 {
			continue
		}
		if s.Other || (n != 0 && n != s.Number) {
			fmt.Fprintf(log, "%s: %d: its page does not give one number in grade %d of %d\n", problemsRu, id, grade, year)
			return 0
		}
		n = s.Number
	}
	return n
}
