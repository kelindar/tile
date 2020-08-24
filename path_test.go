// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPath(t *testing.T) {
	m := mapFrom("9x9.png")
	path, dist, found := m.Path(At(1, 1), At(7, 7), costOf)
	assert.Equal(t, `
.........
. x .   .
. x... ..
. xxx. ..
... x.  .
.   xx  .
.....x...
.    xx .
.........`, plotPath(m, path))
	assert.Equal(t, 12, dist)
	assert.True(t, found)
}

func TestPathTiny(t *testing.T) {
	m := NewGrid(6, 6)
	path, dist, found := m.Path(At(0, 0), At(5, 5), costOf)
	assert.Equal(t, `
 x    
 x    
 x    
 x    
 x    
 xxxx `, plotPath(m, path))
	assert.Equal(t, 10, dist)
	assert.True(t, found)
}

func TestDraw(t *testing.T) {
	m := mapFrom("9x9.png")
	out := drawGrid(m, NewRect(0, 0, 0, 0))
	assert.NotNil(t, out)
}

// BenchmarkPath/9x9-8         	  210472	      5316 ns/op	   16468 B/op	       3 allocs/op
// BenchmarkPath/300x300-8     	     463	   2546373 ns/op	 7801135 B/op	       4 allocs/op
// BenchmarkPath/381x381-8     	     373	   2732657 ns/op	62394362 B/op	       4 allocs/op
// BenchmarkPath/384x384-8     	     153	   7791925 ns/op	62396304 B/op	       5 allocs/op
// BenchmarkPath/6144x6144-8   	     158	   7468206 ns/op	62395377 B/op	       3 allocs/op
// BenchmarkPath/6147x6147-8   	     160	   7468716 ns/op	62395359 B/op	       3 allocs/op
func BenchmarkPath(b *testing.B) {
	b.Run("9x9", func(b *testing.B) {
		m := mapFrom("9x9.png")
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Path(At(1, 1), At(7, 7), costOf)
		}
	})

	b.Run("300x300", func(b *testing.B) {
		m := mapFrom("300x300.png")
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Path(At(115, 20), At(160, 270), costOf)
		}
	})

	b.Run("381x381", func(b *testing.B) {
		m := NewGrid(381, 381)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Path(At(0, 0), At(380, 380), costOf)
		}
	})

	b.Run("384x384", func(b *testing.B) {
		m := NewGrid(384, 384)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Path(At(0, 0), At(380, 380), costOf)
		}
	})

	b.Run("6144x6144", func(b *testing.B) {
		m := NewGrid(6144, 6144)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Path(At(0, 0), At(380, 380), costOf)
		}
	})

	b.Run("6147x6147", func(b *testing.B) {
		m := NewGrid(6147, 6147)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Path(At(0, 0), At(380, 380), costOf)
		}
	})

}

// BenchmarkAround/3r-8         	  352876	      3355 ns/op	     385 B/op	       1 allocs/op
// BenchmarkAround/5r-8         	  162103	      7551 ns/op	     931 B/op	       2 allocs/op
// BenchmarkAround/10r-8        	   62491	     19235 ns/op	    3489 B/op	       2 allocs/op
func BenchmarkAround(b *testing.B) {
	m := mapFrom("300x300.png")
	b.Run("3r", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Around(At(115, 20), 3, costOf, func(_ Point, _ Tile) {})
		}
	})

	b.Run("5r", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Around(At(115, 20), 5, costOf, func(_ Point, _ Tile) {})
		}
	})

	b.Run("10r", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			m.Around(At(115, 20), 10, costOf, func(_ Point, _ Tile) {})
		}
	})
}

func TestAround(t *testing.T) {
	m := mapFrom("9x9.png")

	for i := 0; i < 3; i++ {
		var path []string
		m.Around(At(2, 2), 3, costOf, func(p Point, tile Tile) {
			path = append(path, p.String())
		})
		assert.Equal(t, 10, len(path))
		assert.ElementsMatch(t, []string{
			"2,2", "2,1", "2,3", "1,2", "3,1",
			"1,1", "1,3", "3,3", "4,3", "3,4",
		}, path)
	}
}

