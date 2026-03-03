package sync_test

import (
	"context"
	"io"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestSingleFlightGroup(t *testing.T) {
	g := sync.NewSingleFlightGroup[string]()

	v, err, shared := g.Do("test1", func() (string, error) {
		return "yes", nil
	})
	require.NoError(t, err)
	require.Equal(t, "yes", v)
	require.False(t, shared)
	g.Forget("test1")

	v, err, shared = g.Do("test2", func() (string, error) {
		return "", context.Canceled
	})
	require.Error(t, err)
	require.Empty(t, v)
	require.False(t, shared)
	g.Forget("test2")
}

func TestSingleFlightGroupZeroValue(t *testing.T) {
	var g sync.SingleFlightGroup[string]

	v, err, shared := g.Do("test", func() (string, error) {
		return "yes", nil
	})

	require.NoError(t, err)
	require.Equal(t, "yes", v)
	require.False(t, shared)
}

func TestSingleFlightGroupNilInterfaceValue(t *testing.T) {
	g := sync.NewSingleFlightGroup[io.Reader]()

	v, err, shared := g.Do("test", func() (io.Reader, error) {
		var r io.Reader
		return r, nil
	})

	require.NoError(t, err)
	require.Nil(t, v)
	require.False(t, shared)
}

func TestSingleFlightGroupSharedResult(t *testing.T) {
	g := sync.NewSingleFlightGroup[int]()
	started := make(chan struct{})
	release := make(chan struct{})
	firstDone := make(chan struct{})

	var (
		calls            atomic.Int32
		v1, v2           int
		err1             error
		err2             error
		shared1, shared2 bool
	)

	go func() {
		defer close(firstDone)
		v1, err1, shared1 = g.Do("test", func() (int, error) {
			calls.Add(1)
			close(started)
			<-release
			return 42, nil
		})
	}()

	<-started

	go func() {
		time.Sleep(20 * time.Millisecond)
		close(release)
	}()

	v2, err2, shared2 = g.Do("test", func() (int, error) {
		calls.Add(1)
		return 7, nil
	})
	<-firstDone

	require.NoError(t, err1)
	require.NoError(t, err2)
	require.Equal(t, 42, v1)
	require.Equal(t, 42, v2)
	require.EqualValues(t, 1, calls.Load())
	require.True(t, shared1 || shared2)
}
