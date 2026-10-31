// Package mathse reads the contest-math questions of Mathematics Stack Exchange through the Stack Exchange API, finds the problem of each question and gives the answers as human solutions (spec 05.4). The posts are under CC BY-SA, so the answers are in zone Z1.
package mathse

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/tamnd/olymmath-agent/match"
)

// API is the root of the Stack Exchange API.
const API = "https://api.stackexchange.com/2.3"

// Filter is the API filter that adds the Markdown of the posts and the fields of has_more and quota_remaining. The API made it from the default filter with include=question.body_markdown;answer.body_markdown;answer.link;question.link;.has_more;.quota_remaining;.backoff.
const Filter = "!)RL-JogHwoZuazwxM1GA4rLr"

// Searches are the words of the titles that olymmath looks for, one search for each word.
var Searches = []string{"IMO", "Putnam", "USAMO", "USAJMO", "shortlist"}

// SearchURL gives one page of the questions of the tag contest-math with a word in the title, from page 1, with the most votes first.
func SearchURL(word string, page int) string {
	q := url.Values{"site": {"math"}, "tagged": {"contest-math"}, "title": {word}, "sort": {"votes"}, "order": {"desc"}, "pagesize": {"100"}, "page": {strconv.Itoa(page)}, "filter": {Filter}}
	return API + "/search/advanced?" + q.Encode()
}

// AnswersURL gives one page of the answers of up to 100 questions, with the most votes first.
func AnswersURL(ids []int, page int) string {
	s := make([]string, len(ids))
	for i, id := range ids {
		s[i] = strconv.Itoa(id)
	}
	q := url.Values{"site": {"math"}, "sort": {"votes"}, "order": {"desc"}, "pagesize": {"100"}, "page": {strconv.Itoa(page)}, "filter": {Filter}}
	return API + "/questions/" + strings.Join(s, ";") + "/answers?" + q.Encode()
}

// Owner is the user who wrote a post.
type Owner struct {
	Name string `json:"display_name"`
	Link string `json:"link"`
}

// Post is a question or an answer.
type Post struct {
	Question int    `json:"question_id"`
	Answer   int    `json:"answer_id"`
	Title    string `json:"title"`
	Body     string `json:"body_markdown"`
	Link     string `json:"link"`
	Score    int    `json:"score"`
	Accepted bool   `json:"is_accepted"`
	Owner    Owner  `json:"owner"`
}

// Page is one page of a response of the API.
type Page struct {
	Items   []Post `json:"items"`
	HasMore bool   `json:"has_more"`
	Quota   int    `json:"quota_remaining"`
	Backoff int    `json:"backoff"`
	Error   string `json:"error_message"`
}

// Parse reads a page of the API. The API escapes the HTML of the Markdown and the titles, so Parse unescapes them.
func Parse(b []byte) (Page, error) {
	var p Page
	if err := json.Unmarshal(b, &p); err != nil {
		return p, err
	}
	if p.Error != "" {
		return p, fmt.Errorf("the API says: %s", p.Error)
	}
	for i := range p.Items {
		p.Items[i].Title = html.UnescapeString(p.Items[i].Title)
		p.Items[i].Body = strings.TrimSpace(html.UnescapeString(strings.ReplaceAll(p.Items[i].Body, "\r\n", "\n")))
		p.Items[i].Owner.Name = html.UnescapeString(p.Items[i].Owner.Name)
	}
	return p, nil
}

// Key is the edition and, when the title gives it, the position of a question, for example imo/2009 and p1.
type Key struct {
	Edition  string
	Position string
}

