package sync_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"testing/synctest"
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
	err := sync.Wait(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return context.Canceled
		},
	})
	require.ErrorIs(t, err, context.Canceled)
}

func TestWaitContinue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		done := make(chan struct{})

		err := sync.Wait(t.Context(), time.Second, sync.Hook{
			OnRun: func(context.Context) error {
				defer close(done)
				<-release
				return nil
			},
		})
		require.NoError(t, err)

		close(release)
		<-done
	})
}

func TestWaitNonPositiveTimeoutDoesNotRun(t *testing.T) {
	var called sync.Bool

	require.NoError(t, sync.Wait(t.Context(), 0, sync.Hook{
		OnRun: func(context.Context) error {
			called.Store(true)
			return nil
		},
	}))
	require.False(t, called.Load(), "Wait should not run hook with non-positive timeout")
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
	require.False(t, called.Load(), "Wait should not run hook when context is already canceled")
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
	require.True(t, sync.IsTimeoutError(err), "non-positive Timeout error should be classified as timeout")
	require.False(t, called.Load(), "Timeout should not run hook with non-positive timeout")
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
	require.False(t, sync.IsTimeoutError(err), "context cancellation should not be classified as timeout")
}

func TestHookError(t *testing.T) {
	runErr := errors.New("run failed")
	handledErr := errors.New("handled run failed")

	called := false
	hook := sync.Hook{
		OnError: func(context.Context, error) error {
			called = true
			return handledErr
		},
	}
	require.NoError(t, hook.Error(t.Context(), nil))
	require.False(t, called, "OnError should not be called for nil errors")

	hook = sync.Hook{}
	require.ErrorIs(t, hook.Error(t.Context(), runErr), runErr)

	type contextKey struct{}
	ctx := context.WithValue(t.Context(), contextKey{}, "marker")
	hook = sync.Hook{
		OnError: func(got context.Context, err error) error {
			require.Equal(t, ctx, got)
			require.Equal(t, "marker", got.Value(contextKey{}))
			require.ErrorIs(t, err, runErr)
			return handledErr
		},
	}
	require.ErrorIs(t, hook.Error(ctx, runErr), handledErr)
}

func TestIsTimeoutError(t *testing.T) {
	tests := map[string]struct {
		err     error
		timeout bool
	}{
		"nil": {
			err:     nil,
			timeout: false,
		},
		"package timeout": {
			err:     sync.ErrTimeout,
			timeout: true,
		},
		"deadline exceeded": {
			err:     context.DeadlineExceeded,
			timeout: true,
		},
		"wrapped deadline exceeded": {
			err:     fmt.Errorf("wrapped: %w", context.DeadlineExceeded),
			timeout: true,
		},
		"context canceled": {
			err:     context.Canceled,
			timeout: false,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.timeout, sync.IsTimeoutError(test.err))
		})
	}
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
	synctest.Test(t, func(t *testing.T) {
		err := sync.Timeout(t.Context(), time.Second, sync.Hook{
			OnRun: func(ctx context.Context) error {
				<-ctx.Done()
				return context.Cause(ctx)
			},
		})

		require.Error(t, err)
		require.ErrorIs(t, err, sync.ErrTimeout)
		require.True(t, sync.IsTimeoutError(err), "operation timeout should be classified as timeout")
	})
}

func TestTimeoutReturnsParentCauseWhenContextCanceledDuringRun(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	expected := errors.New("parent canceled")
	started := make(chan struct{})
	release := make(chan struct{})
	done := make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		defer close(done)
		errCh <- sync.Timeout(ctx, time.Second, sync.Hook{
			OnRun: func(ctx context.Context) error {
				close(started)
				<-ctx.Done()
				<-release
				return nil
			},
		})
	}()

	<-started
	cancel(expected)

	require.ErrorIs(t, <-errCh, expected)
	close(release)
	<-done
}

func TestTimeoutReturnsParentCauseWhenContextCanceledAfterRun(t *testing.T) {
	ctx, cancel := context.WithCancelCause(t.Context())
	expected := errors.New("parent canceled")
	runErr := errors.New("ignored after cancel")

	err := sync.Timeout(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			cancel(expected)
			return runErr
		},
	})

	require.ErrorIs(t, err, expected)
	require.NotErrorIs(t, err, runErr)
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
	require.False(t, called.Load(), "Timeout should not run hook when context is already canceled")
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
	require.False(t, called.Load(), "Timeout should not run hook when parent context has a cause")
}
