package loose

import (
	"fmt"
	"hash/fnv"
	"runtime"
	"slices"
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
	// Editions are the editions of the problem, as imo/2006, or for a row the edition that its source names.
	Editions []string
	// Named are the editions that the sources of a loose problem of the content name. The problem has no position in them, so the rule of the edition does not use them.
	Named []string
	// ID names the problem of a text of the content, so that two texts of one problem count once in an edition.
	ID string
	// Derived is true for a text of a loose problem of the content that a derived dataset gave (origin D). Such a problem came from a row of a dataset, so a link to it needs the same numbers as a join does.
	Derived bool
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
	// W, M and O are the Jaccard indexes of the words, of the formulas and of the order of the formulas of the rule of the words (words.go), Words the fewer words of the two texts, and Numbers is true when the two texts have the same numbers.
	W, M, O float64
	Words   int
	Numbers bool
}

// Decision is what the import does with a row.
type Decision struct {
	Kind Kind
	// Target is the index of the problem for Link, the index of the row that made the new problem for Join, and for Hold the problem (0 or more) or the row (-1 - its index) that the row is near.
	Target int
	Signals
	// Reason says why a row is held, or for a link by its edition the scores.
	Reason string
	// Edition is true when the row is the same as a problem of the edition that its source names, by the keyword match.
	Edition bool
}

// Options of Decide.
type Options struct {
	// Limit stops the decisions after this number of new problems. 0 is no limit.
	Limit int
	// Neighbors is the number of the nearest texts of each row that are compared with it.
	Neighbors int
	// Complete are the editions whose problems are all in the content. A row that names one of them and matches none of its problems is held, not made a new problem: it is a copy in other words, or its label is wrong.
	Complete map[string]bool
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
	return s.E >= Near && s.F >= KeyStrong && s.E+s.F+s.T >= KeySum && ((s.Tokens >= keymatch.MinTokens && s.Shared >= keymatch.MinShared) || s.E >= KeyShort) || s.byWords()
}

