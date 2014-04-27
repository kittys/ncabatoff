package motion

import "fmt"
import "image"
import "code.google.com/p/ncabatoff/imglib"

// LAVGN is how many frames get averaged.  Should be a power of two to ensure
// that div() is quick.
const LAVGN = 64

// lnsum is used to do rolling averages cheaply; it's a running total
type lnsum uint32

// return the average of the last LAVGN values
func (l lnsum) div() byte {
	return byte(l / LAVGN)
}

// return the next lnsum by adding a new value and subtracting an older one
// (the value from LAVGN values ago) to the running total
func (l lnsum) roll(oldv, newv byte) lnsum {
	return lnsum(int32(l) + int32(newv) - int32(oldv))
}

// deltaslc holds differences; this is just a helper type to distinguish
// these numbers from other other kinds of []int
type deltaslc []int

// sumslice is used to compute rolling averages of int slices.
type sumslice interface {
	// Fill delta with the difference of current average and b.
	// Add the values in b to the sums used to calculate averages.
	// Subtract old from the sums.
	rollSumDelta(b []byte, delta deltaslc, old []byte)

	// Add the values in b to the sums used to calculate averages.
	add(b []byte)
}

type lnsumslc []lnsum

func (lns lnsumslc) rollSumDelta(b []byte, delta deltaslc, old []byte) {
	for i := range lns {
		delta[i] = int(lns[i].div()) - int(b[i])
		lns[i] = lns[i].roll(old[i], b[i])
	}
}

func (lns lnsumslc) add(b []byte) {
	for i := range lns {
		lns[i] = lns[i].roll(0, b[i])
	}
}

// Given a deltaslc, which is just a slice of ints, a columnDeltaFinder
// knows how to interpret that slice in a color-based way.

type columnDeltaFinder interface {
	// Return the indexes into deltaslc that are
	find(d deltaslc) []int
}

type columnDeltaFinderBuilder interface {
	build() columnDeltaFinder
}

type grayColumnDeltaFinder []int
type grayColumnDeltaFinderBuilder int

func (gcdfb grayColumnDeltaFinderBuilder) build() columnDeltaFinder {
	return make(grayColumnDeltaFinder, int(gcdfb))
}

func getDeltasGray(dy int) int {
	return dy * dy
}

func (gcdf grayColumnDeltaFinder) find(d deltaslc) []int {
	for i := 0; i < len(gcdf); i += 2 {
		gcdf[i/2] = getDeltasGray(d[i])
	}
	return []int(gcdf)
}

type yuvColumnDeltaFinder []int
type yuvColumnDeltaFinderBuilder int

func (ycdfb yuvColumnDeltaFinderBuilder) build() columnDeltaFinder {
	return make(yuvColumnDeltaFinder, int(ycdfb))
}

func getDeltasYuv(dy1, du, dy2, dv int) (int, int) {
	duvs := du*du + dv*dv
	return dy1*dy1 + duvs, dy2*dy2 + duvs
}

func (ycdf yuvColumnDeltaFinder) find(d deltaslc) []int {
	p := 0
	for i := 0; i < len(ycdf); i += 2 {
		ycdf[i], ycdf[i+1] = getDeltasYuv(d[p], d[p+1], d[p+2], d[p+3])
		p += 4
	}
	return []int(ycdf)
}

type rgbColumnDeltaFinder []int
type rgbColumnDeltaFinderBuilder int

func (rcdfb rgbColumnDeltaFinderBuilder) build() columnDeltaFinder {
	return make(rgbColumnDeltaFinder, int(rcdfb))
}

func getDeltasRgb(dr, dg, db int) int {
	return dr*dr + dg*dg + db*db
}

// Populate columnDeltas with the result of calling getDeltasRgb for each pixel.
func (rcdf rgbColumnDeltaFinder) find(d deltaslc) []int {
	p := 0
	for i := 0; i < len(rcdf); i++ {
		rcdf[i] = getDeltasRgb(d[p], d[p+1], d[p+2])
		p += 3
	}
	return []int(rcdf)
}

