package goroutine

import (
	"sync"

	"github.com/xaionaro-go/spinlock"
)

type Locker struct {
	backendLocker   sync.Mutex
	internalLocker  spinlock.Locker
	monopolizedBy   *g
	monopolizedDone chan struct{}
}

func (locker *Locker) LockDo(fn func()) {
	me := GetG()

	var monopolizedByWasSetHere bool
	defer func() {
		locker.internalLocker.Lock()
		if monopolizedByWasSetHere {
			locker.monopolizedBy = nil
			locker.backendLocker.Unlock()
		}
		chPtr := locker.monopolizedDone
		locker.monopolizedDone = nil
		locker.internalLocker.Unlock()
		if chPtr == nil {
			return
		}
		close(chPtr)
	}()

	for {
		var monopolizedBy *g
		locker.internalLocker.Lock()
		if locker.monopolizedBy == nil {
			locker.monopolizedBy = me
			monopolizedBy = me
			monopolizedByWasSetHere = true
			locker.internalLocker.Unlock()
			break
		} else {
			monopolizedBy = locker.monopolizedBy
		}
		monopolizedByMe := monopolizedBy == me
		var ch chan struct{}
		if !monopolizedByMe {
			if locker.monopolizedDone == nil {
				locker.monopolizedDone = make(chan struct{})
			}
			ch = locker.monopolizedDone
		}
		locker.internalLocker.Unlock()
		if monopolizedByMe {
			break
		}
		select {
		case <-ch:
		}
	}

	if monopolizedByWasSetHere {
		locker.backendLocker.Lock()
	}

	fn()
}
