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
func (v *View) Resize(box Rect, fn Iterator) {
	owner := v.Map // The parent map
	prev := v.rect // Previous bounding box
	v.rect = box   // New bounding box

	// Unsubscribe from the pages which are not required anymore
	if prev.Min.X >= 0 || prev.Min.Y >= 0 || prev.Max.X >= 0 || prev.Max.Y >= 0 {
		owner.pagesWithin(prev.Min, prev.Max, func(page *page) {
			if bounds := page.Bounds(); !bounds.Intersects(box) {
				if owner.observers.Unsubscribe(page.point, v) {
					page.SetObserved(false) // Mark the page as not being observed
				}
			}
		})
	}

	// Subscribe to every page which we have not previously subscribed
	owner.pagesWithin(box.Min, box.Max, func(page *page) {
		if bounds := page.Bounds(); !bounds.Intersects(prev) {
			if owner.observers.Subscribe(page.point, v) {
				page.SetObserved(true) // Mark the page as being observed
			}
		}

		// Callback for each new tile in the view
		if fn != nil {
			page.Each(func(p Point, tile Tile) {
				if !prev.Contains(p) && box.Contains(p) {
					fn(p, tile)
				}
			})
		}
	})
}

// MoveBy moves the viewport towards a particular direction.
func (v *View) MoveBy(x, y int16, fn Iterator) {
	v.Resize(Rect{
		Min: v.rect.Min.Add(At(x, y)),
		Max: v.rect.Max.Add(At(x, y)),
	}, fn)
}

// MoveAt moves the viewport to a specific coordinate.
func (v *View) MoveAt(nw Point, fn Iterator) {
	size := v.rect.Max.Subtract(v.rect.Min)
	v.Resize(Rect{
		Min: nw,
		Max: nw.Add(size),
	}, fn)
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

// Pubsub represents a publish/subscribe layer for observers.
type pubsub struct {
	m sync.Map
}

// Notify notifies listeners of an update that happened.
func (p *pubsub) Notify(page, point Point, tile Tile) {
	if v, ok := p.m.Load(page.Integer()); ok {
		v.(*observers).Notify(&Update{
			Point: point,
			Tile:  tile,
		})
	}
}

// Subscribe registers an event listener on a system
func (p *pubsub) Subscribe(at Point, sub observer) bool {
	if v, ok := p.m.Load(at.Integer()); ok {
		return v.(*observers).Subscribe(sub)
	}

	// Slow path
	v, _ := p.m.LoadOrStore(at.Integer(), newObservers())
	return v.(*observers).Subscribe(sub)
}

// Unsubscribe deregisters an event listener from a system
func (p *pubsub) Unsubscribe(at Point, sub observer) bool {
	if v, ok := p.m.Load(at.Integer()); ok {
		return v.(*observers).Unsubscribe(sub)
	}
	return false
}

// -----------------------------------------------------------------------------

// Observers represents a change notifier which notifies the subscribers when
// a specific tile is updated.
type observers struct {
	sync.Mutex
	subs []observer
}

// newObservers creates a new instance of an change observer.
func newObservers() *observers {
	return &observers{
		subs: make([]observer, 0, 8),
	}
}

// Notify notifies listeners of an update that happened.
func (s *observers) Notify(ev *Update) {
	if s == nil {
		return
	}

	s.Lock()
	subs := s.subs
	s.Unlock()

	// Update every subscriber
	for _, sub := range subs {
		sub.onUpdate(ev)
	}
}

// Subscribe registers an event listener on a system
func (s *observers) Subscribe(sub observer) bool {
	s.Lock()
	defer s.Unlock()
	s.subs = append(s.subs, sub)
	return len(s.subs) > 0 // At least one
}

// Unsubscribe deregisters an event listener from a system
func (s *observers) Unsubscribe(sub observer) bool {
	s.Lock()
	defer s.Unlock()

	clean := make([]observer, 0, len(s.subs))
	for _, o := range s.subs {
		if o != sub {
			clean = append(clean, o)
		}
	}
	s.subs = clean
	return len(s.subs) == 0
}
