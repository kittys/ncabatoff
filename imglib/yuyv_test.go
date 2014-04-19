package imglib

import . "gopkg.in/check.v1"
import "image"
import "image/color"
import "image/draw"
import (
// "github.com/BurntSushi/xgbutil"
// "github.com/BurntSushi/xgbutil/xgraphics"
)

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

func getTestRgba64Image(dim image.Point) *image.RGBA64 {
	rgba64 := image.NewRGBA64(image.Rect(0, 0, dim.X, dim.Y))
	rgb := getTestRgbImage(dim)
	draw.Draw(rgba64, rgba64.Rect, rgb, image.ZP, draw.Src)
	return rgba64
}

func getTestYuyvImage(dim image.Point) *YUYV {
	return rgbToYuyv(getTestRgbImage(dim))
}

// Slowest way to convert, using At() in x-y order.  Depending on your CPU's
// ability to recognize memory access patterns and/or your cache size, it may
// be no slower than going in y-x order.  For example, on a Core i3 I see no
// difference, even if I make the image much bigger.
func (s *MySuite) BenchmarkYuyvToRgbWithAtOrderXy(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := NewRGB(yuyv.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		convertImageXy(dest, yuyv)
	}
}

// Slowest sane way to convert, using At() in y-x order.  About 2x slower
// than BenchmarkYuyvToRgbWithGetRowSetRow.
func (s *MySuite) BenchmarkYuyvToRgbWithAtOrderYx(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := NewRGB(yuyv.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		convertImageYx(dest, yuyv)
	}
}

// convertImage converts any image implementing the image.Image interface to
// an RGBA type. This is *slow*, and since we do this in the wrong order, even slower than
// convertImageYx.
func convertImageXy(dest *RGB, src image.Image) {
	i := 0
	for x := dest.Rect.Min.X; x < dest.Rect.Max.X; x++ {
		for y := dest.Rect.Min.Y; y < dest.Rect.Max.Y; y++ {
			r, g, b, _ := src.At(x, y).RGBA()
			dest.Pix[i+0] = uint8(r >> 8)
			dest.Pix[i+1] = uint8(g >> 8)
			dest.Pix[i+2] = uint8(b >> 8)
			i += 3
		}
	}
}

// convertImage converts any image implementing the image.Image interface to
// an RGBA type. This is *slow*.
func convertImageYx(dest *RGB, src image.Image) {
	i := 0
	for y := dest.Rect.Min.Y; y < dest.Rect.Max.Y; y++ {
		for x := dest.Rect.Min.X; x < dest.Rect.Max.X; x++ {
			r, g, b, _ := src.At(x, y).RGBA()
			dest.Pix[i+0] = uint8(r >> 8)
			dest.Pix[i+1] = uint8(g >> 8)
			dest.Pix[i+2] = uint8(b >> 8)
			i += 3
		}
	}
}

// Next slowest way to convert: yuyv.GetRow/rgb.SetRow, about 2x slower than 
// BenchmarkYuyvToRGBGeneric or BenchmarkYuyvToYCbCr+BenchmarkDrawYCbCrToRGBA.
func (s *MySuite) BenchmarkYuyvToRgbWithGetRowSetRow(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyvToRgb(yuyv)
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

func (s *MySuite) BenchmarkYuyvToYCbCr(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyv.ToYCbCrMinZp()
	}
}

// Impressively, converting to YCbCr and then using Draw to set an RGBA image
// is only 2x as slow as the fastest method.
func (s *MySuite) BenchmarkDrawYCbCrToRGBA(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	ycbcr := yuyv.ToYCbCrMinZp()
	rgba := image.NewRGBA(yuyv.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		draw.Draw(rgba, rgba.Rect, ycbcr, image.ZP, draw.Src)
	}
}

// This is barely any faster than using ToYCbCrMinZp+draw.Draw, though it does
// work with subimages unlike ToYCbCrMinZp.  It's also better if you want something
// other than RGB, e.g. BGR.
func (s *MySuite) BenchmarkYuyvToRGBGeneric(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := make([]byte, (len(yuyv.Pix)*3)/2)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyv.ToRGBGeneric(dest, 0, 1, 2, 3)
	}
}

// ToRGBMinZp gives about a 25% speedup over ToRGBGeneric, but doesn't work with subimages.
func (s *MySuite) BenchmarkYuyvToRGBMinZp(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := make([]byte, (len(yuyv.Pix)*3)/2)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		yuyv.ToRGBMinZp(dest, 0, 1, 2, 3)
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

func (s *MySuite) TestConvertToRGBA(c *C) {
	yuyv := rgbToYuyv(getTestRgbImage(image.Point{128, 128}))
	// I'm using the image produced by ToYCbCrMinZp as the reference image here
	// just because it's a pretty trivial implementation and most likely to be correct.
	rgba := drawToRgba(yuyv.ToYCbCrMinZp())
	dest := image.NewRGBA(yuyv.Rect)
	convertYUYV(dest, yuyv)
	c.Check(dest, DeepEquals, rgba)
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

func (s *MySuite) BenchmarkConvertYuv(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	dest := image.NewRGBA(yuyv.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		convertYUYV(dest, yuyv)
	}
}

func (s *MySuite) BenchmarkConvertRGB(c *C) {
	rgb := getTestRgbImage(image.Point{128, 128})
	dest := image.NewRGBA(rgb.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		convertRGB(dest, rgb)
	}
}

func (s *MySuite) BenchmarkConvertRGBA64(c *C) {
	rgba64 := getTestRgba64Image(image.Point{128, 128})
	dest := image.NewRGBA(rgba64.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		convertRGBA64(dest, rgba64)
	}
}

/* On an i3 this is about 1.5x slower than convertRGBA64 despite writing only
3/4 as many bytes, presumably because it's doing two PixOffset calls per pixel.

func (s *MySuite) BenchmarkXgbUtilConvertRGBA64(c *C) {
	X, _ := xgbutil.NewConn()
	rgba64 := getTestRgba64Image(image.Point{128, 128})
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		xgraphics.NewConvert(X, rgba64)
	}
}
*/

func (s *MySuite) BenchmarkConvertYcbcr(c *C) {
	yuyv := getTestYuyvImage(image.Point{128, 128})
	ycbcr := yuyv.ToYCbCrMinZp()
	dest := image.NewRGBA(yuyv.Rect)
	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		convertYCbCr(dest, ycbcr)
	}
}
