// bvm demonstrates the motion package using a directory of images as input.
package main

import (
	"code.google.com/p/ncabatoff/imglib"
	"code.google.com/p/ncabatoff/imgseq"
	"code.google.com/p/ncabatoff/motion"
	"code.google.com/p/ncabatoff/vlib"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"image"
	"image/color"
	"image/draw"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"sort"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

var (
	// The initial width and height of the window.
	flagWidth, flagHeight int

	// If set, the image window will automatically resize to the first image
	// that it displays.
	flagAutoResize bool

	// The amount to increment panning when using h,j,k,l
	flagStepIncrement int

	// Whether to run a CPU profile.
	flagProfile string

	// When set, bv will print all keybindings and exit.
	flagKeybindings bool

	flagDeltaThresh   int
	flagMinArea       int
	flagMaxArea       int
	flagMinSquareness int
	flagMillis int
	flagStart int
)

func init() {
	// Set GOMAXPROCS, since bv can benefit greatly from parallelism.
	runtime.GOMAXPROCS(runtime.NumCPU())

	// Set all of the flags.
	flag.IntVar(&flagWidth, "width", 480,
		"The initial width of the window.")
	flag.IntVar(&flagHeight, "height", 640,
		"The initial height of the window.")
	flag.BoolVar(&flagAutoResize, "auto-resize", false,
		"If set, window will resize to size of first image.")
	flag.IntVar(&flagStepIncrement, "increment", 20,
		"The increment (in pixels) used to pan the image.")
	flag.StringVar(&flagProfile, "profile", "",
		"If set, a CPU profile will be saved to the file name provided.")
	flag.BoolVar(&flagKeybindings, "keybindings", false,
		"If set, bv will output a list all keybindings.")
	flag.IntVar(&flagMillis, "millis", 20,
		"Millisecond delay between frames in play mode")
	flag.IntVar(&flagStart, "start", 0,
		"starting frame")

	flag.IntVar(&flagDeltaThresh, "deltaThresh", 32*69,
		"The delta filter threshold.")
	flag.IntVar(&flagMinArea, "minarea", 20, "minimum rect area")
	flag.IntVar(&flagMaxArea, "maxarea", 200, "maximum rect area")
	flag.IntVar(&flagMinSquareness, "minsquareness", 2, "max h:w or w:h ratio")
	flag.Usage = usage
	flag.Parse()

	// Do some error checking on the flag values... naughty!
	if flagWidth == 0 || flagHeight == 0 {
		glog.Fatal("The width and height must be non-zero values.")
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [flags] image-file [image-file ...]\n",
		filepath.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	defer func() {
		glog.Flush()
	}()
	runtime.GOMAXPROCS(runtime.NumCPU())

	// If we just need the keybindings, print them and be done.
	if flagKeybindings {
		for _, keyb := range vlib.Keybinds {
			fmt.Printf("%-10s %s\n", keyb.Key, keyb.Desc)
		}
		fmt.Printf("%-10s %s\n", "mouse",
			"Left mouse button will pan the image.")
		os.Exit(0)
	}

	// Run the CPU profile if we're instructed to.
	if len(flagProfile) > 0 {
		f, err := os.Create(flagProfile)
		if err != nil {
			glog.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	// Whoops!
	if flag.NArg() == 0 {
		fmt.Fprint(os.Stderr, "\n")
		glog.Errorf("No images specified.\n\n")
		usage()
	}

	viewDir(flag.Arg(0))
}

func filtRects(rects []image.Rectangle) []image.Rectangle {
	out := make([]image.Rectangle, 0, len(rects))
	maxsq := 10 * flagMinSquareness
	for _, r := range rects {
		sz := r.Size()
		ar := sz.X * sz.Y
		if ar >= flagMinArea && ar <= flagMaxArea {
			xdivr := 10 * sz.X / sz.Y
			ydivr := 10 * sz.Y / sz.X
			if xdivr <= maxsq && ydivr <= maxsq {
				out = append(out, r)
			}
		}
	}
	return out
}

func getDirList(path string) imgseq.DirList {
	if dl, err := imgseq.GetDirList(path); err != nil {
		glog.Fatalf("error reading directory '%s': %v", path, err)
		return imgseq.DirList{}
	} else {
		filtdl := dl
		filtdl.Files = make([]string, 0, len(dl.Files))
		for _, fn := range dl.Files {
			if filepath.Ext(fn) == ".yuv" {
				filtdl.Files = append(filtdl.Files, fn)
			}
		}
		return filtdl
	}
}

func viewDir(path string) {
	dl := getDirList(path)
	iinfos := dl.ImgInfos()

	trk := motion.NewTracker()

	lasti := 0
	for i := 0; i <= motion.LAVGN; i++ {
		trk.GetRects(imgseq.LoadRawImgOrDie(dl.ImgInfos()[i]), flagDeltaThresh)
		lasti++
	}

	vlib.ViewImages(func(i int) (int, []imgseq.Img) {
		lg("got %d", i)
		if i >= len(iinfos) {
			i = 0
		}
		if i < motion.LAVGN {
			i = motion.LAVGN - 1
		}
		if i != lasti+1 {
			trk = motion.NewTracker()
			for j := i; j <= i+motion.LAVGN; j++ {
				trk.GetRects(imgseq.LoadRawImgOrDie(dl.ImgInfos()[j]), flagDeltaThresh)
			}
		}
		defer func() {
			lasti = i
		}()

		for {
			img := imgseq.LoadRawImgOrDie(dl.ImgInfos()[i])
			if imgout := filterInactive(trk, img); len(imgout) > 0 {
				return i, imgout
			}
		}
	}, flagMillis, flagStart)
}

func filterInactive(trk *motion.Tracker, simg imgseq.Img) []imgseq.Img {
	oimg := simg.GetImage()
	iinfo := simg.GetImgInfo()
	glog.V(1).Infof("filtering %s img %v", oimg.Bounds(), iinfo)
	rs := trk.GetRects(simg, flagDeltaThresh)
	if len(rs) == 0 {
		return []imgseq.Img{}
	}
	rs = filtRects(rs)
	if len(rs) == 0 {
		return []imgseq.Img{}
	}
	sort.Sort(motion.RectAreaSlice(rs))
	ops := imglib.GetPixelSequence(imglib.StdImage{oimg}.GetRGBA())
	rps := imglib.GetPixelSequence(getRectImage(simg, rs))
	return []imgseq.Img{&imgseq.RawImg{iinfo, ops}, &imgseq.RawImg{iinfo, rps}}
}

func getRectImage(simg imgseq.Img, rects []image.Rectangle) image.Image {
	img := simg.GetImage()
	irect := img.Bounds()
	out := image.NewRGBA(irect)
	for _, r := range rects {
		draw.Draw(out, r.Inset(-1), &image.Uniform{color.White}, image.ZP, draw.Src)
		draw.Draw(out, r, img, r.Min, draw.Src)
	}
	return out
}
