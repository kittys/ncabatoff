package main

import (
	"code.google.com/p/ncabatoff/imglib"
	"code.google.com/p/ncabatoff/imgseq"
	"code.google.com/p/ncabatoff/motion"
	"code.google.com/p/ncabatoff/v4l"
	"code.google.com/p/ncabatoff/vlib"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"image"
	"image/color"
	"image/draw"
	"runtime"
	"sort"
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
var flagDeltaThresh = flag.Int("deltaThresh", 32*69, "delta filter threshold")

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
	go vlib.StreamImages(imgdisp)
	i := 1
	trk := motion.NewTracker()
	for simg := range cs.GetOutput() {
		if i == *flagFrames {
			break
		}
		i++
		display(trk, imgdisp, simg)
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

func display(trk *motion.Tracker, imgdisp chan []imgseq.Img, img imgseq.Img) {
	imginfo := img.GetImgInfo()

	img1 := imgseq.RawImg{imginfo, imglib.GetPixelSequence(convertYuyv(img))}

	var img2 imgseq.Img
	if rs := trk.GetRects(img, *flagDeltaThresh); len(rs) > 0 {
		sort.Sort(motion.RectAreaSlice(rs))

		rimg := getRectImage(img1.GetImage(), rs)
		img2 = &imgseq.RawImg{imginfo, imglib.GetPixelSequence(rimg)}
	} else {
		return
	}

	select {
	case imgdisp <- []imgseq.Img{&img1, img2}:
	default:
	}
}

func logtime(f func(), fs string, opt ...interface{}) {
	start := time.Now()
	f()
	logsince(start, fs, opt...)
}

func getRectImage(img image.Image, rects []image.Rectangle) image.Image {
    irect := img.Bounds()
    out := image.NewRGBA(irect)
    for _, r := range rects {
            draw.Draw(out, r.Inset(-1), &image.Uniform{color.White}, image.ZP, draw.Src)
            draw.Draw(out, r, img, r.Min, draw.Src)
    }
    return out
}

func logsince(start time.Time, fs string, opt ...interface{}) {
	glog.V(1).Infof("[%.3fs] %s", time.Since(start).Seconds(), fmt.Sprintf(fs, opt...))
}

func lp(fs string, opt ...interface{}) {
	glog.V(1).Infof("         %s", fmt.Sprintf(fs, opt...))
}

