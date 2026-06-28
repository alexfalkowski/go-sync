package sync_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestPoolPutNilDoesNotPoisonPool(t *testing.T) {
	t.Parallel()

	pool := sync.NewPool[int]()

	require.NotPanics(t, func() {
		pool.Put(nil)
	})

	value := pool.Get()
	require.NotNil(t, value)
	require.Equal(t, 0, *value, "pool should allocate zero value")
	pool.Put(value)
}

func TestPoolZeroValue(t *testing.T) {
	t.Parallel()

	var pool sync.Pool[int]

	value := pool.Get()
	require.NotNil(t, value)
	require.Equal(t, 0, *value, "zero-value pool should allocate zero value")

	*value = 1
	pool.Put(value)
}

func TestPoolNew(t *testing.T) {
	t.Parallel()

	type item struct {
		values []string
	}

	pool := sync.Pool[item]{
		New: func() *item {
			return &item{values: make([]string, 0, 2)}
		},
	}

	value := pool.Get()
	require.NotNil(t, value)
	require.NotNil(t, value.values)
	require.Empty(t, value.values, "custom constructor should initialize slice length")
	require.Equal(t, 2, cap(value.values), "custom constructor should initialize slice capacity")
}

func TestPoolNilNewAllocatesZeroValue(t *testing.T) {
	t.Parallel()

	pool := sync.Pool[int]{
		New: nil,
	}

	value := pool.Get()
	require.NotNil(t, value)
	require.Equal(t, 0, *value, "nil constructor should allocate zero value")
}

func TestNewPoolDirectCall(t *testing.T) {
	t.Parallel()

	pool := sync.NewPool[int]()
	require.NotNil(t, pool.New, "NewPool should install the default constructor")

	value := pool.Get()
	require.NotNil(t, value)
	require.Equal(t, 0, *value, "pool should allocate zero value")
}
