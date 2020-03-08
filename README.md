# gorex

###`gorex == GORoutine mutual EXclusion`

This package implements `Mutex` and `RWMutex`. They are similar to `sync.Mutex` and `sync.RWMutex`, but
they track which goroutine locked the mutex and will not cause a deadlock if
the same goroutine will try to lock the same mutex again.

```go
type myEntity struct {
    gorex.Mutex
}

func (ent *myEntity) func1() {
    ent.LockDo(func() {
        .. some stuff ..
        ent.func2() // will not get a deadlock here!
        .. other stuff ..
    })
}

func (ent *myEntity) func2() {
    ent.LockDo(func(){
        .. more stuff ..
    })
}
```

```go
locker := &goroutine.RWMutex{}

locker.RLockDo(func() {
    .. do some read-only stuff ..
    if cond {
      return
    }
    locker.LockDo(func() { // will not get a deadlock here!
        .. do write stuff ..
    })
})
```

#### But...

But you still will get a deadlock if you do this way:
```go
var locker = &gorex.RWMutex{}

func someFunc() {
    locker.RLockDo(func() {
        .. do some read-only stuff ..
        if cond {
          return
        }
        locker.LockDo(func() { // you will not get a deadlock here!
            .. do write stuff ..
        })
    })
}()

func main() {
    go someFunc()
    go someFunc()
}
```
because there could be a situation that a resource is blocked by a `RLockDo` from
both goroutines and both goroutines waits (on `LockDo`) until other goroutine
will finish `RLockDo`. On the other side you will easily see this `LockDo`'s in the
call stack trace.

#### Benchmark

It's essentially slower than bare `sync.Mutex`/`sync.RWMutex`:

```
goos: linux
goarch: amd64
pkg: github.com/xaionaro-go/goroutine
Benchmark/Lock-Unlock/single/sync.Mutex-8         	33334248	        35.6 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/single/sync.RWMutex-8       	28840336	        41.5 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/single/Mutex-8              	19785918	        61.2 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/single/RWMutex-8            	12316568	        94.3 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/sync.Mutex-8       	 9204102	       129 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/sync.RWMutex-8     	 7640816	       158 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/Mutex-8            	 7252591	       158 ns/op	       4 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/RWMutex-8          	 4778331	       239 ns/op	       6 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/single/sync.RWMutex-8     	34093424	        35.5 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/single/RWMutex-8          	18106015	        64.6 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/parallel/sync.RWMutex-8   	24507448	        46.4 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/parallel/RWMutex-8        	 9100104	       133 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-ed:Lock-Unlock/single/Mutex-8      	22485800	        47.4 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-ed:Lock-Unlock/single/RWMutex-8    	24491390	        48.7 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-ed:RLock-RUnlock/single/RWMutex-8 	19400768	        61.5 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-ed:RLock-RUnlock/parallel/RWMutex-8         	 8251312	       151 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/xaionaro-go/goroutine	20.463s
```

But sometimes it allows you to think more about strategically problems
("this stuff should be edited atomically, so I'll be able to...")
instead of wasting time on tactical problems ("how to handle those locks") :)

## GetG

Function `GetG` is just the function used to determine which goroutine is this. Two
different active goroutines has different pointers (returned by `GetG`).
