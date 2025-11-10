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

// Lifecycle for operations.
type Lifecycle struct {
	OnRun   Handler
	OnError ErrorHandler
}

// Error will handle the error with the provided handler or use DefaultErrorHandler.
func (l *Lifecycle) Error(ctx context.Context, err error) error {
	if l.OnError == nil {
		l.OnError = DefaultErrorHandler
	}

	return l.OnError(ctx, err)
}

// DefaultErrorHandler for handling errors.
var DefaultErrorHandler ErrorHandler = func(_ context.Context, err error) error {
	return err
}

// IsTimeoutError checks if the error is deadline exceeded.
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// Wait will wait for the handler to complete or continue.
func Wait(ctx context.Context, timeout time.Duration, lc Lifecycle) error {
	ch := make(chan error, 1)
	go func() {
		ch <- lc.Error(ctx, lc.OnRun(ctx))
	}()

	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		return nil
	}
}

// Timeout will wait for the handler to complete or timeout.
func Timeout(ctx context.Context, timeout time.Duration, lc Lifecycle) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := make(chan error, 1)
	go func() {
		ch <- lc.Error(ctx, lc.OnRun(ctx))
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
