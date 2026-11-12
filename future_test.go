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

func TestFutureDoneReportsCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		release := make(chan struct{})
		future := sync.Async(t.Context(), func(context.Context) (int, error) {
			<-release
			return 42, nil
		})
		done := future.Done()
		require.Equal(t, done, future.Done(), "Done must return the same completion channel")
		synctest.Wait()

		select {
		case <-done:
			require.Fail(t, "Done closed before the operation finished")
		default:
		}

		close(release)
		<-done

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		value, err := future.Await(ctx)

		require.NoError(t, err)
		require.Equal(t, 42, value)
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

		select {
		case <-future.Done():
			require.Fail(t, "canceling Await closed Done while the operation was still running")
		default:
		}

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

func TestGoReturnsCachedError(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("task failed")
		var calls sync.Int32
		task := sync.Go(t.Context(), func(context.Context) error {
			calls.Add(1)
			return wantErr
		})
		synctest.Wait()

		require.ErrorIs(t, task.Await(t.Context()), wantErr)
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, task.Await(ctx), wantErr)
		require.EqualValues(t, 1, calls.Load(), "awaiting a task must not repeat its work")
	})
}

func TestTaskDoneReportsCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("task failed")
		release := make(chan struct{})
		task := sync.Go(t.Context(), func(context.Context) error {
			<-release
			return wantErr
		})
		done := task.Done()
		require.Equal(t, done, task.Done(), "Done must return the same completion channel")
		synctest.Wait()

		select {
		case <-done:
			require.Fail(t, "Done closed before the task finished")
		default:
		}

		close(release)
		<-done

		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		require.ErrorIs(t, task.Await(ctx), wantErr)
	})
}

func TestTaskAwaitCancellationDoesNotCancelWork(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		wantErr := errors.New("task failed")
		release := make(chan struct{})
		task := sync.Go(t.Context(), func(ctx context.Context) error {
			<-release
			if err := ctx.Err(); err != nil {
				return err
			}
			return wantErr
		})
		synctest.Wait()

		waitCtx, cancel := context.WithCancelCause(t.Context())
		waitErr := errors.New("caller stopped waiting")
		cancel(waitErr)
		require.ErrorIs(t, task.Await(waitCtx), waitErr)

		select {
		case <-task.Done():
			require.Fail(t, "canceling Await closed Done while the task was still running")
		default:
		}

		close(release)
		synctest.Wait()
		require.ErrorIs(t, task.Await(t.Context()), wantErr)
	})
}

func TestGoInvokesWorkWithAlreadyCanceledContext(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ctx, cancel := context.WithCancel(t.Context())
		cancel()
		called := false
		task := sync.Go(ctx, func(ctx context.Context) error {
			called = true
			return context.Cause(ctx)
		})
		synctest.Wait()

		require.ErrorIs(t, task.Await(t.Context()), context.Canceled)
		require.True(t, called, "Go must invoke the handler even when its work context is canceled")
	})
}
