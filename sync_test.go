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
	err := sync.Timeout(t.Context(), time.Second, func(context.Context) error {
		return context.Canceled
	})
	require.Error(t, err)
	require.True(t, sync.IsTimeoutError(err))
}

func TestTimeoutOperationError(t *testing.T) {
	err := sync.Timeout(t.Context(), time.Microsecond, func(context.Context) error {
		time.Sleep(time.Second)
		return nil
	})
	require.Error(t, err)
	require.True(t, sync.IsTimeoutError(err))
}

func TestPool(t *testing.T) {
	pool := sync.NewBufferPool()
	buffer := pool.Get()
	defer pool.Put(buffer)

	require.NotNil(t, buffer)
}

func BenchmarkPool(b *testing.B) {
	pool := sync.NewBufferPool()

	b.Run("Get", func(b *testing.B) {
		for b.Loop() {
			buffer := pool.Get()
			pool.Put(buffer)
		}
	})
}
