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
	return Rect{Min: nw, Max: se}.Contains(p)
}

// WithinRect checks if the point is within the specified bounding box.
func (p Point) WithinRect(box Rect) bool {
	return box.Contains(p)
}

// WithinSize checks if the point is within the specified bounding box
// which starts at 0,0 until the width/height provided.
func (p Point) WithinSize(size Point) bool {
	return p.X >= 0 && p.Y >= 0 && p.X < size.X && p.Y < size.Y
}

// Move moves a point by one in the specified direction.
func (p Point) Move(direction Direction) Point {
	return p.MoveBy(direction, 1)
}

// MoveBy moves a point by n in the specified direction.
func (p Point) MoveBy(direction Direction, n int16) Point {
	switch direction {
	case North:
		return Point{p.X, p.Y - n}
	case NorthEast:
		return Point{p.X + n, p.Y - n}
	case East:
		return Point{p.X + n, p.Y}
	case SouthEast:
		return Point{p.X + n, p.Y + n}
	case South:
		return Point{p.X, p.Y + n}
	case SouthWest:
		return Point{p.X - n, p.Y + n}
	case West:
		return Point{p.X - n, p.Y}
	case NorthWest:
		return Point{p.X - n, p.Y - n}
	default:
		return p
	}
}

// DistanceTo calculates manhattan distance to the other point
func (p Point) DistanceTo(other Point) uint32 {
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
// left,top,right,bottom correspond to x1,y1,x2,y2
func NewRect(left, top, right, bottom int16) Rect {
	return Rect{Min: At(left, top), Max: At(right, bottom)}
}

// Contains returns whether a point is within the rectangle or not.
func (a Rect) Contains(p Point) bool {
	return a.Min.X <= p.X && p.X < a.Max.X && a.Min.Y <= p.Y && p.Y < a.Max.Y
}

// Intersects returns whether a rectangle intersects with another rectangle or not.
func (a Rect) Intersects(b Rect) bool {
	return b.Min.X < a.Max.X && a.Min.X < b.Max.X && b.Min.Y < a.Max.Y && a.Min.Y < b.Max.Y
}

// Size returns the size of the rectangle
func (a *Rect) Size() Point {
	return Point{
		X: a.Max.X - a.Min.X,
		Y: a.Max.Y - a.Min.Y,
	}
}

// IsZero returns true if the rectangle is zero-value
func (a Rect) IsZero() bool {
	return a.Min.X == a.Max.X && a.Min.Y == a.Max.Y
}

// Difference calculates up to four non-overlapping regions in a that are not covered by b.
// If there are fewer than four distinct regions, the remaining Rects will be zero-value.
func (a Rect) Difference(b Rect) (result [4]Rect) {
	if b.Contains(a.Min) && b.Contains(a.Max) {
		return // Fully covered, return zero-value result
	}

	// Check for non-overlapping cases
	if !a.Intersects(b) {
		result[0] = a // No overlap, return A as is
		return
	}

	left := min(a.Min.X, b.Min.X)
	right := max(a.Max.X, b.Max.X)
	top := min(a.Min.Y, b.Min.Y)
	bottom := max(a.Max.Y, b.Max.Y)

	result[0].Min = Point{X: left, Y: top}
	result[0].Max = Point{X: right, Y: max(a.Min.Y, b.Min.Y)}

	result[1].Min = Point{X: left, Y: min(a.Max.Y, b.Max.Y)}
	result[1].Max = Point{X: right, Y: bottom}

	result[2].Min = Point{X: left, Y: top}
	result[2].Max = Point{X: max(a.Min.X, b.Min.X), Y: bottom}

	result[3].Min = Point{X: min(a.Max.X, b.Max.X), Y: top}
	result[3].Max = Point{X: right, Y: bottom}

	if result[0].Size().X == 0 || result[0].Size().Y == 0 {
		result[0] = Rect{}
	}
	if result[1].Size().X == 0 || result[1].Size().Y == 0 {
		result[1] = Rect{}
	}
	if result[2].Size().X == 0 || result[2].Size().Y == 0 {
		result[2] = Rect{}
	}
	if result[3].Size().X == 0 || result[3].Size().Y == 0 {
		result[3] = Rect{}
	}

	return
}

// -----------------------------------------------------------------------------

// Diretion represents a direction
type Direction byte

// Various directions
const (
	North Direction = iota
	NorthEast
	East
	SouthEast
	South
	SouthWest
	West
	NorthWest
)

// String returns a string representation of a direction
func (v Direction) String() string {
	switch v {
	case North:
		return "🡱N"
	case NorthEast:
		return "🡵NE"
	case East:
		return "🡲E"
	case SouthEast:
		return "🡶SE"
	case South:
		return "🡳S"
	case SouthWest:
		return "🡷SW"
	case West:
		return "🡰W"
	case NorthWest:
		return "🡴NW"
	default:
		return ""
	}
}

// Vector returns a direction vector with a given scale
func (v Direction) Vector(scale int16) Point {
	return Point{}.MoveBy(v, scale)
}
