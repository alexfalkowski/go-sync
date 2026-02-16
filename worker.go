package sync

import (
	"context"
	"sync"
	"time"
)

// NewWorker returns a [Worker] that bounds concurrent execution to count.
//
// The worker uses a buffered channel of size count as a semaphore. A call to
// [Worker.Schedule] acquires one slot before starting work and releases it when
// the work completes.
//
// If count is 0, scheduling will always block until the provided context times
// out or is canceled (because the semaphore has no capacity).
func NewWorker(count uint) *Worker {
	return &Worker{
		requests: make(chan struct{}, count),
		wg:       sync.WaitGroup{},
	}
}

// Worker schedules handlers with a bounded level of concurrency.
//
// Work is scheduled via [Worker.Schedule] and completion is observed via
// [Worker.Wait]. Scheduled handlers run asynchronously in their own goroutines.
type Worker struct {
	requests chan struct{}
	wg       sync.WaitGroup
}

// Schedule attempts to schedule hook.OnRun to run asynchronously, subject to the worker's concurrency limit.
//
// Schedule blocks until one of the following occurs:
//
//  1. A concurrency slot is acquired before the deadline: Schedule starts OnRun in a goroutine and returns nil.
//  2. The derived timeout context is done first: Schedule returns ctx.Err().
//
// The context passed to OnRun is a derived context created by [context.WithTimeout] using the provided timeout.
// This context is also passed to hook.OnError (via hook.Error) if OnRun returns a non-nil error.
//
// Error handling semantics:
//
//   - If hook.OnRun is nil, Schedule returns [ErrNoOnRunProvided].
//   - Errors returned from OnRun are routed to hook.OnError (if set) and are not returned from Schedule.
//     Schedule only reports errors related to scheduling (timeout/cancellation before a slot is acquired).
//
// To wait for all scheduled handlers to complete, call [Worker.Wait].
func (w *Worker) Schedule(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case w.requests <- struct{}{}:
		w.wg.Go(func() {
			_ = hook.Error(ctx, hook.OnRun(ctx))
			<-w.requests
		})
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// Wait blocks until all handlers that have been successfully scheduled have completed.
//
// It does not cancel running handlers. Cancellation is controlled by the contexts
// provided to [Worker.Schedule] and observed by the handlers themselves.
func (w *Worker) Wait() {
	w.wg.Wait()
}
