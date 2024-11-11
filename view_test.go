// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkView/write-24         	 9540012	       125.0 ns/op	      48 B/op	       1 allocs/op
BenchmarkView/move-24          	   16141	     74408 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkView(b *testing.B) {
	m := mapFrom("300x300.png")
	v := NewView[string, string](m, "view 1")
	v.Resize(NewRect(100, 0, 200, 100), nil)

	go func() {
		for range v.Inbox {
		}
	}()

	b.Run("write", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			v.WriteAt(152, 52, Value(0))
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
	v := NewView[string, string](m, "view 1")
	v.Resize(NewRect(100, 0, 200, 100), c.count)
	assert.NotNil(t, v)
	assert.Equal(t, 10000, int(c))

	// Resize to 10x10
	c = counter(0)
	v.Resize(NewRect(0, 0, 10, 10), c.count)
	assert.Equal(t, 100, int(c))

	// Move down-right
	c = counter(0)
	v.MoveBy(2, 2, c.count)
	assert.Equal(t, 48, int(c))

	// Move at location
	c = counter(0)
	v.MoveAt(At(4, 4), c.count)
	assert.Equal(t, 48, int(c))

	// Each
	c = counter(0)
	v.Each(c.count)
	assert.Equal(t, 100, int(c))

	// Update a tile in view
	cursor, _ := v.At(5, 5)
	before := cursor.Value()
	v.WriteAt(5, 5, Value(55))
	update := <-v.Inbox
	assert.Equal(t, At(5, 5), update.New.Point)
	assert.NotEqual(t, before, update.New)

	// Merge a tile in view, but with zero mask (won't do anything)
	cursor, _ = v.At(5, 5)
	before = cursor.Value()
	v.MergeAt(5, 5, Value(66), Value(0)) // zero mask
	update = <-v.Inbox
	assert.Equal(t, At(5, 5), update.New.Point)
	assert.Equal(t, before, update.New.Value)

	// Close the view
	assert.NoError(t, v.Close())
	v.WriteAt(5, 5, Value(66))
	assert.Equal(t, 0, len(v.Inbox))
}

/*
func TestObservers(t *testing.T) {
	ev := newObservers[uint32]()
	assert.NotNil(t, ev)

	// Subscriber which does nothing
	var sub1 fakeView[uint32] = func(e *Update[uint32]) {}
	ev.Subscribe(&sub1)

	// Counting subscriber
	var count int
	var sub2 fakeView[uint32] = func(e *Update[uint32]) {
		count += int(e.X)
	}
	ev.Subscribe(&sub2)

	ev.Notify(&Update[uint32]{Point: At(1, 0)})
	ev.Notify(&Update[uint32]{Point: At(2, 0)})
	ev.Notify(&Update[uint32]{Point: At(3, 0)})

	for count < 6 {
		time.Sleep(1 * time.Millisecond)
	}

	assert.Equal(t, 6, count)
	ev.Unsubscribe(&sub2)

	ev.Notify(&Update[uint32]{Point: At(2, 0)})
	assert.Equal(t, 6, count)
}
*/

func TestUpdates_Simple(t *testing.T) {
	m := mapFrom("300x300.png")
	c := counter(0)
	v := NewView(m, "view 1")
	v.Resize(NewRect(0, 0, 10, 10), c.count)

	assert.NotNil(t, v)
	assert.Equal(t, 100, int(c))

	// Update a tile in view
	cursor, _ := v.At(5, 5)
	cursor.Write(Value(0xF0))
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(5, 5),
		},
		New: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xF0),
		},
	}, <-v.Inbox)

	// Add an object to an observed tile
	cursor.Add("A")
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xF0),
			Add:   "A",
		},
		New: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xF0),
			Add:   "A",
		},
	}, <-v.Inbox)

	// Delete an object from an observed tile
	cursor.Del("A")
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xF0),
			Del:   "A",
		},
		New: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xF0),
			Del:   "A",
		},
	}, <-v.Inbox)

	// Mask a tile in view
	cursor.Mask(0xFF, 0x0F)
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xF0),
		},
		New: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xFF),
		},
	}, <-v.Inbox)

	// Merge a tile in view
	cursor.Merge(func(v Value) Value {
		return 0xAA
	})
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xFF),
		},
		New: UpdateState[string]{
			Point: At(5, 5),
			Value: Value(0xAA),
		},
	}, <-v.Inbox)
}

