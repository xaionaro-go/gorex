package gorex

import (
	"github.com/phuslu/goid"
)

type GoroutineID = uint64

func GetGoroutineID() GoroutineID {
	return GoroutineID(goid.Goid())
}
