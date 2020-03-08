package goroutine

import (
	"sync"

	"github.com/xaionaro-go/spinlock"
)

type Mutex struct {
	backendLocker   sync.Mutex
	internalLocker  spinlock.Locker
	monopolizedBy   *g
	monopolizedDone chan struct{}
}

func (m *Mutex) LockDo(fn func()) {
	me := GetG()

	var monopolizedByWasSetHere bool
	defer func() {
		m.internalLocker.Lock()
		if monopolizedByWasSetHere {
			m.monopolizedBy = nil
			m.backendLocker.Unlock()
		}
		chPtr := m.monopolizedDone
		m.monopolizedDone = nil
		m.internalLocker.Unlock()
		if chPtr == nil {
			return
		}
		close(chPtr)
	}()

	for {
		var monopolizedBy *g
		m.internalLocker.Lock()
		if m.monopolizedBy == nil {
			m.monopolizedBy = me
			monopolizedBy = me
			monopolizedByWasSetHere = true
			m.internalLocker.Unlock()
			break
		} else {
			monopolizedBy = m.monopolizedBy
		}
		monopolizedByMe := monopolizedBy == me
		var ch chan struct{}
		if !monopolizedByMe {
			if m.monopolizedDone == nil {
				m.monopolizedDone = make(chan struct{})
			}
			ch = m.monopolizedDone
		}
		m.internalLocker.Unlock()
		if monopolizedByMe {
			break
		}
		select {
		case <-ch:
		}
	}

	if monopolizedByWasSetHere {
		m.backendLocker.Lock()
	}

	fn()
}
