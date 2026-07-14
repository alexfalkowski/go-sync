package sync_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/alexfalkowski/go-sync/internal/test"
	"github.com/stretchr/testify/require"
)

func TestWorkerSchedule(t *testing.T) {
	const (
		limit = 10
		total = 20
	)

	worker := sync.NewWorker(limit)
	probe := test.NewWorkerScheduleProbe(limit, total)

	for range total {
		probe.Schedule(t.Context(), worker)
	}

	probe.RequireLimitReached(t)
	probe.ReleaseAll()
	probe.RequireScheduled(t)
	require.NoError(t, worker.Wait(t.Context()))
	probe.RequireNeverExceeded(t)
}

func TestNewWorkerDirectCall(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, sync.NewWorker(1).Schedule(t.Context(), time.Second, sync.Hook{}), sync.ErrNoOnRunProvided)
}

func TestWorkerTrySchedule(t *testing.T) {
	worker := sync.NewWorker(1)
	var called sync.Bool

	err := worker.TrySchedule(t.Context(), sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})
	require.NoError(t, err)

	require.NoError(t, worker.Wait(t.Context()))
	require.True(t, called.Load())
}

func TestWorkerTryScheduleFullDoesNotRun(t *testing.T) {
	worker := sync.NewWorker(1)
	started := make(chan struct{})
	release := make(chan struct{})

	err := worker.TrySchedule(t.Context(), sync.Hook{
		OnRun: func(context.Context) error {
			close(started)
			<-release
			return nil
		},
	})
	require.NoError(t, err)
	<-started

	var called sync.Bool
	err = worker.TrySchedule(t.Context(), sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, sync.ErrWorkerFull)
	require.False(t, called.Load(), "TrySchedule should not run hook when worker has no free slot")

	close(release)
	require.NoError(t, worker.Wait(t.Context()))
}

func TestWorkerTryScheduleZeroCapacityDoesNotRun(t *testing.T) {
	t.Parallel()

	worker := sync.NewWorker(0)
	var called sync.Bool

	err := worker.TrySchedule(t.Context(), sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, sync.ErrWorkerFull)
	require.NoError(t, worker.Wait(t.Context()))
	require.False(t, called.Load(), "zero-capacity worker should not run hook")
}

func TestWorkerTryScheduleError(t *testing.T) {
	worker := sync.NewWorker(1)
	runErr := errors.New("run failed")
	handled := make(chan error, 1)

	require.ErrorIs(t, worker.TrySchedule(t.Context(), sync.Hook{}), sync.ErrNoOnRunProvided)

	err := worker.TrySchedule(t.Context(), sync.Hook{
		OnRun: func(context.Context) error {
			return runErr
		},
		OnError: func(_ context.Context, err error) error {
			handled <- err
			return errors.New("wrapped run failed")
		},
	})
	require.NoError(t, err)

	require.NoError(t, worker.Wait(t.Context()))
	require.ErrorIs(t, <-handled, runErr)
}

func TestWorkerTryScheduleContextAlreadyCanceledDoesNotRun(t *testing.T) {
	t.Parallel()

	worker := sync.NewWorker(1)

	ctx, cancel := context.WithCancelCause(t.Context())
	expected := errors.New("parent canceled")
	cancel(expected)

	var called sync.Bool
	err := worker.TrySchedule(ctx, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, expected)
	require.NoError(t, worker.Wait(t.Context()))
	require.False(t, called.Load(), "TrySchedule should not run hook when context is already canceled")
}

func TestWorkerScheduleTimeout(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := sync.NewWorker(1)
		started := make(chan struct{})
		release := make(chan struct{})

		err := worker.Schedule(t.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				close(started)
				<-release
				return nil
			},
		})
		require.NoError(t, err)
		<-started

		err = worker.Schedule(t.Context(), 10*time.Millisecond, sync.Hook{
			OnRun: func(context.Context) error {
				return nil
			},
		})
		require.Error(t, err)
		require.ErrorIs(t, err, sync.ErrTimeout)
		require.True(t, sync.IsTimeoutError(err), "scheduling timeout should be classified as timeout")

		close(release)
		require.NoError(t, worker.Wait(t.Context()))
	})
}

func TestWorkerScheduleNonPositiveTimeoutDoesNotRun(t *testing.T) {
	t.Parallel()

	worker := sync.NewWorker(1)
	var called sync.Bool

	err := worker.Schedule(t.Context(), 0, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, sync.ErrTimeout)
	require.True(t, sync.IsTimeoutError(err), "non-positive schedule timeout should be classified as timeout")
	require.NoError(t, worker.Wait(t.Context()))
	require.False(t, called.Load(), "non-positive schedule timeout should not run hook")
}

func TestWorkerScheduleZeroCapacityTimeoutDoesNotRun(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := sync.NewWorker(0)
		var called sync.Bool

		err := worker.Schedule(t.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				called.Store(true)
				return nil
			},
		})

		require.ErrorIs(t, err, sync.ErrTimeout)
		require.True(t, sync.IsTimeoutError(err), "zero-capacity schedule timeout should be classified as timeout")
		require.NoError(t, worker.Wait(t.Context()))
		require.False(t, called.Load(), "zero-capacity worker should not run hook before timeout")
	})
}

func TestWorkerScheduleError(t *testing.T) {
	worker := sync.NewWorker(1)
	handled := make(chan error, 1)

	require.ErrorIs(t, worker.Schedule(t.Context(), time.Second, sync.Hook{}), sync.ErrNoOnRunProvided)

	err := worker.Schedule(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return context.Canceled
		},
		OnError: func(_ context.Context, err error) error {
			handled <- err
			return err
		},
	})
	require.NoError(t, err)

	require.NoError(t, worker.Wait(t.Context()))
	require.ErrorIs(t, <-handled, context.Canceled)
}

