package canon

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"math"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/locator"
	"github.com/tamnd/olymmath-agent/match"
)

// Record is the plan of one problem: the choice of its primary appearance and the choices of the appearances that merge into it.
type Record struct {
	Primary Choice
	Merged  []Merge
}

// Appearances returns the locators of the record, the primary first.
func (r *Record) Appearances() []string {
	out := []string{r.Primary.At}
	for _, m := range r.Merged {
		out = append(out, m.At)
	}
	return out
}

// Merge is one appearance that joins a problem of another appearance.
type Merge struct {
	At      string
	Into    string
	Signals match.Signals
	// Rule names the known duplicate pattern that gave the key, or "" for a match by text.
	Rule   string
	Choice Choice
}

// Item is one entry of the review queue.
type Item struct {
	ID string `json:"id"`
	// Kind is doubt (a candidate that disagrees with the canonical text of its appearance), pair (two appearances that may be one problem) or dispute (two answers differ).
	Kind   string  `json:"kind"`
	At     string  `json:"at"`
	With   string  `json:"with,omitempty"`
	Source string  `json:"source,omitempty"`
	Ref    string  `json:"ref,omitempty"`
	T      float64 `json:"t"`
	A      string  `json:"a,omitempty"`
	K      string  `json:"k,omitempty"`
	Text   string  `json:"text"`
	Other  string  `json:"other"`
}

// ItemID gives the id of a review item from its kind and its records. The id does not change between imports, so a review stays valid.
func ItemID(parts ...string) string {
	h := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(h[:6])
}

var answerNames = map[match.Answer]string{match.NoAnswer: "", match.AnswerMatch: "match", match.AnswerMismatch: "mismatch"}
var keyNames = map[match.Key]string{match.NoKey: "", match.WeakKey: "weak", match.StrongKey: "strong"}

// Primary is the order of the competitions for the primary appearance of a merged problem. The contest comes before its shortlist, and the shortlist comes before the selection tests that used its problems. A competition that is not in the list goes after the listed ones.
var Primary = []string{"imo", "isl", "usamo", "usajmo", "amc12", "amc10", "amc8"}

// pattern tells if two appearances are a known duplicate pattern of spec 06.5: a shortlist problem that became a contest problem (isl and imo, or a code X-sl and X), an AMC 10 problem that is also an AMC 12 problem, or a USAJMO problem that is also a USAMO problem. Such a pair has a strong key.
func pattern(a, b string) string {
	la, ea := locator.Parse(a)
	lb, eb := locator.Parse(b)
	if ea != nil || eb != nil {
		return ""
	}
	pair := []string{la.Comp, lb.Comp}
	slices.Sort(pair)
	switch {
	case pair[0] == "imo" && pair[1] == "isl" && editionYear(la.Edition) == editionYear(lb.Edition):
		return "isl-imo"
	case pair[1] == pair[0]+"-sl" && editionYear(la.Edition) == editionYear(lb.Edition):
		return pair[0] + "-sl"
	case pair[0] == "amc10" && pair[1] == "amc12" && la.Edition == lb.Edition:
		return "amc10-amc12"
	case pair[0] == "usajmo" && pair[1] == "usamo" && la.Edition == lb.Edition:
		return "usamo-usajmo"
	}
	return ""
}

func editionYear(e string) int {
	y := 0
	for i := 0; i < len(e) && i < 4; i++ {
		if e[i] < '0' || e[i] > '9' {
			return 0
		}
		y = y*10 + int(e[i]-'0')
	}
	return y
}

func primaryOrder(at string) int {
	comp, _, _ := strings.Cut(at, "/")
	if i := slices.Index(Primary, comp); i >= 0 {
		return i
	}
	return len(Primary)
}

