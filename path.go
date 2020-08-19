// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

// Edge represents an edge of the path
type edge struct {
	Point
	Cost uint32
}

type pathFn = func(Point)

// Path calculates a short path and the distance between the two locations
func (m *Map) Path(from, to Point) ([]Point, int, bool) {
	frontier := newHeap32()
	frontier.Push(from.Integer(), 0)

	// Add the first edge
	capacity := int(float32(from.ManhattanDistance(to)) * 1.5)
	edges := make(map[uint32]edge, capacity)
	edges[from.Integer()] = edge{
		Point: from,
		Cost:  0,
	}

	for !frontier.IsEmpty() {
		pCurr, _ := frontier.Pop()
		current := unpackPoint(pCurr)

		// We have a path to the goal
		if current.Equal(to) {
			dist := int(edges[current.Integer()].Cost)
			path := make([]Point, 0, dist)
			curr, _ := edges[current.Integer()]
			for !curr.Point.Equal(from) {
				path = append(path, curr.Point)
				curr = edges[curr.Point.Integer()]
			}

			return path, dist, true
		}

		// Get all of the neighbors
		m.Neighbors(current.X, current.Y, func(next Point, _ Tile) {
			pNext := next.Integer()
			newCost := edges[pCurr].Cost + 1 // cost(current, next)

			if e, ok := edges[pNext]; !ok || newCost < e.Cost {
				priority := newCost + next.ManhattanDistance(to) // heuristic
				frontier.Push(next.Integer(), priority)

				edges[pNext] = edge{
					Point: current,
					Cost:  newCost,
				}
			}

		})
	}

	return nil, 0, false
}
