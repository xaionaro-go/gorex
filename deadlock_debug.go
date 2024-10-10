//go:build deadlockdebug
// +build deadlockdebug

package gorex

import (
	"fmt"
	"reflect"
	"runtime"
	"sync"

	"github.com/huandu/go-tls"
)

type debuggerLockerKey struct {
	LockerPtr uintptr
	IsWrite   bool
}

type debuggerPcs struct {
	pcs *[]uintptr
	n   int
}

type debuggerGStorage struct {
	PCS map[debuggerLockerKey]debuggerPcs
}

var (
	debuggerMap = &sync.Map{}
	exitC       chan struct{}
)

func getOrStoreDebuggerGStorage() *debuggerGStorage {
	stor, _ := debuggerMap.LoadOrStore(GetG(), &debuggerGStorage{})
	return stor.(*debuggerGStorage)
}

func getDebuggerLockerKey(lockPtr sync.Locker, isWrite bool) debuggerLockerKey {
	return debuggerLockerKey{
		LockerPtr: reflect.ValueOf(lockPtr).Elem().UnsafeAddr(),
		IsWrite:   isWrite,
	}
}

func goroutineCheckOnExit() {
	stor := getOrStoreDebuggerGStorage()
	for lKey, pcs := range stor.PCS {
		frames := runtime.CallersFrames((*pcs.pcs)[:pcs.n])
		fmt.Fprintf(debugPanicOut, "an opened lock %+v which was never released (and the goroutine already exited):\n",
			lKey)
		printFrames(frames)
	}
	debuggerMap.Delete(GetG())

	if exitC != nil {
		close(exitC)
		exitC = nil
	}
}

var pcsPool = sync.Pool{New: func() interface{} {
	pcs := make([]uintptr, 128)
	return &pcs
}}

func goroutineOpenedLock(lockPtr sync.Locker, isWrite bool) {
	pcs := pcsPool.Get().(*[]uintptr)
	n := runtime.Callers(2, *pcs)

	stor, lKey := getOrStoreDebuggerGStorage(), getDebuggerLockerKey(lockPtr, isWrite)
	if _, found := stor.PCS[lKey]; found {
		panic("should not happen")
	}
	if stor.PCS == nil {
		stor.PCS = map[debuggerLockerKey]debuggerPcs{}
	}
	stor.PCS[lKey] = debuggerPcs{pcs: pcs, n: n}

	_, ok := tls.Get("gorex-debugger")
	if ok {
		return
	}
	tls.AtExit(goroutineCheckOnExit)
	tls.Set("gorex-debugger", nil)
}

func goroutineClosedLock(lockPtr sync.Locker, isWrite bool) {
	stor, lKey := getOrStoreDebuggerGStorage(), getDebuggerLockerKey(lockPtr, isWrite)
	pcs := stor.PCS[lKey]
	if pcs.pcs != nil {
		pcsPool.Put(pcs.pcs)
	}
	delete(stor.PCS, lKey)
}
