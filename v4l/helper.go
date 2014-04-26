package v4l

import "fmt"
import "time"
import "code.google.com/p/ncabatoff/imgseq"
import "github.com/golang/glog"

// A CaptureStream is a helpful wrapper around Device which sends images to
// an output channel.
type CaptureStream struct {
	dev *Device
	output chan imgseq.Img
	done chan struct{}
}

// NewStream opens and initializes the device and starts streaming captured
// images.  Any errors result in a call to glog.Fatalf.  This also applies
// to errors experienced subsequent to opening the device, i.e. while streaming.
func NewStream(device string, fps int, pxlfmt string, width int, height int, output chan imgseq.Img) *CaptureStream {
	if output == nil {
		output = make(chan imgseq.Img)
	}
	cs := &CaptureStream{output: output, done: make(chan struct{})}

	dev, err := OpenDevice(device, false)
	if err != nil {
		glog.Fatalf("%v", err)
	}
	cs.dev = dev
	cs.setFormatOrDie(fps, pxlfmt, width, height)

	for numbufs := 30; numbufs > 0; numbufs-- {
		if err = dev.InitBuffers(numbufs); err != nil {
			if 1 == numbufs {
				glog.Fatalf("init=%v", err)
			}
		} else {
			break
		}
	}
	if err = dev.Capture(); err != nil {
		glog.Fatalf("capture=%v", err)
	}

	cs.dev = dev
	go cs.fetchImages()
	return cs
}

// GetOutput returns the channel to which captured images are sent.
// Any failures during capture result in a call to glog.Fatalf.
func (cs *CaptureStream) GetOutput() <-chan imgseq.Img {
	return cs.output
}

// Shutdown stops capturing.
func (cs *CaptureStream) Shutdown() {
	cs.done <- struct{}{}
}

func lp(fs string, opt ...interface{}) {
	glog.V(1).Infof("         %s", fmt.Sprintf(fs, opt...))
}

func logsince(start time.Time, fs string, opt ...interface{}) {
	glog.V(1).Infof("[%.3fs] %s", time.Since(start).Seconds(), fmt.Sprintf(fs, opt...))
}

// shutdown stops capturing, closes the output channel, releases buffers, and closes the device.
func (cs *CaptureStream) shutdown() {
	lp("shutting down")
	lp("streamoff result: %v", cs.dev.EndCapture())
	close(cs.output)
	cs.dev.DoneBuffers()
	lp("close result: %v", cs.dev.CloseDevice())
	cs.dev, cs.output, cs.done = nil, nil, nil
}

// setFormatOrDie configures the device, calling glog.Fatalf on failures.
// Since we're writing Img to the output channel and it doesn't presently
// support anything except yuv and rgb, those are the only formats we accept.
func (cs *CaptureStream) setFormatOrDie(fps int, pxlfmt string, width int, height int) {
	fmts, err := cs.dev.GetSupportedFormats()
	if err != nil {
		glog.Fatalf("%v", err)
	}
	lp("supported formats: %v", fmts)

	var fmtid FormatId
	switch(pxlfmt) {
	case "yuv": fmtid = FormatYuyv
	case "rgb": fmtid = FormatRgb
	default: glog.Fatalf("Unsupported format '%s'", fmtid)
	}

	found := false
	for _, f := range fmts {
		if f == fmtid {
			found = true
		}
	}
	if !found {
		glog.Fatalf("requested format %s not supported by device", pxlfmt)
	}

	vf := Format{Height: height, Width: width, FormatId: fmtid}
	if err := cs.dev.SetFormat(vf); err != nil {
		glog.Fatalf("setformat=%v", err)
	}

	if fps != 0 {
		if err := cs.dev.SetFps(fps); err != nil {
			glog.Fatalf("setfps=%v", err)
		}
	}
	nom, denom := cs.dev.GetFps()
	glog.Infof("fps=%d/%d", nom, denom)
}

// fetchImages contains the main capture loop: asking the device for frames,
// copying them, releasing the frame buffer back to the device (in another
// goroutine), and sending the images to the output channel.
func (cs *CaptureStream) fetchImages() {
	lp("starting capture")
	bufrel := make(chan AllocFrame, len(cs.dev.buffers))
	defer func() {
		close(bufrel)
	}()
	go func(in chan AllocFrame) {
		for h := range in {
			start := time.Now()
			err := cs.dev.DoneFrame(h)
			if err != nil {
				glog.Fatalf("error releasing frame: %v", err)
			}
			logsince(start, "released buffer %d, err=%v", h.GetBufNum(), err)
		}
	}(bufrel)
	i := 0
	for {
		select {
		case <-cs.done:
			cs.shutdown()
			return
		default:
		}
		start := time.Now()
		var safeframe FreeFrame
		if frame, err := cs.dev.GetFrame(); err != nil {
			glog.Fatalf("error reading frame: %v", err)
		} else {
			logsince(start, "%d got frame bufnum=%d bytes=%d", i, frame.GetBufNum(), len(frame.Pix))
			safeframe = frame.Copy()
			bufrel <- frame
		}
		if ps, err := safeframe.GetPixelSequence(); err != nil {
			glog.Fatalf("error getting pixel seq: %v", err)
		} else {
			iinfo := imgseq.ImgInfo{SeqNum: i, CreationTs: safeframe.ReqTime}
			select {
			case <-cs.done:
				cs.shutdown()
				break
			case cs.output <- &imgseq.RawImg{iinfo, *ps}:
			}
		}

		i++
		logsince(start, "%d frame complete, fetching next frame", i)
	}
}
