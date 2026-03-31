package sync_test

import (
	"context"
	"testing"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestWorkerSchedule(t *testing.T) {
	startTime := time.Now()
	worker := sync.NewWorker(10)
	for range 20 {
		err := worker.Schedule(t.Context(), 2*time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				time.Sleep(time.Second)
				return nil
			},
		})
		require.NoError(t, err)
	}

	worker.Wait()
	require.WithinDuration(t, time.Now(), startTime, 3*time.Second)
}

func TestNewWorkerDirectCall(t *testing.T) {
	require.ErrorIs(t, sync.NewWorker(1).Schedule(t.Context(), time.Second, sync.Hook{}), sync.ErrNoOnRunProvided)
}

func TestWorkerScheduleTimeout(t *testing.T) {
	worker := sync.NewWorker(1)
	_ = worker.Schedule(t.Context(), 10*time.Millisecond, sync.Hook{
		OnRun: func(context.Context) error {
			time.Sleep(time.Second)
			return nil
		},
	})

	err := worker.Schedule(t.Context(), 10*time.Millisecond, sync.Hook{
		OnRun: func(context.Context) error {
			time.Sleep(time.Second)
			return nil
		},
	})
	require.Error(t, err)
	require.True(t, sync.IsTimeoutError(err))

	worker.Wait()
}

func TestWorkerScheduleError(t *testing.T) {
	worker := sync.NewWorker(1)

	require.ErrorIs(t, worker.Schedule(t.Context(), time.Second, sync.Hook{}), sync.ErrNoOnRunProvided)

	startTime := time.Now()
	err := worker.Schedule(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return context.Canceled
		},
		OnError: func(_ context.Context, err error) error {
			require.ErrorIs(t, err, context.Canceled)
			return err
		},
	})
	require.NoError(t, err)

	worker.Wait()
	require.WithinDuration(t, time.Now(), startTime, time.Second)
}

func TestWorkerScheduleNotCanceledImmediately(t *testing.T) {
	worker := sync.NewWorker(1)
	c := make(chan error, 1)

	err := worker.Schedule(t.Context(), time.Second, sync.Hook{
		OnRun: func(ctx context.Context) error {
			time.Sleep(20 * time.Millisecond)
			c <- ctx.Err()
			return nil
		},
	})
	require.NoError(t, err)

	worker.Wait()
	require.NoError(t, <-c)
}

func TestWorkerScheduleContextAlreadyCanceledDoesNotRun(t *testing.T) {
	worker := sync.NewWorker(1)

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var called sync.Bool
	err := worker.Schedule(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, context.Canceled)
	worker.Wait()
	time.Sleep(20 * time.Millisecond)
	require.False(t, called.Load())
}

func TestWorkerScheduleTimeoutIncludesQueueWait(t *testing.T) {
	worker := sync.NewWorker(1)
	started := make(chan struct{})
	c := make(chan error, 1)

	err := worker.Schedule(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			close(started)
			time.Sleep(150 * time.Millisecond)
			return nil
		},
	})
	require.NoError(t, err)
	<-started

	begin := time.Now()
	err = worker.Schedule(t.Context(), 250*time.Millisecond, sync.Hook{
		OnRun: func(ctx context.Context) error {
			<-ctx.Done()
			c <- ctx.Err()
			return nil
		},
	})
	require.NoError(t, err)

	worker.Wait()
	require.ErrorIs(t, <-c, context.DeadlineExceeded)
	require.Less(t, time.Since(begin), 325*time.Millisecond)
}

func BenchmarkWorker(b *testing.B) {
	worker := sync.NewWorker(uint(b.N)) //nolint:gosec
	for b.Loop() {
		_ = worker.Schedule(b.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				return nil
			},
		})
	}
	worker.Wait()
}
