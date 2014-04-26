// mincapturm demonstrates the motion package using input from a video device;
// this is a stripped-down version of capture used to embed in a presentation.
package main

import (
	"code.google.com/p/ncabatoff/imgseq"
	"code.google.com/p/ncabatoff/v4l"
	"code.google.com/p/ncabatoff/vlib"
)

func main() {
	cs := v4l.NewStream("/dev/video0", 0, "yuv", 640, 480, nil)
	defer func() {
		cs.Shutdown()
	}()

	imgdisp := make(chan []imgseq.Img, 1)
	go vlib.StreamImages(imgdisp)

	for simg := range cs.GetOutput() {
		select {
		case imgdisp <- []imgseq.Img{simg}:
		default:
		}
	}
}
