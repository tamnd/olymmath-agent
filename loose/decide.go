package loose

import (
	"fmt"
	"hash/fnv"
	"runtime"
	"strings"
	"sync"

	"github.com/tamnd/olymmath-agent/keymatch"
	"github.com/tamnd/olymmath-agent/match"
	"github.com/tamnd/olymmath-agent/normalize"
	"github.com/tamnd/olymmath-agent/similar"
)

// Doc is a text to match: the canonical text of a problem of the content, or the text of a row.
type Doc struct {
	Text   string
	Answer string
}

// Kind is the decision for a row.
type Kind int

// The decisions. New makes a new problem. Link adds the row to a problem of the content. Join adds the row to the new problem of an earlier row. Hold keeps the row out: it is near a problem but not surely the same, so a person decides.
const (
	New Kind = iota
	Link
	Join
	Hold
	// Skip is a row after the limit of new problems: the import does not read it.
	Skip
)

func (k Kind) String() string { return [...]string{"new", "link", "join", "hold", "skip"}[k] }

// Signals are the signals of a row and its best match.
type Signals struct {
	// E is the TF-IDF cosine, T the Jaccard index of the word 5-grams and F the score of the keyword match. Shared and Tokens are the weight and the number of the shared tokens of the keyword match.
	E, T, F float64
	Shared  float64
	Tokens  int
	A       match.Answer
	// Exact is true when the two texts have the same normalized tokens.
	Exact bool
}

// Decision is what the import does with a row.
type Decision struct {
	Kind Kind
	// Target is the index of the problem for Link, the index of the row that made the new problem for Join, and for Hold the problem (0 or more) or the row (-1 - its index) that the row is near.
	Target int
	Signals
	// Reason says why a row is held.
	Reason string
}

// Options of Decide.
type Options struct {
	// Limit stops the decisions after this number of new problems. 0 is no limit.
	Limit int
	// Neighbors is the number of the nearest texts of each row that are compared with it.
	Neighbors int
}

// Thresholds of the decisions (spec 06.5 and 06.8).
const (
	// Near is the lowest signal E of a pair that is compared at all.
	Near = 0.5
	// HoldE is the signal E at which a pair with no second signal waits for a review.
	HoldE = 0.7
	// KeyStrong is the lowest keyword match score that makes a pair the same with signal E. The content match places a pair at Strong only when its competition and year agree too; a loose row has no position to agree, so it needs more. On the AoPS rows, pairs from Strong to 0.7 were often two inequalities or two equations of the same shape.
	KeyStrong = 0.7
	// KeySum is the lowest sum of E, F and T of such a pair. Two problems with the same formulas and a different question, such as two inequalities on a, b and c, often have E and T low and F just above KeyStrong.
	KeySum = 1.6
	// KeyShort is the lowest signal E of such a pair that has few shared tokens.
	KeyShort = 0.95
)

// Same tells if the signals of a pair say that the two texts are one problem: the same normalized text, or signal T at match.Same or above, or signal E at Near or above with a keyword match at KeyStrong or above, and in each case answers that do not disagree.
func (s Signals) Same() bool {
	if s.A == match.AnswerMismatch {
		return false
	}
	if s.Exact || s.T >= match.Same {
		return true
	}
	// A pair with other words needs a strong keyword match and enough of all three signals. A pair of two short texts has few tokens to share, so it needs nearly the same words.
	return s.E >= Near && s.F >= KeyStrong && s.E+s.F+s.T >= KeySum && ((s.Tokens >= keymatch.MinTokens && s.Shared >= keymatch.MinShared) || s.E >= KeyShort)
}

// Doubt gives the reason why a pair that is not the same needs a review, or "" when the two texts are two problems.
func (s Signals) Doubt() string {
	switch {
	case s.Same():
		return ""
	case s.A == match.AnswerMismatch && (s.Exact || s.T >= match.Same || s.E >= HoldE):
		return "answer mismatch"
	case s.Exact || s.T >= match.Same:
		return ""
	case s.E >= HoldE:
		return fmt.Sprintf("E %.2f", s.E)
	case s.T >= match.Doubt:
		return fmt.Sprintf("T %.2f", s.T)
	case s.E >= Near && s.F >= keymatch.Doubtful && s.Tokens >= keymatch.MinTokens:
		return fmt.Sprintf("E %.2f and keyword match %.2f", s.E, s.F)
	}
	return ""
}

