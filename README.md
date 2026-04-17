![Gopher](assets/gopher.png)
[![CircleCI](https://circleci.com/gh/alexfalkowski/go-sync.svg?style=shield)](https://circleci.com/gh/alexfalkowski/go-sync)
[![codecov](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexfalkowski/go-sync)](https://goreportcard.com/report/github.com/alexfalkowski/go-sync)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/go-sync.svg)](https://pkg.go.dev/github.com/alexfalkowski/go-sync)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# go-sync

A small Go library (package `sync`) with focused concurrency helpers:

- Convenience aliases for common sync primitives and typed atomics
- Hook-driven execution (`Wait`, `Timeout`, `Worker`)
- Group helpers (`ErrorGroup`, `SingleFlightGroup`)
- Typed wrappers for `sync.Pool`, `sync.Map`, and `atomic.Value`
- A `bytes.Buffer` pool specialized for copy-and-reuse workflows

## Install

```bash
go get github.com/alexfalkowski/go-sync
```

## Package layout

The public API is intentionally small:

- Aliases: `Once`, `Mutex`, `RWMutex`, `WaitGroup`, `Int32`, `Int64`, `Uint32`, `Uint64`, `Uintptr`, `Bool`, `Pointer[T]`
- Hooks and timeout helpers: `Hook`, `ErrTimeout`, `Wait`, `Timeout`, `IsTimeoutError`
- Worker: `NewWorker`, `Worker.Schedule`, `Worker.Wait`
- Groups: `ErrorGroup`, `NewSingleFlightGroup`, `SingleFlightGroup`
- Pools and wrappers: `NewPool`, `Pool[T]`, `NewBufferPool`, `BufferPool`, `NewValue`, `Value[T]`, `NewMap`, `Map[K,V]`

Most wrappers preserve the semantics of the standard library type they wrap while making those semantics easier to use from generic code.

## Aliases

The package re-exports a few commonly used concurrency primitives and helper types for convenience:

- `Once`, `Mutex`, `RWMutex`, and `WaitGroup` alias their counterparts in `sync`.
- `Int32`, `Int64`, `Uint32`, `Uint64`, `Uintptr`, `Bool`, and `Pointer[T]` alias typed atomics from `sync/atomic`.
- `ErrorGroup` aliases `errgroup.Group`.

These are type aliases rather than wrappers, so their behavior is exactly the same as the underlying type.

## Hooks

Most execution helpers accept a `sync.Hook`:

- `OnRun(context.Context) error` is required.
- `OnError(context.Context, error) error` is optional.
- If `OnRun` is nil, helpers return `sync.ErrNoOnRunProvided`.

`OnError` is only called when `OnRun` returns a non-nil error. If `OnError` returns a different error, that new error is returned.

How that returned error is observed depends on the helper:

- `Wait` and `Timeout` return it only if `OnRun` finishes before their timeout/cancellation path wins.
- `Worker` never returns handler errors from `Schedule`; use `OnError` for logging or side effects.

```go
package main

import (
    "context"
    "errors"
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    hook := sync.Hook{
        OnRun: func(context.Context) error {
            return errors.New("boom")
        },
        OnError: func(_ context.Context, err error) error {
            return fmt.Errorf("wrapped: %w", err)
        },
    }

    _ = hook
}
```

## Wait vs Timeout

`Wait` and `Timeout` both run `Hook.OnRun`, but they differ:

- `Wait`: best-effort wait up to `timeout`; returns `nil` on timeout/cancel and does not cancel `OnRun`.
- `Timeout`: derives a timeout context for `OnRun`; returns the context cause when the context ends first (`sync.ErrTimeout`, `context.Canceled`, or a parent-provided cause).
- If the input context is already done, `Wait` returns `nil` immediately and `Timeout` returns the input context's cause immediately (neither invokes `OnRun`).
- If `timeout <= 0`, `Wait` returns `nil` immediately, while `Timeout` returns `sync.ErrTimeout` immediately.
- Neither helper forcibly stops `OnRun`; if the handler ignores context cancellation, it can continue running in the background.

Use `sync.IsTimeoutError(err)` to check if an error matches `sync.ErrTimeout` or `context.DeadlineExceeded`.

### Wait example (best effort)

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "time"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    err := sync.Wait(context.Background(), 10*time.Millisecond, sync.Hook{
        OnRun: func(context.Context) error {
            time.Sleep(time.Second)
            return errors.New("finished too late")
        },
    })

    // true: Wait timed out first.
    fmt.Println(err == nil)
}
```

### Timeout example (propagated cancellation)

```go
package main

import (
    "context"
    "fmt"
    "time"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    err := sync.Timeout(context.Background(), 10*time.Millisecond, sync.Hook{
        OnRun: func(ctx context.Context) error {
            <-ctx.Done()
            return context.Cause(ctx)
        },
    })

    fmt.Println(sync.IsTimeoutError(err))
}
```

## Worker

`Worker` schedules asynchronous handlers with bounded concurrency.

- Zero value is not ready; use `NewWorker(count)`.
- `NewWorker(count)` returns a ready-to-use pointer to a worker with at most `count` in-flight handlers.
- `Schedule` blocks until a slot is acquired or timeout/cancel happens.
- The `timeout` budget starts when `Schedule` is called, so queue wait time and handler run time share the same deadline.
- `Schedule` returns only scheduling errors (the derived context cause or `ErrNoOnRunProvided`).
- Handler errors are routed to `Hook.OnError` and are not returned by `Schedule`.
- Once a handler has been scheduled, `Schedule` returns `nil` even if that handler later observes `ctx.Done()`.
- `Wait` blocks until all successfully scheduled handlers complete.
- If the input context is already canceled, `Schedule` returns the input context's cause immediately and does not schedule `OnRun`.

If `count == 0`, scheduling always blocks until timeout/cancel.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    worker := sync.NewWorker(4)
    defer worker.Wait()

    for i := 0; i < 3; i++ {
        job := i
        err := worker.Schedule(context.Background(), time.Second, sync.Hook{
            OnRun: func(context.Context) error {
                fmt.Println("job", job)
                return nil
            },
            OnError: func(_ context.Context, err error) error {
                log.Printf("job failed: %v", err)
                return err
            },
        })
        if err != nil {
            log.Printf("schedule failed: %v", err)
        }
    }
}
```

## Group

### ErrorGroup / WaitGroup

`sync.ErrorGroup` is a type alias for [`errgroup.Group`](https://pkg.go.dev/golang.org/x/sync/errgroup#Group).
`sync.WaitGroup` is a type alias for [`sync.WaitGroup`](https://pkg.go.dev/sync#WaitGroup).

```go
package main

import (
    "errors"
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    var g sync.ErrorGroup

    g.Go(func() error { return nil })
    g.Go(func() error { return errors.New("boom") })

    fmt.Println(g.Wait() != nil)
}
```

### SingleFlightGroup

`SingleFlightGroup[T]` deduplicates concurrent work by key.

- Zero value is ready for use; `NewSingleFlightGroup[T]()` is optional and returns a ready-to-use pointer.
- `Do(key, fn)` returns `(value, err, shared)`.
- `shared == true` means this call received another call's result.
- On `fn` error, `Do` returns zero `T` plus the error.
- If `T` is an interface type and `fn` returns a nil interface value, `Do` exposes it as zero `T`.

```go
package main

import (
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    var g sync.SingleFlightGroup[int]

    v, err, shared := g.Do("key", func() (int, error) {
        return 42, nil
    })
    fmt.Println(v, err == nil, shared)

    g.Forget("key")
}
```

## Pool

### Generic Pool

`Pool[T]` is a typed wrapper around [`sync.Pool`](https://pkg.go.dev/sync#Pool).

- Stores `*T` values.
- Zero value is not ready; use `NewPool[T]()` which returns a ready-to-use pointer.
- Follows normal `sync.Pool` semantics (runtime may drop entries anytime).
- Does not reset values automatically on `Put`; callers are responsible for reuse hygiene.
- `Put(nil)` is a no-op.

```go
package main

import (
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    type item struct {
        ID int
    }

    pool := sync.NewPool[item]()
    it := pool.Get()
    it.ID = 10
    pool.Put(it)

    fmt.Println("ok")
}
```

### BufferPool

`BufferPool` is a convenience wrapper over `Pool[bytes.Buffer]`.

- Zero value is not ready; use `NewBufferPool()` which returns a ready-to-use pointer.
- `Get` returns `*bytes.Buffer`.
- `Get` returns an empty buffer.
- `Put` resets the buffer (nil-safe no-op).
- `Copy` returns a cloned `[]byte` (non-aliasing, nil-safe).

```go
package main

import (
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    bp := sync.NewBufferPool()
    buf := bp.Get()
    defer bp.Put(buf)

    buf.WriteString("hello")
    out := bp.Copy(buf)
    fmt.Println(string(out))
}
```

## Value

`Value[T]` is a typed wrapper around [`atomic.Value`](https://pkg.go.dev/sync/atomic#Value).

- Zero value is ready (`NewValue` is optional and returns a ready-to-use pointer).
- `Load` and `Swap` return zero `T` if unset.
- Same underlying constraints as `atomic.Value` apply.
- If `T` is an interface type, storing a nil interface value panics just like `atomic.Value.Store(nil)`.
- When `T` is an interface or `any`, stores must still be consistent with `atomic.Value`'s concrete-type rules.
- `CompareAndSwap` follows `atomic.Value.CompareAndSwap`; when `T` is an interface, non-comparable dynamic values in `old` can panic.

```go
package main

import (
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    var v sync.Value[int]
    fmt.Println(v.Load())

    v.Store(1)
    fmt.Println(v.Swap(2))
    fmt.Println(v.CompareAndSwap(2, 3))
}
```

## Map

`Map[K, V]` is a typed wrapper around [`sync.Map`](https://pkg.go.dev/sync#Map).

- Zero value is ready (`NewMap` is optional and returns a ready-to-use pointer).
- `Load`, `LoadOrStore`, `LoadAndDelete`, and `Swap` return zero `V` when needed; use boolean flags to distinguish missing keys.
- If `K` is an interface type and a nil interface key is stored, `Range` exposes it as zero `K` (for example, `nil` for interface `K`).
- If `V` is an interface type and a nil interface value is stored, value-returning methods expose it as zero `V` (for example, `nil` for interface `V`).
- `Range` follows `sync.Map.Range` semantics and does not provide a consistent snapshot during concurrent mutation.
- `Clear` removes all entries.
- `CompareAndSwap` / `CompareAndDelete` follow `sync.Map` comparability rules; non-comparable dynamic `old` values can panic.

```go
package main

import (
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    m := sync.NewMap[string, int]()

    m.Store("one", 1)
    v, ok := m.Load("one")
    fmt.Println(v, ok)

    prev, loaded := m.LoadOrStore("one", 99)
    fmt.Println(prev, loaded)

    m.Range(func(k string, v int) bool {
        fmt.Println(k, v)
        return true
    })
}
```

Nil interface edge-case example:

```go
package main

import (
    "fmt"
    "io"

    sync "github.com/alexfalkowski/go-sync"
)

func main() {
    var m sync.Map[string, io.Reader]
    var r io.Reader

    m.Store("reader", r)
    m.Range(func(_ string, value io.Reader) bool {
        fmt.Println(value == nil)
        return true
    })
}
```

## Background / References

This library draws inspiration from:

- <https://go.dev/blog/pipelines>
- <https://go.dev/talks/2012/concurrency.slide>
- <https://gobyexample.com/timeouts>
- <https://go.dev/wiki/Timeouts>
- <https://github.com/lotusirous/go-concurrency-patterns>

For executable, CI-verified usage examples, see [`example_test.go`](example_test.go). Those examples back the rendered package documentation on pkg.go.dev.
