package imglib

import . "gopkg.in/check.v1"
import "image"
import "image/color"
import "image/draw"

func rgbRowToYuyvRow(rgbarow []color.RGBA, yuyvrow []color.YCbCr) {
	for i, crgba := range rgbarow {
		y, cb, cr := color.RGBToYCbCr(crgba.R, crgba.G, crgba.B)
		yuyvrow[i] = color.YCbCr{Y: y, Cb: cb, Cr: cr}
	}
}

func rgbToYuyv(rgb *RGB) *YUYV {
	yuyv := NewYUYV(rgb.Rect)
	rgbarow := make([]color.RGBA, rgb.Bounds().Dx())
	yuyvrow := make([]color.YCbCr, rgb.Bounds().Dx())
	for j := rgb.Rect.Min.Y; j < rgb.Rect.Max.Y; j++ {
		rgb.GetRow(j, rgbarow)
		rgbRowToYuyvRow(rgbarow, yuyvrow)
		yuyv.SetRow(j, yuyvrow)
	}
	return yuyv
}

func yuyvRowToRgbRow(yuyvrow []color.YCbCr, rgbarow []color.RGBA) {
	for i, cyuyv := range yuyvrow {
		r, g, b := color.YCbCrToRGB(cyuyv.Y, cyuyv.Cb, cyuyv.Cr)
		rgbarow[i] = color.RGBA{R: r, G: g, B: b, A: 0xFF}
	}
}

func yuyvToRgb(yuyv *YUYV) *RGB {
	rgb := NewRGB(yuyv.Rect)
	rgbarow := make([]color.RGBA, yuyv.Bounds().Dx())
	yuyvrow := make([]color.YCbCr, yuyv.Bounds().Dx())
	for j := yuyv.Rect.Min.Y; j < yuyv.Rect.Max.Y; j++ {
		yuyv.GetRow(j, yuyvrow)
		yuyvRowToRgbRow(yuyvrow, rgbarow)
		rgb.SetRow(j, rgbarow)
	}
	return rgb
}

func getTestRgbImage(dim image.Point) *RGB {
	img := NewRGB(image.Rect(0, 0, dim.X, dim.Y))
	p := 0
	for j := 0; j < dim.Y; j++ {
		for i := 0; i < dim.X; i++ {
			img.Pix[p+0] = uint8(i)
			img.Pix[p+1] = uint8(j)
			img.Pix[p+2] = uint8(i+j)
			p += 3
		}
	}
	return img
}

func getTestYuyvImage(dim image.Point) *YUYV {
	return rgbToYuyv(getTestRgbImage(dim))
}

func (s *MySuite) BenchmarkYuyvToRgbWithSetRow(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyvToRgb(yuyv)
	}
}

func (s *MySuite) BenchmarkYuyvToYCbCr(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyv.ToYCbCrMinZp()
	}
}

func (s *MySuite) BenchmarkDrawYCbCrToRGBA(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	ycbcr := yuyv.ToYCbCrMinZp()
	rgba := image.NewRGBA(yuyv.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		draw.Draw(rgba, rgba.Rect, ycbcr, image.ZP, draw.Src)
	}
}

func (s *MySuite) BenchmarkYuyvToRGBGeneric(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := make([]byte, (len(yuyv.Pix)*3)/2)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyv.ToRGBGeneric(dest, 0, 1, 2, 3)
	}
}

func (s *MySuite) BenchmarkYuyvToRGBMinZp(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := make([]byte, (len(yuyv.Pix)*3)/2)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyv.ToRGBMinZp(dest, 0, 1, 2, 3)
	}
}

func (s *MySuite) BenchmarkYuyvToRGB(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := make([]byte, (len(yuyv.Pix)*3)/2)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyv.ToRGB(dest)
	}
}

func (s *MySuite) TestYuyvBasics(c *C) {
	rgb := getTestRgbImage(image.Point{128, 128})
	yuyv := rgbToYuyv(rgb)
	c.Assert(yuyv.Rect, DeepEquals, image.Rect(0, 0, 128, 128))
	c.Check(yuyv.Stride, Equals, 128*2)
	c.Check(yuyv.PixOffset(1, 1), Equals, 128*2+2)
	c.Check(rgb.At(30, 60), DeepEquals, color.RGBA{0x1e, 0x3c, 0x5a, 0xFF})
	c.Check(yuyv.At(30, 60), DeepEquals, color.YCbCr{0x36, 0x94, 0x6f})
}

func (s *MySuite) TestToYCbCrMinZp(c *C) {
	yuyv := rgbToYuyv(getTestRgbImage(image.Point{128, 128}))
	ycbcr := yuyv.ToYCbCrMinZp()
	c.Check(yuyv, DeepEquals, NewYUYVFromYCbCrMinZP(ycbcr))
}

func drawToRgba(img image.Image) *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, img.Bounds().Dx(), img.Bounds().Dy()))
	draw.Draw(rgba, rgba.Rect, img, image.ZP, draw.Src)
	return rgba
}

func drawToRgb(img image.Image) *RGB {
	return NewRGBFromRGBADropAlpha(drawToRgba(img))
}

func (s *MySuite) TestYuyvDraw(c *C) {
	yuyv := rgbToYuyv(getTestRgbImage(image.Point{128, 128}))
	// I'm using the image produced by ToYCbCrMinZp as the reference image here
	// just because it's a pretty trivial implementation and most likely to be correct.
	rgba := drawToRgba(yuyv.ToYCbCrMinZp())
	c.Check(drawToRgba(yuyv), DeepEquals, rgba) // test YUYV.At()
}

func (s *MySuite) TestToRGB(c *C) {
	yuyv := rgbToYuyv(getTestRgbImage(image.Point{128, 128}))
	// I'm using the image produced by ToYCbCrMinZp as the reference image here
	// just because it's a pretty trivial implementation and most likely to be correct.
	rgb := drawToRgb(yuyv.ToYCbCrMinZp())
	pixbuf := make([]byte, len(rgb.Pix))
	yuyv.ToRGB(pixbuf)
	c.Check(pixbuf, DeepEquals, rgb.Pix)
}

func (s *MySuite) TestToRGBGeneric(c *C) {
	yuyv := rgbToYuyv(getTestRgbImage(image.Point{128, 128}))
	// I'm using the image produced by ToYCbCrMinZp as the reference image here
	// just because it's a pretty trivial implementation and most likely to be correct.
	rgb := drawToRgb(yuyv.ToYCbCrMinZp())
	pixbuf := make([]byte, len(rgb.Pix))
	yuyv.ToRGBGeneric(pixbuf, 0, 1, 2, 3)
	c.Check(pixbuf, DeepEquals, rgb.Pix)
}

func (s *MySuite) TestToRGBMinZP(c *C) {
	yuyv := rgbToYuyv(getTestRgbImage(image.Point{128, 128}))
	// I'm using the image produced by ToYCbCrMinZp as the reference image here
	// just because it's a pretty trivial implementation and most likely to be correct.
	rgb := drawToRgb(yuyv.ToYCbCrMinZp())
	pixbuf := make([]byte, len(rgb.Pix))
	yuyv.ToRGBMinZp(pixbuf, 0, 1, 2, 3)
	c.Check(pixbuf, DeepEquals, rgb.Pix)
}
