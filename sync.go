package sync

import (
	"context"
	"errors"
	"time"
)

// ErrNoOnRunProvided is returned when Hook.OnRun is nil.
var ErrNoOnRunProvided = errors.New("no OnRun handler provided")

// Handler is the signature for Hook.OnRun.
type Handler func(context.Context) error

// ErrorHandler is the signature for Hook.OnError.
type ErrorHandler func(context.Context, error) error

// Hook bundles OnRun and OnError handlers for Wait, Timeout, and Worker.
type Hook struct {
	OnRun   Handler
	OnError ErrorHandler
}

// Error calls OnError when err is non-nil and OnError is set; otherwise it returns err.
func (h *Hook) Error(ctx context.Context, err error) error {
	if err != nil {
		if h.OnError != nil {
			return h.OnError(ctx, err)
		}
		return err
	}
	return nil
}

// IsTimeoutError reports whether err is [context.DeadlineExceeded].
func IsTimeoutError(err error) bool {
	return errors.Is(err, context.DeadlineExceeded)
}

// Wait runs hook.OnRun and waits up to timeout for it to complete.
//
// If OnRun completes first, Wait returns the result of hook.Error(ctx, hook.OnRun(ctx)).
// If the timeout expires or ctx is done first, Wait returns nil without waiting for OnRun to finish.
func Wait(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
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

// Timeout runs hook.OnRun with a context that has the given timeout.
//
// If OnRun completes first, Timeout returns the result of hook.Error(ctx, hook.OnRun(ctx)).
// If the context is done first, it returns ctx.Err().
func Timeout(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
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
