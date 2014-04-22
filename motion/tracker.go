package motion

import "code.google.com/p/ncabatoff/imglib"
import "code.google.com/p/ncabatoff/imgseq"
import "image"

// Tracker is fed frames and produces as output the rect slices in those frames
// containing high activity, meaning they have a high color difference with respect
// to the average preceding frames.
type Tracker struct {
	frameNum  int
	frameRing ringbuf
	longSums  lnsumslc
	cdfb      columnDeltaFinderBuilder
}

func NewTracker() *Tracker {
	trk := Tracker{}
	trk.frameRing = ringbuf{data: make([]interface{}, LAVGN)}
	return &trk
}

// Add img to the tracker dataset and return rectangles found in it using 
// image color delta threshold t.
func (trk *Tracker) GetRects(img imgseq.Img, t int) []image.Rectangle {
	if odrs := trk.getRects(img, t); len(odrs) > 0 {
		if rects := FindConnectedRects(img.Bounds().Dx(), odrs); len(rects) > 0 {
			return rects
		}
	}
	return []image.Rectangle{}
}

func (trk *Tracker) getRects(img imgseq.Img, t int) []RowRects {
	nps := imglib.GetPixelSequence(img.Image)

	if len(trk.longSums) == 0 {
		trk.longSums = make(lnsumslc, len(nps.GetBytes()))
		trk.cdfb = rgbColumnDeltaFinderBuilder(img.Bounds().Dx())
		if _, ok := img.Image.(*imglib.YUYV); ok {
			trk.cdfb = yuvColumnDeltaFinderBuilder(img.Bounds().Dx())
		}
	}

	if old := trk.roll(img); old != nil {
		ops := imglib.GetPixelSequence(old.Image)
		return buildHeightOneRects(ops, nps, trk.longSums, t, trk.cdfb)
	}

	trk.longSums.add(nps.GetBytes())
	return []RowRects{}
}

// Return the buffer entry n frames in the past.
func (trk *Tracker) getOldFrame(n int) imgseq.Img {
	return trk.frameRing.data[(trk.frameRing.i+trk.frameRing.cnt+(LAVGN-n))%LAVGN].(imgseq.Img)
}

// Add img to the RingBuffer, returning what falls out of the ring buffer, or nil
// if the buffer wasn't already full.
func (trk *Tracker) roll(img imgseq.Img) *imgseq.Img {
	defer func() {
		if trk.frameRing.Size() == LAVGN {
			trk.frameRing.Dequeue()
		}
		trk.frameRing.Enqueue(img)
		trk.frameNum++
	}()

	if trk.frameRing.Size() == LAVGN {
		old := trk.frameRing.Peek().(imgseq.Img)
		return &old
	}
	return nil
}