func TestMove_Within(t *testing.T) {
	m := mapFrom("300x300.png")
	c := counter(0)
	v := NewView(m, "view 1")
	v.Resize(NewRect(0, 0, 10, 10), c.count)

	// Add an object to an observed tile. This should only fire once since
	// both the old and new states are the observed by the view.
	cursor, _ := v.At(5, 5)
	cursor.Move("A", At(6, 6))
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(5, 5),
			Del:   "A",
		},
		New: UpdateState[string]{
			Point: At(6, 6),
			Add:   "A",
		},
	}, <-v.Inbox)
}

func TestMove_Incoming(t *testing.T) {
	m := mapFrom("300x300.png")
	c := counter(0)
	v := NewView(m, "view 1")
	v.Resize(NewRect(0, 0, 10, 10), c.count)

	// Add an object to an observed tile from outside the view.
	cursor, _ := v.At(20, 20)
	cursor.Move("A", At(5, 5))
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(20, 20),
			Del:   "A",
		},
		New: UpdateState[string]{
			Point: At(5, 5),
			Add:   "A",
		},
	}, <-v.Inbox)
}

func TestMove_Outgoing(t *testing.T) {
	m := mapFrom("300x300.png")
	c := counter(0)
	v := NewView(m, "view 1")
	v.Resize(NewRect(0, 0, 10, 10), c.count)

	// Move an object from an observed tile outside of the view.
	cursor, _ := v.At(5, 5)
	cursor.Move("A", At(20, 20))
	assert.Equal(t, Update[string]{
		Old: UpdateState[string]{
			Point: At(5, 5),
			Del:   "A",
		},
		New: UpdateState[string]{
			Point: At(20, 20),
			Add:   "A",
		},
	}, <-v.Inbox)
}

func TestView_MoveTo(t *testing.T) {
	m := mapFrom("300x300.png")

	// Create a new view
	c := counter(0)
	v := NewView(m, "view 1")
	v.Resize(NewRect(10, 10, 12, 12), c.count)

	assert.NotNil(t, v)
	assert.Equal(t, 4, int(c))
	assert.Equal(t, 9, countObservers(m))

	const distance = 10

	assert.Equal(t, 1, countObserversAt(m, 10, 10))
	for i := 0; i < distance; i++ {
		v.MoveTo(East, 1, c.count)
	}

	assert.Equal(t, 0, countObserversAt(m, 10, 10))
	for i := 0; i < distance; i++ {
		v.MoveTo(South, 1, c.count)
	}

	assert.Equal(t, 0, countObserversAt(m, 10, 10))
	for i := 0; i < distance; i++ {
		v.MoveTo(West, 1, c.count)
	}

	assert.Equal(t, 0, countObserversAt(m, 10, 10))
	for i := 0; i < distance; i++ {
		v.MoveTo(North, 1, c.count)
	}

	// Start should have the observer attached
	assert.Equal(t, 1, countObserversAt(m, 10, 10))
	assert.Equal(t, 0, countObserversAt(m, 100, 100))

	// Count the number of observers, should be the same as before
	assert.Equal(t, 9, countObservers(m))
	assert.NoError(t, v.Close())
}

// ---------------------------------- Mocks ----------------------------------

func countObserversAt(m *Grid[string], x, y int16) (count int) {
	start, _ := m.At(x, y)
	start.Observers(func(view Observer[string]) {
		count++
	})
	return count
}

func countObservers(m *Grid[string]) int {
	var observers int
	m.Each(func(p Point, t Tile[string]) {
		if t.data.IsObserved() {
			observers++
		}
	})
	return observers
}

type fakeView[T comparable] func(*Update[T])

func (f fakeView[T]) Viewport() Rect {
	return Rect{}
}

func (f fakeView[T]) Resize(r Rect, fn func(Point, Tile[T])) {
	// Do nothing
}

func (f fakeView[T]) onUpdate(e *Update[T]) {
	f(e)
}

type counter int

func (c *counter) count(p Point, tile Tile[string]) {
	*c++
}
