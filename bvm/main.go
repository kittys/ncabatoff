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

	// flag.IntVar(&flagStartFrame, "start", 0,
	//		"If set, bv will start at this frame")

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

func viewDir(path string) {
	dl := imgseq.GetDirList(path)
	filtdl := dl
	filtdl.Files = make([]string, 0, len(dl.Files))
	for _, fn := range dl.Files {
		if filepath.Ext(fn) == ".yuv" {
			filtdl.Files = append(filtdl.Files, fn)
		}
	}

	imgin := make(chan imgseq.Img, 100)
	go imgseq.LoadRawImgs(filtdl, imgin)
	imgout := make(chan []imgseq.Img)

	go vlib.StreamImages(imgout)
	filterInactive(imgin, imgout)
	close(imgout)
}

func filterInactive(imgin chan imgseq.Img, imgout chan []imgseq.Img) {
	trk := motion.NewTracker()
	for simg := range imgin {
		oimg := simg.GetImage()
		iinfo := simg.GetImgInfo()
		glog.V(1).Infof("filtering %s img %v", oimg.Bounds(), iinfo)
		if rs := trk.GetRects(simg, flagDeltaThresh); len(rs) > 0 {
			rs = filtRects(rs)
			sort.Sort(motion.RectAreaSlice(rs))
			ops := imglib.GetPixelSequence(imglib.StdImage{oimg}.GetRGBA())
			rps := imglib.GetPixelSequence(getRectImage(simg, rs))
			imgout <- []imgseq.Img{&imgseq.RawImg{iinfo, ops}, &imgseq.RawImg{iinfo, rps}}
		}
	}
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
