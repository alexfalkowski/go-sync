package sync

import "sync"

// NewPool returns an initialized Pool for values of type T.
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
// The zero value is not ready for use; use [NewPool].
type Pool[T any] struct {
	pool *sync.Pool
}

// Get returns an item of type T.
func (p *Pool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put puts an item of type T back into the pool.
func (p *Pool[T]) Put(b *T) {
	p.pool.Put(b)
}