// BenchmarkHeap-8   	   94454	     12303 ns/op	    3968 B/op	       5 allocs/op
func BenchmarkHeap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := newHeap32(16)
		for j := 0; j < 128; j++ {
			h.Push(rand(j), 1)
		}
		for j := 0; j < 128*10; j++ {
			h.Push(rand(j), 1)
			h.Pop()
		}
	}
}

func TestHeap(t *testing.T) {
	h := newHeap32(16)
	h.Push(1, 0)
	h.Pop()
}

func TestNewHeap(t *testing.T) {
	h := newHeap32(16)
	for j := 0; j < 8; j++ {
		h.Push(rand(j), uint32(j))
	}

	val, _ := h.Pop()
	for j := 1; j < 128; j++ {
		newval, ok := h.Pop()
		if ok {
			assert.True(t, val < newval)
			val = newval
		}
	}
}

// very fast semi-random function
func rand(i int) uint32 {
	i = i + 10000
	i = i ^ (i << 16)
	i = (i >> 5) ^ i
	return uint32(i & 0xFF)
}

// -----------------------------------------------------------------------------

// Cost estimation function
func costOf(tile Tile) uint16 {
	if (tile[0])&1 != 0 {
		return 0 // Blocked
	}
	return 1
}

// mapFrom creates a map from ASCII string
func mapFrom(name string) *Grid {
	f, err := os.Open("fixtures/" + name)
	defer f.Close()
	if err != nil {
		panic(err)
	}

	// Decode the image
	img, err := png.Decode(f)
	if err != nil {
		panic(err)
	}

	m := NewGrid(int16(img.Bounds().Dx()), int16(img.Bounds().Dy()))
	for y := int16(0); y < m.Size.Y; y++ {
		for x := int16(0); x < m.Size.X; x++ {
			//fmt.Printf("%+v %T\n", img.At(int(x), int(y)), img.At(int(x), int(y)))
			v := img.At(int(x), int(y)).(color.RGBA)
			switch v.R {
			case 255:
			case 0:
				m.WriteAt(x, y, Tile{0xff, 0, 0, 0, 0, 0})
			}

		}
	}
	return m
}

// plotPath plots the path on ASCII map
func plotPath(m *Grid, path []Point) string {
	out := make([][]byte, m.Size.Y)
	for i := range out {
		out[i] = make([]byte, m.Size.X)
	}

	m.Each(func(l Point, tile Tile) {
		//println(l.String(), int(tile[0]))
		switch {
		case pointInPath(l, path):
			out[l.Y][l.X] = 'x'
		case tile[0]&1 != 0:
			out[l.Y][l.X] = '.'
		default:
			out[l.Y][l.X] = ' '
		}
	})

	var sb strings.Builder
	for _, line := range out {
		sb.WriteByte('\n')
		sb.WriteString(string(line))
	}
	return sb.String()
}

// pointInPath returns whether a point is part of a path or not
func pointInPath(point Point, path []Point) bool {
	for _, p := range path {
		if p.Equal(point) {
			return true
		}
	}
	return false
}

// draw converts the map to a black and white image for debugging purposes.
func drawGrid(m *Grid, rect Rect) image.Image {
	if rect.Max.X == 0 || rect.Max.Y == 0 {
		rect = NewRect(0, 0, m.Size.X, m.Size.Y)
	}

	size := rect.Size()
	output := image.NewRGBA(image.Rect(0, 0, int(size.X), int(size.Y)))
	m.Within(rect.Min, rect.Max, func(p Point, tile Tile) {
		a := uint8(255)
		if tile[0] == 1 {
			a = 0
		}

		output.SetRGBA(int(p.X), int(p.Y), color.RGBA{a, a, a, 255})
	})
	return output
}
