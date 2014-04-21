package imglib

import "fmt"
import "image"
import "image/color"
import "io"
import "io/ioutil"
import "os"

// Each pixel pair is represented by four bytes.
const yuyvBytesPP = 2

// A YUYV is a packed image format also known as YUY2 and YUV 4:2:2.
// *YUYV implements image.Image.
// This happens to be the native format used by the PS3 Eye webcam.
// Typically you don't want to work with the images in this format,
// you just want to be able to convert to and from them.
type YUYV struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

// NewYUYV returns a new blank YUYV with the given bounds.
func NewYUYV(r image.Rectangle) *YUYV {
	w, h := r.Dx(), r.Dy()
	buf := make([]uint8, yuyvBytesPP*w*h)
	return &YUYV{buf, yuyvBytesPP * w, r}
}

// ColorModel returns image/color.YCbCrModel, which I think is not quite
// right but seems to work mostly.
func (img *YUYV) ColorModel() color.Model {
	return color.YCbCrModel
}

// Bounds returns the bounding rectangle.
func (img *YUYV) Bounds() image.Rectangle {
	return img.Rect
}

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).  This doesn't have as well-defined a meaning as the
// image formats in the Go standard library, since the Y component is shared
// between adjacent pixels.
func (img *YUYV) PixOffset(x, y int) int {
	return (y-img.Rect.Min.Y)*img.Stride + (x-img.Rect.Min.X)*yuyvBytesPP
}

// At returns the pixel at (x,y).  While the operation itself is quite fast,
// converting the returned value to another colorspace like RGB is not, and
// reading a large number of pixels is probably better done using GetRow() or
// just by converting the entire image.
func (img *YUYV) At(x, y int) color.Color {
	if !(image.Point{x, y}.In(img.Rect)) {
		return color.YCbCr{}
	}
	i := img.PixOffset(x, y)
	ret := color.YCbCr{Y: img.Pix[i]}
	if x%2 == 0 {
		ret.Cb, ret.Cr = img.Pix[i+1], img.Pix[i+3]
	} else {
		ret.Cb, ret.Cr = img.Pix[i-1], img.Pix[i+1]
	}
	return ret
}

// GetBytesPerPixel returns the number of bytes per pixel, although note that
// you can't store a single pixel in this format.
func (img *YUYV) GetBytesPerPixel() int {
	return yuyvBytesPP
}

// GetBytesPerChunk returns the number of bytes per chunk, where a chunk is the
// minimum size that can be worked with (two pixels in this case.)
func (img *YUYV) GetBytesPerChunk() int {
	return 4
}

// GetStride returns the number of bytes used per row of pixels.
func (img *YUYV) GetStride() int {
	return img.Stride
}

// NewYUYVFromYCbCrMinZP returns a new YUYV using img as input.  This is
// a relatively efficient conversion.
func NewYUYVFromYCbCrMinZP(img *image.YCbCr) *YUYV {
	if img.Rect.Min != image.ZP {
		panic("ToYCbCrMinZp does not work for subimages")
	}
	ret := NewYUYV(img.Rect)
	p := 0
	for i := 0; i < len(img.Y); i += 2 {
		ret.Pix[p+0] = img.Y[i]
		ret.Pix[p+1] = img.Cb[i/2]
		ret.Pix[p+2] = img.Y[i+1]
		ret.Pix[p+3] = img.Cr[i/2]
		p += 4
	}
	return ret
}

// ToYCbCrMinZp returns a new image.YCbCr by converting from img.  This is a relatively efficient conversion.
func (img *YUYV) ToYCbCrMinZp() *image.YCbCr {
	if img.Rect.Min != image.ZP {
		panic("ToYCbCrMinZp does not work for subimages")
	}
	ycbcr := image.NewYCbCr(img.Rect, image.YCbCrSubsampleRatio422)
	i, icbcr, iy := 0, 0, 0
	for iy < len(ycbcr.Y) {
		y1, b, y2, r := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
		ycbcr.Cb[icbcr] = b
		ycbcr.Cr[icbcr] = r
		ycbcr.Y[iy] = y1
		ycbcr.Y[iy+1] = y2
		icbcr++
		iy += 2
		i += 4
	}

	return ycbcr
}

// GetRow stores row y as a slice of color.YCbCr in dest, which the caller must
// allocate.  This is more efficent than using At() but less efficient than
// doing a full conversion, assuming you're going to look at most of the pixels
// in the image.
func (img *YUYV) GetRow(y int, dest interface{}) {
	width := img.Rect.Dx()
	ret := dest.([]color.YCbCr)[:0]
	i := img.PixOffset(img.Rect.Min.X, y)
	end := i + width*yuyvBytesPP

	for i < end {
		y1, b, y2, r := img.Pix[i], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
		ret = append(ret, color.YCbCr{y1, b, r}, color.YCbCr{y2, b, r})
		i += 4
	}
}

