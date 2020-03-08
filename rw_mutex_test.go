package goroutine

import (
	"sync"
	"testing"
	"time"
)

type lockerWrapper struct {
	sync.Mutex
}

func (w *lockerWrapper) LockDo(fn func()) {
	w.Lock()
	defer w.Unlock()

	fn()
}

type rwLockerWrapper struct {
	sync.RWMutex
}

func (w *rwLockerWrapper) LockDo(fn func()) {
	w.Lock()
	defer w.Unlock()

	fn()
}
func (w *rwLockerWrapper) RLockDo(fn func()) {
	w.RLock()
	defer w.RUnlock()

	fn()
}

type locker interface {
	LockDo(func())
}
type rwLocker interface {
	locker
	RLockDo(func())
}

func benchmarkLockUnlock(b *testing.B, locker locker) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		locker.LockDo(func() {})
	}
}

func benchmarkLockedLockUnlock(b *testing.B, locker locker) {
	locker.LockDo(func() {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			locker.LockDo(func() {})
		}
	})
}

func benchmarkRLockedRLockRUnlock(b *testing.B, locker rwLocker) {
	locker.RLockDo(func() {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			locker.RLockDo(func() {})
		}
	})
}

func benchmarkParallelLockUnlock(b *testing.B, locker locker) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker.LockDo(func() {})
		}
	})
}

func benchmarkRLockRUnlock(b *testing.B, locker rwLocker) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		locker.RLockDo(func() {})
	}
}

func benchmarkParallelRLockRUnlock(b *testing.B, locker rwLocker) {
	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			locker.RLockDo(func() {})
		}
	})
}

func benchmarkRLockedParallelRLockRUnlock(b *testing.B, locker rwLocker) {
	locker.RLockDo(func() {
		b.ReportAllocs()
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				locker.RLockDo(func() {})
			}
		})
	})
}

func Benchmark(b *testing.B) {
	b.Run("Lock-Unlock", func(b *testing.B) {
		b.Run("single", func(b *testing.B) {
			b.Run("sync.Mutex", func(b *testing.B) {
				benchmarkLockUnlock(b, &lockerWrapper{sync.Mutex{}})
			})
			b.Run("sync.RWMutex", func(b *testing.B) {
				benchmarkLockUnlock(b, &rwLockerWrapper{sync.RWMutex{}})
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
				benchmarkParallelLockUnlock(b, &lockerWrapper{sync.Mutex{}})
			})
			b.Run("sync.RWMutex", func(b *testing.B) {
				benchmarkParallelLockUnlock(b, &rwLockerWrapper{sync.RWMutex{}})
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
				benchmarkRLockRUnlock(b, &rwLockerWrapper{sync.RWMutex{}})
			})
			b.Run("RWMutex", func(b *testing.B) {
				benchmarkRLockRUnlock(b, &RWMutex{})
			})
		})
		b.Run("parallel", func(b *testing.B) {
			b.Run("sync.RWMutex", func(b *testing.B) {
				benchmarkParallelRLockRUnlock(b, &rwLockerWrapper{sync.RWMutex{}})
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

func TestRWLocker(t *testing.T) {
	testRWLocker(t, &RWMutex{})
}

func testRWLocker(t *testing.T, locker rwLocker) {
	locker.LockDo(func() {})
	locker.LockDo(func() {})
	locker.RLockDo(func() {})

	var wg sync.WaitGroup
	wg.Add(200)
	for i := 0; i < 100; i++ {
		go func() {
			locker.LockDo(func() {})
			wg.Done()
		}()
		go func() {
			locker.RLockDo(func() {})
			wg.Done()
		}()
	}

	wg.Wait()

	wg.Add(200)
	for i := 0; i < 100; i++ {
		go func() {
			locker.LockDo(func() {
				time.Sleep(time.Microsecond)
			})
			wg.Done()
		}()
		go func() {
			locker.RLockDo(func() {
				time.Sleep(time.Microsecond)
			})
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
}
