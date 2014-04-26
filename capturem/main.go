// capturem demonstrates the motion package using input from a video device.
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
var flagWidth = flag.Int("width", 640, "width in pixels")
var flagHeight = flag.Int("height", 480, "height in pixels")
var flagFormat = flag.String("format", "yuv", "format yuv or rgb or jpg")
var flagFrames = flag.Int("frames", 0, "frames to capture")
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
		if rimg := trackRects(*flagDeltaThresh, trk, simg); rimg != nil {
			select {
			case imgdisp <- []imgseq.Img{simg, rimg}:
			default:
			}
		}
	}
}

func trackRects(deltaThresh int, trk *motion.Tracker, img imgseq.Img) imgseq.Img {
	if rs := trk.GetRects(img, deltaThresh); len(rs) > 0 {
		sort.Sort(motion.RectAreaSlice(rs))
		rimg := getRectImage(img.GetImage(), rs)
		return &imgseq.RawImg{img.GetImgInfo(), imglib.GetPixelSequence(rimg)}
	} else {
		return nil
	}

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

