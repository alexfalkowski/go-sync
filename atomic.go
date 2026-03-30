package sync

import "sync/atomic"

// Int32 is an alias for [atomic.Int32].
//
// It is provided for convenience so users of this package can refer to a typed
// atomic integer without importing sync/atomic directly.
type Int32 = atomic.Int32

// Int64 is an alias for [atomic.Int64].
//
// It is provided for convenience so users of this package can refer to a typed
// atomic integer without importing sync/atomic directly.
type Int64 = atomic.Int64

// Uint32 is an alias for [atomic.Uint32].
//
// It is provided for convenience so users of this package can refer to a typed
// atomic integer without importing sync/atomic directly.
type Uint32 = atomic.Uint32

// Uint64 is an alias for [atomic.Uint64].
//
// It is provided for convenience so users of this package can refer to a typed
// atomic integer without importing sync/atomic directly.
type Uint64 = atomic.Uint64

// Uintptr is an alias for [atomic.Uintptr].
//
// It is provided for convenience so users of this package can refer to a typed
// atomic integer without importing sync/atomic directly.
type Uintptr = atomic.Uintptr

// Bool is an alias for [atomic.Bool].
//
// It is provided for convenience so users of this package can refer to a typed
// atomic boolean without importing sync/atomic directly.
type Bool = atomic.Bool

// Pointer is an alias for [atomic.Pointer].
//
// It is provided for convenience so users of this package can refer to a typed
// atomic pointer without importing sync/atomic directly.
type Pointer[T any] = atomic.Pointer[T]

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
//
// When T is an interface type, storing a nil interface value has the same
// behavior as [atomic.Value.Store](nil) and panics.
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
//
// It follows [atomic.Value.Store] semantics. In particular, storing a nil
// interface value panics, and later stores must remain compatible with the
// concrete type established by the first store.
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
//
// It follows [atomic.Value.CompareAndSwap] semantics. If o's dynamic type is
// not comparable, CompareAndSwap panics. As with [Value.Store], interface-typed
// values must also satisfy atomic.Value's concrete-type rules.
func (v *Value[T]) CompareAndSwap(o, n T) bool {
	return v.v.CompareAndSwap(o, n)
}
