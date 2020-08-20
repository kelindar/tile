// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var map9x9 = mapFrom(9, 9, `
.........
.   .   .
.  ... ..
.    . ..
...  .  .
.       .
..... ...
.       .
.........`)

func TestPath(t *testing.T) {
	path, dist, found := map9x9.Path(At(1, 1), At(7, 7))
	assert.Equal(t, `
.........
. x .   .
. x... ..
. xxx. ..
... x.  .
.   xx  .
.....x...
.    xx .
.........`, plotPath(map9x9, path))
	assert.Equal(t, 12, dist)
	assert.True(t, found)
}

// Benchmark_Path/9x9-8         	  266665	      4438 ns/op	    2878 B/op	      36 allocs/op
// Benchmark_Path/9x9-8         	  300206	      3854 ns/op	    1372 B/op	      13 allocs/op
// Benchmark_Path/9x9-8         	  428864	      2737 ns/op	     750 B/op	       7 allocs/op
// Benchmark_Path/9x9-8         	  428580	      2611 ns/op	     678 B/op	       4 allocs/op
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
			if cell == '.' {
				m.UpdateAt(uint16(x), uint16(y), Tile{
					Flags: Blocked,
				})
			}
		}

		y++
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
		case tile.Flags&Blocked != 0:
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
