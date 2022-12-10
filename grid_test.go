// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

/*
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkGrid/each-8         	     799	   1489429 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/neighbors-8    	18933685	        60.70 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/within-8       	   25658	     47338 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/at-8           	80018670	        15.73 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/write-8        	80004266	        15.16 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/merge-8        	75013126	        16.57 ns/op	       0 B/op	       0 allocs/op

cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkGrid/each-8         	    1120	   1144320 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/neighbors-8    	75234746	        15.82 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/within-8       	   34467	     35889 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/at-8           	432614364	         2.781 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/write-8        	169958166	         7.014 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/merge-8        	131469820	         9.190 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkGrid(b *testing.B) {
	var d Cursor
	var p Point
	defer assert.NotNil(b, d)
	m := NewGrid(768, 768)

	b.Run("each", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Each(func(point Point, tile Cursor) {
				p = point
				d = tile
			})
		}
	})

	b.Run("neighbors", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Neighbors(300, 300, func(point Point, tile Cursor) {
				p = point
				d = tile
			})
		}
	})

	b.Run("within", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Within(At(100, 100), At(200, 200), func(point Point, tile Cursor) {
				p = point
				d = tile
			})
		}
	})

	assert.NotZero(b, p.X)
	b.Run("at", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			d, _ = m.At(100, 100)
		}
	})

	b.Run("write", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.WriteAt(100, 100, Tile(0))
		}
	})

	b.Run("merge", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.MergeAt(100, 100, Tile(0), Tile(1))
		}
	})
}

func TestPageSize(t *testing.T) {
	assert.Equal(t, 8, int(unsafe.Sizeof(map[uintptr]Point{})))
	assert.Equal(t, 64, int(unsafe.Sizeof(page{})))
	assert.Equal(t, 36, int(unsafe.Sizeof([9]Tile{})))
}

func TestWithin(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Within(At(1, 1), At(5, 5), func(p Point, tile Cursor) {
		path = append(path, p.String())
	})
	assert.Equal(t, 25, len(path))
	assert.ElementsMatch(t, []string{
		"1,1", "2,1", "1,2", "2,2", "3,1",
		"4,1", "5,1", "3,2", "4,2", "5,2",
		"1,3", "2,3", "1,4", "2,4", "1,5",
		"2,5", "3,3", "4,3", "5,3", "3,4",
		"4,4", "5,4", "3,5", "4,5", "5,5",
	}, path)
}

func TestWithinCorner(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Within(At(7, 6), At(10, 10), func(p Point, tile Cursor) {
		path = append(path, p.String())
	})
	assert.Equal(t, 6, len(path))
	assert.ElementsMatch(t, []string{
		"7,6", "8,6", "7,7",
		"8,7", "7,8", "8,8",
	}, path)
}

func TestWithinOneSide(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Within(At(1, 6), At(4, 10), func(p Point, tile Cursor) {
		path = append(path, p.String())
	})
	assert.Equal(t, 12, len(path))
	assert.ElementsMatch(t, []string{
		"1,6", "2,6", "3,6", "4,6",
		"1,7", "2,7", "3,7", "4,7",
		"1,8", "2,8", "3,8", "4,8",
	}, path)
}

func TestWithinInvalid(t *testing.T) {
	m := NewGrid(9, 9)
	count := 0
	m.Within(At(10, 10), At(20, 20), func(p Point, tile Cursor) {
		count++
	})
	assert.Equal(t, 0, count)
}

func TestEach(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Each(func(p Point, tile Cursor) {
		path = append(path, p.String())
	})
	assert.Equal(t, 81, len(path))
	assert.ElementsMatch(t, []string{
		"0,0", "1,0", "2,0", "0,1", "1,1", "2,1", "0,2", "1,2", "2,2",
		"0,3", "1,3", "2,3", "0,4", "1,4", "2,4", "0,5", "1,5", "2,5",
		"0,6", "1,6", "2,6", "0,7", "1,7", "2,7", "0,8", "1,8", "2,8",
		"3,0", "4,0", "5,0", "3,1", "4,1", "5,1", "3,2", "4,2", "5,2",
		"3,3", "4,3", "5,3", "3,4", "4,4", "5,4", "3,5", "4,5", "5,5",
		"3,6", "4,6", "5,6", "3,7", "4,7", "5,7", "3,8", "4,8", "5,8",
		"6,0", "7,0", "8,0", "6,1", "7,1", "8,1", "6,2", "7,2", "8,2",
		"6,3", "7,3", "8,3", "6,4", "7,4", "8,4", "6,5", "7,5", "8,5",
		"6,6", "7,6", "8,6", "6,7", "7,7", "8,7", "6,8", "7,8", "8,8",
	}, path)
}

func TestNeighbors(t *testing.T) {
	tests := []struct {
		x, y   int16
		expect []string
	}{
		{x: 0, y: 0, expect: []string{"1,0", "0,1"}},
		{x: 1, y: 0, expect: []string{"2,0", "1,1", "0,0"}},
		{x: 1, y: 1, expect: []string{"1,0", "2,1", "1,2", "0,1"}},
		{x: 2, y: 2, expect: []string{"2,1", "3,2", "2,3", "1,2"}},
		{x: 8, y: 8, expect: []string{"8,7", "7,8"}},
	}

	// Create a 9x9 map with labeled tiles
	m := NewGrid(9, 9)
	m.Each(func(p Point, tile Cursor) {
		m.WriteAt(p.X, p.Y, Tile(p.Integer()))
	})

	// Run all the tests
	for _, tc := range tests {
		var out []string
		m.Neighbors(tc.x, tc.y, func(_ Point, tile Cursor) {
			loc := unpackPoint(uint32(tile.Tile()))
			out = append(out, loc.String())
		})
		assert.ElementsMatch(t, tc.expect, out)
	}
}

func TestAt(t *testing.T) {

	// Create a 9x9 map with labeled tiles
	m := NewGrid(9, 9)
	m.Each(func(p Point, tile Cursor) {
		m.WriteAt(p.X, p.Y, Tile(p.Integer()))
	})

	// Make sure our At() and the position matches
	m.Each(func(p Point, tile Cursor) {
		at, _ := m.At(p.X, p.Y)
		assert.Equal(t, p.String(), unpackPoint(uint32(at.Tile())).String())
	})

	// Make sure that points match
	for y := int16(0); y < 9; y++ {
		for x := int16(0); x < 9; x++ {
			at, _ := m.At(x, y)
			assert.Equal(t, At(x, y).String(), unpackPoint(uint32(at.Tile())).String())
		}
	}
}

func TestUpdate(t *testing.T) {

	// Create a 9x9 map with labeled tiles
	m := NewGrid(9, 9)
	i := 0
	m.Each(func(p Point, _ Cursor) {
		i++
		m.WriteAt(p.X, p.Y, Tile(i))
	})

	// Assert the update
	cursor, _ := m.At(8, 8)
	assert.Equal(t, 81, int(cursor.Tile()))

	// 81 = 0b01010001
	delta := Tile(0b00101110) // change last 2 bits and should ignore other bits
	m.MergeAt(8, 8, delta, Tile(0b00000011))

	// original: 0101 0001
	// delta:    0010 1110
	// mask:     0000 0011
	// result:   0101 0010
	cursor, _ = m.At(8, 8)
	assert.Equal(t, 0b01010010, int(cursor.Tile()))
}
