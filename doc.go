// Package sync provides small concurrency helpers.
//
// This module is github.com/alexfalkowski/go-sync and exposes package sync.
//
// # Overview
//
// The package contains:
//
//   - Hook-based helpers for running an operation with shared error handling.
//   - Wait and Timeout helpers for coordinating an operation with a timeout.
//   - Worker: a bounded scheduler for running operations concurrently.
//   - Typed wrappers around sync.Pool, sync.Map, and sync/atomic.Value.
//   - BufferPool: a convenience pool for bytes.Buffer.
//
// # Hooks
//
// Many APIs accept a Hook. Hook.OnRun is the operation to execute and is required.
// Hook.OnError is optional and centralizes error handling; when set, it is invoked
// only when OnRun returns a non-nil error. If OnError is nil, errors are returned
// (or ignored) as described by the calling API.
//
// # Timeouts
//
// There are two timeout-related helpers with different semantics:
//
// Wait runs hook.OnRun and waits up to the provided timeout for it to complete.
// If the timeout expires (or the provided context is canceled) first, Wait returns
// nil without waiting for OnRun to finish. This makes Wait a “best effort”
// coordination helper rather than a cancellation mechanism.
//
// Timeout runs hook.OnRun using a derived context with the provided timeout.
// If the context’s deadline expires (or it is canceled) first, Timeout returns
// ctx.Err() (typically context.DeadlineExceeded or context.Canceled).
//
// In both cases, if Hook.OnRun is nil, the functions return ErrNoOnRunProvided.
//
// # Worker
//
// Worker schedules hook.OnRun to run asynchronously while bounding concurrency.
// Schedule will block until the handler is scheduled or the provided timeout
// (via context.WithTimeout) expires. Errors returned by OnRun are routed to
// hook.OnError (if set) and are not returned by Schedule. Use Worker.Wait to
// wait for all scheduled handlers to finish.
//
// # Typed wrappers
//
// Pool[T] is a typed wrapper around sync.Pool. Its zero value is not ready for use;
// construct one with NewPool[T].
//
// Value[T] is a typed wrapper around atomic.Value. Its zero value is ready for use.
// Load and Swap return the zero value of T if no value has been stored yet.
//
// Map[K,V] is a typed wrapper around sync.Map. Its zero value is ready for use.
// For methods that type-assert stored values (such as Range and LoadOrStore),
// storing a nil interface value (when V is an interface type) can cause a panic.
package sync
