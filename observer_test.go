// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvents(t *testing.T) {
	ev := newObserver()
	assert.NotNil(t, ev)

	// Subscriber which does nothing
	ev.Subscribe(func(_ Point, _ Tile) {})

	// Counting subscriber
	var count int
	cancel := ev.Subscribe(func(p Point, _ Tile) {
		count += int(p.X)
	})
	defer cancel()

	ev.Notify(At(1, 0), Tile{})
	ev.Notify(At(2, 0), Tile{})
	ev.Notify(At(3, 0), Tile{})

	for count < 6 {
		time.Sleep(1 * time.Millisecond)
	}

	assert.Equal(t, 6, count)
	cancel()

	ev.Notify(At(2, 0), Tile{})
	assert.Equal(t, 6, count)
}

func TestObserverNil(t *testing.T) {
	assert.NotPanics(t, func() {
		var ev *observer
		ev.Notify(At(1, 0), Tile{})
	})
}