// Plan groups the choices into records. Two appearances become one record when the merge has two strict signals (rule D03), and when the signal T is at least match.Same, or at least match.Doubt for a known duplicate pattern. Each other pair at or above the review threshold, and each doubt and dispute of a choice, goes to the review queue.
func Plan(choices []Choice) ([]*Record, []Item) {
	var items []Item
	for _, ch := range choices {
		for _, d := range ch.Doubts {
			kind := "doubt"
			if d.A == match.AnswerMismatch {
				kind = "dispute"
			}
			items = append(items, Item{ID: ItemID(kind, ch.At, d.C.Source, d.C.Ref), Kind: kind, At: ch.At, Source: d.C.Source, Ref: d.C.Ref, T: round(d.T), A: answerNames[d.A], K: keyNames[d.C.Key], Text: ch.Best.Text, Other: d.C.Text})
		}
	}
	slices.SortFunc(choices, func(a, b Choice) int {
		return cmp.Or(cmp.Compare(primaryOrder(a.At), primaryOrder(b.At)), strings.Compare(a.At, b.At))
	})
	x := match.NewIndex()
	byAt := map[string]int{}
	for i, ch := range choices {
		byAt[ch.At] = i
		x.Add(ch.At, ch.Best.Shingles())
	}
	// into[i] is the index of the choice that choice i merges into, or -1.
	into := make([]int, len(choices))
	for i := range into {
		into[i] = -1
	}
	type pair struct {
		a, b int
		t    float64
	}
	var pairs []pair
	x.Pairs(match.Doubt, func(a, b string, t float64) {
		i, j := byAt[a], byAt[b]
		if i > j {
			i, j = j, i
		}
		pairs = append(pairs, pair{i, j, t})
	})
	slices.SortFunc(pairs, func(p, q pair) int {
		return cmp.Or(cmp.Compare(q.t, p.t), cmp.Compare(p.a, q.a), cmp.Compare(p.b, q.b))
	})
	signals := map[int]match.Signals{}
	rules := map[int]string{}
	// editions[i] holds the editions of the record of root i. One record has at most one appearance of each edition, because two problems of one paper are two problems.
	editions := map[int]map[string]bool{}
	edOf := func(at string) string {
		l, err := locator.Parse(at)
		if err != nil {
			return at
		}
		return l.Comp + "/" + l.Edition
	}
	has := func(root int, at string) bool {
		if editions[root] == nil {
			editions[root] = map[string]bool{edOf(choices[root].At): true}
		}
		return editions[root][edOf(at)]
	}
	for _, p := range pairs {
		a, b := choices[p.a], choices[p.b]
		s := match.Signals{T: p.t, A: match.Answers(a.Answer, b.Answer)}
		rule := pattern(a.At, b.At)
		if rule != "" {
			s.K = match.StrongKey
		}
		root := p.a
		for into[root] >= 0 {
			root = into[root]
		}
		switch {
		case into[p.b] >= 0 || root == p.b:
			continue
		case (p.t >= match.Same || (rule != "" && p.t >= match.Doubt)) && match.Strict(s) && s.A != match.AnswerMismatch && !has(root, b.At):
			into[p.b] = root
			editions[root][edOf(b.At)] = true
			signals[p.b] = s
			rules[p.b] = rule
		case p.t >= match.Same || rule != "":
			items = append(items, Item{ID: ItemID("pair", a.At, b.At), Kind: "pair", At: a.At, With: b.At, T: round(p.t), A: answerNames[s.A], K: keyNames[s.K], Text: a.Best.Text, Other: b.Best.Text})
		}
	}
	recs := map[int]*Record{}
	var out []*Record
	for i, ch := range choices {
		if into[i] < 0 {
			r := &Record{Primary: ch}
			recs[i] = r
			out = append(out, r)
		}
	}
	for i, ch := range choices {
		if j := into[i]; j >= 0 {
			for into[j] >= 0 {
				j = into[j]
			}
			r := recs[j]
			r.Merged = append(r.Merged, Merge{At: ch.At, Into: r.Primary.At, Signals: signals[i], Rule: rules[i], Choice: ch})
		}
	}
	slices.SortFunc(out, func(a, b *Record) int { return strings.Compare(a.Primary.At, b.Primary.At) })
	slices.SortFunc(items, func(a, b Item) int { return cmp.Or(strings.Compare(a.At, b.At), strings.Compare(a.ID, b.ID)) })
	return out, items
}

func round(f float64) float64 { return math.Round(f*1000) / 1000 }
