package sync_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestValue(t *testing.T) {
	var value sync.Value[int]

	value.Store(1)
	require.Equal(t, 1, value.Load())
	require.Equal(t, 1, value.Swap(2))
	require.True(t, value.CompareAndSwap(2, 3))
}
