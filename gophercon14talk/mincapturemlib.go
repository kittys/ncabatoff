package main

import (
	lib "code.google.com/p/ncabatoff/mincapturemlib"
	"runtime"
)

const deltaThresh = 1380

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	lib.Run("/dev/video0", deltaThresh)
}

