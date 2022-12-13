// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"sync"
)

// Update represents a tile update notification.
type Update[T comparable] struct {
	Point       // The tile location
	Old   Value // Old tile value
	New   Value // New tile value
	Add   T     // An object was added to the tile
	Del   T     // An object was removed from the tile
}

// View represents a view which can monitor a collection of tiles.
type View[T comparable] struct {
	Grid  *Grid[T]       // The associated map
	Inbox chan Update[T] // The update inbox for the view
	rect  Rect           // The view box
}

// Resize resizes the viewport.
func (v *View[T]) Resize(box Rect, fn func(Point, Tile[T])) {
	owner := v.Grid // The parent map
	prev := v.rect  // Previous bounding box
	v.rect = box    // New bounding box

	// Unsubscribe from the pages which are not required anymore
	if prev.Min.X >= 0 || prev.Min.Y >= 0 || prev.Max.X >= 0 || prev.Max.Y >= 0 {
		owner.pagesWithin(prev.Min, prev.Max, func(page *page[T]) {
			if bounds := page.Bounds(); !bounds.Intersects(box) {
				if owner.observers.Unsubscribe(page.point, v) {
					page.SetObserved(false) // Mark the page as not being observed
				}
			}
		})
	}

	// Subscribe to every page which we have not previously subscribed
	owner.pagesWithin(box.Min, box.Max, func(page *page[T]) {
		if bounds := page.Bounds(); !bounds.Intersects(prev) {
			if owner.observers.Subscribe(page.point, v) {
				page.SetObserved(true) // Mark the page as being observed
			}
		}

		// Callback for each new tile in the view
		if fn != nil {
			page.Each(v.Grid, func(p Point, v Tile[T]) {
				if !prev.Contains(p) && box.Contains(p) {
					fn(p, v)
				}
			})
		}
	})
}

// MoveBy moves the viewport towards a particular direction.
func (v *View[T]) MoveBy(x, y int16, fn func(Point, Tile[T])) {
	v.Resize(Rect{
		Min: v.rect.Min.Add(At(x, y)),
		Max: v.rect.Max.Add(At(x, y)),
	}, fn)
}

// MoveAt moves the viewport to a specific coordinate.
func (v *View[T]) MoveAt(nw Point, fn func(Point, Tile[T])) {
	size := v.rect.Max.Subtract(v.rect.Min)
	v.Resize(Rect{
		Min: nw,
		Max: nw.Add(size),
	}, fn)
}

// Each iterates over all of the tiles in the view.
func (v *View[T]) Each(fn func(Point, Tile[T])) {
	v.Grid.Within(v.rect.Min, v.rect.Max, fn)
}

// At returns the tile at a specified position.
func (v *View[T]) At(x, y int16) (Tile[T], bool) {
	return v.Grid.At(x, y)
}

// WriteAt updates the entire tile at a specific coordinate.
func (v *View[T]) WriteAt(x, y int16, tile Value) {
	v.Grid.WriteAt(x, y, tile)
}

// MergeAt updates the bits of tile at a specific coordinate. The bits are specified
// by the mask. The bits that need to be updated should be flipped on in the mask.
func (v *View[T]) MergeAt(x, y int16, tile, mask Value) {
	v.Grid.MergeAt(x, y, tile, mask)
}

// Close closes the view and unsubscribes from everything.
func (v *View[T]) Close() error {
	v.Grid.pagesWithin(v.rect.Min, v.rect.Max, func(page *page[T]) {
		if v.Grid.observers.Unsubscribe(page.point, v) {
			page.SetObserved(false) // Mark the page as not being observed
		}
	})
	return nil
}

// onUpdate occurs when a tile has updated.
func (v *View[T]) onUpdate(ev *Update[T]) {
	if v.rect.Contains(ev.Point) {
		v.Inbox <- *ev // (copy)
	}
}

// -----------------------------------------------------------------------------

// observer represents a tile update observer.
type observer[T comparable] interface {
	onUpdate(*Update[T])
}

// Pubsub represents a publish/subscribe layer for observers.
type pubsub[T comparable] struct {
	m sync.Map
}

// Notify notifies listeners of an update that happened.
func (p *pubsub[T]) Notify(page Point, ev *Update[T]) {
	if v, ok := p.m.Load(page.Integer()); ok {
		v.(*observers[T]).Notify(ev)
	}
}

// Subscribe registers an event listener on a system
func (p *pubsub[T]) Subscribe(at Point, sub observer[T]) bool {
	if v, ok := p.m.Load(at.Integer()); ok {
		return v.(*observers[T]).Subscribe(sub)
	}

	// Slow path
	v, _ := p.m.LoadOrStore(at.Integer(), newObservers[T]())
	return v.(*observers[T]).Subscribe(sub)
}

// Unsubscribe deregisters an event listener from a system
func (p *pubsub[T]) Unsubscribe(at Point, sub observer[T]) bool {
	if v, ok := p.m.Load(at.Integer()); ok {
		return v.(*observers[T]).Unsubscribe(sub)
	}
	return false
}

// -----------------------------------------------------------------------------

// Observers represents a change notifier which notifies the subscribers when
// a specific tile is updated.
type observers[T comparable] struct {
	sync.Mutex
	subs []observer[T]
}

// newObservers creates a new instance of an change observer.
func newObservers[T comparable]() *observers[T] {
	return &observers[T]{
		subs: make([]observer[T], 0, 8),
	}
}

// Notify notifies listeners of an update that happened.
func (s *observers[T]) Notify(ev *Update[T]) {
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
func (s *observers[T]) Subscribe(sub observer[T]) bool {
	s.Lock()
	defer s.Unlock()
	s.subs = append(s.subs, sub)
	return len(s.subs) > 0 // At least one
}

// Unsubscribe deregisters an event listener from a system
func (s *observers[T]) Unsubscribe(sub observer[T]) bool {
	s.Lock()
	defer s.Unlock()

	clean := s.subs[:0]
	for _, o := range s.subs {
		if o != sub {
			clean = append(clean, o)
		}
	}
	s.subs = clean
	return len(s.subs) == 0
}
