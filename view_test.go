// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// BenchmarkView/update-8         	10256436	       115 ns/op	      16 B/op	       1 allocs/op
// BenchmarkView/move-8           	    7485	    153640 ns/op	       0 B/op	       0 allocs/op
func BenchmarkView(b *testing.B) {
	m := mapFrom("300x300.png")
	v := m.View(NewRect(100, 0, 199, 99), nil)

	b.Run("update", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			v.UpdateAt(152, 52, Tile{})
			<-v.Inbox
		}
	})

	b.Run("move", func(b *testing.B) {
		locs := []Point{
			At(100, 0),
			At(200, 100),
		}

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			v.MoveAt(locs[n%2], nil)
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
	tile[0] = 55
	v.UpdateAt(5, 5, tile)
	update := <-v.Inbox
	assert.Equal(t, At(5, 5), update.Point)
	assert.Equal(t, tile, update.Tile)

	// Close the view
	assert.NoError(t, v.Close())
	v.UpdateAt(5, 5, tile)
	assert.Equal(t, 0, len(v.Inbox))
}

type counter int

func (c *counter) count(p Point, tile Tile) {
	*c++
}

func TestObservers(t *testing.T) {
	ev := newObservers()
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

	ev.Notify(&Update{Point: At(1, 0)})
	ev.Notify(&Update{Point: At(2, 0)})
	ev.Notify(&Update{Point: At(3, 0)})

	for count < 6 {
		time.Sleep(1 * time.Millisecond)
	}

	assert.Equal(t, 6, count)
	ev.Unsubscribe(&sub2)

	ev.Notify(&Update{Point: At(2, 0)})
	assert.Equal(t, 6, count)
}

func TestObserversNil(t *testing.T) {
	assert.NotPanics(t, func() {
		var ev *observers
		ev.Notify(&Update{Point: At(1, 0)})
	})
}

type fakeView func(*Update)

func (f fakeView) onUpdate(e *Update) {
	f(e)
}
