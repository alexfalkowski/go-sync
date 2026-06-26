package sync_test

import (
	"context"
	"errors"
	"io"
	"testing"
	"testing/synctest"

	"github.com/alexfalkowski/go-sync"
	"github.com/alexfalkowski/go-sync/internal/test"
	"github.com/stretchr/testify/require"
)

func TestErrorsGroupWaitReturnsNilWithoutErrors(t *testing.T) {
	var g sync.ErrorsGroup

	g.Go(func() error { return nil })
	require.NoError(t, g.Wait())
}

func TestErrorsGroupWaitJoinsErrorsInGoCallOrder(t *testing.T) {
	var g sync.ErrorsGroup

	firstErr := errors.New("first")
	secondErr := errors.New("second")
	firstStarted := make(chan struct{})
	releaseFirst := make(chan struct{})

	g.Go(func() error {
		close(firstStarted)
		<-releaseFirst
		return firstErr
	})
	<-firstStarted

	g.Go(func() error {
		close(releaseFirst)
		return secondErr
	})

	err := g.Wait()
	require.ErrorIs(t, err, firstErr)
	require.ErrorIs(t, err, secondErr)
	require.EqualError(t, err, "first\nsecond")
}

func TestSingleFlightGroup(t *testing.T) {
	g := sync.NewSingleFlightGroup[string]()

	v, err, shared := g.Do("test1", func() (string, error) {
		return "yes", nil
	})
	require.NoError(t, err)
	require.Equal(t, "yes", v)
	require.False(t, shared, "first singleflight call should not be shared")
	g.Forget("test1")

	v, err, shared = g.Do("test2", func() (string, error) {
		return "", context.Canceled
	})
	require.Error(t, err)
	require.Empty(t, v)
	require.False(t, shared, "errored singleflight call should not be shared")
	g.Forget("test2")
}

func TestSingleFlightGroupZeroValue(t *testing.T) {
	var g sync.SingleFlightGroup[string]

	v, err, shared := g.Do("test", func() (string, error) {
		return "yes", nil
	})

	require.NoError(t, err)
	require.Equal(t, "yes", v)
	require.False(t, shared, "zero-value group first call should not be shared")
}

