package sync

import (
	"context"
	"errors"
	"time"
)

// ErrNoOnRunProvided is returned when no OnRun handler is provided.
var ErrNoOnRunProvided = errors.New("no OnRun handler provided")

// Handler used for sync.
type Handler func(context.Context) error

// ErrorHandler used for sync.
type ErrorHandler func(context.Context, error) error

// Hook for operations.
type Hook struct {
	OnRun   Handler
	OnError ErrorHandler
}

// Error will handle the error with the provided handler.
func (h *Hook) Error(ctx context.Context, err error) error {
	if err != nil {
		if h.OnError != nil {
			return h.OnError(ctx, err)
		}
		return err
	}
	return nil
}

// IsTimeoutError checks if the error is deadline exceeded.
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// Wait will wait for the handler to complete or continue.
func Wait(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}

	if ctx.Err() != nil {
		return nil
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ch := make(chan error, 1)
	go func() {
		ch <- hook.Error(ctx, hook.OnRun(ctx))
		close(ch)
	}()

	select {
	case err := <-ch:
		return err
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return nil
	}
}

// Timeout will wait for the handler to complete or timeout.
func Timeout(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}

	if ctx.Err() != nil {
		return ctx.Err()
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ch := make(chan error, 1)
	go func() {
		ch <- hook.Error(ctx, hook.OnRun(ctx))
		close(ch)
	}()

	select {
	case err := <-ch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}
