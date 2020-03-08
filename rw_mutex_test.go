package gorex

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type locker interface {
	sync.Locker
}
type rwLocker interface {
	locker
	RLock()
	RUnlock()
}

func benchmarkLockUnlock(b *testing.B, locker locker) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		locker.Lock()
		locker.Unlock()
	}
}

func benchmarkLockedLockUnlock(b *testing.B, locker locker) {
	locker.Lock()
	defer locker.Unlock()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		locker.Lock()
		locker.Unlock()
	}

	b.StopTimer()
}

func benchmarkRLockedRLockRUnlock(b *testing.B, locker rwLocker) {
	locker.RLock()
	defer locker.RUnlock()

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		locker.RLock()
		locker.RUnlock()
	}
	b.StopTimer()
}

func benchmarkParallelLockUnlock(b *testing.B, locker locker) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker.Lock()
			locker.Unlock()
		}
	})
}

func benchmarkRLockRUnlock(b *testing.B, locker rwLocker) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		locker.RLock()
		locker.RUnlock()
	}
}

func benchmarkParallelRLockRUnlock(b *testing.B, locker rwLocker) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker.RLock()
			locker.RUnlock()
		}
	})
}

func benchmarkRLockedParallelRLockRUnlock(b *testing.B, locker rwLocker) {
	locker.RLock()
	defer locker.RUnlock()

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker.RLock()
			locker.RUnlock()
		}
	})

	b.StopTimer()
}

func Benchmark(b *testing.B) {
	b.Run("Lock-Unlock", func(b *testing.B) {
		b.Run("single", func(b *testing.B) {
			b.Run("sync.Mutex", func(b *testing.B) {
				benchmarkLockUnlock(b, &sync.Mutex{})
			})
			b.Run("sync.RWMutex", func(b *testing.B) {
				benchmarkLockUnlock(b, &sync.RWMutex{})
			})
			b.Run("Mutex", func(b *testing.B) {
				benchmarkLockUnlock(b, &Mutex{})
			})
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkLockUnlock(b, &RWMutex{})
			})
		})
		b.Run("parallel", func(b *testing.B) {
			b.Run("sync.Mutex", func(b *testing.B) {
				benchmarkParallelLockUnlock(b, &sync.Mutex{})
			})
			b.Run("sync.RWMutex", func(b *testing.B) {
				benchmarkParallelLockUnlock(b, &sync.RWMutex{})
			})
			b.Run("Mutex", func(b *testing.B) {
				benchmarkParallelLockUnlock(b, &Mutex{})
			})
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkParallelLockUnlock(b, &RWMutex{})
			})
		})
	})
	b.Run("RLock-RUnlock", func(b *testing.B) {
		b.Run("single", func(b *testing.B) {
			b.Run("sync.RWMutex", func(b *testing.B) {
				benchmarkRLockRUnlock(b, &sync.RWMutex{})
			})
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkRLockRUnlock(b, &RWMutex{})
			})
		})
		b.Run("parallel", func(b *testing.B) {
			b.Run("sync.RWMutex", func(b *testing.B) {
				benchmarkParallelRLockRUnlock(b, &sync.RWMutex{})
			})
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkParallelRLockRUnlock(b, &RWMutex{})
			})
		})
	})
	b.Run("Lock-ed:Lock-Unlock", func(b *testing.B) {
		b.Run("single", func(b *testing.B) {
			b.Run("Mutex", func(b *testing.B) {
				benchmarkLockedLockUnlock(b, &Mutex{})
			})
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkLockedLockUnlock(b, &RWMutex{})
			})
		})
	})
	b.Run("RLock-ed:RLock-RUnlock", func(b *testing.B) {
		b.Run("single", func(b *testing.B) {
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkRLockedRLockRUnlock(b, &RWMutex{})
			})
		})
		b.Run("parallel", func(b *testing.B) {
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkRLockedParallelRLockRUnlock(b, &RWMutex{})
			})
		})
	})
}

