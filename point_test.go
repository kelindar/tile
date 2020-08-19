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

	assert.Equal(t, uint16(10), p.X)
	assert.Equal(t, uint16(20), p.Y)
	assert.Equal(t, uint32(0xa0014), p.Integer())
	assert.Equal(t, p, unpackPoint(p.Integer()))
	assert.Equal(t, "10,20", p.String())
	assert.True(t, p.Equal(At(10, 20)))
	assert.Equal(t, "20,40", p.MultiplyScalar(2).String())
	assert.Equal(t, "5,10", p.DivideScalar(2).String())
	assert.Equal(t, "12,22", p.Add(p2).String())
	assert.Equal(t, "8,18", p.Subtract(p2).String())
	assert.Equal(t, "20,40", p.Multiply(p2).String())
	assert.Equal(t, "5,10", p.Divide(p2).String())
}
