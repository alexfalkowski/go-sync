package sync_test

import (
	"io"
	"strings"
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

func TestMapLoadOrStoreNilInterfaceValue(t *testing.T) {
	m := sync.NewMap[string, io.Reader]()
	defer m.Clear()

	var r io.Reader
	v, loaded := m.LoadOrStore("test", r)
	require.Nil(t, v)
	require.False(t, loaded)

	v, loaded = m.LoadOrStore("test", strings.NewReader("x"))
	require.Nil(t, v)
	require.True(t, loaded)
}

func TestMapRangeNilInterfaceValue(t *testing.T) {
	m := sync.NewMap[string, io.Reader]()
	defer m.Clear()

	var r io.Reader
	m.Store("reader", r)

	called := false
	require.NotPanics(t, func() {
		m.Range(func(key string, value io.Reader) bool {
			called = true
			require.Equal(t, "reader", key)
			require.Nil(t, value)
			return true
		})
	})
	require.True(t, called)
}
