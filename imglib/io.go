package imglib

import "path/filepath"
import "image"
import "os"
import "fmt"

func LoadPixelSequence(path string) (PixelSequence, error) {
	switch(filepath.Ext(path)) {
	case ".yuv":
		if yuyv, err := NewYUYVFromFile(path); err != nil {
			return PixelSequence{}, err
		} else {
			return GetPixelSequence(yuyv), nil
		}
	case ".rgb":
		if rgb, err := NewRGBFromFile(path); err != nil {
			return PixelSequence{}, err
		} else {
			return GetPixelSequence(rgb), nil
		}
	}
	return PixelSequence{}, fmt.Errorf("can't load file '%s', unknown format", path)
}

// Read and possibly convert or decode the input file
func LoadImage(path string) (image.Image, error) {
	if filepath.Ext(path) == ".yuv" {
		if yuyv, err := NewYUYVFromFile(path); err != nil {
			return nil, err
		} else {
			return yuyv, nil
		}
	}

	if file, err := os.Open(path); err != nil {
		return nil, err
	} else {
		defer file.Close()
		img, _, err := image.Decode(file)
		return img, err
	}
}

func guessRect(numpix int) *image.Rectangle {
	var r image.Rectangle
	switch numpix {
	case 1280 * 720:
		r.Max = image.Point{1280, 720}
	case 640 * 480:
		r.Max = image.Point{640, 480}
	case 320 * 240:
		r.Max = image.Point{320, 240}
	default:
		return nil
	}
	return &r
}

func NewYUYVFromFile(path string) (*YUYV, error) {
	if fi, err := os.Stat(path); err != nil {
		return nil, err
	} else {
		bpp := YuyvBytes{}.GetBytesPerPixel()
		if r := guessRect(int(fi.Size())/bpp); r == nil {
			return nil, fmt.Errorf("unknown dims, filesize=%d", fi.Size())
		} else {
			yuyv := NewYUYV(*r)
			if err := yuyv.LoadRaw(path); err != nil {
				return nil, err
			} else {
				return yuyv, nil
			}
		}
	}
}

func NewRGBFromFile(path string) (*RGB, error) {
	if fi, err := os.Stat(path); err != nil {
		return nil, err
	} else {
		bpp := RgbBytes{}.GetBytesPerPixel()
		if r := guessRect(int(fi.Size())/bpp); r == nil {
			return nil, fmt.Errorf("unknown dims, filesize=%d", fi.Size())
		} else {
			rgb := NewRGB(*r)
			if err := rgb.LoadRaw(path); err != nil {
				return nil, err
			} else {
				return rgb, nil
			}
		}
	}
}
