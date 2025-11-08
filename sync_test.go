package sync_test

import (
	"context"
	"testing"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestWaitNoError(t *testing.T) {
	require.NoError(t, sync.Wait(t.Context(), time.Second, func(context.Context) error {
		return nil
	}))
}

func TestWaitError(t *testing.T) {
	require.Error(t, sync.Wait(t.Context(), time.Second, func(context.Context) error {
		return context.Canceled
	}))
}

func TestWaitContinue(t *testing.T) {
	require.NoError(t, sync.Wait(t.Context(), time.Microsecond, func(context.Context) error {
		time.Sleep(time.Second)
		return nil
	}))
}

func TestTimeoutNoError(t *testing.T) {
	require.NoError(t, sync.Timeout(t.Context(), time.Second, func(context.Context) error {
		return nil
	}))
}

func TestTimeoutError(t *testing.T) {
	require.Error(t, sync.Timeout(t.Context(), time.Second, func(context.Context) error {
		return context.Canceled
	}))
}

func TestTimeout(t *testing.T) {
	require.ErrorIs(t, context.DeadlineExceeded, sync.Timeout(t.Context(), time.Microsecond, func(context.Context) error {
		time.Sleep(time.Second)
		return nil
	}))
}
