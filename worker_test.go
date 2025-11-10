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
		worker.Schedule(t.Context(), sync.Lifecycle{
			OnRun: func(context.Context) error {
				time.Sleep(time.Second)
				return nil
			},
		})
	}
	worker.Wait()

	require.WithinDuration(t, time.Now(), startTime, 3*time.Second)
}

func TestWorkerScheduleError(t *testing.T) {
	startTime := time.Now()
	worker := sync.NewWorker(1)
	worker.Schedule(t.Context(), sync.Lifecycle{
		OnRun: func(context.Context) error {
			return context.Canceled
		},
		OnError: func(_ context.Context, err error) error {
			require.ErrorIs(t, err, context.Canceled)
			return err
		},
	})
	worker.Wait()

	require.WithinDuration(t, time.Now(), startTime, time.Second)
}

func BenchmarkWorker(b *testing.B) {
	worker := sync.NewWorker(b.N)
	b.Run("Schedule", func(b *testing.B) {
		for b.Loop() {
			worker.Schedule(b.Context(), sync.Lifecycle{
				OnRun: func(context.Context) error {
					return nil
				},
			})
		}
	})
}
