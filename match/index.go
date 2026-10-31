package match

import (
	"encoding/binary"
	"hash/fnv"
)

// The MinHash signature has Bands times Rows values. With 32 bands of 4 rows, a pair at Jaccard 0.5 shares a band with a probability of about 0.87, and a pair at 0.8 with a probability of more than 0.99.
const (
	Bands = 32
	Rows  = 4
)

// Signature is the MinHash signature of a set of shingles.
type Signature [Bands * Rows]uint64

// seeds are the odd multipliers of the hash functions. They are fixed so that a signature does not change between runs.
var seeds = func() [Bands * Rows]uint64 {
	var s [Bands * Rows]uint64
	x := uint64(0x9E3779B97F4A7C15)
	for i := range s {
		x ^= x << 13
		x ^= x >> 7
		x ^= x << 17
		s[i] = x | 1
	}
	return s
}()

// Sign returns the MinHash signature of a set of shingles.
func Sign(sh []uint64) Signature {
	var sig Signature
	for i := range sig {
		sig[i] = ^uint64(0)
	}
	for _, h := range sh {
		for i, a := range seeds {
			v := (h ^ a) * a
			v ^= v >> 29
			if v < sig[i] {
				sig[i] = v
			}
		}
	}
	return sig
}

// Index finds the texts that may be similar to a text. It holds the shingles of each text, so that Near can give the exact Jaccard similarity.
type Index struct {
	buckets [Bands]map[uint64][]int
	ids     []string
	sh      [][]uint64
}

// NewIndex returns an empty index.
func NewIndex() *Index {
	x := &Index{}
	for i := range x.buckets {
		x.buckets[i] = map[uint64][]int{}
	}
	return x
}

// Len is the number of texts in the index.
func (x *Index) Len() int { return len(x.ids) }

// Add adds a text by its ID and its shingles.
func (x *Index) Add(id string, sh []uint64) {
	if len(sh) == 0 {
		return
	}
	n := len(x.ids)
	x.ids = append(x.ids, id)
	x.sh = append(x.sh, sh)
	sig := Sign(sh)
	for b := range Bands {
		k := band(sig, b)
		x.buckets[b][k] = append(x.buckets[b][k], n)
	}
}

// Hit is a text of the index and its Jaccard similarity to the query.
type Hit struct {
	ID string
	T  float64
}

// Near returns the texts of the index that share a band with sh and have a Jaccard similarity of at least minT.
func (x *Index) Near(sh []uint64, minT float64) []Hit {
	if len(sh) == 0 {
		return nil
	}
	sig := Sign(sh)
	seen := map[int]bool{}
	var hits []Hit
	for b := range Bands {
		for _, n := range x.buckets[b][band(sig, b)] {
			if seen[n] {
				continue
			}
			seen[n] = true
			if t := Jaccard(sh, x.sh[n]); t >= minT {
				hits = append(hits, Hit{ID: x.ids[n], T: t})
			}
		}
	}
	return hits
}

// Pairs calls fn once for each pair of texts of the index with a Jaccard similarity of at least minT.
func (x *Index) Pairs(minT float64, fn func(a, b string, t float64)) {
	done := map[[2]int]bool{}
	for b := range Bands {
		for _, ns := range x.buckets[b] {
			for i := 0; i < len(ns); i++ {
				for j := i + 1; j < len(ns); j++ {
					k := [2]int{ns[i], ns[j]}
					if done[k] {
						continue
					}
					done[k] = true
					if t := Jaccard(x.sh[ns[i]], x.sh[ns[j]]); t >= minT {
						fn(x.ids[ns[i]], x.ids[ns[j]], t)
					}
				}
			}
		}
	}
}

func band(sig Signature, b int) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	for _, v := range sig[b*Rows : (b+1)*Rows] {
		binary.LittleEndian.PutUint64(buf[:], v)
		_, _ = h.Write(buf[:])
	}
	return h.Sum64()
}