func TestSingleFlightGroupDoesNotCacheCompletedResults(t *testing.T) {
	g := sync.NewSingleFlightGroup[int]()
	var calls sync.Int32

	v, err, shared := g.Do("test", func() (int, error) {
		return int(calls.Add(1)), nil
	})
	require.NoError(t, err)
	require.Equal(t, 1, v, "first completed call should execute function")
	require.False(t, shared, "first completed call should not be shared")

	v, err, shared = g.Do("test", func() (int, error) {
		return int(calls.Add(1)), nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, v, "second completed call should execute function again")
	require.False(t, shared, "completed result should not be cached as shared")
	require.EqualValues(t, 2, calls.Load(), "completed calls should not be cached")
}

func TestSingleFlightGroupForgetInFlight(t *testing.T) {
	g := sync.NewSingleFlightGroup[int]()
	var calls sync.Int32

	first := test.StartBlockedSingleFlight(g, "test", func() (int, error) {
		calls.Add(1)
		return 42, nil
	})
	first.WaitStarted()
	g.Forget("test")

	second := test.DoSingleFlight(g, "test", func() (int, error) {
		calls.Add(1)
		return 7, nil
	})
	require.NoError(t, second.Err)
	require.Equal(t, 7, second.Value, "new call after Forget should run independently")
	require.False(t, second.Shared, "new call after Forget should not share the in-flight result")

	first.Release()
	firstResult := first.Result()
	require.NoError(t, firstResult.Err)
	require.Equal(t, 42, firstResult.Value, "forgotten in-flight call should still complete")
	require.False(t, firstResult.Shared, "forgotten in-flight call should not be shared")
	require.EqualValues(t, 2, calls.Load(), "Forget should allow a second execution for the same key")
}

func TestNewSingleFlightGroupDirectCall(t *testing.T) {
	v, err, shared := sync.NewSingleFlightGroup[int]().Do("test", func() (int, error) {
		return 42, nil
	})

	require.NoError(t, err)
	require.Equal(t, 42, v)
	require.False(t, shared, "direct singleflight call should not be shared")
}

func TestSingleFlightGroupNilInterfaceValue(t *testing.T) {
	g := sync.NewSingleFlightGroup[io.Reader]()

	v, err, shared := g.Do("test", func() (io.Reader, error) {
		var r io.Reader
		return r, nil
	})

	require.NoError(t, err)
	require.Nil(t, v)
	require.False(t, shared, "nil interface result should not be shared")
}

func TestSingleFlightGroupDoChan(t *testing.T) {
	g := sync.NewSingleFlightGroup[string]()

	ch := g.DoChan("test", func() (string, error) {
		return "async", nil
	})
	result := <-ch

	require.NoError(t, result.Err)
	require.Equal(t, "async", result.Value)
	require.False(t, result.Shared, "first DoChan result should not be shared")
}

func TestSingleFlightGroupDoChanError(t *testing.T) {
	g := sync.NewSingleFlightGroup[string]()

	ch := g.DoChan("test", func() (string, error) {
		return "ignored", context.Canceled
	})
	result := <-ch

	require.ErrorIs(t, result.Err, context.Canceled)
	require.Empty(t, result.Value)
	require.False(t, result.Shared, "errored DoChan result should not be shared")
}

func TestSingleFlightGroupDoChanNilInterfaceValue(t *testing.T) {
	g := sync.NewSingleFlightGroup[io.Reader]()

	ch := g.DoChan("test", func() (io.Reader, error) {
		var r io.Reader
		return r, nil
	})
	result := <-ch

	require.NoError(t, result.Err)
	require.Nil(t, result.Value)
	require.False(t, result.Shared, "nil interface DoChan result should not be shared")
}

func TestSingleFlightGroupSharedResult(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := sync.NewSingleFlightGroup[int]()
		secondDone := make(chan test.SingleFlightResult[int], 1)

		var calls sync.Int32

		first := test.StartBlockedSingleFlight(g, "test", func() (int, error) {
			calls.Add(1)
			return 42, nil
		})
		first.WaitStarted()

		go func() {
			secondDone <- test.DoSingleFlight(g, "test", func() (int, error) {
				calls.Add(1)
				return 7, nil
			})
		}()

		synctest.Wait()
		first.Release()
		firstResult := first.Result()
		second := <-secondDone

		require.NoError(t, firstResult.Err)
		require.NoError(t, second.Err)
		require.Equal(t, 42, firstResult.Value, "first caller should receive its result")
		require.Equal(t, 42, second.Value, "duplicate caller should receive shared result")
		require.EqualValues(t, 1, calls.Load(), "duplicate caller should not execute its function")
		require.True(t, firstResult.Shared, "first caller should report a shared result")
		require.True(t, second.Shared, "duplicate caller should report a shared result")
	})
}

func TestSingleFlightGroupDoChanSharedResult(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		g := sync.NewSingleFlightGroup[int]()
		var calls sync.Int32
		started := make(chan struct{})
		release := make(chan struct{})

		first := g.DoChan("test", func() (int, error) {
			calls.Add(1)
			close(started)
			<-release
			return 42, nil
		})
		<-started

		second := g.DoChan("test", func() (int, error) {
			calls.Add(1)
			return 7, nil
		})

		close(release)
		firstResult := <-first
		secondResult := <-second

		require.NoError(t, firstResult.Err)
		require.NoError(t, secondResult.Err)
		require.Equal(t, 42, firstResult.Value, "first DoChan caller should receive its result")
		require.Equal(t, 42, secondResult.Value, "duplicate DoChan caller should receive shared result")
		require.EqualValues(t, 1, calls.Load(), "duplicate DoChan caller should not execute its function")
		require.True(t, firstResult.Shared, "first DoChan caller should report a shared result")
		require.True(t, secondResult.Shared, "duplicate DoChan caller should report a shared result")
	})
}
