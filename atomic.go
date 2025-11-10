package sync

import "sync/atomic"

// Value is a generic atomic value.
type Value[T any] struct {
	atomic.Value
}

// Load is an alias for [atomic.Value.Load].
func (v *Value[T]) Load() T {
	return v.Value.Load().(T)
}

// Store is an alias for [atomic.Value.Store].
func (v *Value[T]) Store(val T) {
	v.Value.Store(val)
}

// Swap is an alias for [atomic.Value.Swap].
func (v *Value[T]) Swap(n T) T {
	return v.Value.Swap(n).(T)
}

// CompareAndSwap is an alias for [atomic.Value.CompareAndSwap].
func (v *Value[T]) CompareAndSwap(o, n T) bool {
	return v.Value.CompareAndSwap(o, n)
}
