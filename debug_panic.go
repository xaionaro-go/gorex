package gorex

import (
	"fmt"
	"io"
	"os"
	"runtime"
	_ "unsafe" // for "go:linkname"
)

var debugPanicOut = io.Writer(os.Stderr)

func debugPanic(
	monopolizedBy GoroutineID,
	usedBy map[GoroutineID]*int64,
) {
	if monopolizedBy != 0 {
		fmt.Fprintf(debugPanicOut, "The lock is monopolized by goroutine %dю\n", monopolizedBy)
	}

	if len(usedBy) > 0 {
		fmt.Fprintf(debugPanicOut, "There are %d goroutines holding a read lock on the locker:\n", len(usedBy))
		gCount := 0
		for g, lockCount := range usedBy {
			gCount++
			fmt.Fprintf(debugPanicOut, "\t%d. %d reader-locks by goroutine %d.\n", gCount, *lockCount, g)
		}
	}

	b := make([]byte, 1024*1024)
	n := runtime.Stack(b, true)
	b = b[:n]
	panic(fmt.Sprintf("The InfiniteContext is done...\nSTACKS:\n%s\n", b))
}
