package sync

import (
	"context"
	"errors"
	"time"
)

// Handler used for sync.
type Handler func(context.Context) error

// ErrorHandler used for sync.
type ErrorHandler func(context.Context, error) error

var errorHandler Value[ErrorHandler]

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
	handler := errorHandler.Load()
	if handler != nil {
		return handler(ctx, err)
	}

	return nil
}

// IsTimeoutError checks if the error is deadline exceeded.
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
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
	case <-ctx.Done():
		return ctx.Err()
	}
}
