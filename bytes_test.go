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
