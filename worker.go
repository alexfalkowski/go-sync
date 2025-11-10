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
func (w *Worker) Schedule(ctx context.Context, lc Lifecycle) {
	w.requests <- struct{}{}
	w.wg.Go(func() {
		_ = lc.Error(ctx, lc.OnRun(ctx))
		<-w.requests
	})
}

// Wait will wait for the worker to complete.
func (w *Worker) Wait() {
	w.wg.Wait()
}
