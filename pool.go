package sync

import "sync"

// NewPool returns an initialized [Pool] for values of type T.
//
// The pool creates new values on demand by allocating `new(T)` when empty.
//
// Note: the returned Pool stores *T values. Callers should treat values obtained
// from [Pool.Get] as temporarily borrowed and return them to the pool with
// [Pool.Put] when finished.
func NewPool[T any]() *Pool[T] {
	pool := &sync.Pool{
		New: func() any {
			return new(T)
		},
	}
	return &Pool[T]{pool: pool}
}

// Pool is a typed wrapper around [sync.Pool].
//
// It stores and returns pointers to T (*T) to avoid copying large values.
//
// # Zero value
//
// The zero value is not ready for use. Construct a Pool with [NewPool].
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
type Pool[T any] struct {
	pool *sync.Pool
}

// Get returns a pointer to a T from the pool.
//
// The returned pointer is owned by the caller until it is returned via [Pool.Put].
func (p *Pool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put returns b to the pool.
//
// Callers should ensure b is in an appropriate state for reuse (for example, by
// resetting fields) before calling Put.
func (p *Pool[T]) Put(b *T) {
	p.pool.Put(b)
}
