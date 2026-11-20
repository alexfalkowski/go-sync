package sync_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestSemaphoreWeightedCapacity(t *testing.T) {
	t.Parallel()

	sem := sync.NewSemaphore(3)

	require.NoError(t, sem.Acquire(t.Context(), 2))
	require.False(t, sem.TryAcquire(2), "acquisition should fail when the requested weight exceeds remaining capacity")
	require.True(t, sem.TryAcquire(1), "failed acquisition should leave remaining capacity available")

	sem.Release(3)
	require.True(t, sem.TryAcquire(3), "release should restore the full configured capacity")
	sem.Release(3)
}

func TestSemaphoreAcquireWaitsForRelease(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sem := sync.NewSemaphore(2)
		require.NoError(t, sem.Acquire(t.Context(), 2))

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			done <- sem.Acquire(ctx, 2)
		}()

		synctest.Wait()
		require.Empty(t, done, "Acquire should wait until the requested capacity is released")

		sem.Release(2)
		require.NoError(t, <-done)
		sem.Release(2)
	})
}

func TestSemaphoreAcquireCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		sem := sync.NewSemaphore(3)
		require.NoError(t, sem.Acquire(t.Context(), 2))

		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		done := make(chan error, 1)
		go func() {
			done <- sem.Acquire(ctx, 2)
		}()

		synctest.Wait()
		cancel()
		require.ErrorIs(t, <-done, context.Canceled)
		require.True(t, sem.TryAcquire(1), "canceled acquisition should leave remaining capacity available")
		sem.Release(3)
	})
}
