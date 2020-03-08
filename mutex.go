package goroutine

import (
	"sync"

	"github.com/xaionaro-go/spinlock"
)

type Mutex struct {
	backendLocker    sync.Mutex
	internalLocker   spinlock.Locker
	monopolizedBy    *g
	monopolizedDepth int
	monopolizedDone  chan struct{}
}

func (m *Mutex) Lock() {
	me := GetG()

	for {
		var monopolizedBy *g
		m.internalLocker.Lock()
		if m.monopolizedBy == nil {
			m.monopolizedBy = me
			monopolizedBy = me
			m.monopolizedDepth++
			m.internalLocker.Unlock()
			m.backendLocker.Lock()
			return
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
		if monopolizedByMe {
			m.monopolizedDepth++
		}
		m.internalLocker.Unlock()
		if monopolizedByMe {
			return
		}
		select {
		case <-ch:
		}
	}
}

func (m *Mutex) Unlock() {
	me := GetG()
	m.internalLocker.Lock()
	if me != m.monopolizedBy {
		m.internalLocker.Unlock()
		panic("I'm not the one, who locked this mutex")
	}
	m.monopolizedDepth--
	if m.monopolizedDepth == 0 {
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
}

func (m *Mutex) LockDo(fn func()) {
	m.Lock()
	defer m.Unlock()

	fn()
}
