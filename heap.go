// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

// heapNode represents a ranked node for the heap.
type heapNode struct {
	Value uint32 // The value of the ranked node.
	Rank  uint32 // The rank associated with the ranked node.
}

type heap32 []heapNode

func newHeap32() heap32 {
	return make(heap32, 0, 16)
}

// Push pushes the element x onto the heap.
// The complexity is O(log n) where n = h.Len().
func (h *heap32) Push(v, rank uint32) {
	*h = append(*h, heapNode{
		Value: v,
		Rank:  rank,
	})
	h.up(h.Len() - 1)
}

// Pop removes and returns the minimum element (according to Less) from the heap.
// The complexity is O(log n) where n = h.Len().
// Pop is equivalent to Remove(h, 0).
func (h *heap32) Pop() (uint32, bool) {
	n := h.Len() - 1
	if n < 0 {
		return 0, false
	}

	h.Swap(0, n)
	h.down(0, n)
	return h.pop(), true
}

// Remove removes and returns the element at index i from the heap.
// The complexity is O(log n) where n = h.Len().
func (h *heap32) Remove(i int) uint32 {
	n := h.Len() - 1
	if n != i {
		h.Swap(i, n)
		if !h.down(i, n) {
			h.up(i)
		}
	}
	return h.pop()
}

func (h *heap32) pop() uint32 {
	old := *h
	n := len(old)
	no := old[n-1]
	*h = old[0 : n-1]
	return no.Value
}

func (h *heap32) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !h.Less(j, i) {
			break
		}
		h.Swap(i, j)
		j = i
	}
}

func (h *heap32) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && h.Less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !h.Less(j, i) {
			break
		}
		h.Swap(i, j)
		i = j
	}
	return i > i0
}

func (h heap32) Len() int {
	return len(h)
}

func (h heap32) IsEmpty() bool {
	return len(h) == 0
}

func (h heap32) Less(i, j int) bool {
	return h[i].Rank < h[j].Rank
}

func (h *heap32) Swap(i, j int) {
	(*h)[i], (*h)[j] = (*h)[j], (*h)[i]
}
