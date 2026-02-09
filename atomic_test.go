package sync_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestValue(t *testing.T) {
	value := sync.NewValue[int]()

	require.Equal(t, 0, value.Load())
	require.Equal(t, 0, value.Swap(2))

	value.Store(1)
	require.Equal(t, 1, value.Load())
	require.Equal(t, 1, value.Swap(2))
	require.True(t, value.CompareAndSwap(2, 3))
}
