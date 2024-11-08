// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkPoint/within-24         	1000000000	         0.09854 ns/op	       0 B/op	       0 allocs/op
BenchmarkPoint/within-rect-24    	1000000000	         0.09966 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkPoint(b *testing.B) {
	p := At(10, 20)
	b.Run("within", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			p.Within(At(0, 0), At(100, 100))
		}
	})

	b.Run("within-rect", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			p.WithinRect(NewRect(0, 0, 100, 100))
		}
	})
}

func TestPoint(t *testing.T) {
	p := At(10, 20)
	p2 := At(2, 2)

	assert.Equal(t, int16(10), p.X)
	assert.Equal(t, int16(20), p.Y)
	assert.Equal(t, uint32(0xa0014), p.Integer())
	assert.Equal(t, At(-5, 5), unpackPoint(At(-5, 5).Integer()))
	assert.Equal(t, "10,20", p.String())
	assert.True(t, p.Equal(At(10, 20)))
	assert.Equal(t, "20,40", p.MultiplyScalar(2).String())
	assert.Equal(t, "5,10", p.DivideScalar(2).String())
	assert.Equal(t, "12,22", p.Add(p2).String())
	assert.Equal(t, "8,18", p.Subtract(p2).String())
	assert.Equal(t, "20,40", p.Multiply(p2).String())
	assert.Equal(t, "5,10", p.Divide(p2).String())
	assert.True(t, p.Within(At(1, 1), At(20, 30)))
	assert.True(t, p.WithinRect(NewRect(1, 1, 20, 30)))
	assert.False(t, p.WithinSize(At(10, 20)))
	assert.True(t, p.WithinSize(At(20, 30)))
}

func TestIntersects(t *testing.T) {
	assert.True(t, NewRect(0, 0, 2, 2).Intersects(NewRect(1, 0, 3, 2)))
	assert.False(t, NewRect(0, 0, 2, 2).Intersects(NewRect(2, 0, 4, 2)))
	assert.False(t, NewRect(10, 10, 12, 12).Intersects(NewRect(9, 12, 11, 14)))
}

func TestDirection(t *testing.T) {
	for i := 0; i < 8; i++ {
		dir := Direction(i)
		assert.NotEmpty(t, dir.String())
	}
}

func TestDirection_Empty(t *testing.T) {
	dir := Direction(9)
	assert.Empty(t, dir.String())
}

func TestMove(t *testing.T) {
	tests := []struct {
		dir Direction
		out Point
	}{
		{North, Point{X: 0, Y: -1}},
		{South, Point{X: 0, Y: 1}},
		{East, Point{X: 1, Y: 0}},
		{West, Point{X: -1, Y: 0}},
		{NorthEast, Point{X: 1, Y: -1}},
		{NorthWest, Point{X: -1, Y: -1}},
		{SouthEast, Point{X: 1, Y: 1}},
		{SouthWest, Point{X: -1, Y: 1}},
		{Direction(99), Point{}},
	}

	for _, tc := range tests {
		assert.Equal(t, tc.out, Point{}.Move(tc.dir), tc.dir.String())
	}
}

func TestContains(t *testing.T) {
	tests := map[Point]bool{
		{X: 0, Y: 0}: true,
		{X: 1, Y: 0}: true,
		{X: 0, Y: 1}: true,
		{X: 1, Y: 1}: true,
		{X: 2, Y: 2}: false,
		{X: 3, Y: 3}: false,
		{X: 1, Y: 2}: false,
		{X: 2, Y: 1}: false,
	}

	for point, expect := range tests {
		r := NewRect(0, 0, 2, 2)
		assert.Equal(t, expect, r.Contains(point), point.String())
	}
}

func TestDiff_Right(t *testing.T) {
	a := Rect{At(0, 0), At(2, 2)}
	b := Rect{At(1, 0), At(3, 2)}

	diff := a.Difference(b)
	assert.Equal(t, Rect{At(0, 0), At(1, 2)}, diff[2])
	assert.Equal(t, Rect{At(2, 0), At(3, 2)}, diff[3])
}

func TestDiff_Left(t *testing.T) {
	a := Rect{At(0, 0), At(2, 2)}
	b := Rect{At(-1, 0), At(1, 2)}

	diff := a.Difference(b)
	assert.Equal(t, Rect{At(-1, 0), At(0, 2)}, diff[2])
	assert.Equal(t, Rect{At(1, 0), At(2, 2)}, diff[3])
}

func TestDiff_Up(t *testing.T) {
	a := Rect{At(0, 0), At(2, 2)}
	b := Rect{At(0, -1), At(2, 1)}

	diff := a.Difference(b)
	assert.Equal(t, Rect{At(0, -1), At(2, 0)}, diff[0])
	assert.Equal(t, Rect{At(0, 1), At(2, 2)}, diff[1])
}

func TestDiff_Down(t *testing.T) {
	a := Rect{At(0, 0), At(2, 2)}
	b := Rect{At(0, 1), At(2, 3)}

	diff := a.Difference(b)
	assert.Equal(t, Rect{At(0, 0), At(2, 1)}, diff[0])
	assert.Equal(t, Rect{At(0, 2), At(2, 3)}, diff[1])
}
