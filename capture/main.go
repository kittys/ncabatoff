package main

import (
	"code.google.com/p/ncabatoff/imglib"
	"code.google.com/p/ncabatoff/imgseq"
	"code.google.com/p/ncabatoff/v4l"
	"code.google.com/p/ncabatoff/vlib"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"image"
	"os"
	"runtime"
	"time"
)

// import "github.com/davecheney/profile"

var flagInput = flag.String("in", "/dev/video0", "input capture device")
var flagOutfile = flag.String("outfile", "", "write frames consecutively to output file, overwriting if exists")
var flagNumBufs = flag.Int("numbufs", 30, "number of mmap buffers")
var flagWidth = flag.Int("width", 640, "width in pixels")
var flagHeight = flag.Int("height", 480, "height in pixels")
var flagFormat = flag.String("format", "yuv", "format yuv or rgb or jpg")
var flagFrames = flag.Int("frames", 0, "frames to capture")
var flagDiscard = flag.Bool("discard", false, "discard frames")
var flagFps = flag.Int("fps", 0, "frames per second")
var flagDisplay = flag.Bool("display", false, "display images")

func main() {
	// defer profile.Start(profile.MemProfile).Stop()

	runtime.GOMAXPROCS(runtime.NumCPU())

	flag.Parse()
	defer func() {
		glog.Flush()
	}()

	// imageInChan is where images enter the system
	imageInChan := make(chan imgseq.Img)

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
				pix := simg.GetPixelSequence().ImageBytes.GetBytes()
				if _, err = outfile.Write(pix); err != nil {
					glog.Fatalf("error writing frame %d: %v", simg.GetImgInfo().SeqNum, err)
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
	lp("supported formats: %v", fmts)

	var pxlfmt v4l.FormatId
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

	vf := v4l.Format{Height: *flagHeight, Width: *flagWidth, FormatId: pxlfmt}
	if err := dev.SetFormat(vf); err != nil {
		glog.Fatalf("setformat=%v", err)
	}

	if *flagFps != 0 {
		if err := dev.SetFps(*flagFps); err != nil {
			glog.Fatalf("setfps=%v", err)
		}
	}
	nom, denom := dev.GetFps()
	glog.Infof("fps=%d/%d", nom, denom)

	if err = dev.InitBuffers(*flagNumBufs); err != nil {
		glog.Fatalf("init=%v", err)
	}
	if err = dev.Capture(); err != nil {
		glog.Fatalf("capture=%v", err)
	}
	return dev
}

func writeImages(in chan imgseq.Img) {
	imgdisp := make(chan []imgseq.Img, 1)
	if *flagDisplay {
		go vlib.StreamImages(imgdisp)
	}
	for simg := range in {
		i := simg.GetImgInfo().SeqNum
		cts := simg.GetImgInfo().CreationTs
		fname := imgseq.TimeToFname("test", cts) + ".yuv"
		logsince(cts, "%d D starting write of image %s", i, fname)
		start := time.Now()
		file, err := os.Create(fname)
		if err == nil {
			_, err = file.Write(simg.GetPixelSequence().ImageBytes.GetBytes())
		}
		logsince(start, "%d F wrote image %s, err=%v", i, fname, err)

		if *flagDisplay {
			display(imgdisp, simg)
		}
	}
}

func convertYuyv(img imgseq.Img) image.Image {
	if _, ok := img.GetPixelSequence().ImageBytes.(imglib.YuyvBytes); !ok {
		return img.GetImage()
	}
	r := image.Rect(0, 0, *flagWidth, *flagHeight)
	pix := img.GetPixelSequence().ImageBytes.GetBytes()
	yuyv := &imglib.YUYV{Pix: pix, Rect: r, Stride: *flagWidth * 2}
	var img1 image.Image
	logtime(func() {img1 = imglib.StdImage{yuyv}.GetRGBA()}, "%d converted to RGBA", img.GetImgInfo().SeqNum)
	return img1
}

func display(imgdisp chan []imgseq.Img, img imgseq.Img) {
	sendstart := time.Now()
	imginfo := img.GetImgInfo()
	rawimg := imgseq.RawImg{imginfo, imglib.GetPixelSequence(convertYuyv(img))}
	select {
	case imgdisp <- []imgseq.Img{&rawimg}:
		logsince(sendstart, "%d sent image to be displayed", imginfo.SeqNum)
	default:
	}
}

func logtime(f func(), fs string, opt ...interface{}) {
	start := time.Now()
	f()
	logsince(start, fs, opt...)
}

func logsince(start time.Time, fs string, opt ...interface{}) {
	glog.V(1).Infof("[%.3fs] %s", time.Since(start).Seconds(), fmt.Sprintf(fs, opt...))
}

func lp(fs string, opt ...interface{}) {
	glog.V(1).Infof("         %s", fmt.Sprintf(fs, opt...))
}

func fetchImages(dev *v4l.Device, imageChan chan imgseq.Img) {
	lp("starting capture")
	bufrel := make(chan v4l.AllocFrame, *flagNumBufs)
	defer func() {
		close(bufrel)
	}()
	go func(in chan v4l.AllocFrame) {
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
		var safeframe v4l.FreeFrame
		if frame, err := dev.GetFrame(); err != nil {
			glog.Fatalf("error reading frame: %v", err)
		} else {
			logsince(start, "%d got %s frame bufnum=%d bytes=%d", i, *flagFormat, frame.GetBufNum(), len(frame.Pix))
			safeframe = frame.Copy()
			bufrel <- frame
		}
		if ps, err := safeframe.GetPixelSequence(); err != nil {
			glog.Fatalf("error getting pixel seq: %v", err)
		} else {
			iinfo := imgseq.ImgInfo{SeqNum: i, CreationTs: safeframe.ReqTime}
			imageChan <- &imgseq.RawImg{iinfo, *ps}
		}

		i++
		if *flagFrames == i {
			close(imageChan)
			break
		}
		logsince(start, "%d frame complete, fetching next frame", i)
	}
}
