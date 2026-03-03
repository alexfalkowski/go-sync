![Gopher](assets/gopher.png)
[![CircleCI](https://circleci.com/gh/alexfalkowski/go-sync.svg?style=shield)](https://circleci.com/gh/alexfalkowski/go-sync)
[![codecov](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexfalkowski/go-sync)](https://goreportcard.com/report/github.com/alexfalkowski/go-sync)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/go-sync.svg)](https://pkg.go.dev/github.com/alexfalkowski/go-sync)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# go-sync

A small Go library (package `sync`) with focused concurrency helpers:

- Hook-driven execution (`Wait`, `Timeout`, `Worker`)
- Typed wrappers for `sync.Pool`, `sync.Map`, and `atomic.Value`
- Group helpers (`ErrorGroup`, `SingleFlightGroup`)

## Install

```bash
go get github.com/alexfalkowski/go-sync
```

## Import

```go
import sync "github.com/alexfalkowski/go-sync"
```

## Hooks

Most execution helpers accept a `sync.Hook`:

- `OnRun(context.Context) error` is required.
- `OnError(context.Context, error) error` is optional.
- If `OnRun` is nil, helpers return `sync.ErrNoOnRunProvided`.

`OnError` is only called when `OnRun` returns a non-nil error. If `OnError` returns a different error, that new error is returned.

```go
hook := sync.Hook{
	OnRun: func(context.Context) error {
		return errors.New("boom")
	},
	OnError: func(_ context.Context, err error) error {
		return fmt.Errorf("wrapped: %w", err)
	},
}
```

## Wait vs Timeout

`Wait` and `Timeout` both run `Hook.OnRun`, but they differ:

- `Wait`: best-effort wait up to `timeout`; returns `nil` on timeout/cancel and does not cancel `OnRun`.
- `Timeout`: derives a timeout context for `OnRun`; returns `ctx.Err()` (`context.DeadlineExceeded` or `context.Canceled`) when the context ends first.

Use `sync.IsTimeoutError(err)` to check if an error is `context.DeadlineExceeded`.

### Wait example (best effort)

```go
err := sync.Wait(context.Background(), 10*time.Millisecond, sync.Hook{
	OnRun: func(context.Context) error {
		time.Sleep(time.Second)
		return errors.New("finished too late")
	},
})

// err is nil because Wait timed out first.
_ = err
```

### Timeout example (propagated cancellation)

```go
err := sync.Timeout(context.Background(), 10*time.Millisecond, sync.Hook{
	OnRun: func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	},
})

if sync.IsTimeoutError(err) {
	// Timed out via context.DeadlineExceeded.
}
```

## Worker

`Worker` schedules asynchronous handlers with bounded concurrency.

- `NewWorker(count)` creates a worker with at most `count` in-flight handlers.
- `Schedule` blocks until a slot is acquired or timeout/cancel happens.
- `Schedule` returns only scheduling errors (`ctx.Err()` or `ErrNoOnRunProvided`).
- Handler errors are routed to `Hook.OnError` and are not returned by `Schedule`.
- `Wait` blocks until all successfully scheduled handlers complete.

If `count == 0`, scheduling always blocks until timeout/cancel.

```go
worker := sync.NewWorker(4)
defer worker.Wait()

for i := range 10 {
	err := worker.Schedule(context.Background(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			fmt.Println("job", i)
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
```

## Group

### ErrorGroup

`sync.ErrorGroup` is a type alias for [`errgroup.Group`](https://pkg.go.dev/golang.org/x/sync/errgroup#Group).

```go
var g sync.ErrorGroup

g.Go(func() error { return nil })
g.Go(func() error { return fmt.Errorf("boom") })

if err := g.Wait(); err != nil {
	// first non-nil error
}
```

### SingleFlightGroup

`SingleFlightGroup[T]` deduplicates concurrent work by key.

- Zero value is ready for use.
- `Do(key, fn)` returns `(value, err, shared)`.
- `shared == true` means this call received another call's result.
- On `fn` error, `Do` returns zero `T` plus the error.

```go
var g sync.SingleFlightGroup[int] // zero value is usable

v, err, shared := g.Do("key", func() (int, error) {
	return 42, nil
})
_, _, _ = v, err, shared

g.Forget("key") // next Do("key", ...) executes fn again
```

## Pool

### Generic Pool

`Pool[T]` is a typed wrapper around [`sync.Pool`](https://pkg.go.dev/sync#Pool).

- Stores `*T` values.
- Zero value is not ready; use `NewPool[T]()`.
- Follows normal `sync.Pool` semantics (runtime may drop entries anytime).

```go
type item struct {
	ID int
}

pool := sync.NewPool[item]()
it := pool.Get()
it.ID = 10
pool.Put(it)
```

### BufferPool

`BufferPool` is a convenience wrapper over `Pool[bytes.Buffer]`.

- `Get` returns `*bytes.Buffer`.
- `Put` resets the buffer (nil-safe no-op).
- `Copy` returns a cloned `[]byte` (non-aliasing, nil-safe).

```go
bp := sync.NewBufferPool()
buf := bp.Get()
defer bp.Put(buf)

buf.WriteString("hello")
out := bp.Copy(buf) // []byte("hello"), safe copy
_ = out
```

## Value

`Value[T]` is a typed wrapper around [`atomic.Value`](https://pkg.go.dev/sync/atomic#Value).

- Zero value is ready.
- `Load` and `Swap` return zero `T` if unset.
- Same underlying constraints as `atomic.Value` apply.

```go
var v sync.Value[int] // zero value is usable

current := v.Load() // 0
_ = current

v.Store(1)
prev := v.Swap(2) // 1
ok := v.CompareAndSwap(2, 3)
_, _ = prev, ok
```

## Map

`Map[K, V]` is a typed wrapper around [`sync.Map`](https://pkg.go.dev/sync#Map).

- Zero value is ready (`NewMap` is optional).
- `Load`, `LoadOrStore`, `LoadAndDelete`, and `Swap` return zero `V` when needed; use boolean flags to distinguish missing keys.
- If `V` is an interface type, storing a nil interface value can still panic in methods that type-assert stored values like `Range`.

```go
m := sync.NewMap[string, int]()

m.Store("one", 1)
v, ok := m.Load("one")
_, _ = v, ok

prev, loaded := m.LoadOrStore("one", 99) // prev=1, loaded=true
_, _ = prev, loaded

m.Range(func(k string, v int) bool {
	fmt.Println(k, v)
	return true
})
```

Nil interface edge-case example:

```go
m := sync.NewMap[string, io.Reader]()

var r io.Reader // nil interface value
v, loaded := m.LoadOrStore("reader", r)
_, _ = v, loaded // safe for LoadOrStore

// Avoid calling Range if you may have stored nil interface values;
// Range type-asserts values and may panic on untyped nil.
```

## Background / References

This library draws inspiration from:

- <https://go.dev/blog/pipelines>
- <https://go.dev/talks/2012/concurrency.slide>
- <https://gobyexample.com/timeouts>
- <https://go.dev/wiki/Timeouts>
- <https://github.com/lotusirous/go-concurrency-patterns>
