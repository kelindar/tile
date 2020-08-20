// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// BenchmarkView/update-8         	15583524	        77.1 ns/op	      16 B/op	       1 allocs/op
func BenchmarkView(b *testing.B) {
	m := mapFrom("300x300.png")
	v := m.View(NewRect(100, 0, 199, 99), nil)
	b.Run("update", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			v.UpdateAt(150, 50, Tile{})
			<-v.Inbox
		}
	})
}

func TestView(t *testing.T) {
	m := mapFrom("300x300.png")

	// Create a new view
	c := counter(0)
	v := m.View(NewRect(100, 0, 199, 99), c.count)
	assert.NotNil(t, v)
	assert.Equal(t, 10000, int(c))

	// Resize to 10x10
	c = counter(0)
	v.Resize(NewRect(0, 0, 9, 9), c.count)
	assert.Equal(t, 100, int(c))

	// Move down-right
	c = counter(0)
	v.MoveBy(2, 2, c.count)
	assert.Equal(t, 36, int(c))

	// Move at location
	c = counter(0)
	v.MoveAt(At(4, 4), c.count)
	assert.Equal(t, 36, int(c))

	// Each
	c = counter(0)
	v.Each(c.count)
	assert.Equal(t, 100, int(c))

	// Update a tile in view
	tile, _ := v.At(5, 5)
	tile.Data[0] = 55
	v.UpdateAt(5, 5, tile)
	update := <-v.Inbox
	assert.Equal(t, At(5, 5), update.Point)
	assert.Equal(t, tile, update.Tile)
}

type counter int

func (c *counter) count(p Point, tile Tile) {
	*c++
}

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
