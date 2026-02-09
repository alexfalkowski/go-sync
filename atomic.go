package sync

import "sync/atomic"

// NewValue creates a new atomic value with the given zero value.
func NewValue[T any]() Value[T] {
	return Value[T]{v: atomic.Value{}}
}

// Value is a generic atomic value.
type Value[T any] struct {
	v    atomic.Value
	zero T
}

// Load is an alias for [atomic.Value.Load].
func (v *Value[T]) Load() T {
	value := v.v.Load()
	if value != nil {
		return value.(T)
	}
	return v.zero
}

// Store is an alias for [atomic.Value.Store].
func (v *Value[T]) Store(val T) {
	v.v.Store(val)
}

// Swap is an alias for [atomic.Value.Swap].
func (v *Value[T]) Swap(n T) T {
	value := v.v.Swap(n)
	if value != nil {
		return value.(T)
	}
	return v.zero
}

// CompareAndSwap is an alias for [atomic.Value.CompareAndSwap].
func (v *Value[T]) CompareAndSwap(o, n T) bool {
	return v.v.CompareAndSwap(o, n)
}
