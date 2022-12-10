// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

/*
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkView/write-8         	 7208314	       174.0 ns/op	       8 B/op	       1 allocs/op
BenchmarkView/move-8          	    9231	    120567 ns/op	       0 B/op	       0 allocs/op
BenchmarkView/notify-8        	 7274684	       170.2 ns/op	       8 B/op	       1 allocs/op
*/
func BenchmarkView(b *testing.B) {
	m := mapFrom("300x300.png")
	v := m.View(NewRect(100, 0, 199, 99), nil)
	go func() {
		for range v.Inbox {
		}
	}()

	b.Run("write", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			v.WriteAt(152, 52, Tile(0))
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

	b.Run("notify", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.NotifyAt(150, 50)
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
	cursor, _ := v.At(5, 5)
	before := cursor.Tile()
	v.WriteAt(5, 5, Tile(55))
	update := <-v.Inbox
	assert.Equal(t, At(5, 5), update.Point)
	assert.NotEqual(t, before, update.Tile)

	// Merge a tile in view, but with zero mask (won't do anything)
	cursor, _ = v.At(5, 5)
	before = cursor.Tile()
	v.MergeAt(5, 5, Tile(66), Tile(0)) // zero mask
	update = <-v.Inbox
	assert.Equal(t, At(5, 5), update.Point)
	assert.Equal(t, before, update.Tile)

	// Close the view
	assert.NoError(t, v.Close())
	v.WriteAt(5, 5, Tile(66))
	assert.Equal(t, 0, len(v.Inbox))
}

type counter int

func (c *counter) count(p Point, tile Cursor) {
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

func TestNotifyAt(t *testing.T) {
	m := mapFrom("300x300.png")

	// Create a new view
	c := counter(0)
	v := m.View(NewRect(0, 0, 99, 99), c.count)
	assert.NotNil(t, v)
	assert.Equal(t, 10000, int(c))

	m.NotifyAt(1, 1)
	update := <-v.Inbox
	assert.Equal(t, int16(1), update.X)
	assert.Equal(t, int16(1), update.Y)
}

type fakeView func(*Update)

func (f fakeView) onUpdate(e *Update) {
	f(e)
}
