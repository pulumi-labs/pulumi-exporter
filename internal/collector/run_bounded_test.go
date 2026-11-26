package collector

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestRunBoundedLimitsConcurrency(t *testing.T) {
	t.Parallel()

	const limit = 3
	var running, peak, calls atomic.Int32

	runBounded(t.Context(), limit, make([]int, 20), func(int) {
		n := running.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(5 * time.Millisecond)
		running.Add(-1)
		calls.Add(1)
	})

	if got := calls.Load(); got != 20 {
		t.Errorf("runBounded made %d calls, want 20", got)
	}
	if got := peak.Load(); got > limit {
		t.Errorf("runBounded ran %d calls at once, want at most %d", got, limit)
	}
}

func TestRunBoundedStopsWhenContextDone(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cancelAt  int32 // cancel the context once this many calls have started; 0 cancels up front
		wantCalls int32
	}{
		{name: "cancelled before start", cancelAt: 0, wantCalls: 0},
		{name: "cancelled mid-run", cancelAt: 2, wantCalls: 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tt.cancelAt == 0 {
				cancel()
			}

			var calls atomic.Int32
			runBounded(ctx, 1, make([]int, 100), func(int) {
				if calls.Add(1) == tt.cancelAt {
					cancel()
				}
			})

			if got := calls.Load(); got != tt.wantCalls {
				t.Errorf("runBounded made %d calls, want %d", got, tt.wantCalls)
			}
		})
	}
}
