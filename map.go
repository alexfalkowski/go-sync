package sync

import "sync"

// NewMap creates a new Map instance.
func NewMap[K comparable, V any]() *Map[K, V] {
	return &Map[K, V]{m: &sync.Map{}}
}

// Map is a generic map type that can be used to store key-value pairs.
type Map[K comparable, V any] struct {
	m    *sync.Map
	zero V
}

// Load is an alas for sync.Map.Load.
func (m *Map[K, V]) Load(key K) (V, bool) {
	v, ok := m.m.Load(key)
	if v != nil {
		return v.(V), ok
	}

	return m.zero, ok
}

// Store is an alas for sync.Map.Store.
func (m *Map[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Clear is an alas for sync.Map.Clear.
func (m *Map[K, V]) Clear() {
	m.m.Clear()
}

// LoadOrStore is an alas for sync.Map.LoadOrStore.
func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	v, ok := m.m.LoadOrStore(key, value)
	return v.(V), ok
}

// LoadAndDelete is an alas for sync.Map.LoadAndDelete.
func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	v, ok := m.m.LoadAndDelete(key)
	if v != nil {
		return v.(V), ok
	}
	return m.zero, ok
}

// Delete is an alas for sync.Map.Delete.
func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Swap is an alas for sync.Map.Swap.
func (m *Map[K, V]) Swap(key K, value V) (V, bool) {
	v, ok := m.m.Swap(key, value)
	if v != nil {
		return v.(V), ok
	}

	return m.zero, ok
}

// CompareAndSwap is an alas for sync.Map.CompareAndSwap.
func (m *Map[K, V]) CompareAndSwap(key K, old, new V) bool {
	return m.m.CompareAndSwap(key, old, new)
}

// CompareAndDelete is an alas for sync.Map.CompareAndDelete.
func (m *Map[K, V]) CompareAndDelete(key K, old V) bool {
	return m.m.CompareAndDelete(key, old)
}

// Range is an alas for sync.Map.Range.
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(k, v any) bool { return f(k.(K), v.(V)) })
}
