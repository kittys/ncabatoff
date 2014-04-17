package imglib

import "io/ioutil"
import "io"
import "os"
import "image"
import "image/color"

// A RGB is a packed image format with one byte per color channel,
// the colors ordered by (R,G,B).  It's an image.RGBA without the 'A'.
type RGB struct {
    Pix []uint8
    Stride int
    Rect image.Rectangle
}

// NewRGB returns a new RGB with the given bounds.
func NewRGB(r image.Rectangle) *RGB {
	w, h := r.Dx(), r.Dy()
	buf := make([]uint8, 3*w*h)
	return &RGB{buf, 3 * w, r}
}

// ColorModel returns image/color.RGBAModel, which I think is not quite
// right but seems to work mostly.
func (img *RGB) ColorModel() color.Model { return color.RGBAModel }

// Bounds returns the bounding rectangle.
func (img *RGB) Bounds() image.Rectangle { return img.Rect }

// PixOffset returns the index of the first element of Pix that corresponds to
// the pixel at (x, y).
func (img *RGB) PixOffset(x, y int) int {
	return (y-img.Rect.Min.Y)*img.Stride + (x-img.Rect.Min.X)*3
}

// At returns the pixel at (x,y).  While the operation itself is quite fast,
// converting the returned value to another colorspace is not, and reading a
// large number of pixels is probably better done using GetRow() or just by
// looking at the Pix array directly.
func (img *RGB) At(x, y int) color.Color {
	if !(image.Point{x, y}.In(img.Rect)) {
		return color.RGBA{}
	}
	i := img.PixOffset(x, y)
	return color.RGBA{img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], 0xFF}
}

// GetBytesPerPixel returns the number of bytes per pixel, although note that
// you can't store a single pixel in this format.
func (img *RGB) GetBytesPerPixel() int {
	return 3
}

// GetBytesPerChunk returns the number of bytes per chunk, where a chunk is the
// minimum size that can be worked with (one pixel in this case.)
func (img *RGB) GetBytesPerChunk() int {
	return 3
}

// GetStride returns the number of bytes used per row of pixels.
func (img *RGB) GetStride() int {
	return img.Stride
}

// Set assigns the pixel at (x,y) the color c.
func (img *RGB) Set(x, y int, c color.Color) {
	if !(image.Point{x, y}.In(img.Rect)) {
		return
	}
	i := img.PixOffset(x, y)
	c1 := color.RGBAModel.Convert(c).(color.RGBA)
	img.Pix[i+0] = c1.R
	img.Pix[i+1] = c1.G
	img.Pix[i+2] = c1.B
}

// SetRGBA assigns the pixel at (x,y) the color c.  This is faster than
// Set since it doesn't need to do a colorspace conversion.
func (img *RGB) SetRGBA(x, y int, c color.RGBA) {
	if !(image.Point{x, y}.In(img.Rect)) {
		return
	}
	i := img.PixOffset(x, y)
	img.Pix[i+0] = c.R
	img.Pix[i+1] = c.G
	img.Pix[i+2] = c.B
}

// GetRow stores row y as a slice of color.RGBA in dest, which the caller must
// allocate.  This is more efficent than using At() but less efficient than
// doing a full conversion, assuming you're going to look at most of the pixels
// in the image.
func (img *RGB) GetRow(y int, dest interface{}) {
    ret := dest.([]color.RGBA)[:0]
    start := img.PixOffset(0, y)
    end := start + img.Stride
    for i := start; i < end; i += 3 {
        ret = append(ret, color.RGBA{img.Pix[i+0], img.Pix[i+1], img.Pix[i+2], 0xFF})
    }
}

// SetRow fills row y using the slice of color.RGBA in src.  As with GetRow(), this is
// better than making many calls to At() but still somewhat slower than doing a full
// conversion.
func (img *RGB) SetRow(y int, src interface{}) {
	pix := img.Pix[img.PixOffset(img.Rect.Min.X, y):]
	o := 0

	cols := src.([]color.RGBA)
	for _, c := range cols {
		pix[o+0] = c.R
		pix[o+1] = c.G
		pix[o+2] = c.B
		o += 3
	}
}

