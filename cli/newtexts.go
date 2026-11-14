package cli

import (
	"container/heap"
	"math"
)

// newTexts keeps the new texts of one dataset for the groups, each text once. When the texts need more than limit bytes, it keeps a sample: the texts with the smallest keys, as many as fit in limit. A key is a hash of the text, so the sample is a fair one, and the same texts give the same sample.
type newTexts struct {
	limit, bytes int
	// seen holds the key of each new text, so that a text counts once.
	seen map[uint64]bool
	kept map[uint64]string
	keys keyHeap
	// cut is the smallest key that the sample left out. A text with a key of cut or more is not kept. It is math.MaxUint64 while the sample holds every text.
	cut uint64
}

func newNewTexts(limit int) *newTexts {
	return &newTexts{limit: limit, seen: map[uint64]bool{}, kept: map[uint64]string{}, cut: math.MaxUint64}
}

func (n *newTexts) add(key uint64, text string) {
	if n.seen[key] {
		return
	}
	n.seen[key] = true
	if key >= n.cut {
		return
	}
	n.kept[key] = text
	heap.Push(&n.keys, key)
	n.bytes += len(text)
	for n.bytes > n.limit && n.keys.Len() > 0 {
		k := heap.Pop(&n.keys).(uint64)
		n.bytes -= len(n.kept[k])
		delete(n.kept, k)
		n.cut = k
	}
}

// texts gives the kept texts.
func (n *newTexts) texts() []string {
	out := make([]string, 0, len(n.kept))
	for _, k := range n.keys {
		out = append(out, n.kept[k])
	}
	return out
}

// sampled tells whether the sample left out some texts.
func (n *newTexts) sampled() bool { return len(n.kept) < len(n.seen) }

// count sets the new problems of a: the groups of the texts, or, for a sample, the groups of the sample scaled to all the new texts.
func (n *newTexts) count(a *Against, workers int) {
	a.NewTexts = len(n.seen)
	g := groups(n.texts(), workers)
	if !n.sampled() {
		a.NewProblems = g
		return
	}
	a.Sample, a.SampleProblems = len(n.kept), g
	a.NewProblems = int(math.Round(float64(g) * float64(len(n.seen)) / float64(len(n.kept))))
}

// keyHeap is a heap of keys with the largest key first.
type keyHeap []uint64

func (h keyHeap) Len() int           { return len(h) }
func (h keyHeap) Less(i, j int) bool { return h[i] > h[j] }
func (h keyHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }
func (h *keyHeap) Push(x any)        { *h = append(*h, x.(uint64)) }
func (h *keyHeap) Pop() any {
	old := *h
	x := old[len(old)-1]
	*h = old[:len(old)-1]
	return x
}
