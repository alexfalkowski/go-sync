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

## Errors

As most operations are async, errors need to be handled differently. As an example:

```go
import (
    "context"

    "github.com/alexfalkowski/go-sync"
)

sync.SetErrorHandler(func(ctx context.Context, err error) error {
    // You can check the original context.
    // You can check the original error.
    // You can return the error or ignore it.
    return err
})
```

## Wait

Wait will wait for the handler to complete or continue. As an example:

```go
import (
    "context"
    "time"

    "github.com/alexfalkowski/go-sync"
)


err := sync.Wait(context.Background(), time.Second, func(context.Context) error {
    // Do something important.
    return nil
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

// Do something with err.
err := sync.Timeout(context.Background(), time.Second, func(context.Context) error {
    // Do something important.
    return nil
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
import "github.com/alexfalkowski/go-sync/bytes"

pool := bytes.NewBufferPool()

buffer := pool.Get() // Do something with buffer.
defer pool.Put(buffer)
```

## Atomic

We have a generic value based on [atomic.Value](https://pkg.go.dev/sync/atomic#Value).

```go
import "github.com/alexfalkowski/go-sync/atomic"

var value atomic.Value[int]

value.Store(1)
v := value.Load() // Do something with v.
```

## Worker

We have a worker based on [sync.WaitGroup](https://pkg.go.dev/sync#WaitGroup) and [buffered channels](https://go.dev/tour/concurrency/3).

```go
import (
    "context"

    "github.com/alexfalkowski/go-sync"
)

worker := sync.NewWorker(10)

worker.Schedule(context.Background(), func(context.Context) error {
    // Do something important.
    return nil
})
```
