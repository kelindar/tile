// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"sync"
	"sync/atomic"
)

// Observer represents a tile update Observer.
type Observer[T comparable] interface {
	Viewport() Rect
	Resize(Rect, func(Point, Tile[T]))
	onUpdate(*Update[T])
}

// ValueAt represents a tile and its value.
type ValueAt struct {
	Point // The point of the tile
	Value // The value of the tile
}

// Update represents a tile update notification.
type Update[T comparable] struct {
	Old ValueAt // Old tile + value
	New ValueAt // New tile + value
	Add T       // An object was added to the tile
	Del T       // An object was removed from the tile
}

var _ Observer[string] = (*View[string, string])(nil)

// View represents a view which can monitor a collection of tiles. Type parameters
// S and T are the state and tile types respectively.
type View[S any, T comparable] struct {
	Grid  *Grid[T]       // The associated map
	Inbox chan Update[T] // The update inbox for the view
	State S              // The state of the view
	rect  atomic.Uint64  // The view box
}

// NewView creates a new view for a map with a given state. State can be anything
// that is passed to the view and can be used to store additional information.
func NewView[S any, T comparable](m *Grid[T], state S) *View[S, T] {
	v := &View[S, T]{
		Grid:  m,
		Inbox: make(chan Update[T], 32),
		State: state,
	}
	v.rect.Store(NewRect(-1, -1, -1, -1).pack())
	return v
}

// Viewport returns the current viewport of the view.
func (v *View[S, T]) Viewport() Rect {
	return unpackRect(v.rect.Load())
}

// Resize resizes the viewport and notifies the observers of the changes.
func (v *View[S, T]) Resize(view Rect, fn func(Point, Tile[T])) {
	grid := v.Grid
	prev := unpackRect(v.rect.Swap(view.pack()))

	for _, diff := range view.Difference(prev) {
		if diff.IsZero() {
			continue // Skip zero-value rectangles
		}

		grid.pagesWithin(diff.Min, diff.Max, func(page *page[T]) {
			r := page.Bounds()
			switch {

			// Page is now in view
			case view.Intersects(r) && !prev.Intersects(r):
				if grid.observers.Subscribe(page.point, v) {
					page.SetObserved(true) // Mark the page as being observed
				}

			// Page is no longer in view
			case !view.Intersects(r) && prev.Intersects(r):
				if grid.observers.Unsubscribe(page.point, v) {
					page.SetObserved(false) // Mark the page as not being observed
				}
			}

			// Callback for each new tile in the view
			if fn != nil {
				page.Each(v.Grid, func(p Point, tile Tile[T]) {
					if view.Contains(p) && !prev.Contains(p) {
						fn(p, tile)
					}
				})
			}
		})
	}
}

// MoveTo moves the viewport towards a particular direction.
func (v *View[S, T]) MoveTo(angle Direction, distance int16, fn func(Point, Tile[T])) {
	p := angle.Vector(distance)
	r := v.Viewport()
	v.Resize(Rect{
		Min: r.Min.Add(p),
		Max: r.Max.Add(p),
	}, fn)
}

// MoveBy moves the viewport towards a particular direction.
func (v *View[S, T]) MoveBy(x, y int16, fn func(Point, Tile[T])) {
	r := v.Viewport()
	v.Resize(Rect{
		Min: r.Min.Add(At(x, y)),
		Max: r.Max.Add(At(x, y)),
	}, fn)
}

// MoveAt moves the viewport to a specific coordinate.
func (v *View[S, T]) MoveAt(nw Point, fn func(Point, Tile[T])) {
	r := v.Viewport()
	size := r.Max.Subtract(r.Min)
	v.Resize(Rect{
		Min: nw,
		Max: nw.Add(size),
	}, fn)
}

// Each iterates over all of the tiles in the view.
func (v *View[S, T]) Each(fn func(Point, Tile[T])) {
	r := v.Viewport()
	v.Grid.Within(r.Min, r.Max, fn)
}

// At returns the tile at a specified position.
func (v *View[S, T]) At(x, y int16) (Tile[T], bool) {
	return v.Grid.At(x, y)
}

// WriteAt updates the entire tile at a specific coordinate.
func (v *View[S, T]) WriteAt(x, y int16, tile Value) {
	v.Grid.WriteAt(x, y, tile)
}

