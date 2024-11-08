// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"fmt"
	"io"
	"sync"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkGrid/each-24         	    1452	    	830268 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/neighbors-24    	121583491	         9.861 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/within-24       	   49360	     	 24477 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/at-24           	687659378	         1.741 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/write-24        	191272338	         6.307 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/merge-24        	162536985	         7.332 ns/op	       0 B/op	       0 allocs/op
BenchmarkGrid/mask-24         	158258084	         7.601 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkGrid(b *testing.B) {
	var d Tile[uint32]
	var p Point
	defer assert.NotNil(b, d)
	m := NewGridOf[uint32](768, 768)

	b.Run("each", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Each(func(point Point, tile Tile[uint32]) {
				p = point
				d = tile
			})
		}
	})

	b.Run("neighbors", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Neighbors(300, 300, func(point Point, tile Tile[uint32]) {
				p = point
				d = tile
			})
		}
	})

	b.Run("within", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Within(At(100, 100), At(200, 200), func(point Point, tile Tile[uint32]) {
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
			m.WriteAt(100, 100, Value(0))
		}
	})

	b.Run("merge", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.MergeAt(100, 100, func(v Value) Value {
				v += 1
				return v
			})
		}
	})

	b.Run("mask", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.MaskAt(100, 100, Value(0), Value(1))
		}
	})
}

/*
cpu: 13th Gen Intel(R) Core(TM) i7-13700K
BenchmarkState/range-24         	17017800	        71.14 ns/op	       0 B/op	       0 allocs/op
BenchmarkState/add-24           	72639224	        16.32 ns/op	       0 B/op	       0 allocs/op
BenchmarkState/del-24           	82469125	        13.65 ns/op	       0 B/op	       0 allocs/op
*/
func BenchmarkState(b *testing.B) {
	m := NewGridOf[int](768, 768)
	m.Each(func(p Point, c Tile[int]) {
		for i := 0; i < 10; i++ {
			c.Add(i)
		}
	})

	b.Run("range", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			cursor, _ := m.At(100, 100)
			cursor.Range(func(v int) error {
				return nil
			})
		}
	})

	b.Run("add", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			cursor, _ := m.At(100, 100)
			cursor.Add(100)
		}
	})

	b.Run("del", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			cursor, _ := m.At(100, 100)
			cursor.Del(100)
		}
	})
}

func TestPageSize(t *testing.T) {
	assert.Equal(t, 8, int(unsafe.Sizeof(map[uintptr]Point{})))
	assert.Equal(t, 64, int(unsafe.Sizeof(page[string]{})))
	assert.Equal(t, 36, int(unsafe.Sizeof([9]Value{})))
}

func TestWithin(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Within(At(1, 1), At(5, 5), func(p Point, tile Tile[string]) {
		path = append(path, p.String())
	})
	assert.Equal(t, 16, len(path))
	assert.ElementsMatch(t, []string{
		"1,1", "2,1", "1,2", "2,2",
		"3,1", "4,1", "3,2", "4,2",
		"1,3", "2,3", "1,4", "2,4",
		"3,3", "4,3", "3,4", "4,4",
	}, path)
}

func TestWithinCorner(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Within(At(7, 6), At(10, 10), func(p Point, tile Tile[string]) {
		path = append(path, p.String())
	})
	assert.Equal(t, 6, len(path))
	assert.ElementsMatch(t, []string{
		"7,6", "8,6", "7,7",
		"8,7", "7,8", "8,8",
	}, path)
}

func TestWithinXY(t *testing.T) {
	assert.False(t, At(4, 8).WithinRect(NewRect(1, 6, 4, 10)))
}

func TestWithinOneSide(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Within(At(1, 6), At(4, 10), func(p Point, tile Tile[string]) {
		path = append(path, p.String())
	})
	assert.Equal(t, 9, len(path))
	assert.ElementsMatch(t, []string{
		"1,6", "2,6", "3,6",
		"1,7", "2,7", "3,7",
		"1,8", "2,8", "3,8",
	}, path)
}

func TestWithinInvalid(t *testing.T) {
	m := NewGrid(9, 9)
	count := 0
	m.Within(At(10, 10), At(20, 20), func(p Point, tile Tile[string]) {
		count++
	})
	assert.Equal(t, 0, count)
}

