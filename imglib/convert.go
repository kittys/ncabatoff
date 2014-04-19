package imglib

import (
	"image"
	"image/color"
)

const (
	yuvBpp    = 2
	rgbBpp    = 3
	rgbaBpp    = 4
	rgba64Bpp  = 8
	nrgbaBpp   = 4
	nrgba64Bpp = 8
)

type Image interface {
	GetRGBA() *image.RGBA
	GetRGB() *RGB
}

// StdImage is a wrapper for the standard library's image.Image format.
type StdImage struct {
	image.Image
}

func (si StdImage) GetRGB() *RGB {
	return NewRGBFromRGBADropAlpha(si.GetRGBA())
}

func (si StdImage) GetRGBA() *image.RGBA {
	dest := image.NewRGBA(image.Rect(0, 0, si.Bounds().Dx(), si.Bounds().Dy()))
	switch concrete := si.Image.(type) {
	case *image.RGBA:
		// TODO copy?
		return concrete
	case *image.NRGBA:
		convertNRGBA(dest, concrete)
	case *image.NRGBA64:
		convertNRGBA64(dest, concrete)
	case *image.RGBA64:
		convertRGBA64(dest, concrete)
	case *image.YCbCr:
		convertYCbCr(dest, concrete)
	case *YUYV:
		convertYUYV(dest, concrete)
	case *RGB:
		convertRGB(dest, concrete)
	default:
		convertImageWithAt(dest, si.Image)
	}
	return dest
}

// convertImage converts any image implementing the image.Image interface to
// an RGBA type. This is *slow*.
func convertImageWithAt(dest *image.RGBA, src image.Image) {
	i := 0
	for y := src.Bounds().Min.Y; y < src.Bounds().Max.Y; y++ {
		for x := src.Bounds().Min.X; x < src.Bounds().Max.X; x++ {
			r, g, b, a := src.At(x, y).RGBA()
			dest.Pix[i+0] = uint8(r >> 8)
			dest.Pix[i+1] = uint8(g >> 8)
			dest.Pix[i+2] = uint8(b >> 8)
			dest.Pix[i+3] = uint8(a >> 8)
			i += 4
		}
	}
}

func convertYCbCr(dest *image.RGBA, src *image.YCbCr) {
	di := 0
	yi := src.YOffset(src.Rect.Min.X, src.Rect.Min.Y)
	ci := src.COffset(src.Rect.Min.X, src.Rect.Min.Y)
	yskip := src.YStride - src.Rect.Dx()
	cskip := src.CStride - src.Rect.Dx()/2
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		for x := src.Rect.Min.X; x < src.Rect.Max.X; x++ {
			yv := src.Y[yi]
			bv := src.Cb[ci]
			rv := src.Cr[ci]
			r, g, b := color.YCbCrToRGB(yv, bv, rv)
			dest.Pix[di+0] = r
			dest.Pix[di+1] = g
			dest.Pix[di+2] = b
			dest.Pix[di+3] = 0xff
			di += rgbaBpp
			yi++
			ci += (x % 2)
		}
		yi += yskip
		ci += cskip
	}
}

/* Reference implementation, about 50% slower than the unrolled version in use below.

func convertYUYV(dest *image.RGBA, src *YUYV) {
	di := 0
	si := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	skip := src.Stride - yuvBpp*(src.Rect.Dx())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		endi := si + src.Stride - skip
		for si < endi {
			y1, y2 := src.Pix[si+0], src.Pix[si+2]
			cb, cr := src.Pix[si+1], src.Pix[si+3]
			r1, g1, b1 := color.YCbCrToRGB(y1, cb, cr)
			r2, g2, b2 := color.YCbCrToRGB(y2, cb, cr)
			dest.Pix[di+0] = uint8(r1)
			dest.Pix[di+1] = uint8(g1)
			dest.Pix[di+2] = uint8(b1)
			dest.Pix[di+3] = 0xFF
			dest.Pix[di+4] = uint8(r2)
			dest.Pix[di+5] = uint8(g2)
			dest.Pix[di+6] = uint8(b2)
			dest.Pix[di+7] = 0xFF
			di += rgbaBpp*2
			si += yuvBpp*2
		}
		si += skip
	}
}
*/

