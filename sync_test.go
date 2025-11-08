package sync_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

// NewBufferPool for sync.
func NewBufferPool() *BufferPool {
	return &BufferPool{pool: sync.NewPool[bytes.Buffer]()}
}

// BufferPool for sync.
type BufferPool struct {
	pool *sync.Pool[bytes.Buffer]
}

// Get a new buffer.
func (p *BufferPool) Get() *bytes.Buffer {
	return p.pool.Get()
}

// Put the buffer back.
func (p *BufferPool) Put(buffer *bytes.Buffer) {
	buffer.Reset()
	p.pool.Put(buffer)
}

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

func TestPool(t *testing.T) {
	pool := NewBufferPool()
	buffer := pool.Get()
	defer pool.Put(buffer)

	require.NotNil(t, buffer)
}

func BenchmarkPool(b *testing.B) {
	pool := NewBufferPool()

	b.Run("Get", func(b *testing.B) {
		for b.Loop() {
			buffer := pool.Get()
			pool.Put(buffer)
		}
	})
}
