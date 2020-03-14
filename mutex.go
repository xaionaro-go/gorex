package gorex

import (
	"context"
	"fmt"
	"sync"

	"github.com/xaionaro-go/spinlock"
)

// DefaultInfiniteContext is used as the default context used on any try to lock if
// a custom context is not set (see LockCtx/RLockCtx), but with the difference
// if this context will be done, then it will panic with debugging information.
//
// To specify a context with deadline may be useful for unit tests.
var DefaultInfiniteContext = context.Background()

// Mutex is a goroutine-aware analog of sync.Mutex, so it works
// the same way as sync.Mutex, but tracks which goroutine locked
// it. So it could be locked multiple times with the same routine.
type Mutex struct {
	// InfiniteContext is used as the default context used on any try to lock if
	// a custom context is not set (see LockCtx), but with the difference
	// if this context will be done, then it will panic with debugging information.
	//
	// To specify a context with deadline may be useful for unit tests.
	//
	// The zero-value means to use DefaultInfiniteContext.
	InfiniteContext context.Context

	backendLocker    sync.Mutex
	internalLocker   spinlock.Locker
	monopolizedBy    *G
	monopolizedDepth int
	lockDone         chan struct{}
}

// Lock is analog of `(*sync.Mutex)`.Lock, but it allows one goroutine
// to call it multiple times without calling Unlock.
func (m *Mutex) Lock() {
	m.lock(nil, true)
}

// LockTry is analog of Lock(), but it does not block if it cannot lock
// right away.
//
// Returns `false` if was unable to lock.
func (m *Mutex) LockTry() bool {
	return m.lock(nil, false)
}

// LockCtx is analog of Lock(), but allows to continue the try to lock only until context is done..
//
// Returns `false` if was unable to lock (context finished before it was possible to lock).
func (m *Mutex) LockCtx(ctx context.Context) bool {
	return m.lock(ctx, true)
}

func (m *Mutex) infiniteContext() context.Context {
	if m.InfiniteContext == nil {
		return DefaultInfiniteContext
	}
	return m.InfiniteContext
}

func (m *Mutex) lock(ctx context.Context, shouldWait bool) bool {
	me := GetG()

	for {
		m.internalLocker.Lock()
		switch m.monopolizedBy {
		case nil:
			m.monopolizedBy = me
			m.monopolizedDepth++
			goroutineOpenedLock(m, true)
			m.internalLocker.Unlock()
			m.backendLocker.Lock()
			return true
		case me:
			m.monopolizedDepth++
			m.internalLocker.Unlock()
			return true
		}
		if !shouldWait {
			m.internalLocker.Unlock()
			return false
		}
		var ch chan struct{}
		if m.lockDone == nil {
			m.lockDone = make(chan struct{})
		}
		ch = m.lockDone
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
	}
}

// Unlock is analog of `(*sync.Mutex)`.Unlock, but it cannot be called
// from a routine which does not hold the lock (see `Lock`).
func (m *Mutex) Unlock() {
	me := GetG()
	m.internalLocker.Lock()
	switch {
	case m.monopolizedBy == nil:
		m.internalLocker.Unlock()
		panic("An attempt to unlock a non-locked mutex.")
	case me != m.monopolizedBy:
		m.internalLocker.Unlock()
		panic(fmt.Sprintf("I'm not the one, who locked this mutex: %p != %p", me, m.monopolizedBy))
	}
	m.monopolizedDepth--
	if m.monopolizedDepth == 0 {
		m.monopolizedBy = nil
		goroutineClosedLock(m, true)
		m.backendLocker.Unlock()
	}
	chPtr := m.lockDone
	m.lockDone = nil
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

// LockTryDo is a wrapper around LockTry and Unlock.
//
// See also LockDo and LockTry.
func (m *Mutex) LockTryDo(fn func()) (success bool) {
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
func (m *Mutex) LockCtxDo(ctx context.Context, fn func()) (success bool) {
	if !m.LockCtx(ctx) {
		return false
	}
	defer m.Unlock()

	success = true
	fn()
	return
}

func (m *Mutex) debugPanic() {
	m.internalLocker.Lock()
	defer m.internalLocker.Unlock()
	debugPanic(m, m.monopolizedBy, nil)
}
