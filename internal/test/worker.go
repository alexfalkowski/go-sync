package test

import (
	"context"
	"testing"

	"github.com/alexfalkowski/go-sync"
	"github.com/stretchr/testify/require"
)

// NewWorkerScheduleProbe creates a probe for a concurrency limit and total schedule attempts.
func NewWorkerScheduleProbe(limit int32, total int) *WorkerScheduleProbe {
	return &WorkerScheduleProbe{
		limit:    limit,
		total:    total,
		started:  make(chan struct{}, total),
		release:  make(chan struct{}),
		errs:     make(chan error, total),
		exceeded: make(chan int32, total),
	}
}

// WorkerScheduleProbe coordinates worker scheduling tests.
type WorkerScheduleProbe struct {
	started  chan struct{}
	release  chan struct{}
	errs     chan error
	exceeded chan int32
	running  sync.Int32
	peak     sync.Int32
	limit    int32
	total    int
}

// Schedule starts one scheduling attempt against worker.
func (p *WorkerScheduleProbe) Schedule(ctx context.Context, worker *sync.Worker) {
	go func() {
		p.errs <- worker.Schedule(ctx, sync.Hook{
			OnRun: p.run,
		})
	}()
}

// RequireLimitReached waits for the configured concurrency limit and verifies the peak.
func (p *WorkerScheduleProbe) RequireLimitReached(t *testing.T) {
	t.Helper()

	for range p.limit {
		<-p.started
	}

	require.Equal(t, p.limit, p.running.Load(), "worker should have limit handlers running")
	require.Equal(t, p.limit, p.peak.Load(), "worker should reach the configured concurrency limit")
}

// ReleaseAll unblocks all handlers currently held by the probe.
func (p *WorkerScheduleProbe) ReleaseAll() {
	close(p.release)
}

// RequireScheduled verifies every schedule attempt returned nil.
func (p *WorkerScheduleProbe) RequireScheduled(t *testing.T) {
	t.Helper()

	for range p.total {
		require.NoError(t, <-p.errs)
	}
}

// RequireNeverExceeded verifies the worker never exceeded its configured limit.
func (p *WorkerScheduleProbe) RequireNeverExceeded(t *testing.T) {
	t.Helper()

	select {
	case count := <-p.exceeded:
		require.Failf(t, "worker exceeded concurrency limit", "running handlers: %d, limit: %d", count, p.limit)
	default:
	}
}

func (p *WorkerScheduleProbe) run(context.Context) error {
	current := p.running.Add(1)
	defer p.running.Add(-1)

	p.recordPeak(current)
	if current > p.limit {
		p.exceeded <- current
	}

	p.started <- struct{}{}
	<-p.release
	return nil
}

func (p *WorkerScheduleProbe) recordPeak(current int32) {
	for {
		peak := p.peak.Load()
		if current <= peak || p.peak.CompareAndSwap(peak, current) {
			return
		}
	}
}
