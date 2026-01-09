package sync_test

import (
	"context"
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

func TestWaitContextCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	require.NoError(t, sync.Wait(ctx, time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return nil
		},
	}))
}

func TestTimeoutNoError(t *testing.T) {
	require.NoError(t, sync.Timeout(t.Context(), time.Second, sync.Hook{
		OnRun: func(context.Context) error {
			return nil
		},
	}))
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

func TestTimeoutOperationError(t *testing.T) {
	err := sync.Timeout(t.Context(), time.Microsecond, sync.Hook{
		OnRun: func(ctx context.Context) error {
			time.Sleep(time.Second)
			return ctx.Err()
		},
	})

	require.Error(t, err)
	require.True(t, sync.IsTimeoutError(err))
}
