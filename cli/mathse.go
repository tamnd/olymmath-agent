package cli

import (
	"context"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/tamnd/olymmath-agent/content"
	"github.com/tamnd/olymmath-agent/fetch"
	"github.com/tamnd/olymmath-agent/mathse"
)

// Limits of the lane of Mathematics Stack Exchange.
const (
	// mathsePages is the most pages of questions that one search reads. The API without a key gives 300 requests a day.
	mathsePages = 10
	// mathseScore is the least score of an answer that is not the accepted answer.
	mathseScore = 3
	// mathseAnswers is the most answers that one question gives.
	mathseAnswers = 3
)

// mathseLinks finds the problem of each contest-math question whose title names a competition and a year, and gives the best answers of the question as solutions.
func mathseLinks(ctx context.Context, fc *fetch.Client, corpus *content.Corpus, stderr io.Writer) ([]solutionLink, error) {
	byEdition := map[string][]*content.Problem{}
	at := map[*content.Problem]map[string]string{}
	for _, p := range corpus.All() {
		for _, a := range p.Appearances {
			i := strings.LastIndex(a.At, "/")
			byEdition[a.At[:i]] = append(byEdition[a.At[:i]], p)
			if at[p] == nil {
				at[p] = map[string]string{}
			}
			at[p][a.At[:i]] = a.At
		}
	}
	page := func(u string) (mathse.Page, error) {
		text, err := get(ctx, fc, "mathse", u)
		if err != nil {
			return mathse.Page{}, err
		}
		pg, err := mathse.Parse([]byte(text))
		if err == nil && pg.Backoff > 0 {
			time.Sleep(time.Duration(pg.Backoff) * time.Second)
		}
		return pg, err
	}
	// The problem of each question.
	found := map[int]string{}
	var ids []int
	var failed error
	for _, word := range mathse.Searches {
		for n := 1; n <= mathsePages; n++ {
			pg, err := page(mathse.SearchURL(word, n))
			if err != nil {
				fmt.Fprintf(stderr, "om solutions: mathse: %s page %d: %v\n", word, n, err)
				failed = err
				break
			}
			for _, q := range pg.Items {
				if _, ok := found[q.Question]; ok {
					continue
				}
				key, ok := mathse.TitleKey(q.Title)
				if !ok {
					continue
				}
				if l := locate(byEdition[key.Edition], at, key, q.Title+"\n\n"+q.Body); l != "" {
					found[q.Question] = l
					ids = append(ids, q.Question)
				}
			}
			if !pg.HasMore {
				break
			}
		}
	}
	slices.Sort(ids)
	var out []solutionLink
	for i := 0; i < len(ids); i += 100 {
		batch := ids[i:min(i+100, len(ids))]
		var answers []mathse.Post
		for n := 1; ; n++ {
			pg, err := page(mathse.AnswersURL(batch, n))
			if err != nil {
				fmt.Fprintf(stderr, "om solutions: mathse: answers: %v\n", err)
				failed = err
				break
			}
			answers = append(answers, pg.Items...)
			if !pg.HasMore {
				break
			}
		}
		per := map[int]int{}
		for _, a := range answers {
			if (!a.Accepted && a.Score < mathseScore) || per[a.Question] >= mathseAnswers || !mathse.IsSolution(a.Body) {
				continue
			}
			per[a.Question]++
			l := solutionLink{at: found[a.Question]}
			l.sol = linkedSolution{Origin: "human", URL: fmt.Sprintf("https://math.stackexchange.com/a/%d", a.Answer), Lang: "en", Body: a.Body}
			l.sol.Author.Display = a.Owner.Name
			out = append(out, l)
		}
	}
	return out, failed
}

// locate gives the locator of the problem of a question, or "". When the title gives the position, the problem of that position must occur in the question. Otherwise the best problem of the edition must occur in the question, and the next one must be at least mathse.Margin below it.
func locate(ps []*content.Problem, at map[*content.Problem]map[string]string, key mathse.Key, question string) string {
	best, next, loc := 0.0, 0.0, ""
	for _, p := range ps {
		l := at[p][key.Edition]
		if key.Position != "" && !strings.HasSuffix(l, "/"+key.Position) {
			continue
		}
		c := mathse.Contains(p.Statement, question)
		switch {
		case c > best:
			best, next, loc = c, best, l
		case c > next:
			next = c
		}
	}
	if best < mathse.Least || (key.Position == "" && best-next < mathse.Margin) {
		return ""
	}
	return loc
}
