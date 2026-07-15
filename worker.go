package sync

import (
	"context"
	"errors"
	"sync"
)

// ErrWorkerFull is returned by [Worker.TrySchedule] when no concurrency slot is available immediately.
var ErrWorkerFull = errors.New("worker has no available slot")

// NewWorker returns a pointer to a [Worker] that bounds concurrent execution to count.
//
// The worker uses a buffered channel of size count as a semaphore. A call to
// [Worker.Schedule] or [Worker.TrySchedule] acquires one slot before starting
// work and releases it when the work completes.
//
// If count is 0, [Worker.Schedule] always blocks until the provided context
// times out or is canceled, and [Worker.TrySchedule] returns [ErrWorkerFull]
// immediately.
//
// The zero value of [Worker] is not ready for use; construct one with NewWorker.
func NewWorker(count uint) *Worker {
	return &Worker{
		requests: make(chan struct{}, count),
	}
}

// Worker schedules handlers with a bounded level of concurrency.
//
// Work is scheduled via [Worker.Schedule] or [Worker.TrySchedule], and
// completion is observed via [Worker.Wait]. Scheduled handlers run
// asynchronously in their own goroutines.
//
// The zero value is not ready for use.
// A Worker must not be copied after first use; pass and store *Worker values.
type Worker struct {
	requests chan struct{}
	wg       sync.WaitGroup
}

// Schedule attempts to schedule hook.OnRun to run asynchronously, subject to the worker's concurrency limit.
//
// Schedule blocks until one of the following occurs:
//
//  1. A concurrency slot is acquired: Schedule starts OnRun in a goroutine and returns nil.
//  2. ctx is done first: Schedule returns [context.Cause](ctx).
//
// The context passed to OnRun is the ctx provided to Schedule. This context is
// also passed to hook.OnError (via hook.Error) if OnRun returns a non-nil error.
// Schedule does not derive or bound any deadline itself; to bound the wait for
// a slot, pass a ctx with a deadline. To give the handler its own run budget
// starting when it actually begins, wrap ctx with [context.WithTimeout] (or
// similar) inside OnRun.
//
// Error handling semantics:
//
//   - If hook.OnRun is nil, Schedule returns [ErrNoOnRunProvided].
//     This validation happens before the context shortcut check.
//   - If the input context is already done on entry, Schedule returns its
//     cancellation cause without scheduling OnRun.
//   - Errors returned from OnRun are routed to hook.OnError (if set) and are not returned from Schedule.
//     Schedule only reports errors related to scheduling (cancellation before a slot is acquired).
//   - Once a handler has been scheduled successfully, Schedule returns nil even
//     if ctx later expires while the handler is still running.
//   - Panics from OnRun or OnError are not recovered; see [Hook].
//
// To wait for all scheduled handlers to complete, call [Worker.Wait].
func (w *Worker) Schedule(ctx context.Context, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	select {
	case w.requests <- struct{}{}:
		if ctx.Err() != nil {
			<-w.requests
			return context.Cause(ctx)
		}

		w.wg.Go(func() {
			defer func() {
				<-w.requests
			}()

			_ = hook.Error(ctx, hook.OnRun(ctx))
		})
	case <-ctx.Done():
		return context.Cause(ctx)
	}

	return nil
}

// TrySchedule attempts to schedule hook.OnRun immediately.
//
// If a concurrency slot is available, TrySchedule starts OnRun in a goroutine
// and returns nil. The context passed to OnRun is the ctx provided to
// TrySchedule. This context is also passed to hook.OnError (via hook.Error) if
// OnRun returns a non-nil error.
//
// TrySchedule does not wait for capacity. If no concurrency slot is available
// immediately, it returns [ErrWorkerFull] without scheduling OnRun.
//
// Error handling semantics:
//
//   - If hook.OnRun is nil, TrySchedule returns [ErrNoOnRunProvided].
//     This validation happens before context or capacity shortcut checks.
//   - If the input context is already done on entry, TrySchedule returns its
//     cancellation cause without scheduling OnRun.
//   - Errors returned from OnRun are routed to hook.OnError (if set) and are
//     not returned from TrySchedule. TrySchedule only reports scheduling errors.
//   - Once a handler has been scheduled successfully, TrySchedule returns nil
//     even if the context is later canceled while the handler is still running.
//   - Panics from OnRun or OnError are not recovered; see [Hook].
//
// To wait for all scheduled handlers to complete, call [Worker.Wait].
func (w *Worker) TrySchedule(ctx context.Context, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	select {
	case w.requests <- struct{}{}:
		if ctx.Err() != nil {
			<-w.requests
			return context.Cause(ctx)
		}

		w.wg.Go(func() {
			defer func() {
				<-w.requests
			}()

			_ = hook.Error(ctx, hook.OnRun(ctx))
		})

		return nil
	case <-ctx.Done():
		return context.Cause(ctx)
	default:
		return ErrWorkerFull
	}
}

// Wait waits for all handlers that have been successfully scheduled to
// complete, or for ctx to be done first.
//
// Wait returns nil once every scheduled handler has finished, or
// [context.Cause](ctx) if ctx is done first. Completion is observed by a
// goroutine started for this call rather than a persistent signal, so this
// is a best-effort race: it narrows, but does not eliminate, the window
// where an already-done ctx wins over handlers that finished shortly before
// Wait was called. Wait never cancels a running handler; cancellation is
// controlled by the contexts provided to [Worker.Schedule] or
// [Worker.TrySchedule] and observed by the handlers themselves. A handler
// that ignores its context can keep running after Wait returns; the internal
// goroutine started by Wait also lives until that handler finishes, matching
// the lifetime of the already-outstanding work. Wait can be called multiple
// times; each call waits for the currently scheduled work to finish.
func (w *Worker) Wait(ctx context.Context) error {
	done := make(chan struct{})

	go func() {
		w.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		select {
		case <-done:
			return nil
		default:
			return context.Cause(ctx)
		}
	}
}
