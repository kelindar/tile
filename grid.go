// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"reflect"
	"sync"
	"sync/atomic"
	"unsafe"
)

// Grid represents a 2D tile map. Internally, a map is composed of 3x3 pages.
type Grid[T comparable] struct {
	pages      []page[T] // The pages of the map
	pageWidth  int16     // The max page width
	pageHeight int16     // The max page height
	observers  pubsub    // The map of observers
	Size       Point     // The map size
}

// NewGrid returns a new map of the specified size. The width and height must be both
// multiples of 3.
func NewGrid(width, height int16) *Grid[string] {
	return NewGridOf[string](width, height)
}

// NewGridOf returns a new map of the specified size. The width and height must be both
// multiples of 3.
func NewGridOf[T comparable](width, height int16) *Grid[T] {
	width, height = width/3, height/3

	max := int32(width) * int32(height)
	pages := make([]page[T], max)
	m := &Grid[T]{
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

	for i := 0; i < int(max); i++ {
		pages[i].point = pointAt(i).MultiplyScalar(3)
	}
	return m
}

// Each iterates over all of the tiles in the map.
func (m *Grid[T]) Each(fn func(Point, Cursor[T])) {
	until := int(m.pageHeight) * int(m.pageWidth)
	for i := 0; i < until; i++ {
		m.pages[i].Each(fn)
	}
}

// Within selects the tiles within a specifid bounding box which is specified by
// north-west and south-east coordinates.
func (m *Grid[T]) Within(nw, se Point, fn func(Point, Cursor[T])) {
	m.pagesWithin(nw, se, func(page *page[T]) {
		page.Each(func(p Point, v Cursor[T]) {
			if p.Within(nw, se) {
				fn(p, v)
			}
		})
	})
}

// pagesWithin selects the pages within a specifid bounding box which is specified
// by north-west and south-east coordinates.
func (m *Grid[T]) pagesWithin(nw, se Point, fn func(*page[T])) {
	if !se.WithinSize(m.Size) {
		se = At(m.Size.X-1, m.Size.Y-1)
	}

	for x := nw.X / 3; x <= se.X/3; x++ {
		for y := nw.Y / 3; y <= se.Y/3; y++ {
			fn(m.pageAt(x, y))
		}
	}
}

// At returns the tile at a specified position
func (m *Grid[T]) At(x, y int16) (Cursor[T], bool) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		return m.pageAt(x/3, y/3).At(x, y), true
	}

	return Cursor[T]{}, false
}

// WriteAt updates the entire tile value at a specific coordinate
func (m *Grid[T]) WriteAt(x, y int16, tile Tile) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		if m.pageAt(x/3, y/3).SetTile(x, y, tile) {
			m.observers.Notify(At(x/3*3, y/3*3), At(x, y), tile)
		}
	}
}

// MergeAt updates the bits of tile at a specific coordinate. The bits are specified
// by the mask. The bits that need to be updated should be flipped on in the mask.
func (m *Grid[T]) MergeAt(x, y int16, tile, mask Tile) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		if v, ok := m.pageAt(x/3, y/3).SetBits(x, y, tile, mask); ok {
			m.observers.Notify(At(x/3*3, y/3*3), At(x, y), v)
		}
	}
}

// NotifyAt triggers the notification event for all of the observers at a given tile.
func (m *Grid[T]) NotifyAt(x, y int16) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		m.observers.Notify(At(x/3*3, y/3*3), At(x, y),
			m.pageAt(x/3, y/3).Get(x, y))
	}
}

// Neighbors iterates over the direct neighbouring tiles
func (m *Grid[T]) Neighbors(x, y int16, fn func(Point, Cursor[T])) {

	// First we need to figure out which pages contain the neighboring tiles and
	// then load them. In the best-case we need to load only a single page. In
	// the worst-case: we need to load 3 pages.
	nX, nY := x/3, (y-1)/3 // North
	eX, eY := (x+1)/3, y/3 // East
	sX, sY := x/3, (y+1)/3 // South
	wX, wY := (x-1)/3, y/3 // West

	// Get the North
	if y > 0 {
		fn(At(x, y-1), m.pageAt(nX, nY).At(x, y-1))
	}

	// Get the East
	if eX < m.pageWidth {
		fn(At(x+1, y), m.pageAt(eX, eY).At(x+1, y))
	}

	// Get the South
	if sY < m.pageHeight {
		fn(At(x, y+1), m.pageAt(sX, sY).At(x, y+1))
	}

	// Get the West
	if x > 0 {
		fn(At(x-1, y), m.pageAt(wX, wY).At(x-1, y))
	}
}

// View creates a new view of the map.
func (m *Grid[T]) View(rect Rect, fn func(Point, Cursor[T])) *View[T] {
	view := &View[T]{
		Grid:  m,
		Inbox: make(chan Update, 16),
		rect:  NewRect(-1, -1, -1, -1),
	}

	// Call the resize method
	view.Resize(rect, fn)
	return view
}

// pageAt loads a page at a given page location
func (m *Grid[T]) pageAt(x, y int16) *page[T] {
	index := int(x) + int(m.pageWidth)*int(y)

	// Eliminate bounds checks
	if index >= 0 && index < len(m.pages) {
		return &m.pages[index]
	}

	return nil
}

// ---------------------------------- Tile ----------------------------------

// Tile represents a packed tile information, it must fit on 4 bytes.
type Tile uint32

// ---------------------------------- Page ----------------------------------

// page represents a 3x3 tile page each page should neatly fit on a cache
// line and speed things up.
type page[T comparable] struct {
	mu    sync.Mutex  // State lock, 8 bytes
	state map[T]uint8 // State data, 8 bytes
	flags uint32      // Page flags, 4 bytes
	point Point       // Page X, Y coordinate, 4 bytes
	tiles [9]Tile     // Page tiles, 36 bytes
}

