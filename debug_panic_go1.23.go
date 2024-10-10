//go:build go1.23
// +build go1.23

package gorex

const GO_VERSION = ">=go1.23"

func gcallers(gp *G, skip int, pcbuf []uintptr) int {
	// unfortunately in Go1.23 they forbid access to
	// runtime.* functions even via linkname, so now
	// we just cannot get the stack trace of a specific
	// goroutine.
	return 0
}