func TestWorkerScheduleCallsOnError(t *testing.T) {
	worker := sync.NewWorker(1)
	runErr := errors.New("run failed")
	handled := make(chan error, 1)

	err := worker.Schedule(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return runErr
		},
		OnError: func(_ context.Context, err error) error {
			handled <- err
			return errors.New("wrapped run failed")
		},
	})
	require.NoError(t, err)

	require.NoError(t, worker.Wait(t.Context()))
	require.ErrorIs(t, <-handled, runErr)
}

func TestWorkerScheduleNotCanceledImmediately(t *testing.T) {
	worker := sync.NewWorker(1)
	errCh := make(chan error, 1)
	release := make(chan struct{})

	err := worker.Schedule(t.Context(), time.Second, sync.Hook{
		OnRun: func(ctx context.Context) error {
			<-release
			errCh <- ctx.Err()
			return nil
		},
	})
	require.NoError(t, err)

	close(release)
	require.NoError(t, worker.Wait(t.Context()))
	require.NoError(t, <-errCh)
}

func TestWorkerScheduleContextAlreadyCanceledDoesNotRun(t *testing.T) {
	t.Parallel()

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
	require.NoError(t, worker.Wait(t.Context()))
	require.False(t, called.Load(), "Schedule should not run hook when context is already canceled")
}

func TestWorkerScheduleReturnsContextCause(t *testing.T) {
	t.Parallel()

	worker := sync.NewWorker(1)

	ctx, cancel := context.WithCancelCause(t.Context())
	expected := errors.New("parent canceled")
	cancel(expected)

	var called sync.Bool
	err := worker.Schedule(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, expected)
	require.NoError(t, worker.Wait(t.Context()))
	require.False(t, called.Load(), "Schedule should not run hook when parent context has a cause")
}

func TestWorkerScheduleReturnsParentCauseWhenContextCanceledWhileQueued(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := sync.NewWorker(1)
		started := make(chan struct{})
		release := make(chan struct{})

		err := worker.Schedule(t.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				close(started)
				<-release
				return nil
			},
		})
		require.NoError(t, err)
		<-started

		ctx, cancel := context.WithCancelCause(t.Context())
		expected := errors.New("parent canceled")
		var called sync.Bool
		errCh := make(chan error, 1)

		go func() {
			errCh <- worker.Schedule(ctx, time.Second, sync.Hook{
				OnRun: func(context.Context) error {
					called.Store(true)
					return nil
				},
			})
		}()
		synctest.Wait()
		cancel(expected)

		require.ErrorIs(t, <-errCh, expected)
		require.False(t, called.Load(), "queued Schedule should not run hook after parent cancellation")

		close(release)
		require.NoError(t, worker.Wait(t.Context()))
	})
}

func TestWorkerScheduleTimeoutIncludesQueueWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := sync.NewWorker(1)
		started := make(chan struct{})
		causeCh := make(chan error, 1)

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
				causeCh <- context.Cause(ctx)
				return nil
			},
		})
		require.NoError(t, err)

		require.NoError(t, worker.Wait(t.Context()))
		require.ErrorIs(t, <-causeCh, sync.ErrTimeout)
		require.Equal(t, 250*time.Millisecond, time.Since(begin), "timeout budget should include queue wait")
	})
}

func TestWorkerScheduleTimeoutBudgetExpiresAfterScheduling(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := sync.NewWorker(1)
		causeCh := make(chan error, 1)

		err := worker.Schedule(t.Context(), time.Second, sync.Hook{
			OnRun: func(ctx context.Context) error {
				<-ctx.Done()
				causeCh <- context.Cause(ctx)
				return nil
			},
		})
		require.NoError(t, err)

		require.NoError(t, worker.Wait(t.Context()))
		require.ErrorIs(t, <-causeCh, sync.ErrTimeout)
	})
}

func TestWorkerWaitReturnsNilWhenHandlersFinish(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := sync.NewWorker(1)
		release := make(chan struct{})

		err := worker.Schedule(t.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				<-release
				return nil
			},
		})
		require.NoError(t, err)

		errCh := make(chan error, 1)
		go func() {
			errCh <- worker.Wait(t.Context())
		}()
		synctest.Wait()

		close(release)
		require.NoError(t, <-errCh)
	})
}

func TestWorkerWaitReturnsCauseAtDeadline(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		worker := sync.NewWorker(1)
		release := make(chan struct{})

		err := worker.Schedule(t.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				<-release
				return nil
			},
		})
		require.NoError(t, err)

		ctx, cancel := context.WithTimeoutCause(t.Context(), 10*time.Millisecond, sync.ErrTimeout)
		defer cancel()

		err = worker.Wait(ctx)

		require.ErrorIs(t, err, sync.ErrTimeout)

		close(release)
		require.NoError(t, worker.Wait(t.Context()))
	})
}

func TestWorkerWaitCanReturnCauseEvenWhenAlreadyDone(t *testing.T) {
	t.Parallel()

	worker := sync.NewWorker(1)

	err := worker.Schedule(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error { return nil },
	})
	require.NoError(t, err)
	require.NoError(t, worker.Wait(t.Context()))

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	err = worker.Wait(ctx)

	require.ErrorIs(t, err, context.Canceled,
		"Wait's tie-break is best-effort: an already-done ctx can still win "+
			"over handlers that finished before Wait was called")
}