func TestEach(t *testing.T) {
	m := NewGrid(9, 9)

	var path []string
	m.Each(func(p Point, tile Tile[string]) {
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
	m.Each(func(p Point, tile Tile[string]) {
		m.WriteAt(p.X, p.Y, Value(p.Integer()))
	})

	// Run all the tests
	for _, tc := range tests {
		var out []string
		m.Neighbors(tc.x, tc.y, func(_ Point, tile Tile[string]) {
			loc := unpackPoint(uint32(tile.Value()))
			out = append(out, loc.String())
		})
		assert.ElementsMatch(t, tc.expect, out)
	}
}

func TestAt(t *testing.T) {

	// Create a 9x9 map with labeled tiles
	m := NewGrid(9, 9)
	m.Each(func(p Point, tile Tile[string]) {
		m.WriteAt(p.X, p.Y, Value(p.Integer()))
	})

	// Make sure our At() and the position matches
	m.Each(func(p Point, tile Tile[string]) {
		at, _ := m.At(p.X, p.Y)
		assert.Equal(t, p.String(), unpackPoint(uint32(at.Value())).String())
	})

	// Make sure that points match
	for y := int16(0); y < 9; y++ {
		for x := int16(0); x < 9; x++ {
			at, _ := m.At(x, y)
			assert.Equal(t, At(x, y).String(), unpackPoint(uint32(at.Value())).String())
		}
	}
}

func TestUpdate(t *testing.T) {

	// Create a 9x9 map with labeled tiles
	m := NewGrid(9, 9)
	i := 0
	m.Each(func(p Point, _ Tile[string]) {
		i++
		m.WriteAt(p.X, p.Y, Value(i))
	})

	// Assert the update
	cursor, _ := m.At(8, 8)
	assert.Equal(t, 81, int(cursor.Value()))

	// 81 = 0b01010001
	delta := Value(0b00101110) // change last 2 bits and should ignore other bits
	m.MaskAt(8, 8, delta, Value(0b00000011))

	// original: 0101 0001
	// delta:    0010 1110
	// mask:     0000 0011
	// result:   0101 0010
	cursor, _ = m.At(8, 8)
	assert.Equal(t, 0b01010010, int(cursor.Value()))
}

func TestState(t *testing.T) {
	m := NewGrid(9, 9)
	m.Each(func(p Point, c Tile[string]) {
		c.Add(p.String())
		c.Add(p.String()) // duplicate
	})

	m.Each(func(p Point, c Tile[string]) {
		assert.Equal(t, 1, c.Count())
		assert.NoError(t, c.Range(func(s string) error {
			assert.Equal(t, p.String(), s)
			return nil
		}))

		c.Del(p.String())
		assert.Equal(t, 0, c.Count())
	})
}

func TestStateRangeErr(t *testing.T) {
	m := NewGrid(9, 9)
	m.Each(func(p Point, c Tile[string]) {
		c.Add(p.String())
	})

	m.Each(func(p Point, c Tile[string]) {
		assert.Error(t, c.Range(func(s string) error {
			return io.EOF
		}))
	})
}

func TestPointOf(t *testing.T) {
	truthTable := func(x, y int16, idx uint8) (int16, int16) {
		switch idx {
		case 0:
			return x, y
		case 1:
			return x + 1, y
		case 2:
			return x + 2, y
		case 3:
			return x, y + 1
		case 4:
			return x + 1, y + 1
		case 5:
			return x + 2, y + 1
		case 6:
			return x, y + 2
		case 7:
			return x + 1, y + 2
		case 8:
			return x + 2, y + 2
		default:
			return x, y
		}
	}

	for i := 0; i < 9; i++ {
		at := pointOf(At(0, 0), uint8(i))
		x, y := truthTable(0, 0, uint8(i))
		assert.Equal(t, x, at.X, fmt.Sprintf("idx=%v", i))
		assert.Equal(t, y, at.Y, fmt.Sprintf("idx=%v", i))
	}
}

func TestConcurrentMerge(t *testing.T) {
	const count = 10000
	var wg sync.WaitGroup
	wg.Add(count)

	m := NewGrid(9, 9)
	for i := 0; i < count; i++ {
		go func() {
			m.MergeAt(1, 1, func(v Value) Value {
				v += 1
				return v
			})
			wg.Done()
		}()
	}

	wg.Wait()
	tile, ok := m.At(1, 1)
	assert.True(t, ok)
	assert.Equal(t, uint32(count), tile.Value())
}
