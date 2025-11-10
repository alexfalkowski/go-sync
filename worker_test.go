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
	startTime := time.Now()
	worker := sync.NewWorker(1)
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

func BenchmarkWorker(b *testing.B) {
	worker := sync.NewWorker(b.N)
	for b.Loop() {
		_ = worker.Schedule(b.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				return nil
			},
		})
	}
	worker.Wait()
}
