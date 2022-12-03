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

// WriteAt updates the entire tile value at a specific coordinate
func (m *Grid) WriteAt(x, y int16, tile Tile) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		if m.pages[m.indexOf(x/3, y/3)].SetTile(x, y, tile) {
			m.observers.Notify(At(x/3*3, y/3*3), At(x, y), tile)
		}
	}
}

// MergeAt updates the bits of tile at a specific coordinate. The bits are specified
// by the mask. The bits that need to be updated should be flipped on in the mask.
func (m *Grid) MergeAt(x, y int16, tile, mask Tile) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		if v, ok := m.pages[m.indexOf(x/3, y/3)].SetBits(x, y, tile, mask); ok {
			m.observers.Notify(At(x/3*3, y/3*3), At(x, y), v)
		}
	}
}

// NotifyAt triggers the notification event for all of the observers at a given tile.
func (m *Grid) NotifyAt(x, y int16) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		tile := m.pages[m.indexOf(x/3, y/3)].Get(x, y)
		m.observers.Notify(At(x/3*3, y/3*3), At(x, y), tile)
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
		Inbox: make(chan Update, 16),
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

// SetTile updates the tile at a specific coordinate
func (p *page) SetTile(x, y int16, tile Tile) bool {
	i := (y%3)*3 + (x % 3)

	// Synchronize the update from this point on
	p.Lock()
	p.tiles[i] = tile
	notify := p.flags&1 != 0
	p.Unlock()

	// Return whether tile is observed or not
	return notify
}

// SetBits updates certain tile bits at a specific coordinate
func (p *page) SetBits(x, y int16, tile, mask Tile) (Tile, bool) {
	t := uint64(tile[0]) | uint64(tile[1])<<8 | uint64(tile[2])<<16 |
		uint64(tile[3])<<24 | uint64(tile[4])<<32 | uint64(tile[5])<<40
	m := uint64(mask[0]) | uint64(mask[1])<<8 | uint64(mask[2])<<16 |
		uint64(mask[3])<<24 | uint64(mask[4])<<32 | uint64(mask[5])<<40
	i := (y%3)*3 + (x % 3)

	// Get the tile and do the binary merge
	p.Lock()
	b := &p.tiles[i]
	v := uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 |
		uint64(b[3])<<24 | uint64(b[4])<<32 | uint64(b[5])<<40
	v = (v &^ m) | (t & m)

	// Write the merged result back
	b[0] = byte(v)
	b[1] = byte(v >> 8)
	b[2] = byte(v >> 16)
	b[3] = byte(v >> 24)
	b[4] = byte(v >> 32)
	b[5] = byte(v >> 40)
	merged, notify := *b, p.flags&1 != 0
	p.Unlock()

	// Return the merged tile data and whether tile is observed or not
	return merged, notify
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
