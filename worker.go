package sync

import (
	"context"
	"sync"
	"time"
)

// NewWorker returns a pointer to a [Worker] that bounds concurrent execution to count.
//
// The worker uses a buffered channel of size count as a semaphore. A call to
// [Worker.Schedule] acquires one slot before starting work and releases it when
// the work completes.
//
// If count is 0, scheduling will always block until the provided context times
// out or is canceled (because the semaphore has no capacity).
//
// The zero value of [Worker] is not ready for use; construct one with NewWorker.
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
//
// The zero value is not ready for use.
type Worker struct {
	requests chan struct{}
	wg       sync.WaitGroup
}

// Schedule attempts to schedule hook.OnRun to run asynchronously, subject to the worker's concurrency limit.
//
// Schedule blocks until one of the following occurs:
//
//  1. A concurrency slot is acquired before the deadline: Schedule starts OnRun in a goroutine and returns nil.
//  2. The derived timeout context is done first: Schedule returns
//     [context.Cause] from that derived context.
//
// The context passed to OnRun is a derived context created by
// [context.WithTimeoutCause] using the provided timeout.
// The timeout budget starts when Schedule is called, so time spent waiting for a
// concurrency slot and time spent running OnRun share the same deadline.
// This context is also passed to hook.OnError (via hook.Error) if OnRun returns a non-nil error.
// If OnRun ignores that context and continues running after the deadline, the
// goroutine may outlive the caller of Schedule until the handler eventually returns.
//
// Error handling semantics:
//
//   - If hook.OnRun is nil, Schedule returns [ErrNoOnRunProvided].
//   - If the input context is already done on entry, Schedule returns its
//     cancellation cause without scheduling OnRun.
//   - Errors returned from OnRun are routed to hook.OnError (if set) and are not returned from Schedule.
//     Schedule only reports errors related to scheduling (timeout/cancellation before a slot is acquired).
//   - Once a handler has been scheduled successfully, Schedule returns nil even
//     if the derived context later expires while the handler is still running.
//
// To wait for all scheduled handlers to complete, call [Worker.Wait].
func (w *Worker) Schedule(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, timeout, ErrTimeout)
	if ctx.Err() != nil {
		cancel()
		return context.Cause(ctx)
	}

	select {
	case w.requests <- struct{}{}:
		if ctx.Err() != nil {
			<-w.requests
			cancel()
			return context.Cause(ctx)
		}

		w.wg.Go(func() {
			defer cancel()
			defer func() {
				<-w.requests
			}()

			_ = hook.Error(ctx, hook.OnRun(ctx))
		})
	case <-ctx.Done():
		cancel()
		return context.Cause(ctx)
	}

	return nil
}

// Wait blocks until all handlers that have been successfully scheduled have completed.
//
// It does not cancel running handlers. Cancellation is controlled by the contexts
// provided to [Worker.Schedule] and observed by the handlers themselves.
// Wait can be called multiple times; each call waits for the currently scheduled
// work to finish.
func (w *Worker) Wait() {
	w.wg.Wait()
}
