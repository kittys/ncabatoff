package motion

import . "gopkg.in/check.v1"
import "testing"

import "code.google.com/p/ncabatoff/imglib"
import "code.google.com/p/ncabatoff/imgseq"
import "image"
import "image/color"
import "math/rand"

// Hook up gocheck into the gotest runner.
func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func odr(y0, x0, x1 int) image.Rectangle {
	return image.Rect(x0, y0, x1, y0+1)
}

func zrslc() []image.Rectangle {
	return []image.Rectangle{}
}

func rslc(r ...image.Rectangle) []image.Rectangle {
	return r
}

func rslcslc(r ...[]image.Rectangle) [][]image.Rectangle {
	return r
}

func rrectslc(r ...RowRects) []RowRects {
	return r
}

func randBytes(n int) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = byte(rand.Intn(255))
	}
	return b
}

func byteToDelta(b []byte) deltaslc {
	ret := make(deltaslc, len(b))
	for i := range ret {
		ret[i] = int(b[i])
	}
	return ret
}

func byteToLnsum(b []byte) lnsumslc {
	ret := make(lnsumslc, len(b))
	for i := range ret {
		ret[i] = lnsum(b[i])
	}
	return ret
}

func (s *MySuite) TestRollSumDeltaBytes(c *C) {
	b0 := []byte{0, 0, 0, 0}
	b1 := []byte{1, 2, 3, 4}
	sum := make(lnsumslc, len(b0))
	delta := make(deltaslc, len(b0))
	sum.rollSumDelta(b1, delta, b0)
	c.Check(sum, DeepEquals, byteToLnsum(b1))
	c.Check(delta, DeepEquals, deltaslc{-1, -2, -3, -4})

	for i := 0; i < 62; i++ {
		sum.rollSumDelta(b1, delta, b0)
	}
	c.Check(sum, DeepEquals, lnsumslc{1 * 63, 2 * 63, 3 * 63, 4 * 63})
	c.Check(delta, DeepEquals, deltaslc{-1, -1, -1, -1})

	sum.rollSumDelta(b1, delta, b0)
	c.Check(sum, DeepEquals, lnsumslc{1 * 64, 2 * 64, 3 * 64, 4 * 64})
	c.Check(delta, DeepEquals, deltaslc{-1, -1, -1, -1})

	// Now test subtraction of old
	sum.rollSumDelta(b1, delta, b1)
	c.Check(sum, DeepEquals, lnsumslc{1 * 64, 2 * 64, 3 * 64, 4 * 64})
	c.Check(delta, DeepEquals, byteToDelta(b0))

	sum.rollSumDelta(b0, delta, b1)
	c.Check(sum, DeepEquals, lnsumslc{1 * 63, 2 * 63, 3 * 63, 4 * 63})
	c.Check(delta, DeepEquals, byteToDelta(b1))
}

func (s *MySuite) BenchmarkRollSumDeltaBytes(c *C) {
	rb := randBytes(160)
	sum := make(lnsumslc, len(rb))
	delta := make(deltaslc, len(rb))
	c.ResetTimer()
	// p := profile.Start(profile.CPUProfile)
	for i := 0; i < c.N; i++ {
		sum.rollSumDelta(rb, delta, rb)
	}
}

func (s *MySuite) BenchmarkRollSumDeltaBytesWithRand(c *C) {
	sum := make(lnsumslc, 160)
	delta := make(deltaslc, 160)
	// p := profile.Start(profile.CPUProfile)
	lastrb := make([]byte, 160)
	for i := 0; i < c.N; i++ {
		rb := randBytes(160)
		sum.rollSumDelta(rb, delta, lastrb)
		lastrb = rb
	}
}

func (s *MySuite) TestRgbDeltas(c *C) {
	oimg := imglib.NewRGB(image.Rect(0, 0, 4, 1))
	rimg := imglib.NewRGB(image.Rect(0, 0, 4, 1))
	rimg.SetRGBA(1, 0, color.RGBA{1, 0, 2, 0xFF})
	rimg.SetRGBA(3, 0, color.RGBA{0, 3, 0, 0xFF})
	c.Check(imglib.GetImageBytes(rimg), DeepEquals, imglib.RgbBytes(rimg.Pix))

	sum := make(lnsumslc, len(rimg.Pix))
	deltas := make(deltaslc, len(rimg.Pix))
	sum.rollSumDelta(rimg.Pix, deltas, oimg.Pix)

	rcdf := rgbColumnDeltaFinderBuilder(rimg.Bounds().Dx()).build()
	c.Check(rcdf.find(deltas), DeepEquals, []int{0, 5, 0, 9})
}

