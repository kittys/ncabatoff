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

	cs := v4l.NewStream(*flagInput, *flagFps, *flagFormat, *flagWidth, *flagHeight, nil)
	defer func() {
		cs.Shutdown()
	}()

	imgdisp := make(chan []imgseq.Img, 1)
	if *flagDisplay {
		go vlib.StreamImages(imgdisp)
	}

	var outfile *os.File
	if *flagOutfile != "" {
		if f, err := os.OpenFile(*flagOutfile, os.O_WRONLY|os.O_CREATE, 0777); err != nil {
			glog.Fatalf("unable to open output '%s': %v", *flagOutfile, err)
		} else {
			outfile = f
		}
	}

	i := 1
	for simg := range cs.GetOutput() {
		if i == *flagFrames {
			break
		}
		i++
		if *flagOutfile != "" {
			writeImage(outfile, simg)
		} else if ! *flagDiscard {
			writeImageToNewFile(simg)
		}
		if *flagDisplay {
			display(imgdisp, simg)
		}
	}
}

func writeImageToNewFile(simg imgseq.Img) {
	i := simg.GetImgInfo().SeqNum
	cts := simg.GetImgInfo().CreationTs
	fname := imgseq.TimeToFname("test", cts) + ".yuv"
	logsince(cts, "%d D starting write of image %s", i, fname)
	start := time.Now()
	file, err := os.Create(fname)
	if err == nil {
		writeImage(file, simg)
	}
	logsince(start, "%d F wrote image %s, err=%v", i, fname, err)
}

func writeImage(outfile *os.File, simg imgseq.Img) {
	pix := simg.GetPixelSequence().ImageBytes.GetBytes()
	if _, err := outfile.Write(pix); err != nil {
		glog.Fatalf("error writing frame %d: %v", simg.GetImgInfo().SeqNum, err)
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