// tileAt reads a tile at a page index
func (p *page[T]) tileAt(idx uint8) Tile {
	return Tile(atomic.LoadUint32((*uint32)(&p.tiles[idx])))
}

// writeTile stores the tile and return  whether tile is observed or not
func (p *page[T]) writeTile(idx uint8, tile Tile) bool {
	atomic.StoreUint32((*uint32)(&p.tiles[idx]), uint32(tile))
	return p.isObserved()
}

func (p *page[T]) isObserved() bool {
	return (atomic.LoadUint32(&p.flags))&1 != 0
}

// Bounds returns the bounding box for the tile page.
func (p *page[T]) Bounds() Rect {
	return Rect{p.point, At(p.point.X+3, p.point.Y+3)}
}

// SetTile updates the tile at a specific coordinate
func (p *page[T]) SetTile(x, y int16, tile Tile) bool {
	return p.writeTile(uint8((y%3)*3+(x%3)), tile)
}

// SetBits updates certain tile bits at a specific coordinate
func (p *page[T]) SetBits(x, y int16, tile, mask Tile) (Tile, bool) {
	i := uint8((y%3)*3 + (x % 3))

	// Merge current value with the tile and mask
	value := p.tileAt(i)
	merge := (value &^ mask) | (tile & mask)

	// Swap, if we're not able to re-merge again
	for !atomic.CompareAndSwapUint32((*uint32)(&p.tiles[i]), uint32(value), uint32(merge)) {
		value = p.tileAt(i)
		merge = (value &^ mask) | (tile & mask)
	}

	// Return the merged tile data and whether tile is observed or not
	return merge, p.isObserved()
}

// Get gets a tile at a specific coordinate.
func (p *page[T]) Get(x, y int16) Tile {
	return p.tileAt(uint8((y%3)*3 + (x % 3)))
}

// At returns a cursor at a specific coordinate
func (p *page[T]) At(x, y int16) Cursor[T] {
	return Cursor[T]{data: p, idx: uint8((y%3)*3 + (x % 3))}
}

// Each iterates over all of the tiles in the page.
func (p *page[T]) Each(fn func(Point, Cursor[T])) {
	x, y := p.point.X, p.point.Y
	fn(Point{x, y}, Cursor[T]{data: p, idx: 0})         // NW
	fn(Point{x + 1, y}, Cursor[T]{data: p, idx: 1})     // N
	fn(Point{x + 2, y}, Cursor[T]{data: p, idx: 2})     // NE
	fn(Point{x, y + 1}, Cursor[T]{data: p, idx: 3})     // W
	fn(Point{x + 1, y + 1}, Cursor[T]{data: p, idx: 4}) // C
	fn(Point{x + 2, y + 1}, Cursor[T]{data: p, idx: 5}) // E
	fn(Point{x, y + 2}, Cursor[T]{data: p, idx: 6})     // SW
	fn(Point{x + 1, y + 2}, Cursor[T]{data: p, idx: 7}) // S
	fn(Point{x + 2, y + 2}, Cursor[T]{data: p, idx: 8}) // SE
}

// SetObserved sets the observed flag on the page
func (p *page[T]) SetObserved(observed bool) {
	const flagObserved = 0x1
	for {
		value := atomic.LoadUint32(&p.flags)
		merge := value
		if observed {
			merge = value | flagObserved
		} else {
			merge = value &^ flagObserved
		}

		if atomic.CompareAndSwapUint32(&p.flags, value, merge) {
			break
		}
	}
}

// Lock locks the state. Note: this needs to be named Lock() so go vet will
// complain if the page is copied around.
func (p *page[T]) Lock() {
	p.mu.Lock()
}

// Unlock unlocks the state. Note: this needs to be named Unlock() so go vet will
// complain if the page is copied around.
func (p *page[T]) Unlock() {
	p.mu.Unlock()
}

// Data returns a buffer to the tile data, without allocations.
func (p *page[T]) Data() []byte {
	var out reflect.SliceHeader
	out.Data = reflect.ValueOf(&p.tiles).Pointer()
	out.Len = tileDataSize
	out.Cap = tileDataSize
	return *(*[]byte)(unsafe.Pointer(&out))
}

// ---------------------------------- Cursor ----------------------------------

// Cursor represents an iterator over all state objects at a particular location.
type Cursor[T comparable] struct {
	data *page[T]
	idx  uint8
}

// Count returns number of objects at the current tile.
func (c Cursor[T]) Count() (count int) {
	c.data.Lock()
	defer c.data.Unlock()
	for _, idx := range c.data.state {
		if idx == uint8(c.idx) {
			count++
		}
	}
	return
}

// Tile reads the tile information
func (c Cursor[T]) Tile() Tile {
	return c.data.tileAt(c.idx)
}

// Range iterates over all of the objects in the set
func (c Cursor[T]) Range(fn func(T) error) error {
	c.data.Lock()
	defer c.data.Unlock()
	for v, idx := range c.data.state {
		if idx == uint8(c.idx) {
			if err := fn(v); err != nil {
				return err
			}
		}
	}
	return nil
}

// Add adds object to the set
func (c Cursor[T]) Add(v T) {
	c.data.Lock()
	defer c.data.Unlock()

	// Lazily initialize the map, as most pages might not have anything stored
	// in them (e.g. water or empty tile)
	if c.data.state == nil {
		c.data.state = make(map[T]uint8)
	}

	c.data.state[v] = uint8(c.idx)
}

// Del removes the object from the set
func (c Cursor[T]) Del(v T) {
	c.data.Lock()
	defer c.data.Unlock()
	if c.data.state != nil {
		delete(c.data.state, v)
	}
}
