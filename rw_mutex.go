package gorex

import (
	"context"
	"fmt"
	"sync"

	"github.com/xaionaro-go/spinlock"
)

// RWMutex is a goroutine-aware analog of sync.RWMutex, so it works
// the same way as sync.RWMutex, but tracks which goroutine locked
// it. So it could be locked multiple times with the same routine.
type RWMutex struct {
	// InfiniteContext is used as the default context used on any try to lock if
	// a custom context is not set (see LockCtx/RLockCtx), but with the difference
	// if this context will be done, then it will panic with debugging information.
	//
	// To specify a context with deadline may be useful for unit tests.
	//
	// The zero-value means to use DefaultInfiniteContext.
	InfiniteContext context.Context

	lazyInitOnce sync.Once

	rlockDone      chan struct{}
	lockDone       chan struct{}
	lockCount      int
	lockedBy       GoroutineID
	rlockCount     int64
	backendLocker  sync.Mutex
	internalLocker spinlock.Locker
	usedBy         map[GoroutineID]*int64
	int64Pool      int64Pool
	gcCallCount    uint8
}

func (m *RWMutex) lazyInit() {
	m.lazyInitOnce.Do(func() {
		m.usedBy = map[GoroutineID]*int64{}
	})
}

// Lock is analog of `(*sync.RWMutex)`.Lock, but it allows one goroutine
// to call it and RLock multiple times without calling Unlock/RUnlock.
func (m *RWMutex) Lock() {
	if !m.lock(nil, true) {
		panic("should not happen")
	}
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

func (m *RWMutex) infiniteContext() context.Context {
	if m.InfiniteContext == nil {
		return DefaultInfiniteContext
	}
	return m.InfiniteContext
}

func (m *RWMutex) lock(ctx context.Context, shouldWait bool) bool {
	m.lazyInit()
	me := GetGoroutineID()

	m.internalLocker.Lock()
	if m.lockedBy == me {
		// already locked by me
		m.lockCount++
		m.internalLocker.Unlock()
		return true
	}

	if !m.setLockedByMe(ctx, me, shouldWait) {
		return false
	}
	goroutineOpenedLock(m, true)
	m.internalLocker.Unlock()
	m.backendLocker.Lock()
	return true
}

func (m *RWMutex) setLockedByMe(
	ctx context.Context,
	me GoroutineID,
	shouldWait bool,
) (result bool) {
	defer func() {
		if !result {
			return
		}
		m.lockCount++
		m.lockedBy = me
	}()
	for {
		if m.lockCount == 0 {
			if m.rlockCount == 0 {
				return true
			}
			if myReadersCountPtr, _ := m.usedBy[me]; myReadersCountPtr != nil {
				if m.rlockCount-*myReadersCountPtr == 0 {
					return true
				}
			}
		}
		if !shouldWait {
			m.internalLocker.Unlock()
			return false
		}
		if m.rlockDone == nil {
			m.rlockDone = make(chan struct{})
		}
		rlockDone := m.rlockDone
		if m.lockDone == nil {
			m.lockDone = make(chan struct{})
		}
		lockDone := m.lockDone
		isInfiniteContext := false
		if ctx == nil {
			ctx = m.infiniteContext()
			isInfiniteContext = true
		}
		m.internalLocker.Unlock()
		select {
		case <-rlockDone:
		case <-lockDone:
		case <-ctx.Done():
			if isInfiniteContext {
				m.debugPanic()
			}
			return false
		}
		m.internalLocker.Lock()
	}
}

// Unlock is analog of `(*sync.RWMutex)`.Unlock, but it cannot be called
// from a routine which does not hold the lock (see `Lock`).
func (m *RWMutex) Unlock() {
	me := GetGoroutineID()

	m.internalLocker.Lock()
	switch {
	case m.lockedBy == 0:
		m.internalLocker.Unlock()
		panic("An attempt to unlock a non-locked mutex.")
	case me != m.lockedBy:
		m.internalLocker.Unlock()
		panic(fmt.Sprintf("I'm not the one, who locked this mutex: %X != %X", me, m.lockedBy))
	}

	m.lockCount--
	if m.lockCount == 0 {
		m.lockedBy = 0
		goroutineClosedLock(m, true)
		m.backendLocker.Unlock()
	}

	chPtr := m.lockDone
	m.lockDone = nil
	m.internalLocker.Unlock()
	if chPtr != nil {
		close(chPtr)
	}
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

func (m *RWMutex) incMyReaders(me GoroutineID) {
	if v := m.usedBy[me]; v == nil {
		m.usedBy[me] = m.int64Pool.get()
		goroutineOpenedLock(m, false)
	} else {
		*v++
	}
	m.rlockCount++
}

func (m *RWMutex) gc() {
	m.gcCallCount++
	if m.gcCallCount != 0 {
		return
	}

	for k, v := range m.usedBy {
		if *v != 0 {
			continue
		}
		delete(m.usedBy, k)
		m.int64Pool.put(v)
	}
}

func (m *RWMutex) decMyReaders(me GoroutineID) {
	m.rlockCount--
	v := m.usedBy[me]
	if v == nil || *v == 0 {
		panic("RUnlock()-ing not RLock()-ed")
	}
	*v--
	if *v != 0 {
		return
	}
	goroutineClosedLock(m, false)
	m.gc()
	ch := m.rlockDone
	if ch == nil {
		return
	}
	close(ch)
	m.rlockDone = nil
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

func (m *RWMutex) rLock(
	ctx context.Context,
	shouldWait bool,
) bool {
	m.lazyInit()
	me := GetGoroutineID()

	m.internalLocker.Lock()
	for {
		if m.lockCount == 0 {
			break
		}
		monopolizedBy := m.lockedBy
		if monopolizedBy == me {
			break
		}

		if !shouldWait {
			m.internalLocker.Unlock()
			return false
		}

		if m.lockDone == nil {
			m.lockDone = make(chan struct{})
		}
		ch := m.lockDone

		isInfiniteContext := false
		if ctx == nil {
			ctx = m.infiniteContext()
			isInfiniteContext = true
		}
		m.internalLocker.Unlock()
		select {
		case <-ch:
		case <-ctx.Done():
			if isInfiniteContext {
				m.debugPanic()
			}
			return false
		}
		m.internalLocker.Lock()
	}

	m.incMyReaders(me)
	m.internalLocker.Unlock()
	return true
}

// RUnlock is analog of `(*sync.RWMutex)`.RUnlock, but it cannot be called
// from a routine which does not hold the lock (see `RLock`).
func (m *RWMutex) RUnlock() {
	me := GetGoroutineID()

	m.internalLocker.Lock()
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

func (m *RWMutex) debugPanic() {
	m.internalLocker.Lock()
	defer m.internalLocker.Unlock()
	debugPanic(m.lockedBy, m.usedBy)
}