// deltaFinder is used to update sums and find 1D rects for a single pixel row
type deltaFinder struct {
	oldps     imglib.PixelRow
	newps     imglib.PixelRow
	sums      lnsumslc
	deltaT    int
	y         int
	maxy      int
	deltas    deltaslc
	coldeltas columnDeltaFinder
	rects     []image.Rectangle
}

type deltaFinderJob struct {
	result []RowRects
	deltaFinder
}

func (df deltaFinder) String() string {
	off := df.newps.Offset
	return fmt.Sprintf("y=%d/%d off=%2d", df.y, df.maxy, off)
}

func newDeltaFinderJob(lnsums lnsumslc, oldps, newps imglib.PixelRow, deltaT, y, maxy int, cdf columnDeltaFinder) *deltaFinderJob {
	df := deltaFinder{oldps: oldps, newps: newps, sums: lnsums, deltaT: deltaT, y: y, maxy: maxy}
	df.deltas = make(deltaslc, newps.GetStride())
	df.coldeltas = cdf
	df.rects = make([]image.Rectangle, newps.Dx/2)
	return &deltaFinderJob{deltaFinder: df, result: make([]RowRects, maxy-y)}
}

func (df *deltaFinder) findRects() {
	off := df.newps.Offset
	start, end := off, off+df.newps.GetStride()
	df.sums[start:end].rollSumDelta(df.newps.GetBytes(), df.deltas, df.oldps.GetBytes())

	df.rects = df.rects[:0]
	for x, cd := range df.coldeltas.find(df.deltas) {
		if cd > df.deltaT {
			numr := len(df.rects)
			if numr > 0 && df.rects[numr-1].Max.X == x {
				df.rects[numr-1].Max.X++
			} else {
				df.rects = append(df.rects, image.Rect(x, df.y, x+1, df.y+1))
			}
		}
	}
}

func (df *deltaFinder) next() bool {
	if df.y == df.maxy-1 {
		return false
	}
	df.oldps.Offset += df.oldps.GetStride()
	df.newps.Offset += df.newps.GetStride()
	df.y++
	df.rects = df.rects[:0]
	return true
}

func buildDeltaFinderJobs(dfjs []deltaFinderJob, oldps, newps imglib.PixelSequence, sums lnsumslc, deltaT int, cdfb columnDeltaFinderBuilder) []RowRects {
	dfsize := oldps.Dy / len(dfjs)
	y, p, stride := 0, 0, newps.GetStride()
	rrs := make([]RowRects, oldps.Dy)

	for i := range dfjs {
		o, n := imglib.PixelRow{oldps, p}, imglib.PixelRow{newps, p}
		dfjs[i] = *newDeltaFinderJob(sums[p:p+stride], o, n, deltaT, y, y+dfsize, cdfb.build())
		dfjs[i].result = rrs[y : y+dfsize]
		p += stride
		y += dfsize
	}
	return rrs
}

func (dfj *deltaFinderJob) run() {
	numrows := dfj.maxy - dfj.y
	for i := 0; i < numrows; i++ {
		dfj.findRects()
		if len(dfj.rects) > 0 {
			dfj.result[i] = make([]image.Rectangle, len(dfj.rects))
			copy(dfj.result[i], dfj.rects)
		}
		dfj.next()
	}
}

func buildHeightOneRects(opxq, npxq imglib.PixelSequence, sums lnsumslc, t int, cdfb columnDeltaFinderBuilder) []RowRects {
	numtasks := 1 // npxq.Dy / 16
	if numtasks < 1 {
		numtasks = 1
	}
	dfjs := make([]deltaFinderJob, numtasks)
	rrs := buildDeltaFinderJobs(dfjs, opxq, npxq, sums, t, cdfb)
	runJobs(dfjs)
	return rrs
}

func runJobs(dfjs []deltaFinderJob) {
	n := len(dfjs)
	done := make(chan struct{}, n)

	for i := 0; i < n; i++ {
		go func(ii int) {
			dfjs[ii].run()
			done <- struct{}{}
		}(i)
	}

	for i := 0; i < n; i++ {
		<-done
	}
}

