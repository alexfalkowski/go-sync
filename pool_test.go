package sync_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestGenericPoolPutNilDoesNotPoisonPool(t *testing.T) {
	pool := sync.NewPool[int]()

	require.NotPanics(t, func() {
		pool.Put(nil)
	})

	value := pool.Get()
	require.NotNil(t, value)
	require.Equal(t, 0, *value)
	pool.Put(value)
}
