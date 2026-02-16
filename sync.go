package sync

import (
	"context"
	"errors"
	"sync"
	"time"
)

// Mutex is an alias of [sync.Mutex].
type Mutex = sync.Mutex

// RWMutex is an alias of [sync.RWMutex].
type RWMutex = sync.RWMutex

// ErrNoOnRunProvided is returned when [Hook.OnRun] is nil.
var ErrNoOnRunProvided = errors.New("no OnRun handler provided")

// Handler is the signature for [Hook.OnRun].
//
// The provided [context.Context] is the context used by the operation invoking
// the hook (for example, the original ctx passed to [Wait], or the derived
// timeout context created by [Timeout]).
type Handler func(context.Context) error

// ErrorHandler is the signature for [Hook.OnError].
//
// It is invoked only when a non-nil error is returned from [Hook.OnRun]. If the
// ErrorHandler returns a different error, that error is returned to the caller.
type ErrorHandler func(context.Context, error) error

// Hook bundles handlers used by helpers in this package.
//
// The helpers in this package call [Hook.OnRun] to perform work and then pass the
// returned error to [Hook.Error], which applies [Hook.OnError] if configured.
//
// [Hook.OnRun] must be non-nil; otherwise operations return [ErrNoOnRunProvided].
type Hook struct {
	OnRun   Handler
	OnError ErrorHandler
}

// Error applies OnError when err is non-nil and OnError is set; otherwise it returns err.
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
// Wait is a “best effort” waiting helper. It does not cancel the work started by
// OnRun. Instead, it starts OnRun asynchronously and then waits for whichever
// happens first:
//
//  1. OnRun completes: Wait returns hook.Error(ctx, hook.OnRun(ctx)).
//  2. The timeout elapses: Wait returns nil immediately.
//  3. ctx is done: Wait returns nil immediately.
//
// Important: if the timeout elapses or ctx becomes done, Wait returns without
// waiting for OnRun to finish. The OnRun goroutine may continue running in the
// background, and any error it eventually produces will be discarded.
//
// If hook.OnRun is nil, Wait returns [ErrNoOnRunProvided].
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

// Timeout runs hook.OnRun with a derived context that has the given timeout.
//
// Timeout differs from [Wait] in two key ways:
//
//  1. Cancellation is propagated to OnRun by passing a derived context created by
//     [context.WithTimeout]. Well-behaved OnRun implementations should observe
//     ctx.Done() and exit promptly.
//  2. Timeout reports timeout/cancellation by returning ctx.Err() when the
//     derived context becomes done before OnRun completes.
//
// If OnRun completes before the derived context is done, Timeout returns
// hook.Error(ctx, hook.OnRun(ctx)) where ctx is the derived context.
//
// If hook.OnRun is nil, Timeout returns [ErrNoOnRunProvided].
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
