# Goroutine tools

## Locker

This package implements `Locker` and `RWLocker`. They are similar to `sync.Mutex` and `sync.RWMutex`, but
they track which goroutine locked the locker and will not cause a deadlock if
a goroutine will try to lock something it already locked previously.

```go
type myEntity struct {
    goroutine.Locker
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
locker := &goroutine.RWLocker{}

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
var locker = &goroutine.RWLocker{}

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
Benchmark/Lock-Unlock/single/sync.Mutex-8         	33510717	        35.4 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/single/sync.RWMutex-8       	29253985	        41.2 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/single/Locker-8             	20195449	        59.5 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/single/RWLocker-8           	12731053	        93.8 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/sync.Mutex-8       	 9247928	       131 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/sync.RWMutex-8     	 7582437	       158 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/Locker-8           	 8368058	       182 ns/op	       5 B/op	       0 allocs/op
Benchmark/Lock-Unlock/parallel/RWLocker-8         	 4653135	       255 ns/op	       6 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/single/sync.RWMutex-8     	33995299	        35.9 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/single/RWLocker-8         	18515317	        64.1 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/parallel/sync.RWMutex-8   	23826394	        50.3 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-RUnlock/parallel/RWLocker-8       	 8961405	       141 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-ed:Lock-Unlock/single/Locker-8     	24920245	        47.2 ns/op	       0 B/op	       0 allocs/op
Benchmark/Lock-ed:Lock-Unlock/single/RWLocker-8   	24562058	        48.2 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-ed:RLock-RUnlock/single/RWLocker-8         	19546357	        60.9 ns/op	       0 B/op	       0 allocs/op
Benchmark/RLock-ed:RLock-RUnlock/parallel/RWLocker-8       	 7760634	       159 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/xaionaro-go/goroutine	21.128s
```

But sometimes it allows you to think more about strategically problems
("this stuff should be edited atomically, so I'll be able to...")
instead of wasting time on tactical problems ("how to handle those locks") :)

## GetG

Function `GetG` is just the function used to determine which goroutine is this. Two
different active goroutines has different pointers (returned by `GetG`).
