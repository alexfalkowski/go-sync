package sync

import (
	"bytes"
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

// Handler used for sync.
type Handler func(context.Context) error

// ErrorHandler used for sync.
type ErrorHandler func(context.Context, error) error

var atm atomic.Value

func init() {
	SetErrorHandler(DefaultErrorHandler)
}

// SetErrorHandler that will be used for handling errors.
func SetErrorHandler(handler ErrorHandler) {
	atm.Store(handler)
}

// DefaultErrorHandler for handling errors.
var DefaultErrorHandler ErrorHandler = func(_ context.Context, err error) error {
	return err
}

func handleError(ctx context.Context, err error) error {
	errorHandler := atm.Load().(ErrorHandler)
	if errorHandler != nil {
		return errorHandler(ctx, err)
	}

	return nil
}

// IsTimeoutError checks if the error is deadline exceeded or canceled.
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}

// Wait will wait for the handler to complete or continue.
func Wait(ctx context.Context, timeout time.Duration, handler Handler) error {
	ch := make(chan error, 1)
	go func() {
		ch <- handleError(ctx, handler(ctx))
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
		ch <- handleError(ctx, handler(ctx))
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

// NewBufferPool for sync.
func NewBufferPool() *BufferPool {
	return &BufferPool{NewPool[bytes.Buffer]()}
}

// BufferPool for sync.
type BufferPool struct {
	*Pool[bytes.Buffer]
}

// Get a new buffer.
func (p *BufferPool) Get() *bytes.Buffer {
	return p.Pool.Get()
}

// Put the buffer back.
func (p *BufferPool) Put(buffer *bytes.Buffer) {
	buffer.Reset()
	p.Pool.Put(buffer)
}
