package main

import (
	// "bytes"
	"code.google.com/p/ncabatoff/v4l"
	"flag"
	"fmt"
	"os"
	"time"
	"runtime"
	"github.com/golang/glog"
)

// import "github.com/davecheney/profile"

type simage struct {
	seq int
	tm  time.Time
	pix []byte
}

var flagInput = flag.String("in", "/dev/video0", "input capture device")
var flagOutfile = flag.String("outfile", "", "write frames consecutively to output file, overwriting if exists")
var flagNumBufs = flag.Int("numbufs", 30, "number of mmap buffers")
var flagWidth = flag.Int("width", 640, "width in pixels")
var flagHeight = flag.Int("height", 480, "height in pixels")
var flagFormat = flag.String("format", "yuv", "format yuv or rgb or jpg")
var flagFrames = flag.Int("frames", 0, "frames to capture")
var flagDiscard = flag.Bool("discard", false, "discard frames")

func main() {
	// defer profile.Start(profile.MemProfile).Stop()

	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()

	// imageInChan is where images enter the system
	imageInChan := make(chan simage)

	dev := initCaptureDevice(*flagInput)
	defer func(dev *v4l.Device) {
		lp("streamoff result: %v", dev.EndCapture())
		dev.DoneBuffers()
		lp("close result: %v", dev.CloseDevice())
	}(dev)
	go fetchImages(dev, imageInChan)

	if *flagDiscard {
		for _ = range imageInChan {
		}
	} else if *flagOutfile != "" {
		if outfile, err := os.OpenFile(*flagOutfile, os.O_WRONLY|os.O_CREATE, 0777); err != nil {
			glog.Fatalf("unable to open output '%s': %v", *flagOutfile, err)
		} else {
			for simg := range imageInChan {
				if _, err = outfile.Write(simg.pix); err != nil {
					glog.Fatalf("error writing frame %d: %v", simg.seq, err)
				}
			}
		}
	} else {
		writeImages(imageInChan)
	}
}

func initCaptureDevice(path string) *v4l.Device {
	dev, err := v4l.OpenDevice(path, false)
	if err != nil {
		glog.Fatalf("%v", err)
	}

	fmts, err := dev.GetSupportedFormats()
	if err != nil {
		glog.Fatalf("%v", err)
	}
	pxlfmt := uint32(0)
	switch(*flagFormat) {
	case "yuv": pxlfmt = v4l.FormatYuyv
	case "rgb": pxlfmt = v4l.FormatRgb
	case "jpg": pxlfmt = v4l.FormatJpeg
	default: glog.Fatalf("Unsupported format '%s'", *flagFormat)
	}

	found := false
	for _, f := range fmts {
		if f == pxlfmt {
			found = true
		}
	}
	if !found {
		glog.Fatalf("requested format %s not supported by device", *flagFormat)
	}

	dev.SetFormat(v4l.Format{Height: *flagHeight, Width: *flagWidth, Format: pxlfmt})

	lp("supported formats: %v", fmts)
	if err = dev.InitBuffers(*flagNumBufs); err != nil {
		glog.Fatalf("init=%v", err)
	}
	if err = dev.Capture(); err != nil {
		glog.Fatalf("capture=%v", err)
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
			_, err = file.Write(simg.pix)
		}
		logsince(start, "%d F wrote image %s, err=%v", i, fname, err)
	}
}

func logsince(start time.Time, fs string, opt ...interface{}) {
	glog.V(1).Infof("[%.3fs] %s", time.Since(start).Seconds(), fmt.Sprintf(fs, opt...))
}

func lp(fs string, opt ...interface{}) {
	glog.V(1).Infof("         %s", fmt.Sprintf(fs, opt...))
}

func fetchImages(dev *v4l.Device, imageChan chan simage) {
	lp("starting capture")
	bufrel := make(chan v4l.Frame, *flagNumBufs)
	go func(in chan v4l.Frame) {
		for h := range in {
			start := time.Now()
			err := dev.DoneFrame(h)
			if err != nil {
				glog.Fatalf("error releasing frame: %v", err)
			}
			logsince(start, "released buffer %d, err=%v", h.GetBufNum(), err)
		}
	}(bufrel)
	i := 0
	for {
		start := time.Now()
		frame, err := dev.GetFrame()
		newFrame := frame.CopyPix()
		bufrel <- frame
		logsince(start, "%d got %s frame bytes=%d", i, *flagFormat, len(newFrame.Pix))

		if err == nil {
			imageChan <- simage{pix: newFrame.Pix, seq: i, tm: newFrame.ReqTime}
		}
		i++
		if *flagFrames == i {
			close(imageChan)
			break
		}
		logsince(start, "%d frame complete, fetching next frame", i)
	}
}
