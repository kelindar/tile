// Copyright (c) Roman Atachiants and contributors. All rights reserved.
// Licensed under the MIT license. See LICENSE file in the project root for details.

package tile

import (
	"fmt"
)

const invalid = int16(-1 << 15)

// -----------------------------------------------------------------------------

// Point represents a 2D coordinate.
type Point struct {
	X int16 // X coordinate
	Y int16 // Y coordinate
}

func unpackPoint(v uint32) Point {
	return At(int16(v>>16), int16(v))
}

// At creates a new point at a specified x,y coordinate.
func At(x, y int16) Point {
	return Point{X: x, Y: y}
}

// String returns string representation of a point.
func (p Point) String() string {
	return fmt.Sprintf("%v,%v", p.X, p.Y)
}

// Integer returns a packed 32-bit integer representation of a point.
func (p Point) Integer() uint32 {
	return (uint32(p.X) << 16) | (uint32(p.Y) & 0xffff)
}

// Equal compares two points and returns true if they are equal.
func (p Point) Equal(other Point) bool {
	return p.X == other.X && p.Y == other.Y
}

// Add adds two points together.
func (p Point) Add(p2 Point) Point {
	return Point{p.X + p2.X, p.Y + p2.Y}
}

// Subtract subtracts the second point from the first.
func (p Point) Subtract(p2 Point) Point {
	return Point{p.X - p2.X, p.Y - p2.Y}
}

// Multiply multiplies two points together.
func (p Point) Multiply(p2 Point) Point {
	return Point{p.X * p2.X, p.Y * p2.Y}
}

// Divide divides the first point by the second.
func (p Point) Divide(p2 Point) Point {
	return Point{p.X / p2.X, p.Y / p2.Y}
}

// MultiplyScalar multiplies the given point by the scalar.
func (p Point) MultiplyScalar(s int16) Point {
	return Point{p.X * s, p.Y * s}
}

// DivideScalar divides the given point by the scalar.
func (p Point) DivideScalar(s int16) Point {
	return Point{p.X / s, p.Y / s}
}

// Within checks if the point is within the specified bounding box.
func (p Point) Within(nw, se Point) bool {
	return p.X >= nw.X && p.Y >= nw.Y && p.X <= se.X && p.Y <= se.Y
}

// WithinRect checks if the point is within the specified bounding box.
func (p Point) WithinRect(box Rect) bool {
	return p.X >= box.Min.X && p.Y >= box.Min.Y && p.X <= box.Max.X && p.Y <= box.Max.Y
}

// WithinSize checks if the point is within the specified bounding box
// which starts at 0,0 until the width/height provided.
func (p Point) WithinSize(size Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < size.X && p.Y < size.Y
}

// ManhattanDistance calculates manhattan distance to the other point
func (p Point) ManhattanDistance(other Point) uint32 {
	return abs(int32(p.X)-int32(other.X)) + abs(int32(p.Y)-int32(other.Y))
}

func abs(n int32) uint32 {
	if n < 0 {
		return uint32(-n)
	}
	return uint32(n)
}

// -----------------------------------------------------------------------------

// Rect represents a rectangle
type Rect struct {
	Min Point // Top left point of the rectangle
	Max Point // Bottom right point of the rectangle
}

// NewRect creates a new rectangle
func NewRect(left, top, bottom, right int16) Rect {
	return Rect{Min: At(left, top), Max: At(bottom, right)}
}

// Contains returns whether a point is within the rectangle or not.
func (r *Rect) Contains(p Point) bool {
	return p.X >= r.Min.X && p.Y >= r.Min.Y && p.X <= r.Max.X && p.Y <= r.Max.Y
}

// Intersects returns whether a rectangle intersects with another rectangle or not.
func (r *Rect) Intersects(box Rect) bool {
	return !(box.Max.X < r.Min.X || box.Min.X > r.Max.X || box.Max.Y < r.Min.Y || box.Min.Y > r.Max.Y)
}

// Size returns the size of the rectangle
func (r *Rect) Size() Point {
	return Point{
		X: r.Max.X - r.Min.X,
		Y: r.Max.Y - r.Min.Y,
	}
}
