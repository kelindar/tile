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

func TestPointAngle(t *testing.T) {
	tests := []struct {
		name     string
		from     Point
		to       Point
		expected Direction
	}{
		// Cardinal directions from origin
		{"North", At(0, 0), At(0, -1), North},
		{"East", At(0, 0), At(1, 0), East},
		{"South", At(0, 0), At(0, 1), South},
		{"West", At(0, 0), At(-1, 0), West},

		// Diagonal directions from origin
		{"NorthEast", At(0, 0), At(1, -1), NorthEast},
		{"SouthEast", At(0, 0), At(1, 1), SouthEast},
		{"SouthWest", At(0, 0), At(-1, 1), SouthWest},
		{"NorthWest", At(0, 0), At(-1, -1), NorthWest},

		// Same point (math.Atan2(0,0) = 0, which maps to East after transformation)
		{"Same point", At(5, 5), At(5, 5), East},

		// Non-origin starting points
		{"From 10,10 North", At(10, 10), At(10, 5), North},
		{"From 10,10 East", At(10, 10), At(15, 10), East},
		{"From 10,10 South", At(10, 10), At(10, 15), South},
		{"From 10,10 West", At(10, 10), At(5, 10), West},
		{"From 10,10 NorthEast", At(10, 10), At(15, 5), NorthEast},
		{"From 10,10 SouthEast", At(10, 10), At(15, 15), SouthEast},
		{"From 10,10 SouthWest", At(10, 10), At(5, 15), SouthWest},
		{"From 10,10 NorthWest", At(10, 10), At(5, 5), NorthWest},

		// Edge cases with larger distances
		{"Far North", At(0, 0), At(0, -100), North},
		{"Far East", At(0, 0), At(100, 0), East},
		{"Far South", At(0, 0), At(0, 100), South},
		{"Far West", At(0, 0), At(-100, 0), West},

		// Angles close to boundaries (testing rounding)
		{"Near North boundary", At(0, 0), At(1, -10), North},
		{"Near NorthEast boundary", At(0, 0), At(10, -10), NorthEast},
		{"Near East boundary", At(0, 0), At(10, -1), East},
		{"Near SouthEast boundary", At(0, 0), At(10, 10), SouthEast},

		// Negative coordinates
		{"Negative coords North", At(-5, -5), At(-5, -10), North},
		{"Negative coords East", At(-5, -5), At(0, -5), East},
		{"Negative coords South", At(-5, -5), At(-5, 0), South},
		{"Negative coords West", At(-5, -5), At(-10, -5), West},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.from.Angle(tc.to)
			assert.Equal(t, tc.expected, result,
				"Point %s to %s should be %s, got %s",
				tc.from.String(), tc.to.String(), tc.expected.String(), result.String())
		})
	}
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
