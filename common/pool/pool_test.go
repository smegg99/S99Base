// common/pool/pool_test.go

package pool_test

import (
	"context"
	"sync/atomic"
	"testing"

	"github.com/smegg99/s99base/common/pool"
)

// every item is processed exactly once, in input order, regardless of worker count
func TestRunProcessesAll(t *testing.T) {
	items := make([]int, 200)
	for i := range items {
		items[i] = i
	}

	var calls atomic.Int64
	results := pool.Run(context.Background(), items, 8, func(_ context.Context, v int) (int, error) {
		calls.Add(1)
		return v * 2, nil
	})

	if len(results) != len(items) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(items))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Fatalf("results[%d].Err = %v, want nil", i, r.Err)
		}
		if r.Value != i*2 {
			t.Fatalf("results[%d].Value = %d, want %d", i, r.Value, i*2)
		}
	}
	if calls.Load() != 200 {
		t.Fatalf("calls = %d, want 200", calls.Load())
	}
}

// a worker count above the item count does not panic and still processes every item
func TestRunClampsWorkers(t *testing.T) {
	items := []int{1, 2, 3}

	results := pool.Run(context.Background(), items, 100, func(_ context.Context, v int) (int, error) {
		return v, nil
	})

	if len(results) != 3 {
		t.Fatalf("len(results) = %d, want 3", len(results))
	}
	for i, r := range results {
		if r.Err != nil {
			t.Fatalf("results[%d].Err = %v, want nil", i, r.Err)
		}
		if r.Value != items[i] {
			t.Fatalf("results[%d].Value = %d, want %d", i, r.Value, items[i])
		}
	}
}

// a pre-cancelled context means fn never runs and every result carries ctx.Err()
func TestRunCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	items := []int{1, 2, 3, 4, 5}
	var calls atomic.Int64
	results := pool.Run(ctx, items, 2, func(_ context.Context, v int) (int, error) {
		calls.Add(1)
		return v, nil
	})

	if len(results) != len(items) {
		t.Fatalf("len(results) = %d, want %d", len(results), len(items))
	}
	for i, r := range results {
		if r.Err != context.Canceled {
			t.Fatalf("results[%d].Err = %v, want context.Canceled", i, r.Err)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("calls = %d, want 0: fn must not run against a cancelled context", calls.Load())
	}
}
