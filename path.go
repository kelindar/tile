// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"math"
	"math/bits"
	"sync"

	"github.com/kelindar/intmap"
)

type costFn = func(Value) uint16

// Edge represents an edge of the path
type edge struct {
	Point
	Cost uint32
}

// Around performs a breadth first search around a point.
func (m *Grid[T]) Around(from Point, distance uint32, costOf costFn, fn func(Point, Tile[T])) {
	start, ok := m.At(from.X, from.Y)
	if !ok {
		return
	}

	fn(from, start)

	// For pre-allocating, we use πr2 since BFS will result in a approximation
	// of a circle, in the worst case.
	maxArea := int(math.Ceil(math.Pi * float64(distance*distance)))

	// Acquire a frontier heap for search
	state := acquire(maxArea)
	frontier := state.frontier
	reached := state.edges
	defer release(state)

	frontier.Push(from.Integer(), 0)
	reached.Store(from.Integer(), 0)
	for !frontier.IsEmpty() {
		pCurr := frontier.Pop()
		current := unpackPoint(pCurr)

		// Get all of the neighbors
		m.Neighbors(current.X, current.Y, func(next Point, nextTile Tile[T]) {
			if d := from.DistanceTo(next); d > distance {
				return // Too far
			}

			if cost := costOf(nextTile.Value()); cost == 0 {
				return // Blocked tile, ignore completely
			}

			// Add to the search queue
			pNext := next.Integer()
			if _, ok := reached.Load(pNext); !ok {
				frontier.Push(pNext, 1)
				reached.Store(pNext, 1)
				fn(next, nextTile)
			}
		})
	}
}

// Path calculates a short path and the distance between the two locations
func (m *Grid[T]) Path(from, to Point, costOf costFn) ([]Point, int, bool) {
	distance := float64(from.DistanceTo(to))
	maxArea := int(math.Ceil(math.Pi * float64(distance*distance)))

	// For pre-allocating, we use πr2 since BFS will result in a approximation
	// of a circle, in the worst case.
	state := acquire(maxArea)
	edges := state.edges
	frontier := state.frontier
	defer release(state)

	frontier.Push(from.Integer(), 0)
	edges.Store(from.Integer(), encode(0, Direction(0))) // Starting point has no direction

	for !frontier.IsEmpty() {
		pCurr := frontier.Pop()
		current := unpackPoint(pCurr)

		// Decode the cost to reach the current point
		currentEncoded, _ := edges.Load(pCurr)
		currentCost, _ := decode(currentEncoded)

		// Check if we've reached the destination
		if current.Equal(to) {

			// Reconstruct the path
			path := make([]Point, 0, 64)
			path = append(path, current)
			for !current.Equal(from) {
				currentEncoded, _ := edges.Load(current.Integer())
				_, dir := decode(currentEncoded)
				current = current.Move(oppositeDirection(dir))
				path = append(path, current)
			}

			// Reverse the path to get from source to destination
			for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
				path[i], path[j] = path[j], path[i]
			}

			return path, int(currentCost), true
		}

		// Explore neighbors
		m.Neighbors(current.X, current.Y, func(next Point, nextTile Tile[T]) {
			cNext := costOf(nextTile.Value())
			if cNext == 0 {
				return // Blocked tile
			}

			nextCost := currentCost + uint32(cNext)
			pNext := next.Integer()

			existingEncoded, visited := edges.Load(pNext)
			existingCost, _ := decode(existingEncoded)

			// If we haven't visited this node or we found a better path
			if !visited || nextCost < existingCost {
				angle := angleOf(current, next)
				priority := nextCost + next.DistanceTo(to)

				// Store the edge and push to the frontier
				edges.Store(pNext, encode(nextCost, angle))
				frontier.Push(pNext, priority)
			}
		})
	}

	return nil, 0, false
}

// encode packs the cost and direction into a uint32
func encode(cost uint32, dir Direction) uint32 {
	return (cost << 4) | uint32(dir&0xF)
}

// decode unpacks the cost and direction from a uint32
func decode(value uint32) (cost uint32, dir Direction) {
	cost = value >> 4
	dir = Direction(value & 0xF)
	return
}

// -----------------------------------------------------------------------------

type pathfinder struct {
	edges    *intmap.Map
	frontier *frontier
}

var pathfinders = sync.Pool{
	New: func() any {
		return &pathfinder{
			edges:    intmap.New(32, .95),
			frontier: newFrontier(),
		}
	},
}

// Acquires a new instance of a pathfinding state
func acquire(capacity int) *pathfinder {
	v := pathfinders.Get().(*pathfinder)
	if v.edges.Capacity() < capacity {
		v.edges = intmap.New(capacity, .95)
	}

	return v
}

// release releases a pathfinding state back to the pool
func release(v *pathfinder) {
	v.edges.Clear()
	v.frontier.Reset()
	pathfinders.Put(v)
}

// -----------------------------------------------------------------------------

// frontier is a priority queue implementation that uses buckets to store
// elements. Original implementation by Iskander Sharipov (https://github.com/quasilyte/pathing)
type frontier struct {
	buckets [64][]uint32
	mask    uint64
}

// newFrontier creates a new frontier priority queue
func newFrontier() *frontier {
	h := &frontier{}
	for i := range &h.buckets {
		h.buckets[i] = make([]uint32, 0, 16)
	}
	return h
}

func (q *frontier) Reset() {
	buckets := &q.buckets

	// Reslice storage slices back.
	// To avoid traversing all len(q.buckets),
	// we have some offset to skip uninteresting (already empty) buckets.
	// We also stop when mask is 0 meaning all remaining buckets are empty too.
	// In other words, it would only touch slices between min and max non-empty priorities.
	mask := q.mask
	offset := uint(bits.TrailingZeros64(mask))
	mask >>= offset
	i := offset
	for mask != 0 {
		if i < uint(len(buckets)) {
			buckets[i] = buckets[i][:0]
		}
		mask >>= 1
		i++
	}

	q.mask = 0
}

func (q *frontier) IsEmpty() bool {
	return q.mask == 0
}

func (q *frontier) Push(value, priority uint32) {
	// No bound checks since compiler knows that i will never exceed 64.
	// We also get a cool truncation of values above 64 to store them
	// in our biggest bucket.
	i := priority & 0b111111
	q.buckets[i] = append(q.buckets[i], value)
	q.mask |= 1 << i
}

func (q *frontier) Pop() uint32 {
	buckets := &q.buckets

	// Using uints here and explicit len check to avoid the
	// implicitly inserted bound check.
	i := uint(bits.TrailingZeros64(q.mask))
	if i < uint(len(buckets)) {
		e := buckets[i][len(buckets[i])-1]
		buckets[i] = buckets[i][:len(buckets[i])-1]
		if len(buckets[i]) == 0 {
			q.mask &^= 1 << i
		}
		return e
	}

	// A queue is empty
	return 0
}
