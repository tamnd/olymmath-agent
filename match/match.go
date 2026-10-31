// Package match decides whether two candidates are the same problem (spec 06.5). It uses the signal K (both candidates locate to the same appearance), the signal T (the Jaccard similarity of word 5-grams) and the signal A (both answers are equal). The signal E (the cosine of the TF-IDF weights of the words and the TeX) can count as a text match for a pair with a strong key. Version names the thresholds.
package match

import (
	"hash/fnv"
	"math/big"
	"slices"
	"strings"

	"github.com/tamnd/olymmath-agent/normalize"
)

// Version names the matcher and its thresholds. Change it when a threshold changes.
const Version = "match/2"

// Thresholds of the signal T.
const (
	// Same is the Jaccard similarity at which two texts are the same problem.
	Same = 0.8
	// Doubt is the Jaccard similarity below which a key match is doubtful: the key or the text is wrong.
	Doubt = 0.5
	// Duplicate is the Jaccard similarity at which audit rule D01 requires a variant_of edge or a review.
	Duplicate = 0.95
)

// N is the length of a shingle in words.
const N = 5

// Shingles returns the sorted set of the hashes of the word 5-grams of a normalized text. A text of fewer than N tokens gives one shingle of all of its tokens.
func Shingles(text string) []uint64 {
	toks := normalize.Tokens(text)
	if len(toks) == 0 {
		return nil
	}
	var out []uint64
	add := func(ws []string) {
		h := fnv.New64a()
		_, _ = h.Write([]byte(strings.Join(ws, "\x00")))
		out = append(out, h.Sum64())
	}
	if len(toks) < N {
		add(toks)
	}
	for i := 0; i+N <= len(toks); i++ {
		add(toks[i : i+N])
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// Jaccard returns the Jaccard similarity of two sorted sets.
func Jaccard(a, b []uint64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	i, j, both := 0, 0, 0
	for i < len(a) && j < len(b) {
		switch {
		case a[i] == b[j]:
			both++
			i++
			j++
		case a[i] < b[j]:
			i++
		default:
			j++
		}
	}
	return float64(both) / float64(len(a)+len(b)-both)
}

// Key is the strength of the signal K.
type Key int

// The strengths of the signal K.
const (
	// NoKey means that the candidates do not locate to the same appearance.
	NoKey Key = iota
	// WeakKey means that the same locator comes from the free text of a dataset.
	WeakKey
	// StrongKey means that both locators come from H1 or H2 metadata.
	StrongKey
)

// Answer is the result of the signal A.
type Answer int

// The results of the signal A.
const (
	// NoAnswer means that one candidate has no answer.
	NoAnswer Answer = iota
	// AnswerMatch means that the answers are equal.
	AnswerMatch
	// AnswerMismatch means that the answers differ. This is a strong signal of a variant.
	AnswerMismatch
)

// Signals are the inputs of one decision.
type Signals struct {
	K Key
	T float64
	A Answer
	// E is the cosine of the TF-IDF weights of the words and the TeX of the two texts, or 0 when it was not measured.
	E float64
}

// EStrict is the least signal E that counts as a text match for a pair with a strong key. Two transcriptions of one problem can have a low T, because their TeX and their words differ in form, but they keep a high E.
const EStrict = 0.7

// Decision is the result of the matcher.
type Decision int

// The decisions.
const (
	// New means that the candidate is a new problem.
	New Decision = iota
	// SameAuto means that the candidate is the same problem, with no review.
	SameAuto
	// Review means that a person must decide.
	Review
)

func (d Decision) String() string {
	return [...]string{"new", "same", "review"}[d]
}

// Decide applies the rules of spec 06.5 with the signals K, T and A.
func Decide(s Signals) Decision {
	switch {
	case s.A == AnswerMismatch && (s.T >= Same || s.K != NoKey):
		return Review
	case s.K != NoKey && s.T > 0 && s.T < Doubt:
		return Review
	case s.K == StrongKey:
		return SameAuto
	case s.T >= Same:
		return SameAuto
	case s.K == WeakKey:
		// A weak key with no text to compare.
		return SameAuto
	}
	return New
}

// Strict reports whether a merge has two strict signals: a strong key and a text match, or a key and an answer match with a text match. A text match is T of at least Same, or E of at least EStrict when the key is strong. Audit rule D03 accepts such a merge with no review.
func Strict(s Signals) bool {
	n := 0
	if s.K == StrongKey {
		n++
	}
	if s.T >= Same || (s.K == StrongKey && s.E >= EStrict) {
		n++
	}
	if s.A == AnswerMatch {
		n++
	}
	return n >= 2
}

// Answers compares two answers after a light normalization: no TeX dollars, no spaces, no final period and no \boxed.
func Answers(a, b string) Answer {
	a, b = answerKey(a), answerKey(b)
	switch {
	case a == "" || b == "":
		return NoAnswer
	case a == b:
		return AnswerMatch
	}
	return AnswerMismatch
}

func answerKey(s string) string {
	s = strings.NewReplacer("$", "", " ", "", "\\boxed", "", "\\text", "", "{", "", "}", "", "\\dfrac", "\\frac", "\\tfrac", "\\frac").Replace(s)
	s = strings.TrimRight(s, ".")
	if f, ok := number(s); ok {
		return f
	}
	return strings.ToLower(s)
}

// number gives the canonical form of an integer, so that "042" and "42" are equal.
func number(s string) (string, bool) {
	n, ok := new(big.Int).SetString(s, 10)
	if !ok {
		return "", false
	}
	return n.String(), true
}
