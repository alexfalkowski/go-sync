package sync

import (
	"errors"
	"sync"

	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

// WaitGroup is an alias for [sync.WaitGroup].
//
// It is provided for convenience so users of this package can refer to a
// WaitGroup without importing `sync` directly.
type WaitGroup = sync.WaitGroup

// ErrorGroup is an alias for [errgroup.Group].
//
// It is provided for convenience so users of this package can refer to an
// errgroup without importing `golang.org/x/sync/errgroup` directly.
//
// Note: this is a type alias, not a wrapper. All behavior, including how errors
// are captured and how `Wait` behaves, is defined by `errgroup.Group`.
type ErrorGroup = errgroup.Group

// ErrorsGroup runs functions concurrently and joins all returned errors.
//
// Unlike [ErrorGroup], which returns the first non-nil error reported by an
// errgroup, ErrorsGroup records every non-nil error and returns them from
// [ErrorsGroup.Wait] using [errors.Join].
//
// ErrorsGroup retains recorded errors for its lifetime. Use a fresh ErrorsGroup
// for each independent batch of work.
//
// The zero value of ErrorsGroup is ready for use.
//
// An ErrorsGroup must not be copied after first use.
type ErrorsGroup struct {
	errors []error
	wait   sync.WaitGroup
	mutex  sync.Mutex
}

// Go calls the given function in a new goroutine.
//
// The first call to [ErrorsGroup.Wait] blocks until all functions started by Go
// have returned. Non-nil errors are joined in the order the functions were
// passed to Go, not the order they complete.
func (g *ErrorsGroup) Go(f func() error) {
	index := g.index()

	g.wait.Go(func() {
		if err := f(); err != nil {
			g.add(index, err)
		}
	})
}

// Wait blocks until all functions started by [ErrorsGroup.Go] have returned,
// then returns all non-nil errors joined with [errors.Join].
//
// Wait does not clear recorded errors. A later call to Wait on the same
// ErrorsGroup can return errors from earlier Go calls.
func (g *ErrorsGroup) Wait() error {
	g.wait.Wait()

	g.mutex.Lock()
	defer g.mutex.Unlock()

	return errors.Join(g.errors...)
}

func (g *ErrorsGroup) index() int {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	index := len(g.errors)
	g.errors = append(g.errors, nil)

	return index
}

func (g *ErrorsGroup) add(index int, err error) {
	g.mutex.Lock()
	defer g.mutex.Unlock()

	g.errors[index] = err
}

// NewSingleFlightGroup creates a pointer to a new [SingleFlightGroup] instance.
//
// A SingleFlightGroup is a generic wrapper around [singleflight.Group] that
// provides type-safe results (via the type parameter T) while preserving
// singleflight semantics.
//
// The zero value of [SingleFlightGroup] is already ready for use, so calling
// NewSingleFlightGroup is optional.
func NewSingleFlightGroup[T any]() *SingleFlightGroup[T] {
	return &SingleFlightGroup[T]{}
}

// SingleFlightGroup suppresses duplicate executions of functions associated with the same key.
//
// It is a thin, generic wrapper around [singleflight.Group] that provides
// type-safe results (via the type parameter T) while preserving singleflight
// semantics.
//
// For a given key, the first caller executes the provided function and
// concurrent callers for the same key wait for that execution to complete and
// receive the same result.
//
// The zero value of SingleFlightGroup is ready for use.
//
// The type parameter T describes the value returned from [SingleFlightGroup.Do].
// If the function returns a non-nil error, Do returns the zero value of T along
// with that error.
//
// Implementation detail: the underlying singleflight implementation stores and
// returns values as `any`, so this wrapper performs a type assertion back to T.
// As long as the function passed to Do returns a value of type T, the assertion
// will succeed.
//
// When T is an interface type and fn returns a nil interface value, Do exposes
// that result as the zero value of T.
//
// A SingleFlightGroup must not be copied after first use.
type SingleFlightGroup[T any] struct {
	group singleflight.Group
	zero  T
}

// Do executes fn for the given key, making sure that only one execution is in
// flight at a time for that key.
//
// If another execution for the same key is already running, Do waits for it and
// returns the same results.
//
// It returns (value, err, shared):
//   - value is the successful result of fn (type T), or the zero value of T if err != nil.
//   - err is the error returned by fn.
//   - shared reports whether the result was given to multiple callers.
//
// If fn returns a nil interface value and T is an interface type, value is the
// zero value of T.
func (g *SingleFlightGroup[T]) Do(key string, fn func() (T, error)) (T, error, bool) {
	v, err, shared := g.group.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		return g.zero, err, shared
	}

	// When T is an interface, fn may successfully return a nil interface value.
	// singleflight stores that as an untyped nil, so avoid asserting nil to T.
	if v == nil {
		return g.zero, nil, shared
	}

	return v.(T), nil, shared
}

// Forget forgets an in-flight call for key.
//
// Future calls to [SingleFlightGroup.Do] with the same key will invoke their
// function rather than waiting for the earlier call to complete.
func (g *SingleFlightGroup[T]) Forget(key string) {
	g.group.Forget(key)
}
