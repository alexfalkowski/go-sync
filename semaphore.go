package sync

import "golang.org/x/sync/semaphore"

// NewSemaphore wraps [semaphore.NewWeighted] to construct a [Semaphore] with
// the given maximum combined weight. It panics if size is negative.
func NewSemaphore(size int64) *Semaphore {
	return semaphore.NewWeighted(size)
}

// Semaphore is an alias for [semaphore.Weighted].
//
// It is provided for convenience so users of this package can use a weighted
// semaphore without importing golang.org/x/sync/semaphore directly.
// Use [NewSemaphore] to initialize its capacity.
type Semaphore = semaphore.Weighted
