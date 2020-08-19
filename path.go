// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

// Edge represents an edge of the path
type edge struct {
	Point
	Cost uint32
}

// Path calculates a short path and the distance between the two locations
func (m *Map) Path(from, to Point) (path []Point, distance int, found bool) {
	frontier := newHeap32()
	frontier.Push(from.Integer(), 0)

	// Add the first edge
	edges := make(map[uint32]edge, 8)
	edges[from.Integer()] = edge{
		Point: from,
		Cost:  0,
	}

	for !frontier.IsEmpty() {
		pCurr, _ := frontier.Pop()
		current := unpackPoint(pCurr)

		// We have a path to the goal
		if current.Equal(to) {
			p := []Point{}
			curr, _ := edges[current.Integer()]
			for !curr.Point.Equal(from) {
				p = append(p, curr.Point)
				curr = edges[curr.Point.Integer()]
			}

			return p, int(edges[current.Integer()].Cost), true
		}

		// Get all of the neighbors
		m.Neighbors(current.X, current.Y, func(next Point, _ *Tile) {
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
	return
}
