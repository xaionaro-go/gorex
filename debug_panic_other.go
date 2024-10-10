//go:build !go1.23
// +build !go1.23

package gorex

const GO_VERSION = "<go1.23"

//go:linkname gcallers runtime.gcallers
func gcallers(gp *G, skip int, pcbuf []uintptr) int
