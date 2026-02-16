package sync

import (
	"golang.org/x/sync/errgroup"
	"golang.org/x/sync/singleflight"
)

// ErrorGroup is an alias for [errgroup.Group].
//
// It is provided for convenience so users of this package can refer to an
// errgroup without importing `golang.org/x/sync/errgroup` directly.
//
// Note: this is a type alias, not a wrapper. All behavior, including how errors
// are captured and how `Wait` behaves, is defined by `errgroup.Group`.
type ErrorGroup = errgroup.Group

// NewSingleFlightGroup creates a new [SingleFlightGroup] instance.
//
// A SingleFlightGroup is a generic wrapper around [singleflight.Group] that provides type-safe
// results (via the type parameter T) while preserving singleflight semantics.
func NewSingleFlightGroup[T any]() *SingleFlightGroup[T] {
	return &SingleFlightGroup[T]{
		group: &singleflight.Group{},
	}
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
// The type parameter T describes the value returned from [SingleFlightGroup.Do].
// If the function returns a non-nil error, Do returns the zero value of T along
// with that error.
//
// Implementation detail: the underlying singleflight implementation stores and
// returns values as `any`, so this wrapper performs a type assertion back to T.
// As long as the function passed to Do returns a value of type T, the assertion
// will succeed.
type SingleFlightGroup[T any] struct {
	group *singleflight.Group
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
//   - shared reports whether the result was shared with other callers (i.e. this
//     call did not execute fn itself).
func (g *SingleFlightGroup[T]) Do(key string, fn func() (T, error)) (T, error, bool) {
	v, err, shared := g.group.Do(key, func() (any, error) {
		return fn()
	})
	if err != nil {
		return g.zero, err, shared
	}
	return v.(T), nil, shared
}

// Forget forgets the in-flight or completed result for key.
//
// Future calls to [SingleFlightGroup.Do] with the same key will invoke their function again
// rather than waiting for or receiving a previous result.
func (g *SingleFlightGroup[T]) Forget(key string) {
	g.group.Forget(key)
}
