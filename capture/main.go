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

	cs := v4l.NewStream(*flagInput, *flagFps, *flagFormat, *flagWidth, *flagHeight, nil)
	defer func() {
		cs.Shutdown()
	}()

	imgdisp := make(chan []imgseq.Img, 1)
	if *flagDisplay {
		go vlib.StreamImages(imgdisp)
	}
	i := 1
	for simg := range cs.GetOutput() {
		if i == *flagFrames {
			break
		}
		i++
		if ! *flagDiscard {
			writeImage(simg)
		}
		if *flagDisplay {
			display(imgdisp, simg)
		}
	}
}

func writeImage(simg imgseq.Img) {
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

