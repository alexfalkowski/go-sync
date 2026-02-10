package sync

import (
	"context"
	"sync"
	"time"
)

// NewWorker returns a Worker that limits the number of concurrent scheduled handlers to count.
func NewWorker(count uint) *Worker {
	return &Worker{
		requests: make(chan struct{}, count),
		wg:       sync.WaitGroup{},
	}
}

// Worker schedules handlers with a bounded level of concurrency.
type Worker struct {
	requests chan struct{}
	wg       sync.WaitGroup
}

// Schedule schedules hook.OnRun to run asynchronously if it can be scheduled within the given timeout.
//
// It returns ctx.Err() if the context is done before the handler can be scheduled.
// Errors returned by OnRun are handled via hook.OnError (if set) and are not returned.
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

// Wait blocks until all scheduled handlers have completed.
func (w *Worker) Wait() {
	w.wg.Wait()
}
