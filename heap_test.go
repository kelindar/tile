// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.
// This is a fork of https://github.com/lemire/fastheap

package tile

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHeap(t *testing.T) {
	h := newHeap32()
	h.Push(1, 0)
	h.Pop()
}

func TestNewHeap(t *testing.T) {
	h := newHeap32()
	for j := 0; j < 8; j++ {
		h.Push(rand(j), uint32(j))
	}

	val, _ := h.Pop()
	for j := 1; j < 128; j++ {
		newval, ok := h.Pop()
		if ok {
			assert.True(t, val < newval)
			val = newval
		}
	}
}

func testGoHeap(t *testing.T) {
	pq := make(pqueue, 0)
	for j := 0; j < 128; j++ {
		heap.Push(&pq, rand(j))
	}
	val := heap.Pop(&pq).(uint32)
	for j := 1; j < 128; j++ {
		newval := heap.Pop(&pq).(uint32)
		if val < newval {
			t.Errorf("Failed")
		}
		val = newval
	}
}

// very fast semi-random function
func rand(i int) uint32 {
	i = i + 10000
	i = i ^ (i << 16)
	i = (i >> 5) ^ i
	return uint32(i & 0xFF)
}

func BenchmarkHeap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := newHeap32()
		for j := 0; j < 128; j++ {
			h.Push(rand(j), 1)
		}
		for j := 0; j < 128*10; j++ {
			h.Push(rand(j), 1)
			h.Pop()
		}
	}
}

func BenchmarkGoHeap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pq := make(pqueue, 0)
		for j := 0; j < 128; j++ {
			heap.Push(&pq, rand(j))
		}
		for j := 0; j < 128*10; j++ {
			heap.Push(&pq, rand(j))
			heap.Pop(&pq)
		}
	}
}

// -----------------------------------------------------------------------------

type pqueue []uint32

func (pq pqueue) Len() int { return len(pq) }
func (pq pqueue) Less(i, j int) bool {
	return pq[i] < pq[j]
}
func (pq pqueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *pqueue) Push(x interface{}) {
	*pq = append(*pq, x.(uint32))
}

func (pq *pqueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}
