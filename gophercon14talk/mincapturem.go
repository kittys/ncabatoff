package main

import (
	"code.google.com/p/ncabatoff/imglib"
	"code.google.com/p/ncabatoff/imgseq"
	"code.google.com/p/ncabatoff/motion"
	"code.google.com/p/ncabatoff/v4l"
	"code.google.com/p/ncabatoff/vlib"
	"image"
	"image/color"
	"image/draw"
	"sort"
)

const deltaThresh = 20*69

func main() {
	cs := v4l.NewStream("/dev/video0", 0, "yuv", 640, 480, nil)
	defer func() {
		cs.Shutdown()
	}()

	imgdisp := make(chan []imgseq.Img, 1)
	go vlib.StreamImages(imgdisp)
	trk := motion.NewTracker()
	for simg := range cs.GetOutput() {
		if rimg := trackRects(trk, simg); rimg != nil {
			select {
			case imgdisp <- []imgseq.Img{simg, rimg}:
			default:
			}
		}
	}
}

func trackRects(trk *motion.Tracker, img imgseq.Img) imgseq.Img {
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