// byWords tells if the rule of the words says that the two texts are one problem: nearly the same words and formulas, the same numbers, and for a short text formulas in nearly the same order.
func (s Signals) byWords() bool {
	return s.W >= WordSame && s.M >= FormulaSame && s.Words >= MinWords && s.Numbers && (s.Words >= ShortWords || s.O >= OrderSame)
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

// sameEdition gives the problem of the edition of a row that the keyword match places the row on, the doubtful problem that it comes near, and the signals of the best pair. A row of an edition that the content has is one of its problems more often than not, so the bar is the bar of the content match (#168), where the competition and the year agree: a score of keymatch.Strong, the floors of shared tokens, and a gap of keymatch.Margin to the next problem of the edition. A pair at keymatch.Doubtful, or at Strong with no gap, is held.
func sameEdition(best, next float64, s Signals) (same, doubt bool) {
	if s.A == match.AnswerMismatch || s.Tokens < keymatch.MinTokens || s.Shared < keymatch.MinShared {
		return false, s.A == match.AnswerMismatch && best >= keymatch.Strong
	}
	if best >= keymatch.Strong && best-next >= keymatch.Margin {
		return true, false
	}
	return false, best >= keymatch.Doubtful
}

// Decide gives the decision for each row. The problems of the content come first in docs, then the rows, in order. Texts are compared in the form of matchText. A row is compared with the problems and with the rows before it that became a problem or joined one. The best same text wins: the same normalized text, then the highest T, then the highest E. A row with no same text and a doubtful one is held. A row that names an edition is then compared with each problem of that edition (sameEdition). Each other row is new, until the limit.
func Decide(problems, rows []Doc, o Options) []Decision {
	n := len(problems)
	all := append(append([]Doc{}, problems...), rows...)
	forms := make([]string, len(all))
	texts := make([]string, len(all))
	feats := make([]keymatch.Features, len(all))
	keys := make([]uint64, len(all))
	// stems are the texts with no answer choices, or "" for a text with none.
	stems := make([]string, len(all))
	stemFeats := make([]keymatch.Features, len(all))
	shapes := make([]shape, len(all))
	words := make([]string, len(all))
	parallel(len(all), func(i int) {
		forms[i] = matchText(all[i].Text)
		keys[i] = exactKey(forms[i])
		texts[i] = simText(forms[i])
		feats[i] = keymatch.Extract(forms[i])
		shapes[i] = shapeOf(forms[i])
		words[i] = shapes[i].text
		if st := stem(forms[i]); st != forms[i] {
			stems[i] = st
			stemFeats[i] = keymatch.Extract(st)
		}
	})
	ks := make([]keymatch.Key, len(all))
	for i := range all {
		ks[i] = keymatch.Key{F: feats[i]}
	}
	km := keymatch.NewMatcher(ks)
	byEdition := map[string][]int{}
	for i := range n {
		for _, e := range all[i].Editions {
			if !slices.Contains(byEdition[e], i) {
				byEdition[e] = append(byEdition[e], i)
			}
		}
	}
	// peers are the texts of the content in the editions that a row names.
	peers := func(i int) []int {
		var out []int
		for _, e := range all[i].Editions {
			for _, j := range byEdition[e] {
				if !slices.Contains(out, j) {
					out = append(out, j)
				}
			}
		}
		return out
	}
	near, peerE := similar.FindFromAlso(texts, n, max(o.Neighbors, 1), Near, peers)
	nearW := similar.FindFrom(words, n, max(o.Neighbors, 1), WordNear)
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
		s.W, s.M, s.O = bagJaccard(shapes[i].words, shapes[j].words), bagJaccard(shapes[i].formulas, shapes[j].formulas), bagJaccard(shapes[i].order, shapes[j].order)
		s.Words, s.Numbers = min(shapes[i].n, shapes[j].n), shapes[i].numbers == shapes[j].numbers
		if stems[i] != "" && stems[j] != "" && !s.Exact {
			// Two multiple choice problems of one contest often have the same choices, so the texts without them must agree too.
			t := match.Jaccard(match.Shingles(stems[i]), match.Shingles(stems[j]))
			f, _, _ := km.Score(stemFeats[i], stemFeats[j])
			if t < StemT && f < StemF {
				s.T, s.F = t, f
			}
			// And the choices of one problem are written in many ways, in a formula or not, so the same stem is the same text.
			// But a short stem like "Find the largest number" is shared by many problems, so the keywords of the choices must agree too.
			if t >= match.Same && s.F >= StemF {
				s.T = max(s.T, t)
			}
		}
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
		// A text with other formulas has a low E, so the rule of the words finds its candidates by the words alone. Its E is not known; 0 is below each bar of E.
		for _, m := range nearW[i] {
			if owner[m.Doc] != none && (m.Doc < n || m.Doc < i) && !slices.ContainsFunc(cands, func(c cand) bool { return c.doc == m.Doc }) {
				cands = append(cands, cand{m.Doc, 0})
			}
		}
		var best *Decision
		doubt := Decision{Kind: New}
		for _, c := range cands {
			s := signals(i, c.doc, c.e)
			if s.Same() && !s.Exact && s.T < match.Same && sibling(all[i], all[c.doc], c.doc >= n) {
				// Two rows of one edition with close texts are often two papers of the contest, one for each grade, with one change in the data. They wait for a person.
				if doubt.Kind == New {
					doubt = Decision{Kind: Hold, Target: owner[c.doc], Signals: s, Reason: fmt.Sprintf("same edition, T %.2f", s.T)}
				}
				continue
			}
			if s.Same() {
				d := Decision{Kind: Link, Target: owner[c.doc], Signals: s}
				if !(Signals{E: s.E, T: s.T, F: s.F, Shared: s.Shared, Tokens: s.Tokens, A: s.A, Exact: s.Exact}).Same() {
					d.Reason = fmt.Sprintf("words %.2f, formulas %.2f", s.W, s.M)
				}
				// A link to a loose problem that a derived dataset gave, or through a row that links, is a join of two rows in all but name, so it needs the same numbers too.
				if d.Target >= 0 && (c.doc >= n || all[c.doc].Derived) && shapes[i].numbers != shapes[c.doc].numbers {
					if doubt.Kind == New {
						doubt = Decision{Kind: Hold, Target: owner[c.doc], Signals: s, Reason: "other numbers"}
					}
					continue
				}
				if d.Target < 0 {
					d.Kind, d.Target = Join, -1-d.Target
					// Two rows of a derived dataset with the same words and other numbers are most often two versions of one problem, such as 2021 and 2022 or 33 and 123 points. The text signals and the exact key read past one number, so a join needs the same numbers as the row it joins and as the row that made the new problem.
					if shapes[i].numbers != shapes[c.doc].numbers || shapes[i].numbers != shapes[n+d.Target].numbers {
						if doubt.Kind == New {
							doubt = Decision{Kind: Hold, Target: owner[c.doc], Signals: s, Reason: "other numbers"}
						}
						continue
					}
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
		if best == nil {
			best, doubt = edition(i, all, peers(i), peerE[i], signals, doubt)
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
		case completeOf(all[i].Editions, o.Complete) != "" && len(peers(i)) > 0:
			// The row waits with the problem of the edition that its text is nearest to.
			ps, top := peers(i), 0
			for k := range ps {
				if peerE[i][k] > peerE[i][top] {
					top = k
				}
			}
			out[r] = Decision{Kind: Hold, Target: owner[ps[top]], Reason: fmt.Sprintf("the content has every problem of %s, nearest E %.2f", completeOf(all[i].Editions, o.Complete), peerE[i][top])}
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

// completeOf gives the first of the editions that is complete, or "".
func completeOf(editions []string, complete map[string]bool) string {
	for _, e := range editions {
		if complete[e] {
			return e
		}
	}
	return ""
}

// simText gives the text that E is taken on: the stem of a match form and the grams of its formulas. The choices are left out, because they are a formula in one text and plain text in another, and their grams would outweigh the problem.
func simText(form string) string {
	st := stem(form)
	return st + mathGrams(st)
}

// sibling tells if a row and a doc with no position name a common edition: b is a row when row is true, and else a loose problem of the content.
func sibling(a, b Doc, row bool) bool {
	eds := b.Named
	if row {
		eds = b.Editions
	}
	return slices.ContainsFunc(a.Editions, func(e string) bool { return slices.Contains(eds, e) })
}

// edition compares the row i with the problems of the editions that it names, and gives a link or a doubt (sameEdition). Two texts of one problem in the content count once, at their best score.
func edition(i int, all []Doc, peers []int, peerE []float64, signals func(i, j int, e float64) Signals, doubt Decision) (*Decision, Decision) {
	type scored struct {
		doc int
		s   Signals
	}
	// Each problem of the edition keeps its text with the best keyword match and its text with the best E.
	byF, byE := map[string]scored{}, map[string]scored{}
	for k, j := range peers {
		id := all[j].ID
		if id == "" {
			id = fmt.Sprint(j)
		}
		e := 0.0
		if k < len(peerE) {
			e = peerE[k]
		}
		s := signals(i, j, e)
		if old, ok := byF[id]; !ok || s.F > old.s.F || (s.F == old.s.F && j < old.doc) {
			byF[id] = scored{j, s}
		}
		if old, ok := byE[id]; !ok || s.E > old.s.E || (s.E == old.s.E && j < old.doc) {
			byE[id] = scored{j, s}
		}
	}
	rank := func(m map[string]scored, v func(Signals) float64) (top int, first, second float64, s Signals) {
		top = -1
		for _, c := range m {
			x := v(c.s)
			switch {
			case top < 0 || x > first || (x == first && c.doc < top):
				if top >= 0 {
					second = max(second, first)
				}
				top, first, s = c.doc, x, c.s
			case x > second:
				second = x
			}
		}
		return top, first, second, s
	}
	top, first, second, s := rank(byF, func(s Signals) float64 { return s.F })
	if top < 0 {
		return nil, doubt
	}
	topE, firstE, secondE, sE := rank(byE, func(s Signals) float64 { return s.E })
	if same, _ := sameEdition(first, second, s); same {
		return &Decision{Kind: Link, Target: top, Signals: s, Edition: true, Reason: fmt.Sprintf("edition keyword match %.2f, next %.2f", first, second)}, doubt
	}
	if sameEditionText(firstE, secondE, sE) {
		return &Decision{Kind: Link, Target: topE, Signals: sE, Edition: true, Reason: fmt.Sprintf("edition text match %.2f, next %.2f", firstE, secondE)}, doubt
	}
	// A row that is held already keeps the pair that holds it.
	if doubt.Kind != New {
		return nil, doubt
	}
	if _, unsure := sameEdition(first, second, s); unsure {
		return nil, Decision{Kind: Hold, Target: top, Signals: s, Reason: fmt.Sprintf("edition keyword match %.2f, next %.2f", first, second)}
	}
	if sE.A != match.AnswerMismatch && firstE >= EditionDoubtE {
		return nil, Decision{Kind: Hold, Target: topE, Signals: sE, Reason: fmt.Sprintf("edition text match %.2f, next %.2f", firstE, secondE)}
	}
	return nil, doubt
}

// EditionE, EditionMarginE and EditionDoubtE are the bars of the text match (E) of a row with the problems of an edition that it names: E of EditionE and a gap of EditionMarginE to the next problem make the same problem, E of EditionDoubtE a held pair. They catch the rows that rename the variables of a problem, which the keyword match misses.
const (
	EditionE       = 0.5
	EditionMarginE = 0.2
	EditionDoubtE  = 0.35
	// EditionSureE is the E that links with no other signal, and EditionT and EditionF are the T and the keyword match that a lower E needs.
	EditionSureE = 0.8
	EditionT     = 0.3
	EditionF     = 0.5
)

// StemT and StemF are the T and the keyword match of the texts of two multiple choice problems without their choices, below which the texts are scored without the choices.
const (
	StemT = 0.3
	StemF = 0.5
)

// sameEditionText tells if the best text match of a row with the problems of an edition places the row on that problem.
func sameEditionText(best, next float64, s Signals) bool {
	if s.A == match.AnswerMismatch || best < EditionE || best-next < EditionMarginE {
		return false
	}
	// The content may not have the problem of the row, and then the best problem of the edition is another one on the same topic. Below EditionSureE the words or the keywords must agree a little too.
	return best >= EditionSureE || s.T >= EditionT || s.F >= EditionF
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
