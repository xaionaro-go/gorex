package goroutine

import (
	"sync"
	"testing"
	"time"
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

func TestRWLocker(t *testing.T) {
	testRWLocker(t, &RWMutex{})
}

func testRWLocker(t *testing.T, locker *RWMutex) {
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
}
