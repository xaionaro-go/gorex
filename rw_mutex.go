package gorex

import (
	"context"
	"math"
	"sync"

	"github.com/xaionaro-go/rand/mathrand"

	"github.com/xaionaro-go/spinlock"
)

const (
	blockedByWriter = -math.MaxInt64 + 1
)

// RWMutex is a goroutine-aware analog of sync.RWMutex, so it works
// the same way as sync.RWMutex, but tracks which goroutine locked
// it. So it could be locked multiple times with the same routine.
type RWMutex struct {
	lazyInitOnce sync.Once

	usedDone         chan struct{}
	monopolizedDone  chan struct{}
	monopolizedDepth int
	monopolizedBy    *G

	state          int64
	backendLocker  sync.Mutex
	internalLocker spinlock.Locker
	usedBy         map[*G]*int64
}

func (m *RWMutex) lazyInit() {
	m.lazyInitOnce.Do(func() {
		m.usedBy = map[*G]*int64{}
	})
}

// Lock is analog of `(*sync.RWMutex)`.Lock, but it allows one goroutine
// to call it and RLock multiple times without calling Unlock/RUnlock.
func (m *RWMutex) Lock() {
	m.lock(nil, true)
}

// LockTry is analog of Lock(), but it does not block if it cannot lock
// right away.
//
// Returns `false` if was unable to lock.
func (m *RWMutex) LockTry() bool {
	return m.lock(nil, false)
}

// LockCtx is analog of Lock(), but allows to continue the try to lock only until context is done.
//
// Returns `false` if was unable to lock (context finished before it was possible to lock).
func (m *RWMutex) LockCtx(ctx context.Context) bool {
	return m.lock(ctx, true)
}

func (m *RWMutex) lock(ctx context.Context, shouldWait bool) bool {
	m.lazyInit()
	me := GetG()

	if ctx == nil {
		ctx = infiniteContext
	}

	for {
		m.internalLocker.Lock()
		if m.monopolizedBy == nil {
			m.monopolizedBy = me
			m.monopolizedDepth++
			m.internalLocker.Unlock()
			m.backendLocker.Lock()
			m.setStateBlockedByWriter(me)
			return true
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
			return true
		}
		if !shouldWait {
			return false
		}
		select {
		case <-ch:
		case <-ctx.Done():
			return false
		}
	}
}

