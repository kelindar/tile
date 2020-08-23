// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"reflect"
	"runtime"
	"sync/atomic"
	"unsafe"
)

// Iterator represents an iterator function.
type Iterator = func(Point, Tile)
type pageFn = func(*page)
type indexFn = func(x, y int16) int
type pointFn = func(i int) Point

// Grid represents a 2D tile map. Internally, a map is composed of 3x3 pages.
type Grid struct {
	pages      []page  // The pages of the map
	pageWidth  int16   // The max page width
	pageHeight int16   // The max page height
	observers  pubsub  // The map of observers
	Size       Point   // The map size
	indexOf    indexFn // The page index function
}

// NewGrid returns a new map of the specified size. The width and height must be both
// multiples of 3.
func NewGrid(width, height int16) *Grid {
	width, height = width/3, height/3

	max := int32(width) * int32(height)
	pages := make([]page, max)
	m := &Grid{
		pages:      pages,
		pageWidth:  width,
		pageHeight: height,
		observers:  pubsub{},
		Size:       At(width*3, height*3),
	}

	// Function to calculate a point based on the index
	var pointAt func(i int) Point = func(i int) Point {
		return At(int16(i%int(width)), int16(i/int(width)))
	}
	m.indexOf = m.pointToFlat

	// If the map is square and page count is a power of 2, use z-curve filling instead
	// as this will speed up data access under certain conditions.
	if width == height && (width&(width-1)) == 0 {
		pointAt = deinterleavePoint
		m.indexOf = m.pointToZ
	}

	for i := 0; i < int(max); i++ {
		pages[i].point = pointAt(i).MultiplyScalar(3)
	}
	return m
}

func (m *Grid) pointToFlat(x, y int16) int {
	return int(x) + int(m.pageWidth)*int(y)
}

func (m *Grid) pointToZ(x, y int16) int {
	return int(At(x, y).Interleave())
}

// Each iterates over all of the tiles in the map.
func (m *Grid) Each(fn Iterator) {
	until := int(m.pageHeight) * int(m.pageWidth)
	for i := 0; i < until; i++ {
		m.pages[i].Each(fn)
	}
}

// Within selects the tiles within a specifid bounding box which is specified by
// north-west and south-east coordinates.
func (m *Grid) Within(nw, se Point, fn Iterator) {
	m.pagesWithin(nw, se, func(page *page) {
		page.Each(func(p Point, tile Tile) {
			if p.Within(nw, se) {
				fn(p, tile)
			}
		})
	})
}

// pagesWithin selects the pages within a specifid bounding box which is specified
// by north-west and south-east coordinates.
func (m *Grid) pagesWithin(nw, se Point, fn pageFn) {
	if !se.WithinSize(m.Size) {
		se = At(m.Size.X-1, m.Size.Y-1)
	}

	for x := nw.X / 3; x <= se.X/3; x++ {
		for y := nw.Y / 3; y <= se.Y/3; y++ {
			fn(&m.pages[m.indexOf(x, y)])
		}
	}
}

// At returns the tile at a specified position
func (m *Grid) At(x, y int16) (Tile, bool) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		return m.pages[m.indexOf(x/3, y/3)].Get(x, y), true
	}

	return Tile{}, false
}

// UpdateAt updates the tile at a specific coordinate
func (m *Grid) UpdateAt(x, y int16, tile Tile) {

	// Update the tile in the map
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		if m.pages[m.indexOf(x/3, y/3)].Set(x, y, tile) {
			// Notify the observers, if any
			m.observers.Notify(At(x/3*3, y/3*3), At(x, y), tile)
		}
	}
}

