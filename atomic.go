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

// AnyValue is an alias for [atomic.Value].
//
// It is provided for convenience so users of this package can refer to the
// non-generic atomic value without importing `sync/atomic` directly. For a
// typed value holding T, use [Value].
type AnyValue = atomic.Value

// NewValue returns a pointer to a new [Value] wrapper.
//
// The returned pointer is ready for use.
func NewValue[T any]() *Value[T] {
	return &Value[T]{v: atomic.Value{}}
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
// value of T. [Value.CompareAndSwap] follows [atomic.Value.CompareAndSwap]: an
// unset Value can be initialized only by comparing against a nil interface value.
//
// # Type safety and panics
//
// Internally, [atomic.Value] stores values as `any`. This wrapper type-asserts
// the stored value back to T on Load/Swap. The assertion will succeed as long as
// you only store values of type T in this Value.
//
// Storing or swapping values of different concrete types in the same underlying
// atomic.Value has the same constraints as atomic.Value itself and may panic.
//
// When T is an interface type, storing a nil interface value has the same
// behavior as [atomic.Value.Store](nil) and panics.
//
// A Value must not be copied after first use.
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

// Store atomically stores value.
//
// It follows [atomic.Value.Store] semantics. In particular, storing a nil
// interface value panics, and later stores must remain compatible with the
// concrete type established by the first store.
func (v *Value[T]) Store(value T) {
	v.v.Store(value)
}

// Swap atomically stores new and returns the previous value.
//
// If no value has been stored yet, it returns the zero value of T.
//
// It follows [atomic.Value.Swap] semantics. In particular, swapping a nil
// interface value panics, and later swaps must remain compatible with the
// concrete type established by the first store or swap.
func (v *Value[T]) Swap(new T) T {
	value := v.v.Swap(new)
	if value != nil {
		return value.(T)
	}
	return v.zero
}

// CompareAndSwap executes the atomic compare-and-swap operation.
//
// It follows [atomic.Value.CompareAndSwap] semantics. If old's dynamic type is
// not comparable, CompareAndSwap panics. As with [Value.Store], interface-typed
// values must also satisfy atomic.Value's nil and concrete-type rules. In
// particular, CompareAndSwap with a nil interface value for new panics.
//
// If no value has been stored yet, CompareAndSwap can initialize the Value only
// when old is a nil interface value. Comparing against T's zero value returns
// false when that zero value is converted to a non-nil interface, such as a zero
// number or typed nil pointer.
func (v *Value[T]) CompareAndSwap(old, new T) bool {
	return v.v.CompareAndSwap(old, new)
}
