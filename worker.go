package sync

import (
	"context"
	"sync"
	"time"
)

// NewWorker for sync.
func NewWorker(count int) *Worker {
	return &Worker{
		requests: make(chan struct{}, count),
		wg:       sync.WaitGroup{},
	}
}

// Worker for sync.
type Worker struct {
	requests chan struct{}
	wg       sync.WaitGroup
}

// Schedule a handler if it can be scheduled within the timeout.
func (w *Worker) Schedule(ctx context.Context, timeout time.Duration, hook Hook) error {
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

// Wait will wait for the worker to complete.
func (w *Worker) Wait() {
	w.wg.Wait()
}
