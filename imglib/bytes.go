package imglib

import (
	"image"
)

// ImageBytes is a wrapper interface for []byte that tells us something
// about how to interpret them as pixels.
type ImageBytes interface {
	GetBytes() []byte
	GetBytesPerPixel() int
}

type RgbBytes []byte

func (rb RgbBytes) GetBytes() []byte {
	return []byte(rb)
}
func (rb RgbBytes) GetBytesPerPixel() int {
	return 3
}

type RgbaBytes []byte

func (rb RgbaBytes) GetBytes() []byte {
	return []byte(rb)
}
func (rb RgbaBytes) GetBytesPerPixel() int {
	return 4
}

type YuyvBytes []byte

func (yb YuyvBytes) GetBytes() []byte {
	return []byte(yb)
}
func (yb YuyvBytes) GetBytesPerPixel() int {
	return 2
}

func GetImageBytes(img image.Image) ImageBytes {
	switch raw := img.(type) {
	case *YUYV:
		return YuyvBytes(raw.Pix)
	case *RGB:
		return RgbBytes(raw.Pix)
	case *image.RGBA:
		return RgbaBytes(raw.Pix)
		// should we use a narrower argument type than image.Image?
	default: panic("Unknown image type")
	}
}

// PixelSequence is like image.Image, only non-SubImage-able in the interest of speed.
type PixelSequence struct {
	ImageBytes
	Dx, Dy int
}

func (ps PixelSequence) GetStride() int {
	return ps.Dx * ps.ImageBytes.GetBytesPerPixel()
}

func (ps PixelSequence) PixOffset(x, y int) int {
	return y*ps.GetStride() + x*ps.ImageBytes.GetBytesPerPixel()
}

func GetPixelSequence(img image.Image) PixelSequence {
	return PixelSequence{Dx: img.Bounds().Dx(), Dy: img.Bounds().Dy(), ImageBytes: GetImageBytes(img)}
}

// PixelRow represents a single row from a PixelSequence.
type PixelRow struct {
	PixelSequence
	Offset int
}

func (ps PixelRow) GetBytes() []byte {
	start, end := ps.Offset, ps.Offset+ps.GetStride()
	return ps.PixelSequence.ImageBytes.GetBytes()[start:end]
}
