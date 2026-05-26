package sync_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestValue(t *testing.T) {
	value := sync.NewValue[int]()

	require.Equal(t, 0, value.Load(), "new Value should load zero value")
	require.Equal(t, 0, value.Swap(2), "first Swap should return zero value")

	value.Store(1)
	require.Equal(t, 1, value.Load(), "Load should return stored value")
	require.Equal(t, 1, value.Swap(2), "Swap should return previous value")
	require.True(t, value.CompareAndSwap(2, 3), "CompareAndSwap should update matching value")
	require.False(t, value.CompareAndSwap(2, 4), "CompareAndSwap should reject stale value")
}

func TestNewValueDirectCall(t *testing.T) {
	require.Equal(t, 0, sync.NewValue[int]().Load(), "new Value should load zero value")
}

func TestValueCompareAndSwapPanicsWithNonComparableValues(t *testing.T) {
	value := sync.NewValue[any]()
	value.Store([]int{1})

	require.Panics(t, func() {
		value.CompareAndSwap([]int{1}, []int{2})
	})
}
