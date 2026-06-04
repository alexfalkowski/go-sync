package test

import "github.com/alexfalkowski/go-sync"

// SingleFlightResult captures the values returned by SingleFlightGroup.Do.
type SingleFlightResult[T any] struct {
	Value  T
	Err    error
	Shared bool
}

// DoSingleFlight calls SingleFlightGroup.Do and returns the results as a single value.
func DoSingleFlight[T any](g *sync.SingleFlightGroup[T], key string, fn func() (T, error)) SingleFlightResult[T] {
	value, err, shared := g.Do(key, fn)
	return SingleFlightResult[T]{
		Value:  value,
		Err:    err,
		Shared: shared,
	}
}

// BlockedSingleFlight holds an in-flight SingleFlightGroup.Do call until Release.
type BlockedSingleFlight[T any] struct {
	started chan struct{}
	release chan struct{}
	done    chan SingleFlightResult[T]
}

// StartBlockedSingleFlight starts a SingleFlightGroup.Do call and waits until Release before invoking fn.
func StartBlockedSingleFlight[T any](
	g *sync.SingleFlightGroup[T],
	key string,
	fn func() (T, error),
) *BlockedSingleFlight[T] {
	blocked := &BlockedSingleFlight[T]{
		started: make(chan struct{}),
		release: make(chan struct{}),
		done:    make(chan SingleFlightResult[T], 1),
	}

	go func() {
		blocked.done <- DoSingleFlight(g, key, func() (T, error) {
			close(blocked.started)
			<-blocked.release
			return fn()
		})
	}()

	return blocked
}

// WaitStarted blocks until the SingleFlightGroup.Do function has started.
func (b *BlockedSingleFlight[T]) WaitStarted() {
	<-b.started
}

// Release unblocks the SingleFlightGroup.Do function.
func (b *BlockedSingleFlight[T]) Release() {
	close(b.release)
}

// Result returns the completed SingleFlightGroup.Do result.
func (b *BlockedSingleFlight[T]) Result() SingleFlightResult[T] {
	return <-b.done
}
