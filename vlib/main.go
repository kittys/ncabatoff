package vlib

import (
	"github.com/BurntSushi/xgbutil"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/golang/glog"
)

var (
	// A list of keybindings. Each value corresponds to a triple of the key
	// sequence to bind to, the action to run when that key sequence is
	// pressed and a quick description of what the keybinding does.
	Keybinds = []Keyb{
		{
			"left", "Cycle to the previous image.",
			func(w *window) { w.chans.prevImg <- struct{}{} },
		},
		{
			"right", "Cycle to the next image.",
			func(w *window) { w.chans.nextImg <- struct{}{} },
		},
		{
			"shift-h", "Cycle to the previous image.",
			func(w *window) { w.chans.prevImg <- struct{}{} },
		},
		{
			"shift-l", "Cycle to the next image.",
			func(w *window) { w.chans.nextImg <- struct{}{} },
		},
		{
			"r", "Resize the window to fit the current image.",
			func(w *window) { w.chans.resizeToImageChan <- struct{}{} },
		},
		{
			"h", "Pan left.", func(w *window) { w.stepLeft() },
		},
		{
			"j", "Pan down.", func(w *window) { w.stepDown() },
		},
		{
			"k", "Pan up.", func(w *window) { w.stepUp() },
		},
		{
			"l", "Pan right.", func(w *window) { w.stepRight() },
		},
		{
			"q", "Quit.", func(w *window) { xevent.Quit(w.X) },
		},
	}
)

func ViewImages(fetcher ImageFetcher) {
	// Connect to X and quit if we fail.
	X, err := xgbutil.NewConn()
	if err != nil {
		glog.Fatal(err)
	}

	// Create the X window before starting anything so that the user knows
	// something is going on.
	Canvas(X, fetcher)

	// Start the main X event loop.
	xevent.Main(X)
}
