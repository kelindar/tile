// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"strings"
	"testing"
	"unsafe"

	"github.com/stretchr/testify/assert"
)

var map9x9 = mapFrom(9, 9, `
xxxxxxxxx
x___x___x
x__xxx_xx
x___xx_xx
xxx_xx__x
x_______x
xxxxx_xxx
x_______x
xxxxxxxxx`)

func TestPath(t *testing.T) {
	path, dist, found := map9x9.Path(At(1, 1), At(7, 7))
	assert.Equal(t, `
xxxxxxxxx
x_._x___x
x_.xxx_xx
x_..xx_xx
xxx.xx__x
x__...__x
xxxxx.xxx
x____.._x
xxxxxxxxx`, plotPath(map9x9, path))
	assert.Equal(t, 12, dist)
	assert.True(t, found)
}

// Benchmark_Path/9x9-8         	  266665	      4438 ns/op	    2878 B/op	      36 allocs/op
// Benchmark_Path/9x9-8         	  300206	      3854 ns/op	    1372 B/op	      13 allocs/op
func Benchmark_Path(b *testing.B) {
	var d [6]byte
	defer assert.NotNil(b, d)

	b.Run("9x9", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			map9x9.Path(At(1, 1), At(7, 7))
		}
	})
}

// -----------------------------------------------------------------------------

// mapFrom creates a map from ASCII string
func mapFrom(height, width int, str string) *Map {
	m := NewMap(uint16(height), uint16(width))
	var y uint16
	for _, row := range strings.Split(str, "\n") {
		row = strings.TrimSpace(row)
		if len(row) != width {
			continue
		}

		for x, cell := range row {
			tile, _ := m.At(uint16(x), uint16(y))
			if cell == 'x' {
				tile.Flags = tile.Flags | Blocked
			}
		}

		y++
	}
	return m
}

// plotPath plots the path on ASCII map
func plotPath(m *Map, path []Point) string {
	out := make([][]byte, m.Height)
	for i := range out {
		out[i] = make([]byte, m.Width)
	}

	m.Each(func(l Point, tile *Tile) {
		switch {
		case pointInPath(l, path):
			out[l.Y][l.X] = '.'
		case tile.Flags&Blocked != 0:
			out[l.Y][l.X] = 'x'
		default:
			out[l.Y][l.X] = '_'
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

func TestSizeOfNode(t *testing.T) {
	assert.Equal(t, 40, int(unsafe.Sizeof(node{})))
}
