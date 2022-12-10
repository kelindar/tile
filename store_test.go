// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"bytes"
	"compress/flate"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

/*
cpu: Intel(R) Core(TM) i7-9700K CPU @ 3.60GHz
BenchmarkStore/save-8         	    9068	    129974 ns/op	       8 B/op	       1 allocs/op
BenchmarkStore/read-8         	    2967	    379663 ns/op	  647465 B/op	       8 allocs/op
*/
func BenchmarkStore(b *testing.B) {
	m := mapFrom("300x300.png")

	b.Run("save", func(b *testing.B) {
		out := bytes.NewBuffer(make([]byte, 0, 550000))

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			out.Reset()
			m.WriteTo(out)
		}
	})

	b.Run("read", func(b *testing.B) {
		enc := new(bytes.Buffer)
		m.WriteTo(enc)

		b.ReportAllocs()
		b.ResetTimer()
		for n := 0; n < b.N; n++ {
			ReadFrom[string](bytes.NewBuffer(enc.Bytes()))
		}
	})

}

func TestSaveLoad(t *testing.T) {
	m := mapFrom("300x300.png")

	// Save the map
	enc := new(bytes.Buffer)
	n, err := m.WriteTo(enc)
	assert.NoError(t, err)
	assert.Equal(t, int64(360008), n)

	// Load the map back
	out, err := ReadFrom[string](enc)
	assert.NoError(t, err)
	assert.Equal(t, m.pages, out.pages)
}

func TestSaveLoadFlate(t *testing.T) {
	m := mapFrom("300x300.png")

	// Save the map
	output := new(bytes.Buffer)
	writer, err := flate.NewWriter(output, flate.BestSpeed)
	assert.NoError(t, err)

	n, err := m.WriteTo(writer)
	assert.NoError(t, writer.Close())
	assert.NoError(t, err)
	assert.Equal(t, int64(360008), n)
	assert.Equal(t, int(16533), output.Len())

	// Load the map back
	reader := flate.NewReader(output)
	out, err := ReadFrom[string](reader)
	assert.NoError(t, err)
	assert.Equal(t, m.pages, out.pages)
}

func TestSaveLoadFile(t *testing.T) {
	temp, err := os.CreateTemp("", "*")
	assert.NoError(t, err)
	defer os.Remove(temp.Name())

	// Write a test map into temp file
	m := mapFrom("300x300.png")
	assert.NoError(t, m.WriteFile(temp.Name()))

	fi, _ := temp.Stat()
	assert.Equal(t, int64(16533), fi.Size())

	// Read the map back
	out, err := ReadFile[string](temp.Name())
	assert.NoError(t, err)
	assert.Equal(t, m.pages, out.pages)
}