func (s *MySuite) TestYuvDeltas(c *C) {
	oimg := imglib.NewYUYV(image.Rect(0, 0, 4, 1))
	yimg := imglib.NewYUYV(image.Rect(0, 0, 4, 1))
	z := color.YCbCr{}
	newrow := []color.YCbCr{z, {Y: 1, Cb: 2}, z, {Cr: 3}}
	yimg.SetRow(0, newrow)
	c.Check(yimg.Pix, DeepEquals, []byte{0, 2, 1, 0, 0, 0, 0, 3})

	c.Check(imglib.GetImageBytes(yimg), DeepEquals, imglib.YuyvBytes(yimg.Pix))

	sum := make(lnsumslc, len(yimg.Pix))
	deltas := make(deltaslc, len(yimg.Pix))
	sum.rollSumDelta(yimg.Pix, deltas, oimg.Pix)

	ycdf := yuvColumnDeltaFinderBuilder(yimg.Bounds().Dx()).build()
	c.Check(ycdf.find(deltas), DeepEquals, []int{4, 5, 9, 9})
}

func heightOneRectsRgb(oimg, nimg *imglib.RGB, t int) ([]RowRects, lnsumslc) {
	sums := make(lnsumslc, len(nimg.Pix))
	opxq := imglib.GetPixelSequence(oimg)
	npxq := imglib.GetPixelSequence(nimg)
	return buildHeightOneRects(opxq, npxq, sums, t, rgbColumnDeltaFinderBuilder(npxq.Dx)), sums
}

func (s *MySuite) TestHeightOneRects(c *C) {
	img := imglib.NewRGB(image.Rect(0, 0, 4, 4))
	old := imglib.NewRGB(image.Rect(0, 0, 4, 4))
	img.SetRGBA(1, 1, color.RGBA{0, 2 * LAVGN, 3 * LAVGN, 0xFF})
	ssqr := 128*128 + 192*192

	imgr1, oldr1 := img.SubImage(image.Rect(0, 0, 4, 1)), old.SubImage(image.Rect(0, 0, 4, 1))
	odrss, sums := heightOneRectsRgb(oldr1.(*imglib.RGB), imgr1.(*imglib.RGB), ssqr)
	c.Check(sums, DeepEquals, make(lnsumslc, len(sums)))
	c.Check(odrss, DeepEquals, rrectslc(RowRects(nil)))

	imgr2, oldr2 := img.SubImage(image.Rect(0, 1, 4, 2)), old.SubImage(image.Rect(0, 1, 4, 2))
	odrss, sums = heightOneRectsRgb(oldr2.(*imglib.RGB), imgr2.(*imglib.RGB), ssqr)
	r2sums := make(lnsumslc, len(sums))
	r2sums[4] = 2 * LAVGN
	r2sums[5] = 3 * LAVGN
	c.Check(sums, DeepEquals, r2sums)
	c.Check(odrss, DeepEquals, rrectslc(RowRects(nil)))
}

func (s *MySuite) TestAllHeightOneRects(c *C) {
	img := imglib.NewRGB(image.Rect(0, 0, 4, 4))
	old := imglib.NewRGB(image.Rect(0, 0, 4, 4))
	img.SetRGBA(1, 1, color.RGBA{0, 2 * LAVGN, 3 * LAVGN, 0xFF})
	ssqr := 128*128 + 192*192

	odrss, sums := heightOneRectsRgb(old, img, ssqr)
	nilrr := RowRects(nil)
	c.Check(odrss, DeepEquals, rrectslc(nilrr, nilrr, nilrr, nilrr))
	explns := make(lnsumslc, len(img.Pix))
	explns[img.PixOffset(1, 1)+1] = 2 * LAVGN
	explns[img.PixOffset(1, 1)+2] = 3 * LAVGN
	c.Check(sums, DeepEquals, explns)

	//  odrss, sums = heightOneRectsRgb(old, img, ssqr)
	//  c.Check(odrss, DeepEquals, rslcslc(zrslc(), rslc(odr(1, 1, 2)), zrslc(), zrslc(), zrslc()))

	//  img.SetRGBA(2, 2, color.RGBA{0, 2*LAVGN, 3*LAVGN, 0xFF})
	//  odrss, sums = heightOneRectsRgb(old, img, ssqr)
	//  c.Check(odrss, DeepEquals, rslcslc(zrslc(), rslc(odr(1, 1, 2)), rslc(odr(2, 2, 3)), zrslc(), zrslc()))
}

