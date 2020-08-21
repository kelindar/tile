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

/*func TestPath2(t *testing.T) {
	m := mapFrom("300x300.png")
	path, dist, found := m.Path(At(115, 20), At(160, 270))
	assert.Equal(t, ``, plotPath(m, path))
	ioutil.WriteFile("path.txt", []byte(plotPath(m, path)), os.ModePerm)
	assert.Equal(t, 12, dist)
	assert.True(t, found)
}*/

func TestDraw(t *testing.T) {
	m := mapFrom("9x9.png")
	out := drawMap(m, NewRect(0, 0, 0, 0))
	assert.NotNil(t, out)
	/*f, err := os.Create("image.png")
	defer f.Close()

	assert.NoError(t, err)
	assert.NoError(t, png.Encode(f, out))
	assert.NoError(t, f.Close())*/
}

// Benchmark_Path/9x9-8         	  342571	      3342 ns/op	     712 B/op	       4 allocs/op
// Benchmark_Path/300x300-8     	     579	   2098446 ns/op	  532455 B/op	     255 allocs/op
func Benchmark_Path(b *testing.B) {
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
}

// BenchmarkHeap-8   	   94454	     12303 ns/op	    3968 B/op	       5 allocs/op
func BenchmarkHeap(b *testing.B) {
	for i := 0; i < b.N; i++ {
		h := newHeap32()
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
	h := newHeap32()
	h.Push(1, 0)
	h.Pop()
}

func TestNewHeap(t *testing.T) {
	h := newHeap32()
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
func mapFrom(name string) *Map {
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

	m := NewMap(int16(img.Bounds().Dx()), int16(img.Bounds().Dy()))
	for y := int16(0); y < m.Size.Y; y++ {
		for x := int16(0); x < m.Size.X; x++ {
			//fmt.Printf("%+v %T\n", img.At(int(x), int(y)), img.At(int(x), int(y)))
			v := img.At(int(x), int(y)).(color.RGBA)
			switch v.R {
			case 255:
			case 0:
				m.UpdateAt(x, y, Tile{0xff, 0, 0, 0, 0, 0})
			}
		}
	}
	return m
}

// plotPath plots the path on ASCII map
func plotPath(m *Map, path []Point) string {
	out := make([][]byte, m.Size.Y)
	for i := range out {
		out[i] = make([]byte, m.Size.X)
	}

	m.Each(func(l Point, tile Tile) {
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
func drawMap(m *Map, rect Rect) image.Image {
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
