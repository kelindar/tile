// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"context"
	"sync"
)

// Observer represents a change observer which notifies the subscribers when
// a specific tile is updated.
type observer struct {
	sync.RWMutex
	subs []*subscriber
}

// Update represents an update notification
type update struct {
	Point // The point of the update
	Tile  // The tile which was updated
}

// newObserver creates a new instance of an change observer.
func newObserver() *observer {
	return &observer{
		subs: make([]*subscriber, 0, 8),
	}
}

// Notify notifies listeners of an update that happened.
func (o *observer) Notify(point Point, tile Tile) {
	if o == nil {
		return
	}

	o.RLock()
	defer o.RUnlock()
	for _, h := range o.subs {
		h.buffer <- update{
			Point: point,
			Tile:  tile,
		}
	}
}

// Subscribe registers an event listener on a system
func (o *observer) Subscribe(callback rangeFn) context.CancelFunc {
	o.Lock()
	defer o.Unlock()

	// Create the handler
	ctx, cancel := context.WithCancel(context.Background())
	subscriber := &subscriber{
		buffer:   make(chan update, 1),
		callback: &callback,
		cancel:   cancel,
	}

	// Add the listener
	o.subs = append(o.subs, subscriber)
	go subscriber.listen(ctx)
	return o.unsubscribe(&callback)
}

// unsubscribe deregisters an event listener from a system
func (o *observer) unsubscribe(callback *rangeFn) context.CancelFunc {
	return func() {
		o.Lock()
		defer o.Unlock()

		clean := make([]*subscriber, 0, len(o.subs))
		for _, h := range o.subs {
			if h.callback != callback { // Compare address
				clean = append(clean, h)
			} else {
				h.cancel()
			}
		}
	}
}

// -----------------------------------------------------------------------------

type subscriber struct {
	buffer   chan update
	callback *rangeFn
	cancel   context.CancelFunc
}

// Listen listens on the buffer and invokes the callback
func (s *subscriber) listen(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case value := <-s.buffer:
			(*s.callback)(value.Point, value.Tile)
		}
	}
}
