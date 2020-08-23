// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	assert.True(t, p.Within(At(1, 1), At(10, 20)))
	assert.True(t, p.WithinRect(NewRect(1, 1, 10, 20)))
	assert.False(t, p.WithinSize(At(10, 20)))
	assert.True(t, p.WithinSize(At(20, 30)))
}

func TestMorton(t *testing.T) {
	p := At(8191, 8191)
	assert.Equal(t, 67108863, int(p.Interleave()))
	assert.Equal(t, p, deinterleavePoint(67108863))
}

// BenchmarkPoint/within-8         	1000000000	         0.220 ns/op	       0 B/op	       0 allocs/op
// BenchmarkPoint/within-rect-8    	1000000000	         0.219 ns/op	       0 B/op	       0 allocs/op
// BenchmarkPoint/interleave-8     	1000000000	         0.657 ns/op	       0 B/op	       0 allocs/op
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

	b.Run("interleave", func(b *testing.B) {
		out := int32(0)
		p := At(8191, 8191)
		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			out = p.Interleave()
		}
		assert.NotZero(b, out)
	})
}
