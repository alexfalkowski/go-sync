![Gopher](assets/gopher.png)
[![CircleCI](https://circleci.com/gh/alexfalkowski/go-sync.svg?style=shield)](https://circleci.com/gh/alexfalkowski/go-sync)
[![codecov](https://codecov.io/gh/alexfalkowski/go-sync/graph/badge.svg?token=Q7B3VZYL9K)](https://codecov.io/gh/alexfalkowski/go-sync)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexfalkowski/go-sync)](https://goreportcard.com/report/github.com/alexfalkowski/go-sync)
[![Go Reference](https://pkg.go.dev/badge/github.com/alexfalkowski/go-sync.svg)](https://pkg.go.dev/github.com/alexfalkowski/go-sync)
[![Stability: Active](https://masterminds.github.io/stability/active.svg)](https://masterminds.github.io/stability/active.html)

# go-sync

A library to handle concurrency.

## Wait

Wait will wait for the handler to complete or continue. As an example:

```go
import (
	"context"
	"time"

	"github.com/alexfalkowski/go-sync"
)

// Do something with err.
err := sync.Wait(context.Background(), time.Second, func(context.Context) error {
    // Do something important.
	  return nil
})
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
```
