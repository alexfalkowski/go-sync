package sync_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestWaitNoError(t *testing.T) {
	require.NoError(t, sync.Wait(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return nil
		},
	}))
}

func TestWaitError(t *testing.T) {
	require.ErrorIs(t, sync.Wait(t.Context(), time.Second, sync.Hook{}), sync.ErrNoOnRunProvided)
	require.Error(t, sync.Wait(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return context.Canceled
		},
	}))
}

func TestWaitContinue(t *testing.T) {
	require.NoError(t, sync.Wait(t.Context(), time.Microsecond, sync.Hook{
		OnRun: func(context.Context) error {
			time.Sleep(time.Second)
			return nil
		},
	}))
}

func TestWaitNonPositiveTimeoutDoesNotRun(t *testing.T) {
	var called sync.Bool

	require.NoError(t, sync.Wait(t.Context(), 0, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	}))
	require.False(t, called.Load())
}

func TestWaitContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, sync.Wait(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return context.Canceled
		},
	}))
}

func TestWaitContextAlreadyCanceledDoesNotRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var called sync.Bool
	require.NoError(t, sync.Wait(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	}))
	time.Sleep(20 * time.Millisecond)
	require.False(t, called.Load())
}

func TestWaitReturnsNilWhenContextCanceledDuringRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		defer close(done)
		errCh <- sync.Wait(ctx, time.Second, sync.Hook{
			OnRun: func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				<-release
				return nil
			},
		})
	}()

	<-started
	cancel()

	require.NoError(t, <-errCh)
	close(release)
	<-done
}

func TestWaitReturnsNilWhenContextCanceledAfterRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())

	err := sync.Wait(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			cancel()
			return errors.New("ignored after cancel")
		},
	})

	require.NoError(t, err)
}

func TestTimeoutNoError(t *testing.T) {
	require.NoError(t, sync.Timeout(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return nil
		},
	}))
}

func TestTimeoutNonPositiveTimeoutDoesNotRun(t *testing.T) {
	var called sync.Bool

	err := sync.Timeout(t.Context(), 0, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, sync.ErrTimeout)
	require.True(t, sync.IsTimeoutError(err))
	require.False(t, called.Load())
}

func TestTimeoutError(t *testing.T) {
	require.ErrorIs(t, sync.Timeout(t.Context(), time.Second, sync.Hook{}), sync.ErrNoOnRunProvided)

	err := sync.Timeout(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return context.Canceled
		},
		OnError: func(_ context.Context, err error) error {
			return err
		},
	})

	require.ErrorIs(t, err, context.Canceled)
	require.False(t, sync.IsTimeoutError(err))
}

func TestWaitErrorHandlerReplacesError(t *testing.T) {
	runErr := errors.New("run failed")
	wrappedErr := errors.New("wrapped run failed")
	handled := make(chan error, 1)

	err := sync.Wait(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return runErr
		},
		OnError: func(_ context.Context, err error) error {
			handled <- err
			return wrappedErr
		},
	})

	require.ErrorIs(t, err, wrappedErr)
	require.NotErrorIs(t, err, runErr)
	require.ErrorIs(t, <-handled, runErr)
}

func TestTimeoutErrorHandlerReplacesError(t *testing.T) {
	runErr := errors.New("run failed")
	wrappedErr := errors.New("wrapped run failed")
	handled := make(chan error, 1)

	err := sync.Timeout(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return runErr
		},
		OnError: func(_ context.Context, err error) error {
			handled <- err
			return wrappedErr
		},
	})

	require.ErrorIs(t, err, wrappedErr)
	require.NotErrorIs(t, err, runErr)
	require.ErrorIs(t, <-handled, runErr)
}

func TestTimeoutOperationError(t *testing.T) {
	err := sync.Timeout(t.Context(), time.Microsecond, sync.Hook{
		OnRun: func(ctx context.Context) error {
			time.Sleep(time.Second)
			return context.Cause(ctx)
		},
	})

	require.Error(t, err)
	require.ErrorIs(t, err, sync.ErrTimeout)
	require.True(t, sync.IsTimeoutError(err))
}

func TestTimeoutContextAlreadyCanceledDoesNotRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	var called sync.Bool
	err := sync.Timeout(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, context.Canceled)
	time.Sleep(20 * time.Millisecond)
	require.False(t, called.Load())
}

func TestTimeoutReturnsContextCause(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	expected := errors.New("parent canceled")
	cancel(expected)

	var called sync.Bool
	err := sync.Timeout(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	})

	require.ErrorIs(t, err, expected)
	time.Sleep(20 * time.Millisecond)
	require.False(t, called.Load())
}
