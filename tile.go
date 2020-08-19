// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

type rangeFn = func(Point, Tile)

// Map represents a 2D tile map. Internally, a map is composed of 3x3 pages.
type Map struct {
	Width      uint16   // The width of the map
	Height     uint16   // The height of the map
	pageWidth  uint16   // The max page width
	pageHeight uint16   // The max page height
	pages      [][]page // The pages of the map
}

// NewMap returns a new map of the specified size. The width and height must be both
// multiples of 3.
func NewMap(width, height uint16) *Map {
	width, height = width/3, height/3
	pages := make([][]page, height)
	for x := uint16(0); x < width; x++ {
		pages[x] = make([]page, height)
	}

	return &Map{
		pages:      pages,
		Width:      width * 3,
		Height:     height * 3,
		pageWidth:  width,
		pageHeight: height,
	}
}

// Each iterates over all of the tiles in the map.
func (m *Map) Each(fn rangeFn) {
	for y := uint16(0); y < m.pageHeight; y++ {
		for x := uint16(0); x < m.pageWidth; x++ {
			m.pages[x][y].Each(x*3, uint16(y)*3, fn)
		}
	}
}

// At returns the tile at a specified position
func (m *Map) At(x, y uint16) (Tile, bool) {
	if x < 0 || y < 0 || x >= m.Height || y >= m.Width {
		return Tile{}, false
	}

	return m.pages[x/3][y/3].Get(x, y), true
}

// UpdateAt updates the tile at a specific coordinate
func (m *Map) UpdateAt(x, y uint16, tile Tile) {
	if x < 0 || y < 0 || x >= m.Height || y >= m.Width {
		return
	}

	m.pages[x/3][y/3].Set(x, y, tile)
}

// Neighbors iterates over the direct neighbouring tiles
func (m *Map) Neighbors(x, y uint16, fn rangeFn) {

	// First we need to figure out which pages contain the neighboring tiles and
	// then load them. In the best-case we need to load only a single page. In
	// the worst-case: we need to load 3 pages.
	nX, nY := x/3, (y-1)/3 // North
	eX, eY := (x+1)/3, y/3 // East
	sX, sY := x/3, (y+1)/3 // South
	wX, wY := (x-1)/3, y/3 // West

	// Get the North
	if y > 0 {
		if tile := m.pages[nX][nY].Get(x, y-1); !tile.IsBlocked() {
			fn(At(x, y-1), tile)
		}
	}

	// Get the East
	if eX < m.pageWidth {
		if tile := m.pages[eX][eY].Get(x+1, y); !tile.IsBlocked() {
			fn(At(x+1, y), tile)
		}
	}

	// Get the South
	if sY < m.pageHeight {
		if tile := m.pages[sX][sY].Get(x, y+1); !tile.IsBlocked() {
			fn(At(x, y+1), tile)
		}
	}

	// Get the West
	if x > 0 {
		if tile := m.pages[wX][wY].Get(x-1, y); !tile.IsBlocked() {
			fn(At(x-1, y), tile)
		}
	}
}

func (m *Map) Around(x, y, distance uint16, fn rangeFn) {
	// BFS
	// https://www.redblobgames.com/pathfinding/a-star/introduction.html
}

// -----------------------------------------------------------------------------

// Tile represents a packed tile information, it must fit on 6 bytes.
type Tile struct {
	Flags         // The flags of the tile
	Data  [5]byte // The data of the tile
}

// Flags represents a tile flags, used for pathfinding and such.
type Flags byte

// IsBlocked returns whether the tile is blocked or not
func (f Flags) IsBlocked() bool {
	return f&Blocked != 0
}

// Various tile flags
const (
	Blocked   Flags = 1 << iota // Whether the tile is impassable or not
	Container                   // Whether the tile contains a container
	Mobile                      // Whether the tile contains a mobile (player or NPC)
	// Door ?
	// Roof ?
	// Status ?
	// Object ?
)

//func Set(b, flag Flags) Flags    { return b | flag }
//func Clear(b, flag Flags) Flags  { return b &^ flag }
//func Toggle(b, flag Flags) Flags { return b ^ flag }
//func Has(b, flag Flags) bool     { return b&flag != 0 }

// -----------------------------------------------------------------------------

// page represents a 3x3 tile page each page should neatly fit on a cache
// line and speed things up.
type page struct {
	Event *signal // Page signals, 8 bytes
	Tiles [9]Tile // Page tiles, 54 bytes
	Flags uint16  // Page flags, 2 bytes
}

// Get gets a tile at a specific coordinate.
func (p *page) Get(x, y uint16) Tile {
	return p.Tiles[(y%3)*3+(x%3)]
}

// Set updates the tile at a specific coordinate
func (p *page) Set(x, y uint16, tile Tile) {
	p.Tiles[(y%3)*3+(x%3)] = tile
}

// UpdateEach iterates over all of the tiles in the page.
func (p *page) Each(x, y uint16, fn rangeFn) {
	fn(At(x, y), p.Tiles[0])     // NW
	fn(At(x+1, y), p.Tiles[1])   // N
	fn(At(x+2, y), p.Tiles[2])   // NE
	fn(At(x, y+1), p.Tiles[3])   // W
	fn(At(x+1, y+1), p.Tiles[4]) // C
	fn(At(x+2, y+1), p.Tiles[5]) // E
	fn(At(x, y+2), p.Tiles[6])   // SW
	fn(At(x+1, y+2), p.Tiles[7]) // S
	fn(At(x+2, y+2), p.Tiles[8]) // SE
}
