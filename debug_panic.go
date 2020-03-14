package gorex

import (
	"fmt"
	"io"
	"os"
	"runtime"
	_ "unsafe" // for "go:linkname"
)

//go:linkname gcallers runtime.gcallers
func gcallers(gp *G, skip int, pcbuf []uintptr) int

var debugPanicOut = io.Writer(os.Stderr)

func printGCallStack(g *G) {
	pcbuf := make([]uintptr, 65536)
	n := gcallers(g, 0, pcbuf)
	frames := runtime.CallersFrames(pcbuf[:n])
	printFrames(frames)
}

func printFrames(frames *runtime.Frames) {
	for {
		frame, ok := frames.Next()
		if !ok {
			return
		}
		fmt.Fprintf(debugPanicOut, "%v:%v (%v)\n", frame.File, frame.Line, frame.Function)
	}
}

func debugPanic(locker interface{}, monopolizedBy *G, usedBy map[*G]*int64) {
	if monopolizedBy != nil {
		fmt.Fprintf(debugPanicOut, "monopolized %p:%+v by:\n", locker, locker)
		printGCallStack(monopolizedBy)
	}

	for g, count := range usedBy {
		fmt.Fprintf(debugPanicOut, "reader of %p:%+v with count %d by\n", locker, locker, *count)
		printGCallStack(g)
	}

	b := make([]byte, 1024*1024)
	n := runtime.Stack(b, true)
	b = b[:n]
	panic(fmt.Sprintf("The InfiniteContext is done...\nSTACKS:\n%s\n---\nCURRENT ROUTINE:\n", b))
}
