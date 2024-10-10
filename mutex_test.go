package gorex

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestMutex(t *testing.T) {
	t.Run("Unlock", func(t *testing.T) {
		t.Run("negative", func(t *testing.T) {
			t.Run("notLocked", func(t *testing.T) {
				var result interface{}
				func() {
					defer func() {
						result = recover()
					}()
					locker := &Mutex{}
					locker.Unlock()
				}()
				assert.NotNil(t, result)
				assert.Equal(t, -1, strings.Index(fmt.Sprint(result), "pointer dereference"), result)
			})
		})
	})
	t.Run("LockDo", func(t *testing.T) {
		t.Run("positive", func(t *testing.T) {
			locker := &Mutex{}
			locker.LockDo(func() {
				locker.LockDo(func() {
				})
			})

			var wg sync.WaitGroup
			wg.Add(1)
			i := 0
			locker.LockDo(func() {
				go locker.LockDo(func() {
					defer wg.Done()
					i = 2
				})
				locker.LockDo(func() {
					time.Sleep(time.Microsecond)
					i = 1
				})
			})

			wg.Wait()
			assert.Equal(t, 2, i)
		})
		t.Run("negative", func(t *testing.T) {
			t.Run("endOfInfinityContext", func(t *testing.T) {
				debugPanicOut = io.Discard

				var result interface{}
				func() {
					var wg0 sync.WaitGroup
					defer func() {
						result = recover()
						wg0.Done()
					}()

					var cancelFn context.CancelFunc
					locker := &Mutex{}
					locker.InfiniteContext, cancelFn = context.WithDeadline(context.Background(), time.Now())
					defer cancelFn()
					var wg1 sync.WaitGroup
					wg0.Add(1)
					wg1.Add(1)
					go locker.LockDo(func() {
						wg1.Done()
						wg0.Wait()
					})
					wg1.Wait()
					locker.Lock()
				}()

				assert.NotNil(t, result, result)
			})
		})
	})
	t.Run("LockTryDo", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			locker := &Mutex{}
			i := 0
			assert.True(t, locker.LockTryDo(func() {
				i = 1
			}))
			assert.Equal(t, 1, i)
		})
		t.Run("false", func(t *testing.T) {
			locker := &Mutex{}
			i := 0
			var wg0 sync.WaitGroup
			var wg1 sync.WaitGroup
			wg0.Add(1)
			wg1.Add(1)
			go locker.LockDo(func() {
				wg1.Done()
				wg0.Wait()
			})
			wg1.Wait()
			assert.False(t, locker.LockTryDo(func() {
				i = 1
			}))
			assert.Equal(t, 0, i)
			wg0.Done()
		})
	})
	t.Run("LockCtxDo", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			locker := &Mutex{}
			i := 0
			assert.True(t, locker.LockCtxDo(context.Background(), func() {
				i = 1
			}))
			assert.Equal(t, 1, i)
		})
		t.Run("false", func(t *testing.T) {
			locker := &Mutex{}
			i := 0
			var wg0 sync.WaitGroup
			var wg1 sync.WaitGroup
			wg0.Add(1)
			wg1.Add(1)
			go locker.LockDo(func() {
				wg1.Done()
				wg0.Wait()
			})
			wg1.Wait()
			ctx, cancelFn := context.WithDeadline(context.Background(), time.Now().Add(time.Microsecond))
			defer cancelFn()
			assert.False(t, locker.LockCtxDo(ctx, func() {
				i = 1
			}))
			assert.Equal(t, 0, i)
			wg0.Done()
		})
	})
}
