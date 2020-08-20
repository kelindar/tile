// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"sync"
)

// Update represents a tile update notification.
type Update struct {
	Point // The tile location
	Tile  // The tile data
}

// View represents a view which can monitor a collection of tiles.
type View struct {
	Map   *Map        // The associated map
	Inbox chan Update // The update inbox for the view
	rect  Rect        // The view box
}

// Resize resizes the viewport.
func (v *View) Resize(box Rect) {
	prev := v.rect // Previous bounding box
	v.rect = box   // New bounding box

	// Unsubscribe from the pages which are not required anymore
	v.Map.pagesWithin(prev.Min, prev.Max, func(x, y int16, page *page) {
		if bounds := NewRect(x, y, x+3, y+3); !bounds.Intersects(box) {
			page.Unsubscribe(v)
		}
	})

	// Subscribe to every page which we have not previously subscribed
	v.Map.pagesWithin(box.Min, box.Max, func(x, y int16, page *page) {
		if bounds := NewRect(x, y, x+3, y+3); !bounds.Intersects(prev) {
			page.Subscribe(v)
		}

		// Notify of the new tiles by invoking the update
		page.Each(x, y, func(p Point, tile Tile) {
			if !prev.Contains(p) {
				v.Inbox <- Update{Point: p, Tile: tile}
			}
		})
	})
}

// MoveBy moves the viewport towards a particular direction.
func (v *View) MoveBy(x, y int16) {
	v.Resize(Rect{
		Min: v.rect.Min.Add(At(x, y)),
		Max: v.rect.Max.Add(At(x, y)),
	})
}

// MoveAt moves the viewport to a specific coordinate.
func (v *View) MoveAt(nw Point) {
	size := v.rect.Max.Subtract(v.rect.Min)
	v.Resize(Rect{
		Min: nw,
		Max: nw.Add(size),
	})
}

// Each iterates over all of the tiles in the view.
func (v *View) Each(fn Iterator) {
	v.Map.Within(v.rect.Min, v.rect.Max, fn)
}

// At returns the tile at a specified position.
func (v *View) At(x, y int16) (Tile, bool) {
	return v.Map.At(x, y)
}

// UpdateAt updates the tile at a specific coordinate.
func (v *View) UpdateAt(x, y int16, tile Tile) {
	v.Map.UpdateAt(x, y, tile)
}

// Neighbors iterates over the direct neighbouring tiles.
func (v *View) Neighbors(x, y int16, fn Iterator) {
	v.Map.Neighbors(x, y, fn)
}

// onUpdate occurs when a tile has updated.
func (v *View) onUpdate(ev *Update) {
	if !v.rect.Contains(ev.Point) {
		return // Point is outside of the view
	}

	// Push the update to the buffered channel
	v.Inbox <- *ev // (copy)
}

// -----------------------------------------------------------------------------

// observer represents a tile update observer.
type observer interface {
	onUpdate(*Update)
}

// Signal represents a change notifier which notifies the subscribers when
// a specific tile is updated.
type signal struct {
	sync.RWMutex
	subs []observer
}

// newNotifier creates a new instance of an change observer.
func newSignal() *signal {
	return &signal{
		subs: make([]observer, 0, 8),
	}
}

// Notify notifies listeners of an update that happened.
func (s *signal) Notify(point Point, tile Tile) {
	if s == nil {
		return
	}

	s.RLock()
	defer s.RUnlock()
	event := &Update{Point: point, Tile: tile}
	for _, sub := range s.subs {
		sub.onUpdate(event)
	}
}

// Subscribe registers an event listener on a system
func (s *signal) Subscribe(sub observer) {
	s.Lock()
	defer s.Unlock()
	s.subs = append(s.subs, sub)
}

// Unsubscribe deregisters an event listener from a system
func (s *signal) Unsubscribe(sub observer) {
	if s == nil {
		return
	}

	s.Lock()
	defer s.Unlock()

	clean := make([]observer, 0, len(s.subs))
	for _, o := range s.subs {
		if o != sub {
			clean = append(clean, o)
		}
	}
	s.subs = clean
}
