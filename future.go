package sync

import "context"

// Future represents the eventual result of an asynchronous operation.
//
// A Future is safe for concurrent use. Its result is cached, so Await can be
// called repeatedly by one or more callers after the operation completes.
// The zero value is not ready for use; construct a Future with Async.
// Do not copy a Future after first use; pass and store *Future values.
type Future[T any] struct {
	done  chan struct{}
	value T
	err   error
}

// Async starts fn in a new goroutine and returns a Future for its result.
//
// ctx is passed to fn and controls the operation. Async invokes fn even if
// ctx is already done, so fn is responsible for observing cancellation.
// Async does not cancel the operation when a caller's Await context is done.
// If fn returns an error, Async caches that error and returns it from every
// subsequent Await call. fn must be non-nil and must not panic; Async does not
// recover panics from fn.
func Async[T any](ctx context.Context, fn func(context.Context) (T, error)) *Future[T] {
	future := &Future[T]{done: make(chan struct{})}

	go func() {
		future.value, future.err = fn(ctx)
		close(future.done)
	}()

	return future
}

// Await waits for the Future to complete or for ctx to be done.
//
// When ctx.Done is selected, Await checks completion once more before
// returning. A result published by that check wins; otherwise Await returns
// ctx's cancellation cause without canceling the operation. A later Await can
// still retrieve the operation's eventual result. Once the operation
// completes, its cached value and error are returned to every caller.
func (f *Future[T]) Await(ctx context.Context) (T, error) {
	select {
	case <-f.done:
		return f.value, f.err
	case <-ctx.Done():
		select {
		case <-f.done:
			return f.value, f.err
		default:
			var zero T
			return zero, context.Cause(ctx)
		}
	}
}
