package main

import (
	"fmt"
	"time"
)

import "github.com/golang/glog"

func dbg(ctx string, format string, args ...interface{}) {
	glog.V(2).Infof(format, args...)
}

// lg is a convenient alias for printing verbose output.
func lg(format string, args ...interface{}) {
	glog.V(1).Infof(format, args...)
}

func logsince(start time.Time, fs string, opt ...interface{}) {
	lg("[%.3fs] %s", time.Since(start).Seconds(), fmt.Sprintf(fs, opt...))
}

func lp(fs string, opt ...interface{}) {
	lg("         %s", fmt.Sprintf(fs, opt...))
}
