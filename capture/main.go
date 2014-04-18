package main

import (
	// "bytes"
	"code.google.com/p/ncabatoff/imglib"
	"code.google.com/p/ncabatoff/v4l"
	"flag"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"time"
)

// import "github.com/davecheney/profile"

type simage struct {
	seq int
	tm  time.Time
	img image.Image
}

var flagInput = flag.String("in", "/dev/video0", "input capture device")
var flagNumBufs = flag.Int("numbufs", 15, "number of mmap buffers")
var flagWidth = flag.Int("width", 640, "width in pixels")
var flagHeight = flag.Int("height", 480, "height in pixels")
var flagFormat = flag.String("format", "yuv", "format yuv or rgb")
var flagFrames = flag.Int("frames", 0, "frames to capture")

func main() {
	// defer profile.Start(profile.MemProfile).Stop()

	// log.SetFlags(log.Ltime | log.Lmicroseconds | log.Lshortfile)
	log.SetFlags(log.Ltime | log.Lmicroseconds)
	flag.Parse()

	// imageInChan is where images enter the system
	imageInChan := make(chan simage, 100)

	dev := initCaptureDevice(*flagInput)
	defer func(dev *v4l.Device) {
		lp("streamoff result: %v", dev.EndCapture())
		dev.DoneBuffers()
		lp("close result: %v", dev.CloseDevice())
	}(dev)
	go fetchImage(dev, imageInChan)

	writeImages(imageInChan)
}

func initCaptureDevice(path string) *v4l.Device {
	dev, err := v4l.OpenDevice(path, false)
	if err != nil {
		log.Fatalf("%v", err)
	}

	fmts, err := dev.GetSupportedFormats()
	if err != nil {
		log.Fatalf("%v", err)
	}
	pxlfmt := uint32(0)
	switch(*flagFormat) {
	case "yuv": pxlfmt = v4l.FormatYuyv
	case "rgb": pxlfmt = v4l.FormatRgb
	case "jpg": pxlfmt = v4l.FormatJpeg
	default: log.Fatalf("Unsupported format '%s'", *flagFormat)
	}

	found := false
	for _, f := range fmts {
		if f == pxlfmt {
			found = true
		}
	}
	if !found {
		log.Fatalf("requested format %s not supported by device", *flagFormat)
	}

	dev.SetFormat(v4l.Format{Height: *flagHeight, Width: *flagWidth, Format: pxlfmt})

	lp("supported formats: %v", fmts)
	if err = dev.InitBuffers(*flagNumBufs); err != nil {
		log.Fatalf("init=%v", err)
	}
	if err = dev.Capture(); err != nil {
		log.Fatalf("capture=%v", err)
	}
	return dev
}

func writeImages(in chan simage) {
	for simg := range in {
		i := simg.seq
		fname := fmt.Sprintf("test%019d.%s", simg.tm.UnixNano(), *flagFormat)
		logsince(simg.tm, "%d D starting write of image %s", i, fname)
		start := time.Now()
		file, err := os.Create(fname)
		if err == nil {
			switch *flagFormat {
			case "yuv":
				_, err = file.Write(simg.img.(*imglib.YUYV).Pix)
			case "rgb":
				_, err = file.Write(simg.img.(*imglib.RGB).Pix)
			case "jpg":
				err = jpeg.Encode(file, simg.img, nil)
			}
		}
		logsince(start, "%d F wrote image %s, err=%v", i, fname, err)
	}
}

func logsince(start time.Time, fs string, opt ...interface{}) {
	log.Printf("[%.3fs] %s", time.Since(start).Seconds(), fmt.Sprintf(fs, opt...))
}

func lp(fs string, opt ...interface{}) {
	log.Printf("         %s", fmt.Sprintf(fs, opt...))
}

func fetchImage(dev *v4l.Device, imageChan chan simage) {
	lp("starting capture")
	bufrel := make(chan v4l.Frame, *flagNumBufs)
	go func(in chan v4l.Frame) {
		for h := range in {
			// start := time.Now()
			err := dev.DoneFrame(h)
			if err != nil {
				log.Fatalf("error releasing frame: %v", err)
			}
			// logsince(start, "released buffer %d, err=%v", h, err)
		}
	}(bufrel)
	i := 0
	for {
		start := time.Now()
		frame, err := dev.GetFrame()
		if *flagFormat == "jpg" {
			// We want to release the buffer ASAP and decoding isn't fast...
			newFrame := frame.CopyPix()
			bufrel <- frame
			frame = newFrame
		}
		logsince(start, "%d B got %s frame bytes=%d", i, *flagFormat, len(frame.Pix))

		// TODO for compressed formats like JPEG, copy/release buffer before decoding
		getImgStart := time.Now()
		img, err := dev.GetImage(frame)
		logsince(getImgStart, "%d C got image dim=%v err=%v", i, img.Bounds(), err)
		// give buffer back to v4l device if we haven't already
		if *flagFormat != "jpg" {
			bufrel <- frame
		}

		if err == nil {
			imageChan <- simage{img: img, seq: i, tm: frame.ReqTime}
		}
		i++
		if *flagFrames == i {
			close(imageChan)
			break
		}
		logsince(start, "%d A frame complete, fetching next frame", i)
	}
}
