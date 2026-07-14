package sync_test

import (
	"context"
	"testing"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

// BenchmarkBufferPool measures the package-owned Get/Copy/Put lifecycle because
// Copy promises detached bytes while BufferPool keeps buffer reuse cheap.
func BenchmarkBufferPool(b *testing.B) {
	b.ReportAllocs()

	bs := make([]byte, 1024)
	pool := sync.NewBufferPool()

	for b.Loop() {
		buffer := pool.Get()
		if _, err := buffer.Write(bs); err != nil {
			require.NoError(b, err)
		}
		_ = pool.Copy(buffer)
		pool.Put(buffer)
	}
}

// BenchmarkWorker measures public bounded scheduling overhead because Worker
// owns the goroutine, semaphore, hook, and wait lifecycle around each task.
func BenchmarkWorker(b *testing.B) {
	b.ReportAllocs()

	worker := sync.NewWorker(16)
	for b.Loop() {
		if err := worker.Schedule(b.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				return nil
			},
		}); err != nil {
			require.NoError(b, err)
		}
	}
	if err := worker.Wait(b.Context()); err != nil {
		require.NoError(b, err)
	}
}