// Decide gives the decision for each row. The problems of the content come first in docs, then the rows, in order. Texts are compared in the form of matchText. A row is compared with the problems and with the rows before it that became a problem or joined one. The best same text wins: the same normalized text, then the highest T, then the highest E. A row with no same text and a doubtful one is held. Each other row is new, until the limit.
func Decide(problems, rows []Doc, o Options) []Decision {
	n := len(problems)
	all := append(append([]Doc{}, problems...), rows...)
	forms := make([]string, len(all))
	texts := make([]string, len(all))
	feats := make([]keymatch.Features, len(all))
	keys := make([]uint64, len(all))
	parallel(len(all), func(i int) {
		forms[i] = matchText(all[i].Text)
		texts[i] = forms[i] + mathGrams(forms[i])
		feats[i] = keymatch.Extract(forms[i])
		keys[i] = exactKey(forms[i])
	})
	ks := make([]keymatch.Key, len(all))
	for i := range all {
		ks[i] = keymatch.Key{F: feats[i]}
	}
	km := keymatch.NewMatcher(ks)
	near := similar.FindFrom(texts, n, max(o.Neighbors, 1), Near)
	shingles := make([][]uint64, len(all))
	sh := func(i int) []uint64 {
		if shingles[i] == nil {
			shingles[i] = match.Shingles(forms[i])
		}
		return shingles[i]
	}
	// owner gives the problem (0 or more) or the new problem of a row (-1 - the index of its first row) of each doc that a later row can join.
	owner := make([]int, len(all))
	for i := range owner {
		owner[i] = i
		if i >= n {
			owner[i] = none
		}
	}
	exact := map[uint64]int{}
	for i := range n {
		if _, ok := exact[keys[i]]; !ok {
			exact[keys[i]] = i
		}
	}
	signals := func(i, j int, e float64) Signals {
		s := Signals{E: e, T: match.Jaccard(sh(i), sh(j)), A: match.Answers(all[i].Answer, all[j].Answer), Exact: keys[i] == keys[j]}
		s.F, s.Shared, s.Tokens = km.Score(feats[i], feats[j])
		return s
	}
	out := make([]Decision, len(rows))
	created := 0
	for r := range rows {
		i := n + r
		if o.Limit > 0 && created >= o.Limit {
			for k := r; k < len(rows); k++ {
				out[k] = Decision{Kind: Skip}
			}
			break
		}
		type cand struct {
			doc int
			e   float64
		}
		var cands []cand
		if j, ok := exact[keys[i]]; ok {
			cands = append(cands, cand{j, 1})
		}
		for _, m := range near[i] {
			if owner[m.Doc] != none && (m.Doc < n || m.Doc < i) {
				cands = append(cands, cand{m.Doc, m.Score})
			}
		}
		var best *Decision
		doubt := Decision{Kind: New}
		for _, c := range cands {
			s := signals(i, c.doc, c.e)
			if s.Same() {
				d := Decision{Kind: Link, Target: owner[c.doc], Signals: s}
				if d.Target < 0 {
					d.Kind, d.Target = Join, -1-d.Target
				}
				if best == nil || better(s, best.Signals) {
					best = &d
				}
				continue
			}
			if why := s.Doubt(); why != "" && doubt.Kind == New {
				doubt = Decision{Kind: Hold, Target: owner[c.doc], Signals: s, Reason: why}
			}
		}
		switch {
		case best != nil:
			out[r] = *best
			if best.Kind == Link {
				owner[i] = best.Target
			} else {
				owner[i] = -1 - best.Target
			}
		case doubt.Kind == Hold:
			out[r] = doubt
		default:
			out[r] = Decision{Kind: New, Target: r}
			owner[i] = -1 - r
			created++
		}
		if out[r].Kind != Hold {
			if _, ok := exact[keys[i]]; !ok {
				exact[keys[i]] = i
			}
		}
	}
	return out
}

// none marks a doc that no row can join: a held row, or a row that has no decision yet.
const none = int(^uint(0) >> 1)

// better tells if the signals a of a same pair beat the signals b.
func better(a, b Signals) bool {
	if a.Exact != b.Exact {
		return a.Exact
	}
	if a.T != b.T {
		return a.T > b.T
	}
	return a.E > b.E
}

// exactKey hashes the normalized tokens of a text.
func exactKey(text string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.Join(normalize.Tokens(normalize.Text(text)), " ")))
	return h.Sum64()
}

// mathGrams gives the runs of four characters of the TeX of a text, with no spaces, as words, in the same way as om dups (signal E of spec 06.5).
func mathGrams(text string) string {
	var b strings.Builder
	for _, m := range mathRE.FindAllString(text, -1) {
		t := strings.Join(strings.Fields(strings.Trim(m, "$")), "")
		for i := 0; i+4 <= len(t); i++ {
			fmt.Fprintf(&b, " m%x", t[i:i+4])
		}
	}
	return b.String()
}

// parallel calls fn for each index from 0 to n-1 on all the CPUs.
func parallel(n int, fn func(int)) {
	var wg sync.WaitGroup
	w := max(runtime.NumCPU(), 1)
	for k := range w {
		wg.Go(func() {
			for i := k; i < n; i += w {
				fn(i)
			}
		})
	}
	wg.Wait()
}
