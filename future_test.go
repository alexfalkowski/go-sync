package sync_test

import (
	"context"
	"errors"
	"testing"
	"testing/synctest"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestAsyncReturnsValue(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		future := sync.Async(t.Context(), func(context.Context) (string, error) {
			return "done", nil
		})
		synctest.Wait()

		value, err := future.Await(t.Context())

		require.NoError(t, err)
		require.Equal(t, "done", value)
	})
}

func TestAsyncReturnsError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("work failed")
		future := sync.Async(t.Context(), func(context.Context) (int, error) {
			return 0, wantErr
		})
		synctest.Wait()

		value, err := future.Await(t.Context())
		secondValue, secondErr := future.Await(t.Context())

		require.ErrorIs(t, err, wantErr)
		require.Zero(t, value)
		require.ErrorIs(t, secondErr, wantErr)
		require.Zero(t, secondValue)
	})
}

func TestFutureAwaitCanBeRepeated(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		var calls sync.Int32
		future := sync.Async(t.Context(), func(context.Context) (int, error) {
			calls.Add(1)
			return 42, nil
		})
		synctest.Wait()

		first, firstErr := future.Await(t.Context())
		second, secondErr := future.Await(t.Context())

		require.NoError(t, firstErr)
		require.NoError(t, secondErr)
		require.Equal(t, 42, first)
		require.Equal(t, 42, second)
		require.EqualValues(t, 1, calls.Load(), "Future should execute work once")
	})
}

func TestFutureAwaitCanBeCalledConcurrently(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		future := sync.Async(t.Context(), func(context.Context) (int, error) {
			<-release
			return 42, nil
		})
		synctest.Wait()

		results := make(chan int, 2)
		errCh := make(chan error, 2)
		for range 2 {
			go func() {
				value, err := future.Await(t.Context())
				results <- value
				errCh <- err
			}()
		}
		close(release)
		synctest.Wait()

		require.NoError(t, <-errCh)
		require.NoError(t, <-errCh)
		require.Equal(t, 42, <-results)
		require.Equal(t, 42, <-results)
	})
}

func TestFutureAwaitCachedErrorCanBeCalledConcurrently(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("work failed")
		future := sync.Async(t.Context(), func(context.Context) (int, error) {
			return 0, wantErr
		})
		synctest.Wait()

		errCh := make(chan error, 2)
		for range 2 {
			go func() {
				_, err := future.Await(t.Context())
				errCh <- err
			}()
		}
		synctest.Wait()

		require.ErrorIs(t, <-errCh, wantErr)
		require.ErrorIs(t, <-errCh, wantErr)
	})
}

func TestFutureAwaitContextCancellationDoesNotCancelWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		future := sync.Async(t.Context(), func(context.Context) (string, error) {
			<-release
			return "done", nil
		})
		synctest.Wait()

		waitCtx, cancel := context.WithCancel(t.Context())
		cancel()
		value, err := future.Await(waitCtx)

		require.ErrorIs(t, err, context.Canceled)
		require.Empty(t, value)

		close(release)
		synctest.Wait()
		value, err = future.Await(t.Context())

		require.NoError(t, err)
		require.Equal(t, "done", value)
	})
}

func TestAsyncWorkContextCancellation(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		workCtx, cancel := context.WithCancel(t.Context())
		future := sync.Async(workCtx, func(ctx context.Context) (int, error) {
			<-ctx.Done()
			return 0, context.Cause(ctx)
		})
		cancel()
		synctest.Wait()

		value, err := future.Await(t.Context())

		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, value)
	})
}

func TestAsyncInvokesWorkWithAlreadyCanceledContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		workCtx, cancel := context.WithCancel(t.Context())
		cancel()
		future := sync.Async(workCtx, func(ctx context.Context) (int, error) {
			return 0, context.Cause(ctx)
		})
		synctest.Wait()

		value, err := future.Await(t.Context())

		require.ErrorIs(t, err, context.Canceled)
		require.Zero(t, value)
	})
}
