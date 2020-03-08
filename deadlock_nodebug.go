// +build !deadlockdebug

package gorex

import (
	"sync"
)

func goroutineOpenedLock(lockPtr sync.Locker, isWrite bool) {}
func goroutineClosedLock(lockPtr sync.Locker, isWrite bool) {}
