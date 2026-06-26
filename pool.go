package sync

import "sync"

// NewPool returns a pointer to an initialized [Pool] for values of type T.
//
// The returned pool creates new values on demand by allocating `new(T)` when empty.
//
// Note: the returned Pool stores *T values. Callers should treat values obtained
// from [Pool.Get] as temporarily borrowed and return them to the pool with
// [Pool.Put] when finished.
func NewPool[T any]() *Pool[T] {
	return &Pool[T]{
		New: func() *T {
			return new(T)
		},
	}
}

// Pool is a typed wrapper around [sync.Pool].
//
// It stores and returns pointers to T (*T) to avoid copying large values.
// Pool does not reset values automatically on Put. If New is non-nil, Get calls
// it to create a value when the pool is empty. If New is nil, Get allocates
// new(T) when the pool is empty.
//
// # Zero value
//
// The zero value is ready for use.
//
// # Semantics
//
// Pool has the same semantics as [sync.Pool]:
//
//   - Items may be dropped at any time by the runtime.
//   - Items are meant to be reused to reduce allocations, not to manage
//     resource lifetimes.
//   - Values taken from the pool should be considered ephemeral and should not
//     be assumed to be unique or to remain in the pool.
//
// Callers are responsible for resetting any state on values before reusing them,
// if needed.
//
// A Pool must not be copied after first use.
type Pool[T any] struct {
	New  func() *T
	pool sync.Pool
}

// Get returns a pointer to a T from the pool.
//
// The returned pointer is owned by the caller until it is returned via [Pool.Put].
// If New is set and returns nil, Get returns nil.
func (p *Pool[T]) Get() *T {
	value := p.pool.Get()
	if value != nil {
		return value.(*T)
	}
	if p.New != nil {
		return p.New()
	}
	return new(T)
}

// Put returns b to the pool.
//
// Callers should ensure b is in an appropriate state for reuse (for example, by
// resetting fields) before calling Put.
//
// If b is nil, Put is a no-op.
func (p *Pool[T]) Put(b *T) {
	if b == nil {
		return
	}

	p.pool.Put(b)
}