func TestRWMutex(t *testing.T) {
	t.Run("Unlock", func(t *testing.T) {
		t.Run("negative", func(t *testing.T) {
			t.Run("notLocked", func(t *testing.T) {
				var result interface{}
				func() {
					defer func() {
						result = recover()
					}()
					locker := &RWMutex{}
					locker.RLock()
					locker.Unlock()
				}()
				assert.NotNil(t, result)
				assert.Equal(t, -1, strings.Index(fmt.Sprint(result), "pointer dereference"), result)
			})
		})
	})
	t.Run("RUnlock", func(t *testing.T) {
		t.Run("negative", func(t *testing.T) {
			t.Run("notLocked", func(t *testing.T) {
				var result interface{}
				func() {
					defer func() {
						result = recover()
					}()
					locker := &RWMutex{}
					locker.Lock()
					locker.RUnlock()
				}()
				assert.NotNil(t, result)
				assert.Equal(t, -1, strings.Index(fmt.Sprint(result), "pointer dereference"), result)
			})
		})
	})
	t.Run("LockDo", func(t *testing.T) {
		locker := &RWMutex{}
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
	t.Run("RLockDo", func(t *testing.T) {
		locker := &RWMutex{}
		locker.RLockDo(func() {
			locker.RLockDo(func() {
			})
		})

		var wg sync.WaitGroup
		wg.Add(1)
		i := 0
		locker.RLockDo(func() {
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

		wg.Add(1)
		locker.LockDo(func() {
			go locker.RLockDo(func() {
				defer wg.Done()
				i = 2
			})
			locker.RLockDo(func() {
				time.Sleep(time.Microsecond)
				i = 1
			})
		})

		wg.Wait()
		assert.Equal(t, 2, i)
	})
	t.Run("extraTests", func(t *testing.T) {
		locker := &RWMutex{}
		locker.Lock()
		locker.Unlock()
		locker.Lock()
		locker.Unlock()
		locker.RLock()
		locker.RUnlock()

		var wg sync.WaitGroup
		wg.Add(200)
		for i := 0; i < 100; i++ {
			go func() {
				locker.Lock()
				locker.Unlock()
				wg.Done()
			}()
			go func() {
				locker.RLock()
				locker.RUnlock()
				wg.Done()
			}()
		}

		wg.Wait()

		wg.Add(200)
		for i := 0; i < 100; i++ {
			go func() {
				locker.Lock()
				time.Sleep(time.Microsecond)
				locker.Unlock()
				wg.Done()
			}()
			go func() {
				locker.RLock()
				time.Sleep(time.Microsecond)
				locker.RUnlock()
				wg.Done()
			}()
		}

		wg.Wait()

		locker.RLockDo(func() {
			locker.LockDo(func() {
				locker.RLockDo(func() {
				})
			})
		})
		locker.LockDo(func() {
			locker.RLockDo(func() {
				locker.LockDo(func() {
				})
			})
		})

		locker.RLockDo(func() {
			locker.LockDo(func() {
				locker.RLockDo(func() {
					locker.LockDo(func() {
						locker.RLockDo(func() {
							locker.LockDo(func() {
							})
						})
					})
				})
			})
		})
	})
	t.Run("LockTryDo", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			locker := &RWMutex{}
			i := 0
			assert.True(t, locker.LockTryDo(func() {
				i = 1
			}))
			assert.Equal(t, 1, i)
		})
		t.Run("false", func(t *testing.T) {
			locker := &RWMutex{}
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
			locker := &RWMutex{}
			i := 0
			assert.True(t, locker.LockCtxDo(context.Background(), func() {
				i = 1
			}))
			assert.Equal(t, 1, i)
		})
		t.Run("false", func(t *testing.T) {
			locker := &RWMutex{}
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
	t.Run("RLockTryDo", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			locker := &RWMutex{}
			i := 0
			assert.True(t, locker.RLockTryDo(func() {
				i = 1
			}))
			assert.Equal(t, 1, i)
		})
		t.Run("false", func(t *testing.T) {
			locker := &RWMutex{}
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
			assert.False(t, locker.RLockTryDo(func() {
				i = 1
			}))
			assert.Equal(t, 0, i)
			wg0.Done()
		})
	})
	t.Run("RLockCtxDo", func(t *testing.T) {
		t.Run("true", func(t *testing.T) {
			locker := &RWMutex{}
			i := 0
			assert.True(t, locker.RLockCtxDo(context.Background(), func() {
				i = 1
			}))
			assert.Equal(t, 1, i)
		})
		t.Run("false", func(t *testing.T) {
			locker := &RWMutex{}
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
			assert.False(t, locker.RLockCtxDo(ctx, func() {
				i = 1
			}))
			assert.Equal(t, 0, i)
			wg0.Done()
		})
	})
}

func TestRWLocker(t *testing.T) {
	testRWMutex(t, &RWMutex{})
}

func testRWMutex(t *testing.T, locker *RWMutex) {
}
