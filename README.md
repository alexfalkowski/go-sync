![Gopher](assets/gopher.png)
[![CircleCI](https://circleci.com/gh/alexfalkowski/go-sync.svg?style=shield)](https://circleci.com/gh/alexfalkowski/go-sync)
[![codecov](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)](https://codecov.io/gh/alexfalkowski/go-sync)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexfalkowski/go-sync)](https://goreportcard.com/report/github.com/alexfalkowski/go-sync)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/go-sync.svg)](https://pkg.go.dev/github.com/alexfalkowski/go-sync)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# go-sync

A library to handle concurrency.

## Background

These are some examples that this library was inspired by:

- <https://go.dev/blog/pipelines>
- <https://go.dev/talks/2012/concurrency.slide>
- <https://gobyexample.com/timeouts>
- <https://go.dev/wiki/Timeouts>
- <https://github.com/lotusirous/go-concurrency-patterns>

## Wait

Wait will wait for the handler to complete or continue. As an example:

```go
import (
    "context"
    "time"

    "github.com/alexfalkowski/go-sync"
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

Timeout will wait for the handler to complete or timeout. As an example:

```go
import (
    "context"
    "time"

    "github.com/alexfalkowski/go-sync"
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
import "github.com/alexfalkowski/go-sync"

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
import "github.com/alexfalkowski/go-sync"

value := sync.NewValue[int]()

value.Store(1)
v := value.Load() // Do something with v.
```

## Worker

We have a worker based on [sync.WaitGroup](https://pkg.go.dev/sync#WaitGroup) and [buffered channels](https://go.dev/tour/concurrency/3).

```go
import (
    "context"
    "time"

    "github.com/alexfalkowski/go-sync"
)

worker := sync.NewWorker(10)

err := worker.Schedule(context.Background(), time.Second, sync.Hook{
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

## Map

We have a generic map based on [sync.Map](https://pkg.go.dev/sync#Map).

```go
import "github.com/alexfalkowski/go-sync"

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
