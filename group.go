package sync

import "golang.org/x/sync/singleflight"

// NewGroup creates a new [Group] instance.
//
// A Group is a generic wrapper around [singleflight.Group] that provides type-safe
// results (via the type parameter T) while preserving singleflight semantics.
func NewGroup[T any]() *Group[T] {
	return &Group[T]{
		group: &singleflight.Group{},
	}
}

// Group suppresses duplicate executions of functions associated with the same key.
//
// It is a thin, generic wrapper around [singleflight.Group]. For a given key, the
// first caller runs the provided function, and concurrent callers for the same
// key wait for the first call to complete and receive the same result.
//
// The type parameter T describes the value returned from [Group.Do]. If the
// function returns a non-nil error, Do returns the zero value of T along with
// that error.
//
// Note: this type relies on the underlying singleflight implementation storing
// and returning values as interface{} (any) and performing a type assertion back
// to T. As long as all calls to Do on a given Group use the same T (enforced by
// the generic type) and the provided function returns a value of type T, the
// assertion will succeed.
type Group[T any] struct {
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
func (g *Group[T]) Do(key string, fn func() (T, error)) (T, error, bool) {
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
// Future calls to [Group.Do] with the same key will invoke their function again
// rather than waiting for or receiving a previous result.
func (g *Group[T]) Forget(key string) {
	g.group.Forget(key)
}
