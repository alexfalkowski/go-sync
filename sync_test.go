package sync_test

import (
	"context"
	"testing"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestWaitNoError(t *testing.T) {
	sync.SetErrorHandler(nil)
	require.NoError(t, sync.Wait(t.Context(), time.Second, func(context.Context) error {
		return nil
	}))
}

func TestWaitError(t *testing.T) {
	sync.SetErrorHandler(sync.DefaultErrorHandler)
	require.Error(t, sync.Wait(t.Context(), time.Second, func(context.Context) error {
		return context.Canceled
	}))
}

func TestWaitContinue(t *testing.T) {
	sync.SetErrorHandler(sync.DefaultErrorHandler)
	require.NoError(t, sync.Wait(t.Context(), time.Microsecond, func(context.Context) error {
		time.Sleep(time.Second)
		return nil
	}))
}

func TestTimeoutNoError(t *testing.T) {
	sync.SetErrorHandler(sync.DefaultErrorHandler)
	require.NoError(t, sync.Timeout(t.Context(), time.Second, func(context.Context) error {
		return nil
	}))
}

func TestTimeoutError(t *testing.T) {
	sync.SetErrorHandler(func(_ context.Context, err error) error {
		require.ErrorIs(t, err, context.Canceled)
		return err
	})
	err := sync.Timeout(t.Context(), time.Second, func(context.Context) error {
		return context.Canceled
	})

	require.Error(t, err)
	require.True(t, sync.IsTimeoutError(err))
}

func TestTimeoutOperationError(t *testing.T) {
	sync.SetErrorHandler(sync.DefaultErrorHandler)
	err := sync.Timeout(t.Context(), time.Microsecond, func(ctx context.Context) error {
		time.Sleep(time.Second)
		return ctx.Err()
	})

	require.Error(t, err)
	require.True(t, sync.IsTimeoutError(err))
}

func TestWorkerSchedule(t *testing.T) {
	startTime := time.Now()
	sync.SetErrorHandler(sync.DefaultErrorHandler)
	w := sync.NewWorker(10)
	for range 20 {
		w.Schedule(t.Context(), func(context.Context) error {
			time.Sleep(time.Second)
			return nil
		})
	}
	w.Wait()

	require.WithinDuration(t, time.Now(), startTime, 3*time.Second)
}

func TestWorkerScheduleError(t *testing.T) {
	startTime := time.Now()
	sync.SetErrorHandler(func(_ context.Context, err error) error {
		require.ErrorIs(t, err, context.Canceled)
		return err
	})
	w := sync.NewWorker(1)
	w.Schedule(t.Context(), func(context.Context) error {
		return context.Canceled
	})
	w.Wait()

	require.WithinDuration(t, time.Now(), startTime, time.Second)
}
