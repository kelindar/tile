// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

//import "container/heap"

// Path calculates a short path and the distance between the two locations
func (m *Map) Path(from, to Point) (path []Point, distance int, found bool) {
	came_from := make(map[uint32]uint32, 8)
	cost_so_far := make(map[uint32]uint32, 8)

	frontier := newHeap32()
	frontier.Push(from.Integer(), 0)

	came_from[from.Integer()] = from.Integer()
	cost_so_far[from.Integer()] = 0

	for !frontier.IsEmpty() {
		cint, _ := frontier.Pop()
		current := unpackPoint(cint)

		// We have a path to the goal
		if current.Equal(to) {
			p := []Point{}
			curr, _ := came_from[current.Integer()]
			for curr != from.Integer() {
				p = append(p, unpackPoint(curr))
				curr = came_from[curr]
			}
			return p, int(cost_so_far[current.Integer()]), true
		}

		// Get all of the neighbors
		m.Neighbors(current.X, current.Y, func(next Point, _ *Tile) {
			new_cost := cost_so_far[cint] + 1 // cost(current, next)

			if cost, ok := cost_so_far[next.Integer()]; !ok || new_cost < cost {
				cost_so_far[next.Integer()] = new_cost
				priority := new_cost + next.ManhattanDistance(to) // heuristic
				frontier.Push(next.Integer(), priority)
				came_from[next.Integer()] = cint
			}

		})
	}
	return
}

// Path calculates a short path and the distance between the two locations
/*func (m *Map) Path(from, to Point) (path []Point, distance int, found bool) {

	nm := make(nodeMap, 200)
	//nq := &priorityQueue{}
	// heap.Init(nq)
	nq := newHeap32()
	fromNode := nm.get(from)
	fromNode.open = true
	//heap.Push(nq, fromNode)

	nq.Push(from.Integer(), 0)
	for nq.Len() > 0 {
		//current := heap.Pop(nq).(*node)
		index, _ := nq.Pop()
		current := nm[index]

		current.open = false
		current.closed = true

		//if current == nm.get(to) {
		if current.Equal(to) {
			// Found a path to the goal.
			p := []Point{}
			curr := current
			for curr != nil {
				p = append(p, curr.Point)
				curr = curr.parent
			}
			return p, int(current.cost), true
		}

		// Get all of the neighbors
		m.Neighbors(current.X, current.Y, func(point Point, _ *Tile) {
			//cost := current.cost + current.pather.PathNeighborCost(neighbor)
			cost := current.cost + 1
			neighborNode := nm.get(point)
			if cost < neighborNode.cost {
				if neighborNode.open {
					//heap.Remove(nq, neighborNode.index)
					nq.Remove(neighborNode.index)
				}
				neighborNode.open = false
				neighborNode.closed = false
			}

			if !neighborNode.open && !neighborNode.closed {
				neighborNode.cost = cost
				neighborNode.open = true
				//neighborNode.rank = cost + neighbor.PathEstimatedCost(to)
				neighborNode.rank = cost + 1
				neighborNode.parent = current
				//heap.Push(nq, neighborNode)
				nq.Push(neighborNode.Integer(), neighborNode.rank)
			}
		})
	}
	return
}*/

// -----------------------------------------------------------------------------

// node is a wrapper to store A* data for a Pather node.
type node struct {
	Point
	parent *node
	cost   uint32
	rank   uint32
	open   bool
	closed bool
	index  int
}

// nodeMap is a collection of nodes keyed by Pather nodes for quick reference.
type nodeMap map[uint32]*node

// get gets the Pather object wrapped in a node, instantiating if required.
func (nm nodeMap) get(loc Point) *node {
	index := loc.Integer()
	n, ok := nm[index]
	if !ok {
		n = &node{
			Point: loc,
		}
		nm[index] = n
	}

	return n
}

// -----------------------------------------------------------------------------

// A priorityQueue implements heap.Interface and holds Nodes.  The
// priorityQueue is used to track open nodes by rank.
/*type priorityQueue []*node

func (pq priorityQueue) Len() int {
	return len(pq)
}

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].rank < pq[j].rank
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	no := x.(*node)
	no.index = n
	*pq = append(*pq, no)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	no := old[n-1]
	no.index = -1
	*pq = old[0 : n-1]
	return no
}
*/
