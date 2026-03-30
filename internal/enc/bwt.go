// SPDX-License-Identifier: 0BSD
// Copyright (c)2026 Quad4.io

package enc

import "sort"

// Scratch holds reusable buffers for block encoding (BWT and MTF stages).
type Scratch struct {
	sa, rank, tmp []int
	mtfv            []uint16
	yy              []byte
	mtfFreq         []int32
}

func (s *Scratch) grow(n int) {
	if cap(s.sa) >= n {
		s.sa = s.sa[:n]
		s.rank = s.rank[:n]
		s.tmp = s.tmp[:n]
		return
	}
	s.sa = make([]int, n)
	s.rank = make([]int, n)
	s.tmp = make([]int, n)
}

// buildCyclicSuffixArray builds the sorted cyclic suffix array of block and returns the
// index of the original string (rotation starting at 0) in that order.
func buildCyclicSuffixArray(block []byte, sc *Scratch) (sa []int, origPtr int) {
	n := len(block)
	if n == 0 {
		return nil, 0
	}
	sc.grow(n)
	sa = sc.sa
	rank := sc.rank
	tmp := sc.tmp
	for i := range sa {
		sa[i] = i
	}
	for i := range rank {
		rank[i] = int(block[i])
	}
	for k := 1; k < n; k *= 2 {
		sort.Slice(sa, func(i, j int) bool {
			a, b := sa[i], sa[j]
			ra, rb := rank[a], rank[b]
			if ra != rb {
				return ra < rb
			}
			return rank[(a+k)%n] < rank[(b+k)%n]
		})
		tmp[sa[0]] = 0
		for i := 1; i < n; i++ {
			a, b := sa[i-1], sa[i]
			same := rank[a] == rank[b] && rank[(a+k)%n] == rank[(b+k)%n]
			tmp[sa[i]] = tmp[sa[i-1]]
			if !same {
				tmp[sa[i]]++
			}
		}
		copy(rank, tmp)
		if rank[sa[n-1]] == n-1 {
			break
		}
	}
	for i := range sa {
		if sa[i] == 0 {
			origPtr = i
			break
		}
	}
	return sa, origPtr
}
