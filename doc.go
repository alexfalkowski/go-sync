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
//   - Future: typed asynchronous operations with context-aware waiting.
//   - Group helpers built on errgroup, errors.Join, and singleflight.
//   - Typed wrappers around sync.Pool, sync.Map, and sync/atomic.Value.
//   - BufferPool: a convenience pool for bytes.Buffer.
//
// The package is intentionally small. Most types are either thin wrappers over
// the standard library or type aliases for widely used synchronization helpers.
//
// # Aliases
//
// Once, Mutex, RWMutex, and WaitGroup are aliases for their counterparts in the
// standard library sync package.
//
// Int32, Int64, Uint32, Uint64, Uintptr, Bool, and Pointer[T] are aliases for
// atomic types from sync/atomic.
//
// ErrorGroup is an alias for errgroup.Group. ErrorsGroup runs functions
// concurrently and returns all non-nil errors joined with errors.Join.
//
// AnyPool, AnyMap, AnyValue, AnySingleFlightGroup, and AnySingleFlightResult are
// aliases for the non-generic sync.Pool, sync.Map, atomic.Value,
// singleflight.Group, and singleflight.Result; use Pool[T], Map[K,V], Value[T],
// SingleFlightGroup[T], and SingleFlightResult[T] for the typed wrappers.
//
// # Hooks
//
// Many APIs accept a Hook. Hook.OnRun is the operation to execute and is required.
// Hook.OnError is optional and centralizes error handling; when set, it is invoked
// only when OnRun returns a non-nil error. If OnError is nil, errors are returned
// (or ignored) as described by the calling API.
// Hook.OnRun is validated before timeout or context-cancellation shortcuts, so
// helpers return [ErrNoOnRunProvided] first when OnRun is nil.
// Hook callbacks must not panic. Helpers do not recover panics from Hook.OnRun
// or Hook.OnError; a panic is not routed through Hook.OnError and is not
// returned as an error.
//
// Wait and Timeout return the result of hook.Error when the operation finishes
// before their own deadline logic wins the race. Worker never returns handler
// errors from Schedule or TrySchedule; it only invokes hook.Error for side effects.
//
// # Timeouts
//
// There are two timeout-related helpers with different semantics:
//
// Wait runs hook.OnRun and waits up to the provided timeout for it to complete.
// After Hook.OnRun validation, if the timeout expires (or the provided context
// is canceled) first, Wait returns nil without waiting for OnRun to finish.
// This makes Wait a “best effort” coordination helper rather than a cancellation
// mechanism. A non-positive timeout behaves the same way and returns nil without
// invoking Hook.OnRun.
//
// Timeout runs hook.OnRun using a derived context with the provided timeout.
// After Hook.OnRun validation, if the context’s deadline expires (or it is
// canceled) first, Timeout returns the derived context's cancellation cause
// (typically [ErrTimeout], context.Canceled, or a parent-provided cause). A
// non-positive timeout produces an already-expired derived context, so Timeout
// returns [ErrTimeout] without invoking Hook.OnRun.
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
// Schedule blocks until a slot is acquired or the provided context is done,
// and is context-only: it does not derive or bound any deadline itself. To
// bound the wait for a slot, pass a ctx with a deadline; to give the handler
// its own run budget starting when it actually begins, wrap ctx inside OnRun.
// TrySchedule attempts to schedule only if capacity is available immediately
// and returns ErrWorkerFull otherwise. Errors returned by OnRun are routed to
// hook.OnError (if set) and are not returned by either scheduling method. Use
// Worker.Wait to wait for all scheduled handlers to finish, or return early
// with the provided context's cancellation cause if the handlers have not
// finished first.
//
// The zero value of Worker is not ready for use; construct one with NewWorker.
// A Worker must not be copied after first use; pass and store *Worker values.
//
// # Future
//
// Async starts a typed operation in a new goroutine and returns a Future for its
// result. Future caches the operation's value and error, so Await can be called
// repeatedly by one or more callers after completion.
//
// The context passed to Async is the operation's work context. Async invokes
// the operation even when that context is already done; the operation is
// responsible for observing cancellation. The context passed to Future.Await
// controls only how long that caller waits; cancellation of the await context
// does not cancel the operation. A later Await can still retrieve the eventual
// result. When await cancellation is selected, Await checks completion once
// more, so a result published by that check wins; otherwise it returns the
// await context's cause.
//
// Future does not recover panics from the operation. The operation callback must
// not panic.
//
// Do not copy a Future after first use; pass and store *Future values.
//
// # Groups
//
// ErrorsGroup runs functions concurrently and waits for all of them to finish.
// Wait returns all non-nil errors joined with errors.Join in the order the
// functions were passed to Go. ErrorsGroup retains recorded errors for its
// lifetime, so use a fresh ErrorsGroup for each independent batch of work.
// SetLimit(n) bounds how many functions run concurrently: a negative n means
// unbounded, which is also the default for a zero-value ErrorsGroup. A limit
// of 0 means every subsequent call to Go blocks forever, since a concurrency
// slot is never available. TryGo starts a function only if a concurrency slot
// is currently free, returning false without starting it otherwise.
// Start the first function before calling Wait for an empty group, and wait for
// a batch to finish before starting the next independent batch.
// Do not copy an ErrorsGroup after first use.
//
// SingleFlightGroup[T] is a generic wrapper around singleflight.Group. Its zero
// value is ready for use. Do returns typed values directly, while DoChan returns
// a channel of typed SingleFlightResult[T] values for select-based workflows.
// Both methods preserve singleflight's shared-result behavior: only an
// in-flight call is shared, so completed results are not cached and a later
// call for the same key invokes the function again. When T is an interface
// type and the function returns a nil interface value, they expose that
// result as the zero value of T. Do not copy a SingleFlightGroup[T] after
// first use.
//
// # Typed wrappers
//
// Pool[T] is a typed wrapper around sync.Pool. Its zero value is ready for use.
// NewPool[T] returns a pointer with the default new(T) constructor. Set Pool.New
// when pooled values need custom initialization; if Pool.New is nil, Get
// allocates new(T) when the pool is empty.
//
// BufferPool is a convenience wrapper over Pool[bytes.Buffer]. Unlike Pool[T],
// its zero value is not ready for use; construct one with NewBufferPool. Put
// resets a buffer's length but not its capacity, so an oversized buffer keeps
// that capacity in the pool.
//
// Value[T] is a typed wrapper around atomic.Value. Its zero value is ready for use.
// Load and Swap return the zero value of T if no value has been stored yet. When
// T is an interface type, storing a nil interface value has the same behavior as
// atomic.Value.Store(nil) and panics. Do not copy a Value after first use.
//
// Map[K,V] is a typed wrapper around sync.Map. Its zero value is ready for use.
// If K is an interface type and a nil interface key is stored, Range exposes
// that entry's key as the zero value of K.
// If V is an interface type and a nil interface value is stored, methods that
// return values expose that entry as the zero value of V. Range follows the same
// semantics as sync.Map.Range and does not provide a consistent snapshot. Do not
// copy a Map after first use.
package sync
