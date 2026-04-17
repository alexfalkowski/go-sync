package sync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// Once is an alias of [sync.Once].
//
// It is provided for convenience so users of this package can refer to a
// once value without importing the standard library sync package directly.
type Once = sync.Once

// Mutex is an alias of [sync.Mutex].
//
// It is provided for convenience so users of this package can refer to a mutex
// without importing the standard library sync package directly.
type Mutex = sync.Mutex

// RWMutex is an alias of [sync.RWMutex].
//
// It is provided for convenience so users of this package can refer to a
// read/write mutex without importing the standard library sync package directly.
type RWMutex = sync.RWMutex

// ErrNoOnRunProvided is returned when [Hook.OnRun] is nil.
var ErrNoOnRunProvided = errors.New("no OnRun handler provided")

// ErrTimeout is the timeout cause used by derived contexts in this package.
//
// It wraps [context.DeadlineExceeded], so [errors.Is] matches both values.
var ErrTimeout = fmt.Errorf("timeout: %w", context.DeadlineExceeded)

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
//
// Whether the value returned from [Hook.Error] is observed depends on the
// calling helper:
//   - [Wait] returns it only if OnRun finishes before timeout/cancellation wins.
//   - [Timeout] returns it only if OnRun finishes before the derived context ends.
//   - [Worker.Schedule] never returns it; handler errors are only observed via
//     [Hook.OnError] side effects.
type Hook struct {
	OnRun   Handler
	OnError ErrorHandler
}

// Error applies [Hook.OnError] when err is non-nil and OnError is set.
//
// Otherwise, it returns err unchanged. A nil err always yields nil.
func (h *Hook) Error(ctx context.Context, err error) error {
	if err != nil {
		if h.OnError != nil {
			return h.OnError(ctx, err)
		}
		return err
	}
	return nil
}

// IsTimeoutError reports whether err matches [ErrTimeout] or
// [context.DeadlineExceeded].
//
// It uses [errors.Is], so wrapped deadline-exceeded errors also report true.
func IsTimeoutError(err error) bool {
	return errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded)
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
// Even after receiving the OnRun result, Wait re-checks timeout/context state
// before returning. If timeout has elapsed or ctx is done at that point, Wait
// returns nil.
//
// Important: if the timeout elapses or ctx becomes done, Wait returns without
// waiting for OnRun to finish. The OnRun goroutine may continue running in the
// background. If OnRun later returns an error, Hook.OnError may still run in
// that goroutine, but Wait discards the final return value.
//
// If ctx is already done on entry (or timeout <= 0), Wait returns nil without
// invoking OnRun.
//
// If hook.OnRun is nil, Wait returns [ErrNoOnRunProvided].
func Wait(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}
	if ctx.Err() != nil || timeout <= 0 {
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
		select {
		case <-timer.C:
			return nil
		default:
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
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
//     [context.WithTimeoutCause]. Well-behaved OnRun implementations should observe
//     ctx.Done() and exit promptly.
//  2. Timeout reports timeout/cancellation by returning [context.Cause] when the
//     derived context becomes done before OnRun completes.
//
// If OnRun completes before the derived context is done, Timeout returns
// hook.Error(ctx, hook.OnRun(ctx)) where ctx is the derived context.
//
// Even after receiving the OnRun result, Timeout re-checks the derived context.
// If it is done at that point, Timeout returns [context.Cause].
//
// If the input ctx is already done on entry, Timeout returns its cancellation
// cause without invoking OnRun. If timeout <= 0, Timeout returns [ErrTimeout]
// without invoking OnRun.
//
// As with [Wait], returning from Timeout does not forcibly stop the goroutine
// running OnRun. If OnRun ignores ctx.Done(), it may continue running in the
// background. Hook.OnError may still run there, but Timeout discards its return
// value once the derived context has already ended.
//
// If hook.OnRun is nil, Timeout returns [ErrNoOnRunProvided].
func Timeout(ctx context.Context, timeout time.Duration, hook Hook) error {
	if hook.OnRun == nil {
		return ErrNoOnRunProvided
	}
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	ctx, cancel := context.WithTimeoutCause(ctx, timeout, ErrTimeout)
	defer cancel()
	if ctx.Err() != nil {
		return context.Cause(ctx)
	}

	ch := make(chan error, 1)
	go func() {
		ch <- hook.Error(ctx, hook.OnRun(ctx))
		close(ch)
	}()

	select {
	case err := <-ch:
		if ctx.Err() != nil {
			return context.Cause(ctx)
		}
		return err
	case <-ctx.Done():
		return context.Cause(ctx)
	}
}