/*
func (s *MySuite) TestGetHeightOneRects(c *C) {
    rs := make([]image.Rectangle, 6)
    d1 := make([]int32, 15)
    c.Check(heightOneRectsRgb(rs, d1, 0, 0, 1), DeepEquals, []image.Rectangle{})

    y0odr := func(x0, x1 int) image.Rectangle {
        return image.Rectangle{image.Point{x0, 0}, image.Point{x1, 1}}
    }

    // delta val must exceed t to count
    d1[0] = 1
    c.Check(heightOneRectsRgb(rs, d1, 0, 0, 1), DeepEquals, []image.Rectangle{})

    d1[1] = 1
    c.Check(heightOneRectsRgb(rs, d1, 0, 0, 1), DeepEquals, rslc(y0odr(0, 1)))

    d1[3] = 2
    c.Check(heightOneRectsRgb(rs, d1, 0, 0, 3), DeepEquals, rslc(y0odr(1, 2)))

    d1[6] = 2
    c.Check(heightOneRectsRgb(rs, d1, 0, 0, 3), DeepEquals, rslc(y0odr(1, 3)))

    d1[12] = 2
    c.Check(heightOneRectsRgb(rs, d1, 0, 0, 3), DeepEquals, rslc(y0odr(1, 3), y0odr(4,5)))
}
*/

/*
func (s *MySuite) BenchmarkGetHeightOneRects(c *C) {
    rs := make([]image.Rectangle, 80)
    sum := make(lnsumslc, 160)
    delta := make(deltaslc, 160)
    // p := profile.Start(profile.CPUProfile)
    lastrb := make([]byte, 160)
    t := int32(LAVGT)
    for i := 0; i < c.N; i++ {
        rb := randBytes(160)
        sum.rollSumDelta(rb, delta, lastrb)
        heightOneRects(rs, delta, 0, 0, t)
        lastrb = rb
    }
}
*/

func (s *MySuite) TestFindConnectedRects(c *C) {
	odrss := make([]RowRects, 5)
	odrss[1] = rslc(odr(1, 1, 2))
	c.Check(FindConnectedRects(5, odrss), DeepEquals, rslc(image.Rect(1, 1, 2, 2)))

	odrss[2] = rslc(odr(2, 2, 3))
	c.Check(FindConnectedRects(5, odrss), DeepEquals, rslc(image.Rect(1, 1, 3, 3)))

	odrss[4] = rslc(odr(4, 4, 5))
	c.Check(FindConnectedRects(5, odrss), DeepEquals, rslc(image.Rect(1, 1, 3, 3), image.Rect(4, 4, 5, 5)))

	odrss[1] = append(odrss[1], odr(1, 4, 5))
	c.Check(FindConnectedRects(5, odrss), DeepEquals, rslc(image.Rect(4, 1, 5, 2), image.Rect(1, 1, 3, 3), image.Rect(4, 4, 5, 5)))

	odrss[2] = append(odrss[2], odr(2, 4, 5))
	c.Check(FindConnectedRects(5, odrss), DeepEquals, rslc(image.Rect(1, 1, 3, 3), image.Rect(4, 1, 5, 3), image.Rect(4, 4, 5, 5)))

	odrss[3] = rslc(odr(3, 2, 3))
	c.Check(FindConnectedRects(5, odrss), DeepEquals, rslc(image.Rect(4, 1, 5, 3), image.Rect(1, 1, 3, 4), image.Rect(4, 4, 5, 5)))

	odrss[3] = append(odrss[3], odr(3, 4, 5))
	c.Check(FindConnectedRects(5, odrss), DeepEquals, rslc(image.Rect(1, 1, 3, 4), image.Rect(4, 1, 5, 5)))
}

func (s *MySuite) TestTracker(c *C) {
	rimg := imglib.NewRGB(image.Rect(0, 0, 4, 4))
	rimg.SetRGBA(1, 1, color.RGBA{1, 2, 3, 0xFF})
	rimg.SetRGBA(2, 2, color.RGBA{1, 2, 3, 0xFF})
	img := &imgseq.RawImg{PixelSequence: imglib.GetPixelSequence(rimg)}

	trk := NewTracker()
	for i := 0; i < LAVGN; i++ {
		c.Check(trk.GetRects(img, 12), DeepEquals, []image.Rectangle{})
	}

	c.Check(trk.frameNum, Equals, LAVGN)
	explns := make(lnsumslc, len(trk.longSums))
	explns[rimg.PixOffset(1, 1)+0] = 1 * LAVGN
	explns[rimg.PixOffset(1, 1)+1] = 2 * LAVGN
	explns[rimg.PixOffset(1, 1)+2] = 3 * LAVGN
	explns[rimg.PixOffset(2, 2)+0] = 1 * LAVGN
	explns[rimg.PixOffset(2, 2)+1] = 2 * LAVGN
	explns[rimg.PixOffset(2, 2)+2] = 3 * LAVGN
	c.Check(trk.longSums, DeepEquals, explns)

	c.Check(trk.GetRects(img, 12), DeepEquals, []image.Rectangle{})
	rimg = imglib.NewRGB(image.Rect(0, 0, 4, 4))
	emptyimg := &imgseq.RawImg{PixelSequence: imglib.GetPixelSequence(rimg)}
	c.Check(trk.GetRects(emptyimg, 12), DeepEquals, rslc(image.Rect(1, 1, 3, 3)))
}
