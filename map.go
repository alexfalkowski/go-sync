package sync

import "sync"

// NewMap returns a [Map] ready for use.
//
// The zero value of Map is also ready for use; NewMap is purely optional.
func NewMap[K comparable, V any]() Map[K, V] {
	return Map[K, V]{m: sync.Map{}}
}

// Map is a typed wrapper around [sync.Map].
//
// It provides a generic API while preserving sync.Map’s concurrency properties.
//
// # Zero value
//
// The zero value is ready for use.
//
// # Missing keys vs stored zero values
//
// Methods such as [Map.Load], [Map.LoadAndDelete], and [Map.Swap] return the zero
// value of V when a key is not present. Use the returned boolean to distinguish
// “not present” from a stored zero value.
//
// # Nil interface values
//
// Internally, sync.Map stores values as `any`. This wrapper type-asserts stored
// values back to V for some operations. If V is an interface type, storing a nil
// interface value (for example, `var r io.Reader = nil`) results in an untyped nil
// being stored.
//
// Methods that return values from the map (such as [Map.Load], [Map.LoadOrStore],
// [Map.LoadAndDelete], [Map.Swap], and [Map.Range]) treat this as the zero value of V.
// Use the returned booleans where available to distinguish stored zero values from
// absent keys.
type Map[K comparable, V any] struct {
	zero V
	m    sync.Map
}

// Load returns the value stored in the map for key.
//
// It returns the zero value of V when the key is not present; ok reports whether
// the key was present.
func (m *Map[K, V]) Load(key K) (V, bool) {
	v, ok := m.m.Load(key)
	if v != nil {
		return v.(V), ok
	}

	return m.zero, ok
}

// Store sets the value for key.
//
// If V is an interface type and value is nil, the map stores an untyped nil.
// Load-like methods and Range expose this as the zero value of V.
func (m *Map[K, V]) Store(key K, value V) {
	m.m.Store(key, value)
}

// Clear deletes all keys and values.
func (m *Map[K, V]) Clear() {
	m.m.Clear()
}

// LoadOrStore returns the existing value for key if present.
//
// Otherwise, it stores and returns the given value.
//
// If the stored value is nil (for example, when V is an interface type and a nil
// value was stored), it returns the zero value of V.
func (m *Map[K, V]) LoadOrStore(key K, value V) (V, bool) {
	v, ok := m.m.LoadOrStore(key, value)
	if v != nil {
		return v.(V), ok
	}

	return m.zero, ok
}

// LoadAndDelete deletes the value for key, returning the previous value if any.
//
// It returns the zero value of V when the key is not present; loaded reports whether
// the key was present.
func (m *Map[K, V]) LoadAndDelete(key K) (V, bool) {
	v, ok := m.m.LoadAndDelete(key)
	if v != nil {
		return v.(V), ok
	}
	return m.zero, ok
}

// Delete deletes the value for key.
func (m *Map[K, V]) Delete(key K) {
	m.m.Delete(key)
}

// Swap swaps the value for key and returns the previous value if any.
//
// It returns the zero value of V when the key is not present; loaded reports whether
// the key was present.
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
// If f returns false, Range stops the iteration.
//
// If a stored value is nil (for example, when V is an interface type and a nil
// value was stored), Range passes the zero value of V to f.
func (m *Map[K, V]) Range(f func(key K, value V) bool) {
	m.m.Range(func(k, v any) bool {
		if v != nil {
			return f(k.(K), v.(V))
		}
		return f(k.(K), m.zero)
	})
}
