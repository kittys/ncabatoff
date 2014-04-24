package imglib

import (
	"image"
	"fmt"
)

// ImageBytes is a wrapper interface for []byte that tells us something
// about how to interpret them as pixels.
type ImageBytes interface {
	GetBytes() []byte
	GetBytesPerPixel() int
	AsImage(width int) image.Image
}

type RgbBytes []byte

func getRect(ib ImageBytes, width int) image.Rectangle {
	pixcount := len(ib.GetBytes()) / ib.GetBytesPerPixel()
	height := pixcount/width
	return image.Rect(0, 0, width, height)
}

func (rgb RgbBytes) GetBytes() []byte {
	return []byte(rgb)
}
func (rgb RgbBytes) GetBytesPerPixel() int {
	return 3
}
func (rgb RgbBytes) AsImage(width int) image.Image {
	return &RGB{rgb.GetBytes(), width * rgb.GetBytesPerPixel(), getRect(rgb, width)}
}
func (rgb RgbBytes) String() string {
	return fmt.Sprintf("RgbBytes[%d]", len(rgb))
}

type RgbaBytes []byte

func (rgba RgbaBytes) GetBytes() []byte {
	return []byte(rgba)
}
func (rgba RgbaBytes) GetBytesPerPixel() int {
	return 4
}
func (rgba RgbaBytes) AsImage(width int) image.Image {
	return &image.RGBA{rgba.GetBytes(), width * rgba.GetBytesPerPixel(), getRect(rgba, width)}
}
func (rgba RgbaBytes) String() string {
	return fmt.Sprintf("RgbaBytes[%d]", len(rgba))
}

type YuyvBytes []byte

func (yuyv YuyvBytes) GetBytes() []byte {
	return []byte(yuyv)
}
func (yuyv YuyvBytes) GetBytesPerPixel() int {
	return 2
}
func (yuyv YuyvBytes) AsImage(width int) image.Image {
	return &YUYV{yuyv.GetBytes(), width * yuyv.GetBytesPerPixel(), getRect(yuyv, width)}
}
func (yuyv YuyvBytes) String() string {
	return fmt.Sprintf("YuyvBytes[%d]", len(yuyv))
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

func (ps PixelSequence) GetImage() image.Image {
	return ps.AsImage(ps.Dx)
}
