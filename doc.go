// Package sync provides small concurrency helpers.
//
// This module is github.com/alexfalkowski/go-sync and exposes package sync.
//
// # Overview
//
// The package contains:
//
//   - Convenience aliases for common synchronization primitives and atomics.
//   - Hook-based helpers for running an operation with centralized error handling.
//   - Wait and Timeout helpers for coordinating an operation with a timeout.
//   - Worker: a bounded scheduler for running operations concurrently.
//   - Group helpers built on errgroup and singleflight.
//   - Typed wrappers around sync.Pool, sync.Map, and sync/atomic.Value.
//   - BufferPool: a convenience pool for bytes.Buffer.
//
// The package is intentionally small. Most types are either thin wrappers over
// the standard library or type aliases for widely used synchronization helpers.
//
// # Aliases
//
// Mutex, RWMutex, and WaitGroup are aliases for their counterparts in the
// standard library sync package.
//
// Int32, Bool, and Pointer are aliases for atomic types from sync/atomic.
//
// ErrorGroup is an alias for errgroup.Group.
//
// # Hooks
//
// Many APIs accept a Hook. Hook.OnRun is the operation to execute and is required.
// Hook.OnError is optional and centralizes error handling; when set, it is invoked
// only when OnRun returns a non-nil error. If OnError is nil, errors are returned
// (or ignored) as described by the calling API.
//
// Wait and Timeout return the result of hook.Error when the operation finishes
// before their own deadline logic wins the race. Worker never returns handler
// errors from Schedule; it only invokes hook.Error for side effects.
//
// # Timeouts
//
// There are two timeout-related helpers with different semantics:
//
// Wait runs hook.OnRun and waits up to the provided timeout for it to complete.
// If the timeout expires (or the provided context is canceled) first, Wait returns
// nil without waiting for OnRun to finish. This makes Wait a “best effort”
// coordination helper rather than a cancellation mechanism. A non-positive
// timeout behaves the same way and returns nil without invoking Hook.OnRun.
//
// Timeout runs hook.OnRun using a derived context with the provided timeout.
// If the context’s deadline expires (or it is canceled) first, Timeout returns
// ctx.Err() (typically context.DeadlineExceeded or context.Canceled). A
// non-positive timeout produces an already-expired derived context, so Timeout
// returns context.DeadlineExceeded without invoking Hook.OnRun.
//
// In both helpers, returning from Wait or Timeout does not forcibly stop the
// goroutine running Hook.OnRun. If OnRun ignores context cancellation, it may
// continue running in the background even after the helper has returned.
//
// In both cases, if Hook.OnRun is nil, the functions return ErrNoOnRunProvided.
//
// # Worker
//
// Worker schedules hook.OnRun to run asynchronously while bounding concurrency.
// Schedule blocks until the handler is scheduled or the provided timeout
// (via context.WithTimeout) expires. Errors returned by OnRun are routed to
// hook.OnError (if set) and are not returned by Schedule. Use Worker.Wait to
// wait for all scheduled handlers to finish.
//
// The zero value of Worker is not ready for use; construct one with NewWorker.
//
// # Groups
//
// SingleFlightGroup[T] is a generic wrapper around singleflight.Group. Its zero
// value is ready for use. Do returns a typed result and preserves singleflight's
// shared-result behavior. When T is an interface type and the function returns a
// nil interface value, Do exposes that result as the zero value of T.
//
// # Typed wrappers
//
// Pool[T] is a typed wrapper around sync.Pool. Its zero value is not ready for use;
// construct one with NewPool[T].
//
// BufferPool is a convenience wrapper over Pool[bytes.Buffer]. Its zero value is
// also not ready for use; construct one with NewBufferPool.
//
// Value[T] is a typed wrapper around atomic.Value. Its zero value is ready for use.
// Load and Swap return the zero value of T if no value has been stored yet. When
// T is an interface type, storing a nil interface value has the same behavior as
// atomic.Value.Store(nil) and panics.
//
// Map[K,V] is a typed wrapper around sync.Map. Its zero value is ready for use.
// If V is an interface type and a nil interface value is stored, methods that
// return values expose that entry as the zero value of V. Range follows the same
// semantics as sync.Map.Range and does not provide a consistent snapshot.
package sync
