package sync_test

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestMapLoadOrStore(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.LoadOrStore("test", "test")
	require.Equal(t, "test", v)
	require.False(t, ok, "first LoadOrStore should store value")

	v, ok = m.LoadOrStore("test", "test")
	require.Equal(t, "test", v)
	require.True(t, ok, "second LoadOrStore should load existing value")
}

func TestNewMapDirectCall(t *testing.T) {
	t.Parallel()

	v, ok := sync.NewMap[string, string]().Load("test")
	require.Empty(t, v)
	require.False(t, ok, "new map should not contain key")
}

func TestMapLoad(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.Load("test")
	require.Empty(t, v)
	require.False(t, ok, "Load should report missing key")

	m.Store("test", "test")

	v, ok = m.Load("test")
	require.Equal(t, "test", v)
	require.True(t, ok, "Load should report stored key")
}

func TestMapLoadAndDelete(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.LoadAndDelete("test")
	require.Empty(t, v)
	require.False(t, ok, "LoadAndDelete should report missing key")

	m.Store("test", "test")

	v, ok = m.LoadAndDelete("test")
	require.Equal(t, "test", v)
	require.True(t, ok, "LoadAndDelete should report deleted key")
}

func TestMapDelete(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, string]()
	defer m.Clear()

	m.Store("test", "test")
	m.Delete("test")

	v, ok := m.Load("test")
	require.Empty(t, v)
	require.False(t, ok, "Delete should remove stored key")
}

func TestMapSwap(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, string]()
	defer m.Clear()

	v, ok := m.Swap("test", "test")
	require.Empty(t, v)
	require.False(t, ok, "Swap should report missing previous value")

	m.Store("test", "bob")

	v, ok = m.Swap("test", "test")
	require.Equal(t, "bob", v)
	require.True(t, ok, "Swap should report replaced value")
}

func TestMapCompare(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, string]()
	defer m.Clear()

	require.False(t, m.CompareAndSwap("test", "test", "test"), "CompareAndSwap should reject missing key")
	require.False(t, m.CompareAndDelete("test", "test"), "CompareAndDelete should reject missing key")

	m.Store("test", "test")
	require.True(t, m.CompareAndSwap("test", "test", "updated"), "CompareAndSwap should update matching value")

	v, ok := m.Load("test")
	require.True(t, ok, "Load should find swapped key")
	require.Equal(t, "updated", v)

	require.True(t, m.CompareAndDelete("test", "updated"), "CompareAndDelete should delete matching value")

	_, ok = m.Load("test")
	require.False(t, ok, "CompareAndDelete should remove key")
}

func TestMapComparePanicsWithNonComparableValues(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, any]()
	defer m.Clear()

	m.Store("test", []int{1})

	require.Panics(t, func() {
		m.CompareAndSwap("test", []int{1}, []int{2})
	}, "CompareAndSwap should panic with non-comparable values")

	require.Panics(t, func() {
		m.CompareAndDelete("test", []int{1})
	}, "CompareAndDelete should panic with non-comparable values")
}

func TestMapRange(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

	m := sync.NewMap[string, io.Reader]()
	defer m.Clear()

	var nilReader io.Reader
	v, loaded := m.LoadOrStore("test", nilReader)
	require.Nil(t, v)
	require.False(t, loaded, "first nil interface LoadOrStore should store value")

	v, loaded = m.LoadOrStore("test", strings.NewReader("x"))
	require.Nil(t, v)
	require.True(t, loaded, "second nil interface LoadOrStore should load stored nil")
}

func TestMapLoadNilInterfaceValue(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, io.Reader]()
	defer m.Clear()

	var nilReader io.Reader
	m.Store("test", nilReader)

	v, ok := m.Load("test")
	require.Nil(t, v)
	require.True(t, ok, "Load should report stored nil interface value")
}

func TestMapLoadAndDeleteNilInterfaceValue(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, io.Reader]()
	defer m.Clear()

	var nilReader io.Reader
	m.Store("test", nilReader)

	v, ok := m.LoadAndDelete("test")
	require.Nil(t, v)
	require.True(t, ok, "LoadAndDelete should report stored nil interface value")

	v, ok = m.Load("test")
	require.Nil(t, v)
	require.False(t, ok, "LoadAndDelete should remove stored nil interface value")
}

func TestMapSwapNilInterfaceValue(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, io.Reader]()
	defer m.Clear()

	var nilReader io.Reader
	m.Store("test", nilReader)

	v, ok := m.Swap("test", strings.NewReader("x"))
	require.Nil(t, v)
	require.True(t, ok, "Swap should report stored nil interface value")
}

func TestMapRangeNilInterfaceValue(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[string, io.Reader]()
	defer m.Clear()

	var nilReader io.Reader
	m.Store("reader", nilReader)

	called := false
	require.NotPanics(t, func() {
		m.Range(func(key string, value io.Reader) bool {
			called = true
			require.Equal(t, "reader", key)
			require.Nil(t, value)
			return true
		})
	})
	require.True(t, called, "Range should visit nil interface value")
}

func TestMapRangeNilInterfaceKey(t *testing.T) {
	t.Parallel()

	m := sync.NewMap[fmt.Stringer, string]()
	defer m.Clear()

	var key fmt.Stringer
	m.Store(key, "test")

	called := false
	require.NotPanics(t, func() {
		m.Range(func(key fmt.Stringer, value string) bool {
			called = true
			require.Nil(t, key)
			require.Equal(t, "test", value)
			return true
		})
	})
	require.True(t, called, "Range should visit nil interface key")
}