var (
	yearRE = `((?:19[5-9]|20[0-4])[0-9])`
	keyREs = []struct {
		comp string
		re   *regexp.Regexp
	}{
		{"isl", regexp.MustCompile(`(?i)\b(?:ISL|IMO\s+short\s*list|short\s*list(?:ed)?(?:\s+IMO)?)\s*'?` + yearRE + `\b(?:[\s,/#(:-]*(?:problem\s*)?([ACGN])\s*([0-9]{1,2})\b)?`)},
		{"isl", regexp.MustCompile(`(?i)\bIMO\s*'?` + yearRE + `\s+short\s*list(?:ed)?\b(?:[\s,/#(:-]*(?:problem\s*)?([ACGN])\s*([0-9]{1,2})\b)?`)},
		{"isl", regexp.MustCompile(`(?i)\b` + yearRE + `\s+(?:ISL|IMO\s+short\s*list)\b(?:[\s,/#(:-]*(?:problem\s*)?([ACGN])\s*([0-9]{1,2})\b)?`)},
		{"usajmo", regexp.MustCompile(`(?i)\bUSAJMO\s*'?` + yearRE + `\b(?:[\s,/#(:-]*(?:problem|P|Q)?\s*#?([1-6])\b)?`)},
		{"usamo", regexp.MustCompile(`(?i)\bUSAMO\s*'?` + yearRE + `\b(?:[\s,/#(:-]*(?:problem|P|Q)?\s*#?([1-6])\b)?`)},
		{"putnam", regexp.MustCompile(`(?i)\bPutnam\s*(?:exam\s*|competition\s*)?'?` + yearRE + `\b(?:[\s,/#(:-]*(?:problem\s*)?([AB])\s*-?\s*([1-6])\b)?`)},
		{"imo", regexp.MustCompile(`(?i)\bIMO\s*'?` + yearRE + `\b(?:[\s,/#(:-]*(?:problem|P|Q|question)?\s*#?([1-6])\b)?`)},
		{"imo", regexp.MustCompile(`(?i)\b` + yearRE + `\s+IMO\b(?:[\s,/#(:-]*(?:problem|P|Q|question)?\s*#?([1-6])\b)?`)},
	}
)

// otherRE matches a title about another problem than the one that it names, for example a generalization. The answers to such a question solve the other problem.
var otherRE = regexp.MustCompile(`(?i)generali[sz]|extension|extend|variant|variation|similar|inspired|analog|harder|stronger|weaker|converse|related|modified|version|longlist`)

// TitleKey reads the competition, the year and the position from the title of a question. It gives false when the title names no known competition with a year, or when the title is about another problem, for example a generalization.
func TitleKey(title string) (Key, bool) {
	if otherRE.MatchString(title) {
		return Key{}, false
	}
	for _, k := range keyREs {
		m := k.re.FindStringSubmatch(title)
		if m == nil {
			continue
		}
		key := Key{Edition: k.comp + "/" + m[1]}
		switch {
		case len(m) == 4 && m[2] != "":
			key.Position = strings.ToLower(m[2]) + m[3]
		case len(m) == 3 && m[2] != "":
			key.Position = "p" + m[2]
		}
		return key, true
	}
	return Key{}, false
}

// Words is the least number of words of an answer that olymmath takes as a solution.
const Words = 60

// notSolutionRE finds an answer that is not a solution: a review of the attempt of the user, a hint, a comment, or a copy of a text from another site.
var notSolutionRE = regexp.MustCompile(`(?i)\b(you|your|yours|hints?|comments?|verbatim|reproduced)\b`)

// IsSolution reports if the Markdown of an answer can be a solution of the problem. It is a solution only when it has at least Words words, it does not speak to the user, and it is not a hint, a comment or a copy.
func IsSolution(body string) bool {
	return len(strings.Fields(body)) >= Words && !notSolutionRE.MatchString(body)
}

// Contains gives the share of the shingles of a problem that occur in a question. A question often has the attempt of the user after the problem, so the Jaccard similarity is too low for it.
func Contains(problem, question string) float64 {
	a, b := match.Shingles(problem), match.Shingles(question)
	if len(a) == 0 {
		return 0
	}
	in := map[uint64]bool{}
	for _, h := range b {
		in[h] = true
	}
	n := 0
	for _, h := range a {
		if in[h] {
			n++
		}
	}
	return float64(n) / float64(len(a))
}

// Thresholds of the match of a question to a problem.
const (
	// Least is the least share of the problem in the question.
	Least = 0.5
	// Margin is the least distance between the best problem and the next one, when the title gives no position.
	Margin = 0.2
)