// Unlock is analog of `(*sync.RWMutex)`.Unlock, but it cannot be called
// from a routine which does not hold the lock (see `Lock`).
func (m *RWMutex) Unlock() {
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
		m.state -= blockedByWriter
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
func (m *RWMutex) LockDo(fn func()) {
	m.Lock()
	defer m.Unlock()

	fn()
}

// LockTryDo is a wrapper around LockTry and Unlock.
//
// See also LockDo and LockTry.
func (m *RWMutex) LockTryDo(fn func()) (success bool) {
	if !m.LockTry() {
		return false
	}
	defer m.Unlock()

	success = true
	fn()
	return
}

// LockCtxDo is a wrapper around LockCtx and Unlock.
//
// See also LockDo and LockCtx.
func (m *RWMutex) LockCtxDo(ctx context.Context, fn func()) (success bool) {
	if !m.LockCtx(ctx) {
		return false
	}
	defer m.Unlock()

	success = true
	fn()
	return
}

func (m *RWMutex) setStateBlockedByWriter(me *G) {
	m.internalLocker.Lock()
	defer m.internalLocker.Unlock()
	for {
		m.state += blockedByWriter
		if m.state == blockedByWriter {
			return
		}
		if myReadersCountPtr, _ := m.usedBy[me]; myReadersCountPtr != nil {
			if m.state-*myReadersCountPtr == blockedByWriter {
				return
			}
		}
		m.state -= blockedByWriter

		if m.usedDone == nil {
			m.usedDone = make(chan struct{})
		}
		ch := m.usedDone
		m.internalLocker.Unlock()
		select {
		case <-ch:
		}
		m.internalLocker.Lock()
	}
}

type int64PoolT []*int64

func (pool *int64PoolT) put(v *int64) {
	*v = 1
	*pool = append(*pool, v)
}

func (pool *int64PoolT) get() *int64 {
	if len(*pool) == 0 {
		for i := 0; i < 100; i++ {
			pool.put(&[]int64{1}[0])
		}
	}

	idx := len(*pool) - 1
	v := (*pool)[idx]
	*pool = (*pool)[:idx]

	return v
}

var int64Pool = &int64PoolT{}

func (m *RWMutex) incMyReaders(me *G) {
	if v := m.usedBy[me]; v == nil {
		m.usedBy[me] = int64Pool.get()
	} else {
		*v++
	}
}

var prng = mathrand.New()

func (m *RWMutex) gc() {
	if prng.Uint32MultiplyAdd()>>24 != 0 {
		return
	}

	for k, v := range m.usedBy {
		if *v != 0 {
			continue
		}
		delete(m.usedBy, k)
	}
}

func (m *RWMutex) decMyReaders(me *G) {
	v := m.usedBy[me]
	if v == nil {
		panic("RUnlock()-ing not RLock()-ed")
	}
	*v--
	if *v != 0 {
		return
	}
	m.gc()
	ch := m.usedDone
	if ch == nil {
		return
	}
	close(ch)
	m.usedDone = nil
	int64Pool.put(v)
}

// RLock is analog of `(*sync.RWMutex)`.RLock, but it allows one goroutine
// to call Lock and RLock multiple times without calling Unlock/RUnlock.
func (m *RWMutex) RLock() {
	m.rLock(nil, true)
}

// RLockTry is analog of RLock(), but it does not block if it cannot lock
// right away.
//
// Returns `false` if was unable to lock.
func (m *RWMutex) RLockTry() bool {
	return m.rLock(nil, false)
}

// RLockCtx is analog of RLock(), but allows to continue the try to lock only until context is done.
//
// Returns `false` if was unable to lock.
func (m *RWMutex) RLockCtx(ctx context.Context) bool {
	return m.rLock(ctx, true)
}

func (m *RWMutex) rLock(ctx context.Context, shouldWait bool) bool {
	m.lazyInit()
	me := GetG()
	if ctx == nil {
		ctx = infiniteContext
	}

	for {
		m.internalLocker.Lock()
		m.state++
		isOK := m.state > 0
		if isOK {
			break
		}
		monopolizedBy := m.monopolizedBy
		if monopolizedBy == me {
			break
		}

		m.state--
		if !shouldWait {
			m.internalLocker.Unlock()
			return false
		}

		if m.monopolizedDone == nil {
			m.monopolizedDone = make(chan struct{})
		}
		ch := m.monopolizedDone
		m.internalLocker.Unlock()

		select {
		case <-ch:
		case <-ctx.Done():
			return false
		}
	}

	m.incMyReaders(me)
	m.internalLocker.Unlock()
	return true
}

// RUnlock is analog of `(*sync.RWMutex)`.RUnlock, but it cannot be called
// from a routine which does not hold the lock (see `RLock`).
func (m *RWMutex) RUnlock() {
	me := GetG()

	m.internalLocker.Lock()
	m.state--
	m.decMyReaders(me)
	m.internalLocker.Unlock()
}

// RLockDo is a wrapper around RLock and RUnlock.
// It's a handy function to see in the call stack trace which locker where was locked.
// Also it's handy not to forget to unlock the locker.
func (m *RWMutex) RLockDo(fn func()) {
	m.RLock()
	defer m.RUnlock()

	fn()
}

// RLockTryDo is a wrapper around RLockTry and RUnlock.
//
// See also RLockDo and RLockTry.
func (m *RWMutex) RLockTryDo(fn func()) (success bool) {
	if !m.RLockTry() {
		return false
	}
	defer m.RUnlock()

	success = true
	fn()
	return
}

// RLockCtxDo is a wrapper around RLockTry and RUnlock.
//
// See also RLockDo and RLockCtx.
func (m *RWMutex) RLockCtxDo(ctx context.Context, fn func()) (success bool) {
	if !m.RLockCtx(ctx) {
		return false
	}
	defer m.RUnlock()

	success = true
	fn()
	return
}
