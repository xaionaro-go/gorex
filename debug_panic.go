package gorex

import (
	"context"
	"fmt"
	"runtime"
	_ "unsafe" // for "go:linkname"
)

//go:linkname gcallers runtime.gcallers
func gcallers(gp *G, skip int, pcbuf []uintptr) int

// InfiniteContext is used as the default context used on any try to lock if
// a custom context is not set (see LockCtx/RLockCtx), but with the difference
// if this context will be done, then it will panic with debugging information.
//
// To specify a context with deadline may be useful for unit tests.
var InfiniteContext = context.Background()

func printGCallStack(g *G) {
	pcbuf := make([]uintptr, 65536)
	n := gcallers(g, 0, pcbuf)
	frames := runtime.CallersFrames(pcbuf[:n])
	for {
		frame, ok := frames.Next()
		if !ok {
			return
		}
		fmt.Printf("%v:%v (%v)\n", frame.File, frame.Line, frame.Function)
	}
}

func debugPanic(locker interface{}, monopolizedBy *G, usedBy map[*G]*int64) {
	if monopolizedBy != nil {
		fmt.Println("monopolized by:")
		printGCallStack(monopolizedBy)
	}

	for g, count := range usedBy {
		fmt.Printf("reader with count %d by\n", *count)
		printGCallStack(g)
	}
	panic("The InfinityContext is done...")
}
