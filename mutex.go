package goroutine

import (
	"sync"

	"github.com/xaionaro-go/spinlock"
)

// Mutex is a goroutine-aware analog of sync.Mutex, so it works
// the same way as sync.Mutex, but tracks which goroutine locked
// it. So it could be locked multiple times with the same routine.
type Mutex struct {
	backendLocker    sync.Mutex
	internalLocker   spinlock.Locker
	monopolizedBy    *G
	monopolizedDepth int
	monopolizedDone  chan struct{}
}

// Lock is analog of `(*sync.Mutex)`.Lock, but it allows one goroutine
// to call it multiple times without calling Unlock.
func (m *Mutex) Lock() {
	me := GetG()

	for {
		m.internalLocker.Lock()
		if m.monopolizedBy == nil {
			m.monopolizedBy = me
			m.monopolizedDepth++
			m.internalLocker.Unlock()
			m.backendLocker.Lock()
			return
		}
		monopolizedByMe := m.monopolizedBy == me
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

// Unlock is analog of `(*sync.Mutex)`.Unlock, but it cannot be called
// from a routine which does not hold the lock (see `Lock`).
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

// LockDo is a wrapper around Lock and Unlock.
// It's a handy function to see in the call stack trace which locker where was locked.
// Also it's handy not to forget to unlock the locker.
func (m *Mutex) LockDo(fn func()) {
	m.Lock()
	defer m.Unlock()

	fn()
}
