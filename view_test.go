// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSignal(t *testing.T) {
	ev := newSignal()
	assert.NotNil(t, ev)

	// Subscriber which does nothing
	var sub1 fakeView = func(e *Update) {}
	ev.Subscribe(&sub1)

	// Counting subscriber
	var count int
	var sub2 fakeView = func(e *Update) {
		count += int(e.X)
	}
	ev.Subscribe(&sub2)

	ev.Notify(At(1, 0), Tile{})
	ev.Notify(At(2, 0), Tile{})
	ev.Notify(At(3, 0), Tile{})

	for count < 6 {
		time.Sleep(1 * time.Millisecond)
	}

	assert.Equal(t, 6, count)
	ev.Unsubscribe(&sub2)

	ev.Notify(At(2, 0), Tile{})
	assert.Equal(t, 6, count)
}

func TestSignalNil(t *testing.T) {
	assert.NotPanics(t, func() {
		var ev *signal
		ev.Notify(At(1, 0), Tile{})
	})
}

type fakeView func(*Update)

func (f fakeView) onUpdate(e *Update) {
	f(e)
}