// MergeAt updates the bits of tile at a specific coordinate. The bits are specified
// by the mask. The bits that need to be updated should be flipped on in the mask.
func (v *View[S, T]) MergeAt(x, y int16, tile, mask Value) {
	v.Grid.MaskAt(x, y, tile, mask)
}

// Close closes the view and unsubscribes from everything.
func (v *View[S, T]) Close() error {
	r := v.Viewport()
	v.Grid.pagesWithin(r.Min, r.Max, func(page *page[T]) {
		if v.Grid.observers.Unsubscribe(page.point, v) {
			page.SetObserved(false) // Mark the page as not being observed
		}
	})
	return nil
}

// onUpdate occurs when a tile has updated.
func (v *View[S, T]) onUpdate(ev *Update[T]) {
	v.Inbox <- *ev // (copy)
}

// -----------------------------------------------------------------------------

// Pubsub represents a publish/subscribe layer for observers.
type pubsub[T comparable] struct {
	m   sync.Map  // Concurrent map of observers
	tmp sync.Pool // Temporary observer sets for notifications
}

// Subscribe registers an event listener on a system
func (p *pubsub[T]) Subscribe(page Point, sub Observer[T]) bool {
	if v, ok := p.m.Load(page.Integer()); ok {
		return v.(*observers[T]).Subscribe(sub)
	}

	// Slow path
	v, _ := p.m.LoadOrStore(page.Integer(), newObservers[T]())
	return v.(*observers[T]).Subscribe(sub)
}

// Unsubscribe deregisters an event listener from a system
func (p *pubsub[T]) Unsubscribe(page Point, sub Observer[T]) bool {
	if v, ok := p.m.Load(page.Integer()); ok {
		return v.(*observers[T]).Unsubscribe(sub)
	}
	return false
}

// Notify notifies listeners of an update that happened.
func (p *pubsub[T]) Notify1(ev *Update[T], page, at Point) {
	p.Each1(func(sub Observer[T]) {
		sub.onUpdate(ev)
	}, page, at)
}

// Notify notifies listeners of an update that happened.
func (p *pubsub[T]) Notify2(ev *Update[T], pages, locs [2]Point) {
	p.Each2(func(sub Observer[T]) {
		sub.onUpdate(ev)
	}, pages, locs)
}

// Each iterates over each observer in a page
func (p *pubsub[T]) Each1(fn func(sub Observer[T]), page, at Point) {
	if v, ok := p.m.Load(page.Integer()); ok {
		v.(*observers[T]).Each(func(sub Observer[T]) {
			if sub.Viewport().Contains(at) {
				fn(sub)
			}
		})
	}
}

// Each2 iterates over each observer in a page
func (p *pubsub[T]) Each2(fn func(sub Observer[T]), pages, locs [2]Point) {
	targets := p.tmp.Get().(map[Observer[T]]struct{})
	clear(targets)
	defer p.tmp.Put(targets)

	// Collect all observers from all pages
	for _, page := range pages {
		if v, ok := p.m.Load(page.Integer()); ok {
			v.(*observers[T]).Each(func(sub Observer[T]) {
				targets[sub] = struct{}{}
			})
		}
	}

	// Invoke the callback for each observer, once
	for sub := range targets {
		if sub.Viewport().Contains(locs[0]) || sub.Viewport().Contains(locs[1]) {
			fn(sub)
		}
	}
}

// -----------------------------------------------------------------------------

// Observers represents a change notifier which notifies the subscribers when
// a specific tile is updated.
type observers[T comparable] struct {
	sync.Mutex
	subs []Observer[T]
}

// newObservers creates a new instance of an change observer.
func newObservers[T comparable]() *observers[T] {
	return &observers[T]{
		subs: make([]Observer[T], 0, 8),
	}
}

// Each iterates over each observer
func (s *observers[T]) Each(fn func(sub Observer[T])) {
	if s == nil {
		return
	}

	s.Lock()
	defer s.Unlock()
	for _, sub := range s.subs {
		fn(sub)
	}
}

// Subscribe registers an event listener on a system
func (s *observers[T]) Subscribe(sub Observer[T]) bool {
	s.Lock()
	defer s.Unlock()
	s.subs = append(s.subs, sub)
	return len(s.subs) > 0 // At least one
}

// Unsubscribe deregisters an event listener from a system
func (s *observers[T]) Unsubscribe(sub Observer[T]) bool {
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
