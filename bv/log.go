package main

import (
	"fmt"
	"log"
	"os"
	"time"
)
import "github.com/seehuhn/trace"

var listener trace.ListenerHandle

func init() {
    listener = trace.Register(printTrace, "pv", trace.PrioAll)
}

func printTrace(t time.Time, path string, prio trace.Priority, msg string) {
    caller := trace.Callers()[1]
    fmt.Printf("%s %s %s %s\n", t.Format("15:04:05.000"), path, caller, msg)
}

func dbg(ctx string, format string, args ...interface{}) {
    trace.T("pv/" + ctx, trace.PrioDebug, format, args...)
}

var errLg = log.New(os.Stderr, "[bv error] ", log.Lshortfile)

// lg is a convenient alias for printing verbose output.
func lg(format string, v ...interface{}) {
	if !flagVerbose {
		return
	}
	log.Printf(format, v...)
}

func logsince(start time.Time, fs string, opt ...interface{}) {
	log.Printf("[%.3fs] %s", time.Since(start).Seconds(), fmt.Sprintf(fs, opt...))
}

func lp(fs string, opt ...interface{}) {
		log.Printf("         %s", fmt.Sprintf(fs, opt...))
}
