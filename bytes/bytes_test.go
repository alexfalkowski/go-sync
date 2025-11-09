package bytes_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync/bytes"
	"github.com/stretchr/testify/require"
)

func TestPool(t *testing.T) {
	pool := bytes.NewBufferPool()
	buffer := pool.Get()
	defer pool.Put(buffer)

	require.NotNil(t, buffer)
}

func BenchmarkPool(b *testing.B) {
	pool := bytes.NewBufferPool()

	b.Run("Get", func(b *testing.B) {
		for b.Loop() {
			buffer := pool.Get()
			pool.Put(buffer)
		}
	})
}
