package sync_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestPool(t *testing.T) {
	pool := sync.NewBufferPool()
	buffer := pool.Get()
	defer pool.Put(buffer)

	require.NotNil(t, buffer)
	require.Empty(t, pool.Copy(buffer))
	require.NotPanics(t, func() { pool.Put(nil) })
	require.NotPanics(t, func() { pool.Copy(nil) })
}

func TestNewBufferPoolDirectCall(t *testing.T) {
	require.Nil(t, sync.NewBufferPool().Copy(nil))
}

func TestBufferPoolPutResetsBuffer(t *testing.T) {
	pool := sync.NewBufferPool()
	buffer := pool.Get()
	buffer.WriteString("hello")

	pool.Put(buffer)

	require.Empty(t, buffer.String(), "Put should reset returned buffer")
}

func TestBufferPoolCopyDoesNotAliasBuffer(t *testing.T) {
	pool := sync.NewBufferPool()
	buffer := pool.Get()
	defer pool.Put(buffer)

	buffer.WriteString("hello")
	copy := pool.Copy(buffer)
	buffer.Reset()
	buffer.WriteString("changed")

	require.Equal(t, "hello", string(copy), "Copy should not alias buffer storage")
}

func BenchmarkPool(b *testing.B) {
	bs := make([]byte, 1024)
	pool := sync.NewBufferPool()

	for b.Loop() {
		buffer := pool.Get()
		buffer.Write(bs)
		_ = pool.Copy(buffer)
		pool.Put(buffer)
	}
}
