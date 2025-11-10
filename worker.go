package sync

import (
	"context"
	"sync"
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

// Schedule a handler.
func (w *Worker) Schedule(ctx context.Context, handler Handler) {
	w.ScheduleWithError(ctx, handler, DefaultErrorHandler)
}

// ScheduleWithError schedules a handler with error handling.
func (w *Worker) ScheduleWithError(ctx context.Context, handler Handler, errorHandler ErrorHandler) {
	w.requests <- struct{}{}
	w.wg.Go(func() {
		_ = errorHandler(ctx, handler(ctx))
		<-w.requests
	})
}

// Wait will wait for the worker to complete.
func (w *Worker) Wait() {
	w.wg.Wait()
}
