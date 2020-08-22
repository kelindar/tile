// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"bufio"
	"encoding/binary"
	"io"
)

const tileDataSize = 54

// WriteTo writes the grid to a specific writer.
func (m *Grid) WriteTo(dst io.Writer) (n int64, err error) {
	var c int
	p1 := At(0, 0)
	p2 := At(m.Size.X-1, m.Size.Y-1)

	// Write the viewport size
	header := make([]byte, 8)
	binary.BigEndian.PutUint16(header[0:2], uint16(p1.X))
	binary.BigEndian.PutUint16(header[2:4], uint16(p1.Y))
	binary.BigEndian.PutUint16(header[4:6], uint16(p2.X))
	binary.BigEndian.PutUint16(header[6:8], uint16(p2.Y))
	if c, err = dst.Write(header); err != nil {
		return
	}
	n += int64(c)

	// Write the grid data
	m.pagesWithin(p1, p2, func(page *page) {
		if c, err = dst.Write(page.Data()); err != nil {
			return
		}
		n += int64(c)
	})
	return
}

// ReadFrom reads the grid from the reader.
func ReadFrom(rdr io.Reader) (grid *Grid, err error) {
	reader := bufio.NewReader(rdr)
	header := make([]byte, 8)
	if _, err = io.ReadFull(reader, header); err != nil {
		return
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
		if _, err = io.ReadFull(reader, buf); err != nil {
			return
		}

		copy(page.Data(), buf)
	})
	return
}
