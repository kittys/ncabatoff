package motion

import "fmt"
import "math"
import "image"

// RowRects contain height-one rects that share the same row, thus each has the
// same Min.Y and have Max.Y=Min.Y+1.
type RowRects []image.Rectangle

// A RectAreaSlice is a []image.Rectangle which sorts by decreasing area.
type RectAreaSlice []image.Rectangle

func (r RectAreaSlice) Len() int {
	return len(r)
}
func (r RectAreaSlice) Swap(a, b int) {
	r[a], r[b] = r[b], r[a]
}
func (r RectAreaSlice) Less(a, b int) bool {
	asz, bsz := r[a].Size(), r[b].Size()
	return asz.X*asz.Y > bsz.X*bsz.Y
}

// XY is used for floating-point 2D image coordinates.  They string as (###.#,###.#).
type XY struct {
	X, Y float64
}

func (xy XY) String() string {
	return fmt.Sprintf("(%5.1f,%5.1f)", xy.X, xy.Y)
}

// Return the integer image.Point obtained by flooring (X,Y).
func (xy XY) Point() image.Point {
	return image.Point{int(xy.X), int(xy.Y)}
}

// Distance return sqrt(X^2 + Y^2).
func (p1 XY) Distance(p2 XY) float64 {
	dx, dy := p2.X-p1.X, p2.Y-p1.Y
	return math.Sqrt(float64(dx*dx) + float64(dy*dy))
}