func convertYUYV(dest *image.RGBA, src *YUYV) {
	di := 0
	si := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	skip := src.Stride - yuvBpp*(src.Rect.Dx())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		endi := si + src.Stride - skip
		for si < endi {
			yy1, yy2 := int(src.Pix[si+0])<<16+1<<15, int(src.Pix[si+2])<<16+1<<15
			cb1, cr1 := int(src.Pix[si+1])-128, int(src.Pix[si+3])-128
			t1, t2, t3, t4 := 91881*cr1, 46802*cr1, 22554*cb1, 116130*cb1

			r1, g1, b1 := (yy1+t1)>>16, (yy1-t3-t2)>>16, (yy1+t4)>>16
			r2, g2, b2 := (yy2+t1)>>16, (yy2-t3-t2)>>16, (yy2+t4)>>16
			if r1 < 0 {
				r1 = 0
			} else if r1 > 255 {
				r1 = 255
			}
			if g1 < 0 {
				g1 = 0
			} else if g1 > 255 {
				g1 = 255
			}
			if b1 < 0 {
				b1 = 0
			} else if b1 > 255 {
				b1 = 255
			}
			if r2 < 0 {
				r2 = 0
			} else if r2 > 255 {
				r2 = 255
			}
			if g2 < 0 {
				g2 = 0
			} else if g2 > 255 {
				g2 = 255
			}
			if b2 < 0 {
				b2 = 0
			} else if b2 > 255 {
				b2 = 255
			}

			dest.Pix[di+0] = uint8(r1)
			dest.Pix[di+1] = uint8(g1)
			dest.Pix[di+2] = uint8(b1)
			dest.Pix[di+3] = 0xFF
			dest.Pix[di+4] = uint8(r2)
			dest.Pix[di+5] = uint8(g2)
			dest.Pix[di+6] = uint8(b2)
			dest.Pix[di+7] = 0xFF

			di += rgbaBpp*2
			si += yuvBpp*2
		}
		si += skip
	}
}

func convertRGBA64(dest *image.RGBA, src *image.RGBA64) {
	di := 0
	si := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	skip := src.Stride - rgba64Bpp*(src.Rect.Dx())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		endi := si + src.Stride - skip
		for si < endi {
			dest.Pix[di+0] = src.Pix[si+0]
			dest.Pix[di+1] = src.Pix[si+2]
			dest.Pix[di+2] = src.Pix[si+4]
			dest.Pix[di+3] = src.Pix[si+6]
			di += rgbaBpp
			si += rgba64Bpp
		}
		si += skip
	}

}

func convertNRGBA(dest *image.RGBA, src *image.NRGBA) {
	di := 0
	si := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	skip := src.Stride - nrgbaBpp*(src.Rect.Dx())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		endi := si + src.Stride - skip
		for si < endi {
			a := uint16(src.Pix[si+3])
			dest.Pix[di+0] = uint8((uint16(src.Pix[si+0]) * a) / 0xff)
			dest.Pix[di+1] = uint8((uint16(src.Pix[si+1]) * a) / 0xff)
			dest.Pix[di+2] = uint8((uint16(src.Pix[si+2]) * a) / 0xff)
			dest.Pix[di+3] = src.Pix[si+3]
			di += rgbaBpp
			si += nrgbaBpp
		}
		si += skip
	}
}

func convertNRGBA64(dest *image.RGBA, src *image.NRGBA64) {
	di := 0
	si := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	skip := src.Stride - nrgba64Bpp*(src.Rect.Dx())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		endi := si + src.Stride - skip
		for si < endi {
			a := uint16(src.Pix[si+6])
			dest.Pix[di+0] = uint8((uint16(src.Pix[si+0]) * a) / 0xff)
			dest.Pix[di+1] = uint8((uint16(src.Pix[si+2]) * a) / 0xff)
			dest.Pix[di+2] = uint8((uint16(src.Pix[si+4]) * a) / 0xff)
			dest.Pix[di+3] = src.Pix[si+6]
			di += rgbaBpp
			si += nrgba64Bpp
		}
		si += skip
	}
}

// convertRGB builds RGBA by providing 0xFF for the alpha channel.
func convertRGB(dest *image.RGBA, src *RGB) {
	di := 0
	si := src.PixOffset(src.Rect.Min.X, src.Rect.Min.Y)
	skip := src.Stride - rgbBpp*(src.Rect.Dx())
	for y := src.Rect.Min.Y; y < src.Rect.Max.Y; y++ {
		endi := si + src.Stride - skip
		for si < endi {
			dest.Pix[di+0] = src.Pix[si+0]
			dest.Pix[di+1] = src.Pix[si+1]
			dest.Pix[di+2] = src.Pix[si+2]
			dest.Pix[di+3] = 0xFF
			di += rgbaBpp
			si += rgbBpp
		}
		si += skip
	}
}
