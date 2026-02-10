package sync

import "sync"

// NewMap returns a Map ready for use.
//
// The zero value of Map is also ready for use.
func NewMap[K comparable, V any]() Map[K, V] {
	return Map[K, V]{m: sync.Map{}}
}

// Map is a typed wrapper around [sync.Map].
//
// The zero value is ready for use.
type Map[K comparable, V any] struct {
	m    sync.Map
	zero V
}

// Load returns the value stored in the map for a key.
//
// It returns the zero value of V when the key is not present.
func (m *Map[K, V]) Load(key K) (V, bool) {
	v, ok := m.m.Load(key)
	if v != nil {
		return v.(V), ok
	}

	return m.zero, ok
}

// Store sets the value for a key.
func (m *Map[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Clear deletes all keys and values.
func (m *Map[K, V]) Clear() {
	m.m.Clear()
}

// LoadOrStore returns the existing value for the key if present.
//
// Otherwise, it stores and returns the given value.
//
// This method panics if the stored value is nil (for example, when V is an interface
// type and a nil value was stored).
func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	v, ok := m.m.LoadOrStore(key, value)
	return v.(V), ok
}

// LoadAndDelete deletes the value for a key, returning the previous value if any.
//
// It returns the zero value of V when the key is not present.
func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	v, ok := m.m.LoadAndDelete(key)
	if v != nil {
		return v.(V), ok
	}
	return m.zero, ok
}

// Delete deletes the value for a key.
func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Swap swaps the value for a key and returns the previous value if any.
//
// It returns the zero value of V when the key is not present.
func (m *Map[K, V]) Swap(key K, value V) (V, bool) {
	v, ok := m.m.Swap(key, value)
	if v != nil {
		return v.(V), ok
	}

	return m.zero, ok
}

// CompareAndSwap executes the compare-and-swap operation.
func (m *Map[K, V]) CompareAndSwap(key K, old, new V) bool {
	return m.m.CompareAndSwap(key, old, new)
}

// CompareAndDelete executes the compare-and-delete operation.
func (m *Map[K, V]) CompareAndDelete(key K, old V) bool {
	return m.m.CompareAndDelete(key, old)
}

// Range calls f sequentially for each key and value present in the map.
//
// If f returns false, range stops the iteration.
//
// This method panics if a stored value is nil (for example, when V is an interface
// type and a nil value was stored).
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(k, v any) bool { return f(k.(K), v.(V)) })
}