// SetRow fills row y using the slice of color.YCbCr in src.  As with GetRow(), this is
// better than making many calls to At() but still somewhat slower than doing a full
// conversion.
func (img *YUYV) SetRow(y int, src interface{}) {
	pix := img.Pix[img.PixOffset(img.Rect.Min.X, y):]
	o := 0

	cols := src.([]color.YCbCr)
	for i := 0; i < len(cols); i += 2 {
		c1, c2 := cols[i], cols[i+1]
		pix[o], pix[o+1], pix[o+2], pix[o+3] = c1.Y, c1.Cb|c2.Cb, c2.Y, c1.Cr|c2.Cr
		o += 4
	}
}

// ToRGBGeneric is a fairly fast generic convertor to arbitrary packed RGB
// formats (e.g. works for BGR).  or, og, and ob give the offset of the R, G,
// and B components respectively.  xs is the 'x stride', a.k.a. how many bytes
// the desired format should use per pixel.  If xs==4, the fourth byte of each
// pixel will be set to 0xFF.  Only values of xs==3 and xs==4 are supported.
func (img *YUYV) ToRGBGeneric(dest []byte, or, og, ob, xs int) {
	if xs != 3 && xs != 4 {
		panic("only supports xs==3 and xs==4")
	}
	j := 0
	row := make([]color.YCbCr, img.Rect.Dx())
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		img.GetRow(y, row)
		for _, v := range row {
			dest[j+or], dest[j+og], dest[j+ob] = color.YCbCrToRGB(v.Y, v.Cb, v.Cr)
			if xs == 4 {
				dest[j+3] = 0xFF
			}
			j += xs
		}
	}
}

// ToRGBMinZp adheres to the same interfaces as ToRGBGeneric, but
// assumes img.Rect.Min == image.ZP and is a bit faster.
func (img *YUYV) ToRGBMinZp(dest []byte, or, og, ob, xs int) {
	if xs != 3 && xs != 4 {
		panic("only supports xs==3 and xs==4")
	}
	if img.Rect.Min != image.ZP {
		panic("ToRGBMinZp only supports images with Rect.Min = (0,0)")
	}

	j := 0
	for i := 0; i+3 < len(img.Pix); i += 4 {
		y1, cb, y2, cr := img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], img.Pix[i+3]
		r1, g1, b1 := color.YCbCrToRGB(y1, cb, cr)
		r2, g2, b2 := color.YCbCrToRGB(y2, cb, cr)
		dest[j+or], dest[j+og], dest[j+ob] = uint8(r1), uint8(g1), uint8(b1)
		dest[j+xs+or], dest[j+xs+og], dest[j+xs+ob] = uint8(r2), uint8(g2), uint8(b2)
		if xs == 4 {
			dest[j+3] = 0xFF
			dest[j+xs+3] = 0xFF
		}
		j += xs + xs
	}
}

// SubImage returns an image representing the portion of the image img visible
// through r. The returned value shares pixels with the original image.
func (img *YUYV) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(img.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &YUYV{}
	}
	i := img.PixOffset(r.Min.X, r.Min.Y)
	return &YUYV{
		Pix:    img.Pix[i:],
		Stride: img.Stride,
		Rect:   r,
	}
}

// StrictSubImage is like SubImage except that the returned *YUYV will have a Pix field
// fully constrained by r.  SubImage allows Pix to extend all the way to the end of img.Pix,
// which seems wrong to me but it's what the standard library does in its SubImage
// implementations so maybe I'm wrong...
func (img *YUYV) StrictSubImage(r image.Rectangle) image.Image {
	r = r.Intersect(img.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &YUYV{}
	}
	i := img.PixOffset(r.Min.X, r.Min.Y)
	endi := img.PixOffset(r.Max.X, r.Max.Y)
	return &YUYV{
		Pix:    img.Pix[i:endi],
		Stride: img.Stride,
		Rect:   r,
	}
}

// Opaque scans the entire image and returns whether or not it is fully opaque.
func (img *YUYV) Opaque() bool {
	return true
}

// StoreRaw dumps the Pix to path.
func (img *YUYV) StoreRaw(path string) error {
	return ioutil.WriteFile(path, img.Pix, 0644)
}

func (img *YUYV) loadRaw(file *os.File) error {
	_, err := io.ReadFull(file, img.Pix)
	return err
}

// LoadRaw reads path to populate img.Pix, returning an error if the open or read
// returned an error or if the read yields fewer than len(img.Pix) bytes.
func (img *YUYV) LoadRaw(path string) error {
	if file, err := os.Open(path); err != nil {
		return err
	} else {
		defer file.Close()
		return img.loadRaw(file)
	}
}

func NewYUYVFromFile(path string) (*YUYV, error) {
	var yuyv *YUYV
	if file, err := os.Open(path); err != nil {
		return nil, err
	} else if fi, err := file.Stat(); err != nil {
		return nil, err
	} else {
		r := image.Rectangle{}
		switch fi.Size() / 2 {
		case 640 * 480:
			r.Max = image.Point{640, 480}
		case 320 * 240:
			r.Max = image.Point{320, 240}
		default:
			return nil, fmt.Errorf("unknown dims, filesize=%d", fi.Size())
		}

		yuyv = NewYUYV(r)
		yuyv.loadRaw(file)
		return yuyv, nil
	}
}
