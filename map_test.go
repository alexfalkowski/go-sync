package sync_test

import (
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestMapLoadOrStore(t *testing.T) {
	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.LoadOrStore("test", "test")
	require.Equal(t, "test", v)
	require.False(t, ok)

	v, ok = m.LoadOrStore("test", "test")
	require.Equal(t, "test", v)
	require.True(t, ok)
}

func TestMapLoad(t *testing.T) {
	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.Load("test")
	require.Empty(t, v)
	require.False(t, ok)

	m.Store("test", "test")

	v, ok = m.Load("test")
	require.Equal(t, "test", v)
	require.True(t, ok)
}

func TestMapLoadAndDelete(t *testing.T) {
	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.LoadAndDelete("test")
	require.Empty(t, v)
	require.False(t, ok)

	m.Store("test", "test")

	v, ok = m.LoadAndDelete("test")
	require.Equal(t, "test", v)
	require.True(t, ok)
}

func TestMapDelete(t *testing.T) {
	m := sync.NewMap[string, string]()
	defer m.Clear()

	m.Delete("test")
}

func TestMapSwap(t *testing.T) {
	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.Swap("test", "test")
	require.Empty(t, v)
	require.False(t, ok)

	m.Store("test", "bob")

	v, ok = m.Swap("test", "test")
	require.Equal(t, "bob", v)
	require.True(t, ok)
}

func TestMapCompare(t *testing.T) {
	m := sync.NewMap[string, string]()
	defer m.Clear()

	require.False(t, m.CompareAndSwap("test", "test", "test"))
	require.False(t, m.CompareAndDelete("test", "test"))
}

func TestMapRange(t *testing.T) {
	m := sync.NewMap[string, string]()
	defer m.Clear()

	m.Store("test", "test")

	m.Range(func(key string, value string) bool {
		require.Equal(t, "test", key)
		require.Equal(t, "test", value)
		return true
	})
}
