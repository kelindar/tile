// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"sync"
)

// View represents a view which can monitor a collection of tiles.
type View struct {
	world *Map // The associated map
	rect  Rect // The view box
}

// Resize resizes the viewport.
func (v *View) Resize(box Rect) {
	v.rect = box
}

// MoveBy moves the viewport towards a particular direction.
func (v *View) MoveBy(x, y int16) {
	v.rect.Min = v.rect.Min.Add(At(x, y))
	v.rect.Max = v.rect.Max.Add(At(x, y))
}

// MoveAt moves the viewport to a specific coordinate.
func (v *View) MoveAt(nw Point) {
	size := v.rect.Max.Subtract(v.rect.Min)
	v.rect.Min = nw
	v.rect.Max = nw.Add(size)
}

// Each iterates over all of the tiles in the view.
func (v *View) Each(fn Iterator) {
	v.world.Within(v.rect.Min, v.rect.Max, fn)
}

// At returns the tile at a specified position.
func (v *View) At(x, y int16) (Tile, bool) {
	return v.world.At(x, y)
}

// UpdateAt updates the tile at a specific coordinate.
func (v *View) UpdateAt(x, y int16, tile Tile) {
	v.world.UpdateAt(x, y, tile)
}

// Neighbors iterates over the direct neighbouring tiles.
func (v *View) Neighbors(x, y int16, fn Iterator) {
	v.world.Neighbors(x, y, fn)
}

// OnTileUpdate occurs when a tile has updated.
func (v *View) onTileUpdate(at Point, tile Tile) {

}

// -----------------------------------------------------------------------------

// Observer represents a tile change observer.
type Observer interface {
	OnTileUpdate(Point, Tile)
}

// Signal represents a change notifier which notifies the subscribers when
// a specific tile is updated.
type signal struct {
	sync.RWMutex
	subs []Observer
}

// newNotifier creates a new instance of an change observer.
func newSignal() *signal {
	return &signal{
		subs: make([]Observer, 0, 8),
	}
}

// Notify notifies listeners of an update that happened.
func (s *signal) Notify(point Point, tile Tile) {
	if s == nil {
		return
	}

	s.RLock()
	defer s.RUnlock()
	for _, sub := range s.subs {
		sub.OnTileUpdate(point, tile)
	}
}

// Subscribe registers an event listener on a system
func (s *signal) Subscribe(sub Observer) {
	s.Lock()
	defer s.Unlock()
	s.subs = append(s.subs, sub)
}

// Unsubscribe deregisters an event listener from a system
func (s *signal) Unsubscribe(sub Observer) {
	s.Lock()
	defer s.Unlock()

	clean := make([]Observer, 0, len(s.subs))
	for _, o := range s.subs {
		if o != sub {
			clean = append(clean, o)
		}
	}
	s.subs = clean
}
