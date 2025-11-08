package sync

import (
	"context"
	"time"
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
