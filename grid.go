// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"sync"
	"sync/atomic"
)

// Grid represents a 2D tile map. Internally, a map is composed of 3x3 pages.
type Grid[T comparable] struct {
	pages      []page[T] // The pages of the map
	pageWidth  int16     // The max page width
	pageHeight int16     // The max page height
	observers  pubsub[T] // The map of observers
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
		observers:  pubsub[T]{},
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
func (m *Grid[T]) Each(fn func(Point, Tile[T])) {
	until := int(m.pageHeight) * int(m.pageWidth)
	for i := 0; i < until; i++ {
		m.pages[i].Each(m, fn)
	}
}

// Within selects the tiles within a specifid bounding box which is specified by
// north-west and south-east coordinates.
func (m *Grid[T]) Within(nw, se Point, fn func(Point, Tile[T])) {
	m.pagesWithin(nw, se, func(page *page[T]) {
		page.Each(m, func(p Point, v Tile[T]) {
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
func (m *Grid[T]) At(x, y int16) (Tile[T], bool) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		return m.pageAt(x/3, y/3).At(m, x, y), true
	}

	return Tile[T]{}, false
}

// WriteAt updates the entire tile value at a specific coordinate
func (m *Grid[T]) WriteAt(x, y int16, tile Value) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		m.pageAt(x/3, y/3).writeTile(m, uint8((y%3)*3+(x%3)), tile)
	}
}

// MaskAt atomically updates the bits of tile at a specific coordinate. The bits are
// specified by the mask. The bits that need to be updated should be flipped on in the mask.
func (m *Grid[T]) MaskAt(x, y int16, tile, mask Value) {
	m.MergeAt(x, y, func(value Value) Value {
		return (value &^ mask) | (tile & mask)
	})
}

// Merge atomically merges the tile by applying a merging function at a specific coordinate.
func (m *Grid[T]) MergeAt(x, y int16, merge func(Value) Value) {
	if x >= 0 && y >= 0 && x < m.Size.X && y < m.Size.Y {
		m.pageAt(x/3, y/3).mergeTile(m, uint8((y%3)*3+(x%3)), merge)
	}
}

// Neighbors iterates over the direct neighbouring tiles
func (m *Grid[T]) Neighbors(x, y int16, fn func(Point, Tile[T])) {

	// First we need to figure out which pages contain the neighboring tiles and
	// then load them. In the best-case we need to load only a single page. In
	// the worst-case: we need to load 3 pages.
	nX, nY := x/3, (y-1)/3 // North
	eX, eY := (x+1)/3, y/3 // East
	sX, sY := x/3, (y+1)/3 // South
	wX, wY := (x-1)/3, y/3 // West

	// Get the North
	if y > 0 {
		fn(At(x, y-1), m.pageAt(nX, nY).At(m, x, y-1))
	}

	// Get the East
	if eX < m.pageWidth {
		fn(At(x+1, y), m.pageAt(eX, eY).At(m, x+1, y))
	}

	// Get the South
	if sY < m.pageHeight {
		fn(At(x, y+1), m.pageAt(sX, sY).At(m, x, y+1))
	}

	// Get the West
	if x > 0 {
		fn(At(x-1, y), m.pageAt(wX, wY).At(m, x-1, y))
	}
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

// Value represents a packed tile information, it must fit on 4 bytes.
type Value = uint32

// ---------------------------------- Page ----------------------------------

// page represents a 3x3 tile page each page should neatly fit on a cache
// line and speed things up.
type page[T comparable] struct {
	mu    sync.Mutex  // State lock, 8 bytes
	state map[T]uint8 // State data, 8 bytes
	flags uint32      // Page flags, 4 bytes
	point Point       // Page X, Y coordinate, 4 bytes
	tiles [9]Value    // Page tiles, 36 bytes
}

// tileAt reads a tile at a page index
func (p *page[T]) tileAt(idx uint8) Value {
	return Value(atomic.LoadUint32((*uint32)(&p.tiles[idx])))
}

// IsObserved returns whether the tile is observed or not
func (p *page[T]) IsObserved() bool {
	return (atomic.LoadUint32(&p.flags))&1 != 0
}

// Bounds returns the bounding box for the tile page.
func (p *page[T]) Bounds() Rect {
	return Rect{p.point, At(p.point.X+3, p.point.Y+3)}
}

// At returns a cursor at a specific coordinate
func (p *page[T]) At(grid *Grid[T], x, y int16) Tile[T] {
	return Tile[T]{grid: grid, data: p, idx: uint8((y%3)*3 + (x % 3))}
}

// Each iterates over all of the tiles in the page.
func (p *page[T]) Each(grid *Grid[T], fn func(Point, Tile[T])) {
	x, y := p.point.X, p.point.Y
	fn(Point{x, y}, Tile[T]{grid: grid, data: p, idx: 0})         // NW
	fn(Point{x + 1, y}, Tile[T]{grid: grid, data: p, idx: 1})     // N
	fn(Point{x + 2, y}, Tile[T]{grid: grid, data: p, idx: 2})     // NE
	fn(Point{x, y + 1}, Tile[T]{grid: grid, data: p, idx: 3})     // W
	fn(Point{x + 1, y + 1}, Tile[T]{grid: grid, data: p, idx: 4}) // C
	fn(Point{x + 2, y + 1}, Tile[T]{grid: grid, data: p, idx: 5}) // E
	fn(Point{x, y + 2}, Tile[T]{grid: grid, data: p, idx: 6})     // SW
	fn(Point{x + 1, y + 2}, Tile[T]{grid: grid, data: p, idx: 7}) // S
	fn(Point{x + 2, y + 2}, Tile[T]{grid: grid, data: p, idx: 8}) // SE
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

// ---------------------------------- Mutations ----------------------------------

// writeTile stores the tile and return  whether tile is observed or not
func (p *page[T]) writeTile(grid *Grid[T], idx uint8, tile Value) {
	value := p.tileAt(idx)
	for !atomic.CompareAndSwapUint32(&p.tiles[idx], uint32(value), uint32(tile)) {
		value = p.tileAt(idx)
	}

	// If observed, notify the observers of the tile
	if p.IsObserved() {
		grid.observers.Notify(p.point, &Update[T]{
			Point: pointOf(p.point, idx),
			Old:   value,
			New:   tile,
		})
	}
}

// mergeTile atomically merges the tile bits given a function
func (p *page[T]) mergeTile(grid *Grid[T], idx uint8, fn func(Value) Value) Value {
	value := p.tileAt(idx)
	merge := fn(value)

	// Swap, if we're not able to re-merge again
	for !atomic.CompareAndSwapUint32(&p.tiles[idx], uint32(value), uint32(merge)) {
		value = p.tileAt(idx)
		merge = fn(value)
	}

	// If observed, notify the observers of the tile
	if p.IsObserved() {
		grid.observers.Notify(p.point, &Update[T]{
			Point: pointOf(p.point, idx),
			Old:   value,
			New:   merge,
		})
	}

	// Return the merged tile data
	return merge
}

// addObject adds object to the set
func (p *page[T]) addObject(grid *Grid[T], idx uint8, object T) {
	p.Lock()

	// Lazily initialize the map, as most pages might not have anything stored
	// in them (e.g. water or empty tile)
	if p.state == nil {
		p.state = make(map[T]uint8)
	}

	p.state[object] = uint8(idx)
	p.Unlock()

	// If observed, notify the observers of the tile
	if p.IsObserved() {
		value := p.tileAt(idx)
		grid.observers.Notify(p.point, &Update[T]{
			Point: pointOf(p.point, idx),
			Old:   value,
			New:   value,
			Add:   object,
		})
	}
}

// delObject removes the object from the set
func (p *page[T]) delObject(grid *Grid[T], idx uint8, object T) {
	p.Lock()
	if p.state != nil {
		delete(p.state, object)
	}
	p.Unlock()

	// If observed, notify the observers of the tile
	if p.IsObserved() {
		value := p.tileAt(idx)
		grid.observers.Notify(p.point, &Update[T]{
			Point: pointOf(p.point, idx),
			Old:   value,
			New:   value,
			Del:   object,
		})
	}
}

// ---------------------------------- Tile Cursor ----------------------------------

// Tile represents an iterator over all state objects at a particular location.
type Tile[T comparable] struct {
	grid *Grid[T] // grid pointer
	data *page[T] // page pointer
	idx  uint8    // tile index
}

// Count returns number of objects at the current tile.
func (c Tile[T]) Count() (count int) {
	c.data.Lock()
	defer c.data.Unlock()
	for _, idx := range c.data.state {
		if idx == uint8(c.idx) {
			count++
		}
	}
	return
}

// Value reads the tile information
func (c Tile[T]) Value() Value {
	return c.data.tileAt(c.idx)
}

// Range iterates over all of the objects in the set
func (c Tile[T]) Range(fn func(T) error) error {
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

// Observers iterates over all views observing this tile
func (c Tile[T]) Observers(fn func(view Observer[T])) {
	if !c.data.IsObserved() {
		return
	}

	c.grid.observers.Each(c.data.point, func(sub Observer[T]) {
		if view, ok := sub.(Observer[T]); ok {
			fn(view)
		}
	})
}

// Add adds object to the set
func (c Tile[T]) Add(v T) {
	c.data.addObject(c.grid, c.idx, v)
}

// Del removes the object from the set
func (c Tile[T]) Del(v T) {
	c.data.delObject(c.grid, c.idx, v)
}

// Write updates the entire tile value.
func (c Tile[T]) Write(tile Value) {
	c.data.writeTile(c.grid, c.idx, tile)
}

// Merge atomically merges the tile by applying a merging function.
func (c Tile[T]) Merge(merge func(Value) Value) Value {
	return c.data.mergeTile(c.grid, c.idx, merge)
}

// Mask updates the bits of tile. The bits are specified by the mask. The bits
// that need to be updated should be flipped on in the mask.
func (c Tile[T]) Mask(tile, mask Value) Value {
	return c.data.mergeTile(c.grid, c.idx, func(value Value) Value {
		return (value &^ mask) | (tile & mask)
	})
}

// pointOf returns the point given an index
func pointOf(page Point, idx uint8) Point {
	return Point{
		X: page.X + int16(idx)%3,
		Y: page.Y + int16(idx)/3,
	}
}
