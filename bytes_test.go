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
}

func BenchmarkPool(b *testing.B) {
	pool := sync.NewBufferPool()
	for b.Loop() {
		buffer := pool.Get()
		pool.Put(buffer)
	}
}
