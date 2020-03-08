// +build deadlockdebug

package gorex

import (
	"testing"
	"time"
)

type writeWaiter struct {
	c chan struct{}
}

func (w *writeWaiter) Write(b []byte) (int, error) {
	close(w.c)
	return len(b), nil
}

func TestDeadlockDebug(t *testing.T) {
	t.Run("positive", func(t *testing.T) {
		t.Run("(*Mutex).Lock", func(t *testing.T) {
			exitC = make(chan struct{})
			waiter := writeWaiter{c: make(chan struct{})}
			debugPanicOut = &waiter
			go func() {
				locker := &Mutex{}
				locker.Lock()
			}()

			time.Sleep(time.Microsecond)
			select {
			case <-waiter.c:
			case <-time.After(time.Millisecond):
				t.Errorf("no debug info received :(")
			case <-exitC:
				t.Errorf("no debug info received :(")
			}
		})
	})
	t.Run("negative", func(t *testing.T) {
		t.Run("(*Mutex).Lock&Unlock", func(t *testing.T) {
			exitC = make(chan struct{})
			waiter := writeWaiter{c: make(chan struct{})}
			debugPanicOut = &waiter
			go func() {
				locker := &Mutex{}
				locker.Lock()
				locker.Unlock()
			}()

			time.Sleep(time.Microsecond)
			select {
			case <-waiter.c:
				t.Errorf("received debug info, while shouldn't")
			case <-time.After(time.Millisecond):
			case <-exitC:
			}
		})
	})
}
