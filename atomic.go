package sync

import "sync/atomic"

// NewValue returns a new [Value] wrapper.
//
// The returned value is ready for use.
func NewValue[T any]() Value[T] {
	return Value[T]{v: atomic.Value{}}
}

// Value is a typed wrapper around [atomic.Value].
//
// It provides a generic API while preserving the semantics and constraints of
// atomic.Value.
//
// # Zero value
//
// The zero value is ready for use.
//
// # Unset values
//
// If no value has been stored yet, [Value.Load] and [Value.Swap] return the zero
// value of T.
//
// # Type safety and panics
//
// Internally, [atomic.Value] stores values as `any`. This wrapper type-asserts
// the stored value back to T on Load/Swap. The assertion will succeed as long as
// you only store values of type T in this Value.
//
// Storing values of different concrete types in the same underlying atomic.Value
// has the same constraints as atomic.Value itself and may panic.
type Value[T any] struct {
	v    atomic.Value
	zero T
}

// Load returns the stored value.
//
// If no value has been stored yet, it returns the zero value of T.
func (v *Value[T]) Load() T {
	value := v.v.Load()
	if value != nil {
		return value.(T)
	}
	return v.zero
}

// Store atomically stores val.
func (v *Value[T]) Store(val T) {
	v.v.Store(val)
}

// Swap atomically stores n and returns the previous value.
//
// If no value has been stored yet, it returns the zero value of T.
func (v *Value[T]) Swap(n T) T {
	value := v.v.Swap(n)
	if value != nil {
		return value.(T)
	}
	return v.zero
}

// CompareAndSwap executes the atomic compare-and-swap operation.
func (v *Value[T]) CompareAndSwap(o, n T) bool {
	return v.v.CompareAndSwap(o, n)
}
