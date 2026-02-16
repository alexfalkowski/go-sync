package sync_test

import (
	"context"
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

func TestGroup(t *testing.T) {
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
