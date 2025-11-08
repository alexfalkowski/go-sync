package sync

import (
	"context"
	"sync"
	"time"
)

type (
	// Mutex is an alias of sync.Mutex.
	Mutex = sync.Mutex

	// RWMutex is an alias of sync.RWMutex.
	RWMutex = sync.RWMutex
)

// Handler used for sync.
type Handler func(context.Context) error

// Wait will wait for the handler to complete or continue.
func Wait(ctx context.Context, timeout time.Duration, handler Handler) error {
	ch := make(chan error, 1)
	go func() {
		ch <- handler(ctx)
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return nil
	}
}

// Timeout will wait for the handler to complete or timeout.
func Timeout(ctx context.Context, timeout time.Duration, handler Handler) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := make(chan error, 1)
	go func() {
		ch <- handler(ctx)
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return ctx.Err()
	}
}

// NewPool of type T.
func NewPool[T any]() *Pool[T] {
	pool := &sync.Pool{
		New: func() any {
			return new(T)
		},
	}
	return &Pool[T]{pool: pool}
}

// Pool of type T.
type Pool[T any] struct {
	pool *sync.Pool
}

// Get an item of type T.
func (p *Pool[T]) Get() *T {
	return p.pool.Get().(*T)
}

// Put an item of type T back.
func (p *Pool[T]) Put(b *T) {
	p.pool.Put(b)
}