// SubImage returns an image representing the portion of the image visible
// through r. The returned value shares pixels with the original image.
func (img *RGB) SubImage(r image.Rectangle) image.Image {
	r = r.Intersect(img.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &RGB{}
	}
	i := img.PixOffset(r.Min.X, r.Min.Y)
	return &RGB{
		Pix:    img.Pix[i:],
		Stride: img.Stride,
		Rect:   r,
	}
}

// StrictSubImage is like SubImage except that the returned *RGB will have a Pix field
// fully constrained by r.  SubImage allows Pix to extend all the way to the end of img.Pix,
// which seems wrong to me but it's what the standard library does in its SubImage
// implementations so maybe I'm wrong...
func (img *RGB) StrictSubImage(r image.Rectangle) image.Image {
	r = r.Intersect(img.Rect)
	// If r1 and r2 are Rectangles, r1.Intersect(r2) is not guaranteed to be inside
	// either r1 or r2 if the intersection is empty. Without explicitly checking for
	// this, the Pix[i:] expression below can panic.
	if r.Empty() {
		return &RGB{}
	}
	i := img.PixOffset(r.Min.X, r.Min.Y)
	endi := img.PixOffset(r.Max.X, r.Max.Y)
	return &RGB{
		Pix:    img.Pix[i:endi],
		Stride: img.Stride,
		Rect:   r,
	}
}

// Opaque scans the entire image and returns whether or not it is fully opaque.
func (img *RGB) Opaque() bool {
	return true
}

// StoreRaw dumps the Pix to path.
func (img *RGB) StoreRaw(path string) error {
	return ioutil.WriteFile(path, img.Pix, 0644)
}

// LoadRaw reads path to populate Pix, returning an error if the open or read
// returned an error or if the read yields fewer than len(Pix) bytes.
func (img *RGB) LoadRaw(path string) error {
	if file, err := os.Open(path); err != nil {
		return err
	} else {
		defer func(f *os.File) { f.Close() }(file)
		_, err := io.ReadFull(file, img.Pix)
		return err
	}
}

// ToRGBA returns an RGBA image built from RGB by providing 0xFF for the alpha channel.
func (img *RGB) ToRGBA() *image.RGBA {
	rgba := image.NewRGBA(image.Rect(0, 0, img.Rect.Dx(), img.Rect.Dy()))
	po := 0
	pi := img.PixOffset(img.Rect.Min.X, img.Rect.Min.Y)
	skip := img.Stride - 3*(img.Rect.Dx())
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		endi := pi + 3*img.Rect.Dx()
		for pi < endi {
			rgba.Pix[po+0] = img.Pix[pi+0]
			rgba.Pix[po+1] = img.Pix[pi+1]
			rgba.Pix[po+2] = img.Pix[pi+2]
			rgba.Pix[po+3] = 0xFF
			po += 4
			pi += 3
		}
		pi += skip
	}
	return rgba
}

// NewRGBFromRGBA returns an RGB image built from rgba by discarding the alpha channel.
func NewRGBFromRGBADropAlpha(rgba *image.RGBA) *RGB {
	rgb := NewRGB(image.Rect(0, 0, rgba.Rect.Dx(), rgba.Rect.Dy()))
	po := 0
	pi := rgba.PixOffset(rgba.Rect.Min.X, rgba.Rect.Min.Y)
	skip := rgba.Stride - 4*(rgba.Rect.Dx())
	for y := rgba.Rect.Min.Y; y < rgba.Rect.Max.Y; y++ {
		endi := pi + 4*rgba.Rect.Dx()
		for pi < endi {
			rgb.Pix[po+0] = rgba.Pix[pi+0]
			rgb.Pix[po+1] = rgba.Pix[pi+1]
			rgb.Pix[po+2] = rgba.Pix[pi+2]
			pi += 4
			po += 3
		}
		pi += skip
	}
	return rgb
}
