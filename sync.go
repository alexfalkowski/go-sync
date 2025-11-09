package sync

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/alexfalkowski/go-sync/atomic"
)

// Handler used for sync.
type Handler func(context.Context) error

// ErrorHandler used for sync.
type ErrorHandler func(context.Context, error) error

var errorHandler atomic.Value[ErrorHandler]

func init() {
	SetErrorHandler(DefaultErrorHandler)
}

// SetErrorHandler that will be used for handling errors.
func SetErrorHandler(handler ErrorHandler) {
	errorHandler.Store(handler)
}

// DefaultErrorHandler for handling errors.
var DefaultErrorHandler ErrorHandler = func(_ context.Context, err error) error {
	return err
}

func handleError(ctx context.Context, err error) error {
	errorHandler := errorHandler.Load()
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
