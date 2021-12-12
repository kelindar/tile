// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"encoding/binary"
	"io"

	"github.com/kelindar/iostream"
)

const tileDataSize = 54

// WriteTo writes the grid to a specific writer.
func (m *Grid) WriteTo(dst io.Writer) (n int64, err error) {
	p1 := At(0, 0)
	p2 := At(m.Size.X-1, m.Size.Y-1)

	// Write the viewport size
	w := iostream.NewWriter(dst)
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], uint16(p1.X))
	binary.BigEndian.PutUint16(header[2:4], uint16(p1.Y))
	binary.BigEndian.PutUint16(header[4:6], uint16(p2.X))
	binary.BigEndian.PutUint16(header[6:8], uint16(p2.Y))
	if _, err := w.Write(header); err != nil {
		return w.Offset(), err
	}

	// Write the grid data
	m.pagesWithin(p1, p2, func(page *page) {
		if _, err := w.Write(page.Data()); err != nil {
			return
		}
	})
	return w.Offset(), nil
}

// ReadFrom reads the grid from the reader.
func ReadFrom(src io.Reader) (grid *Grid, err error) {
	r := iostream.NewReader(src)
	header := make([]byte, 8)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, err
	}

	// Read the size
	var view Rect
	view.Min.X = int16(binary.BigEndian.Uint16(header[0:2]))
	view.Min.Y = int16(binary.BigEndian.Uint16(header[2:4]))
	view.Max.X = int16(binary.BigEndian.Uint16(header[4:6]))
	view.Max.Y = int16(binary.BigEndian.Uint16(header[6:8]))

	// Allocate a new grid
	grid = NewGrid(view.Max.X+1, view.Max.Y+1)
	buf := make([]byte, tileDataSize)
	grid.pagesWithin(view.Min, view.Max, func(page *page) {
		if _, err := io.ReadFull(r, buf); err != nil {
			return
		}

		copy(page.Data(), buf)
	})
	return
}
