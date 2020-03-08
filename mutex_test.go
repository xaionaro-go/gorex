package goroutine

import (
	"fmt"
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
}
