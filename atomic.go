package sync

import "sync/atomic"

// NewValue returns a new Value.
func NewValue[T any]() Value[T] {
	return Value[T]{v: atomic.Value{}}
}

// Value is a typed wrapper around [atomic.Value].
//
// The zero value is ready for use. Load and Swap return the zero value of T
// when no value has been stored yet.
type Value[T any] struct {
	v    atomic.Value
	zero T
}

// Load returns the stored value, or the zero value of T if none has been stored.
func (v *Value[T]) Load() T {
	value := v.v.Load()
	if value != nil {
		return value.(T)
	}
	return v.zero
}

// Store stores val.
func (v *Value[T]) Store(val T) {
	v.v.Store(val)
}

// Swap stores n and returns the previous value, or the zero value of T if none was stored.
func (v *Value[T]) Swap(n T) T {
	value := v.v.Swap(n)
	if value != nil {
		return value.(T)
	}
	return v.zero
}

// CompareAndSwap executes the compare-and-swap operation.
func (v *Value[T]) CompareAndSwap(o, n T) bool {
	return v.v.CompareAndSwap(o, n)
}
