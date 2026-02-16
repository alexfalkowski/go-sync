![Gopher](assets/gopher.png)
[![CircleCI](https://circleci.com/gh/alexfalkowski/go-sync.svg?style=shield)](https://circleci.com/gh/alexfalkowski/go-sync)
[![codecov](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexfalkowski/go-sync)](https://goreportcard.com/report/github.com/alexfalkowski/go-sync)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/go-sync.svg)](https://pkg.go.dev/github.com/alexfalkowski/go-sync)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# go-sync

A small Go library (package `sync`) that provides concurrency helpers.

## Background

These are some examples that this library was inspired by:

- <https://go.dev/blog/pipelines>
- <https://go.dev/talks/2012/concurrency.slide>
- <https://gobyexample.com/timeouts>
- <https://go.dev/wiki/Timeouts>
- <https://github.com/lotusirous/go-concurrency-patterns>

## Wait

Wait runs `Hook.OnRun` asynchronously and waits up to the provided timeout for it to complete.

If `OnRun` completes first, `Wait` returns the result of `Hook.OnError` (if provided) applied to the error returned by `OnRun`.

If the timeout expires (or the context is canceled) first, `Wait` returns `nil` immediately **without waiting for `OnRun` to finish**. This means `OnRun` may continue running in the background, and any error it eventually produces is discarded.

Use `Wait` when you want a best-effort “wait up to N” behavior and you do not need cancellation to be propagated into the work. If you need cancellation and a timeout error, use `Timeout`.

As an example:

```go
import (
    "context"
    "time"

    sync "github.com/alexfalkowski/go-sync"
)

err := sync.Wait(context.Background(), time.Second, sync.Hook{
    OnRun: func(context.Context) error {
        // Do something important.
        return nil
    },
    OnError: func(_ context.Context, err error) error {
        // Do something with err.
        return err
    },
})
if err != nil {
    // Do something with err.
}
```

## Timeout

Timeout runs `Hook.OnRun` with a derived context created by `context.WithTimeout`.

If `OnRun` completes first, `Timeout` returns the result of `Hook.OnError` (if provided) applied to the error returned by `OnRun`.

If the timeout expires (or the context is canceled) first, `Timeout` returns `ctx.Err()` (typically `context.DeadlineExceeded` or `context.Canceled`). Unlike `Wait`, cancellation is propagated to `OnRun` via the derived context, so well-behaved handlers should observe `ctx.Done()` and exit promptly.

As an example:

```go
import (
    "context"
    "time"

    sync "github.com/alexfalkowski/go-sync"
)

err := sync.Timeout(context.Background(), time.Second, sync.Hook{
    OnRun: func(context.Context) error {
        // Do something important.
        return nil
    },
    OnError: func(_ context.Context, err error) error {
        // Do something with err.
        return err
    },
})
if err != nil {
    if sync.IsTimeoutError(err) {
        // Do something with timeout.
    }

    // Do something with error.
}
```

## Pool

We have a generic pool based on [sync.Pool](https://pkg.go.dev/sync#Pool) and a [bytes.Buffer](https://pkg.go.dev/bytes#Buffer) pool.

```go
import sync "github.com/alexfalkowski/go-sync"

pool := sync.NewBufferPool()

buffer := pool.Get() // Do something with buffer.
defer pool.Put(buffer)

// Do something with buffer, then copy it to a []byte.
bs := pool.Copy(buffer)
_ = bs
```

## Atomic

We have a generic value based on [atomic.Value](https://pkg.go.dev/sync/atomic#Value).

```go
import sync "github.com/alexfalkowski/go-sync"

value := sync.NewValue[int]()

value.Store(1)
v := value.Load() // Do something with v.
```

## Worker

We have a worker based on [sync.WaitGroup](https://pkg.go.dev/sync#WaitGroup) and [buffered channels](https://go.dev/tour/concurrency/3) to bound concurrent execution.

Schedule returns ctx.Err() if it cannot schedule the handler before the timeout expires or the context is canceled, and returns ErrNoOnRunProvided if OnRun is nil. Handler errors are passed to Hook.OnError (if set) and are not returned.

```go
import (
    "context"
    "time"

    sync "github.com/alexfalkowski/go-sync"
)

worker := sync.NewWorker(10)

err := worker.Schedule(context.Background(), time.Second, sync.Hook{
    OnRun: func(context.Context) error {
        // Do something important.
        return nil
    },
    OnError: func(_ context.Context, err error) error {
        // Do something with err.
        return nil
    },
})
if err != nil {
    if sync.IsTimeoutError(err) {
        // Do something with timeout.
    }

    // Do something with scheduling error.
}

worker.Wait()
```

## Group

### ErrorGroup

`ErrorGroup` is a convenience alias for [`errgroup.Group`](https://pkg.go.dev/golang.org/x/sync/errgroup#Group). It behaves exactly like `errgroup.Group`.

```go
import (
    "context"
    "fmt"

    sync "github.com/alexfalkowski/go-sync"
)

var g sync.ErrorGroup

g.Go(func() error {
    // Do something important.
    return nil
})

g.Go(func() error {
    return fmt.Errorf("boom")
})

// Wait returns the first non-nil error (if any), once all goroutines complete.
if err := g.Wait(); err != nil {
    _ = context.Cause(context.Background()) // no-op example usage; remove in real code
    // Do something with err.
}
```

### SingleFlightGroup

We have a generic group based on [singleflight.Group](https://pkg.go.dev/golang.org/x/sync/singleflight#Group) to suppress duplicate function calls.

For a given key, the first caller executes the provided function, and concurrent callers for the same key wait for the first call to complete and receive the same (value, error) result.

`Do` returns the value, error, and whether the result was shared with other callers (`shared == true` means this call did not execute the function itself).

If the function returns a non-nil error, `Do` returns the zero value of `T` along with that error.

```go
import sync "github.com/alexfalkowski/go-sync"

g := sync.NewSingleFlightGroup[int]()

v, err, shared := g.Do("key", func() (int, error) {
    // Do something important.
    return 1, nil
})
_, _, _ = v, err, shared

// Forget clears any in-flight or completed result for the key,
// so a future Do("key", ...) will execute the function again.
g.Forget("key")
```

## Map

We have a generic map based on [sync.Map](https://pkg.go.dev/sync#Map).

The zero value of Map is ready for use (NewMap is optional). Note: storing a nil interface value can cause methods that type-assert internally (for example, Range) to panic.

```go
import sync "github.com/alexfalkowski/go-sync"

m := sync.NewMap[string, int]()

m.Store("one", 1)

v, ok := m.Load("one")
if ok {
    // Do something with v.
}

m.Range(func(k string, v int) bool {
    // Do something with k and v.
    return true
})
```