// Neighbors iterates over the direct neighbouring tiles
func (m *Grid) Neighbors(x, y int16, fn Iterator) {

	// First we need to figure out which pages contain the neighboring tiles and
	// then load them. In the best-case we need to load only a single page. In
	// the worst-case: we need to load 3 pages.
	nX, nY := x/3, (y-1)/3 // North
	eX, eY := (x+1)/3, y/3 // East
	sX, sY := x/3, (y+1)/3 // South
	wX, wY := (x-1)/3, y/3 // West

	// Get the North
	if y > 0 {
		fn(At(x, y-1), m.pages[m.indexOf(nX, nY)].Get(x, y-1))
	}

	// Get the East
	if eX < m.pageWidth {
		fn(At(x+1, y), m.pages[m.indexOf(eX, eY)].Get(x+1, y))
	}

	// Get the South
	if sY < m.pageHeight {
		fn(At(x, y+1), m.pages[m.indexOf(sX, sY)].Get(x, y+1))
	}

	// Get the West
	if x > 0 {
		fn(At(x-1, y), m.pages[m.indexOf(wX, wY)].Get(x-1, y))
	}
}

// View creates a new view of the map.
func (m *Grid) View(rect Rect, fn Iterator) *View {
	view := &View{
		Grid:  m,
		Inbox: make(chan Update, 8),
		rect:  NewRect(-1, -1, -1, -1),
	}

	// Call the resize method
	view.Resize(rect, fn)
	return view
}

// -----------------------------------------------------------------------------

// Tile represents a packed tile information, it must fit on 6 bytes.
type Tile [6]byte

// -----------------------------------------------------------------------------

// page represents a 3x3 tile page each page should neatly fit on a cache
// line and speed things up.
type page struct {
	lock  int32   // Page spin-lock, 4 bytes
	flags uint16  // Page flags, 2 bytes
	point Point   // Page X, Y coordinate, 4 bytes
	tiles [9]Tile // Page tiles, 54 bytes
}

// Bounds returns the bounding box for the tile page.
func (p *page) Bounds() Rect {
	return Rect{p.point, At(p.point.X+3, p.point.Y+3)}
}

// Set updates the tile at a specific coordinate
func (p *page) Set(x, y int16, tile Tile) (observed bool) {
	p.Lock()
	p.tiles[(y%3)*3+(x%3)] = tile // Update the tile
	observed = p.flags&1 != 0     // Are there any observers?
	p.Unlock()
	return
}

// Get gets a tile at a specific coordinate.
func (p *page) Get(x, y int16) (tile Tile) {
	i := (y%3)*3 + (x % 3)
	p.Lock()
	tile = p.tiles[i]
	p.Unlock()
	return
}

// Each iterates over all of the tiles in the page.
func (p *page) Each(fn Iterator) {
	p.Lock()
	tiles := p.tiles
	p.Unlock()

	x, y := p.point.X, p.point.Y
	fn(Point{x, y}, tiles[0])         // NW
	fn(Point{x + 1, y}, tiles[1])     // N
	fn(Point{x + 2, y}, tiles[2])     // NE
	fn(Point{x, y + 1}, tiles[3])     // W
	fn(Point{x + 1, y + 1}, tiles[4]) // C
	fn(Point{x + 2, y + 1}, tiles[5]) // E
	fn(Point{x, y + 2}, tiles[6])     // SW
	fn(Point{x + 1, y + 2}, tiles[7]) // S
	fn(Point{x + 2, y + 2}, tiles[8]) // SE
}

// SetObserved sets the observed flag on the page
func (p *page) SetObserved(observed bool) {
	p.Lock()
	defer p.Unlock()

	if observed {
		p.flags = p.flags | 1
	} else {
		p.flags = p.flags &^ 1
	}
}

// Lock locks the spin lock. Note: this needs to be named Lock() so go vet will
// complain if the page is copied around.
func (p *page) Lock() {
	for !atomic.CompareAndSwapInt32(&p.lock, 0, 1) {
		runtime.Gosched()
	}
}

// Unlock unlocks the page. Note: this needs to be named Unlock() so go vet will
// complain if the page is copied around.
func (p *page) Unlock() {
	atomic.StoreInt32(&p.lock, 0)
}

// Data returns a buffer to the tile data, without allocations.
func (p *page) Data() []byte {
	var out reflect.SliceHeader
	out.Data = reflect.ValueOf(&p.tiles).Pointer()
	out.Len = tileDataSize
	out.Cap = tileDataSize
	return *(*[]byte)(unsafe.Pointer(&out))
}
