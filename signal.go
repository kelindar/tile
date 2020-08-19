// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"sync"
)

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
