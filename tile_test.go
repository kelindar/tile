// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

// Benchmark_Map/each-8         	     279	   4444495 ns/op	       0 B/op	       0 allocs/op
// Benchmark_Map/neighbors-8    	38763946	        30.9 ns/op	       0 B/op	       0 allocs/op
// Benchmark_Map/within-8       	   17290	     69175 ns/op	       0 B/op	       0 allocs/op
func Benchmark_Map(b *testing.B) {
	var d [5]byte
	defer assert.NotNil(b, d)
	m := NewMap(900, 900)

	b.Run("each", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Each(func(_ Point, tile Tile) {
				d = tile.Data // Pull data out
			})
		}
	})

	b.Run("neighbors", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Neighbors(300, 300, func(_ Point, tile Tile) {
				d = tile.Data // Pull data out
			})
		}
	})

	b.Run("within", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Within(At(100, 100), At(200, 200), func(p Point, tile Tile) {
				d = tile.Data // Pull data out
			})
		}
	})
}

func TestPageSize(t *testing.T) {
	assert.LessOrEqual(t, int(unsafe.Sizeof(page{})), 64)
}

func TestWithin(t *testing.T) {
	m := NewMap(9, 9)

	var path []string
	m.Within(At(1, 1), At(5, 5), func(p Point, tile Tile) {
		path = append(path, p.String())
	})
	assert.Equal(t, 25, len(path))
	assert.Equal(t, []string{
		"1,1", "2,1", "1,2", "2,2", "3,1",
		"4,1", "5,1", "3,2", "4,2", "5,2",
		"1,3", "2,3", "1,4", "2,4", "1,5",
		"2,5", "3,3", "4,3", "5,3", "3,4",
		"4,4", "5,4", "3,5", "4,5", "5,5",
	}, path)
}

func TestWithinCorner(t *testing.T) {
	m := NewMap(9, 9)

	var path []string
	m.Within(At(7, 6), At(10, 10), func(p Point, tile Tile) {
		path = append(path, p.String())
	})
	assert.Equal(t, 6, len(path))
	assert.Equal(t, []string{
		"7,6", "8,6", "7,7",
		"8,7", "7,8", "8,8",
	}, path)
}

func TestWithinInvalid(t *testing.T) {
	m := NewMap(9, 9)
	count := 0
	m.Within(At(10, 10), At(20, 20), func(p Point, tile Tile) {
		count++
	})
	assert.Equal(t, 0, count)
}

func TestEach(t *testing.T) {
	m := NewMap(9, 9)

	var path []string
	m.Each(func(p Point, tile Tile) {
		path = append(path, p.String())
	})
	assert.Equal(t, 81, len(path))
	assert.Equal(t, []string{
		"0,0", "1,0", "2,0",
		"0,1", "1,1", "2,1",
		"0,2", "1,2", "2,2",
		"3,0", "4,0", "5,0",
		"3,1", "4,1", "5,1",
		"3,2", "4,2", "5,2",
	}, path[:18])
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
	m := NewMap(9, 9)
	m.Each(func(p Point, tile Tile) {
		copy(tile.Data[:], p.String()[:3])
		m.UpdateAt(p.X, p.Y, tile)
	})

	// Run all the tests
	for _, tc := range tests {
		var out []string
		m.Neighbors(tc.x, tc.y, func(_ Point, tile Tile) {
			out = append(out, string(tile.Data[:3]))
		})
		assert.Equal(t, tc.expect, out)
	}
}

func TestAt(t *testing.T) {

	// Create a 9x9 map with labeled tiles
	m := NewMap(9, 9)
	m.Each(func(p Point, tile Tile) {
		copy(tile.Data[:], p.String()[:3])
		m.UpdateAt(p.X, p.Y, tile)
	})

	// Make sure our At() and the position matches
	m.Each(func(p Point, tile Tile) {
		at, _ := m.At(p.X, p.Y)
		assert.Equal(t, p.String(), string(at.Data[:3]))
	})
}
