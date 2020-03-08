package goroutine

import (
	"math"
	"sync"

	"github.com/xaionaro-go/rand/mathrand"

	"github.com/xaionaro-go/spinlock"
)

const (
	blockedByWriter = -math.MaxInt64 + 1
)

type RWLocker struct {
	lazyInitOnce sync.Once

	usedDone        chan struct{}
	monopolizedDone chan struct{}

	state          int64
	backendLocker  sync.Mutex
	internalLocker spinlock.Locker
	usedBy         map[*g]*int64
	monopolizedBy  *g
}

func (locker *RWLocker) lazyInit() {
	locker.lazyInitOnce.Do(func() {
		locker.usedBy = map[*g]*int64{}
	})
}

func (locker *RWLocker) LockDo(fn func()) {
	locker.lazyInit()

	me := GetG()

	var monopolizedByWasSetHere bool
	defer func() {
		locker.internalLocker.Lock()
		if monopolizedByWasSetHere {
			locker.monopolizedBy = nil
			locker.backendLocker.Unlock()
			locker.state -= blockedByWriter
		}

		chPtr := locker.monopolizedDone
		locker.monopolizedDone = nil
		locker.internalLocker.Unlock()
		if chPtr == nil {
			return
		}
		close(chPtr)
	}()

	for {
		var monopolizedBy *g
		locker.internalLocker.Lock()
		if locker.monopolizedBy == nil {
			locker.monopolizedBy = me
			monopolizedBy = me
			monopolizedByWasSetHere = true
			locker.internalLocker.Unlock()
			break
		} else {
			monopolizedBy = locker.monopolizedBy
		}
		monopolizedByMe := monopolizedBy == me
		var ch chan struct{}
		if !monopolizedByMe {
			if locker.monopolizedDone == nil {
				locker.monopolizedDone = make(chan struct{})
			}
			ch = locker.monopolizedDone
		}
		locker.internalLocker.Unlock()
		if monopolizedByMe {
			break
		}
		select {
		case <-ch:
		}
	}
	if monopolizedByWasSetHere {
		locker.backendLocker.Lock()
		locker.setStateBlockedByWriter(me)
	}

	fn()
}

func (locker *RWLocker) setStateBlockedByWriter(me *g) {
	locker.internalLocker.Lock()
	defer locker.internalLocker.Unlock()
	for {
		locker.state += blockedByWriter
		if locker.state == blockedByWriter {
			return
		}
		if myReadersCountPtr, _ := locker.usedBy[me]; myReadersCountPtr != nil {
			if locker.state-*myReadersCountPtr == blockedByWriter {
				return
			}
		}
		locker.state -= blockedByWriter

		if locker.usedDone == nil {
			locker.usedDone = make(chan struct{})
		}
		ch := locker.usedDone
		locker.internalLocker.Unlock()
		select {
		case <-ch:
		}
		locker.internalLocker.Lock()
	}
}

type int64PoolT []*int64

func (pool *int64PoolT) Put(v *int64) {
	*v = 1
	*pool = append(*pool, v)
}

func (pool *int64PoolT) Get() *int64 {
	if len(*pool) == 0 {
		for i := 0; i < 100; i++ {
			pool.Put(&[]int64{1}[0])
		}
	}

	idx := len(*pool) - 1
	v := (*pool)[idx]
	*pool = (*pool)[:idx]

	return v
}

var int64Pool = &int64PoolT{}

func (locker *RWLocker) incMyReaders(me *g) {
	if v := locker.usedBy[me]; v == nil {
		locker.usedBy[me] = int64Pool.Get()
	} else {
		*v++
	}
}

var prng = mathrand.New()

func (locker *RWLocker) gc() {
	if prng.Uint32MultiplyAdd()>>24 != 0 {
		return
	}

	for k, v := range locker.usedBy {
		if *v != 0 {
			continue
		}
		delete(locker.usedBy, k)
	}
}

func (locker *RWLocker) decMyReaders(me *g) {
	v := locker.usedBy[me]
	*v--
	if *v != 0 {
		return
	}
	locker.gc()
	ch := locker.usedDone
	if ch == nil {
		return
	}
	close(ch)
	locker.usedDone = nil
	int64Pool.Put(v)
}

func (locker *RWLocker) RLockDo(fn func()) {
	locker.lazyInit()

	me := GetG()

	for {
		locker.internalLocker.Lock()
		locker.state++
		isOK := locker.state > 0
		if isOK {
			break
		}
		monopolizedBy := locker.monopolizedBy
		if monopolizedBy == me {
			break
		}

		locker.state--
		if locker.monopolizedDone == nil {
			locker.monopolizedDone = make(chan struct{})
		}
		ch := locker.monopolizedDone
		locker.internalLocker.Unlock()

		select {
		case <-ch:
		}
	}

	locker.incMyReaders(me)
	locker.internalLocker.Unlock()

	defer func() {
		locker.internalLocker.Lock()
		locker.state--
		locker.decMyReaders(me)
		locker.internalLocker.Unlock()
	}()

	fn()
}
