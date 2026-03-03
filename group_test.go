package sync_test

import (
	"context"
	"io"
	"testing"

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
